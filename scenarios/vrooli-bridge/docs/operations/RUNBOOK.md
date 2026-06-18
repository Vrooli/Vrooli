# Runbook — Vrooli Bridge

This document records operator procedures for running, diagnosing,
recovering, and maintaining the bridge control plane and the fleet of
node-agents it manages.

Bridge is the fleet control plane for an owner's trusted Vrooli nodes.
Operations span two surfaces: the **control plane** (a normal Vrooli
scenario) and the **nodes** (each a full Vrooli install running the
node-agent service that dials out to the control plane). No product code
exists yet — procedures below describe the intended operating model and
note where a capability is not yet implemented.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the control plane?
- What do I check when a node goes offline, a provision fails, or a job hangs?
- How do I back up and restore the control-plane store?
- How do I pin a fleet revision, revoke a node, or prune history?
- Where should operational issues be recorded and escalated?

## Start / Stop / Status

### Control plane

Use lifecycle-managed commands from the scenario directory:

```bash
make setup
make start
make status
make logs
make stop
make test
```

The CLI alternative is `vrooli scenario start vrooli-bridge` /
`vrooli scenario status vrooli-bridge`. Do not start the API/UI binaries
directly — the lifecycle owns process naming, ports, health checks, and
logs.

### Nodes

Nodes are not started from the control-plane host. Each node-agent is
installed as an OS-native service (systemd / launchd / Windows Service)
and starts with the machine; it dials out to the control plane and
maintains presence. Operate nodes **through** the control plane:

- List fleet + presence/health: `vrooli-bridge nodes list` (planned).
- Inspect one node: `vrooli-bridge nodes show <node>` (planned).
- Revoke a node: `vrooli-bridge nodes revoke <node>` (planned) — atomically
  kills its job and provisioning rights.

On the node itself, the agent service is managed with the platform's
service manager (`systemctl`, `launchctl`, or the Windows Service
control panel) only for local recovery; routine fleet operations go
through the control plane.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Control plane does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). |
| Node shows offline / dial-out channel dropped | Node presence in `nodes list`; on the node, agent service status (`systemctl`/`launchctl`/Service mgr) and its logs; control-plane reachability (LAN direct, or tunnel-manager URL off-LAN) | Restart the agent service on the node; confirm outbound path to the control plane; if off-LAN, verify tunnel-manager tunnel is up | If a node repeatedly drops, suspect tunnel or network; check tunnel-manager and `../internal/PROBLEMS.md`. |
| Provision fails (sync to revision R) | Provisioning audit entry, node toolchain/disk headroom, `vrooli setup` output captured back at the control plane | Re-run provisioning (idempotent); the tier auto-rolls the node back to its prior revision on setup failure | If setup fails repeatedly on one OS, capture logs and escalate; review `../internal/SECURITY.md` for the privileged tier's expectations. |
| Job stuck / not completing | Job status in the control plane; the durable run is server-owned and re-attachable by id — block once with the wait verb, do **not** poll | Re-attach by run id; if genuinely wedged, abort the job (abort ≠ cancel) and inspect node-side run logs | Mirror test-genie discipline: one job per node at a time; a thrashing node points to scheduling/health issues. |
| Version drift across fleet | Per-node Vrooli revision in `nodes list`; protocol-compatibility flags | Pin the fleet to target revision R and re-provision drifted nodes; nodes on an incompatible agent are flagged "needs update" and excluded from incompatible work | If drift recurs, evaluate self-healing re-provisioning (OT-P2-004, not yet implemented). |
| Job rejected by node | Node's accepted verb-namespace scopes vs the dispatched `{scenario, verb}` | Expected behavior — bridge runs only allowlisted typed verbs; widen the node's scope deliberately if the verb should be permitted | Never work around the allowlist with raw shell; see `../internal/SECURITY.md`. |

## Backup / Restore

The control-plane store is SQLite via `api-core/storage` and is the
durable record of the fleet: nodes, pairings, capability snapshots, jobs,
dispatch/provisioning audit, and version history. Losing it loses fleet
identity and the audit trail, so it is the primary backup target.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| Control-plane SQLite store | Stop the control plane (or quiesce writes) and copy the SQLite file at `SQLITE_PATH`; retain per a configurable policy | Restore the file to `SQLITE_PATH` and restart the control plane | Procedure defined; automated backup wiring not yet implemented. |
| Job logs / result artifacts | Streamed back from nodes and retained per a configurable retention policy | Re-fetch from retained run records where present | Retention policy not yet implemented. |
| Build artifacts | Not stored by bridge — moved through device-sync-hub; back up there | Re-distribute via device-sync-hub | N/A to bridge's own store. |

Bridge does not keep large binaries in its own store; non-git artifacts
move through device-sync-hub's transport, so their durability is owned
there, not here.

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect control-plane logs | as needed | `make logs` |
| Pin / roll fleet revision | per release or on drift | Set target revision R and provision the fleet (or one node) toward it (planned `vrooli-bridge` provisioning verb) |
| Revoke a node | when a machine leaves the trust boundary | `vrooli-bridge nodes revoke <node>` (planned) — atomic kill of job + provisioning rights |
| Prune audit / run history | per retention policy | Trim old dispatch/provisioning audit and run records beyond the retention window (policy + tooling not yet implemented) |
| Rotate node pairing | on suspected compromise | Revoke and re-bootstrap the node with a fresh pairing token |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).
Anything touching remote execution, the typed-verb allowlist, the two
trust tiers, or mutual auth is security-sensitive — escalate via
[`../internal/SECURITY.md`](../internal/SECURITY.md) and do not work
around the allowlist with raw shell.

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers, packaging, release checklist, rollback
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — fleet health, logs, metrics, and signals
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — composed scenarios and dependencies
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — trust tiers, allowlist, mutual auth, audit
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../../PRD.md`](../../PRD.md) — scope and operational targets
