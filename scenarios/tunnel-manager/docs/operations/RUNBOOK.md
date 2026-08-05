# Runbook — Tunnel Manager

This document records operator procedures for running, diagnosing,
recovering, and maintaining the scenario.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What checks should I run during an incident?
- How do I back up or restore state?
- Where should operational issues be recorded?

> **Status:** The API, CLI, UI, and background schedulers are implemented
> for the Tier 1 local stack. Run all lifecycle operations through
> `make`/`vrooli scenario`; use the `tunnel-manager ...` CLI commands
> below for operator workflows.

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

## Operator Procedures

All scenario commands support proto-typed `--json`. Domains map to the
CLI verbs in [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md).

| Goal | Command | Notes |
|---|---|---|
| Check tunnel status | `tunnel-manager tunnel status` | cloudflared systemd state, `/ready`, HA connections, degraded-mode signal. |
| List / inspect exposure | `tunnel-manager routes list` / `tunnel-manager exposure list` | Reads the exposure manifest (SSOT) or reconciled exposure state. |
| Expose / lease a scenario | `tunnel-manager exposure expose <scenario>` | Creates a LEASED route, ensures the scenario is running, and requests ingress. Default TTL ≈ 1 week. |
| Extend a lease | `tunnel-manager exposure extend <lease_id>` | Pushes out `expires_at`; expired leases are auto-reaped unless the scenario is also CORE. |
| Revoke a lease | `tunnel-manager exposure revoke <lease_id>` | Removes the LEASED route + ingress (CORE routes are never revoked this way). |
| Run probes | `tunnel-manager probes run` | Internal probes the local port; external probes the public URL end-to-end. |
| Run port audit | `tunnel-manager audit run` | Verifies each exposed scenario's `service.json` fixed UI port matches the manifest; reports mismatches/missing/ranged ports. |
| Manually trigger recovery | `tunnel-manager recovery run` | Forces a recovery cycle (`reset-failed` + restart cloudflared). Background recovery is **default-on** (opt out with `TUNNEL_MANAGER_RECOVERY_SCHEDULER_DISABLED=1`); use manual recovery when escalating an incident. |
| Configure Cloudflare credentials | `tunnel-manager config credentials-status` / `credentials-set` / `credentials-clear` | Reads and writes the canonical Vrooli credential authority; environment variables are not accepted as credential sources. |
| Switch remote/local mode | `tunnel-manager config mode --target <remote\|local>` | Remote = Cloudflare API ingress (needs complete credentials); local = generate `~/.cloudflared/config.yml`. **Pure — never writes ingress; run `config sync` after to apply.** |
| Inspect / sync config | `tunnel-manager config get` / `tunnel-manager config sync` | Additive reconcile: adds desired hostnames, preserves unmanaged/foreign ones. Add `--prune true` to remove orphaned entries. |
| Inspect ingress drift | `tunnel-manager drift list` | Classifies every live/desired/tracked hostname (managed/missing/external/orphaned/ignored/unmanaged). Read-only. |
| Adopt an unmanaged hostname | `tunnel-manager drift adopt <host> [--scenario <s> \| --target <url>]` | Brings drift under management as a scenario or external route + records ownership. |
| Acknowledge an external hostname | `tunnel-manager drift ignore <host> [--note <text>]` | Marks it IGNORED; reconcile never pushes or prunes it. |
| Remove one ingress hostname | `tunnel-manager drift prune <host>` | The only per-entry removal path; clears live ingress + ledger. |
| Add an external route | `tunnel-manager routes create --external --subdomain <s> --target <url>` | Exposes a non-scenario target through the tunnel; reconciles as `external`. |

### Auto-recovery: default-on, presence-gated, sudoers-backed

Background recovery is **default-on** (opt out with
`TUNNEL_MANAGER_RECOVERY_SCHEDULER_DISABLED=1`, symmetric with the probe and
exposure schedulers). Every minute the engine probes cloudflared's `/ready`
(`http://127.0.0.1:20241/ready`); after 3 consecutive failures it runs
`sudo systemctl reset-failed cloudflared && sudo systemctl restart cloudflared`
and polls `/ready` back to 200.

- **`reset-failed` before `restart`** — once cloudflared flaps past systemd's
  `StartLimitBurst` (5), systemd marks the unit failed and a bare
  `systemctl restart` is *rejected* until the start-limit is cleared. The
  `reset-failed` clears it; its own failure is non-fatal (a healthy unit has
  nothing to reset). This is exactly the slow/hung/flap-exhausted case TM
  covers that systemd's own `Restart=on-failure` cannot.
- **Tunnel-presence self-gate** — on a host with **no** `cloudflared.service`
  unit, recovery stays dormant (logs `no cloudflared unit present; recovery
  dormant`, status stays idle, no failures counted, no restart attempted). A
  cloudflared installed after start is picked up on the next tick without a
  scenario restart.
- **The sudoers grant is provisioned once, at setup.** tunnel-manager runs
  non-root; cloudflared is a root unit. `sudo vrooli setup` applies the
  `cloudflared_recovery_privileges` safeguard, which writes
  `/etc/sudoers.d/tunnel-manager` (mode 0440, visudo-validated) granting the
  invoking user NOPASSWD `systemctl restart cloudflared` + `reset-failed
  cloudflared` — exact argv, no wildcards. Without it the restart prompts for
  a password and fails non-interactively. Re-running setup is idempotent.
  - **Precondition:** `/ready` depends on cloudflared exposing metrics
    (`--metrics`/`TUNNEL_METRICS` on `:20241`). A token tunnel started without
    metrics makes `/ready` always-fail; the presence-gate still prevents
    flapping (unit present but never ready → 3 fails → restart → still not
    ready → backoff → circuit), but recovery cannot confirm success. Ensure
    metrics are enabled on hosts that rely on auto-recovery.

#### When auto-recovery trips the circuit breaker

The `recovery` engine uses exponential backoff and a circuit breaker. After
5 failed recoveries the breaker opens and attempts stop for a cooldown to
avoid a restart storm:

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

### Reviewing and resolving ingress drift

TM is non-destructive: it never removes ingress it does not own. After
switching to remote mode (or any time the live tunnel may have entries TM
did not author), reconcile the picture before applying:

1. `tunnel-manager drift list` — see every hostname classified. `unmanaged`
   entries are live but TM neither created nor acknowledged them.
2. For each `unmanaged` entry, decide:
   - It belongs to a scenario you want TM to manage →
     `tunnel-manager drift adopt <host> --scenario <name>` (the local port
     is read from the live target).
   - It points at a non-scenario service you want to keep →
     `tunnel-manager drift adopt <host> --target <url>` (becomes an
     external route).
   - It is legitimate but TM should never touch it →
     `tunnel-manager drift ignore <host> --note "<why>"`.
   - It is stale and should go → `tunnel-manager drift prune <host>`.
3. `tunnel-manager config sync` — additively publish the desired manifest.
   Foreign/ignored entries remain intact. Add `--prune true` only to clear
   **orphaned** entries (routes TM made that are now gone).

**Local-mode round-trip caveat:** in `local` mode, "live" is parsed from
`~/.cloudflared/config.yml`, and only the ingress shapes TM understands
(`- hostname:` / `service:` pairs) round-trip. A sync merges and preserves
those, but an entry written in a shape TM cannot parse will not survive a
rewrite. For full-fidelity drift handling on a tunnel with hand-authored
config, prefer `remote` mode (the Cloudflare API is the source of truth)
or back up `config.yml` before the first sync. The previous file is always
backed up to `config.yml.backup.<timestamp>` on write.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in `../internal/PROBLEMS.md`. |
| API unhealthy | `/health`, SQLite path, API logs | Run `make setup`, verify writable data dir | Check `INTEGRATIONS.md` for dependency expectations. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `tunnel-manager status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

## Backup / Restore

Tunnel Manager uses local SQLite state under `SQLITE_PATH`. Back up the
database before major host maintenance or before replacing the local
stack data directory.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database | Stop the scenario, copy the file referenced by `SQLITE_PATH` or `${SCENARIO_DATA_DIR}/tunnel-manager.db`, then restart. | Stop the scenario, restore the copied DB file to the configured path, ensure ownership/permissions match the runtime user, then `make start`. | active |
| Blob files | n/a | n/a | No blob domains today. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |
| Validate auto-recovery (induced-failure soak) | after recovery/cloudflared changes; operator-attended | see below |

### Induced-failure soak (verify the recovery loop end-to-end)

Proves detection → actuation → readiness end to end. Needs sudo and the
`cloudflared_recovery_privileges` grant already applied (`sudo vrooli setup`).

1. `vrooli scenario restart tunnel-manager`; confirm the recovery scheduler
   started (API log; no `recovery dormant` line — the cloudflared unit is
   present).
2. `sudo systemctl stop cloudflared`. A **clean stop** is not an
   `on-failure` exit, so systemd's `Restart=on-failure` won't mask it — this
   isolates TM's recovery.
3. Within ~3 evaluation ticks (~3 min) expect: detection → `reset-failed` +
   `restart` → `/ready` back to 200 → a new `recovery_events` row with
   `outcome=success` (`tunnel-manager recovery events` or the SQLite
   `recovery_events` table).
4. Confirm a tunnel hostname is reachable end-to-end through Cloudflare again.

If the breaker is open from prior testing, `tunnel-manager recovery run
--force true` resets it.

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
