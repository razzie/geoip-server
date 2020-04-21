package geoip

import (
	"context"
)

// Client is used to retrieve location data from a provider
type Client func(ctx context.Context, hostname string) (*Location, error)

// NewClient returns a new provider client
func (p *Provider) NewClient(params ...interface{}) Client {
	return func(ctx context.Context, hostname string) (*Location, error) {
		return p.doRequest(ctx, hostname, params...)
	}
}
