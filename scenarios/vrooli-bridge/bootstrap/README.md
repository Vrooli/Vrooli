# vrooli-bridge node bootstrap

`bootstrap.sh` takes a fresh node from a raw OS to a **paired, ONLINE,
auto-starting** fleet agent in one idempotent run. It is the core node-local
artifact every other onboarding surface (the phase-5 `onboard` orchestrator, the
CLI `onboard` verbs, the UI form) drives.

The one-shot working-tree path does **not** build on the node. The control plane
detects the node's OS/architecture, cross-builds `vrooli`, `vrooli-bridge`, and
`vrooli-bridge-agent` from the exact live tree it ships (including uncommitted
and untracked non-ignored files), and transfers all three binaries plus their
`.fp` sidecars. A raw node therefore reaches ONLINE with no preinstalled Go and
without pulling a release artifact or GitHub clone. One run:

1. **detect-os** — identify platform (`linux`/`darwin`; Windows uses the
   PowerShell installer).
2. **prebuilt-artifacts** — verify the three transferred executables and their
   shared source fingerprint.
3. **prereqs** — skipped for a pre-synced tree with a complete prebuilt bundle;
   the pinned/manual fallback still ensures its clone prerequisites.
4. **clone** — verify the pre-synced live tree, or clone/converge a pinned
   revision in the manual fallback.
5. **setup** — run the transferred `vrooli setup` directly with
   `VROOLI_SOURCE_ROOT` set to the shipped tree. The matching sidecar keeps it
   fresh before setup installs Go. The sentinel remains keyed by revision,
   source digest, and setup profile.
6. **toolchain** — skipped for the complete prebuilt path because no node-side
   compile follows; retained for the manual source-build fallback, including
   off-PATH recovery.
7. **build-agent / build-cli** — report the received prebuilt binaries and skip
   compilation. The manual fallback can still build both from source.
8. **node-key** — load-or-generate the node's Ed25519 keypair; print its public
   key.
9. **pair-redeem** — redeem the single-use pairing code against the control
   plane. This **pins the control-plane key** (`control_plane.pub`, `0600`)
   **before** the code is burned, so the agent can verify every server push
   (`SECURITY.md` boundary 2).
10. **pin-verify** — assert the pinned key is present.
11. **service-install** — install + start the platform-native background service
    (systemd `--user` unit on Linux; launchd on macOS).
12. **autostart** — enable headless auto-start (`loginctl enable-linger` on
    Linux; launchd `KeepAlive` + auto-login on macOS).
13. **verify-online** — wait (bounded) for the agent to report a live dial-out
    channel.

Every step checks current state before acting, so a re-run after a partial
failure converges and a re-run of a fully-onboarded node changes nothing.

## Usage

```sh
# First onboarding — the pairing code is read ONLY from the environment.
BRIDGE_PAIRING_CODE="$(vrooli-bridge pair issue --name web-01 --json | jq -r .code)" \
  ./bootstrap.sh \
    --control-plane-url https://cp.example.com \
    --node-name web-01 \
    --revision <git-sha>

# Re-run to converge / repair — no code needed once paired.
./bootstrap.sh --control-plane-url https://cp.example.com --node-name web-01 --revision <git-sha>
```

### Options

| Flag | Env | Default | Meaning |
|------|-----|---------|---------|
| `--control-plane-url` | `BRIDGE_CONTROL_PLANE_URL` | *(required)* | Control-plane base URL. |
| *(none — env only)* | `BRIDGE_PAIRING_CODE` | *(required until paired)* | Single-use pairing code. **Env only, never a flag** (see Security). |
| `--node-name` | `BRIDGE_NODE_NAME` | hostname | Fleet label. |
| `--repo-url` | `BRIDGE_REPO_URL` | `https://github.com/Vrooli/Vrooli.git` | Source to clone. |
| `--revision` | `BRIDGE_REVISION` | repo default branch | Git commit/branch/tag to pin. |
| `--checkout-dir` | `BRIDGE_CHECKOUT_DIR` | `$HOME/vrooli` | Node checkout location. |
| `--source-dir` | `BRIDGE_SOURCE_DIR` | *(none — pinned)* | Verify a **pre-synced working tree** here instead of cloning (working-tree source mode). |
| `--source-digest` | `BRIDGE_SOURCE_DIGEST` | *(none)* | Content digest of the pre-synced tree; keys the setup sentinel so a re-shipped tree re-runs setup. |
| `--state-dir` | `BRIDGE_AGENT_STATE_DIR` | `$XDG_STATE_HOME/vrooli-bridge-agent` | Credential/pin/state dir. |
| `--work-dir` | `BRIDGE_WORK_DIR` | checkout dir | Dir the agent runs jobs in. |
| `--service-user` | `BRIDGE_SERVICE_USER` | current user | OS principal the service runs as. |
| `--provision-service-user` | `BRIDGE_PROVISION_SERVICE_USER` | unset | Separate OS principal for the privileged provisioning helper. When set, the bootstrap requires non-interactive elevation and installs a machine-wide `vrooli-bridge-provisioner` unit. |
| `--provision-socket` | `BRIDGE_PROVISION_SOCKET` | `/run/vrooli-bridge/provision.sock` | Absolute local IPC socket shared by the runner and provisioner. Linux peer credentials authorize the runner UID. |
| `--capabilities` | `BRIDGE_CAPABILITIES` | *(none)* | Comma-separated verb namespaces to self-report. When supplied by Bridge onboarding, it also opts the agent into typed control frames; the registry's approved execution scopes remain the authorization source. |
| `--verify-timeout` | `BRIDGE_VERIFY_TIMEOUT` | `120` | Dial-out verification budget (seconds). |
| `--setup-environment` | `BRIDGE_SETUP_ENVIRONMENT` | *(node default)* | Node-side `vrooli setup --environment`: `development` \| `production` \| `minimal`. |
| `--setup-resources` | `BRIDGE_SETUP_RESOURCES` | *(node default)* | Node-side `vrooli setup --resources`: `enabled` \| `none` \| `<comma-list>`. |
| `--setup-scenarios` | `BRIDGE_SETUP_SCENARIOS` | *(node default)* | Node-side `vrooli setup --scenarios`: `none` \| `all` \| `<comma-list>`. |
| `--include-optional` | *(none — flag)* | off | Also apply optional (non-required) host safeguards. |

### Elevated, profile-driven setup

The **setup** step treats project-level `vrooli setup` as the **sole
machine-provisioning authority** on the node:

- **Elevation.** When passwordless sudo is available (already root, or `sudo -n`
  succeeds — the onboarding orchestrator provisions this at first touch, see the
  `--provision-sudo` handover), setup runs **under sudo** so root-required
  requirements are actually applied rather than skipped. When it is not, setup
  runs unprivileged and the **skipped** root-required requirements are named
  **loudly** in the step detail (`… N root-required requirement(s) skipped for
  privilege: …`) — a warning surfaced in the op step events / CLI watch / UI, not
  a failure. `prereqs` and `autostart` likewise use non-interactive `sudo -n`
  only, so a headless run degrades loudly instead of hanging on a password
  prompt. macOS setup stays under the SSH user because Homebrew refuses root;
  individual requirements use the provisioned sudo capability when needed.
- **Profile.** The setup-profile flags above are threaded into
  transferred `vrooli setup` invocation (the manual fallback retains the root
  Makefile pass-through). Every value is metacharacter-validated at the
  API boundary (`api/internal/onboard.validateSetupProfile`, mirroring the cprev
  revision filter) **and** re-checked here before it is spliced in, so no
  shell-injectable value can ever reach the command. **Empty values are the
  default fleet posture** — they fall through to `vrooli setup`'s own defaults
  (development environment, default resources), i.e. the bootstrap does not
  reshape a node's setup unless the operator explicitly asks.
- **Sentinel.** The per-revision setup sentinel is keyed on the **profile** as
  well, so changing any profile value re-runs setup while an identical profile at
  the same revision stays a no-op.

### Source mode — pinned vs working-tree

The **clone** step acquires the source in either mode:

- **Pinned revision (default).** The node clones/fetches `--revision` from the
  clone remote. The revision must be **pushed** — the control-plane preflight
  hard-fails an unpushed commit, because a node can only ever fetch pushed
  history. This is the fleet-safe path and the only one fleet rolls use.
- **Working-tree (owner development/validation only).** The control plane ships
  its **local working tree** (tracked + modified + untracked-non-ignored files)
  to the node over the established SSH channel, then passes `--source-dir` (where
  the tree landed) and `--source-digest` (its content digest). The **clone** step
  verifies that pre-synced tree instead of cloning, and records **dirty
  provenance** — the base commit plus the digest, rendered as `"<base>+dirty"`.
  It then cross-builds only the detected node target from that same local root,
  transfers the binaries/sidecars, and passes their paths to bootstrap. Use it
  when uncommitted work must onboard without a commit or push, or when the node
  must start without Go.

A working-tree node is pinned to no fetchable commit, so fleet rolls **exclude**
it with a `needs-reprovision` disposition; re-onboard it in pinned mode (or
provision it to a pushed revision) to make it rollable again. The setup sentinel
folds in `--source-digest`, so re-shipping changed work re-runs node-side setup
while an identical tree stays a no-op.

### Prerequisites & toolchains

With a pre-synced working tree and complete prebuilt bundle, bootstrap needs no
clone or build tools: **prereqs** and the build-toolchain guard skip. The
transferred Vrooli CLI runs setup, which installs git and Go as bootstrap host
tools and owns every other selected requirement. Docker remains a lazy demand
of enabled container-backed resources.

The manual pinned fallback still installs only the true clone prerequisites,
`git` and `curl`, and retains the off-PATH `go`/`pnpm` guard before its legacy
source builds. Bootstrap deliberately does not create a second tool installer.

- **Linux.** Missing `git`/`curl` are installed via non-interactive `apt-get`
  (package name == command name), gated on passwordless sudo like every other
  privileged action — a headless run degrades loudly rather than hanging on a
  password prompt.
- **macOS manual fallback.** macOS always ships `curl`, and `git` arrives with
  the **Xcode Command Line Tools**. When the pinned/manual clone path lacks git,
  bootstrap directs the operator to **`xcode-select --install`**. In the
  prebuilt working-tree path no clone tool is needed; `vrooli setup` remains the
  owner and bootstraps Homebrew when absent before installing git/Go.

In the manual fallback, the **toolchain** step is the guard that makes the
"setup owns the toolchains" split safe. After setup runs it confirms `go` and
`pnpm` actually resolve before source-build steps rely on them, and it closes a
real headless-onboarding gap: a
**non-interactive SSH shell does not source the login profile**, so a toolchain
`setup` just installed can be present on disk yet invisible on `PATH` for the
same bootstrap process. The guard therefore probes the fixed locations setup
installs into — `$HOME/.vrooli/bin` (no-sudo standalone tools such as `pnpm`),
`$HOME/.local/bin` (setup's symlink target for go-installed binaries), plus the
conventional system toolchain directories (`toolchain_dirs` in `bootstrap.sh`
is the authoritative, ordered list) — and:

- **found off-PATH** → prepends that directory to `PATH` (exported for the rest
  of the run so the build steps see it) and notes the recovery in the step
  detail;
- **found nowhere** → fails with an actionable detail naming the missing tool(s)
  **and** pointing at the `vrooli setup` requirements that should have delivered
  them (run `vrooli setup status` on the node to see why — often a group skipped
  for privilege), instead of letting a build die with a confusing
  `go: command not found`.

### Pre-satisfied-prerequisite shortcuts

For nodes where parts of the environment are already provisioned (or for the
phase-5 orchestrator, which may stage the checkout itself):

| Flag | Effect |
|------|--------|
| `--skip-prereqs` | Assume `git`/`curl` (the clone prerequisites) are already installed. |
| `--skip-setup` | Skip `vrooli setup`. Pairing, online, and auto-start still work, but the node **cannot run jobs** until setup is run later. |
| `--force-setup` | Run `vrooli setup` even if its per-revision sentinel exists. |
| `--vrooli-bin PATH` | Use a transferred prebuilt Vrooli CLI for setup; `PATH.fp` is required. |
| `--agent-bin PATH` | Use a prebuilt node-agent instead of building one. |
| `--bridge-cli PATH` | Use a prebuilt `vrooli-bridge` CLI instead of building one. |

## Security — the pairing code never touches argv

The pairing code is a single-use fleet-join secret. It is read **only** from
`$BRIDGE_PAIRING_CODE`, never a command-line flag, so it cannot leak to other
local users via `ps`. The script never echoes it. The `vrooli-bridge pair
redeem` command likewise reads `$BRIDGE_PAIRING_CODE` (with `--code` kept as an
explicit interactive override), so the code stays out of the redeem process's
argv too.

## Idempotency & recovery

- **clone** converges an existing checkout instead of failing.
- **setup** is skipped when a `setup-<revision>.done` sentinel exists.
- **pair-redeem** is skipped once `control_plane.pub` + `node_id` are present —
  a second code is never spent. Because the key is pinned *before* the code is
  burned, a redeem that cannot reach the server leaves **nothing** spent.
- **service-install** is skipped when the service is already installed, running,
  and its unit file byte-matches what would be rendered now — so a converged
  re-run does **not** restart the agent.
- **autostart** is skipped when linger is already enabled.
- A rejected/expired/already-used code exits **4** with guidance to reissue,
  never a half-installed wedge.

## macOS

macOS onboarding has **unavoidable manual pre-steps** the script cannot
perform for you:

1. **Remote Login (SSH)** enabled (System Settings → General → Sharing), and
   **auto-login** enabled if the node is headless — a launchd gui-domain agent
   only survives logout when the console user is auto-logged-in (the macOS
   analogue of systemd linger).

Docker is **not** an onboarding pre-step: nothing in pairing or coming ONLINE
needs it. It is a `vrooli setup` requirement pulled in only for container
workloads, and on macOS its daemon (Docker Desktop) runs per GUI session — so a
headless mini needs Docker Desktop running in the auto-logged-in session before
such workloads dispatch, not to onboard.

**Darwin degradation contract:** if this host's `vrooli` build still refuses
setup on darwin, the script fails fast at the **setup** step with exit
**3** and an actionable message pointing at the macOS-compatibility /
`workspace-sandbox-cross-platform` workstream — it never half-installs. The
refusal is detected dynamically from the setup output, so once the darwin gate
opens the same script converges with no change.

## Machine contract (for the phase-5 orchestrator)

**STDOUT carries progress markers only**; all human/diagnostic logging goes to
**STDERR**. STDOUT is therefore a clean stream the orchestrator can parse and
persist step-by-step.

### Marker grammar

One marker per line:

```
VBOOTSTRAP v=1 event=<event> [step=<step-id>] [detail="<single-line text>"]
```

- `event` ∈ `run-start | run-ok | run-fail | step-start | step-ok | step-skip | step-fail`.
- Fields are space-separated `key=value`. `detail` is always **last**,
  double-quoted, and newline-free.
- A run emits exactly one `run-start` first and exactly one `run-ok` **or**
  `run-fail` last.
- Each step emits `step-start` then exactly one of `step-ok | step-skip |
  step-fail`.
- Stable `step` ids, in order: `detect-os`, `prereqs`, `clone`, `setup`,
  `toolchain`, `build-agent`, `build-cli`, `node-key`, `pair-redeem`,
  `pin-verify`, `service-install`, `autostart`, `verify-online`.

Parse rule: consume lines beginning with `VBOOTSTRAP `; ignore everything else
on stdout (there is nothing else). The node id appears in the `detail` of
`pair-redeem`/`pin-verify`/`run-ok` and can be captured from there if needed.

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success — node paired, online, auto-start enabled. |
| `2` | Usage/config error (missing `--control-plane-url`, or unpaired with no `$BRIDGE_PAIRING_CODE`). |
| `3` | Unsupported platform, incl. a darwin host whose `vrooli` still refuses setup. |
| `4` | Pairing failed (code invalid/expired/already used) — reissue and re-run. Nothing was spent. |
| `1` | Any other failure. |

## Tests

Step-function tests stub every external command (`git`, `make`, `loginctl`,
`journalctl`) and both produced binaries (node-agent, CLI), then run the script
twice to assert the first run does the work and the second converges to a no-op:

```sh
./bootstrap_test.sh      # exit 0 = all assertions passed
```

`shellcheck bootstrap.sh bootstrap_test.sh` must be clean.
