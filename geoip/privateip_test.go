package geoip

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPrivateIP(t *testing.T) {
	for _, address := range []string{
		"127.0.0.1", "127.255.255.255", "10.0.0.0", "10.255.255.255",
		"172.16.0.0", "172.31.255.255", "192.168.0.0", "192.168.255.255",
		"169.254.0.1", "169.254.255.255", "224.0.0.1", "::1",
		"fe80::1", "febf:ffff::1", "fc00::1", "fdff:ffff::1", "ff02::1",
		"::ffff:192.168.1.1",
	} {
		t.Run(address, func(t *testing.T) {
			ip := net.ParseIP(address)
			require.NotNil(t, ip)
			assert.True(t, IsPrivateIP(ip))
		})
	}
	for _, address := range []string{
		"8.8.8.8", "9.255.255.255", "11.0.0.0", "172.15.255.255",
		"172.32.0.0", "192.167.255.255", "192.169.0.0", "169.253.255.255",
		"169.255.0.0", "2001:4860:4860::8888", "fe7f:ffff::1", "fec0::1",
	} {
		t.Run(address, func(t *testing.T) {
			ip := net.ParseIP(address)
			require.NotNil(t, ip)
			assert.False(t, IsPrivateIP(ip))
		})
	}
	assert.False(t, IsPrivateIP(nil))
}
