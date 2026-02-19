# Tunnel Manager

Central management, monitoring, and self-healing for the Cloudflare secure tunnel that provides remote access to all Vrooli scenarios.

## Overview

Tunnel Manager is the authoritative owner of Vrooli's external access layer. It maintains a **route manifest** — the single source of truth mapping subdomains to scenarios and their required UI ports — and continuously enforces that:

1. Every published scenario has the correct fixed port in its `service.json`
2. The cloudflared tunnel process is healthy with all HA connections established
3. Every published route is actually reachable end-to-end (local → Cloudflare edge → public DNS)
4. Failures are automatically recovered with exponential backoff and circuit-breaking

Supports both **locally-managed** (config YAML) and **remotely-managed** (Cloudflare API) tunnel configurations, with remote as the default.

## Quick Start

```bash
# Build and install
cd scenarios/tunnel-manager && make setup

# Start the scenario
make start

# Check tunnel health
tunnel-manager status

# View route manifest
tunnel-manager routes

# Verify all routes are reachable
tunnel-manager probe

# Audit scenario port compliance
tunnel-manager audit
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `tunnel-manager status` | Overall tunnel health: systemd, HA connections, metrics, error rate |
| `tunnel-manager routes` | Display the route manifest with live status per route |
| `tunnel-manager probe` | Run internal + external liveness probes on all published routes |
| `tunnel-manager audit` | Check all manifested scenarios for port compliance in `service.json` |
| `tunnel-manager recover` | Manually trigger tunnel recovery (restart with verification) |
| `tunnel-manager config sync` | Sync route manifest to cloudflared config (local) or Cloudflare API (remote) |
| `tunnel-manager config mode` | Show or switch management mode (local/remote) |

## Architecture

```
tunnel-manager/
├── api/           # Go API server
├── cli/           # CLI (tunnel-manager binary)
├── ui/            # React dashboard (Vite + TypeScript)
├── .vrooli/       # Lifecycle configuration
├── requirements/  # Technical requirements
└── docs/          # Internal documentation
```

```
┌──────────────────────────────────────────────────────────────────┐
│  UI (React + Vite)             │  CLI (Go)                       │
│  Port: TBD (lifecycle)         │  tunnel-manager                 │
│  Real-time route status board  │  status / routes / probe / ...  │
└──────────┬─────────────────────┴──────────┬──────────────────────┘
           │                                │
           ▼                                ▼
┌──────────────────────────────────────────────────────────────────┐
│  API (Go)   Port: TBD (lifecycle)                                │
│  Route manifest ←→ service.json auditor ←→ Cloudflare API        │
│  Prometheus scraper (cloudflared /metrics)                        │
│  Liveness prober (internal + external)                            │
│  Auto-recovery engine (backoff + circuit breaker)                 │
└──────┬──────┬────────┬──────────────────────────────────────────┘
       │      │        │
       ▼      ▼        ▼
  cloudflared  scenarios/*/service.json  Cloudflare API
  /metrics     (port enforcement)        (remote config)
```

## Dependencies

### Resources
| Resource | Purpose | Required |
|----------|---------|----------|
| None | Self-contained — uses SQLite for state and cloudflared metrics endpoint | — |

### External Services
| Service | Purpose | Required |
|---------|---------|----------|
| cloudflared | The tunnel daemon being managed | Yes |
| Cloudflare API | Route management (remote mode only) | No (local mode fallback) |

### Scenario Dependencies
| Scenario | Purpose |
|----------|---------|
| None | Tunnel Manager is foundational infrastructure |

## Management Modes

### Remote (Default)
- Tunnel configuration stored on Cloudflare's edge
- Ingress rules managed via `PUT /accounts/{id}/cfd_tunnel/{tunnel_id}/configurations`
- Config changes apply via hot-reload — no cloudflared restart needed
- Requires Cloudflare API token with tunnel configuration permissions

### Local
- Tunnel configuration in `~/.cloudflared/config.yml`
- Config changes require cloudflared restart
- No API token needed
- Useful for air-gapped environments or when API access is unavailable

## Relationship with vrooli-autoheal

vrooli-autoheal has a basic `infra-cloudflared` health check that monitors binary installation, systemd status, and basic connectivity. That check remains for defense-in-depth. Tunnel Manager provides:

- **Route-level granularity**: Probes each published route individually, not just one test port
- **Port contract enforcement**: Ensures scenarios haven't changed their fixed ports
- **Prometheus metrics**: Scrapes cloudflared's metrics endpoint for HA connection count, request errors, RTT
- **Cloudflare API integration**: Can read and write tunnel configuration programmatically
- **External probing**: Verifies routes through the full public DNS → Cloudflare edge → tunnel → local path
- **Intelligent recovery**: Distinguishes "tunnel is down" from "scenario is down" from "Cloudflare outage"
