package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"

	"github.com/razzie/geoip-server/geoip"
)

// Server ...
type Server struct {
	mux     http.ServeMux
	db      *DB
	clients []geoip.Client
}

// NewServer creates a new Server
func NewServer(db *DB, clients []geoip.Client) *Server {
	srv := &Server{
		db:      db,
		clients: clients,
	}
	srv.mux.HandleFunc("/", srv.handleRequest)
	srv.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not found", http.StatusNotFound)
	})
	return srv
}

func (srv *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	var hostname string
	if len(r.URL.Path) <= 1 {
		hostname = r.Header.Get("X-REAL-IP")
		if len(hostname) == 0 {
			hostname, _, _ = net.SplitHostPort(r.RemoteAddr)
		}
	} else {
		hostname = r.URL.Path[1:]
	}

	if srv.db != nil {
		if loc, _ := srv.db.GetLocation(hostname); loc != nil {
			writeLocation(w, loc)
			return
		}
	}

	var err error
	var loc *geoip.Location

	for _, c := range srv.clients {
		loc, err = c.GetLocation(r.Context(), hostname)
		if err != nil {
			log.Println(c.Provider(), ":", err)
			continue
		}

		log.Println(c.Provider(), ":", loc.String())
		_ = srv.db.SetLocation(loc)
		writeLocation(w, loc)
		return
	}

	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.mux.ServeHTTP(w, r)
}

func writeLocation(w http.ResponseWriter, loc *geoip.Location) {
	json, _ := json.MarshalIndent(loc, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}
