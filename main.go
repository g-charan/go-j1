package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
	pausedUntil atomic.Int64 // unix seconds; /admin/pause fakes a crash until then
	cpuPct      atomic.Int64 // phone-wide CPU busy %, sampled from /proc/stat
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

func paused() bool { return time.Now().Unix() < pausedUntil.Load() }

// pause makes this backend look dead for a few seconds: /stats and /user
// answer 503 so the balancer's health check fails and traffic moves away.
func pause(w http.ResponseWriter, r *http.Request) {
	secs, _ := strconv.Atoi(r.URL.Query().Get("secs"))
	if secs < 1 || secs > 15 {
		secs = 10
	}
	pausedUntil.Store(time.Now().Unix() + int64(secs))
	fmt.Fprintf(w, "paused for %ds\n", secs)
}

// sampleCPU keeps cpuPct updated from /proc/stat (busy jiffies / total jiffies).
func sampleCPU() {
	var prevBusy, prevTotal int64
	for {
		if b, err := os.ReadFile("/proc/stat"); err == nil {
			f := strings.Fields(strings.SplitN(string(b), "\n", 2)[0])
			var total, idle int64
			for i := 1; i < len(f) && i <= 8; i++ {
				v, _ := strconv.ParseInt(f[i], 10, 64)
				total += v
				if i == 4 || i == 5 { // idle, iowait
					idle += v
				}
			}
			busy := total - idle
			if prevTotal > 0 && total > prevTotal {
				cpuPct.Store(100 * (busy - prevBusy) / (total - prevTotal))
			}
			prevBusy, prevTotal = busy, total
		}
		time.Sleep(2 * time.Second)
	}
}

func user(w http.ResponseWriter, r *http.Request) {
	if paused() {
		http.Error(w, "backend paused", http.StatusServiceUnavailable)
		return
	}
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
	CPU       int64  `json:"cpu_pct"`
	Load      string `json:"load"`
}

func snap() snapshot {
	load := ""
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		load = strings.Join(strings.Fields(string(b))[:1], "")
	}
	return snapshot{*port, time.Since(started).Round(time.Second).String(), hits.Load(),
		cacheHits.Load(), cacheMiss.Load(), cacheShared.Load(),
		cache.len(), *cacheSize, cache.evictions.Load(), cache.expired.Load(),
		cpuPct.Load(), load}
}

func stats(w http.ResponseWriter, r *http.Request) {
	if paused() {
		http.Error(w, "backend paused", http.StatusServiceUnavailable)
		return
	}
	s := snap()
	fmt.Fprintf(w, "uptime: %s\nrequests: %d\ncache hits: %d\ncache misses: %d\ncache shared: %d\ncache size: %d/%d\ncache evictions: %d\ncache expired: %d\n",
		s.Uptime, s.Requests, s.Hits, s.Misses, s.Shared, s.Size, s.Max, s.Evictions, s.Expired)
}

func statsJSON(w http.ResponseWriter, r *http.Request) {
	if paused() {
		http.Error(w, "backend paused", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snap())
}

func main() {
	flag.Parse()
	cache = newLRU(*cacheSize, *cacheTTL)
	go sampleCPU()
	http.HandleFunc("/", hello)
	http.HandleFunc("/admin/pause", pause)
	http.HandleFunc("/user/", user)
	http.HandleFunc("/stats", stats)
	http.HandleFunc("/stats.json", statsJSON)
	log.Printf("listening on :%s (cache=%v)", *port, *useCache)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
