# Deployment — Vrooli Bridge

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness for the bridge control
plane and the per-node agent service.

Bridge has two distinct deployable units:

- **Control plane** — the bridge scenario (Go API + React UI + Go CLI)
  running as a normal Vrooli scenario inside the owner's Vrooli stack.
- **Node-agent** — a single cross-compiled Go binary installed on each
  trusted node as an OS-native background service. The node-agent dials
  **out** to the control plane and holds a persistent channel; nodes
  expose **no inbound ports** (NAT/firewall-proof, like Tailscale or
  GitHub Actions runners). A node is a full Vrooli install plus the
  agent, because a node is a real build/test environment, not a thin
  runner.

> No product code exists yet. This is the documentation-first
> foundation; deployment mechanics below describe the intended shape and
> are marked where they are not yet implemented.

## Purpose Of This Document

Use this document to answer:

- Where can the control plane and node-agents run?
- What runtime assumptions must hold on the control plane and on each node?
- How is the cross-compiled node-agent packaged per OS?
- What must pass before a control-plane + agent rollout?
- How do we roll back a control-plane change or a bad node provision?

## Supported Tiers

| Tier | Status | Requirements | Blockers |
|---|---|---|---|
| Local Vrooli stack (control plane on Linux) | target | Vrooli lifecycle, Go, Node/pnpm, SQLite path; scenario-authenticator for owner auth | Not yet implemented; this is the documentation-first foundation. |
| Per-node agent service (Linux / macOS / Windows) | target | Full Vrooli install with root `vrooli` CLI on the node; node-agent installed as systemd / launchd / Windows Service | Cross-compiled agent + per-OS service installers not yet built. |
| Control plane on macOS / Windows | gated P2 | Vrooli-the-platform installable/runnable on those OSes | Depends on platform portability work outside this scenario (OT-P2-001). Bridge is written cross-platform so it is never the blocker. |
| Managed cloud / SaaS | out of scope (v1) | — | Bridge is single-owner fleet infrastructure, not multi-tenant SaaS. |

Today's deployment tier is the **Tier 1 local stack** described in the
[Deployment Hub](../../../../docs/deployment/README.md): the control
plane runs as an ordinary Vrooli scenario, and each node runs a full
Vrooli install plus the node-agent service.

## Runtime Requirements

### Control plane

- API port: assigned by lifecycle as `API_PORT` (Connect-RPC + SSE dial-out edge).
- UI port: assigned by lifecycle as `UI_PORT` (React fleet dashboard).
- Storage: SQLite via `api-core/storage` (`SQLITE_PATH`) holding control-plane
  metadata — nodes, pairings, capability snapshots, jobs, dispatch and
  provisioning audit, and version history. Schema is Postgres-compatible
  for forward scale.
- Owner identity: scenario-authenticator (fail-closed) gates control-plane access.
- Off-LAN reach: tunnel-manager supplies the public tunnel URL that
  off-LAN node-agents dial; on a trusted LAN, agents reach the control
  plane directly (mDNS auto-discovery is a P1 convenience).
- No third-party runtime resource is required for the core control plane.

### Node

- A full Vrooli install with the root `vrooli` CLI present (bootstrap can
  install this as part of provisioning).
- The node-agent service, plus an **outbound** path to the control plane
  (direct on LAN, or via the control plane's tunnel URL off-LAN). No
  inbound ports are opened on the node.
- Toolchain/runtime headroom appropriate to the work the node accepts
  (the agent self-reports readiness: toolchain present, disk headroom,
  container runtime up).

## Packaging

| Surface | Packaging Details |
|---|---|
| Control-plane API | Go binary built by the scenario lifecycle. |
| Control-plane UI | Vite production bundle served by `ui/server.js`. |
| Control-plane CLI | Go CLI installed through scenario manifest install hooks; full headless parity with the UI. |
| Proto | Wire contracts for control-plane↔CLI and control-plane↔node live under `packages/proto/schemas/vrooli-bridge/`; generated clients are shared artifacts. The node↔control-plane protocol is proto-versioned with a `DiscardUnknown` backward-compat policy. |
| Node-agent | One Go codebase cross-compiled `CGO_ENABLED=0` for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, and `windows/arm64`. Cross-compilation is **built** (`agent/Makefile` `matrix` target + the `agent/build/crosscompile_test.sh` gate, all six targets green). Each build registers as the platform-native service via the rendered unit (see Node-agent below). Per-OS installer wrappers and code-signing are not yet implemented. |

A bootstrap installer is the single intended manual touch on a new node:
it installs Vrooli, runs `vrooli setup`, installs the node-agent service,
and pairs the node to the control plane out-of-band (pairing code/token).
Everything after bootstrap is remote.

## Node-agent

The node-agent is the cross-compiled client installed on each trusted node
(OT-P0-007). It holds a dial-out channel to the control plane, runs allowlisted
jobs as the **non-privileged runner** (`internal/exec`), and runs provisioning
through the **structurally separate privileged helper** (`internal/privsep`) —
two distinct OS principals, never one flagged process (DECISIONS.md two trust
tiers).

**Service install (built — `agent/internal/service`).** The agent renders its own
platform-native background-service unit; the bootstrap installer writes it to the
OS unit location and enables it:

- `vrooli-bridge-agent --print-service-unit --control-plane-url <url> --node-id <id> [--service-user vrooli-agent]`
  emits the native unit for the host OS:
  - **Linux** → a systemd unit (`[Service] ExecStart=…`, `Restart=on-failure`,
    `User=<service-user>`, `WantedBy=multi-user.target`).
  - **macOS** → a launchd plist (`ProgramArguments` argv, `KeepAlive`,
    `UserName`).
  - **Windows** → the `sc.exe create … binPath= … start= auto` argv.
- The renderer selects by `platform.NativeServiceManager()` (a pure function of
  `runtime.GOOS`); there are no scattered GOOS checks and no hardcoded POSIX
  paths — every path comes from the resolved config.
- The **privileged provisioning helper** installs as its own unit under a
  separate principal (`--service-user vrooli-provisioner`), so the two trust
  tiers are distinct OS users at install time.

**Provisioning (built — `internal/privsep` + control-plane `internal/provision`).**
A `provision sync <node> --revision R` brings the node to revision R via a typed
step plan — `git fetch` → `git checkout R` → `vrooli setup` — never a shell
string. It is idempotent (re-running converges) and rolls back to the prior
revision on a failed setup so a bad revision never strands the node. Progress,
the resulting version, and the terminal outcome stream back as durable, audited
`ProvisionEvent`s the operator blocks on with `provision wait <op-id>`.

## Release Checklist

### Control plane

- [ ] `make setup` passes.
- [ ] `make test` passes.
- [ ] PRD operational targets have linked requirements.
- [ ] Proto wire contracts are regenerated and drift-checked; the
      node↔control-plane protocol version is bumped if the wire changed,
      and the compatibility window for older agents is documented.
- [ ] `docs/manifest.json` maturity reflects current docs.
- [ ] `RUNBOOK.md`, `OBSERVABILITY.md`, and `SECURITY.md` are active or
      explicitly not-applicable.

### Node-agent / fleet rollout

- [ ] New agent build is cross-compiled and packaged for every supported
      OS, and protocol-compatible with the control plane being shipped.
- [ ] Roll the agent to a canary node first; confirm dial-out presence,
      health, and a no-op typed job succeed before fleet-wide rollout.
- [ ] Compatibility gate: nodes on an agent too old for the new protocol
      are flagged "needs update" and excluded from incompatible work
      rather than dispatched to (OT-P1-001).
- [ ] Provisioning to a target revision R is idempotent and re-runnable;
      verify a node already at R is a clean no-op.
- [ ] Audit trail records the rollout (who, which nodes, outcome).

## Rollback

### Control plane

Control-plane rollback is source-control based: revert the bridge
scenario to the prior project revision and restart via the lifecycle
(`make restart`). If a release changed the SQLite schema, restore the
control-plane database from backup before downgrading — see the
[`RUNBOOK.md`](RUNBOOK.md) backup/restore procedures. If the wire
protocol was bumped, downgrading the control plane may strand nodes on a
newer agent; pin or roll those agents back in step.

### Node

A failed provision must not strand a node. The privileged provisioning
tier is designed to automatically roll a node back to its prior revision
when `vrooli setup` fails (OT-P1-001), leaving the node on a known-good
build. For a bad agent rollout, re-provision the affected node(s) to a
known-good agent + revision, or revoke and re-bootstrap a node whose
trust or install is in doubt (revocation atomically kills its job and
provisioning rights). Automatic re-provisioning is the intended path;
manual re-bootstrap is the fallback. Self-healing re-provisioning of
drifted nodes is a later capability (OT-P2-004).

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures, backup/restore, fleet maintenance
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — fleet health and telemetry signals
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — control plane / node-agent / dial-out design
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — composed scenarios (device-sync-hub, test-genie, tunnel-manager, scenario-authenticator)
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — trust tiers, mutual auth, allowlist posture
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
- [`../../PRD.md`](../../PRD.md) — scope, operational targets, launch sequencing
