package geoip

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderGetURL(t *testing.T) {
	tests := []struct {
		name     string
		template string
		params   []string
		want     string
	}{
		{"hostname", "https://example.com/%[1]s", nil, "https://example.com/8.8.8.8"},
		{"token", "https://example.com/%[1]s?token=%[2]s", []string{"secret"}, "https://example.com/8.8.8.8?token=secret"},
		{"extra parameters", "https://example.com/%s", []string{"unused"}, "https://example.com/8.8.8.8"},
		{"reordered parameters", "https://example.com/%[3]s/%[1]s?token=%[2]s", []string{"secret", "v2"}, "https://example.com/v2/8.8.8.8?token=secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Provider{TemplateURL: tt.template}
			assert.Equal(t, tt.want, p.getURL("8.8.8.8", tt.params))
		})
	}
}

func TestProviderDoMapping(t *testing.T) {
	p := Provider{
		Name: "test-provider",
		Mappings: Location{
			CountryCode: "data.country_code", Country: "data.country",
			RegionCode: "data.region_code", Region: "data.region", City: "data.city",
			TimeZone: "data.timezone", ISP: "data.isp",
		},
	}
	var loc Location
	p.doMapping(&loc, strings.NewReader(`{"data":{"country_code":"HU","region_code":123,"region":"Pest","city":"Budapest","timezone":"Europe/Budapest","isp":"Test ISP"}}`))
	assert.Equal(t, Location{
		CountryCode: "HU", Country: "Hungary", RegionCode: "123", Region: "Pest",
		City: "Budapest", TimeZone: "Europe/Budapest", ISP: "Test ISP", Source: "test-provider",
	}, loc)

	t.Run("country code inferred and unmapped fields ignored", func(t *testing.T) {
		p := Provider{Name: "country-only", Mappings: Location{Country: "country"}}
		var loc Location
		p.doMapping(&loc, strings.NewReader(`{"country":"Hungary","city":"Budapest"}`))
		assert.Equal(t, Location{Country: "Hungary", CountryCode: "HU", Source: "country-only"}, loc)
	})

	t.Run("missing and null fields", func(t *testing.T) {
		var loc Location
		p.doMapping(&loc, strings.NewReader(`{"data":{"city":null}}`))
		assert.Equal(t, Location{Source: "test-provider"}, loc)
	})
}

func TestProviderRejectsInvalidAndPrivateHosts(t *testing.T) {
	p := Provider{Name: "test-provider", TemplateURL: "://invalid/%[1]s"}
	for _, tt := range []struct{ host, message string }{
		{"", "not a hostname:"},
		{"not a host!", "not a hostname:"},
		{"127.0.0.1", "IP 127.0.0.1 is private"},
		{"10.0.0.1", "IP 10.0.0.1 is private"},
		{"::1", "IP ::1 is private"},
	} {
		t.Run(tt.host, func(t *testing.T) {
			loc, err := p.NewClient().GetLocation(context.Background(), tt.host)
			require.ErrorContains(t, err, tt.message)
			assert.Nil(t, loc)
		})
	}
}

func TestProviderRequestErrors(t *testing.T) {
	t.Run("invalid URL", func(t *testing.T) {
		p := Provider{TemplateURL: "://invalid/%[1]s"}
		loc, err := p.doRequest(context.Background(), "8.8.8.8", nil)
		require.Error(t, err)
		assert.Nil(t, loc)
	})

	t.Run("HTTP error and request parameters", func(t *testing.T) {
		requests := make(chan *http.Request, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests <- r
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer srv.Close()
		p := Provider{Name: "test-provider", TemplateURL: srv.URL + "/%[1]s?token=%[2]s"}
		c := p.NewClient("secret")
		assert.Equal(t, p.Name, c.Provider())
		loc, err := c.GetLocation(context.Background(), "8.8.8.8")
		require.EqualError(t, err, "429 Too Many Requests")
		assert.Nil(t, loc)
		request := <-requests
		require.NotNil(t, request)
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "/8.8.8.8", request.URL.Path)
		assert.Equal(t, "secret", request.URL.Query().Get("token"))
		assert.Equal(t, "application/json", request.Header.Get("Accept"))
		assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		p := Provider{TemplateURL: "http://127.0.0.1/%[1]s"}
		loc, err := p.doRequest(ctx, "8.8.8.8", nil)
		require.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, loc)
	})
}

func TestProviderRequestWithDNSUnavailable(t *testing.T) {
	// Keep forward and reverse DNS failures deterministic and offline.
	originalResolver := net.DefaultResolver
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("DNS unavailable in test")
		},
	}
	t.Cleanup(func() { net.DefaultResolver = originalResolver })

	t.Run("hostname lookup fails", func(t *testing.T) {
		p := Provider{TemplateURL: "http://127.0.0.1/%[1]s"}
		loc, err := p.doRequest(context.Background(), "geoip-test.invalid", nil)
		require.EqualError(t, err, "failed to get IP from: geoip-test.invalid")
		assert.Nil(t, loc)
	})

	t.Run("successful response without reverse DNS", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"country_code":"HU","city":"Budapest"}`))
		}))
		defer srv.Close()
		p := Provider{
			Name: "test-provider", TemplateURL: srv.URL + "/%[1]s",
			Mappings: Location{CountryCode: "country_code", City: "city"},
		}
		loc, err := p.NewClient().GetLocation(context.Background(), "8.8.8.8")
		require.NoError(t, err)
		assert.Equal(t, &Location{
			IP: "8.8.8.8", CountryCode: "HU", Country: "Hungary",
			City: "Budapest", Source: "test-provider",
		}, loc)
	})
}

func TestGetIP(t *testing.T) {
	for _, address := range []string{"8.8.8.8", "2001:4860:4860::8888"} {
		assert.True(t, net.ParseIP(address).Equal(getIP(address)))
	}
	assert.Nil(t, getIP("invalid host!"))
}

func TestGetClients(t *testing.T) {
	clients := GetClients(" , unknown-provider, ipinfo.io secret, \t ip-api.com , ipinfo.io second-token, ")
	require.Len(t, clients, 3)
	assert.Equal(t, "ipinfo.io", clients[0].Provider())
	assert.Equal(t, "ip-api.com", clients[1].Provider())
	assert.Equal(t, "ipinfo.io", clients[2].Provider())
	assert.Equal(t, []string{"secret"}, clients[0].(*client).params)
	assert.Empty(t, clients[1].(*client).params)
	assert.Equal(t, []string{"second-token"}, clients[2].(*client).params)
	assert.Empty(t, GetClients(""))
	assert.Empty(t, GetClients("unknown-provider, ,"))
}
