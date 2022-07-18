package main

import (
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/razzie/geoip-server/geoip"
)

// DB ...
type DB struct {
	ExpirationTime time.Duration
	client         *redis.Client
}

// NewDB returns a new DB
func NewDB(redisUrl string) (*DB, error) {
	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	if err := client.Ping().Err(); err != nil {
		client.Close()
		return nil, err
	}

	return &DB{
		ExpirationTime: time.Hour,
		client:         client,
	}, nil
}

// GetLocation returns a saved location
func (db *DB) GetLocation(hostname string) (*geoip.Location, error) {
	data, err := db.client.Get(hostname).Result()
	if err != nil {
		return nil, err
	}

	var loc geoip.Location
	if err = json.Unmarshal([]byte(data), &loc); err != nil {
		return nil, err
	}

	return &loc, nil
}

// SetLocation saves a location
func (db *DB) SetLocation(loc *geoip.Location) error {
	data, err := json.Marshal(loc)
	if err != nil {
		return err
	}

	if err = db.client.Set(loc.IP, string(data), db.ExpirationTime).Err(); err != nil {
		return err
	}

	if len(loc.Hostname) > 0 {
		if err = db.client.Set(loc.Hostname, string(data), db.ExpirationTime).Err(); err != nil {
			return err
		}
	}

	return nil
}
