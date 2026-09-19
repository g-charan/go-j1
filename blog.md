---
title: I turned a $30 Galaxy J1 into a load-balanced, cached web server to learn system design
published: false
tags: go, android, systemdesign, learning
---

I have been studying system design from books and diagrams. Load balancers, caches, health checks, eviction policies. It all made sense on paper and none of it felt real. So I decided to build the smallest possible real thing, on the worst possible hardware, and measure everything.

I had a Samsung Galaxy J1 (Verizon) lying around. 1 GB of RAM, a 32-bit ARM chip, Android 5.1, locked bootloader, no root. I had already wiped it down to the bare OS. Could it be a server?

It can. By the end of one afternoon it was running a load balancer, two application servers, and a bounded LRU cache, and I had a table of numbers that taught me more than a month of reading.

## Part 1: hello from a phone

A server is just a program that listens on the network and answers. The machine does not matter. The J1 has a Linux kernel and Wi-Fi, which is enough.

The trick is that you do not install anything on the phone. Go cross-compiles a single static binary on my Mac that runs on the phone's chip. Then `adb`, the Android Debug Bridge, copies it over and runs it.

```go
func hello(w http.ResponseWriter, r *http.Request) {
	host, _ := os.Hostname()
	fmt.Fprintf(w, "hello from %s, you asked for %s\n", host, r.URL.Path)
}

func main() {
	http.HandleFunc("/", hello)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

```bash
GOOS=linux GOARCH=arm GOARM=7 go build -o server .
adb push server /data/local/tmp/
adb shell "chmod 755 /data/local/tmp/server"
adb shell "trap '' HUP; /data/local/tmp/server > /data/local/tmp/server.log 2>&1 &"
```

Port 8080 because a non-root user cannot open ports below 1024. `/data/local/tmp` is the one folder on an unrooted phone where you may execute files. `trap '' HUP` makes the server ignore the hang-up signal the phone sends when adb disconnects, so I can unplug the cable.

Things that went wrong, in order:

- `adb push` failed with `device unauthorized`. The phone was waiting for me to tap "Allow USB debugging" on its screen.
- `ip addr` prints `192.168.0.142/24`. I pasted the `/24` into curl. It is the subnet mask, not part of the address.
- `nohup server &` inside `adb shell` was a race. adb closes the session milliseconds after the `&`, and if the hang-up signal arrives before nohup is ready, the server dies with an empty log. Half my deploys silently failed until I understood that.
- Android 5.1 has no `pkill`, `pgrep`, `awk`, `cut`, `tail` or `tr`. My "kill the old server" line was silently doing nothing. I rewrote it as `ps | grep tmp/ | while read user pid rest; do kill $pid; done`.
- `ps` shows a process by the exact path it was started with. I started servers as `./server`, then grepped for `tmp/server`. Nothing matched, and I accumulated orphaned servers holding every port.

```
$ curl http://192.168.0.142:8080/test
hello from localhost, you asked for /test
```

A phone from 2015 answering HTTP on my desk, unplugged, on Wi-Fi and a charger.

## Part 2: a load balancer, in 60 lines

One server can only do so much, and if it dies everything is down. So I ran two copies on ports 8081 and 8082 and wrote a balancer for port 8080. Go's standard library does the forwarding (`httputil.ReverseProxy`), so the balancer is only two ideas:

- **Round robin.** Request 1 goes to A, request 2 to B, request 3 to A.
- **Health checks.** A goroutine asks each backend `/stats` every two seconds. If it fails, stop routing there. When it recovers, resume.

The first thing the balancer did was mark backend 8082 dead. The backend was fine. I had typed `http//127.0.0.1:8082` in the config, no colon. The system stayed up on one backend and the only way I found out was reading the log line the health checker printed when the state changed. Lesson: log state changes, not steady state.

Then I killed a backend on purpose. The next one or two requests failed, the health check noticed, and everything flowed to the survivor. That gap is a real design parameter. A shorter check interval catches failures faster and burns more CPU on checking. That trade-off is the whole job.

## Part 3: a cache, with numbers

A cache only makes sense when something is slow, and you only believe it when you measure. So I added `/user/<id>`, which "looks up" a user in a fake database that takes 50 ms, and an in-memory cache in front of it: a Go map behind a read-write lock. I load tested from the Mac with `hey`.

| Test through the balancer | Requests/sec | 99% latency |
|---|---|---|
| `/` trivial, 50 concurrent | 280 | 481 ms |
| `/user/42` cache off, 50 concurrent | 251 | 403 ms |
| `/user/42` cache off, 5 concurrent | 67 | 190 ms |
| `/user/42` cache on, 5 concurrent | 223 | 96 ms |
| `/user/42` cache on, 50 concurrent | 290 | 510 ms |
| `/` direct to one backend, 50 concurrent | 1930 | 64 ms |

Four things I did not expect:

1. **Concurrency hides latency.** With 50 requests in flight, the 50 ms database barely mattered. The cache only showed its 3x gain at 5 concurrent. Throughput and latency are different problems.
2. **The balancer was the bottleneck, by 7x.** One backend alone did 1930 requests/sec. Through the balancer, 280.
3. **Cache stampede.** Each backend showed 2 or 3 misses for one user, not 1. Several concurrent first requests all missed before the first had filled the cache. Scale that to a hot key expiring under real traffic and it is the burst that takes down a database.
4. **Two caches, not one.** Each backend missed independently. Ten backends would mean ten misses per user. That is the argument for a shared cache node.

## Part 4: fixing what the numbers showed

**Bounded cache with LRU eviction.** The map grew forever. I replaced it with a map plus a doubly linked list (`container/list`): every get or set moves the entry to the front, and when the count exceeds the limit the back entry, the least recently used, is removed. I set the limit to 200 and requested 1000 distinct users: each backend ended at 200/200 with exactly 300 evictions. One consequence I had not appreciated: an LRU turns reads into writes, because every hit mutates the list, so the read-write lock had to become a plain mutex.

**TTL.** Nothing in my fake database changes, but the cache does not know that. The day I add a write path, every server would serve the old value until eviction. Every entry now expires after a configurable time.

**Single-flight.** The first request to miss on a key does the lookup. Everyone else arriving for the same key meanwhile waits for that result. Forty simultaneous first requests for a new user: 1 database call, 39 waited.

**Connection pooling in the balancer.** Go's default transport keeps 2 idle connections per host, so under 50 concurrent requests the balancer was opening and closing TCP connections constantly. One shared transport with 100 idle connections per backend, plus removing a per-request log write to flash:

| Through the balancer, 50 concurrent | Before | After |
|---|---|---|
| `/` | 280 rps | 396 rps |
| `/user/42` cached | 290 rps | 404 rps |

A 40% gain, not the 7x I hoped for. `top` on the phone explained it: 99% CPU, the balancer alone taking 39% and the kernel 34% moving packets. Every request through the balancer is handled twice on the same four slow cores. The rest of the gap is not a code problem. It is the balancer living on the same box as the things it balances. The fix is a second machine, which is the next post.

## Making it public, from the phone

The phone serves a live dashboard at `/dashboard`: request flow, per-backend routing counts, cache hits, misses, shared lookups, evictions and expiries, refreshed every second, with a button that fires 100 requests so you can watch the counters move.

First attempt at a public URL was a Cloudflare quick tunnel from my Mac pointed at the phone. One command, worked in a minute. Then I asked myself the question this whole lab exists for: what if the Mac is down? The link dies. The Mac was a single point of failure.

So the tunnel had to run on the phone. Cloudflare ships a 32-bit ARM build, it ran, and it failed on the first DNS lookup. Go programs on Linux read `/etc/resolv.conf` to find a DNS server. Android has no such file, so Go falls back to asking `localhost:53`, where nothing listens. Go's own source has a comment: "DNS requests don't work on Android". Our server never noticed because it never resolves a name. I could not run a forwarder on port 53 either, that needs root.

Cloudflare's anonymous quick tunnel has no way around that. ngrok's agent does: `dns_resolver_ips` tells it which DNS server to use by IP, and `crl_noverify` skips one side request that still used the broken resolver. With those two lines in its config and a free account, the phone registered its own tunnel and got a fixed hostname:

https://wispy-uplifted-recycler.ngrok-free.dev/dashboard

Nothing but the phone, a charger and Wi-Fi. Tailscale, my first thought, needs Android 8 and builds a private network rather than a public link.

## What I actually learned

Everything above is in every system design book. The difference is that I now have a number attached to each concept, a failure I caused with my own hands, and a phone on my desk that still answers. When someone asks me about health check intervals, I remember the two failed requests. When they ask about cache eviction, I remember 300.

Next: move the balancer to my Mac, make the phone one backend and the Mac another, and pull the phone's Wi-Fi to watch failover happen across a real network. That is where this stops being one box and starts being distributed systems.

Code: https://github.com/g-charan/go-j1
