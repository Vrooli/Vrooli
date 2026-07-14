# vrooli-bridge node bootstrap

`bootstrap.sh` takes a fresh node from a raw OS to a **paired, ONLINE,
auto-starting** fleet agent in one idempotent run. It is the core node-local
artifact every other onboarding surface (the phase-5 `onboard` orchestrator, the
CLI `onboard` verbs, the UI form) drives.

The agent is **built from cloned source** on the node — there is no binary
distribution or code-signing to manage. One run:

1. **detect-os** — identify platform (`linux`/`darwin`; Windows uses the
   PowerShell installer).
2. **prereqs** — ensure `git`, `curl`, `go`, `pnpm` (installs the missing ones;
   `apt-get` on Linux, Homebrew on macOS).
3. **clone** — clone the repo at a pinned revision, or converge an existing
   checkout (`git fetch` + checkout).
4. **setup** — `make setup` (idempotent; sentinel-guarded per revision).
5. **build-agent** — `make -C scenarios/vrooli-bridge/agent build`.
6. **build-cli** — build the `vrooli-bridge` CLI from the checkout (needed to
   redeem).
7. **node-key** — load-or-generate the node's Ed25519 keypair; print its public
   key.
8. **pair-redeem** — redeem the single-use pairing code against the control
   plane. This **pins the control-plane key** (`control_plane.pub`, `0600`)
   **before** the code is burned, so the agent can verify every server push
   (`SECURITY.md` boundary 2).
9. **pin-verify** — assert the pinned key is present.
10. **service-install** — install + start the platform-native background service
    (systemd `--user` unit on Linux; launchd on macOS).
11. **autostart** — enable headless auto-start (`loginctl enable-linger` on
    Linux; launchd `KeepAlive` + auto-login on macOS).
12. **verify-online** — wait (bounded) for the agent to report a live dial-out
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
| `--state-dir` | `BRIDGE_AGENT_STATE_DIR` | `$XDG_STATE_HOME/vrooli-bridge-agent` | Credential/pin/state dir. |
| `--work-dir` | `BRIDGE_WORK_DIR` | checkout dir | Dir the agent runs jobs in. |
| `--service-user` | `BRIDGE_SERVICE_USER` | current user | OS principal the service runs as. |
| `--capabilities` | `BRIDGE_CAPABILITIES` | *(none)* | Comma-separated verb namespaces to self-report. |
| `--verify-timeout` | `BRIDGE_VERIFY_TIMEOUT` | `120` | Dial-out verification budget (seconds). |

### Pre-satisfied-prerequisite shortcuts

For nodes where parts of the environment are already provisioned (or for the
phase-5 orchestrator, which may stage the checkout itself):

| Flag | Effect |
|------|--------|
| `--skip-prereqs` | Assume `git`/`go`/`pnpm`/`curl` are already installed. |
| `--skip-setup` | Skip `make setup`. Pairing, online, and auto-start still work, but the node **cannot run jobs** until `make setup` is run later. |
| `--force-setup` | Run `make setup` even if its per-revision sentinel exists. |
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

macOS onboarding has two **unavoidable manual pre-steps** the script cannot
perform for you:

1. **Docker Desktop** installed and running (the container runtime the agent
   reports readiness against).
2. **Remote Login (SSH)** enabled (System Settings → General → Sharing), and
   **auto-login** enabled if the node is headless — a launchd gui-domain agent
   only survives logout when the console user is auto-logged-in (the macOS
   analogue of systemd linger).

**Darwin degradation contract:** if this host's `vrooli` build still refuses
`make setup` on darwin, the script fails fast at the **setup** step with exit
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
  `build-agent`, `build-cli`, `node-key`, `pair-redeem`, `pin-verify`,
  `service-install`, `autostart`, `verify-online`.

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
