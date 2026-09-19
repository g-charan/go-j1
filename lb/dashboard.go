package main

// Served at /dashboard. Plain HTML, CSS and JS with no dependencies: the
// phone serves it, so nothing is downloaded from anywhere else. It polls
// /dashboard.json once a second, derives per-second rates client-side, and
// draws an animated request flow, two charts and an event log.
const dashboardHTML = `<!doctype html>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Galaxy J1 server lab</title>
<style>
:root{
  color-scheme:dark;
  --page:#0d0d0d;--surface:#1a1a19;--surface-2:#222221;--border:rgba(255,255,255,.10);
  --ink:#fff;--ink-2:#c3c2b7;--muted:#898781;--grid:#2c2c2a;--axis:#383835;
  --b1:#3987e5;--b2:#d95926;--hit:#199e70;--miss:#c98500;--shared:#d55181;
  --good:#0ca30c;--warn:#fab219;--crit:#d03b3b;
}
*{box-sizing:border-box}
body{margin:0;background:var(--page);color:var(--ink);font:14px/1.5 system-ui,-apple-system,"Segoe UI",sans-serif;padding:20px 16px 48px}
.wrap{max-width:1040px;margin:0 auto}
header{display:flex;flex-wrap:wrap;align-items:baseline;gap:10px 16px;margin-bottom:6px}
h1{font-size:22px;margin:0;letter-spacing:-.01em}
.live{display:inline-flex;align-items:center;gap:6px;font-size:12px;color:var(--ink-2);border:1px solid var(--border);border-radius:999px;padding:2px 10px}
.live i{width:8px;height:8px;border-radius:50%;background:var(--good);box-shadow:0 0 0 0 rgba(12,163,12,.6);animation:pulse 1.6s infinite}
.live.stale i{background:var(--crit);animation:none}
@keyframes pulse{0%{box-shadow:0 0 0 0 rgba(12,163,12,.6)}100%{box-shadow:0 0 0 10px rgba(12,163,12,0)}}
.chips{display:flex;flex-wrap:wrap;gap:6px;margin:0 0 18px;padding:0;list-style:none}
.chips li{font-size:12px;color:var(--ink-2);background:var(--surface);border:1px solid var(--border);border-radius:6px;padding:2px 8px}
.tiles{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:10px;margin-bottom:14px}
.tile{background:var(--surface);border:1px solid var(--border);border-radius:10px;padding:12px 14px;min-width:0}
.tile h3{margin:0 0 4px;font-size:12px;font-weight:500;color:var(--muted)}
.tile .v{font-size:28px;font-weight:600;line-height:1.1;letter-spacing:-.02em}
.tile .v small{font-size:13px;font-weight:400;color:var(--ink-2);margin-left:4px}
.tile .s{font-size:12px;color:var(--ink-2);margin-top:4px;min-height:18px}
.tile .bar{height:4px;background:var(--grid);border-radius:2px;margin-top:8px;overflow:hidden}
.tile .bar b{display:block;height:100%;background:var(--b1);border-radius:2px;transition:width .4s}
.tile .bar.warn b{background:var(--warn)}.tile .bar.crit b{background:var(--crit)}
.card{background:var(--surface);border:1px solid var(--border);border-radius:10px;padding:14px;margin-bottom:14px;min-width:0}
.card h2{font-size:13px;font-weight:600;margin:0 0 2px;color:var(--ink)}
.card p.sub{margin:0 0 10px;font-size:12px;color:var(--muted)}
.flow{width:100%;height:auto;display:block}
.flow text{font:12px system-ui,sans-serif;fill:var(--ink-2)}
.flow .name{font-weight:600;fill:var(--ink)}
.flow .edge{fill:none;stroke:var(--axis);stroke-width:2}
.flow .node{fill:var(--surface-2);stroke:var(--border)}
.flow .node.up{stroke:var(--good);stroke-width:1.5}
.flow .node.down{stroke:var(--crit);stroke-width:2;stroke-dasharray:4 3}
.flow .cap{fill:var(--grid)}
.flow .fill{fill:var(--b1)}
.ctl{display:flex;flex-wrap:wrap;gap:8px;align-items:center;margin:0 0 6px}
button{background:var(--surface-2);color:var(--ink);border:1px solid var(--border);border-radius:8px;padding:8px 12px;font:inherit;font-size:13px;cursor:pointer;transition:background .15s}
button:hover{background:#2b2b29}
button:disabled{opacity:.45;cursor:default}
button.primary{background:#1c5cab;border-color:#2a78d6}button.primary:hover{background:#256abf}
button.danger{border-color:#7a2626}button.danger:hover{background:#3a1a1a}
.hint{font-size:12px;color:var(--muted)}
.charts{display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:14px}
.legend{display:flex;flex-wrap:wrap;gap:12px;font-size:12px;color:var(--ink-2);margin:0 0 6px}
.legend span{display:inline-flex;align-items:center;gap:6px}
.legend i{width:10px;height:10px;border-radius:2px;display:inline-block}
.chart{position:relative}
.chart svg{width:100%;height:auto;display:block}
.chart .grid{stroke:var(--grid);stroke-width:1}
.chart .base{stroke:var(--axis);stroke-width:1}
.chart .tick{font:11px system-ui,sans-serif;fill:var(--muted);font-variant-numeric:tabular-nums}
.chart .lbl{font:11px system-ui,sans-serif;fill:var(--ink-2)}
.chart .line{fill:none;stroke-width:2;stroke-linejoin:round;stroke-linecap:round}
.chart .xh{stroke:var(--ink-2);stroke-width:1;stroke-dasharray:3 3}
.tip{position:absolute;pointer-events:none;background:#000;border:1px solid var(--border);border-radius:6px;padding:6px 8px;font-size:12px;color:var(--ink-2);white-space:nowrap;display:none;z-index:2}
.tip b{color:var(--ink);font-variant-numeric:tabular-nums}
.tip i{width:8px;height:8px;border-radius:2px;display:inline-block;margin-right:6px}
table{width:100%;border-collapse:collapse;font-size:12px;font-variant-numeric:tabular-nums}
th,td{text-align:right;padding:4px 6px;border-bottom:1px solid var(--grid)}
th:first-child,td:first-child{text-align:left}
th{color:var(--muted);font-weight:500}
.bes{display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:10px}
.be{border-left:4px solid var(--good)}
.be.down{border-left-color:var(--crit)}
.be h3{margin:0 0 6px;font-size:13px;display:flex;justify-content:space-between}
.be h3 .st{font-weight:500;font-size:12px;color:var(--good)}.be.down h3 .st{color:var(--crit)}
.kv{display:grid;grid-template-columns:1fr auto;gap:1px 12px;font-size:12px}
.kv span:nth-child(odd){color:var(--muted)}
.kv span:nth-child(even){font-variant-numeric:tabular-nums;color:var(--ink-2)}
.log{list-style:none;margin:0;padding:0;max-height:200px;overflow:auto;font-size:12px}
.log li{display:flex;gap:10px;padding:3px 0;border-bottom:1px solid var(--grid)}
.log time{color:var(--muted);font-variant-numeric:tabular-nums;flex:none}
.log .good{color:var(--good)}.log .crit{color:var(--crit)}.log .warn{color:var(--warn)}
details{color:var(--ink-2);font-size:13px}
summary{cursor:pointer;color:var(--ink);font-weight:600;font-size:13px}
details p{margin:8px 0}
code{background:var(--surface-2);padding:1px 5px;border-radius:4px;font-size:12px;color:var(--ink)}
.sr{position:absolute;left:-9999px}
@media (max-width:560px){.tile .v{font-size:22px}}
</style>

<div class="wrap">
<header>
  <h1>Galaxy J1 server lab</h1>
  <span class="live" id="live"><i></i><span id="livetxt">live</span></span>
</header>
<ul class="chips">
  <li>Samsung Galaxy J1, 2015</li><li>1 GB RAM</li><li>32-bit ARM, 4 cores</li><li>Android 5.1</li><li>no root</li><li>served from the phone</li>
</ul>

<section class="tiles" aria-label="Key numbers">
  <div class="tile"><h3>Requests / s through balancer</h3><div class="v" id="t-rps">0</div><div class="s" id="t-rps-s">nothing in flight</div></div>
  <div class="tile"><h3>Cache hit ratio, last 60 s</h3><div class="v" id="t-hit">&ndash;</div><div class="s" id="t-hit-s">no lookups yet</div><div class="bar"><b id="t-hit-b" style="width:0"></b></div></div>
  <div class="tile"><h3>Latency from this browser</h3><div class="v" id="t-lat">&ndash;</div><div class="s" id="t-lat-s">send some requests</div></div>
  <div class="tile"><h3>Phone CPU</h3><div class="v" id="t-cpu">&ndash;</div><div class="s" id="t-cpu-s"></div><div class="bar" id="t-cpu-bar"><b id="t-cpu-b" style="width:0"></b></div></div>
  <div class="tile"><h3>Backends healthy</h3><div class="v" id="t-up">&ndash;</div><div class="s" id="t-up-s">health check every 2 s</div></div>
  <div class="tile"><h3>Balancer uptime</h3><div class="v" id="t-up-time">&ndash;</div><div class="s">since last deploy</div></div>
</section>

<section class="card">
  <h2>Request flow</h2>
  <p class="sub">Each dot is a real request from the last second. Dots that reach the database are cache misses (50 ms each); the rest were answered from the cache.</p>
  <svg class="flow" viewBox="0 0 840 250" role="img" aria-label="Diagram of requests flowing from you to the load balancer, to two backends, to the database">
    <path id="e-you-lb" class="edge" d="M112,125 L226,125"/>
    <path id="e-lb-b1" class="edge" d="M334,125 C400,125 400,62 470,62"/>
    <path id="e-lb-b2" class="edge" d="M334,125 C400,125 400,188 470,188"/>
    <path id="e-b1-db" class="edge" d="M610,62 C670,62 670,125 730,125"/>
    <path id="e-b2-db" class="edge" d="M610,188 C670,188 670,125 730,125"/>
    <circle cx="70" cy="125" r="40" class="node"/>
    <text x="70" y="121" text-anchor="middle" class="name">You</text>
    <text x="70" y="137" text-anchor="middle" id="f-you">0 sent</text>
    <rect x="226" y="88" width="108" height="74" rx="10" class="node up"/>
    <text x="280" y="110" text-anchor="middle" class="name">Balancer</text>
    <text x="280" y="126" text-anchor="middle">:8080</text>
    <text x="280" y="146" text-anchor="middle" id="f-lb">round robin</text>
    <g id="f-b1"><rect x="470" y="24" width="140" height="76" rx="10" class="node up"/>
      <text x="482" y="44" class="name">Backend :8081</text><text x="482" y="60" id="f-b1-t">0 routed</text>
      <rect x="482" y="68" width="116" height="5" rx="2" class="cap"/><rect x="482" y="68" width="0" height="5" rx="2" class="fill" id="f-b1-c"/><text x="482" y="88" style="font-size:10px" id="f-b1-cl">cache 0/200</text></g>
    <g id="f-b2"><rect x="470" y="150" width="140" height="76" rx="10" class="node up"/>
      <text x="482" y="170" class="name">Backend :8082</text><text x="482" y="186" id="f-b2-t">0 routed</text>
      <rect x="482" y="194" width="116" height="5" rx="2" class="cap"/><rect x="482" y="194" width="0" height="5" rx="2" class="fill" id="f-b2-c" style="fill:var(--b2)"/><text x="482" y="214" style="font-size:10px" id="f-b2-cl">cache 0/200</text></g>
    <ellipse cx="770" cy="98" rx="40" ry="10" class="node"/>
    <path d="M730,98 v54 a40,10 0 0 0 80,0 v-54" class="node"/>
    <text x="770" y="132" text-anchor="middle" class="name">Database</text>
    <text x="770" y="146" text-anchor="middle">fake, 50 ms</text>
    <text x="770" y="180" text-anchor="middle" id="f-db">0 lookups</text>
    <g id="dots"></g>
  </svg>
  <div class="ctl">
    <button class="primary" id="b-rand">Send 100 random users</button>
    <button id="b-same">Send 100 for one user</button>
    <button id="b-sus">Sustained load, 30 s</button>
    <button class="danger" id="b-k1">Knock out :8081 for 10 s</button>
    <button class="danger" id="b-k2">Knock out :8082 for 10 s</button>
  </div>
  <div class="hint" id="hint">Random users fill the cache past its 200-entry limit and force LRU evictions. One user shows hits. Knocking out a backend fails its health check; watch the balancer route around it.</div>
</section>

<section class="charts">
  <div class="card">
    <h2>Throughput per backend</h2>
    <p class="sub">requests per second, last 60 s</p>
    <div class="legend"><span><i style="background:var(--b1)"></i>:8081</span><span><i style="background:var(--b2)"></i>:8082</span></div>
    <div class="chart" id="c-rps"><svg viewBox="0 0 600 170"></svg><div class="tip"></div></div>
  </div>
  <div class="card">
    <h2>Cache outcomes</h2>
    <p class="sub">lookups per second by how they were answered</p>
    <div class="legend"><span><i style="background:var(--hit)"></i>hit</span><span><i style="background:var(--miss)"></i>miss, went to DB</span><span><i style="background:var(--shared)"></i>shared another request's lookup</span></div>
    <div class="chart" id="c-cache"><svg viewBox="0 0 600 170"></svg><div class="tip"></div></div>
  </div>
</section>

<section class="card">
  <div class="ctl" style="justify-content:space-between"><h2 style="margin:0">Backends</h2><button id="b-table">Show table view</button></div>
  <div class="bes" id="bes"></div>
  <div id="tablewrap" hidden style="margin-top:12px;overflow-x:auto"><table id="table"><thead><tr><th>time</th><th>:8081 rps</th><th>:8082 rps</th><th>hits</th><th>misses</th><th>shared</th><th>evicted</th><th>expired</th></tr></thead><tbody></tbody></table></div>
</section>

<section class="card">
  <h2>Event log</h2>
  <p class="sub">state changes, derived from the numbers</p>
  <ul class="log" id="log"></ul>
</section>

<details class="card">
  <summary>What you are looking at</summary>
  <p>Every process on this page runs on a 2015 Samsung Galaxy J1 with 1 GB of RAM and Android 5.1, no root, deployed over <code>adb</code> as static Go binaries. A load balancer on port 8080 round-robins to two backends on 8081 and 8082 and health-checks them every two seconds. Each backend keeps an in-memory LRU cache of at most 200 users with a 20 second TTL in front of a fake database that takes 50 ms per lookup. Concurrent misses for the same user share one lookup (single-flight). The public URL is an ngrok tunnel running on the phone itself.</p>
  <p>Endpoints: <code>/user/&lt;id&gt;</code>, <code>/stats</code>, <code>/stats.json</code>, <code>/dashboard.json</code>, <code>/admin/pause?backend=8081&amp;secs=10</code>. Code and write-up: <a href="https://github.com/g-charan/go-j1" style="color:var(--b1)">github.com/g-charan/go-j1</a>.</p>
</details>
</div>

<script>
(function(){
'use strict';
var $ = function(id){ return document.getElementById(id); };
var PORTS = ['8081', '8082'];
var COL = { b: ['#3987e5', '#d95926'], hit: '#199e70', miss: '#c98500', shared: '#d55181' };
var MAXH = 60;
var S = { prev: null, hist: [], lat: [], sent: 0, alive: [null, null], stale: 0, log: [], cacheMax: 200 };

function fmtTime(t){ var d = new Date(t); return ('0'+d.getHours()).slice(-2)+':'+('0'+d.getMinutes()).slice(-2)+':'+('0'+d.getSeconds()).slice(-2); }
function pct(n, d){ return d ? Math.round(100*n/d) : null; }
function niceMax(v){ if (v <= 0) return 1; var p = Math.pow(10, Math.floor(Math.log(v)/Math.LN10)); var m = v/p; var n = m <= 1 ? 1 : m <= 2 ? 2 : m <= 5 ? 5 : 10; return n*p; }
function quantile(arr, q){ if (!arr.length) return null; var a = arr.slice().sort(function(x,y){return x-y;}); return a[Math.min(a.length-1, Math.floor(q*(a.length-1)))]; }

function logEvent(msg, cls){
  S.log.unshift({ t: Date.now(), msg: msg, cls: cls || '' });
  if (S.log.length > 40) S.log.pop();
  $('log').innerHTML = S.log.map(function(e){ return '<li><time>'+fmtTime(e.t)+'</time><span class="'+e.cls+'">'+e.msg+'</span></li>'; }).join('');
}

/* ---------- polling and per-second derivation ---------- */
function poll(){
  fetch('/dashboard.json', { cache: 'no-store' }).then(function(r){ return r.json(); }).then(function(d){
    S.stale = 0; $('live').className = 'live'; $('livetxt').textContent = 'live';
    var now = Date.now();
    var cur = { t: now, up: d.uptime, be: d.backends.map(function(b){
      var s = b.stats || null;
      return { url: b.url, alive: b.alive, routed: b.routed, s: s };
    }) };
    if (cur.be[0] && cur.be[0].s) S.cacheMax = cur.be[0].s.cache_max;
    if (S.prev) {
      var dt = (now - S.prev.t) / 1000;
      var sample = { t: now, be: [], cpu: null, load: null };
      cur.be.forEach(function(b, i){
        var p = S.prev.be[i];
        var d0 = function(k){ return (b.s && p && p.s) ? Math.max(0, b.s[k] - p.s[k]) : 0; };
        sample.be.push({ rps: p ? Math.max(0, (b.routed - p.routed) / dt) : 0,
          hits: d0('cache_hits'), misses: d0('cache_misses'), shared: d0('cache_shared'),
          evict: d0('cache_evictions'), expired: d0('cache_expired') });
        if (b.s) { sample.cpu = b.s.cpu_pct; sample.load = b.s.load; }
        if (S.alive[i] !== null && S.alive[i] !== b.alive) {
          logEvent('Backend :'+PORTS[i]+(b.alive ? ' is healthy again, traffic resumes' : ' failed its health check, balancer stopped routing to it'), b.alive ? 'good' : 'crit');
        }
        S.alive[i] = b.alive;
      });
      var ev = sample.be.reduce(function(a, b){ return a + b.evict; }, 0);
      var ex = sample.be.reduce(function(a, b){ return a + b.expired; }, 0);
      var sh = sample.be.reduce(function(a, b){ return a + b.shared; }, 0);
      if (ev) logEvent('LRU evicted '+ev+' entr'+(ev>1?'ies':'y')+': cache full at '+S.cacheMax+', least recently used dropped', 'warn');
      if (ex) logEvent('TTL expired '+ex+' entr'+(ex>1?'ies':'y')+', next request for those pays the 50 ms again');
      if (sh) logEvent('Single-flight merged '+sh+' duplicate lookup'+(sh>1?'s':'')+' into one database call', 'good');
      S.hist.push(sample); if (S.hist.length > MAXH) S.hist.shift();
      spawnDots(sample);
    } else {
      cur.be.forEach(function(b, i){ S.alive[i] = b.alive; });
      logEvent('Connected to the phone. Balancer up '+d.uptime+'.');
    }
    S.prev = cur;
    render(cur, d);
  }).catch(function(){
    S.stale++; if (S.stale >= 3) { $('live').className = 'live stale'; $('livetxt').textContent = 'no data for '+S.stale+' s'; }
  });
}

/* ---------- rendering ---------- */
function render(cur, d){
  var last = S.hist[S.hist.length-1];
  var w = S.hist.slice(-3);
  var rps = w.length ? w.reduce(function(a, s){ return a + s.be.reduce(function(x, b){ return x + b.rps; }, 0); }, 0) / w.length : 0;
  $('t-rps').textContent = rps < 10 ? rps.toFixed(1) : Math.round(rps);
  $('t-rps-s').textContent = last ? PORTS.map(function(p, i){ return ':'+p+' '+Math.round(last.be[i].rps); }).join(' · ') : '';

  var H = 0, M = 0, Sh = 0, E = 0;
  S.hist.forEach(function(s){ s.be.forEach(function(b){ H += b.hits; M += b.misses; Sh += b.shared; E += b.evict; }); });
  var ratio = pct(H, H + M + Sh);
  $('t-hit').innerHTML = ratio === null ? '&ndash;' : ratio + '<small>%</small>';
  $('t-hit-s').textContent = (H + M + Sh) ? H+' hits, '+M+' misses, '+Sh+' shared' : 'no lookups yet';
  $('t-hit-b').style.width = (ratio || 0) + '%';

  var p50 = quantile(S.lat, .5), p99 = quantile(S.lat, .99);
  $('t-lat').innerHTML = p50 === null ? '&ndash;' : Math.round(p50) + '<small>ms p50</small>';
  $('t-lat-s').textContent = p99 === null ? 'send some requests' : 'p99 '+Math.round(p99)+' ms over '+S.lat.length+' requests';

  var cpu = null, load = null;
  cur.be.forEach(function(b){ if (b.s) { cpu = b.s.cpu_pct; load = b.s.load; } });
  $('t-cpu').innerHTML = cpu === null ? '&ndash;' : cpu + '<small>%</small>';
  $('t-cpu-s').textContent = load ? 'load average '+load : '';
  $('t-cpu-b').style.width = (cpu || 0) + '%';
  $('t-cpu-bar').className = 'bar' + (cpu > 85 ? ' crit' : cpu > 60 ? ' warn' : '');

  var up = cur.be.filter(function(b){ return b.alive; }).length;
  $('t-up').innerHTML = up + '<small>of '+cur.be.length+'</small>';
  $('t-up').style.color = up === cur.be.length ? '' : up ? '#fab219' : '#d03b3b';
  $('t-up-time').textContent = cur.up;

  $('f-you').textContent = S.sent + ' sent';
  var totalRouted = cur.be.reduce(function(a, b){ return a + b.routed; }, 0);
  $('f-lb').textContent = totalRouted + ' routed';
  var lookups = 0;
  cur.be.forEach(function(b, i){
    var n = i + 1;
    $('f-b'+n).querySelector('rect').setAttribute('class', 'node ' + (b.alive ? 'up' : 'down'));
    $('f-b'+n+'-t').textContent = b.alive ? b.routed + ' routed' : 'DOWN, ' + b.routed + ' routed';
    if (b.s) {
      $('f-b'+n+'-c').setAttribute('width', 116 * b.s.cache_size / b.s.cache_max);
      $('f-b'+n+'-cl').textContent = 'cache ' + b.s.cache_size + '/' + b.s.cache_max;
      lookups += b.s.cache_misses;
    }
  });
  $('f-db').textContent = lookups + ' lookups';

  $('bes').innerHTML = cur.be.map(function(b, i){
    var s = b.s;
    return '<div class="card be'+(b.alive ? '' : ' down')+'" style="margin:0"><h3><span>Backend :'+PORTS[i]+'</span><span class="st">'+(b.alive ? '● healthy' : '● down')+'</span></h3>' +
      (s ? '<div class="kv"><span>routed here</span><span>'+b.routed+'</span><span>cache hits</span><span>'+s.cache_hits+'</span><span>misses (50 ms each)</span><span>'+s.cache_misses+'</span><span>shared a lookup</span><span>'+s.cache_shared+'</span><span>cache size</span><span>'+s.cache_size+' / '+s.cache_max+'</span><span>evicted (LRU)</span><span>'+s.cache_evictions+'</span><span>expired (TTL)</span><span>'+s.cache_expired+'</span><span>uptime</span><span>'+s.uptime+'</span></div>'
         : '<div class="kv"><span>routed here</span><span>'+b.routed+'</span><span>status</span><span>not answering</span></div>') + '</div>';
  }).join('');

  drawRps(); drawCache(); drawTable();
}

/* ---------- charts ---------- */
var L = { l: 34, r: 48, t: 10, b: 22, W: 600, H: 170 };
function frame(ymax, unit){
  var pw = L.W - L.l - L.r, ph = L.H - L.t - L.b, out = '';
  for (var i = 0; i <= 4; i++) {
    var y = L.t + ph - ph * i / 4;
    out += '<line class="'+(i ? 'grid' : 'base')+'" x1="'+L.l+'" x2="'+(L.l+pw)+'" y1="'+y+'" y2="'+y+'"/>';
    out += '<text class="tick" x="'+(L.l-6)+'" y="'+(y+4)+'" text-anchor="end">'+(+(ymax*i/4).toFixed(2))+'</text>';
  }
  out += '<text class="tick" x="'+L.l+'" y="'+(L.H-6)+'">60 s ago</text><text class="tick" x="'+(L.l+pw)+'" y="'+(L.H-6)+'" text-anchor="end">now</text>';
  return out;
}
function xOf(i){ return L.l + (L.W - L.l - L.r) * i / (MAXH - 1); }
function yOf(v, ymax){ var ph = L.H - L.t - L.b; return L.t + ph - ph * v / ymax; }
function pad(){ var n = MAXH - S.hist.length; var a = []; for (var i = 0; i < n; i++) a.push(null); return a.concat(S.hist); }

function drawRps(){
  var rows = pad(), ymax = niceMax(Math.max.apply(null, [1].concat(S.hist.map(function(s){ return Math.max(s.be[0].rps, s.be[1] ? s.be[1].rps : 0); }))));
  var out = frame(ymax), ends = [];
  PORTS.forEach(function(p, k){
    var d = '', lastPt = null;
    rows.forEach(function(s, i){ if (!s || !s.be[k]) return; var x = xOf(i), y = yOf(s.be[k].rps, ymax); d += (d ? 'L' : 'M') + x.toFixed(1) + ',' + y.toFixed(1); lastPt = [x, y, s.be[k].rps]; });
    out += '<path class="line" stroke="'+COL.b[k]+'" d="'+d+'"/>';
    ends[k] = lastPt;
  });
  var ly = ends.map(function(e){ return e ? e[1] : null; });
  if (ends[0] && ends[1] && Math.abs(ly[0] - ly[1]) < 13) { var mid = (ly[0] + ly[1]) / 2; ly[0] = mid - 7; ly[1] = mid + 7; }
  ends.forEach(function(e, k){
    if (!e) return;
    out += '<circle cx="'+e[0]+'" cy="'+e[1]+'" r="3.5" fill="'+COL.b[k]+'" stroke="#1a1a19" stroke-width="2"/><text class="lbl" x="'+(e[0]+8)+'" y="'+(ly[k]+4)+'">:'+PORTS[k]+' '+Math.round(e[2])+'</text>';
  });
  out += '<line class="xh" id="xh-rps" x1="0" x2="0" y1="'+L.t+'" y2="'+(L.H-L.b)+'" style="display:none"/>';
  setChart('c-rps', out, function(i){ var s = rows[i]; if (!s) return null; return fmtTime(s.t) + '<br>' + PORTS.map(function(p, k){ return '<i style="background:'+COL.b[k]+'"></i>:'+p+' <b>'+s.be[k].rps.toFixed(1)+'</b> rps'; }).join('<br>'); }, 'xh-rps');
}
function drawCache(){
  var rows = pad(), ymax = niceMax(Math.max.apply(null, [1].concat(S.hist.map(function(s){ return s.be.reduce(function(a, b){ return a + b.hits + b.misses + b.shared; }, 0); }))));
  var out = frame(ymax), bw = (L.W - L.l - L.r) / MAXH - 2;
  rows.forEach(function(s, i){
    if (!s) return;
    var x = L.l + (L.W - L.l - L.r) * i / MAXH + 1, y0 = yOf(0, ymax);
    var h = s.be.reduce(function(a, b){ return a + b.hits; }, 0), m = s.be.reduce(function(a, b){ return a + b.misses; }, 0), sh = s.be.reduce(function(a, b){ return a + b.shared; }, 0);
    [[h, COL.hit], [m, COL.miss], [sh, COL.shared]].forEach(function(seg){
      if (!seg[0]) return;
      var y1 = yOf(seg[0], ymax), hh = y0 - y1;
      out += '<rect x="'+x.toFixed(1)+'" y="'+(y0 - hh).toFixed(1)+'" width="'+bw.toFixed(1)+'" height="'+Math.max(0, hh - 2).toFixed(1)+'" fill="'+seg[1]+'" rx="1.5"/>';
      y0 -= hh;
    });
  });
  out += '<line class="xh" id="xh-cache" x1="0" x2="0" y1="'+L.t+'" y2="'+(L.H-L.b)+'" style="display:none"/>';
  setChart('c-cache', out, function(i){ var s = rows[i]; if (!s) return null; var h = 0, m = 0, sh = 0; s.be.forEach(function(b){ h += b.hits; m += b.misses; sh += b.shared; }); return fmtTime(s.t)+'<br><i style="background:'+COL.hit+'"></i>hit <b>'+h+'</b><br><i style="background:'+COL.miss+'"></i>miss <b>'+m+'</b><br><i style="background:'+COL.shared+'"></i>shared <b>'+sh+'</b>'; }, 'xh-cache');
}
var hoverIdx = {};
function setChart(id, svgInner, tipFn, xhId){
  var el = $(id), svg = el.querySelector('svg'), tip = el.querySelector('.tip');
  svg.innerHTML = svgInner;
  if (!svg._bound) {
    svg._bound = true;
    svg.addEventListener('mousemove', function(e){
      var r = svg.getBoundingClientRect(), x = (e.clientX - r.left) * L.W / r.width;
      var i = Math.round((x - L.l) / (L.W - L.l - L.r) * (MAXH - 1));
      if (i < 0 || i >= MAXH) { hoverIdx[id] = null; tip.style.display = 'none'; return; }
      hoverIdx[id] = i; showTip(id);
    });
    svg.addEventListener('mouseleave', function(){ hoverIdx[id] = null; tip.style.display = 'none'; });
  }
  el._tipFn = tipFn; el._xhId = xhId;
  showTip(id);
}
function showTip(id){
  var el = $(id), tip = el.querySelector('.tip'), i = hoverIdx[id];
  if (i === null || i === undefined) return;
  var html = el._tipFn(i), xh = $(el._xhId), svg = el.querySelector('svg');
  if (!html) { tip.style.display = 'none'; if (xh) xh.style.display = 'none'; return; }
  var r = svg.getBoundingClientRect(), px = xOf(i) * r.width / L.W;
  xh.setAttribute('x1', xOf(i)); xh.setAttribute('x2', xOf(i)); xh.style.display = '';
  tip.innerHTML = html; tip.style.display = 'block';
  tip.style.left = Math.min(px + 12, r.width - tip.offsetWidth - 4) + 'px'; tip.style.top = '8px';
}
function drawTable(){
  if ($('tablewrap').hidden) return;
  $('table').querySelector('tbody').innerHTML = S.hist.slice(-20).reverse().map(function(s){
    var t = function(k){ return s.be.reduce(function(a, b){ return a + b[k]; }, 0); };
    return '<tr><td>'+fmtTime(s.t)+'</td><td>'+s.be[0].rps.toFixed(1)+'</td><td>'+(s.be[1] ? s.be[1].rps.toFixed(1) : '')+'</td><td>'+t('hits')+'</td><td>'+t('misses')+'</td><td>'+t('shared')+'</td><td>'+t('evict')+'</td><td>'+t('expired')+'</td></tr>';
  }).join('');
}
$('b-table').onclick = function(){ var w = $('tablewrap'); w.hidden = !w.hidden; this.textContent = w.hidden ? 'Show table view' : 'Hide table view'; drawTable(); };

/* ---------- animated dots along the flow ---------- */
var dots = [], dotsG = $('dots'), edges = {};
['e-you-lb', 'e-lb-b1', 'e-lb-b2', 'e-b1-db', 'e-b2-db'].forEach(function(id){ var p = $(id); edges[id] = { p: p, len: p.getTotalLength() }; });
function spawnDots(sample){
  sample.be.forEach(function(b, k){
    var n = Math.min(Math.round(b.rps), 14), misses = Math.min(b.misses + b.shared, n);
    for (var i = 0; i < n; i++) {
      var toDb = i < misses;
      var path = ['e-you-lb', k ? 'e-lb-b2' : 'e-lb-b1'].concat(toDb ? [k ? 'e-b2-db' : 'e-b1-db'] : []);
      if (dots.length > 150) break;
      var c = document.createElementNS('http://www.w3.org/2000/svg', 'circle');
      c.setAttribute('r', toDb ? 4 : 3); c.setAttribute('fill', toDb ? COL.miss : COL.b[k]);
      dotsG.appendChild(c);
      dots.push({ el: c, path: path, t0: performance.now() + i * (900 / n), dur: 700 + path.length * 250 });
    }
  });
}
function tick(now){
  for (var i = dots.length - 1; i >= 0; i--) {
    var d = dots[i], f = (now - d.t0) / d.dur;
    if (f < 0) { d.el.setAttribute('opacity', 0); continue; }
    if (f >= 1) { dotsG.removeChild(d.el); dots.splice(i, 1); continue; }
    var total = d.path.reduce(function(a, id){ return a + edges[id].len; }, 0), dist = f * total, pt = null;
    for (var j = 0; j < d.path.length; j++) { var e = edges[d.path[j]]; if (dist <= e.len || j === d.path.length - 1) { pt = e.p.getPointAtLength(Math.min(dist, e.len)); break; } dist -= e.len; }
    d.el.setAttribute('cx', pt.x); d.el.setAttribute('cy', pt.y); d.el.setAttribute('opacity', 1);
  }
  requestAnimationFrame(tick);
}
requestAnimationFrame(tick);

/* ---------- traffic buttons ---------- */
var buttons = ['b-rand', 'b-same', 'b-sus', 'b-k1', 'b-k2'].map($);
function lock(on){ buttons.forEach(function(b){ b.disabled = on; }); }
function hit(id){
  var t = performance.now();
  return fetch('/user/' + id, { cache: 'no-store' }).then(function(r){ S.lat.push(performance.now() - t); if (S.lat.length > 300) S.lat.shift(); S.sent++; return r.ok; }).catch(function(){ return false; });
}
function burst(idFn, label){
  lock(true); var t0 = performance.now(), ps = [];
  for (var i = 0; i < 100; i++) ps.push(hit(idFn()));
  Promise.all(ps).then(function(res){
    var fails = res.filter(function(ok){ return !ok; }).length;
    logEvent(label + ': 100 requests in ' + Math.round(performance.now() - t0) + ' ms' + (fails ? ', ' + fails + ' failed' : ''));
    lock(false);
  });
}
$('b-rand').onclick = function(){ burst(function(){ return Math.floor(Math.random() * 1000); }, 'Burst, random users'); };
$('b-same').onclick = function(){ var id = Math.floor(Math.random() * 1000); burst(function(){ return id; }, 'Burst, user ' + id + ' only'); };
$('b-sus').onclick = function(){
  lock(true); var btn = this, end = Date.now() + 30000, n = 0;
  logEvent('Sustained load started: 10 requests/s over 300 users for 30 s. Cache holds 200, so expect steady evictions.');
  var iv = setInterval(function(){
    var left = Math.ceil((end - Date.now()) / 1000);
    btn.textContent = 'Sustained load, ' + left + ' s left';
    if (Date.now() >= end) { clearInterval(iv); btn.textContent = 'Sustained load, 30 s'; logEvent('Sustained load finished, ' + n + ' requests.'); lock(false); return; }
    hit(Math.floor(Math.random() * 300)); n++;
  }, 100);
};
function knock(port){
  lock(true);
  fetch('/admin/pause?backend=' + port + '&secs=10').then(function(r){ return r.text(); }).then(function(){
    logEvent('You knocked out backend :' + port + ' for 10 s. Its next health check will fail; requests already in flight to it may error.', 'crit');
    var iv = setInterval(function(){ hit(Math.floor(Math.random() * 50)); }, 200);
    setTimeout(function(){ clearInterval(iv); lock(false); }, 13000);
  }).catch(function(){ lock(false); });
}
$('b-k1').onclick = function(){ knock('8081'); };
$('b-k2').onclick = function(){ knock('8082'); };

poll(); setInterval(poll, 1000);
})();
</script>
`
