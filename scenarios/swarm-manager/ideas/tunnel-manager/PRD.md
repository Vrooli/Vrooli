# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Central management, monitoring, and self-healing for the Cloudflare secure tunnel that provides remote access to all Vrooli scenarios. Maintains a route manifest as the single source of truth, enforces fixed-port contracts on published scenarios, and auto-recovers from tunnel failures.
- **Primary users/verticals**: Vrooli operators, automated infrastructure agents, DevOps engineers
- **Deployment surfaces**: CLI (`tunnel-manager status/routes/probe/audit/recover`), API (route management, health metrics, configuration), UI (real-time route status dashboard)
- **Value promise**: Guarantees that published Vrooli scenarios remain accessible remotely through Cloudflare Tunnel, even after outages, config drift, or scenario port changes. Eliminates the need for manual tunnel babysitting.

### Why It Matters

1. **Remote access is mission-critical**: Without the tunnel, Vrooli is inaccessible outside the local network. Every minute of tunnel downtime means lost productivity and unreachable scenarios.
2. **Port drift prevention**: AI agents building scenarios can inadvertently change fixed ports. Tunnel Manager catches this before it breaks published routes.
3. **Intelligent failure diagnosis**: Distinguishes between "tunnel is down," "scenario is down," and "Cloudflare is having an outage" — enabling targeted recovery instead of blind restarts.
4. **Hands-off recovery**: Auto-recovers from common tunnel failures (process crash, stale connections, Cloudflare edge rotation) without requiring physical access to the server.
5. **Foundation for multi-server future**: When Vrooli scales to multiple servers, centralized tunnel management becomes the networking control plane.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Route manifest | Declarative route manifest (`routes.json`) mapping subdomains to scenarios and expected UI ports, serving as the single source of truth for published routes
- [ ] OT-P0-002 | Port compliance auditor | Scan scenario `service.json` files to verify published scenarios have fixed `port` fields matching the route manifest; report violations
- [ ] OT-P0-003 | Tunnel health monitor | Monitor cloudflared process health via systemd status, Prometheus metrics endpoint (`/metrics`), and `/ready` endpoint
- [ ] OT-P0-004 | Internal liveness probes | HTTP probe each published route's local port to verify the scenario is listening
- [ ] OT-P0-005 | External liveness probes | HTTP probe each published route via its public URL (e.g., `https://agent-manager.itsagitime.com`) to verify end-to-end connectivity
- [ ] OT-P0-006 | Auto-recovery engine | Automatically restart cloudflared when `/ready` returns non-200 or HA connections drop to 0, with exponential backoff and circuit breaker
- [ ] OT-P0-007 | CLI status command | `tunnel-manager status` showing tunnel health, HA connections, error rate, management mode
- [ ] OT-P0-008 | CLI routes command | `tunnel-manager routes` displaying the route manifest with live per-route status (up/down/degraded)
- [ ] OT-P0-009 | CLI probe command | `tunnel-manager probe` running all internal + external probes and reporting results
- [ ] OT-P0-010 | CLI audit command | `tunnel-manager audit` checking port compliance and reporting violations

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Cloudflare API integration (remote mode) | Read/write tunnel configuration via Cloudflare API for hot-reload route management without cloudflared restart
- [ ] OT-P1-002 | Local config management (local mode) | Generate and maintain `~/.cloudflared/config.yml` from the route manifest, with restart on config change
- [ ] OT-P1-003 | Management mode switching | CLI command to switch between remote and local management modes with configuration migration
- [ ] OT-P1-004 | Prometheus metrics scraping | Scrape cloudflared's Prometheus endpoint for HA connections, request errors, RTT, active streams; store time-series data
- [ ] OT-P1-005 | Web UI dashboard | React dashboard showing: tunnel status, per-route health, metrics charts, recent recovery events, management mode
- [ ] OT-P1-006 | Route config sync | `tunnel-manager config sync` to push the route manifest to cloudflared config (local mode) or Cloudflare API (remote mode)
- [ ] OT-P1-007 | Recovery event log | Persist recovery attempts with timestamps, actions taken, and outcomes for post-incident review
- [ ] OT-P1-008 | Failure classification | Categorize failures as: tunnel-down, scenario-down, cloudflare-outage, dns-failure, config-drift — enabling targeted alerts and recovery
- [ ] OT-P1-009 | Degraded mode detection | Detect when HA connections < 4 or RTT spikes, reporting degraded status before full failure

### 🟢 P2 – Future / expansion ideas

- [ ] OT-P2-001 | Multi-tunnel support | Manage multiple tunnels for different domains or server roles
- [ ] OT-P2-002 | Automatic route registration | When a new scenario with a fixed port is started, auto-add a route entry and sync config
- [ ] OT-P2-003 | Webhook/notification alerts | Send alerts to Slack/Discord/email on tunnel failures, route outages, or port compliance violations
- [ ] OT-P2-004 | Cloudflare dashboard link integration | Deep-link to the Cloudflare Zero Trust dashboard for the managed tunnel
- [ ] OT-P2-005 | SSL certificate monitoring | Monitor tunnel certificate expiration and warn before renewal is needed
- [ ] OT-P2-006 | Bandwidth and request analytics | Track per-route request volumes and bandwidth using cloudflared metrics
- [ ] OT-P2-007 | Grafana dashboard export | Generate Grafana dashboard JSON for cloudflared metrics visualization
- [ ] OT-P2-008 | Mobile status view | Lightweight mobile-friendly view of tunnel and route status

## 🧱 Tech Direction Snapshot

- **Preferred stacks/frameworks**: Go API (lightweight, matches cloudflared ecosystem), React + Vite UI (TypeScript), Go CLI binary
- **Data + storage expectations**: SQLite for route manifest, recovery event log, and metrics history (no external database dependency — this is foundational infrastructure that must work even when other resources are down)
- **Integration strategy**:
  - Scrapes cloudflared Prometheus metrics endpoint (configurable, default `127.0.0.1:20241`)
  - Reads scenario `service.json` files directly from filesystem for port auditing
  - Uses Cloudflare API v4 for remote tunnel configuration (`/accounts/{id}/cfd_tunnel/{tunnel_id}/configurations`)
  - Uses `systemctl` for cloudflared service management (Linux)
- **Non-goals / guardrails**:
  - Will NOT replace vrooli-autoheal's basic cloudflared check (defense-in-depth)
  - Will NOT manage cloudflared installation (handled by setup scripts)
  - Will NOT manage scenario lifecycle (starting/stopping scenarios is out of scope — only monitors their ports)
  - Will NOT implement a full Cloudflare Zero Trust dashboard replacement

## 🤝 Dependencies & Launch Plan

- **Required resources**: None (self-contained with SQLite)
- **Optional resources**:
  - `redis` — Pub/sub for real-time UI updates (fallback: HTTP polling)
- **External dependencies**:
  - `cloudflared` daemon (must be installed and running as systemd service)
  - Cloudflare API token (remote mode only, stored securely)
- **Scenario dependencies**: None (foundational infrastructure — other scenarios benefit from it, not the reverse)
- **Launch prerequisites**:
  1. cloudflared installed and running with at least one tunnel configured
  2. At least one published route to validate against
  3. Fixed Prometheus metrics port configured on cloudflared (`--metrics 127.0.0.1:20241`)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
Reliable, self-healing external access management. Vrooli gains the ability to guarantee that its published scenarios remain reachable remotely, detect and diagnose connectivity issues automatically, and recover without human intervention.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Agents can verify their deployed scenarios are publicly accessible without manual testing
- Infrastructure agents gain a reliable signal for "is remote access working?" to inform recovery decisions
- The route manifest provides a machine-readable inventory of all publicly available Vrooli services
- Failure classification data trains future agents to diagnose network issues

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Deployment Manager**: Can verify deployments are reachable post-deploy using tunnel probes
2. **SLA Monitor**: Track uptime percentages per route using probe history
3. **Multi-Server Networking**: Extend tunnel management across multiple Vrooli instances
4. **Customer-Facing Status Page**: Expose route status for external users
5. **Automated DNS Manager**: Coordinate tunnel routes with DNS records programmatically
