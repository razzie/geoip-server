package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/razzie/geoip-server/geoip"
)

func main() {
	port := flag.Int("port", 8080, "http port")
	provider := flag.String("provider", "ip-api.com", "IP geolocation data provider name")
	flag.Parse()

	client := geoip.Providers[*provider].NewClient()
	serveLocation := func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) <= 1 {
			return
		}

		loc, err := client(r.Context(), r.URL.Path[1:])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json, _ := json.MarshalIndent(loc, "", "  ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	}

	http.HandleFunc("/", serveLocation)
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not found", http.StatusNotFound)
	})
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}
