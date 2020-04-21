package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/razzie/geoip-server/geoip"
)

func main() {
	redisAddr := flag.String("redis-addr", "localhost:6379", "Redis hostname:port")
	redisPw := flag.String("redis-pw", "", "Redis password")
	redisDb := flag.Int("redis-db", 0, "Redis database (0-15)")
	port := flag.Int("port", 8080, "http port")
	providers := flag.String("provider", "ip-api.com", "provider list, eg: \"ipinfo.io xxtokenxx, ip-api.com, freegeoip.app\"")
	listProviders := flag.Bool("list-providers", false, "List the available providers and then exit")
	flag.Parse()

	if *listProviders {
		for p := range geoip.Providers {
			fmt.Println(p)
		}
		return
	}

	db, err := NewDB(*redisAddr, *redisPw, *redisDb)
	if err != nil {
		log.Println("failed to connect to database:", err)
		db = nil
	}

	clients := geoip.GetClients(*providers)
	serveLocation := func(w http.ResponseWriter, r *http.Request) {
		var hostname string
		if len(r.URL.Path) <= 1 {
			hostname = r.Header.Get("X-REAL-IP")
			if len(hostname) == 0 {
				hostname, _, _ = net.SplitHostPort(r.RemoteAddr)
			}
		} else {
			hostname = r.URL.Path[1:]
		}

		if db != nil {
			if loc, _ := db.GetLocation(hostname); loc != nil {
				loc.Serve(w)
				return
			}
		}

		for _, c := range clients {
			loc, err := c.GetLocation(r.Context(), hostname)
			if err != nil {
				log.Println(c.Provider(), ":", err)
				continue
			}

			_ = db.SetLocation(loc)
			loc.Serve(w)
			return
		}

		http.Error(w, "All providers failed", http.StatusInternalServerError)
	}

	http.HandleFunc("/", serveLocation)
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not found", http.StatusNotFound)
	})
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}
