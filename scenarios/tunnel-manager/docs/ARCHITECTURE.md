# Tunnel Manager Architecture

## System Context

```
                         ┌─────────────────────┐
                         │   Cloudflare Edge    │
                         │   (DNS + CDN + TLS)  │
                         └──────────┬───────────┘
                                    │
                                    │ QUIC (4 HA connections)
                                    │
┌───────────────────────────────────┼───────────────────────────────────┐
│  Vrooli Server                    │                                   │
│                                   │                                   │
│   ┌───────────────────────────────▼──────────────────────────┐       │
│   │  cloudflared (systemd service)                            │       │
│   │  Metrics: 127.0.0.1:20241/metrics                        │       │
│   │  Health:  127.0.0.1:20241/ready                          │       │
│   │  Config:  ~/.cloudflared/config.yml (local mode)         │       │
│   │           Cloudflare API (remote mode)                    │       │
│   └──┬──────┬──────┬──────┬──────┬──────┬──────┬─────────────┘       │
│      │      │      │      │      │      │      │                      │
│      ▼      ▼      ▼      ▼      ▼      ▼      ▼                      │
│   :35000 :36110 :36221 :36222 :36232 :36234 :36238  ...              │
│   app-   eco-   issue  maint  sys-   swarm  agent                    │
│   mon    mgr    track  orch   mon    mgr    mgr                      │
│                                                                       │
│   ┌───────────────────────────────────────────────────────────┐       │
│   │  Tunnel Manager                                           │       │
│   │  ┌─────────────┐ ┌──────────────┐ ┌───────────────┐      │       │
│   │  │ Route        │ │ Probe Engine │ │ Recovery      │      │       │
│   │  │ Manifest     │ │ (internal +  │ │ Engine        │      │       │
│   │  │ (SQLite)     │ │  external)   │ │ (backoff +    │      │       │
│   │  └──────┬───────┘ └──────┬───────┘ │  circuit brk) │      │       │
│   │         │                │          └───────┬───────┘      │       │
│   │         ▼                ▼                  ▼              │       │
│   │  ┌─────────────┐ ┌──────────────┐ ┌───────────────┐      │       │
│   │  │ Port         │ │ Metrics      │ │ Cloudflare    │      │       │
│   │  │ Auditor      │ │ Scraper      │ │ API Client    │      │       │
│   │  │ (reads       │ │ (Prometheus) │ │ (remote mode) │      │       │
│   │  │ service.json)│ └──────────────┘ └───────────────┘      │       │
│   │  └──────────────┘                                         │       │
│   └───────────────────────────────────────────────────────────┘       │
└───────────────────────────────────────────────────────────────────────┘
```

## Component Design

### Route Manifest

The route manifest is the single source of truth for published tunnel routes. Stored in SQLite.

```json
{
  "routes": [
    {
      "subdomain": "agent-manager",
      "domain": "itsagitime.com",
      "scenario": "agent-manager",
      "port": 36238,
      "health_path": "/health",
      "enabled": true,
      "noTLSVerify": false
    }
  ]
}
```

### Probe Engine

Two-tier probing:

1. **Internal probes**: `GET http://127.0.0.1:{port}{health_path}` — verifies the scenario is locally reachable
2. **External probes**: `GET https://{subdomain}.{domain}{health_path}` — verifies end-to-end tunnel path

Classification matrix:

| Internal | External | Classification |
|----------|----------|----------------|
| Pass     | Pass     | **Up** — route fully operational |
| Pass     | Fail     | **Tunnel Issue** — scenario is fine, tunnel/Cloudflare problem |
| Fail     | Fail     | **Scenario Down** — local service not running |
| Fail     | Pass     | **Anomaly** — shouldn't happen, investigate |

### Recovery Engine

State machine:

```
                ┌──────────┐
                │  Healthy  │◄─── recovery succeeds
                └─────┬─────┘
                      │ failure detected (N consecutive)
                      ▼
                ┌──────────┐
                │ Recovering│──── restart + verify
                └─────┬─────┘
                      │ failure persists
                      ▼
                ┌──────────┐
          ┌────►│ Backoff   │──── wait exponentially
          │     └─────┬─────┘
          │           │ retry
          └───────────┘
                      │ max retries exceeded
                      ▼
                ┌──────────┐
                │ Circuit   │──── stop auto-recovery
                │ Open      │     alert, wait for cooldown
                └──────────┘     or manual reset
```

### Port Auditor

Reads scenario `service.json` files from disk. For each route in the manifest:

1. Find `scenarios/{scenario}/.vrooli/service.json`
2. Parse `ports.ui.port` field
3. Compare against manifest's expected port
4. Report: compliant, mismatch (expected vs actual), missing port field, missing scenario

### Management Modes

| Aspect | Local Mode | Remote Mode (Default) |
|--------|-----------|----------------------|
| Config storage | `~/.cloudflared/config.yml` | Cloudflare edge |
| Route changes | Write YAML + restart cloudflared | API PUT + hot-reload |
| API token required | No | Yes |
| Restart needed | Yes, for config changes | No (hot-reload) |
| Fallback | N/A | Falls back to local if API unreachable |

## Data Model (SQLite)

### Tables

```sql
-- Route manifest
CREATE TABLE routes (
    id INTEGER PRIMARY KEY,
    subdomain TEXT UNIQUE NOT NULL,
    domain TEXT NOT NULL DEFAULT 'itsagitime.com',
    scenario TEXT NOT NULL,
    port INTEGER NOT NULL,
    health_path TEXT DEFAULT '/',
    enabled BOOLEAN DEFAULT TRUE,
    no_tls_verify BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Probe results
CREATE TABLE probe_results (
    id INTEGER PRIMARY KEY,
    route_id INTEGER REFERENCES routes(id),
    probe_type TEXT NOT NULL,       -- 'internal' or 'external'
    status TEXT NOT NULL,           -- 'pass' or 'fail'
    response_time_ms INTEGER,
    http_status INTEGER,
    error TEXT,
    classification TEXT,            -- 'up', 'tunnel-issue', 'scenario-down', 'anomaly'
    probed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Recovery events
CREATE TABLE recovery_events (
    id INTEGER PRIMARY KEY,
    trigger_reason TEXT NOT NULL,
    action TEXT NOT NULL,
    outcome TEXT NOT NULL,          -- 'success' or 'failure'
    duration_ms INTEGER,
    backoff_level INTEGER,
    error TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Metrics snapshots
CREATE TABLE metrics_snapshots (
    id INTEGER PRIMARY KEY,
    ha_connections INTEGER,
    request_errors INTEGER,
    active_streams INTEGER,
    rtt_ms REAL,
    scraped_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Configuration
CREATE TABLE config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Monitoring Loop

The API server runs a background loop (configurable interval, default 60s):

1. **Scrape metrics** from cloudflared `/metrics`
2. **Check health** via `/ready` endpoint
3. **Run probes** on all enabled routes (concurrent, bounded)
4. **Evaluate recovery** triggers (consecutive failures, HA loss)
5. **Execute recovery** if triggered (with backoff/circuit-breaker)
6. **Persist** all results to SQLite
7. **Emit** events for UI consumption
