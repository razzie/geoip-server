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
func NewDB(addr, password string, db int) (*DB, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	err := client.Ping().Err()
	if err != nil {
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
	err = json.Unmarshal([]byte(data), &loc)
	if err != nil {
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

	err = db.client.Set(loc.IP, string(data), db.ExpirationTime).Err()
	if err != nil {
		return err
	}

	if len(loc.Hostname) > 0 {
		err = db.client.Set(loc.Hostname, string(data), db.ExpirationTime).Err()
		if err != nil {
			return err
		}
	}

	return nil
}
