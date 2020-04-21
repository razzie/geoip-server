package geoip

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/asaskevich/govalidator"
	"github.com/thedevsaddam/gojsonq"
)

// Provider contains the URL template to a 3rd party IP geolocation API
// and the mappings from the provider's json fields to Location fields
type Provider struct {
	TemplateURL string
	Mappings    Location
}

func (p *Provider) doRequest(ctx context.Context, hostname string, params ...interface{}) (*Location, error) {
	if !govalidator.IsHost(hostname) {
		return nil, fmt.Errorf("not a hostname: %s", hostname)
	}

	ip := getIP(hostname)
	if ip == nil {
		return nil, fmt.Errorf("failed to get IP from: %s", hostname)
	}

	if IsPrivateIP(ip) {
		return nil, fmt.Errorf("IP %s is private", ip.String())
	}

	apiurl := fmt.Sprintf(p.TemplateURL, append([]interface{}{ip.String()}, params...)...)
	req, _ := http.NewRequest("GET", apiurl, nil)
	req.Header.Set("User-Agent", browser.Random())
	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}

	var loc Location
	p.doMapping(&loc, resp.Body)

	loc.IP = ip.String()

	if govalidator.IsDNSName(hostname) {
		loc.Hostname = hostname
	} else {
		hostnames, _ := net.LookupAddr(hostname)
		if len(hostnames) > 0 {
			loc.Hostname = strings.TrimRight(hostnames[0], ".")
		}
	}

	return &loc, nil
}

func (p *Provider) doMapping(loc *Location, data io.Reader) {
	jq := gojsonq.New().Reader(data)

	m := func(fieldName string) string {
		if len(fieldName) == 0 {
			return fieldName
		}

		v := jq.Find(fieldName)
		jq.Reset()
		return fmt.Sprint(v)
	}

	loc.CountryCode = m(p.Mappings.CountryCode)
	loc.Country = m(p.Mappings.Country)
	loc.RegionCode = m(p.Mappings.RegionCode)
	loc.Region = m(p.Mappings.Region)
	loc.City = m(p.Mappings.City)
	loc.TimeZone = m(p.Mappings.TimeZone)
	loc.ISP = m(p.Mappings.ISP)
}

func getIP(hostname string) net.IP {
	if govalidator.IsDNSName(hostname) {
		ips, _ := net.LookupIP(hostname)
		if len(ips) > 0 {
			return ips[0]
		}
		return nil
	}

	return net.ParseIP(hostname)
}
