package geoip

// Providers is a map of known providers
var Providers = map[string]*Provider{
	"freegeoip.app": &Provider{
		TemplateURL: "https://freegeoip.app/json/%s",
		Mappings: Location{
			CountryCode: "country_code",
			Country:     "country_name",
			RegionCode:  "region_code",
			Region:      "region_name",
			City:        "city",
			TimeZone:    "time_zone",
		},
	},
	"db-ip.com": &Provider{
		TemplateURL: "http://api.db-ip.com/v2/free/%s",
		Mappings: Location{
			CountryCode: "countryCode",
			Country:     "countryName",
			RegionCode:  "stateProvCode",
			Region:      "stateProv",
			City:        "city",
		},
	},
	"keycdn.com": &Provider{
		TemplateURL: "https://tools.keycdn.com/geo.json?host=%s",
		Mappings: Location{
			CountryCode: "data.geo.country_code",
			Country:     "data.geo.country_name",
			RegionCode:  "data.geo.region_code",
			Region:      "data.geo.region_name",
			City:        "data.geo.city",
			TimeZone:    "data.geo.time_zone",
			ISP:         "data.geo.isp",
		},
	},
	"ip-api.com": &Provider{
		TemplateURL: "http://ip-api.com/json/%s",
		Mappings: Location{
			CountryCode: "countryCode",
			Country:     "country",
			RegionCode:  "region",
			Region:      "regionName",
			City:        "city",
			TimeZone:    "timezone",
			ISP:         "isp",
		},
	},
	"ipinfo.io": &Provider{
		TemplateURL: "http://ipinfo.io/%s?token=%s",
		Mappings: Location{
			Country:    "country",
			RegionCode: "region",
			City:       "city",
			TimeZone:   "timezone",
			ISP:        "org",
		},
	},
}
