package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/razzie/geoip-server/geoip"
)

func main() {
	redisConnStr := flag.String("redis", "redis://localhost:6379", "Redis connection string")
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

	db, err := NewDB(*redisConnStr)
	if err != nil {
		log.Println("failed to connect to database:", err)
		db = nil
	}

	clients := geoip.GetClients(*providers)
	server := NewServer(db, clients)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), server))
}
