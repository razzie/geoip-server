package geoip

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
