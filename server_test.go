package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/razzie/geoip-server/geoip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubClient struct {
	name   string
	lookup func(context.Context, string) (*geoip.Location, error)
}

func (c stubClient) Provider() string { return c.name }

func (c stubClient) GetLocation(ctx context.Context, hostname string) (*geoip.Location, error) {
	return c.lookup(ctx, hostname)
}

func TestServerGetLocationFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	want := &geoip.Location{IP: "8.8.8.8", Country: "United States"}
	var calls []string
	srv := NewServer(nil, []geoip.Client{
		stubClient{name: "failing", lookup: func(gotCtx context.Context, host string) (*geoip.Location, error) {
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "example.com", host)
			calls = append(calls, "failing")
			return nil, errors.New("provider unavailable")
		}},
		stubClient{name: "working", lookup: func(gotCtx context.Context, host string) (*geoip.Location, error) {
			assert.Same(t, ctx, gotCtx)
			assert.Equal(t, "example.com", host)
			calls = append(calls, "working")
			return want, nil
		}},
		stubClient{name: "unused", lookup: func(context.Context, string) (*geoip.Location, error) {
			t.Error("providers after the first success must not be called")
			return nil, nil
		}},
	})
	loc, err := srv.getLocation(ctx, "example.com")
	require.NoError(t, err)
	assert.Same(t, want, loc)
	assert.Equal(t, []string{"failing", "working"}, calls)
}

func TestServerGetLocationAllProvidersFail(t *testing.T) {
	firstErr := errors.New("first failed")
	lastErr := errors.New("last failed")
	srv := NewServer(nil, []geoip.Client{
		stubClient{name: "first", lookup: func(context.Context, string) (*geoip.Location, error) {
			return nil, firstErr
		}},
		stubClient{name: "last", lookup: func(context.Context, string) (*geoip.Location, error) {
			return nil, lastErr
		}},
	})
	loc, err := srv.getLocation(context.Background(), "example.com")
	require.ErrorIs(t, err, lastErr)
	assert.Nil(t, loc)
}

func TestServerGetLocationCachesProviderResult(t *testing.T) {
	db, _ := newTestDB(t)
	want := &geoip.Location{IP: "8.8.8.8", Hostname: "example.com", Country: "United States"}
	calls := 0
	srv := NewServer(db, []geoip.Client{
		stubClient{name: "test-provider", lookup: func(context.Context, string) (*geoip.Location, error) {
			calls++
			return want, nil
		}},
	})
	for _, host := range []string{want.Hostname, want.Hostname, want.IP} {
		loc, err := srv.getLocation(context.Background(), host)
		require.NoError(t, err)
		assert.Equal(t, want, loc)
	}
	assert.Equal(t, 1, calls)
}

func TestServerGetLocationToleratesCacheErrors(t *testing.T) {
	for _, name := range []string{"invalid cached JSON", "unavailable cache"} {
		t.Run(name, func(t *testing.T) {
			db, redisServer := newTestDB(t)
			if name == "invalid cached JSON" {
				require.NoError(t, redisServer.Set("example.com", "invalid JSON"))
			} else {
				redisServer.SetError("ERR database unavailable")
			}
			want := &geoip.Location{IP: "8.8.8.8", Hostname: "example.com"}
			calls := 0
			srv := NewServer(db, []geoip.Client{
				stubClient{name: "test-provider", lookup: func(context.Context, string) (*geoip.Location, error) {
					calls++
					return want, nil
				}},
			})
			loc, err := srv.getLocation(context.Background(), "example.com")
			require.NoError(t, err)
			assert.Equal(t, want, loc)
			assert.Equal(t, 1, calls)
		})
	}
}

func TestServerServeHTTPLocation(t *testing.T) {
	for _, tt := range []struct {
		name      string
		path      string
		forwarded string
		wantHost  string
		wantIP    string
	}{
		{"explicit hostname", "/example.com", "", "example.com", "192.0.2.1"},
		{"explicit IP", "/8.8.8.8", "", "8.8.8.8", "192.0.2.1"},
		{"remote address", "/", "", "192.0.2.1", "192.0.2.1"},
		{"forwarded address", "/", "198.51.100.2, 192.0.2.1", "198.51.100.2", "198.51.100.2"},
		{"explicit hostname behind proxy", "/example.com", "198.51.100.2", "example.com", "198.51.100.2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			want := &geoip.Location{IP: "8.8.8.8", Country: "United States", CountryCode: "US", Source: "test-provider"}
			var hosts []string
			var contexts []context.Context
			srv := NewServer(nil, []geoip.Client{
				stubClient{name: "test-provider", lookup: func(ctx context.Context, host string) (*geoip.Location, error) {
					hosts = append(hosts, host)
					contexts = append(contexts, ctx)
					return want, nil
				}},
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil).WithContext(ctx)
			req.RemoteAddr = "192.0.2.1:1234"
			if tt.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.forwarded)
			}
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			var loc geoip.Location
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &loc))
			assert.Equal(t, *want, loc)
			// The second lookup supplies the requesting IP's location for logging.
			assert.Equal(t, []string{tt.wantHost, tt.wantIP}, hosts)
			require.Len(t, contexts, 2)
			assert.Same(t, ctx, contexts[0])
			assert.Equal(t, context.Background(), contexts[1])
		})
	}
}

func TestServerServeHTTPProviderError(t *testing.T) {
	srv := NewServer(nil, []geoip.Client{
		stubClient{name: "test-provider", lookup: func(context.Context, string) (*geoip.Location, error) {
			return nil, errors.New("provider unavailable")
		}},
	})
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/example.com", nil))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "provider unavailable\n", rec.Body.String())
}

func TestServerServeHTTPFavicon(t *testing.T) {
	srv := NewServer(nil, []geoip.Client{
		stubClient{lookup: func(context.Context, string) (*geoip.Location, error) {
			t.Error("favicon requests must not trigger location lookups")
			return nil, nil
		}},
	})
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/favicon.ico", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))
	assert.Equal(t, strconv.Itoa(rec.Body.Len()), rec.Header().Get("Content-Length"))
	require.GreaterOrEqual(t, rec.Body.Len(), 8)
	assert.Equal(t, []byte{137, 80, 78, 71, 13, 10, 26, 10}, rec.Body.Bytes()[:8])
	assert.Equal(t, favicon, rec.Body.Bytes())
}
