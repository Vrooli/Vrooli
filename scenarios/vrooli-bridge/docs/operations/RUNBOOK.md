# Runbook — Vrooli Bridge

This document records operator procedures for running, diagnosing,
recovering, and maintaining the bridge control plane and the fleet of
node-agents it manages.

Bridge is the fleet control plane for an owner's trusted Vrooli nodes.
Operations span two surfaces: the **control plane** (a normal Vrooli
scenario) and the **nodes** (each a full Vrooli install running the
node-agent service that dials out to the control plane). The control
plane, node-agent, OS-native service install, and one-shot onboarding are
built and tested; procedures below note where a step is real versus where
a capability (automated backup wiring, retention pruning) is still
intended-but-unimplemented.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the control plane?
- How do I onboard a new node (including a headless Mac mini)?
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

- List fleet + presence/health: `vrooli-bridge nodes list`.
- Inspect one node: `vrooli-bridge nodes get <node>`.
- Revoke a node: `vrooli-bridge nodes revoke <node>` — atomically kills its
  job and provisioning rights.

All owner-gated verbs require an owner token. Set one once with
`vrooli-bridge configure token "<owner-token>"` (or export
`VROOLI_BRIDGE_API_TOKEN`) — the cli-core framework token source. In the
**UI**, the console shows a sign-in screen instead: sign in (or create the
owner account) and it proxies same-origin to scenario-authenticator via the
bridge's `IdentityService`, keeping the JWT in browser storage until you sign
out; an expired token returns you to the sign-in screen. To add a
new node, use **onboarding** (below) rather than a manual per-node installer.

On the node itself, the agent service is managed with the platform's
service manager (`systemctl`, `launchctl`, or the Windows Service
control panel) only for local recovery; routine fleet operations go
through the control plane.

## Onboarding A Node (one-shot)

Onboarding turns a raw, SSH-reachable host into a paired, ONLINE,
auto-starting fleet node in **one durable operation** — there is no
manual per-node installer to run. The `onboard` domain drives SSH
first-touch → push `bootstrap/bootstrap.sh` → issue a single-use pairing
code server-side (injected over SSH stdin, never argv/logs) → run the
script → confirm the node is ONLINE. The op is server-owned: it survives
your client disconnecting and is re-attachable by id.

CLI path (owner-authenticated; `configure token` first):

```bash
# `start` NEVER prompts. Supply the SSH password one of three ways (or the
# UI onboard form on the fleet dashboard — the equivalent browser path):
#   --password-stdin       pipe it in (e.g. from `read -s` or a secret manager)
#   --prompt-password      explicit opt-in masked TTY prompt
#   $BRIDGE_SSH_PASSWORD   ambient env var for programmatic runs
# With none of these, the host is assumed to already trust the bridge key.
# The password rides once in the request body — never argv, never stored.
# Passwordless sudo is provisioned at first touch by default
# (--provision-sudo); pass --no-provision-sudo to opt out and let
# root-required setup steps degrade loudly.
read -rs SSH_PW && printf '%s' "$SSH_PW" | vrooli-bridge onboard start \
  --host mini-01.local --user admin --password-stdin && unset SSH_PW

# Shape the node's setup with an explicit profile (else its own defaults apply):
vrooli-bridge onboard start --host mini-01.local --user admin --prompt-password --setup-environment production --setup-resources enabled

# Block ONCE on the terminal outcome (never poll); exits non-zero on failure:
vrooli-bridge onboard watch "<op-id>"

# Inspect full step history at any time:
vrooli-bridge onboard status "<op-id>"
vrooli-bridge onboard list --host mini-01.local
```

Key flags: `--revision` (default `@cp` = the control plane's current
commit; pass an explicit already-pushed SHA/branch when HEAD is
unpushed), `--name`, `--capabilities`, `--control-plane-url` (the dial-back
URL the node pairs to; defaults to the server's `$BRIDGE_CONTROL_PLANE_URL`,
else the control plane's own derived LAN address — zero configuration
required), `--verify-timeout`, `--skip-setup`, `--skip-prereqs`. The UI path
is the **OnboardNodeForm** on the fleet dashboard (same durable op, live step
states, failure-taxonomy rendering), including the same password,
control-plane-URL, setup-profile, and source-mode inputs.

**Elevated, profile-driven setup.** The node-side `vrooli setup` is the sole
machine-provisioning authority. When passwordless sudo is available on the node
(provision it at onboarding via `--provision-sudo`, the default), setup runs
**under sudo** so no requirement is skipped for privilege. Without it, setup runs
unprivileged and the **skipped root-required requirements are named loudly** in
the `setup` step detail — a warning in the op step events / `onboard watch` /
UI, not a failure. The setup profile is operator-chosen and reaches the node's
`make setup SETUP_ARGS='…'`:

| Flag (CLI + UI advanced options) | Maps to | Values |
|------|---------|--------|
| `--setup-environment` | `vrooli setup --environment` | `development` \| `production` \| `minimal` |
| `--setup-resources`   | `vrooli setup --resources`   | `enabled` \| `none` \| `<comma-list>` |
| `--setup-scenarios`   | `vrooli setup --scenarios`   | `none` \| `all` \| `<comma-list>` |
| `--include-optional`  | `vrooli setup --include-optional` | (flag) also apply optional safeguards |

**Default:** all four are left **empty/off**, the sensible fleet posture — the
node falls through to `vrooli setup`'s own defaults (development environment,
default resources) and the bootstrap does not reshape a node's setup unless you
ask. Every value is metacharacter-validated at the API boundary (no
shell-injectable token can reach the node), and the setup sentinel is keyed on
the profile, so changing it re-runs setup while an identical profile no-ops.

**Source mode — pinned vs working-tree.** By default the node clones the
`--revision` from the clone remote, which must be **pushed** (the control-plane
preflight hard-fails an unpushed commit). For owner **development/validation**,
`--source working-tree` ships the control plane's **local working tree** over SSH
so uncommitted work onboards without a commit or push:

```bash
vrooli-bridge onboard start --host mini-01.local --user admin --source working-tree
```

A working-tree node records **dirty provenance** — its revision renders
`"<base>+dirty"` in `nodes list`, node detail, and the fleet UI, so it is
visibly not a pinned node. Because it is pinned to no fetchable commit, **fleet
rolls exclude it** with a `needs-reprovision` disposition; re-onboard it without
`--source working-tree` (pinned mode) to make it rollable again. Keep fleet-wide
rolls on pinned nodes; working-tree is a single-host development affordance.

If an op FAILS, it records a machine-branchable reason
(`ssh_setup_failed`, `pairing_failed`, `bootstrap_failed`,
`working_tree_sync_failed`, `verify_online_failed`, …). Every step is idempotent, so a FAILED op is
always safe to re-run — `onboard start` again converges. `onboard cancel
<op-id>` cancels a non-terminal op (the remote host may be partially set
up; a re-run converges).

### Mac mini onboarding

Onboarding a headless Mac mini uses the same `onboard start` flow, but
macOS imposes manual pre-steps the control plane cannot perform remotely,
and a **darwin-gate dependency** (below) that must be satisfied first.

**Darwin-gate dependency.** Bringing a node ONLINE runs `vrooli setup` on
it. On macOS this depends on the macOS-compatibility workstream that
flips the darwin gate and supplies the pnpm host tool + launchd supervisor
install (tracked as project `macos-compatibility-phase-a`). Until a given
Mac mini's Vrooli install can complete `vrooli setup`, onboarding will
fail at the `setup` step with exit code `3` (unsupported platform). If
setup is not yet viable on the target mini, run onboarding with
`--skip-setup` to pair + come ONLINE for presence, and complete setup out
of band before dispatching jobs.

**Manual pre-steps on the Mac mini (operator, one-time):**

1. **Enable Remote Login (SSH).** System Settings → General → Sharing →
   Remote Login → on, for the admin user. This is the SSH target
   `onboard start --host … --user …` connects to.
2. **Enable auto-login for the agent's user.** System Settings → Users &
   Groups → Automatically log in as `<user>`. This is **required**: the
   node-agent installs as a launchd **LaunchAgent**
   (`com.vrooli.bridge.vrooli-bridge-agent`), which runs only while its
   user is GUI-logged-in. A headless mini with no auto-login will pair
   but the agent will not auto-start after reboot — there is no
   `loginctl enable-linger` equivalent on macOS.

**Docker is not an onboarding pre-step.** Nothing in pairing or coming
ONLINE needs Docker — it matters only later, when a dispatched job runs a
container workload. Docker is a `vrooli setup` requirement (the `docker`
requirement), so setup — not onboarding — owns installing it. On macOS its
daemon (Docker Desktop) runs **per GUI session**, so a headless mini must
have Docker Desktop running in the auto-logged-in user's session before
container workloads dispatch to it; it is never needed to bring the node
ONLINE.

**Then onboard from the control-plane host:**

```bash
vrooli-bridge configure token "<owner-token>"
vrooli-bridge onboard start --host mini-01.local --user admin --name mac-mini-01 --revision "<already-pushed-sha>"
vrooli-bridge onboard watch "<op-id>"
```

Verify success: the op reaches SUCCEEDED, `nodes list` shows the node
ONLINE, and on the mini `launchctl print gui/$(id -u)/com.vrooli.bridge.vrooli-bridge-agent`
shows the agent running. Local agent recovery on the mini uses
`vrooli-bridge-agent service status|install|uninstall`; routine fleet
operations go through the control plane.

See the macOS-compat parallel workstream for the current darwin-gate
status; do not assume `vrooli setup` succeeds on a mini until that track
confirms it for the target hardware.

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
| Onboard a new node | when adding a machine to the fleet | `vrooli-bridge onboard start --host … --user …` then `onboard watch <op-id>` (see [Onboarding A Node](#onboarding-a-node-one-shot)) |
| Revoke a node | when a machine leaves the trust boundary | `vrooli-bridge nodes revoke <node>` — atomic kill of job + provisioning rights |
| Prune audit / run history | per retention policy | Trim old dispatch/provisioning audit and run records beyond the retention window (policy + tooling not yet implemented) |
| Rotate node pairing | on suspected compromise | Revoke and re-onboard the node (a fresh pairing code is issued server-side) |
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
- [`../../bootstrap/README.md`](../../bootstrap/README.md) — node bootstrap script contract (used by onboarding and manual first touch)
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md#one-shot-node-onboarding) — the durable onboarding state machine
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — fleet health, logs, metrics, and signals
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — composed scenarios and dependencies
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — trust tiers, allowlist, mutual auth, audit
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../../PRD.md`](../../PRD.md) — scope and operational targets
