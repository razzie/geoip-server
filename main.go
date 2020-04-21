package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
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
		var hostname string
		if len(r.URL.Path) <= 1 {
			hostname, _, _ = net.SplitHostPort(r.RemoteAddr)
		} else {
			hostname = r.URL.Path[1:]
		}

		loc, err := client(r.Context(), hostname)
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
