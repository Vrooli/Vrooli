# Runbook — Tunnel Manager

This document records operator procedures for running, diagnosing,
recovering, and maintaining the scenario.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What checks should I run during an incident?
- How do I back up or restore state?
- Where should operational issues be recorded?

> **Status (Phase 1, documentation-first):** No product code exists yet
> beyond the template scaffold. The `tunnel-manager ...` commands below
> are the **planned CLI surface** (PRD OT-P0-012); they are not yet
> implemented. Lifecycle commands (`make`, `vrooli scenario`) work today.
> Runs in the Tier 1 local stack alongside `cloudflared` (systemd).

## Start / Stop / Status

Use lifecycle-managed commands from the scenario directory:

```bash
make setup
make start
make status
make logs
make stop
make test
```

Do not start API/UI binaries directly. The lifecycle owns process
naming, ports, health checks, and logs.

## Operator Procedures (planned CLI surface)

All commands support proto-typed `--json`. Domains map to the planned
CLI verbs (see [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)).

| Goal | Command | Notes |
|---|---|---|
| Check tunnel status | `tunnel-manager tunnel status` | cloudflared systemd state, `/ready`, HA connections, degraded-mode signal. |
| List / inspect exposure | `tunnel-manager routes list` / `tunnel-manager exposure list` | Reads the exposure manifest (SSOT) or reconciled exposure state. |
| Expose / lease a scenario | `tunnel-manager exposure expose <scenario>` | Creates a LEASED route, ensures the scenario is running, and requests ingress. Default TTL ≈ 1 week. |
| Extend a lease | `tunnel-manager exposure extend <lease_id>` | Pushes out `expires_at`; expired leases are auto-reaped unless the scenario is also CORE. |
| Revoke a lease | `tunnel-manager exposure revoke <lease_id>` | Removes the LEASED route + ingress (CORE routes are never revoked this way). |
| Run probes | `tunnel-manager probes run` | Internal probes the local port; external probes the public URL end-to-end. |
| Run port audit | `tunnel-manager audit run` | Verifies each exposed scenario's `service.json` fixed UI port matches the manifest; reports mismatches/missing/ranged ports. |
| Manually trigger recovery | `tunnel-manager recovery run` | Forces a recovery cycle (restart cloudflared). Background recovery is opt-in with `TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED`; use manual recovery when escalating an incident. |
| Switch remote/local mode | `tunnel-manager config mode <remote\|local>` | Remote = Cloudflare API ingress (needs API token); local = generate `~/.cloudflared/config.yml`. Migrates ingress on switch. |
| Inspect / sync config | `tunnel-manager config get` / `tunnel-manager config sync` | Reconciles ingress with the manifest. |

### When auto-recovery trips the circuit breaker

The `recovery` engine uses exponential backoff and a circuit breaker.
Background evaluation is opt-in (`TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED`)
because acted evaluations restart cloudflared. When the breaker opens,
recovery attempts stop to avoid a restart storm:

1. `tunnel-manager tunnel status`, `tunnel-manager probes run`, and
   `tunnel-manager probes classify` — classify the failure. Current
   classes are healthy / tunnel-down / scenario-down / config-drift;
   Cloudflare outage and DNS failure require future signals.
2. If it is **config drift**, run `tunnel-manager config sync`.
3. Once root cause is addressed, `tunnel-manager recovery run --force true` resets and
   re-attempts; this clears the breaker on success.
4. Review the recovery event log (see
   [`OBSERVABILITY.md`](OBSERVABILITY.md)) before and after.

### Relationship with vrooli-autoheal (single owner)

Tunnel Manager is the **single authoritative owner of cloudflared
restart**. `vrooli-autoheal`'s cloudflared check is downgraded to
**alert-only** (defense-in-depth) so the two never duel over restarts. If
you observe autoheal restarting cloudflared, that is a misconfiguration —
confirm autoheal is alert-only before manually recovering. See
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in `../internal/PROBLEMS.md`. |
| API unhealthy | `/health`, SQLite path, API logs | Run `make setup`, verify writable data dir | Check `INTEGRATIONS.md` for dependency expectations. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `tunnel-manager status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

## Backup / Restore

The generated template uses local SQLite state. Product scenarios must
define backup and restore procedures before production deployment.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database | deferred | deferred | Define before deployment. |
| Blob files | deferred | deferred | Define if binary/blob domains remain. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
