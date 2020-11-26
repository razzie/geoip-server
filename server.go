package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/razzie/geoip-server/geoip"
	"github.com/razzie/reqip"
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
	srv.mux.HandleFunc("/favicon.ico", writeFavicon)
	return srv
}

func (srv *Server) getLocation(ctx context.Context, hostname string) (loc *geoip.Location, err error) {
	if srv.db != nil {
		if loc, _ = srv.db.GetLocation(hostname); loc != nil {
			return
		}
	}

	for _, c := range srv.clients {
		loc, err = c.GetLocation(ctx, hostname)
		if err != nil {
			log.Println(c.Provider(), ":", hostname, ":", err)
			continue
		}

		//log.Println(c.Provider(), ":", hostname, ":", loc.String())
		if srv.db != nil {
			_ = srv.db.SetLocation(loc)
		}
		return loc, nil
	}

	return
}

func (srv *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	var hostname string
	if len(r.URL.Path) <= 1 {
		hostname = reqip.GetClientIP(r)
	} else {
		hostname = r.URL.Path[1:]
	}

	defer srv.logRequest(r)

	loc, err := srv.getLocation(r.Context(), hostname)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeLocation(w, loc)
}

func (srv *Server) logRequest(r *http.Request) {
	ip := reqip.GetClientIP(r)

	if loc, _ := srv.getLocation(context.Background(), ip); loc != nil {
		log.Println(ip, loc.String(), r.RequestURI)
	} else {
		log.Println(ip, r.RequestURI)
	}
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.mux.ServeHTTP(w, r)
}

var favicon, _ = base64.StdEncoding.DecodeString("" +
	"iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAMAAACdt4HsAAAABGdBTUEAALGPC/xhBQAAACBjSFJNAAB6" +
	"JgAAgIQAAPoAAACA6AAAdTAAAOpgAAA6mAAAF3CculE8AAACuFBMVEUAAADbSjf/Sy88R0pHR0nsSjUi" +
	"Rk21STywSTyOSECcST9BR0qdST+ASEKBSEK/STpDR0oAQmTcSjc0R0voSjU+R0rfSjfdSjekST6gST5F" +
	"R0nbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfb" +
	"SjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfb" +
	"SjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjdHR0lHR0lHR0lHR0nbSjfbSjfb" +
	"SjfbSjdHR0lHR0lHR0lHR0lHR0lHR0nbSjfbSjfbSjfbSjfbSjdHR0lHR0lHR0nbSjfbSjfbSjfbSjfb" +
	"SjdHR0lHR0nbSjfbSjfbSjdHR0lHR0nbSjfbSjdHR0lHR0nbSjfbSjfbSjdHR0lHR0nbSjfbSjfbSjdH" +
	"R0lHR0nbSjfbSjdHR0nbSjfbSjdHR0nbSjdHR0lHR0lHR0lHR0nbSjfbSjfbSjfbSjdHR0lHR0nbSjfb" +
	"SjdHR0lHR0nbSjfbSjfbSjfbSjdHR0nbSjfbSjfbSjdHR0lHR0nbSjdHR0lHR0nbSjfbSjfbSjdHR0nb" +
	"SjfbSjdHR0lHR0nbSjfbSjdHR0lHR0lHR0lHR0nbSjdHR0lHR0lHR0nbSjfbSjfbSjdHR0lHR0lHR0lH" +
	"R0nbSjfbSjdHR0nbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfb" +
	"SjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfbSjfb" +
	"SjfbSjfbSjfbSjdHR0n////VFNPRAAAA5XRSTlMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEI1mQ" +
	"uNvr/AxHltbzBkmw8JTuRcgCWuL4AV7q9dGddFtMTuapUhou0/7ATQkO95MbqmN/CgghP1J+y/uJEVqn" +
	"3e/1CMwdahM3tfevbRG/N0/ZQu+IQd/kLxzFs6cHeP3c/Wwd0uw+9uMhfxcEmpkD1VhcFgag2hkBk1T2" +
	"5yhzKUryQO49ELw8vhBb1PEMoguyIyav+1UUcdQk3rlqpsviorUbykjJx44tA3DfTx8NBSD0nHNiQ43O" +
	"XaDYZqMqlSJE6H0UOeD5D9lnugOP6wAAAAFiS0dE57ZrapMAAAAHdElNRQfkBBYADSBPMwMyAAAEu0lE" +
	"QVRYw52X+V8TZxCHGR2PVut9G0QTQwBFo2KIgqICYogCAgVPSj1axQM1HgieiNgreCPWk4qogAreCiKK" +
	"iEY8q9Z6V03+jr777gZC8u5mw/zCh535Pnnn3Xdn5vXyYhgQ66/wHuAzcJBSOWigzwBvRX/umZc840JV" +
	"g9W+GmujaXzVg1VyESTMzz9giNXJhgT4+8kikCDF0EArwwKHKmQQSMiw4VYRGz7MLQFAO2Kkg2SURjPK" +
	"4d+RI7TSBICg0Tp7dLB+zNiQ0NCQsWP0wfZnutFBUgSywHHj7ZsWNmEifXfkjU6cEGbf1PHjpLIAmBTO" +
	"x0VETo4C/r3RP1GTIyN4T/gkcQDAlDA+yhBtBGjVGgVr3QrAGG3gfVOniBIApvFBwTGx0KYtYtz0+ISE" +
	"+OlxiG3bQGwivxOGaWIA8ivf05Ck5Chohzhj5qzZc+bOnTN71swZiO0gJTmJun9IFSEA/MhHBMyD9ojz" +
	"Fyy0CbZwwXzE9jDvJ+pW/iwGWLSYf9lp8A3ikqU2B1u6BPFbSOOPyOJFwF7AsuXUnx4EHXDFSlszW7kC" +
	"O0BQOg1Yvoy5BIAY6l5lgo64eo3Nydasxo5gWkVDYtgA1VrqXZcBiOsznQGZ6xEhYx0NWatiAlKzOOeG" +
	"jQC4abPNxTZvQoCNG7iYrFQmIG0LTXArfIcJ2a6A7ATsBFvpNm1JYwJCaQnalkMy2G5j2HaSQ842WqBC" +
	"mQBvegx9cwF3/MIC/PobQu7v9DD+wQSY6ZevzwPcuYsF2LUTIU9Pq4RZYgW79wDu3ccC7NuLsGc3XYE3" +
	"E7A/n3MeKCB7cJAFOEj24M8DXEz+fibg0GFhgzrjkaOu+qPx2FnY6MOHmIBjfC1OJOfgeKEroPA4OQeJ" +
	"fH3+iwkooMuzniiCLnjS5SBkn8QuUHTCKqTJAuQWU+8pf+iKp884A86cxq7gf4qGFOcyASWlQs0qgU5Y" +
	"5pREYRk5hyVT+YhS9rcA6UIjDIFu3fHsufNN8vPnzmL3bhAitMt0kc+5XCibxSbo0RMrLlwUPsnMixcq" +
	"sGcPMBULJbdcBHDpslD8r+RAr96IZVevXb9x4/q1q2WIvXtBzhXBffkSu6aBX6W9AVUVAPQh9Tyu4ubN" +
	"ClKUsQ9AQZXdW5khVlT5s8hZ9S3Slfr249tCv76kO92qsfvyy0Wrct7txjaqu1N7Fxrtbu2dxpZpvZ0n" +
	"3hjqlE2d+J66/v4Di1ZreXC/Xn2v6bmyTqIzPdQ7jgMNWT6PoqMf+WQ1OD7VP5Rqjo+VVjemfCzZnY01" +
	"7gA1RukB4YlBWm94IjmiABSVSgNKi9zNOE91UnrdUzdTFoDlmRTgmcX9mKZ4Lq5/rnA7KJJjZ24Q0zf8" +
	"LWvSjH0hBnhRIm/WrX3J1r+slTVtk1W+ymfpDa9kj+uqShbgH5XcGwPA63BXffhrT24cZo2zXmOWf2Uh" +
	"hJR/nQHqFPl6jvCmurm++o0nei6J+ma3lsB6DxLgCZbkiCZ9RLLbb8CVYIxsAkQaPdVzSbx9Z9e/e+th" +
	"AjxB+56fva1J77We6znCB6EXVX1oiZ4jfPzE6T99bJme24Y6Mvn9V9eCDbATPpM7VNjnluo5gunLF1PL" +
	"9VwSX7+6SeB/lxUzLyESsCUAAAAldEVYdGRhdGU6Y3JlYXRlADIwMjAtMDMtMTlUMTA6NTE6MDQrMDA6" +
	"MDACwiffAAAAJXRFWHRkYXRlOm1vZGlmeQAyMDE5LTAxLTA4VDE3OjI1OjEwKzAwOjAwky0vkQAAACB0" +
	"RVh0c29mdHdhcmUAaHR0cHM6Ly9pbWFnZW1hZ2ljay5vcme8zx2dAAAAGHRFWHRUaHVtYjo6RG9jdW1l" +
	"bnQ6OlBhZ2VzADGn/7svAAAAGHRFWHRUaHVtYjo6SW1hZ2U6OkhlaWdodAAxMjhDfEGAAAAAF3RFWHRU" +
	"aHVtYjo6SW1hZ2U6OldpZHRoADEyONCNEd0AAAAZdEVYdFRodW1iOjpNaW1ldHlwZQBpbWFnZS9wbmc/" +
	"slZOAAAAF3RFWHRUaHVtYjo6TVRpbWUAMTU0Njk2ODMxMKq+e/kAAAARdEVYdFRodW1iOjpTaXplADM4" +
	"ODNCBI61NAAAAFh0RVh0VGh1bWI6OlVSSQBmaWxlOi8vL2RhdGEvd3d3cm9vdC93d3cuZWFzeWljb24u" +
	"bmV0L2Nkbi1pbWcuZWFzeWljb24uY24vZmlsZXMvNTUvNTU3ODIzLnBuZ0wFAx0AAAAASUVORK5CYII=")

func writeFavicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(favicon)))
	_, _ = w.Write(favicon)
}

func writeLocation(w http.ResponseWriter, loc *geoip.Location) {
	json, _ := json.MarshalIndent(loc, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}
