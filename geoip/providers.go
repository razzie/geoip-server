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
}
