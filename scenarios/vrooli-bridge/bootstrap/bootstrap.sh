#!/usr/bin/env bash
#
# vrooli-bridge node bootstrap — take a fresh node from raw OS to a paired,
# ONLINE, auto-starting fleet agent in one idempotent run.
#
# It clones the Vrooli source at a pinned revision, sets the node up, builds the
# node-agent (and the bridge CLI it needs to redeem) from that source, generates
# the node keypair, redeems a single-use pairing code (pinning the control-plane
# key BEFORE the code is burned), installs the platform-native background service,
# enables headless auto-start, and verifies the agent's dial-out channel is live.
#
# Every step checks current state before acting, so a re-run after a partial
# failure converges and a fully-onboarded node re-run changes nothing.
#
# CONTRACT (parsed by the phase-5 orchestrator):
#   * Progress markers are emitted on STDOUT only; all human/diagnostic logging
#     goes to STDERR. STDOUT is therefore a clean machine stream.
#   * Marker grammar (one per line):
#       VBOOTSTRAP v=1 event=<event> [step=<step-id>] [detail="<single-line text>"]
#     events: run-start | run-ok | run-fail | step-start | step-ok | step-skip | step-fail
#     A run emits exactly one run-start first and one run-ok|run-fail last.
#     Each step emits step-start then exactly one of step-ok|step-skip|step-fail.
#     `detail` is always the last field, double-quoted, newline-free.
#   * Exit codes: 0 success · 2 usage/config error · 3 unsupported platform
#     (incl. a darwin host whose vrooli still refuses setup) · 4 pairing failed,
#     reissue the code · 1 any other failure.
#   * The pairing code is read ONLY from $BRIDGE_PAIRING_CODE (never argv), so it
#     cannot leak through `ps`, and it is never echoed.
#
# See ./README.md for the full operator + orchestrator contract.

set -euo pipefail

# --- constants ---------------------------------------------------------------

readonly MARKER_VERSION=1
readonly UNIT_NAME="vrooli-bridge-agent.service"
readonly PIN_FILE="control_plane.pub"
readonly DEFAULT_REPO_URL="https://github.com/Vrooli/Vrooli.git"
# The exact line the agent logs when the control plane accepts its dial-out
# stream (HTTP 200). This is the node-local "connected/ONLINE" signal.
readonly CONNECTED_MARKER="dial-out stream open"

# --- output helpers ----------------------------------------------------------
# Human logs -> stderr. Markers -> stdout.

log() { printf '%s\n' "$*" >&2; }

# marker <event> [step] [detail]
marker() {
  local event="$1" step="${2:-}" detail="${3:-}"
  local line="VBOOTSTRAP v=${MARKER_VERSION} event=${event}"
  [ -n "$step" ] && line="${line} step=${step}"
  [ -n "$detail" ] && line="${line} detail=\"${detail}\""
  printf '%s\n' "$line"
}

# Current step id, tracked so an unexpected error (set -e trap) can attribute a
# step-fail + run-fail to the right place before exiting.
CURRENT_STEP=""

step_start() { CURRENT_STEP="$1"; marker step-start "$1" "${2:-}"; log "==> ${1}: ${2:-}"; }
step_ok()    { marker step-ok "$CURRENT_STEP" "${1:-}"; log "    ok: ${CURRENT_STEP} ${1:-}"; CURRENT_STEP=""; }
step_skip()  { marker step-skip "$CURRENT_STEP" "${1:-}"; log "    skip: ${CURRENT_STEP} ${1:-}"; CURRENT_STEP=""; }

# fail <exit-code> <detail> — emit step-fail (if a step is open) + run-fail, exit.
fail() {
  local code="$1" detail="$2"
  if [ -n "$CURRENT_STEP" ]; then
    marker step-fail "$CURRENT_STEP" "$detail"
    log "    FAIL: ${CURRENT_STEP}: ${detail}"
  fi
  marker run-fail "" "$detail"
  exit "$code"
}

# set -e safety net: any uncaught non-zero command lands here.
on_err() {
  local code=$?
  # Avoid double-reporting when fail() already ran (it exits directly).
  if [ -n "$CURRENT_STEP" ]; then
    marker step-fail "$CURRENT_STEP" "unexpected error (exit ${code})"
    marker run-fail "" "unexpected error in step ${CURRENT_STEP} (exit ${code})"
    log "    FAIL: ${CURRENT_STEP}: unexpected error (exit ${code})"
  fi
}
trap on_err ERR

# --- configuration -----------------------------------------------------------

CONTROL_PLANE_URL="${BRIDGE_CONTROL_PLANE_URL:-}"
NODE_NAME="${BRIDGE_NODE_NAME:-}"
REPO_URL="${BRIDGE_REPO_URL:-$DEFAULT_REPO_URL}"
REVISION="${BRIDGE_REVISION:-}"
CHECKOUT_DIR="${BRIDGE_CHECKOUT_DIR:-$HOME/vrooli}"
STATE_DIR="${BRIDGE_AGENT_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/vrooli-bridge-agent}"
WORK_DIR="${BRIDGE_WORK_DIR:-}"
SERVICE_USER="${BRIDGE_SERVICE_USER:-}"
CAPABILITIES="${BRIDGE_CAPABILITIES:-}"
VERIFY_TIMEOUT="${BRIDGE_VERIFY_TIMEOUT:-120}"

SKIP_PREREQS=0
SKIP_SETUP=0
FORCE_SETUP=0
AGENT_BIN_OVERRIDE=""
BRIDGE_CLI_OVERRIDE=""

usage() {
  cat >&2 <<'USAGE'
Usage: bootstrap.sh --control-plane-url URL [options]
       BRIDGE_PAIRING_CODE=<code> bootstrap.sh --control-plane-url URL ...

Required:
  --control-plane-url URL   Control-plane base URL to pair and dial out to.
                            (env BRIDGE_CONTROL_PLANE_URL)
  $BRIDGE_PAIRING_CODE      Single-use pairing code (env ONLY, never a flag, so
                            it cannot leak via ps). Not needed once paired.

Options (flag overrides env in parentheses):
  --node-name NAME          Fleet label for this node.        (BRIDGE_NODE_NAME; default: hostname)
  --repo-url URL            Source repo to clone.             (BRIDGE_REPO_URL; default: Vrooli GitHub)
  --revision REV            Git commit/branch/tag to pin.     (BRIDGE_REVISION; default: repo default branch)
  --checkout-dir DIR        Where the node's checkout lives.  (BRIDGE_CHECKOUT_DIR; default: $HOME/vrooli)
  --state-dir DIR           Agent credential/state dir.       (BRIDGE_AGENT_STATE_DIR; default: XDG state)
  --work-dir DIR            Dir the agent runs jobs in.       (BRIDGE_WORK_DIR; default: checkout dir)
  --service-user USER       OS principal the service runs as. (BRIDGE_SERVICE_USER; default: current user)
  --capabilities LIST       Comma-separated verb namespaces.  (BRIDGE_CAPABILITIES)
  --verify-timeout SECONDS  Dial-out verification budget.     (BRIDGE_VERIFY_TIMEOUT; default: 120)

Pre-satisfied-prerequisite shortcuts (each documented in README.md):
  --skip-prereqs            Assume git/go/pnpm/curl are already installed.
  --skip-setup              Skip `make setup` (node cannot run jobs until it is
                            run later, but pairing/online/auto-start still work).
  --force-setup             Run `make setup` even if its revision sentinel exists.
  --agent-bin PATH          Use a prebuilt node-agent instead of building it.
  --bridge-cli PATH         Use a prebuilt vrooli-bridge CLI instead of building it.

  -h, --help                Show this help.
USAGE
}

while [ $# -gt 0 ]; do
  case "$1" in
    --control-plane-url) CONTROL_PLANE_URL="$2"; shift 2 ;;
    --node-name)         NODE_NAME="$2"; shift 2 ;;
    --repo-url)          REPO_URL="$2"; shift 2 ;;
    --revision)          REVISION="$2"; shift 2 ;;
    --checkout-dir)      CHECKOUT_DIR="$2"; shift 2 ;;
    --state-dir)         STATE_DIR="$2"; shift 2 ;;
    --work-dir)          WORK_DIR="$2"; shift 2 ;;
    --service-user)      SERVICE_USER="$2"; shift 2 ;;
    --capabilities)      CAPABILITIES="$2"; shift 2 ;;
    --verify-timeout)    VERIFY_TIMEOUT="$2"; shift 2 ;;
    --skip-prereqs)      SKIP_PREREQS=1; shift ;;
    --skip-setup)        SKIP_SETUP=1; shift ;;
    --force-setup)       FORCE_SETUP=1; shift ;;
    --agent-bin)         AGENT_BIN_OVERRIDE="$2"; shift 2 ;;
    --bridge-cli)        BRIDGE_CLI_OVERRIDE="$2"; shift 2 ;;
    -h|--help)           usage; exit 0 ;;
    *) log "unknown argument: $1"; usage; exit 2 ;;
  esac
done

# --- derived paths & validation ----------------------------------------------

[ -n "$CONTROL_PLANE_URL" ] || { log "error: --control-plane-url (or \$BRIDGE_CONTROL_PLANE_URL) is required"; usage; exit 2; }
[ -n "$NODE_NAME" ] || NODE_NAME="$(hostname)"
[ -n "$WORK_DIR" ] || WORK_DIR="$CHECKOUT_DIR"

readonly BOOTSTRAP_STATE_DIR="${STATE_DIR}/.bootstrap"
readonly NODE_ID_FILE="${STATE_DIR}/node_id"
AGENT_BIN="$AGENT_BIN_OVERRIDE"
BRIDGE_CLI="$BRIDGE_CLI_OVERRIDE"
NODE_ID=""
NODE_PUBLIC_KEY=""

# Common agent config argv shared by --print-public-key / --print-service-unit /
# service verbs, built once NODE_ID is known.
agent_service_args() {
  local args=(--control-plane-url "$CONTROL_PLANE_URL" --node-id "$NODE_ID" --state-dir "$STATE_DIR")
  [ -n "$WORK_DIR" ] && args+=(--work-dir "$WORK_DIR")
  [ -n "$SERVICE_USER" ] && args+=(--service-user "$SERVICE_USER")
  printf '%s\n' "${args[@]}"
}

# --- steps -------------------------------------------------------------------

OS=""
ARCH=""

step_detect_os() {
  step_start detect-os "identify platform"
  local uname_s uname_m
  uname_s="$(uname -s)"
  uname_m="$(uname -m)"
  case "$uname_s" in
    Linux)  OS="linux" ;;
    Darwin) OS="darwin" ;;
    *) fail 3 "unsupported OS ${uname_s}: this bootstrap supports linux and darwin (windows nodes use the PowerShell installer)" ;;
  esac
  case "$uname_m" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) ARCH="$uname_m" ;;
  esac
  if [ "$OS" = "darwin" ]; then
    log "    note: macOS nodes require Docker Desktop running and Remote Login enabled (manual pre-steps; see README)."
  fi
  step_ok "os=${OS} arch=${ARCH}"
}

# ensure_cmd <name> — is a command available?
have() { command -v "$1" >/dev/null 2>&1; }

step_prereqs() {
  step_start prereqs "ensure git/go/pnpm/curl"
  if [ "$SKIP_PREREQS" -eq 1 ]; then
    step_skip "--skip-prereqs"
    return
  fi
  local missing=()
  local c
  for c in git curl go pnpm; do
    have "$c" || missing+=("$c")
  done
  if [ "${#missing[@]}" -eq 0 ]; then
    step_skip "all prerequisites present"
    return
  fi
  log "    installing missing prerequisites: ${missing[*]}"
  case "$OS" in
    linux)  install_prereqs_linux "${missing[@]}" ;;
    darwin) install_prereqs_darwin "${missing[@]}" ;;
  esac
  for c in "${missing[@]}"; do
    have "$c" || fail 1 "prerequisite ${c} still missing after install attempt (install it manually and re-run)"
  done
  step_ok "installed ${missing[*]}"
}

install_prereqs_linux() {
  have apt-get || fail 1 "no apt-get: install these manually and re-run: $*"
  local sudo=""
  [ "$(id -u)" -eq 0 ] || sudo="sudo"
  $sudo apt-get update >&2
  local pkg c
  for c in "$@"; do
    case "$c" in
      go) pkg="golang-go" ;;
      *)  pkg="$c" ;;
    esac
    $sudo apt-get install -y "$pkg" >&2
  done
}

install_prereqs_darwin() {
  if ! have brew; then
    log "    installing Homebrew (non-interactive)"
    NONINTERACTIVE=1 /bin/bash -c \
      "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)" >&2
    # brew is on PATH under /opt/homebrew (arm) or /usr/local (intel).
    eval "$(/opt/homebrew/bin/brew shellenv 2>/dev/null || /usr/local/bin/brew shellenv 2>/dev/null || true)"
  fi
  local c
  for c in "$@"; do
    brew install "$c" >&2
  done
}

step_clone() {
  step_start clone "clone/converge ${REPO_URL}"
  mkdir -p "$(dirname "$CHECKOUT_DIR")"
  if [ -d "${CHECKOUT_DIR}/.git" ]; then
    log "    existing checkout — fetching"
    git -C "$CHECKOUT_DIR" fetch --tags --prune origin >&2
    if [ -n "$REVISION" ]; then
      git -C "$CHECKOUT_DIR" checkout --quiet "$REVISION" >&2
      # If REVISION is a branch, fast-forward to its remote tip.
      git -C "$CHECKOUT_DIR" merge --ff-only "origin/${REVISION}" >&2 2>/dev/null || true
    fi
  else
    log "    cloning fresh"
    git clone "$REPO_URL" "$CHECKOUT_DIR" >&2
    [ -n "$REVISION" ] && git -C "$CHECKOUT_DIR" checkout --quiet "$REVISION" >&2
  fi
  local sha
  sha="$(git -C "$CHECKOUT_DIR" rev-parse HEAD)"
  REVISION_SHA="$sha"
  step_ok "at ${sha}"
}

step_setup() {
  step_start setup "vrooli setup"
  if [ "$SKIP_SETUP" -eq 1 ]; then
    step_skip "--skip-setup (node cannot run jobs until setup is run later)"
    return
  fi
  mkdir -p "$BOOTSTRAP_STATE_DIR"
  local sentinel="${BOOTSTRAP_STATE_DIR}/setup-${REVISION_SHA}.done"
  if [ "$FORCE_SETUP" -eq 0 ] && [ -f "$sentinel" ]; then
    step_skip "already set up at ${REVISION_SHA}"
    return
  fi
  local out rc
  out="$(mktemp)"
  if make -C "$CHECKOUT_DIR" setup >"$out" 2>&1; then
    rc=0
  else
    rc=$?
  fi
  cat "$out" >&2
  if [ "$rc" -ne 0 ]; then
    # Darwin degradation contract: a host whose vrooli build still refuses setup
    # on darwin must fail fast with an actionable pointer, not half-install.
    if [ "$OS" = "darwin" ] && grep -qiE 'darwin|macos|not supported|unsupported' "$out"; then
      rm -f "$out"
      fail 3 "vrooli setup refuses to run on this macOS host — the darwin setup gate is not yet open here. Track the macOS-compatibility / workspace-sandbox-cross-platform workstream, then re-run. (Do NOT --skip-setup unless prerequisites are already provisioned.)"
    fi
    rm -f "$out"
    fail 1 "make setup failed (exit ${rc}) — see output above"
  fi
  rm -f "$out"
  touch "$sentinel"
  step_ok "setup complete at ${REVISION_SHA}"
}

step_build_agent() {
  step_start build-agent "build node-agent from source"
  if [ -n "$AGENT_BIN_OVERRIDE" ]; then
    [ -x "$AGENT_BIN_OVERRIDE" ] || fail 1 "--agent-bin ${AGENT_BIN_OVERRIDE} is not executable"
    AGENT_BIN="$AGENT_BIN_OVERRIDE"
    step_skip "using prebuilt ${AGENT_BIN}"
    return
  fi
  make -C "${CHECKOUT_DIR}/scenarios/vrooli-bridge/agent" build >&2
  AGENT_BIN="${CHECKOUT_DIR}/scenarios/vrooli-bridge/agent/bin/vrooli-bridge-agent"
  [ -x "$AGENT_BIN" ] || fail 1 "agent build produced no binary at ${AGENT_BIN}"
  step_ok "built ${AGENT_BIN}"
}

step_build_cli() {
  step_start build-cli "build vrooli-bridge CLI from source"
  if [ -n "$BRIDGE_CLI_OVERRIDE" ]; then
    [ -x "$BRIDGE_CLI_OVERRIDE" ] || fail 1 "--bridge-cli ${BRIDGE_CLI_OVERRIDE} is not executable"
    BRIDGE_CLI="$BRIDGE_CLI_OVERRIDE"
    step_skip "using prebuilt ${BRIDGE_CLI}"
    return
  fi
  local cli_dir="${CHECKOUT_DIR}/scenarios/vrooli-bridge/cli"
  BRIDGE_CLI="${cli_dir}/bin/vrooli-bridge"
  ( cd "$cli_dir" && go build -o bin/vrooli-bridge . ) >&2
  [ -x "$BRIDGE_CLI" ] || fail 1 "CLI build produced no binary at ${BRIDGE_CLI}"
  step_ok "built ${BRIDGE_CLI}"
}

step_node_key() {
  step_start node-key "generate/load node keypair"
  mkdir -p "$STATE_DIR"
  chmod 700 "$STATE_DIR" 2>/dev/null || true
  # LoadOrCreate: reuses an existing key, generates one on first run. Idempotent.
  NODE_PUBLIC_KEY="$("$AGENT_BIN" --print-public-key --state-dir "$STATE_DIR")"
  [ -n "$NODE_PUBLIC_KEY" ] || fail 1 "agent produced no public key"
  step_ok "node public key ready"
}

step_pair_redeem() {
  step_start pair-redeem "redeem pairing code + pin control-plane key"
  # Already paired? The pin file + recorded node id together mean a prior redeem
  # succeeded; never spend another code.
  if [ -s "${STATE_DIR}/${PIN_FILE}" ] && [ -s "$NODE_ID_FILE" ]; then
    NODE_ID="$(cat "$NODE_ID_FILE")"
    step_skip "already paired as ${NODE_ID}"
    return
  fi
  [ -n "${BRIDGE_PAIRING_CODE:-}" ] || fail 2 "not yet paired and \$BRIDGE_PAIRING_CODE is unset — issue a code (\`vrooli-bridge pair issue\`) and pass it via the environment (never a flag)"

  local args=(--api-base "$CONTROL_PLANE_URL" pair redeem
    --public-key "$NODE_PUBLIC_KEY" --name "$NODE_NAME"
    --os "$OS" --arch "$ARCH" --state-dir "$STATE_DIR" --json)
  [ -n "$CAPABILITIES" ] && args+=(--capabilities "$CAPABILITIES")

  # The code rides the environment (BRIDGE_PAIRING_CODE), never argv. The CLI
  # pins control_plane.pub BEFORE burning the single-use code, so a redeem that
  # cannot reach the server leaves nothing spent.
  local out rc
  out="$(mktemp)"
  if "$BRIDGE_CLI" "${args[@]}" >"$out" 2>"${out}.err"; then
    rc=0
  else
    rc=$?
  fi
  if [ "$rc" -ne 0 ]; then
    cat "${out}.err" >&2
    local errtxt
    errtxt="$(cat "${out}.err")"
    rm -f "$out" "${out}.err"
    if printf '%s' "$errtxt" | grep -qiE 'invalid|expired|already|not found|redeem'; then
      fail 4 "pairing code rejected (invalid/expired/already used) — issue a fresh code and re-run: nothing was spent on this node"
    fi
    fail 1 "redeem failed (exit ${rc}) — see error above"
  fi
  NODE_ID="$(jq -r '.node_id // empty' "$out")"
  rm -f "$out" "${out}.err"
  [ -n "$NODE_ID" ] || fail 1 "redeem returned no node id"
  printf '%s' "$NODE_ID" >"$NODE_ID_FILE"
  step_ok "paired as ${NODE_ID}"
}

step_pin_verify() {
  step_start pin-verify "verify pinned control-plane key"
  local pin="${STATE_DIR}/${PIN_FILE}"
  [ -s "$pin" ] || fail 1 "pinned control-plane key missing at ${pin} (the agent refuses to start without it)"
  # NODE_ID may be empty if pair-redeem was skipped on a fresh var; reload it.
  [ -n "$NODE_ID" ] || NODE_ID="$(cat "$NODE_ID_FILE" 2>/dev/null || true)"
  [ -n "$NODE_ID" ] || fail 1 "no recorded node id at ${NODE_ID_FILE}"
  step_ok "pinned key present, node ${NODE_ID}"
}

step_service_install() {
  step_start service-install "install + start node-agent service"
  local cfg
  mapfile -t cfg < <(agent_service_args)

  # Idempotent: if the service is already installed, running, and its unit file
  # byte-matches what we would render now, do nothing (no restart).
  local status_json
  status_json="$("$AGENT_BIN" service status "${cfg[@]}" --json 2>/dev/null || true)"
  if [ -n "$status_json" ]; then
    local installed running unit_path
    installed="$(printf '%s' "$status_json" | jq -r '.installed // false')"
    running="$(printf '%s' "$status_json" | jq -r '.running // false')"
    unit_path="$(printf '%s' "$status_json" | jq -r '.unit_path // .unitPath // empty')"
    if [ "$installed" = "true" ] && [ "$running" = "true" ] && [ -n "$unit_path" ] && [ -f "$unit_path" ]; then
      local desired
      desired="$("$AGENT_BIN" --print-service-unit "${cfg[@]}")"
      if [ "$desired" = "$(cat "$unit_path")" ]; then
        step_skip "service already installed, running, and up to date"
        return
      fi
    fi
  fi

  local install_json
  install_json="$("$AGENT_BIN" service install "${cfg[@]}" --json)"
  local running
  running="$(printf '%s' "$install_json" | jq -r '.running // false')"
  [ "$running" = "true" ] || fail 1 "service installed but not running — inspect: journalctl --user -u ${UNIT_NAME}"
  step_ok "service installed and running"
}

step_autostart() {
  step_start autostart "enable headless auto-start"
  if [ "$OS" = "darwin" ]; then
    # launchd KeepAlive restarts the agent on crash; surviving logout on a
    # headless Mac requires auto-login (a manual node pre-step; see README).
    step_ok "launchd KeepAlive handles restart (auto-login is a manual pre-step)"
    return
  fi
  # Linux: a systemd --user service stops at logout unless the user lingers.
  local target_user="${SERVICE_USER:-$(id -un)}"
  local linger
  linger="$(loginctl show-user "$target_user" -p Linger --value 2>/dev/null || echo no)"
  if [ "$linger" = "yes" ]; then
    step_skip "linger already enabled for ${target_user}"
    return
  fi
  local sudo=""
  [ "$(id -u)" -eq 0 ] || [ "$target_user" = "$(id -un)" ] || sudo="sudo"
  if loginctl enable-linger "$target_user" >&2 2>/dev/null || $sudo loginctl enable-linger "$target_user" >&2; then
    step_ok "enabled linger for ${target_user}"
  else
    fail 1 "could not enable linger for ${target_user} (needed so the agent survives logout/reboot): run 'loginctl enable-linger ${target_user}' as an admin and re-run"
  fi
}

# journal_channel_state — echoes the most recent channel lifecycle verdict from
# the user journal: "open" (connected), "down" (session ended / refused), or ""
# (no channel activity yet). Used to confirm the dial-out is currently live.
journal_channel_state() {
  local lines
  lines="$(journalctl --user -u "$UNIT_NAME" --no-pager 2>/dev/null \
    | grep -E "channel: (${CONNECTED_MARKER}|session ended|refused the channel)" | tail -1 || true)"
  if printf '%s' "$lines" | grep -q "$CONNECTED_MARKER"; then
    echo open
  elif [ -n "$lines" ]; then
    echo down
  else
    echo ""
  fi
}

step_verify_online() {
  step_start verify-online "confirm dial-out channel is live"
  local have_journal=0
  journalctl --user -n0 >/dev/null 2>&1 && have_journal=1

  local cfg
  mapfile -t cfg < <(agent_service_args)

  local deadline=$(( $(date +%s) + VERIFY_TIMEOUT ))
  while :; do
    local running sj
    sj="$("$AGENT_BIN" service status "${cfg[@]}" --json 2>/dev/null || true)"
    running="$(printf '%s' "$sj" | jq -r '.running // false' 2>/dev/null || echo false)"

    if [ "$have_journal" -eq 1 ]; then
      case "$(journal_channel_state)" in
        open) step_ok "agent connected (dial-out stream open)"; return ;;
      esac
    else
      # Degraded check: no user journal on this host, so the channel log is not
      # observable here. Running service is the best available signal.
      if [ "$running" = "true" ]; then
        step_ok "service running (journal unavailable: could not confirm channel log directly)"
        return
      fi
    fi

    if [ "$(date +%s)" -ge "$deadline" ]; then
      fail 1 "agent did not report a live dial-out channel within ${VERIFY_TIMEOUT}s — inspect: journalctl --user -u ${UNIT_NAME}"
    fi
    sleep 2
  done
}

# --- run ---------------------------------------------------------------------

REVISION_SHA=""

main() {
  marker run-start "" "vrooli-bridge node bootstrap"
  log "vrooli-bridge bootstrap: node=${NODE_NAME} cp=${CONTROL_PLANE_URL} checkout=${CHECKOUT_DIR} state=${STATE_DIR}"
  step_detect_os
  step_prereqs
  step_clone
  step_setup
  step_build_agent
  step_build_cli
  step_node_key
  step_pair_redeem
  step_pin_verify
  step_service_install
  step_autostart
  step_verify_online
  marker run-ok "" "node ${NODE_ID} paired and online"
  log "bootstrap complete: node ${NODE_ID} is paired, online, and set to auto-start."
}

main "$@"
