# Requirements Context: Tunnel Manager

> This document is a synthesized requirements context for `prd-control-tower requirements generate`.
> Sources: requirements/index.json (10 requirement modules), clarify/questions.json, docs/ARCHITECTURE.md, docs/RESEARCH.md

## Existing Requirements Modules

The following 10 requirement modules already exist and should be reviewed/updated during processing:

1. **01-route-manifest** — Route manifest data model and CRUD operations
2. **02-port-compliance** — Port compliance auditing against service.json files
3. **03-tunnel-health** — Tunnel health monitoring (systemd, Prometheus, /ready)
4. **04-liveness-probes** — Internal + external HTTP liveness probes
5. **05-auto-recovery** — Auto-recovery engine with backoff and circuit breaker
6. **06-cli-interface** — CLI commands (status, routes, probe, audit, etc.)
7. **07-cloudflare-api** — Cloudflare API v4 integration for remote mode
8. **08-local-config** — Local config.yml management
9. **09-web-dashboard** — React UI dashboard
10. **10-observability** — Metrics, logging, alerting

## Clarification-Driven Requirement Updates

The following clarification answers introduce new constraints that must be reflected in requirements:

### Recovery Engine (module 05) — Privilege Escalation
**From Q1**: The recovery engine must NOT assume sudo access. Implementation hierarchy:
1. Try D-Bus systemd API (`org.freedesktop.systemd1.Manager.RestartUnit()`) with polkit rule
2. Try user-level systemd (`systemctl --user`) if cloudflared can run as user service
3. Last resort: UX-grantable sudoers rule — the UI/CLI must provide a setup flow for this

**Validation**: Test recovery without sudo on a clean system. Verify D-Bus path works with polkit. Verify UX-grantable flow works end-to-end if sudo is needed.

### Dual Mode Support (modules 07 + 08) — Day One Requirement
**From Q2**: Both local and remote modes are P0 (not P1). Auto-detect on startup by:
- Checking for `~/.cloudflared/config.yml` existence (local indicators)
- Testing Cloudflare API reachability with provided token (remote indicators)
- Default to remote if both are available; fall back to local if API unreachable

**Validation**: Test startup with local-only config, remote-only config, and both present.

### Interactive Seeding (module 01 + 06) — CLI Init Command
**From Q3**: First-run experience must include interactive `tunnel-manager init`:
- Parse current cloudflared config (YAML or API) to discover existing routes
- Display each discovered route with details (subdomain, port, scenario mapping)
- Prompt user to confirm/edit/skip each route
- Persist confirmed routes to SQLite manifest

**Validation**: Test init with various existing configs (0 routes, 1 route, many routes). Test edit flow. Test skip flow.

### Lock File Coordination (module 05) — Cross-Scenario Safety
**From Q4**: Use `flock(2)` on `/tmp/cloudflared-restart.lock`:
- Acquire exclusive lock before any cloudflared restart
- Hold lock during restart + verification (wait for /ready to return 200)
- Release lock after verification completes or times out
- vrooli-autoheal must also use this same lock file

**Validation**: Test concurrent restart attempts from both tunnel-manager and a simulated autoheal process. Verify only one restart executes.

### Monitoring Interval (module 03 + 04)
**From Q6**: Default 60s monitoring interval. Configurable via SQLite `config` table.
- External probes: 3 consecutive failures before classification (worst case ~3 min detection)
- Internal probes: can act on single failure (local network is reliable)

**Validation**: Test with configurable intervals. Verify 3-failure threshold for external. Verify single-failure threshold for internal.

## Technical Constraints

### cloudflared Metrics Port
**From PROBLEMS.md Q2**: cloudflared binds metrics to first available port in 20241-20245 range. Must configure `--metrics 127.0.0.1:20241` explicitly in systemd unit as a prerequisite.

### Cloudflare API Limitation
**From RESEARCH.md**: No PATCH endpoint for individual route add/remove. Must use read-modify-write pattern for entire ingress configuration. Need locking around API config updates to prevent race conditions.

### External Probe Reliability
**From PROBLEMS.md Q3**: External probes traverse the full internet path. Transient failures are expected. Require 3 consecutive failures. Single failures logged but not acted upon.

## Validation Approach

### Unit Testing
- Route manifest CRUD operations
- Port compliance scanner logic (mock service.json files)
- Probe classification matrix (all 4 combinations: pass/pass, pass/fail, fail/fail, fail/pass)
- Recovery state machine transitions
- Circuit breaker behavior
- Lock file acquisition/release

### Integration Testing
- cloudflared Prometheus metrics scraping (real endpoint or mock)
- Cloudflare API read-modify-write cycle (mock API or test tunnel)
- service.json file scanning across scenario directories
- SQLite persistence (probe results, recovery events, metrics snapshots)

### End-to-End Testing
- Full monitoring loop: scrape → probe → evaluate → recover → persist
- Interactive CLI init flow
- Mode detection and switching
- Lock file coordination with simulated concurrent restarts

## Dependency Relationships

```
01-route-manifest ←── 02-port-compliance (reads manifest for expected ports)
01-route-manifest ←── 04-liveness-probes (reads manifest for probe targets)
01-route-manifest ←── 07-cloudflare-api (syncs manifest to API)
01-route-manifest ←── 08-local-config (syncs manifest to config.yml)
03-tunnel-health ──→ 05-auto-recovery (health failures trigger recovery)
04-liveness-probes ─→ 05-auto-recovery (probe failures trigger recovery)
05-auto-recovery ──→ 07-cloudflare-api OR 08-local-config (restart mechanism)
06-cli-interface ──→ ALL (CLI is the primary interaction surface)
09-web-dashboard ──→ ALL (dashboard displays all data)
10-observability ──→ 03-tunnel-health + 04-liveness-probes (metrics collection)
```
