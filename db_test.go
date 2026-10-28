package main

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v7"
	"github.com/razzie/geoip-server/geoip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) (*DB, *miniredis.Miniredis) {
	t.Helper()
	srv := miniredis.RunT(t)
	db, err := NewDB("redis://" + srv.Addr())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, db.client.Close()) })
	return db, srv
}

func TestNewDB(t *testing.T) {
	t.Run("default expiration", func(t *testing.T) {
		db, _ := newTestDB(t)
		assert.Equal(t, time.Hour, db.ExpirationTime)
	})
	t.Run("invalid URL", func(t *testing.T) {
		db, err := NewDB("http://localhost:6379")
		require.Error(t, err)
		assert.Nil(t, db)
	})
	t.Run("ping failure", func(t *testing.T) {
		srv := miniredis.RunT(t)
		srv.RequireAuth("secret")
		db, err := NewDB("redis://" + srv.Addr())
		require.ErrorContains(t, err, "NOAUTH")
		assert.Nil(t, db)
	})
}

func TestDBLocationRoundTrip(t *testing.T) {
	for _, hostname := range []string{"example.com", ""} {
		t.Run("hostname="+hostname, func(t *testing.T) {
			db, srv := newTestDB(t)
			db.ExpirationTime = 2 * time.Minute
			want := &geoip.Location{
				IP: "8.8.8.8", Hostname: hostname, CountryCode: "HU", Country: "Hungary",
				RegionCode: "BU", Region: "Budapest", City: "Budapest",
				TimeZone: "Europe/Budapest", ISP: "Test ISP", Source: "test-provider",
			}
			require.NoError(t, db.SetLocation(want))
			keys := []string{want.IP}
			if hostname != "" {
				keys = append(keys, hostname)
			}
			assert.ElementsMatch(t, keys, srv.Keys())
			for _, key := range keys {
				loc, err := db.GetLocation(key)
				require.NoError(t, err)
				assert.Equal(t, want, loc)
				assert.Equal(t, db.ExpirationTime, srv.TTL(key))
			}

			srv.FastForward(db.ExpirationTime)
			for _, key := range keys {
				loc, err := db.GetLocation(key)
				require.ErrorIs(t, err, redis.Nil)
				assert.Nil(t, loc)
			}
		})
	}
}

func TestDBGetLocationErrors(t *testing.T) {
	db, srv := newTestDB(t)
	loc, err := db.GetLocation("missing")
	require.ErrorIs(t, err, redis.Nil)
	assert.Nil(t, loc)

	require.NoError(t, srv.Set("invalid", "not JSON"))
	loc, err = db.GetLocation("invalid")
	require.Error(t, err)
	assert.Nil(t, loc)

	srv.SetError("ERR database unavailable")
	loc, err = db.GetLocation("8.8.8.8")
	require.EqualError(t, err, "ERR database unavailable")
	assert.Nil(t, loc)
}

func TestDBSetLocationError(t *testing.T) {
	db, srv := newTestDB(t)
	srv.SetError("ERR database unavailable")
	err := db.SetLocation(&geoip.Location{IP: "8.8.8.8", Hostname: "example.com"})
	require.EqualError(t, err, "ERR database unavailable")
	assert.Empty(t, srv.Keys())
}
