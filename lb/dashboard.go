package main

// Served at /dashboard. Plain HTML and JS, no dependencies, polls
// /dashboard.json once a second and lets visitors generate traffic.
const dashboardHTML = `<!doctype html>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Galaxy J1 server lab</title>
<style>
:root{color-scheme:dark}
body{margin:0;padding:24px 16px;background:#0e1116;color:#e6edf3;font:15px/1.5 -apple-system,system-ui,sans-serif;max-width:900px;margin:auto}
h1{font-size:22px;margin:0 0 4px}
.sub{color:#8b949e;margin:0 0 24px}
.flow{display:flex;align-items:stretch;gap:12px;flex-wrap:wrap;margin:0 0 24px}
.node{flex:1 1 150px;border:1px solid #30363d;border-radius:8px;padding:12px;background:#161b22;min-width:0}
.node h3{margin:0 0 6px;font-size:14px;color:#8b949e;font-weight:600}
.node .big{font-size:26px;font-weight:700;font-variant-numeric:tabular-nums}
.node small{color:#8b949e}
.arrow{align-self:center;color:#8b949e;font-size:22px}
.backends{display:flex;flex-direction:column;gap:8px;flex:2 1 300px}
.be{border-left:4px solid #3fb950;transition:background .15s}
.be.dead{border-left-color:#f85149;opacity:.6}
.be.hot{background:#1f2a1f}
.kv{display:grid;grid-template-columns:auto 1fr;gap:2px 12px;font-size:13px;margin-top:6px}
.kv span:nth-child(odd){color:#8b949e}
.kv span:nth-child(even){font-variant-numeric:tabular-nums}
.ctl{display:flex;gap:8px;flex-wrap:wrap;align-items:center;margin:0 0 24px}
button{background:#238636;color:#fff;border:0;border-radius:6px;padding:8px 14px;font:inherit;cursor:pointer}
button:disabled{opacity:.5}
button.alt{background:#30363d}
#log{color:#8b949e;font-size:13px}
.about{border-top:1px solid #30363d;padding-top:16px;color:#8b949e;font-size:14px}
.about code{color:#e6edf3}
</style>

<h1>Galaxy J1 server lab</h1>
<p class="sub">Everything below is running on a 2015 Samsung Galaxy J1: 1 GB RAM, 32-bit ARM, Android 5.1, no root. Live, refreshed every second.</p>

<div class="ctl">
  <button id="hit">Send 100 requests</button>
  <button id="same" class="alt">Send 100 for one user</button>
  <span id="log">Requests go to /user/&lt;id&gt;. Random ids show misses and evictions, one id shows cache hits.</span>
</div>

<div class="flow">
  <div class="node"><h3>You</h3><div class="big" id="sent">0</div><small>requests sent from this page</small></div>
  <div class="arrow">&rarr;</div>
  <div class="node"><h3>Load balancer :<span id="lbport"></span></h3><div class="big" id="total">0</div><small>routed &middot; up <span id="lbup"></span></small><small><br>round robin + 2s health checks</small></div>
  <div class="arrow">&rarr;</div>
  <div class="backends" id="backends"></div>
</div>

<div class="about">
  <p>Each backend is a Go HTTP server with an in-memory LRU cache (max 200 users, 20s TTL) in front of a fake database that takes 50 ms per lookup. Concurrent misses for the same user share one lookup (single-flight). Kill one backend and the balancer routes around it within two seconds.</p>
  <p>Endpoints: <code>/user/&lt;id&gt;</code>, <code>/stats</code>, <code>/stats.json</code>, <code>/dashboard.json</code>.</p>
</div>

<script>
var sent = 0, last = {};
var $ = function(id){ return document.getElementById(id); };

function render(d){
  $('lbport').textContent = d.port; $('lbup').textContent = d.uptime;
  var total = 0, html = '';
  d.backends.forEach(function(b){
    total += b.routed;
    var s = b.stats || {};
    var hot = last[b.url] !== undefined && last[b.url] !== b.routed;
    last[b.url] = b.routed;
    html += '<div class="node be' + (b.alive ? '' : ' dead') + (hot ? ' hot' : '') + '">' +
      '<h3>Backend ' + b.url.replace('http://127.0.0.1', '') + (b.alive ? '' : ' &middot; DOWN') + '</h3>' +
      '<div class="big">' + b.routed + '</div><small>routed here</small>' +
      (b.stats ? '<div class="kv">' +
        '<span>cache hits</span><span>' + s.cache_hits + '</span>' +
        '<span>misses (50 ms each)</span><span>' + s.cache_misses + '</span>' +
        '<span>shared a lookup</span><span>' + s.cache_shared + '</span>' +
        '<span>cache size</span><span>' + s.cache_size + ' / ' + s.cache_max + '</span>' +
        '<span>evicted (LRU)</span><span>' + s.cache_evictions + '</span>' +
        '<span>expired (TTL)</span><span>' + s.cache_expired + '</span>' +
        '<span>uptime</span><span>' + s.uptime + '</span></div>' : '') +
      '</div>';
  });
  $('backends').innerHTML = html; $('total').textContent = total;
}

function poll(){
  fetch('/dashboard.json').then(function(r){ return r.json(); }).then(render).catch(function(){});
}
setInterval(poll, 1000); poll();

function blast(idFn){
  var btns = document.querySelectorAll('button'); btns.forEach(function(b){ b.disabled = true; });
  var t0 = Date.now(), done = 0, fails = 0, ps = [];
  for (var i = 0; i < 100; i++) {
    ps.push(fetch('/user/' + idFn()).then(function(r){ if (!r.ok) fails++; done++; sent++; $('sent').textContent = sent; }).catch(function(){ fails++; }));
  }
  Promise.all(ps).then(function(){
    $('log').textContent = done + ' requests in ' + (Date.now() - t0) + ' ms' + (fails ? ', ' + fails + ' failed' : '') + '. Watch the counters.';
    btns.forEach(function(b){ b.disabled = false; });
  });
}
$('hit').onclick = function(){ blast(function(){ return Math.floor(Math.random() * 1000); }); };
$('same').onclick = function(){ var id = Math.floor(Math.random() * 1000); blast(function(){ return id; }); };
</script>
`
