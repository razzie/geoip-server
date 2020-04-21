package geoip

// Location contains geographical location and other data of an IP address or hostname
type Location struct {
	CountryCode string `json:"country_code"`
	Country     string `json:"country"`
	RegionCode  string `json:"region_code"`
	Region      string `json:"region"`
	City        string `json:"city"`
	TimeZone    string `json:"timezone"`
	ISP         string `json:"isp,omitempty"`
	Hostname    string `json:"hostname,omitempty"`
}
