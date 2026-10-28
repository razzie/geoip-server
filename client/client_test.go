package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/razzie/geoip-server/geoip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientProvider(t *testing.T) {
	for _, tt := range []struct{ address, want string }{
		{"https://geoip.example.com", "geoip.example.com"},
		{"http://localhost:8080/prefix", "localhost:8080"},
		{"://invalid", "://invalid"},
	} {
		t.Run(tt.address, func(t *testing.T) {
			assert.Equal(t, tt.want, NewClient(tt.address).Provider())
		})
	}
}

func TestClientGetLocationAndCache(t *testing.T) {
	want := &geoip.Location{IP: "8.8.8.8", Country: "United States", CountryCode: "US", Source: "test-provider"}
	var calls int32
	requests := make(chan *http.Request, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		requests <- r
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	assert.Equal(t, srv.URL, c.ServerAddress)
	_, ok := c.getCachedLocation("8.8.8.8")
	assert.False(t, ok)

	loc, err := c.GetLocation(context.Background(), "8.8.8.8")
	require.NoError(t, err)
	assert.Equal(t, want, loc)
	request := <-requests
	assert.Equal(t, http.MethodGet, request.Method)
	assert.Equal(t, "/8.8.8.8", request.URL.Path)

	cached, err := c.GetLocation(context.Background(), "8.8.8.8")
	require.NoError(t, err)
	assert.Same(t, loc, cached)
	assert.EqualValues(t, 1, atomic.LoadInt32(&calls))

	_, err = c.GetLocation(context.Background(), "example.com")
	require.NoError(t, err)
	assert.Equal(t, "/example.com", (<-requests).URL.Path)
	assert.EqualValues(t, 2, atomic.LoadInt32(&calls))
}

func TestClientDoesNotCacheFailedResponses(t *testing.T) {
	for _, tt := range []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"invalid JSON", func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, "invalid JSON")
		}},
		{"incompatible JSON", func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `{"ip":123}`)
		}},
		{"truncated response body", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "100")
			_, _ = io.WriteString(w, `{}`)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var calls int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&calls, 1)
				tt.handler(w, r)
			}))
			defer srv.Close()
			c := NewClient(srv.URL)
			for i := 0; i < 2; i++ {
				loc, err := c.GetLocation(context.Background(), "8.8.8.8")
				require.Error(t, err)
				assert.Nil(t, loc)
			}
			assert.EqualValues(t, 2, atomic.LoadInt32(&calls))
			_, ok := c.getCachedLocation("8.8.8.8")
			assert.False(t, ok)
		})
	}
}

func TestClientGetLocationCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewClient("http://127.0.0.1")
	loc, err := c.GetLocation(ctx, "8.8.8.8")
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, loc)
	_, ok := c.getCachedLocation("8.8.8.8")
	assert.False(t, ok)
}

func TestClientCachePreservesFirstLocation(t *testing.T) {
	c := NewClient("http://example.com")
	first := &geoip.Location{IP: "8.8.8.8", Country: "United States"}
	c.cacheLocation("example.com", first)
	c.cacheLocation("example.com", &geoip.Location{IP: "1.1.1.1"})
	loc, ok := c.getCachedLocation("example.com")
	require.True(t, ok)
	assert.Same(t, first, loc)
}

func TestClientConcurrentCacheAccess(t *testing.T) {
	c := NewClient("http://example.com")
	want := &geoip.Location{IP: "8.8.8.8"}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.cacheLocation("example.com", want)
			loc, ok := c.getCachedLocation("example.com")
			assert.True(t, ok)
			assert.Same(t, want, loc)
		}()
	}
	wg.Wait()
}
