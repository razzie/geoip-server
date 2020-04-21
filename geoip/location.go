package geoip

import (
	"encoding/json"
	"net/http"
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

// Serve serves the location over http
func (loc *Location) Serve(w http.ResponseWriter) {
	json, _ := json.MarshalIndent(loc, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}
