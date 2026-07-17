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

> The control plane, the cross-compiled node-agent, the OS-native
> service install, and the one-shot onboarding path are **built and
> tested**. Deployment mechanics below note where a step is real versus
> where a convenience (installer wrappers, code-signing, automated
> backup wiring) is still intended-but-unimplemented.

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
| Local Vrooli stack (control plane on Linux) | supported | Vrooli lifecycle, Go, Node/pnpm, SQLite path; owner token (cli-core `configure token` or `VROOLI_BRIDGE_API_TOKEN`) | Built and running as an ordinary Vrooli scenario. |
| Per-node agent service (Linux) | supported | Full Vrooli install with `vrooli` CLI on the node; node-agent installed as a systemd user service (requires linger) | Cross-compiled agent + `vrooli-bridge-agent service install` + one-shot onboarding are built and tested. Per-OS installer wrappers and code-signing are still convenience gaps. |
| Per-node agent service (macOS) | build-verified | Full Vrooli install with `vrooli` CLI on the node; node-agent installed as a launchd LaunchAgent (requires a logged-in user) | Darwin builds and Linux-runnable launchd contract tests pass. Real-Mac lifecycle, reconnect, and onboarding evidence are required before support can be claimed; see the [platform support matrix](../../../../docs/reference/platform-support.md). |
| Per-node agent service (Windows) | gated | Full Vrooli install with `vrooli` CLI on the node; node-agent installed as a Windows Service | Service unit is render-only (`sc.exe create` argv); live install/onboarding on Windows is not yet exercised. |
| Control plane on macOS | build-verified | Vrooli-the-platform installable/runnable on macOS | Real-Mac qualification is still required; Bridge does not add a separate platform gate. See the [platform support matrix](../../../../docs/reference/platform-support.md). |
| Control plane on Windows | gated P2 | Vrooli-the-platform installable/runnable on Windows | Native Windows full-project lifecycle remains outside the current qualification. |

### Bridge advertised endpoint

Bridge reserves API port `18767`. The endpoint selection order is an explicit
saved owner configuration, `BRIDGE_CONTROL_PLANE_URL`, `BRIDGE_TUNNEL_URL`,
then a derived LAN address. Set and inspect the saved default through
`vrooli-bridge readiness configure` and `vrooli-bridge readiness status`.
`swarminator.local` is only a candidate name an operator may choose; Bridge
never enables mDNS or silently changes host naming. Tunnel mode is for
off-LAN/segmented nodes and is never selected automatically after a LAN block.
| Managed cloud / SaaS | out of scope (v1) | — | Bridge is single-owner fleet infrastructure, not multi-tenant SaaS. |

Today's deployment tier is the **Tier 1 local stack** described in the
[Deployment Hub](../../../../docs/deployment/README.md): the control
plane runs as an ordinary Vrooli scenario, and each node runs a full
Vrooli install plus the node-agent service.

## Runtime Requirements

### Control plane

- API port: fixed at **18767** by the Bridge service manifest (and injected as
  `API_PORT`). This stable control-plane port is the only LAN firewall port a
  candidate needs; lifecycle fails rather than silently moving it on collision.
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

Onboarding a fresh node is one durable, control-plane-driven operation —
there is **no manual per-node installer to run**. Two equivalent surfaces
exist:

- **One-shot onboarding (recommended).** The owner points the control
  plane at a raw SSH host (`vrooli-bridge onboard start …`, or the UI
  OnboardNodeForm). The `onboard` domain establishes passwordless SSH
  first-touch and drives either source mode. With `--source working-tree`, it
  ships the live tree, detects the node target, cross-builds `vrooli`,
  `vrooli-bridge`, and the node-agent on the control plane, transfers them with
  freshness sidecars, pushes `bootstrap/bootstrap.sh`, and issues a single-use
  pairing code server-side (injected over SSH stdin, never argv/logs),
  runs the script, and confirms the node is ONLINE — as one durable,
  cancellable, restart-reconciled `OnboardingOp` the operator blocks on
  with `onboard watch`. See
  [`RUNBOOK.md`](RUNBOOK.md#onboarding-a-node-one-shot) and
  [`../concepts/FLOWS.md`](../concepts/FLOWS.md#one-shot-node-onboarding).
- **Manual bootstrap (fallback / air-gapped first touch).** Run
  `bootstrap/bootstrap.sh` directly on the node with
  `BRIDGE_PAIRING_CODE` in the environment. The script installs **only**
  the clone prerequisites (`git` and `curl`) and delegates all heavier
  provisioning to `vrooli setup`, which is the **sole machine-provisioning
  authority** — bootstrap never duplicates a toolchain install. It clones
  Vrooli at a pinned revision, runs `vrooli setup` (**elevated** when
  passwordless sudo is available — the one-shot path provisions a
  `sudoers.d` drop-in at first touch; otherwise setup runs unprivileged and
  names the skipped root-required requirements loudly), builds the agent +
  CLI, generates the node key, redeems the pairing code (pinning the
  control-plane key **before** the code is burned), installs the OS-native
  service, enables headless auto-start, and verifies the dial-out channel
  is live. It is idempotent — a re-run after a partial failure converges,
  and a fully-onboarded node re-run changes nothing. Marker grammar, step
  list, and exit codes are documented in
  [`../../bootstrap/README.md`](../../bootstrap/README.md).

The working-tree one-shot path never pulls a GitHub release or builds on the
node: release binaries cannot represent local uncommitted work. Its transferred
Vrooli sidecar is computed from the same tree sent over SSH, so setup runs before
the node has Go and installs Go afterward. The pinned/manual fallback remains
available for fetchable revisions.

Everything after onboarding is remote.

## Node-agent

The node-agent is the cross-compiled client installed on each trusted node
(OT-P0-007). It holds a dial-out channel to the control plane, runs allowlisted
jobs as the **non-privileged runner** (`internal/exec`), and runs provisioning
through the **structurally separate privileged helper** (`internal/privsep`) —
two distinct OS principals, never one flagged process (DECISIONS.md two trust
tiers).

**Service install (built — `agent/internal/service`).** The agent both renders
and installs its own platform-native background-service unit. The rendered unit
and the installed unit share one `serviceDefinition`, so the running service argv
byte-matches what `--print-service-unit` prints.

- `vrooli-bridge-agent service install|status|uninstall` — install writes the
  native unit to the OS location and enables it; status reports whether the unit
  is loaded/running; uninstall stops and removes it. Install is idempotent (a
  re-run converges to the same enabled unit).
- `vrooli-bridge-agent --print-service-unit …` still emits the native unit as
  text (for review or hand-install) without touching the system.
- Native unit per OS (`service.NewManager()` selects by `runtime.GOOS`; there are
  no scattered GOOS checks and no hardcoded paths):
  - **Linux** → a `systemctl --user` unit `vrooli-bridge-agent.service` under
    `~/.config/systemd/user`. Headless auto-start requires
    `loginctl enable-linger <user>` (the onboarding path runs this in the
    `autostart` step).
  - **macOS** → a launchd LaunchAgent labelled
    `com.vrooli.bridge.vrooli-bridge-agent` under `~/Library/LaunchAgents`. A
    LaunchAgent runs only while its user is GUI-logged-in, so a headless Mac mini
    needs **auto-login enabled** (there is no linger equivalent — see
    [`RUNBOOK.md`](RUNBOOK.md#mac-mini-onboarding)).
  - **Windows** → render-only today (`sc.exe create … binPath= … start= auto`
    argv); live install is not yet exercised.
- The two trust tiers install as distinct OS principals: the non-privileged
  runner as its service user, and the **privileged provisioning helper** as its
  own separately-installed unit.

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

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures, onboarding, backup/restore, fleet maintenance
- [`../../bootstrap/README.md`](../../bootstrap/README.md) — node bootstrap script contract (marker grammar, steps, exit codes)
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — fleet health and telemetry signals
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — control plane / node-agent / dial-out design
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — composed scenarios (device-sync-hub, test-genie, tunnel-manager, scenario-authenticator)
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — trust tiers, mutual auth, allowlist posture
- [`../reference/configuration.md`](../reference/configuration.md) — env vars and lifecycle config
- [`../../PRD.md`](../../PRD.md) — scope, operational targets, launch sequencing
