# Enhanced Plan: Tunnel Manager

## Overview
Tunnel Manager is a central infrastructure scenario for managing, monitoring, and self-healing the Cloudflare secure tunnel that provides remote access to all Vrooli scenarios. It maintains a route manifest as the single source of truth, enforces fixed-port contracts on published scenarios, supports both locally-managed and remotely-managed tunnel configurations (auto-detecting the current mode on startup), and auto-recovers from tunnel failures with exponential backoff and circuit-breaking.

## Clarifications Applied

| Question | Answer | Impact |
|----------|--------|--------|
| Q1: Sudo access for systemctl restart | Avoid sudo if possible. Research patterns from vrooli-autoheal/workspace-sandbox. If elevated privileges unavoidable, must be grantable from UX (hard requirement — user won't always be at computer) | Recovery engine must explore non-sudo approaches first: D-Bus systemd API, user-level systemd units, or polkit. If sudo is required, the UI must provide a UX flow for granting access (e.g., a one-time setup prompt that configures the sudoers rule). This is a **hard requirement**. |
| Q2: Remote-managed tunnel migration | A: Support both modes from day one; auto-detect current mode on startup | Architecture must include mode detection on startup (check for `~/.cloudflared/config.yml` vs Cloudflare API config). Both local and remote code paths are P0, not P1. |
| Q3: Route manifest initial seeding | A: Interactive CLI prompt showing discovered routes, ask to confirm each | First-run experience includes an interactive `tunnel-manager init` command that reads current cloudflared config, displays discovered routes, and asks the user to confirm/edit/skip each one before persisting to the manifest. |
| Q4: Coordination with vrooli-autoheal | A: Defense-in-depth with a lock file to prevent concurrent restarts | A shared lock file (e.g., `/tmp/cloudflared-restart.lock`) prevents both tunnel-manager and vrooli-autoheal from attempting cloudflared restarts simultaneously. Both systems must acquire the lock before restarting and release it after. |
| Q5: Cloudflare API token scope and storage | Delegated to implementation (user said "You decide") | Use environment variable `CLOUDFLARE_TUNNEL_TOKEN` with scopes: `Account:Cloudflare Tunnel:Edit`, `Zone:DNS:Read`. Fallback: check `~/.cloudflared/credentials.json` for existing tunnel credentials. Document required scopes in setup guide. Environment variable chosen for consistency with Vrooli's existing secrets pattern and to avoid building a custom secrets store. |
| Q6: Monitoring interval | A: 60s default — conservative, low overhead, approx 3 min detection time | Default monitoring loop interval is 60 seconds. With 3 consecutive failure threshold for external probes, worst-case detection time is ~3 minutes. Configurable via `config` table in SQLite. |

## Suggestions Integrated

### Accepted
No suggestions were submitted for this item.

### Not Accepted
No suggestions were submitted for this item.

## Refined Scope

### Included (Must Have — P0)
- **Route manifest**: Declarative `routes` table in SQLite mapping subdomains to scenarios and expected ports, seeded via interactive CLI on first run
- **Both management modes from day one**: Auto-detect local vs remote mode on startup; support reading/writing config in both modes
- **Port compliance auditor**: Scan scenario `service.json` files to verify fixed ports match the route manifest
- **Tunnel health monitor**: Monitor cloudflared via systemd status, Prometheus `/metrics`, and `/ready` endpoint at 60s intervals
- **Internal liveness probes**: HTTP probe each published route's local port
- **External liveness probes**: HTTP probe each published route via its public URL (3 consecutive failures before classifying as tunnel-issue)
- **Auto-recovery engine**: Restart cloudflared with exponential backoff and circuit breaker; use lock file for coordination with vrooli-autoheal; **must not require sudo** (explore D-Bus/polkit/user systemd first; if sudo unavoidable, provide UX-grantable mechanism)
- **CLI commands**: `status`, `routes`, `probe`, `audit`, `init` (interactive first-run seeding), `recover` (manual trigger)
- **Cloudflare API integration**: Read/write tunnel configuration via API for remote mode (hot-reload, no restart needed)
- **Local config management**: Generate/maintain `~/.cloudflared/config.yml` from route manifest for local mode

### Included (Should Have — P1)
- **Management mode switching**: CLI command to switch between remote and local modes with config migration
- **Web UI dashboard**: React dashboard showing tunnel status, per-route health, metrics charts, recovery events
- **Prometheus metrics scraping**: Scrape and store time-series data from cloudflared's metrics endpoint
- **Recovery event log**: Persist recovery attempts with timestamps, actions, outcomes
- **Failure classification**: Categorize as tunnel-down, scenario-down, cloudflare-outage, dns-failure, config-drift
- **Degraded mode detection**: Alert when HA connections < 4 or RTT spikes
- **Route config sync CLI**: `tunnel-manager config sync` to push manifest to cloudflared config or Cloudflare API

### Excluded (Out of Scope)
- **cloudflared installation** — Handled by Vrooli setup scripts (`scripts/lib/service/cloudflare-tunnel.sh`)
- **Scenario lifecycle management** — Starting/stopping scenarios is out of scope; only monitors their ports
- **Cloudflare Access/Zero Trust policies** — Adds significant complexity; most routes are currently public (deferred idea D1)
- **Full Cloudflare dashboard replacement** — Tunnel Manager provides operational monitoring, not a full admin UI
- **Multi-server tunnel coordination** — Requires shared manifest and replica-aware routing; deferred to future architecture phase (D3)

### Deferred (Future)
- **Multi-tunnel support** — Manage multiple tunnels for different domains (P2)
- **Automatic route registration** — Auto-add routes when new fixed-port scenarios start (P2)
- **Webhook/notification alerts** — Slack/Discord/email on failures (P2)
- **Per-route bandwidth tracking** — Requires log parsing or Cloudflare analytics API (D2)
- **Mobile status view** — Lightweight mobile-friendly status page (P2)

## Implementation Notes

### Technical Approach
- **Language**: Go for API and CLI (matches cloudflared ecosystem, consistent with Vrooli patterns)
- **UI**: React + Vite + TypeScript (standard Vrooli UI stack)
- **Storage**: SQLite — self-contained, no external database dependency. Critical for infrastructure that must work even when other resources are down
- **Mode detection**: On startup, check for local config file existence and Cloudflare API reachability to auto-detect management mode
- **Lock file coordination**: Use `flock(2)` on `/tmp/cloudflared-restart.lock` for atomic restart coordination with vrooli-autoheal

### Privilege Escalation Strategy (Critical — from Q1)
The recovery engine needs to restart cloudflared. The approach hierarchy:
1. **Preferred: D-Bus systemd API** — Go's `godbus` package can call `org.freedesktop.systemd1.Manager.RestartUnit()`. If the vrooli user has a polkit rule allowing this, no sudo is needed.
2. **Fallback: User-level systemd** — If cloudflared can run as a user service (`systemctl --user`), no privileges needed at all.
3. **Last resort: UX-grantable sudo** — If D-Bus/polkit aren't viable, the UI/CLI must provide a setup flow that configures a narrow sudoers NOPASSWD rule (`vrooli ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart cloudflared`). The user must be able to approve this from the dashboard without physical terminal access.

### Integration Points
- **cloudflared**: Prometheus metrics (`:20241/metrics`), readiness (`:20241/ready`), systemd unit management, config file or API
- **Cloudflare API v4**: `PUT /accounts/{id}/cfd_tunnel/{tunnel_id}/configurations` for remote config (read-modify-write pattern; no PATCH endpoint per GitHub issue #1437)
- **Scenario service.json files**: Read `scenarios/{name}/.vrooli/service.json` for port auditing
- **vrooli-autoheal**: Shared lock file for restart coordination; autoheal's `infra-cloudflared` check continues as defense-in-depth
- **app-monitor**: Monitored as one of the published routes (port 35000)

### Dependencies
- **cloudflared** (systemd service, must be running): Core dependency — the daemon being managed
- **Cloudflare API token** (remote mode only): `CLOUDFLARE_TUNNEL_TOKEN` env var with `Account:Cloudflare Tunnel:Edit` + `Zone:DNS:Read` scopes
- **Explicit metrics port**: cloudflared systemd unit must set `--metrics 127.0.0.1:20241` for deterministic scraping (currently binds to first available in 20241-20245 range)
- **At least one published route**: Needed to validate against during initial setup

### Risks & Mitigations
| Risk | Mitigation |
|------|------------|
| Sudo requirement for cloudflared restart | Explore D-Bus/polkit first; if unavoidable, provide UX-grantable mechanism (hard requirement from user) |
| Cloudflare API rate limiting | Cache API responses, batch config updates, respect rate limit headers |
| External probe false positives | Require 3 consecutive failures before classification; configurable threshold |
| Restart loops during Cloudflare outages | Circuit breaker: max 5 consecutive restart attempts before stopping auto-recovery |
| Cloudflare API read-modify-write race | Lock around API config updates; retry on conflict |
| cloudflared metrics port non-deterministic | Prerequisite: configure explicit `--metrics` port in systemd unit before development |

## Success Criteria
- [ ] Route manifest persists in SQLite and can be seeded interactively from current cloudflared config
- [ ] Both local and remote management modes work, with auto-detection on startup
- [ ] Port auditor correctly identifies mismatches between manifest and scenario service.json files
- [ ] Internal and external probes detect route failures within 3 minutes (60s interval × 3 failures)
- [ ] Recovery engine restarts cloudflared without requiring manual sudo (D-Bus/polkit or UX-grantable)
- [ ] Lock file prevents concurrent restarts between tunnel-manager and vrooli-autoheal
- [ ] CLI commands (status, routes, probe, audit, init, recover) produce correct, actionable output
- [ ] Cloudflare API integration can read and write tunnel configuration in remote mode
- [ ] Local mode generates valid config.yml and triggers cloudflared restart on changes

## Readiness Gate
- [x] All critical questions answered (Q1-Q6 all have answers)
- [x] Scope clearly defined (included/excluded/deferred sections above)
- [x] No unresolved conflicts (all sources consistent)
- [x] Technical approach viable (research confirms feasibility)
- [x] Important questions answered (all answered)
- [x] Archive materials reviewed (docs/ARCHITECTURE.md, docs/PROBLEMS.md, docs/RESEARCH.md, PRD.md, README.md, requirements/ all incorporated)

**Ready for processing:** Yes

## Staging Artifacts Produced
- `enhance/prd-context.md` — Synthesized PRD context incorporating all clarification answers, updated scope, and privilege escalation strategy
- `enhance/requirements-context.md` — Requirements context with validation approach, technical constraints, and dependency details
- `enhance/doc-outlines.md` — Documentation outlines for README, RESEARCH, PROBLEMS, and PROGRESS entries
