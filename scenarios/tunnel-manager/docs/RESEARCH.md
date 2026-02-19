# Research Summary: Tunnel Manager

## Executive Summary

Tunnel Manager is a feasible and high-value scenario that fills a critical gap in Vrooli's infrastructure. The cloudflared ecosystem provides all the APIs and metrics endpoints needed. The existing vrooli-autoheal cloudflared check proves the basic health monitoring approach works, and this scenario extends it with route-level granularity, Cloudflare API integration, and intelligent recovery.

## Feasibility Assessment

### Technical Feasibility

**Fully feasible.** All required integrations have well-documented, stable APIs:

- cloudflared exposes Prometheus metrics on `/metrics` and a readiness endpoint on `/ready` (ports 20241-20245)
- Cloudflare API v4 supports full tunnel configuration management via REST
- `systemctl` provides reliable process management for cloudflared on Linux
- Scenario `service.json` files are JSON — trivially parsed for port auditing

### Resource Requirements

Minimal. SQLite for persistence (no external database dependency). No GPU, no ML models, no heavy compute. The scenario runs as a lightweight Go binary scraping metrics and making HTTP requests.

## Existing Code Analysis

### vrooli-autoheal cloudflared check

**File**: `scenarios/vrooli-autoheal/api/internal/checks/infra/cloudflared.go`

What it does well:
- Binary detection via `exec.LookPath`
- systemd service status via `systemctl is-active`
- HTTP connectivity test (local port)
- Error rate counting from journalctl
- Recovery actions (start/restart/diagnose)

What Tunnel Manager adds:
- **Route-level probing** (each published route individually, not just one test port)
- **External probing** (through the full public DNS path)
- **Prometheus metrics scraping** (HA connections, RTT, request errors — not just journal log parsing)
- **Port contract enforcement** (audit `service.json` files)
- **Cloudflare API integration** (read/write tunnel config, hot-reload)
- **Failure classification** (tunnel-down vs scenario-down vs cloudflare-outage)
- **Circuit breaker** (prevent restart loops)

### cloudflared installation

**File**: `scripts/lib/service/cloudflare-tunnel.sh`

Handles apt-based installation. Tunnel Manager does NOT need to replicate this — it assumes cloudflared is already installed and running.

### Port allocation

**File**: `scripts/lib/network/ports.sh`

Vrooli's port system supports both fixed ports (in `service.json`) and dynamic range allocation. Tunnel Manager enforces that published scenarios use fixed ports, preventing the dynamic allocator from assigning random ports to tunnel-facing scenarios.

## Cloudflare API Capabilities

### Remotely-Managed Tunnel Configuration

- **Endpoint**: `PUT /accounts/{ACCOUNT_ID}/cfd_tunnel/{TUNNEL_ID}/configurations`
- **Model**: Full replacement — must read current config, merge changes, PUT back
- **Hot-reload**: cloudflared automatically picks up new config without restart
- **Metric**: `cloudflared_config_local_config_pushes` tracks config updates

### Key Limitation

No PATCH-style endpoint for individual route add/remove (GitHub issue #1437). Must always do read-modify-write for the entire ingress configuration.

### Tunnel Status

- `GET /accounts/{ACCOUNT_ID}/cfd_tunnel/{TUNNEL_ID}` — tunnel status, connections, edge locations

## cloudflared Metrics Endpoint

Binds to first available port in range 20241-20245 (non-containerized). Key metrics:

| Metric | Type | Purpose |
|--------|------|---------|
| `cloudflared_tunnel_ha_connections` | Gauge | Number of HA connections (normal: 4) |
| `cloudflared_tunnel_request_errors` | Counter | Total request errors |
| `cloudflared_tunnel_active_streams` | Gauge | Currently active streams |
| `quic_client_smoothed_rtt` | Gauge | QUIC round-trip time |
| `cloudflared_config_local_config_pushes` | Counter | Config reload count |

**Recommendation**: Set `--metrics 127.0.0.1:20241` explicitly in the cloudflared systemd unit for deterministic scraping.

## Resilience Patterns

### Built-in cloudflared Resilience

Each cloudflared instance establishes 4 outbound connections to Cloudflare, spread across at least 2 data centers. If one connection/DC fails, traffic routes through remaining connections.

### Health State Machine (Cloudflare edge)

- **Healthy → Down**: Immediate if 3 health probe retries fail
- **Down → Degraded**: 3 consecutive successful probes
- **Degraded → Healthy**: Failure rate < 0.1% over 30 consecutive probes

### Tunnel Manager Recovery Strategy

Builds on cloudflared's built-in resilience with server-side monitoring:
1. Detect when cloudflared's own resilience isn't enough (process crash, stale connections)
2. Restart with exponential backoff to avoid restart loops
3. Circuit-break after repeated failures to prevent masking deeper issues

## Related Scenarios

| Scenario | Relationship |
|----------|-------------|
| vrooli-autoheal | Keeps basic `infra-cloudflared` check for defense-in-depth. Tunnel Manager is the authoritative, detailed monitor. |
| app-monitor | Serves as the proxy for all tunnel traffic (port 35000). Tunnel Manager monitors it as one of the published routes. |
| system-monitor | Monitors system-level metrics. Tunnel Manager focuses specifically on tunnel and route health. |
| deployment-manager | Documents cloudflared setup procedures. Tunnel Manager operationalizes the running tunnel. |

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Cloudflare API rate limiting | Low | Medium | Cache API responses, batch config updates, respect rate limit headers |
| cloudflared metrics port collision | Low | Low | Configure explicit `--metrics` port in systemd unit |
| External probes blocked by Cloudflare WAF | Low | Medium | Use lightweight GET requests with proper User-Agent, test during development |
| Restart loop during Cloudflare outage | Medium | High | Circuit breaker prevents more than 5 consecutive restart attempts |
| API token compromise | Low | High | Store token in environment variable, not config file; minimal permission scope |
| False positive recovery triggers | Medium | Medium | Require N consecutive failures before triggering; configurable thresholds |
