package geoip

import (
	"github.com/biter777/countries"
)

// Location contains geographical location and other data of an IP address or hostname
type Location struct {
	IP          string `json:"ip"`
	Hostname    string `json:"hostname,omitempty"`
	CountryCode string `json:"country_code"`
	Country     string `json:"country"`
	RegionCode  string `json:"region_code"`
	Region      string `json:"region"`
	City        string `json:"city"`
	TimeZone    string `json:"timezone"`
	ISP         string `json:"isp,omitempty"`
}

func (loc *Location) fixCountry() {
	if len(loc.Country) == 0 && len(loc.CountryCode) > 0 {
		loc.Country = countries.ByName(loc.CountryCode).String()
	}
	if len(loc.CountryCode) == 0 && len(loc.Country) > 0 {
		loc.CountryCode = countries.ByName(loc.Country).Alpha2()
	}
}
