# go-j1

A system design lab running on a 2015 Samsung Galaxy J1 (1 GB RAM, 32-bit ARM, Android 5.1, no root).
Two Go HTTP servers with a bounded LRU cache, a round-robin load balancer with health checks, and a live dashboard. Deployed over `adb`, nothing installed on the phone.

Live: https://wispy-uplifted-recycler.ngrok-free.dev/dashboard (served from the phone, click through ngrok's interstitial)

Write-up: [blog.md](blog.md)

## Layout

```
main.go          backend: /user/<id> (fake 50 ms DB), /stats, /stats.json
cache.go         LRU with TTL + single-flight
cache_test.go    go test -race .
lb/main.go       load balancer: round robin, 2 s health checks, pooled connections
lb/dashboard.go  /dashboard HTML (inline, no deps) + /dashboard.json
deploy.sh        build for ARMv7, kill old processes, push, start all three
```

## Run

Prereqs: Go, `adb` (`brew install go android-platform-tools`), USB debugging enabled on the phone, phone and Mac on the same Wi-Fi.

```bash
./deploy.sh                                   # builds, pushes, restarts everything
adb shell "ip addr show wlan0 | grep 'inet '" # phone IP
open http://<phone-ip>:8080/dashboard
```

Ports: 8080 balancer, 8081 and 8082 backends. Backend flags: `-port`, `-cache`, `-cache-size` (default 1000), `-cache-ttl` (default 1m). Balancer flags: `-port`, `-backends`.

Load test:

```bash
brew install hey
hey -n 2000 -c 50 http://<phone-ip>:8080/user/42
```

Public URL, from the phone itself (no laptop in the path). Go programs cannot resolve DNS on Android 5.1, so the ngrok agent needs its DNS override. One-time setup, then `deploy.sh` restarts it:

```bash
# ngrok-v3-stable-linux-arm.tgz -> adb push ngrok /data/local/tmp/
# /data/local/tmp/ngrok.yml (not in this repo):
#   version: "2"
#   authtoken: <yours>
#   dns_resolver_ips: [1.1.1.1, 8.8.8.8]
#   crl_noverify: true
#   update_check: false
```

## Numbers (through the balancer, 50 concurrent)

| | rps |
|---|---|
| one backend directly | 1930 |
| via balancer, before pooling | 280 |
| via balancer, after pooling | 396 |

The phone is at 99% CPU under load; the balancer alone takes 39%. Next step is moving it off the phone.

## Android 5.1 gotchas

- No `pkill`, `pgrep`, `awk`, `cut`, `tail`, `tr`. Kill by `ps | grep tmp/ | while read u pid rest; do kill $pid; done`.
- `ps` shows the exact path a process was started with. Start by absolute path or your grep will not match.
- `nohup cmd &` from `adb shell` is racy. Use `trap '' HUP` in the shell before starting.
- Only `/data/local/tmp` is executable without root. Ports below 1024 are off limits.
- No `/etc/resolv.conf`, so static Go binaries fall back to `[::1]:53` and every DNS lookup fails. cloudflared's quick tunnel cannot work here; ngrok's `dns_resolver_ips` + `crl_noverify` can.
