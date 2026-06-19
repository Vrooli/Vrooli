# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Tunnel Manager is Vrooli's external-access control plane — an **exposure broker** and **self-healing tunnel manager**. It programmatically controls which scenarios are reachable from the public internet through the Cloudflare tunnel, maintains a route/exposure manifest as the single source of truth, enforces fixed-port contracts, and auto-recovers the tunnel from failure. It replaces the operator's current manual step of adding public hostnames in the Cloudflare dashboard.
- **Primary users/verticals**: Vrooli operators; automated infrastructure agents; other scenarios that need to be (or need another scenario to be) publicly reachable.
- **Deployment surfaces**: CLI (`tunnel-manager tunnel|routes|exposure|probes|audit|recovery|config`), API (Connect-RPC: exposure broker, health/metrics, configuration, exposure-query for app-monitor), UI (5-surface operator dashboard).
- **Value promise**: Published Vrooli scenarios stay reachable remotely without manual tunnel babysitting. Exposure becomes programmatic, tiered, and budget-aware; the tunnel self-heals; and other scenarios can request their own reachability on demand.

### Why It Matters
1. **Remote access is mission-critical.** Without the tunnel, Vrooli is unreachable outside the local network. Every minute of downtime means unreachable scenarios.
2. **Exposure is manual today.** Exposing a scenario means hand-adding a public hostname in the Cloudflare dashboard, pointed at the scenario's fixed UI port. This must be programmatic and native.
3. **The hostname budget is finite.** A Cloudflare tunnel supports a limited number of public hostnames. Exposure must be tiered (core always-on vs. leased on-demand) and budget-aware so essential scenarios are never crowded out.
4. **Core scenarios must always be reachable.** The self-improvement loop depends on a known core set; those must be guaranteed exposed.
5. **Intelligent, live self-healing.** Distinguishing "tunnel down" from "scenario down" from "Cloudflare outage" enables targeted, automatic recovery instead of blind restarts.
6. **Foundation for multi-server.** Centralized tunnel/exposure management becomes the networking control plane as Vrooli scales to multiple servers.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Exposure manifest (SSOT) | SQLite-backed manifest of routes: subdomain, scenario, domain (a field, not a constant), local UI port, tier (core/leased), lease expiry, enabled flag, health path — the single source of truth for what is publicly exposed
- [x] OT-P0-002 | Programmatic Cloudflare ingress management | Add/remove/sync a scenario's public hostname → `localhost:<fixed UI port>` via the Cloudflare API (remote mode), with hot-reload and no manual dashboard step
- [x] OT-P0-003 | Core-tier always-on exposure | Reconcile the manifest so every scenario in `packages/api-core/coreset` is always exposed and never auto-expired
- [x] OT-P0-004 | Leased-tier on-demand exposure | Request/extend/revoke a time-bounded exposure (default TTL ≈ 1 week) with automatic reaping of expired leases
- [x] OT-P0-005 | Exposure-request API | Other scenarios (and the operator) can request exposure of a scenario via API ("expose me, I'll be used soon")
- [x] OT-P0-006 | Ensure-running delegation | When exposing a scenario, ensure it is running via the existing `internal/lifecycle` seam; Tunnel Manager does not reimplement lifecycle/process management
- [x] OT-P0-007 | Port-compliance auditor | Verify each exposed scenario declares a fixed UI port in `service.json` matching the manifest; report violations
- [x] OT-P0-008 | Tunnel health monitor | Monitor cloudflared via systemd status, Prometheus metrics endpoint, and `/ready`
- [x] OT-P0-009 | Internal liveness probes | HTTP-probe each exposed route's local port to verify the scenario is listening
- [x] OT-P0-010 | External liveness probes | HTTP-probe each exposed route via its public URL to verify end-to-end connectivity
- [x] OT-P0-011 | Auto-recovery engine (live) | Automatically restart cloudflared / re-push config on `/ready` failure or HA-connections=0, with exponential backoff + circuit breaker; Tunnel Manager is the single authoritative owner of cloudflared restart
- [x] OT-P0-012 | CLI surface | `tunnel`, `routes`, `exposure`, `probes`, `audit`, `recovery`, `config` command groups — all with proto-typed `--json`

### 🟠 P1 – Should have post-launch
- [x] OT-P1-001 | Failure classification | Categorize failures as tunnel-down / scenario-down / cloudflare-outage / dns-failure / config-drift to drive targeted recovery and alerts. Current implementation produces healthy / tunnel-down / scenario-down / config-drift from internal/external probe pairs; Cloudflare-outage and DNS-failure need additional signals.
- [x] OT-P1-002 | Local config mode + switching | Generate/maintain `~/.cloudflared/config.yml` from the manifest as a fallback, with remote↔local mode switching and migration
- [x] OT-P1-003 | Prometheus metrics scraping | Scrape cloudflared's metrics endpoint for HA connections, request errors, RTT, active streams; persist time-series in SQLite
- [x] OT-P1-004 | Web UI dashboard (5-surface) | Overview, Exposure (lease management), Recovery & Events, Metrics, Audit
- [x] OT-P1-005 | Recovery event log | Persist recovery attempts with timestamps, actions, and outcomes for post-incident review
- [x] OT-P1-006 | Degraded-mode detection | Detect HA connections < 4 or RTT spikes and report degraded status before full failure
- [x] OT-P1-007 | Exposure-query API for app-monitor | `is-<scenario>-exposed?` + create-lease-and-return-tunnel-URL, consumed by app-monitor's "open in new tab" feature (the app-monitor-side change is a separate task)

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Hostname-budget management | Track exposed-hostname count vs the Cloudflare cap; warn near cap; evict LRU expired/idle leased routes to make room
- [ ] OT-P2-002 | Multi-tunnel / multi-domain support | Manage multiple tunnels/domains for different server roles (networking control plane for multi-server)
- [ ] OT-P2-003 | Usage-based idle spin-down | Spin down idle leased scenarios before TTL based on observed usage
- [ ] OT-P2-004 | Webhook / notification alerts | Alert to Slack/Discord/email on tunnel failures, route outages, or port-compliance violations
- [ ] OT-P2-005 | Cloudflare dashboard deep-link | Deep-link to the Zero Trust dashboard for the managed tunnel
- [ ] OT-P2-006 | Certificate monitoring | Monitor tunnel/SSL certificate expiration and warn before renewal
- [ ] OT-P2-007 | Per-route analytics | Track per-route request volumes and bandwidth from cloudflared metrics
- [ ] OT-P2-008 | Grafana dashboard export | Generate Grafana dashboard JSON for cloudflared metrics visualization

> **Note on budget tiering.** OT-P2-001 (hostname-budget management) is parked at P2 because the Cloudflare cap is likely relaxed under API/config-managed exposure (vs. the ~100 dashboard limit), and core+leased tiering already bounds growth. If the real cap is confirmed low against the live plan, promote it to P0.

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API on **Connect-RPC** (proto contracts under `packages/proto/schemas/tunnel-manager`), React + Vite + Tailwind UI (vrooli-default design kit), Go CLI via `cli-core`/`cliapp.ArgSchema`. Screaming-architecture domains: `routes`, `audit`, `tunnel`, `probes`, `recovery`, `config`, `exposure` (+ `health`).
- Data + storage expectations: **SQLite only** (manifest, leases, metrics history, probe history, recovery log). No external database — foundational infra must keep working when other resources are down.
- Integration strategy: Cloudflare API v4 for remote ingress config; scrape cloudflared Prometheus endpoint (default `127.0.0.1:20241`); read scenario `service.json` for port auditing; `api-core/coreset` for the core set; `internal/lifecycle` for ensure-running; `systemctl` for cloudflared service management.
- Non-goals / guardrails: will NOT reimplement scenario lifecycle (delegates to `internal/lifecycle`); will NOT replace app-monitor's reverse proxy (stays in `packages/api-base`; only the new-tab feature integrates); will NOT manage cloudflared installation (setup handles it); will NOT replace vrooli-autoheal — autoheal's cloudflared check downgrades to alert-only but remains as defense-in-depth.

## 🤝 Dependencies & Launch Plan
- Required resources: none (SQLite, self-contained).
- Optional resources: `redis` (UI pub/sub for real-time updates; fallback to HTTP polling).
- External dependencies: `cloudflared` daemon (systemd); Cloudflare API token (remote mode only).
- Scenario dependencies (runtime seams, not hard deps): `packages/api-core/coreset` (core set), `internal/lifecycle` (ensure-running).
- Operational risks: live auto-recovery acting on foundational infra (mitigated by circuit breaker + single-owner restart contract with vrooli-autoheal); hostname-budget exhaustion (mitigated by tiering and, later, budget management/LRU eviction); Tunnel Manager must itself declare a fixed UI port — the very contract it enforces on others.
- Launch sequencing: (1) CLI + API first, then the dashboard; (2) seed core-tier exposure; (3) enable leasing; (4) confirm the real Cloudflare hostname cap against the live plan; (5) flip vrooli-autoheal's cloudflared check to alert-only.

## 🎨 UX & Branding
- Look & feel: clean infrastructure dashboard, status-oriented, color-coded health (green/yellow/red), minimal — operators need quick glances, not complex interactions. vrooli-default tokens, light/dark.
- Accessibility: CLI output parseable (`--json` everywhere); UI follows standard Vrooli React a11y (roles, `aria-*`, `data-testid` selectors, i18n, WCAG AA contrast).
- Voice & messaging: terse, operational, factual.
- Branding hooks: keep the seeded PWA manifest/icons valid; replace generic icons with tunnel-manager branding when available.

## 🎯 Capability Definition

### Core Capability
Reliable, self-healing, **programmable external-access management**. Vrooli gains the ability to decide — by policy and on demand — which scenarios are reachable from the internet, to guarantee core scenarios are always reachable, to lease temporary reachability to others, and to detect, diagnose, and recover connectivity failures automatically.

### Intelligence Amplification
- Agents can expose a scenario they just built and verify it is publicly reachable, with no manual Cloudflare steps.
- Infrastructure agents get a reliable "is remote access working?" signal to inform recovery decisions.
- The exposure manifest is a machine-readable inventory of everything publicly reachable, with tier and lease state.
- Failure-classification data trains future agents to diagnose network issues.

### Recursive Value
1. **Deployment Manager**: verify deployments are reachable post-deploy via tunnel probes.
2. **SLA Monitor**: track per-route uptime from probe history.
3. **Multi-Server Networking**: extend exposure management across Vrooli instances.
4. **Customer-Facing Status Page**: expose route status to external users.
5. **On-demand compute fabric**: scenarios lease their own reachability for bounded windows, enabling spin-up-on-intent workflows.

## 📎 Appendix
- Regeneration & adoption plan: `docs/plans/tunnel-manager-regen-adoption-plan.md` (repo root).
- Pre-regen reference (port source): `/tmp/tunnel-manager-OLD-reference`.
