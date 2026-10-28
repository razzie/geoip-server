package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/razzie/geoip-server/geoip"
)

func main() {
	redisConnStr := flag.String("redis", "redis://localhost:6379", "Redis connection string")
	redisFrom := flag.String("redis-from", "", "URL to GET a Redis connection string from (text/uri-list); -redis takes precedence")
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

	redisSpecified := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "redis" {
			redisSpecified = true
		}
	})
	connStr, err := redisConnectionString(*redisConnStr, *redisFrom, redisSpecified)
	var db *DB
	if err == nil {
		db, err = NewDB(connStr)
	}
	if err != nil {
		log.Println("failed to connect to database:", err)
		db = nil
	}

	clients := geoip.GetClients(*providers)
	server := NewServer(db, clients)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), server))
}

func redisConnectionString(connStr, from string, explicitlySet bool) (string, error) {
	if explicitlySet || from == "" {
		return connStr, nil
	}

	req, err := http.NewRequest(http.MethodGet, from, nil)
	if err != nil {
		return "", fmt.Errorf("create Redis connection string request: %w", err)
	}
	req.Header.Set("Accept", "text/uri-list")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch Redis connection string: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("fetch Redis connection string: HTTP %s", resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		uri := strings.TrimSpace(scanner.Text())
		if uri != "" && !strings.HasPrefix(uri, "#") {
			return uri, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read Redis connection string: %w", err)
	}
	return "", fmt.Errorf("Redis connection string response contains no URIs")
}
