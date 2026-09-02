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

## LAN endpoint and admission

Bridge reserves API port **18767**. Before an onboarding operation transfers
source, runs setup, or issues a pairing code, it executes a bounded remote
probe from the candidate over the already-established SSH channel against the
exact selected endpoint's `/health`. A failed `control_plane_unreachable`
admission therefore consumes no pairing code and is not a pairing failure.
The selected endpoint and reachability mode are saved on the durable onboarding
operation; use `vrooli-bridge onboard status <op-id>` or `onboard list` to audit
the exact URL and mode that were tested after a run completes.
The Fleet dashboard obtains the same host-level summary from the owner-only
`GET /api/v1/readiness` endpoint: it reports the fixed port, configured/tunnel/
derived endpoint source, local API health, and the latest candidate evidence.
Use `vrooli-bridge readiness status` for the authenticated operator view. Set a
durable default with `vrooli-bridge readiness configure --endpoint <url>
--reachability-mode <lan|tunnel|manual>`; an explicit `onboard start
--control-plane-url` continues to take precedence for that single attempt.

For a Linux host using UFW, the owner can inspect, verify, allow, and revoke
the **exact source IP recorded for the latest failed admission** from the Fleet
dashboard or owner CLI once setup has installed the privilege broker:

```bash
vrooli-bridge readiness firewall-inspect --candidate-ip <recorded-ip>
vrooli-bridge readiness firewall-allow --candidate-ip <recorded-ip> --confirm true
vrooli-bridge readiness firewall-verify --candidate-ip <recorded-ip>
vrooli-bridge readiness firewall-revoke --candidate-ip <recorded-ip> --confirm true
```

The API independently binds each action to that durable failed-admission
evidence; a different IP is rejected even if a client attempts to submit one.
The broker is installed by an elevated `sudo vrooli setup`, runs only as root
on a local Unix socket, and accepts only fixed UFW argv for TCP port 18767. It
never stores a sudo password, exposes a TCP service, or accepts a shell command.
If readiness says the broker is unavailable, re-run `sudo vrooli setup` from
the account that runs Bridge, then use the same UI/CLI flow—do not paste a
manual sudo rule into a scenario terminal. A hostname or `hostname.local` is only valid when the node can resolve it;
`.local` depends on mDNS, not on macOS. Prefer a DHCP reservation or managed
DNS for stable LAN addressing. Tunnel and manual URLs are also probed from the
candidate; Bridge does not silently select a tunnel to evade a blocked firewall.

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

All owner-gated verbs require an owner token. Sign the CLI in once with
`vrooli-bridge auth login --email "you@example.com"`; it prompts for the
password without echoing it and saves the returned owner session in the
per-user CLI config. For non-interactive use, pipe the password to
`vrooli-bridge auth login --email "you@example.com" --password-stdin`.
`VROOLI_BRIDGE_API_TOKEN` remains available for managed environments. In the
normal CLI flow, `vrooli-bridge auth refresh` can rotate the saved owner
session explicitly, and owner-gated commands make one transparent refresh
attempt on an expired access token.
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
 first-touch → candidate DNS/TCP/HTTP admission probe → push `bootstrap/bootstrap.sh` → issue a single-use pairing
code server-side (injected over SSH stdin, never argv/logs) → run the
script → confirm the node is ONLINE. The op is server-owned: it survives
your client disconnecting and is re-attachable by id.

After the node is ONLINE, the canonical `connect` flow performs a named
target-bound protection step. On an interactive terminal it asks for a
break-glass passphrase, seals that passphrase locally to the node's pinned
identity key, and sends only the opaque envelope to Bridge. Bridge dispatches
the typed cleanup helper and waits for its terminal result; it never sees the
plaintext. Type the explicit token `SKIP` when protection must be declined;
an empty read is never treated as a decline because an Enter key pressed while
the long onboarding wait is still running can remain buffered until this
prompt appears. Running headlessly records the protection capability as
missing rather than claiming the node is protected.
The step is idempotent: matching material already present on the node is
reported as unchanged, while foreign or incomplete material is refused.

To complete protection later for an already-succeeded onboarding operation,
use the explicit recovery command. It uses the same opaque authorization JSON
contract as the standalone cleanup protection command:

```bash
vrooli-bridge onboard complete-protection \
  --onboarding-op-id <onboarding-op-id> --machine <machine-id> \
  --node <node-id> --target mini-01.local --scope all < authorization.json
```

The `onboard status` event history shows the `break-glass-provision` step as
completed, skipped/declined, or failed, so a node that declined protection is
visible as incomplete in both CLI output and fleet readiness.

### Canonical one-command connection

Use this command for the normal operator flow:

```bash
vrooli-bridge onboard connect --host mini-01.local --user admin
```

The target must be a remote machine. Bridge rejects its own control-plane
hostname, loopback address, or local interface address so an operator working
inside an SSH session cannot accidentally onboard the machine running Bridge.
If multiple active Machines share a locator, pass `--machine-id` after the
duplicate identity has been reconciled; Bridge never guesses between them.

On first touch, Bridge preflight resolves one durable Machine and the command
opens one masked SSH-password prompt. Bridge installs its own per-Machine key,
records only key/host fingerprints and non-secret connection metadata, pairs the
node, performs a final key-only SSH check, and waits for ONLINE. The same
command after a Bridge restart reuses that Machine and key without prompting.

If the key is missing or revoked, `connect` stops before starting an operation
unless an explicit password recovery path is supplied. Ambiguous Machines and
changed host keys fail closed and require operator review. The SSH password is
never placed in argv, SQLite, operation events, logs, or evidence. Use
`--password-stdin` or `$BRIDGE_SSH_PASSWORD` for headless first touch/recovery;
the lower-level `onboard start` verb remains available for automation and uses
the same server-owned preflight resolver.

CLI path (owner-authenticated; `auth login` first):

```bash
# `start` NEVER prompts automatically. Supply the SSH password one of three ways (or the
# UI onboard form on the fleet dashboard — the equivalent browser path):
#   --password-stdin       pipe it in (e.g. from `read -s` or a secret manager)
#   --prompt-password      explicit opt-in masked TTY prompt
#   $BRIDGE_SSH_PASSWORD   ambient env var for programmatic runs
# With none of these, start proceeds only when preflight proves the Bridge key is trusted.
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
required), `--verify-timeout`, `--skip-setup`, `--skip-prereqs`. A non-empty
`--capabilities` value during onboarding enables the agent's typed control
frame handler; it does not grant authority. The registry's approved execution
scopes still decide which jobs may run, and an empty scope grant remains
presence-only. The UI path
is the **OnboardNodeForm** on the fleet dashboard (same durable op, live step
states, failure-taxonomy rendering), including the same password,
control-plane-URL, setup-profile, and source-mode inputs.

**Profile-driven setup.** The node-side `vrooli setup` is the sole
machine-provisioning authority. Setup runs as the target user. On Linux Bridge
may invoke the whole setup under verified passwordless sudo; on macOS it remains
unprivileged so Homebrew can run, and only a genuinely privileged operation may
use its explicit policy. Without available privilege, **blocked root-required
requirements are named loudly** in the `setup` step detail — a warning in the op
step events / `onboard watch` / UI, not a platform-gate failure. The setup
profile is operator-chosen and reaches the node's
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

For Darwin working-tree onboarding, omitted resource and scenario selections are
finalized as `none` so the target does not inherit enabled workloads from the
shipped checkout before typed onboarding selection is applied.

**Source mode — pinned vs working-tree.** By default the node clones the
`--revision` from the clone remote, which must be **pushed** (the control-plane
preflight hard-fails an unpushed commit). For owner **development/validation**,
`--source working-tree` ships the control plane's **local working tree** over SSH
so uncommitted work onboards without a commit or push:

```bash
vrooli-bridge onboard start --host mini-01.local --user admin --source working-tree
```

After the tree lands, the control plane detects the node's OS/architecture and
cross-builds exactly one target for `vrooli`, `vrooli-bridge`, and
`vrooli-bridge-agent`. The node receives those binaries plus `.fp` sidecars,
runs the transferred `vrooli setup`, and performs no `go build`; a node that
starts without Go can therefore reach ONLINE. Watch output includes the
`prebuilt-artifacts` step and `received prebuilt binaries` detail.

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
and an **evidence-tier dependency** (below) that must be satisfied first.

**Evidence-tier dependency.** The Darwin setup gate is open and Bridge now
uses setup's structured terminal result: only an explicit
`unsupported_platform` category maps to onboarding exit `3`. A missing tool,
privilege policy, network, or checksum failure remains a normal setup failure
(exit `1`) with its remediation preserved. macOS remains build-verified until
the real-hardware ladder is recorded; do not use `--skip-setup` as a substitute
for qualification. See the canonical [platform support matrix](../../../../docs/reference/platform-support.md).

**Manual pre-steps on the Mac mini (operator, one-time):**

1. **Enable Remote Login (SSH).** System Settings → General → Sharing →
   Remote Login → on, for the admin user. This is the SSH target
   `onboard start --host … --user …` connects to.
2. **If onboarding uses a GUI-domain LaunchAgent**, enable auto-login for
   the agent's user. SSH-only/headless onboarding installs the agent as the
   machine-wide `com.vrooli.bridge.vrooli-bridge-agent` LaunchDaemon under
   `/Library/LaunchDaemons`; that mode is independent of GUI login and does
   not require auto-login.

**Docker is not an onboarding pre-step.** Nothing in pairing or coming
ONLINE needs Docker — it matters only later, when a dispatched job runs a
container workload. Docker is a `vrooli setup` requirement (the `docker`
requirement), so setup — not onboarding — owns provider reconciliation. On
macOS the ladder first adopts a healthy local or remote Docker Engine-compatible
provider (for example OrbStack, Rancher Desktop, or Docker Desktop), and
otherwise provisions/verifies headless Colima. These are optional provider
choices, not a macOS contract. A GUI session is required only when the selected
provider requires one; it is never needed to bring the node ONLINE.

**Then onboard from the control-plane host:**

```bash
vrooli-bridge auth login --email "you@example.com"
vrooli-bridge onboard start --host mini-01.local --user admin --name mac-mini-01 --revision "<already-pushed-sha>"
vrooli-bridge onboard watch "<op-id>"
```

Verify success: the op reaches SUCCEEDED, `nodes list` shows the node
ONLINE, and on the mini `launchctl print gui/$(id -u)/com.vrooli.bridge.vrooli-bridge-agent`
shows the agent running. Local agent recovery on the mini uses
`vrooli-bridge-agent service status|install|uninstall`; routine fleet
operations go through the control plane.

See the canonical [platform support matrix](../../../../docs/reference/platform-support.md)
for the current evidence tier. Do not assume `vrooli setup` succeeds on a mini
until the target hardware has completed the recorded qualification ladder.

## Protected cleanup and enrollment-based recovery

Bridge-managed cleanup is the supported path for removing Vrooli-owned state
from a remote node. The operator terminal procedure is emergency-only: use it
only when the node is both unreachable and unmanageable, and never target the
current control-plane host.

### Protect a node during onboarding or repair

Break-glass protection can be established remotely through the Bridge agent.
The operator supplies the passphrase as JSON on stdin; the CLI seals it locally
to the node's published key and sends only the opaque envelope through Bridge.
The passphrase is not accepted from an environment variable, argv, or a file.

```bash
read -r -s -p 'Break-glass passphrase: ' BRIDGE_BREAK_GLASS_PASSPHRASE
printf '\n'
jq -n --arg passphrase "$BRIDGE_BREAK_GLASS_PASSPHRASE" '{passphrase:$passphrase}' |
vrooli-bridge cleanup provision-break-glass \
  --machine "<machine-id>" \
  --node "<node-id>" \
  --target "<target-hostname>" \
  --scope "vrooli:uninstall"
unset BRIDGE_BREAK_GLASS_PASSPHRASE
```

Prefer a secret-manager pipe over a shell variable for automation. Never place
the passphrase in a persistent file, argv, or exported environment variable;
unset the temporary shell variable immediately. Matching
protection requests are idempotent. A request with a different target,
operator, scope, or operation identity is refused rather than replacing
existing material. The command prints a capability report covering agent
presence, runtime, provisioning, SSH management and approval, cleanup
planning, cleanup application, and target-bound break-glass. The next command
shown by the report is `vrooli-bridge cleanup get <operation-id>`.

### Protected cleanup lifecycle

The cleanup flow is deliberately staged:

1. `vrooli-bridge cleanup start` collects a read-only inventory and freezes a
   plan.
2. `vrooli-bridge cleanup get <operation-id>` displays the exact `Remove`,
   `Keep`, and `Cannot attribute` sections and the plan hash.
3. `vrooli-bridge cleanup confirm <operation-id>` requires the exact target,
   plan hash, and sealed operator authorization.
4. `vrooli-bridge cleanup get <operation-id>` returns the durable receipt.
5. `vrooli-bridge cleanup verify <operation-id>` performs the post-apply
   verification.

An apply resumes the same frozen plan and hash; it does not rediscover or
expand the removal set. The privileged node helper accepts named typed
operations only. It is not an interactive shell and does not receive a raw
removal command. If the paired agent is unavailable, Bridge may use verified
and policy-approved SSH management as the transport for that same typed
helper. Without either path it returns a typed blocked result naming the
missing capability and performs no cleanup.

Break-glass envelopes use the VCS1 X25519/AES-GCM format and are bound to the
machine, node, target, scope, plan hash, operation, and operator. Locally
minted operator sessions have a maximum lifetime of 15 minutes; revocation is
therefore bounded by that lifetime. Enrollment requires the authenticator,
but an enrolled local session can be verified and renewed without contacting
it on every request. Capability verification tolerates at most two minutes of
clock skew at either endpoint; outside that window it refuses with a named
clock-skew reason. Operators must correct node time before retrying.

No destructive action may target the current host. Before a live operation,
confirm the machine id, node id, target, and pre-removal inventory in the
durable operation record.

## Machine identity repair and merge

Machine enrollment is idempotent for active locator evidence. Re-running an
enrollment against the same host converges on the existing Machine UUID and
creates a new immutable `EnrollmentAttempt`; it does not create a replacement
Machine. Resolution order is explicit Machine ID, current Node ID, SSH
host-key fingerprint, then normalized hostname. A conflicting match is
reported for operator action rather than guessed.

To repair a host in place, reuse the durable identity and let Bridge drive the
normal SSH/key/bootstrap flow:

```bash
vrooli-bridge machines repair <machine-id>
vrooli-bridge onboard watch <onboarding-op-id>
vrooli-bridge machines get <machine-id>
```

Repair is safe to retry. Each invocation creates a distinct attempt linked to
the same Machine; terminal attempts are never reopened or overwritten.

When historical duplicates exist, choose the surviving target explicitly and
merge the source. The source is archived, locators and Node lineage are folded
into the target, attempts remain attached to the target, and one audit record is
written:

```bash
vrooli-bridge machines merge <duplicate-machine-id> <surviving-machine-id>
vrooli-bridge machines get <surviving-machine-id>
```

The startup migration reconciles legacy rows before installing the global
partial unique index on current `node_id`; the newest lineage wins and each
superseded row gets a migration audit event. Do not add a manual duplicate
Machine to bypass this invariant.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Control plane does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). |
| Node shows offline / dial-out channel dropped | Node presence in `nodes list`; on the node, agent service status (`systemctl`/`launchctl`/Service mgr) and its logs; control-plane reachability (LAN direct, or tunnel-manager URL off-LAN) | Restart the agent service on the node; confirm outbound path to the control plane; if off-LAN, verify tunnel-manager tunnel is up | If a node repeatedly drops, suspect tunnel or network; check tunnel-manager and `../internal/PROBLEMS.md`. |
| Provision fails (sync to revision R) | Provisioning audit entry, node toolchain/disk headroom, `vrooli setup` output captured back at the control plane | Re-run provisioning (idempotent); the tier auto-rolls the node back to its prior revision on setup failure | If setup fails repeatedly on one OS, capture logs and escalate; review `../internal/SECURITY.md` for the privileged tier's expectations. |
| Job stuck / not completing | Job status in the control plane; the durable run is server-owned and re-attachable by id — block once with the wait verb, do **not** poll | Re-attach by run id; if genuinely wedged, abort the job (abort ≠ cancel) and inspect node-side run logs | Mirror test-genie discipline: one job per node at a time; a thrashing node points to scheduling/health issues. |
| Version drift across fleet | Per-node Vrooli revision in `nodes list`; protocol-compatibility flags | Pin the fleet to target revision R and re-provision drifted nodes; nodes on an incompatible agent are flagged "needs update" and excluded from incompatible work | If drift recurs, evaluate self-healing re-provisioning (OT-P2-004, not yet implemented). |
| Job rejected by node | Node's transport and `<namespace>:<effect>` catalog grants vs the dispatched `{scenario, verb}` | Expected behavior — Bridge runs only manifest-derived typed verbs; grant the named missing catalog scope deliberately if the verb should be permitted | Never work around the allowlist with raw shell; see `../internal/SECURITY.md`. |

## Backup / Restore

The control-plane store is SQLite via `api-core/storage` and is the
durable record of the fleet: nodes, pairings, capability snapshots, jobs,
dispatch/provisioning audit, and version history. Losing it loses fleet
identity and the audit trail, so it is the primary backup target.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| Control-plane SQLite store | Stop the control plane (or quiesce writes) and copy the scenario's SQLite file; retain per a configurable policy | Restore the file to the scenario's data directory and restart the control plane | Procedure defined; automated backup wiring not yet implemented. |
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
