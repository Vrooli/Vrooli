# PRD Context Brief: Tunnel Manager

> This document is a synthesized context brief for `prd-control-tower prd generate`.
> Sources: spec.json, clarify/questions.json (6 answered questions), docs/ARCHITECTURE.md, docs/RESEARCH.md, docs/PROBLEMS.md, PRD.md, README.md, requirements/

## Overview & Value Proposition

Tunnel Manager is foundational infrastructure for Vrooli — the central scenario responsible for ensuring all published Vrooli scenarios remain accessible remotely through the Cloudflare secure tunnel. It eliminates manual tunnel babysitting by providing automated monitoring, intelligent failure diagnosis, and self-healing recovery.

**Primary users**: Vrooli operators, automated infrastructure agents, DevOps engineers

**Deployment surfaces**: Go CLI binary, Go API server, React+Vite UI dashboard

**Value promise**: Guarantees that published Vrooli scenarios remain remotely accessible through Cloudflare Tunnel, even after outages, config drift, or scenario port changes.

**Why it matters**:
1. Remote access is mission-critical — tunnel downtime = Vrooli inaccessible outside LAN
2. Port drift prevention — AI agents building scenarios can inadvertently change fixed ports
3. Intelligent failure diagnosis — distinguishes tunnel-down vs scenario-down vs Cloudflare outage
4. Hands-off recovery — auto-recovers from common failures without physical server access
5. Foundation for multi-server future — centralized tunnel management becomes the networking control plane

## P0 Operational Targets (Core Capabilities)

These are must-ship for viability:

1. **Route manifest** — Declarative `routes` table in SQLite mapping subdomains → scenarios → ports. Seeded via interactive CLI on first run (user confirmed: Q3 answer = interactive prompt with per-route confirmation).

2. **Dual management mode from day one** — Both local (config.yml) and remote (Cloudflare API) modes supported at launch, with auto-detection on startup (user confirmed: Q2 answer = option A). This is NOT a P1 deferral.

3. **Port compliance auditor** — Scan `service.json` files to verify published scenarios have fixed ports matching the route manifest.

4. **Tunnel health monitor** — Monitor cloudflared via systemd status, Prometheus `/metrics`, and `/ready` endpoint. Default 60s interval (user confirmed: Q6 answer = conservative 60s).

5. **Internal liveness probes** — HTTP probe each route's local port to verify the scenario is listening.

6. **External liveness probes** — HTTP probe each route via public URL. 3 consecutive failures required before classifying as tunnel-issue (from PROBLEMS.md Q3).

7. **Auto-recovery engine** — Restart cloudflared with exponential backoff and circuit breaker (max 5 retries). **Critical constraint**: Must not require sudo if avoidable. Explore D-Bus systemd API / polkit first. If elevated privileges are unavoidable, must provide a UX-grantable mechanism so the user can approve access from the dashboard without physical terminal access (hard requirement from Q1 answer).

8. **Lock file coordination** — Shared lock file (`flock(2)`) to prevent concurrent restarts between tunnel-manager and vrooli-autoheal (user confirmed: Q4 answer = defense-in-depth with lock file).

9. **Cloudflare API integration** — Read/write tunnel config via Cloudflare API v4 for remote mode. Read-modify-write pattern (no PATCH endpoint). Hot-reload without cloudflared restart.

10. **Local config management** — Generate/maintain `~/.cloudflared/config.yml` from route manifest. Requires cloudflared restart on config changes.

11. **CLI commands** — `status`, `routes`, `probe`, `audit`, `init` (interactive seeding), `recover` (manual trigger), `config sync`, `config mode`.

## P1 Operational Targets (Important Enhancements)

- Management mode switching CLI (switch local↔remote with config migration)
- Web UI dashboard (React — tunnel status, per-route health, metrics charts, recovery events)
- Prometheus metrics time-series storage
- Recovery event log persistence
- Failure classification (tunnel-down, scenario-down, cloudflare-outage, dns-failure, config-drift)
- Degraded mode detection (HA connections < 4, RTT spikes)
- Route config sync command

## P2 Operational Targets (Nice-to-Have Polish)

- Multi-tunnel support
- Automatic route registration for new fixed-port scenarios
- Webhook/notification alerts (Slack/Discord/email)
- Per-route bandwidth tracking
- Grafana dashboard export
- Mobile-friendly status view
- SSL certificate monitoring

## Tech Direction Snapshot

- **API + CLI**: Go (lightweight, matches cloudflared ecosystem)
- **UI**: React + Vite + TypeScript
- **Storage**: SQLite (self-contained — must work even when other resources are down)
- **Metrics scraping**: cloudflared Prometheus endpoint at `127.0.0.1:20241/metrics`
- **Privilege escalation hierarchy**: (1) D-Bus `godbus` → polkit rule, (2) user-level systemd, (3) UX-grantable sudoers rule
- **Cloudflare API token**: Environment variable `CLOUDFLARE_TUNNEL_TOKEN` with `Account:Cloudflare Tunnel:Edit` + `Zone:DNS:Read` scopes (decided in Q5 — user delegated, implementation chose env var for consistency)
- **Lock file**: `flock(2)` on `/tmp/cloudflared-restart.lock`

## Dependencies & Launch Plan

**Required**:
- cloudflared installed and running as systemd service
- cloudflared metrics port set explicitly: `--metrics 127.0.0.1:20241`
- At least one published route to validate against

**Optional**:
- Redis for pub/sub real-time UI updates (fallback: HTTP polling)
- Cloudflare API token (remote mode only)

**No scenario dependencies** — Tunnel Manager is foundational infrastructure.

**Relationship with vrooli-autoheal**: Defense-in-depth. Autoheal's basic `infra-cloudflared` check continues running. Tunnel Manager adds route-level granularity, external probing, Prometheus metrics, Cloudflare API integration, failure classification, and circuit-breaking. Lock file prevents concurrent restarts.
