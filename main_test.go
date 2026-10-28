package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisConnectionString(t *testing.T) {
	for _, tt := range []struct {
		name string
		body string
		want string
	}{
		{"single URI", "redis://user:secret@redis:6379/1", "redis://user:secret@redis:6379/1"},
		{"URI list", "# Redis endpoints\r\n\r\nrediss://redis:6380/2\r\nredis://backup:6379\r\n", "rediss://redis:6380/2"},
		{"whitespace", "\n  redis://redis:6379  \n", "redis://redis:6379"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/redis?service=geoip", r.URL.String())
				assert.Equal(t, "text/uri-list", r.Header.Get("Accept"))
				w.Header().Set("Content-Type", "text/uri-list")
				fmt.Fprint(w, tt.body)
			}))
			defer srv.Close()

			got, err := redisConnectionString("redis://localhost:6379", srv.URL+"/redis?service=geoip", false)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedisConnectionStringPrecedence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("explicit -redis must prevent HTTP requests")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	for _, connStr := range []string{"redis://explicit:6379", "redis://localhost:6379", ""} {
		got, err := redisConnectionString(connStr, srv.URL, true)
		require.NoError(t, err)
		assert.Equal(t, connStr, got)
	}

	got, err := redisConnectionString("redis://localhost:6379", "", false)
	require.NoError(t, err)
	assert.Equal(t, "redis://localhost:6379", got)
}

func TestRedisConnectionStringErrors(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
		err    string
	}{
		{"HTTP error", http.StatusServiceUnavailable, "redis://redis:6379", "HTTP 503"},
		{"empty response", http.StatusOK, "", "contains no URIs"},
		{"comments only", http.StatusOK, "# no endpoints\r\n\r\n", "contains no URIs"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.body)
			}))
			defer srv.Close()

			got, err := redisConnectionString("redis://localhost:6379", srv.URL, false)
			require.ErrorContains(t, err, tt.err)
			assert.Empty(t, got)
		})
	}

	for _, from := range []string{"://invalid", "unsupported://redis"} {
		_, err := redisConnectionString("redis://localhost:6379", from, false)
		require.Error(t, err)
	}
}
