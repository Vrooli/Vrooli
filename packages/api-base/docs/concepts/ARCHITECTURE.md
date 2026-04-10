# Proxy System Architecture

End-to-end overview of how `@vrooli/api-base` routes browser requests through a host scenario to child scenarios.

[CODE: packages/api-base/src/server/host.ts] - Core proxy host implementation
[CODE: packages/api-base/src/server/proxy.ts] - HTTP/WebSocket proxy functions
[CODE: packages/api-base/src/server/inject.ts] - HTML metadata injection
[CODE: packages/api-base/src/server/template.ts] - Server template factory
[CODE: packages/api-base/src/shared/types.ts] - Type definitions

---

## System Overview

All running scenarios are accessed through a single **host** (app-monitor) instead of individual ports:

```
                         BROWSER
   ┌───────────────────────────────────────────────────────────┐
   │  http://localhost:21774/apps/my-scenario/proxy/     (UI)  │
   │  http://localhost:21774/apps/my-scenario/proxy/api/ (API) │
   │  ws://localhost:21774/apps/my-scenario/proxy/ws     (WS)  │
   └─────────────────────────┬─────────────────────────────────┘
                             │
                             ▼
   ┌───────────────────────────────────────────────────────────┐
   │  APP-MONITOR (host scenario)                              │
   │  UI: port 21774  |  API: port 21600+                      │
   │                                                           │
   │  ┌───────────────────────────────────────────────────┐    │
   │  │  ui/server.js (Express + api-base)                │    │
   │  │                                                   │    │
   │  │  createScenarioProxyHost({                        │    │
   │  │    hostScenario: 'app-monitor',                   │    │
   │  │    fetchAppMetadata: (appId) => axios.get(...)    │    │
   │  │  })                                               │    │
   │  │                                                   │    │
   │  │  Routes:                                          │    │
   │  │    /apps/:appId/proxy/*           -> primary port │    │
   │  │    /apps/:appId/ports/:key/proxy/* -> named port  │    │
   │  └─────────────┬─────────────────────────────────────┘    │
   │                │                                          │
   │                │  fetchAppMetadata("my-scenario")         │
   │                ▼                                          │
   │  ┌───────────────────────────────────────────────────┐    │
   │  │  Go API (app_service.go)                          │    │
   │  │                                                   │    │
   │  │  GET /api/v1/apps/:id  ->  {                      │    │
   │  │    "id": "my-scenario",                           │    │
   │  │    "status": "running",                           │    │
   │  │    "port_mappings": { "ui": 8080, "api": 8090 }  │    │
   │  │  }                                                │    │
   │  └───────────────────────────────────────────────────┘    │
   └───────────────────────────┬───────────────────────────────┘
                               │
                               │  Forward to 127.0.0.1:PORT
                               ▼
          ┌────────────────────┴────────────────────┐
          ▼                                         ▼
   ┌─────────────────┐                   ┌─────────────────┐
   │  SCENARIO A     │                   │  SCENARIO B     │
   │  UI:  port 8080 │                   │  UI:  port 9080 │
   │  API: port 8090 │                   │  API: port 9090 │
   └─────────────────┘                   └─────────────────┘
```

---

## Request Decision Tree

[CODE: packages/api-base/src/server/host.ts#handleScenarioProxyRequest]

When a request hits `/apps/:appId/proxy/...`, this decision tree executes:

```
Incoming: /apps/:appId/proxy/...
          │
          ▼
   ┌──────────────┐
   │ Path          │──── Has ".." ? ──> 400 Bad Request
   │ traversal?    │
   └──────┬───────┘
          │ clean
          ▼
   ┌──────────────┐
   │ Get metadata  │──> fetchAppMetadata(appId)
   │ (cached?)     │    ┌────────────────────────┐
   │               │    │ Metadata Cache          │
   │               │    │ TTL: configurable       │
   │               │    │ Key: appId              │
   └──────┬───────┘    └────────────────────────┘
          │
          ▼
   ┌──────────────┐     YES     ┌───────────────────┐
   │ Path starts  │────────────>│ Forward to         │
   │ with /api?   │             │ apiPort            │
   │              │             │ No HTML injection  │
   └──────┬───────┘             └───────────────────┘
          │ NO
          ▼
   ┌──────────────┐     NO      ┌───────────────────┐
   │ HTML-like?   │────────────>│ Forward as-is      │
   │ (GET +       │             │ (CSS, JS, images)  │
   │  text/html)  │             │ No injection       │
   └──────┬───────┘             └───────────────────┘
          │ YES
          ▼
   ┌──────────────┐     HIT     ┌───────────────────┐
   │ HTML cache   │────────────>│ Check background   │
   │ lookup       │             │ health state       │
   └──────┬───────┘             │ then serve cached  │
          │ MISS                └───────────────────┘
          ▼
   ┌─────────────────────────────────────┐
   │ Stream HTML from upstream           │
   │                                     │
   │  StreamingHeadInjector:             │
   │  1. Buffer chunks until </head>     │
   │  2. Inject proxy metadata JSON      │
   │  3. Inject <base href="...">        │
   │  4. Flush remaining chunks          │
   │  5. Cache result if 200 + text/html │
   │                                     │
   │  Safety: 512KB buffer limit         │
   └─────────────────────────────────────┘
```

### Detection Functions

[CODE: packages/api-base/src/server/host.ts#isApiPath] - Checks if path starts with `/api`
[CODE: packages/api-base/src/server/host.ts#isHtmlLikeRequest] - GET + Accept: text/html, or path is `/` or `*.html`
[CODE: packages/api-base/src/server/host.ts#hasPathTraversal] - Detects `..` sequences after URL decoding

---

## HTML Injection Pipeline

[CODE: packages/api-base/src/server/inject.ts#injectProxyMetadata]
[CODE: packages/api-base/src/server/inject.ts#injectBaseTag]

When the proxy serves an HTML page, it intercepts `</head>` and injects metadata:

```
Upstream HTML                          Delivered HTML
┌──────────────────┐                   ┌──────────────────────────────────┐
│ <html>           │                   │ <html>                           │
│   <head>         │                   │   <head>                         │
│     <title>      │                   │     ┌─── INJECTED ────────────┐  │
│       App        │                   │     │                         │  │
│     </title>     │  StreamingHead    │     │ <base href="/apps/      │  │
│   </head>        │───Injector───────>│     │   my-scenario/proxy/"   │  │
│   <body>         │                   │     │   data-proxy-host />    │  │
│     ...          │                   │     │                         │  │
│   </body>        │                   │     │ <script id="vrooli-     │  │
│ </html>          │                   │     │   proxy-metadata">      │  │
└──────────────────┘                   │     │   window.__VROOLI_      │  │
                                       │     │     PROXY_INFO__ = {    │  │
                                       │     │       appId, ports,     │  │
                                       │     │       primary, hosts    │  │
                                       │     │     }                   │  │
                                       │     │   // + Proxy index      │  │
                                       │     │   // + Fetch patching   │  │
                                       │     │ </script>               │  │
                                       │     └─────────────────────────┘  │
                                       │     <title>App</title>           │
                                       │   </head>                        │
                                       │   <body>...</body>               │
                                       │ </html>                          │
                                       └──────────────────────────────────┘
```

### What Gets Injected

| Element | Purpose |
|---------|---------|
| `<base href="/apps/{id}/proxy/">` | Makes relative asset URLs (CSS, JS) resolve through the proxy |
| `window.__VROOLI_PROXY_INFO__` | Full proxy metadata (appId, ports, hosts, hostEndpoints) |
| `window.__VROOLI_PROXY_INDEX__` | Optimized alias-to-port lookup map |
| Fetch/XHR/WebSocket patches | Rewrites `http://localhost:PORT/...` to `/apps/{id}/proxy/...` |

### Fetch Patching Detail

[CODE: packages/api-base/src/server/inject.ts#buildProxyBootstrapScript]

When `patchFetch: true`, the injected script intercepts network calls:

```
Scenario code:  fetch("http://localhost:8090/api/v1/users")
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │ Patched fetch()      │
                         │                      │
                         │ 1. Parse URL         │
                         │ 2. Is localhost?      │──> NO ──> original fetch()
                         │ 3. Port in aliasMap?  │──> NO ──> original fetch()
                         │ 4. Lookup port entry  │
                         │ 5. Rewrite URL        │
                         └──────────┬────────────┘
                                    │ YES
                                    ▼
Rewritten:      fetch("/apps/my-scenario/proxy/api/v1/users")
                                    │
                                    ▼
                         Routes through proxy host
```

This patching covers `fetch()`, `XMLHttpRequest.open()`, and `new WebSocket()`.

---

## Three-Layer Cache

[CODE: packages/api-base/src/server/host.ts#getProxyContext] - Metadata cache
[CODE: packages/api-base/src/server/host.ts#getCachedHtmlEntry] - HTML cache
[CODE: packages/api-base/src/server/host.ts#HealthTracker] - Background health checker

```
┌─────────────────────────────────────────────────────────┐
│                    THREE-LAYER CACHE                      │
│                                                          │
│  Layer 1: METADATA CACHE                                 │
│  ┌──────────────────────────────────────────────────┐    │
│  │  Map<appId, { context: ProxyContext, timestamp }> │    │
│  │  TTL: configurable (app-monitor uses 5 min)       │    │
│  │  Contains: port_mappings, PortEntry[], ProxyInfo   │    │
│  │  Invalidation: proxyHost.invalidate(appId)         │    │
│  └──────────────────────────────────────────────────┘    │
│                                                          │
│  Layer 2: HTML CACHE                                     │
│  ┌──────────────────────────────────────────────────┐    │
│  │  Map<appId, Map<path, HtmlCacheEntry>>            │    │
│  │  Max entries: 200 (configurable, FIFO eviction)   │    │
│  │  Only caches: 200 OK + text/html + no Set-Cookie  │    │
│  │  Stores fully-injected HTML (metadata baked in)    │    │
│  └──────────────────────────────────────────────────┘    │
│                                                          │
│  Layer 3: BACKGROUND HEALTH CHECK                        │
│  ┌──────────────────────────────────────────────────┐    │
│  │  HealthTracker probes upstream ports periodically   │    │
│  │  (default: every 5s via TCP connect, 500ms timeout) │    │
│  │  If unhealthy: invalidate entry, return 502         │    │
│  └──────────────────────────────────────────────────┘    │
│                                                          │
│  INVALIDATION TRIGGER:                                   │
│  Go API --> POST /__cache/invalidate { appId }           │
│  Fired on: scenario start / stop / restart               │
│                                                          │
│  PRE-WARMING:                                            │
│  On UI server startup, fetches metadata for all          │
│  running scenarios to populate metadata cache             │
└──────────────────────────────────────────────────────────┘
```

### Cache Flow Per Request

```
Request: GET /apps/scenario-a/proxy/

  1. Metadata cache lookup (appId: "scenario-a")
     ├── HIT (age < TTL)  --> use cached ProxyContext
     └── MISS             --> fetchAppMetadata("scenario-a")
                               └── store in cache

  2. HTML cache lookup (appId: "scenario-a", path: "/")
     ├── HIT --> 3. Background health state lookup
     │           ├── Healthy      --> serve cached HTML (inject fresh timestamp)
     │           └── Unhealthy    --> evict entry, fall through to 4
     └── MISS --> 4. Stream from upstream
                      └── if cacheable (200 + text/html + no Set-Cookie)
                          └── store in HTML cache with injection baked in
```

---

## WebSocket Tunneling

[CODE: packages/api-base/src/server/proxy.ts#proxyWebSocketUpgrade]
[CODE: packages/api-base/src/server/host.ts#handleUpgrade]

WebSocket connections are tunneled via raw TCP piping:

```
Browser                      Proxy (host.ts)                 Scenario
  │                              │                               │
  │  GET /apps/id/proxy/ws      │                               │
  │  Upgrade: websocket          │                               │
  │  Sec-WebSocket-Key: abc      │                               │
  │  Sec-WebSocket-Version: 13   │                               │
  │  ─────────────────────────>  │                               │
  │                              │  1. Extract appId             │
  │                              │  2. Resolve target port       │
  │                              │     (API path? -> apiPort     │
  │                              │      else -> uiPort)          │
  │                              │                               │
  │                              │  net.connect(port, host)      │
  │                              │  ─────────────────────────>   │
  │                              │                               │
  │                              │  Forward full upgrade request │
  │                              │  (ALL headers preserved)      │
  │                              │  ─────────────────────────>   │
  │                              │                               │
  │                              │  <── 101 Switching ───────────│
  │  <── 101 Switching ──────── │                               │
  │                              │                               │
  │  <═══════ Bidirectional pipe (socket.pipe) ═════════════════>│
  │            upstream.pipe(clientSocket)                        │
  │            clientSocket.pipe(upstream)                        │
  │                              │                               │
  │  Socket close / error        │  Teardown both sockets        │
  │  ─────────────────────────>  │  ─────────────────────────>   │
```

Key details:
- **All headers preserved** - Unlike HTTP proxy, WebSocket headers (including `Sec-WebSocket-*`) are not filtered
- **TCP tunnel** - Uses `net.connect()` for true bidirectional streaming
- **No buffering** - Data flows directly between sockets via `.pipe()`
- **Keepalive enabled** - Both sockets have `setKeepAlive(true)` and `setNoDelay(true)`

---

## Scenario Discovery and Registration

Scenarios don't explicitly register with the proxy. Discovery is dynamic:

```
                              ┌──────────────────┐
                              │ Scenario starts   │
                              │ (via Makefile or   │
                              │  vrooli CLI)       │
                              └────────┬─────────┘
                                       │
                                       ▼
                              ┌──────────────────┐
                              │ Lifecycle system   │
                              │ assigns ports from │
                              │ service.json       │
                              │ ranges             │
                              └────────┬─────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────┐
│  app-monitor Go API                                          │
│                                                              │
│  1. Queries orchestrator for running scenarios               │
│  2. Gets port_mappings for each scenario                     │
│  3. Stores in PostgreSQL (with 90s orchestrator cache)       │
│  4. Calls POST /__cache/invalidate on UI server              │
│     when scenario status changes                             │
└────────────────────────────────┬────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────┐
│  app-monitor UI server (proxy host)                          │
│                                                              │
│  1. User navigates to /apps/scenario-id/proxy/               │
│  2. fetchAppMetadata("scenario-id") calls Go API             │
│  3. Go API returns { port_mappings: { ui: 8080, api: 8090 }}│
│  4. Proxy context built, HTML streamed + injected            │
│  5. Metadata cached for 5 minutes                            │
└─────────────────────────────────────────────────────────────┘
```

---

## Performance Monitoring

[CODE: packages/api-base/src/server/perf.ts]

When `enableServerTiming: true` (default) or `enableMetrics: true`:

### Server-Timing Header

Every proxied response includes timing breakdown:

```
Server-Timing: total;dur=150, ctx;dur=30, fwd;dur=120
```

| Phase | Key | Description |
|-------|-----|-------------|
| TOTAL | `total` | End-to-end request time |
| CTX | `ctx` | Time to fetch/resolve proxy context (metadata cache) |
| HTML_CACHE | `cache` | Time to serve from HTML cache |
| HTML_STREAM | `stream` | Time to stream + inject from upstream |
| FWD_HTTP | `fwd` | Time to forward non-HTML request |

### Metrics Endpoint

When `enableMetrics: true`:

```
GET  /__perf       -> JSON snapshot of latency percentiles + cache hit rates
POST /__perf/reset -> Clear counters
```

---

## Route Registration

[CODE: packages/api-base/src/server/host.ts#registerRoutes]

Two route patterns are registered on the Express router:

```
Pattern 1 (named port):
  /apps/:appId/ports/:portKey/proxy/*
  /apps/:appId/ports/:portKey/proxy
  Handler: handlePortProxyRequest
  Use: Access a specific non-primary port by name or number

Pattern 2 (primary port):
  /apps/:appId/proxy/*
  /apps/:appId/proxy
  Handler: handleScenarioProxyRequest
  Use: Default routing (UI port, or API if path starts with /api)
```

Port routes are registered first so they take priority over the general proxy route.

---

## Security Boundaries

### Path Traversal Prevention

[CODE: packages/api-base/src/server/host.ts#hasPathTraversal]

Applied to: `appId`, `relativeUrl`, `portKey`, WebSocket paths

```
Validation:
  1. URL-decode the value
  2. Normalize backslashes to forward slashes
  3. Collapse double slashes
  4. Test against: /(^|\/|\\)\.\.(?=\/|\\|$)/
  5. If match -> 400 Bad Request (HTTP) or socket.destroy() (WS)
```

### Header Filtering

| Context | Filtering |
|---------|-----------|
| HTTP proxy (`proxyToApi`) | Removes hop-by-hop headers (Connection, Keep-Alive, etc.) |
| HTML streaming (`streamProxiedHtml`) | Removes `Content-Length` (chunked), sets `Cache-Control: no-store` |
| WebSocket upgrade (`proxyWebSocketUpgrade`) | Preserves ALL headers including `Sec-WebSocket-*` |

---

## Key Types

[CODE: packages/api-base/src/shared/types.ts]

| Type | Purpose |
|------|---------|
| `ScenarioProxyHostOptions` | Configuration for `createScenarioProxyHost()` |
| `ScenarioProxyHostController` | Return value: `{ router, handleUpgrade, invalidate, clearCache, getMetrics }` |
| `ScenarioProxyAppMetadata` | Raw metadata from host API (port_mappings, config, etc.) |
| `ProxyContext` | Normalized internal state (uiPort, apiPort, portLookup, metadata) |
| `ProxyInfo` | Injected into HTML (appId, ports, primary, hosts, hostEndpoints) |
| `PortEntry` | Single port descriptor (port, label, slug, isPrimary, path, aliases) |
| `HostEndpointDefinition` | Paths that bypass fetch patching (host-owned routes) |

---

## See Also

- [Deployment Contexts](./deployment-contexts.md) - The three environments where scenarios run
- [Proxy Resolution](./proxy-resolution.md) - Client-side URL resolution algorithm
- [Host Scenario Pattern](../guides/host-scenario-pattern.md) - Step-by-step implementation guide
- [Server API Reference](../api/server.md) - Function signatures and parameters
- [WebSocket Support](./websocket-support.md) - Client-side WS/WSS resolution
- [Runtime Configuration](./runtime-config.md) - Config injection system
