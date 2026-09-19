package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

type backend struct {
	url    string
	proxy  *httputil.ReverseProxy
	alive  atomic.Bool
	routed atomic.Int64
}

func main() {
	port := flag.String("port", "8080", "port to listen on")
	list := flag.String("backends", "http://127.0.0.1:8081,http://127.0.0.1:8082", "comma-serpareted bURLs")
	flag.Parse()

	// one shared pool: keep connections to the backends open and reuse them
	// instead of opening a new TCP connection per request
	pool := &http.Transport{MaxIdleConnsPerHost: 100, IdleConnTimeout: 90 * time.Second}

	var backends []*backend
	for _, raw := range strings.Split(*list, ",") {
		u, err := url.Parse(raw)
		if err != nil {
			log.Fatal(err)
		}
		b := &backend{url: raw, proxy: httputil.NewSingleHostReverseProxy(u)}
		b.proxy.Transport = pool
		b.alive.Store(true)
		backends = append(backends, b)
	}

	go func() {
		client := &http.Client{Timeout: time.Second}
		for {
			for _, b := range backends {
				resp, err := client.Get(b.url + "/stats")
				ok := err == nil && resp.StatusCode == http.StatusOK
				if err == nil {
					resp.Body.Close()
				}
				if b.alive.Load() != ok {
					log.Printf("%s alive=%v", b.url, ok)
				}
				b.alive.Store(ok)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	started := time.Now()
	// /admin/pause?backend=8081&secs=10 forwards to that one backend, not round robin
	http.HandleFunc("/admin/pause", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		for _, b := range backends {
			if strings.HasSuffix(b.url, ":"+q.Get("backend")) {
				resp, err := http.Get(b.url + "/admin/pause?secs=" + q.Get("secs"))
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
				defer resp.Body.Close()
				w.WriteHeader(resp.StatusCode)
				io.Copy(w, resp.Body)
				return
			}
		}
		http.Error(w, "unknown backend", http.StatusNotFound)
	})
	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(dashboardHTML))
	})
	http.HandleFunc("/dashboard.json", func(w http.ResponseWriter, r *http.Request) {
		type be struct {
			URL    string          `json:"url"`
			Alive  bool            `json:"alive"`
			Routed int64           `json:"routed"`
			Stats  json.RawMessage `json:"stats"`
		}
		out := struct {
			Port     string `json:"port"`
			Uptime   string `json:"uptime"`
			Backends []be   `json:"backends"`
		}{*port, time.Since(started).Round(time.Second).String(), nil}
		client := &http.Client{Timeout: time.Second, Transport: pool}
		for _, b := range backends {
			e := be{URL: b.url, Alive: b.alive.Load(), Routed: b.routed.Load(), Stats: json.RawMessage("null")}
			if resp, err := client.Get(b.url + "/stats.json"); err == nil {
				var raw json.RawMessage
				if json.NewDecoder(resp.Body).Decode(&raw) == nil {
					e.Stats = raw
				}
				resp.Body.Close()
			}
			out.Backends = append(out.Backends, e)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	})

	var next atomic.Uint64
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		for range backends {
			b := backends[next.Add(1)%uint64(len(backends))]
			if b.alive.Load() {
				b.routed.Add(1)
				b.proxy.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, "no health backends", http.StatusServiceUnavailable)
	})
	log.Printf("lb on :%s -> %s", *port, *list)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
