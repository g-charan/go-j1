package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

var (
	port      = flag.String("port", "8080", "port to listen on")
	useCache  = flag.Bool("cache", false, "cache user lookups")
	cacheSize = flag.Int("cache-size", 1000, "max cached users before evicting")
	cacheTTL  = flag.Duration("cache-ttl", time.Minute, "how long a cached user stays valid")

	started = time.Now()
	hits    atomic.Int64

	cache       *lru
	inflight    = newFlight()
	cacheHits   atomic.Int64
	cacheMiss   atomic.Int64
	cacheShared atomic.Int64
)

func hello(w http.ResponseWriter, r *http.Request) {
	hits.Add(1)
	host, _ := os.Hostname()
	fmt.Fprintf(w, "hello from %s:%s, you asked for %s\n", host, *port, r.URL.Path)
}

// pretend this is a database 50ms away
func lookupUser(id string) string {
	time.Sleep(50 * time.Millisecond)
	return fmt.Sprintf(`{"id":%q,"name":"user-%s"}`, id, id)
}

func user(w http.ResponseWriter, r *http.Request) {
	hits.Add(1)
	id := strings.TrimPrefix(r.URL.Path, "/user/")

	if *useCache {
		if v, ok := cache.get(id); ok {
			cacheHits.Add(1)
			fmt.Fprintln(w, v)
			return
		}
	}

	v, leader := inflight.do(id, func() string { return lookupUser(id) })
	if leader {
		cacheMiss.Add(1)
		if *useCache {
			cache.set(id, v)
		}
	} else {
		cacheShared.Add(1) // waited on another request's lookup
	}
	fmt.Fprintln(w, v)
}

type snapshot struct {
	Port      string `json:"port"`
	Uptime    string `json:"uptime"`
	Requests  int64  `json:"requests"`
	Hits      int64  `json:"cache_hits"`
	Misses    int64  `json:"cache_misses"`
	Shared    int64  `json:"cache_shared"`
	Size      int    `json:"cache_size"`
	Max       int    `json:"cache_max"`
	Evictions int64  `json:"cache_evictions"`
	Expired   int64  `json:"cache_expired"`
}

func snap() snapshot {
	return snapshot{*port, time.Since(started).Round(time.Second).String(), hits.Load(),
		cacheHits.Load(), cacheMiss.Load(), cacheShared.Load(),
		cache.len(), *cacheSize, cache.evictions.Load(), cache.expired.Load()}
}

func stats(w http.ResponseWriter, r *http.Request) {
	s := snap()
	fmt.Fprintf(w, "uptime: %s\nrequests: %d\ncache hits: %d\ncache misses: %d\ncache shared: %d\ncache size: %d/%d\ncache evictions: %d\ncache expired: %d\n",
		s.Uptime, s.Requests, s.Hits, s.Misses, s.Shared, s.Size, s.Max, s.Evictions, s.Expired)
}

func statsJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snap())
}

func main() {
	flag.Parse()
	cache = newLRU(*cacheSize, *cacheTTL)
	http.HandleFunc("/", hello)
	http.HandleFunc("/user/", user)
	http.HandleFunc("/stats", stats)
	http.HandleFunc("/stats.json", statsJSON)
	log.Printf("listening on :%s (cache=%v)", *port, *useCache)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
