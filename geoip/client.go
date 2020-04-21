package geoip

import (
	"context"
)

// Client is used to retrieve location data from a provider
type Client interface {
	GetLocation(ctx context.Context, hostname string) (*Location, error)
}

type client struct {
	provider *Provider
	params   []interface{}
}

func (c *client) GetLocation(ctx context.Context, hostname string) (*Location, error) {
	return c.provider.doRequest(ctx, hostname, c.params...)
}

// NewClient returns a new client to the provider
func (p *Provider) NewClient(params ...interface{}) Client {
	return &client{
		provider: p,
		params:   params,
	}
}
