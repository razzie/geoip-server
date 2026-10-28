package geoip

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocationString(t *testing.T) {
	tests := []struct {
		name string
		loc  Location
		want string
	}{
		{"empty", Location{}, ""},
		{"country only", Location{Country: "Hungary", Region: "Budapest"}, "Hungary"},
		{"city and country", Location{City: "Budapest", Country: "Hungary"}, "Budapest (Hungary)"},
		{"region name", Location{City: "Austin", Country: "United States", Region: "Texas", RegionCode: "TX"}, "Austin (United States, Texas)"},
		{"region code fallback", Location{City: "Austin", Country: "United States", RegionCode: "TX"}, "Austin (United States, TX)"},
		{"region matches city", Location{City: "Budapest", Country: "Hungary", Region: "Budapest"}, "Budapest (Hungary)"},
		{"region code matches city", Location{City: "Budapest", Country: "Hungary", RegionCode: "Budapest"}, "Budapest (Hungary)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.loc.String())
		})
	}
}

func TestLocationFixCountry(t *testing.T) {
	tests := []struct {
		name string
		loc  Location
		want Location
	}{
		{"country from code", Location{CountryCode: "HU"}, Location{CountryCode: "HU", Country: "Hungary"}},
		{"code from country", Location{Country: "Hungary"}, Location{CountryCode: "HU", Country: "Hungary"}},
		{"preserve supplied values", Location{CountryCode: "HU", Country: "Custom name"}, Location{CountryCode: "HU", Country: "Custom name"}},
		{"empty", Location{}, Location{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.loc.fixCountry()
			assert.Equal(t, tt.want, tt.loc)
		})
	}
}
