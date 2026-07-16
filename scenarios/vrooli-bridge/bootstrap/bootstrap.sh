#!/usr/bin/env bash
#
# vrooli-bridge node bootstrap — take a fresh node from raw OS to a paired,
# ONLINE, auto-starting fleet agent in one idempotent run.
#
# It converges the Vrooli source, receives control-plane-cross-built binaries,
# runs the transferred Vrooli CLI to set the node up, generates
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
CHECKOUT_DIR="${BRIDGE_CHECKOUT_DIR:-$HOME/Vrooli}"

# Working-tree source mode: when SOURCE_DIR is set the control plane has already
# shipped its LOCAL tree here over SSH, so step_clone verifies that pre-synced tree
# instead of cloning/fetching (the node has no git remote to fetch and no .git
# history — provenance is REVISION as a base commit + SOURCE_DIGEST). Empty ==
# pinned mode (clone/fetch REVISION).
SOURCE_DIR="${BRIDGE_SOURCE_DIR:-}"
SOURCE_DIGEST="${BRIDGE_SOURCE_DIGEST:-}"
STATE_DIR="${BRIDGE_AGENT_STATE_DIR:-${XDG_STATE_HOME:-$HOME/.local/state}/vrooli-bridge-agent}"
WORK_DIR="${BRIDGE_WORK_DIR:-}"
SERVICE_USER="${BRIDGE_SERVICE_USER:-}"
CAPABILITIES="${BRIDGE_CAPABILITIES:-}"
VERIFY_TIMEOUT="${BRIDGE_VERIFY_TIMEOUT:-120}"

# Setup profile: the operator-chosen shape of the node-side `vrooli setup` this
# script runs. Empty values fall through to `vrooli setup`'s own defaults.
SETUP_ENVIRONMENT="${BRIDGE_SETUP_ENVIRONMENT:-}"
SETUP_RESOURCES="${BRIDGE_SETUP_RESOURCES:-}"
SETUP_SCENARIOS="${BRIDGE_SETUP_SCENARIOS:-}"
INCLUDE_OPTIONAL=0

SKIP_PREREQS=0
SKIP_SETUP=0
FORCE_SETUP=0
VROOLI_BIN_OVERRIDE=""
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
  --checkout-dir DIR        Where the node's checkout lives.  (BRIDGE_CHECKOUT_DIR; default: $HOME/Vrooli)
  --source-dir DIR          Verify a pre-synced working tree here instead of
                            cloning (working-tree source mode; the control plane
                            ships its local tree over SSH). (BRIDGE_SOURCE_DIR)
  --source-digest DIGEST    Content digest of the pre-synced tree; keys the setup
                            sentinel so re-shipped work re-runs setup. (BRIDGE_SOURCE_DIGEST)
  --state-dir DIR           Agent credential/state dir.       (BRIDGE_AGENT_STATE_DIR; default: XDG state)
  --work-dir DIR            Dir the agent runs jobs in.       (BRIDGE_WORK_DIR; default: checkout dir)
  --service-user USER       OS principal the service runs as. (BRIDGE_SERVICE_USER; default: current user)
  --capabilities LIST       Comma-separated verb namespaces.  (BRIDGE_CAPABILITIES)
  --verify-timeout SECONDS  Dial-out verification budget.     (BRIDGE_VERIFY_TIMEOUT; default: 120)

Setup profile (shapes the node-side `vrooli setup`; empty = vrooli setup default):
  --setup-environment ENV   development|production|minimal.   (BRIDGE_SETUP_ENVIRONMENT)
  --setup-resources SEL     enabled|none|<comma-list>.        (BRIDGE_SETUP_RESOURCES)
  --setup-scenarios SEL     none|all|<comma-list>.            (BRIDGE_SETUP_SCENARIOS)
  --include-optional        Also apply optional host safeguards.

Pre-satisfied-prerequisite shortcuts (each documented in README.md):
  --skip-prereqs            Assume git/curl (the clone prerequisites) are present.
  --skip-setup              Skip `make setup` (node cannot run jobs until it is
                            run later, but pairing/online/auto-start still work).
  --force-setup             Run `make setup` even if its revision sentinel exists.
  --vrooli-bin PATH         Use a prebuilt Vrooli CLI for setup (with PATH.fp).
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
    --source-dir)        SOURCE_DIR="$2"; shift 2 ;;
    --source-digest)     SOURCE_DIGEST="$2"; shift 2 ;;
    --state-dir)         STATE_DIR="$2"; shift 2 ;;
    --work-dir)          WORK_DIR="$2"; shift 2 ;;
    --service-user)      SERVICE_USER="$2"; shift 2 ;;
    --capabilities)      CAPABILITIES="$2"; shift 2 ;;
    --verify-timeout)    VERIFY_TIMEOUT="$2"; shift 2 ;;
    --setup-environment) SETUP_ENVIRONMENT="$2"; shift 2 ;;
    --setup-resources)   SETUP_RESOURCES="$2"; shift 2 ;;
    --setup-scenarios)   SETUP_SCENARIOS="$2"; shift 2 ;;
    --include-optional)  INCLUDE_OPTIONAL=1; shift ;;
    --skip-prereqs)      SKIP_PREREQS=1; shift ;;
    --skip-setup)        SKIP_SETUP=1; shift ;;
    --force-setup)       FORCE_SETUP=1; shift ;;
    --vrooli-bin)        VROOLI_BIN_OVERRIDE="$2"; shift 2 ;;
    --agent-bin)         AGENT_BIN_OVERRIDE="$2"; shift 2 ;;
    --bridge-cli)        BRIDGE_CLI_OVERRIDE="$2"; shift 2 ;;
    -h|--help)           usage; exit 0 ;;
    *) log "unknown argument: $1"; usage; exit 2 ;;
  esac
done

# --- derived paths & validation ----------------------------------------------

[ -n "$CONTROL_PLANE_URL" ] || { log "error: --control-plane-url (or \$BRIDGE_CONTROL_PLANE_URL) is required"; usage; exit 2; }
[ -n "$NODE_NAME" ] || NODE_NAME="$(hostname)"

# Defence-in-depth: the API boundary already rejects shell metacharacters in the
# setup-profile values (api/internal/onboard.validateSetupProfile, mirroring the
# cprev revision filter), but this script may also be run by hand, so it re-checks
# before the values are spliced into `make setup SETUP_ARGS=…`. A metachar here is
# a config error (exit 2), never a silent injection. The set mirrors
# cprev.shellMetachars; a comma is allowed (resource/scenario selections are
# comma lists). Checked char-by-char so every metachar (including glob and
# bracket characters) is matched as a literal.
SETUP_METACHARS='|&;<>()$`\"'"'"'*?[]{}!#~ '
validate_setup_value() {
  local name="$1" value="$2" i c
  local n=${#value}
  for (( i = 0; i < n; i++ )); do
    c="${value:i:1}"
    case "$c" in
      $'\t'|$'\n'|$'\r')
        log "error: --${name} value contains whitespace; setup-profile values are spliced into the node setup command"
        exit 2 ;;
    esac
    case "$SETUP_METACHARS" in
      *"$c"*)
        log "error: --${name} value contains a disallowed shell character (${c}); setup-profile values are spliced into the node setup command"
        exit 2 ;;
    esac
  done
}
validate_setup_value setup-environment "$SETUP_ENVIRONMENT"
validate_setup_value setup-resources   "$SETUP_RESOURCES"
validate_setup_value setup-scenarios   "$SETUP_SCENARIOS"
case "$SETUP_ENVIRONMENT" in
  ""|development|production|minimal) ;;
  *) log "error: --setup-environment must be development, production, or minimal"; exit 2 ;;
esac

readonly BOOTSTRAP_STATE_DIR="${STATE_DIR}/.bootstrap"
readonly NODE_ID_FILE="${STATE_DIR}/node_id"
AGENT_BIN="$AGENT_BIN_OVERRIDE"
BRIDGE_CLI="$BRIDGE_CLI_OVERRIDE"
PREBUILT_MODE=0
if [ -n "$VROOLI_BIN_OVERRIDE" ] && [ -n "$AGENT_BIN_OVERRIDE" ] && [ -n "$BRIDGE_CLI_OVERRIDE" ]; then
  PREBUILT_MODE=1
fi
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
    log "    note: macOS nodes need Remote Login enabled (and auto-login if headless); Docker is only for container workloads later, not to onboard (see README)."
  fi
  step_ok "os=${OS} arch=${ARCH}"
}

step_receive_artifacts() {
  step_start prebuilt-artifacts "verify transferred prebuilt binaries"
  if [ "$PREBUILT_MODE" -ne 1 ]; then
    step_skip "complete prebuilt bundle not supplied; legacy source-build path remains available"
    return
  fi
  local bin sidecar fingerprint="" value
  for bin in "$VROOLI_BIN_OVERRIDE" "$BRIDGE_CLI_OVERRIDE" "$AGENT_BIN_OVERRIDE"; do
    [ -x "$bin" ] || fail 1 "received prebuilt binary is not executable: ${bin}"
    sidecar="${bin}.fp"
    [ -s "$sidecar" ] || fail 1 "received prebuilt binary has no freshness sidecar: ${sidecar}"
    value="$(tr -d '[:space:]' <"$sidecar")"
    [ -n "$value" ] || fail 1 "received prebuilt freshness sidecar is empty: ${sidecar}"
    if [ -z "$fingerprint" ]; then
      fingerprint="$value"
    elif [ "$fingerprint" != "$value" ]; then
      fail 1 "received prebuilt binaries do not share one source fingerprint"
    fi
  done
  step_ok "received prebuilt binaries for ${OS}/${ARCH} (fingerprint ${fingerprint:0:12})"
}

# ensure_cmd <name> — is a command available?
have() { command -v "$1" >/dev/null 2>&1; }

# have_passwordless_sudo — true when privileged commands can run non-interactively:
# already root, or `sudo -n` works without a password. Never prompts, so a headless
# run can branch on it instead of hanging on a password prompt.
have_passwordless_sudo() {
  [ "$(id -u)" -eq 0 ] && return 0
  sudo -n true 2>/dev/null
}

# as_root <cmd...> — run a command with root privilege: directly when already root,
# else via non-interactive `sudo -n`. NEVER prompts (a headless run must degrade
# loudly, never wedge on a password prompt). Caller must have checked
# have_passwordless_sudo first.
as_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
  else
    sudo -n "$@"
  fi
}

# profile_hash <string> — a short, stable, filesystem-safe digest of its input,
# used to key the setup sentinel on the setup profile so a changed profile re-runs
# setup while an identical one stays a no-op. Prefers sha256; falls back to cksum
# (always present) so no extra dependency is required.
profile_hash() {
  local input="$1"
  if have sha256sum; then
    printf '%s' "$input" | sha256sum | cut -c1-16
  elif have shasum; then
    printf '%s' "$input" | shasum -a 256 | cut -c1-16
  else
    printf '%s' "$input" | cksum | tr -d ' ' | cut -c1-16
  fi
}

# compose_setup_args — the SETUP_ARGS string threaded into `make setup`. Each value
# was metachar-validated at parse time, so a plain space-join is safe (make word-
# splits SETUP_ARGS into `vrooli setup` argv). Empty values are omitted, falling
# through to `vrooli setup`'s own defaults.
compose_setup_args() {
  local a=""
  [ -n "$SETUP_ENVIRONMENT" ] && a="${a} --environment ${SETUP_ENVIRONMENT}"
  [ -n "$SETUP_RESOURCES" ]   && a="${a} --resources ${SETUP_RESOURCES}"
  [ -n "$SETUP_SCENARIOS" ]   && a="${a} --scenarios ${SETUP_SCENARIOS}"
  [ "$INCLUDE_OPTIONAL" -eq 1 ] && a="${a} --include-optional"
  printf '%s' "${a# }"
}

# parse_needs_sudo <setup-output-file> — echo a single-line, marker-safe summary of
# the requirements `vrooli setup` skipped for privilege, or nothing when none were.
# It reads the grouped renderer's "Needs sudo — …(N):" block (internal/setup/
# requirements_report.go): the header line followed by two-space-indented item
# rows. Degrades gracefully if the format shifts — a header with no parseable items
# still yields a generic loud detail rather than a false all-clear.
parse_needs_sudo() {
  local out="$1"
  grep -qE '^Needs sudo' "$out" 2>/dev/null || return 0
  local names
  names="$(awk '
    /^Needs sudo/ { insec=1; next }
    insec && /^[^ ]/ { insec=0 }
    insec && /^  [^ ]/ { printf "%s%s", sep, $2; sep="," }
  ' "$out" 2>/dev/null)"
  # Keep the detail marker-safe: no embedded double quotes (the marker wraps
  # detail in quotes) and no newlines (awk already produced one line).
  names="$(printf '%s' "$names" | tr -d '"')"
  if [ -z "$names" ]; then
    printf 'ran unprivileged; root-required requirements were skipped (see setup output above)'
    return 0
  fi
  local count
  count="$(printf '%s' "$names" | awk -F, '{print NF}')"
  printf '%s root-required requirement(s) skipped for privilege: %s' "$count" "$names"
}

step_prereqs() {
  # Only the true chicken-and-egg set needed to CLONE the source: git + curl.
  # Every heavier toolchain (go, pnpm, node, docker…) is owned by project-level
  # `vrooli setup`, which runs from the cloned tree in step_setup — installing
  # them here too would duplicate that authority and drift from it. The
  # post-setup toolchain guard (step_toolchain_guard) confirms setup actually
  # delivered go/pnpm before the build steps rely on them.
  step_start prereqs "ensure git/curl (clone prerequisites)"
  if [ "$PREBUILT_MODE" -eq 1 ] && [ -n "$SOURCE_DIR" ]; then
    step_skip "pre-synced tree + prebuilt binaries require no clone prerequisites"
    return
  fi
  if [ "$SKIP_PREREQS" -eq 1 ]; then
    step_skip "--skip-prereqs"
    return
  fi
  local missing=()
  local c
  for c in git curl; do
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
  # git and curl are the only prerequisites this step installs; the package name
  # matches the command name for both, so no per-tool mapping is needed.
  have apt-get || fail 1 "no apt-get: install these manually and re-run: $*"
  # apt-get needs root. Use non-interactive sudo (or run directly as root); never
  # fall back to interactive sudo, which would hang a headless onboarding run.
  if ! have_passwordless_sudo; then
    fail 1 "installing prerequisites (${*}) needs root, but passwordless sudo is unavailable on this node — provision it (onboard --provision-sudo) or install these manually and re-run with --skip-prereqs"
  fi
  as_root apt-get update >&2
  local c
  for c in "$@"; do
    as_root apt-get install -y "$c" >&2
  done
}

# install_prereqs_darwin — deliberately does NOT auto-install Homebrew. Only git
# and curl can reach here (setup owns every heavier toolchain), and on macOS curl
# always ships while git arrives with the Xcode Command Line Tools. Directing the
# operator to `xcode-select --install` provisions git through Apple's own supported
# path rather than bootstrapping a whole package manager during a headless
# onboarding run — a smaller, more predictable footprint on someone else's Mac.
install_prereqs_darwin() {
  fail 1 "missing macOS clone prerequisite(s): $* — run 'xcode-select --install' (installs git; curl ships with macOS) then re-run. This bootstrap does not auto-install Homebrew; project 'vrooli setup' owns every other toolchain."
}

step_clone() {
  step_start clone "clone/converge ${REPO_URL}"

  # Working-tree source mode: the control plane pre-shipped its LOCAL tree into
  # SOURCE_DIR over SSH, so there is nothing to clone/fetch. Verify the tree
  # landed (exists, non-empty) and record its dirty provenance instead of a git
  # sha — the node has no .git history here. The setup sentinel keys on
  # SOURCE_DIGEST so re-shipping changed work re-runs setup.
  if [ -n "$SOURCE_DIR" ]; then
    [ -d "$SOURCE_DIR" ] || fail 1 "--source-dir ${SOURCE_DIR} does not exist (the control plane's working-tree ship did not land here)"
    [ -n "$(ls -A "$SOURCE_DIR" 2>/dev/null)" ] || fail 1 "--source-dir ${SOURCE_DIR} is empty (the working-tree ship is incomplete) — re-run onboarding"
    CHECKOUT_DIR="$SOURCE_DIR"
    if [ -n "$REVISION" ]; then
      REVISION_SHA="${REVISION}+dirty"
    else
      REVISION_SHA="working-tree"
    fi
    SETUP_DIGEST_KEY="$SOURCE_DIGEST"
    if [ -n "$SOURCE_DIGEST" ]; then
      step_ok "using pre-synced working tree at ${SOURCE_DIR} (${REVISION_SHA}, digest ${SOURCE_DIGEST})"
    else
      step_ok "using pre-synced working tree at ${SOURCE_DIR} (${REVISION_SHA})"
    fi
    return
  fi

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

  # The sentinel is keyed on BOTH the revision and the setup profile, so changing
  # the profile (environment/resources/scenarios/include-optional) re-runs setup
  # while an identical profile at the same revision stays a no-op.
  # SETUP_DIGEST_KEY is the working-tree content digest (empty in pinned mode); it
  # folds into the sentinel key so re-shipping changed uncommitted work (same base
  # revision, new digest) re-runs setup while an identical tree stays a no-op.
  local setup_args profile_token sentinel
  setup_args="$(compose_setup_args)"
  profile_token="$(profile_hash "${REVISION_SHA}|${setup_args}|${SETUP_DIGEST_KEY}")"
  sentinel="${BOOTSTRAP_STATE_DIR}/setup-${REVISION_SHA}-${profile_token}.done"
  if [ "$FORCE_SETUP" -eq 0 ] && [ -f "$sentinel" ]; then
    step_skip "already set up at ${REVISION_SHA} for this profile"
    return
  fi

  # Project-level `vrooli setup` is the sole machine-provisioning authority on the
  # node. Run it ELEVATED when passwordless sudo is available so no requirement is
  # skipped for privilege; otherwise run unprivileged and surface the skipped
  # root-required items LOUDLY (a warning in the step detail, never a failure).
  local elevated=0
  local -a cmd
  if [ "$PREBUILT_MODE" -eq 1 ]; then
    # Point freshness checks at the exact plain tree the control plane shipped.
    # The transferred .fp sidecar must match it, so this invocation cannot try a
    # node-side rebuild before setup has installed Go.
    cmd=(env "VROOLI_SOURCE_ROOT=${CHECKOUT_DIR}" "$VROOLI_BIN_OVERRIDE" setup)
    [ -n "$SETUP_ENVIRONMENT" ] && cmd+=(--environment "$SETUP_ENVIRONMENT")
    [ -n "$SETUP_RESOURCES" ] && cmd+=(--resources "$SETUP_RESOURCES")
    [ -n "$SETUP_SCENARIOS" ] && cmd+=(--scenarios "$SETUP_SCENARIOS")
    [ "$INCLUDE_OPTIONAL" -eq 1 ] && cmd+=(--include-optional)
  else
    cmd=(make -C "$CHECKOUT_DIR" setup "SETUP_ARGS=${setup_args}")
  fi
  # Homebrew refuses to run as root. On macOS, keep setup under the SSH user and
  # let its individual host requirements use the provisioned passwordless sudo
  # where needed. Linux retains whole-setup elevation.
  if [ "$OS" != "darwin" ] && have_passwordless_sudo; then
    elevated=1
    [ "$(id -u)" -eq 0 ] || cmd=(sudo -n "${cmd[@]}")
  fi

  local out rc
  out="$(mktemp)"
  if ( cd "$CHECKOUT_DIR" && "${cmd[@]}" ) >"$out" 2>&1; then
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
    fail 1 "vrooli setup failed (exit ${rc}) — see output above"
  fi

  # Name the skipped root-required requirements loudly so they are visible in the
  # op step events, CLI watch output, and the UI — a warning, not a failure.
  local skipped
  skipped="$(parse_needs_sudo "$out")"
  rm -f "$out"
  touch "$sentinel"

  local mode="unprivileged"
  [ "$elevated" -eq 1 ] && mode="elevated"
  if [ -n "$skipped" ]; then
    step_ok "setup complete at ${REVISION_SHA} (${mode}) — ${skipped}"
  else
    step_ok "setup complete at ${REVISION_SHA} (${mode})"
  fi
}

# toolchain_dirs — the known, PATH-independent locations project `vrooli setup`
# installs the build toolchains into. setup writes NO shell-profile PATH entry: it
# installs into these fixed locations and relies on the operator's interactive login
# shell already carrying them. A non-interactive SSH shell (which is what a headless
# onboarding run is) does not source the login profile, so a tool setup just
# installed can be present on disk yet invisible on this process's PATH. The guard
# probes these to recover such a tool and amend PATH for the build steps.
#   * $HOME/.vrooli/bin  — no-sudo standalone tools, e.g. pnpm (internal/runtime tool installer)
#   * $HOME/.local/bin    — setup's symlink target for go-installed binaries
#   * $HOME/go/bin        — Go's default install target ($GOBIN)
#   * /usr/local/go/bin   — official Go tarball layout
#   * /opt/homebrew/bin, /usr/local/bin — Homebrew (arm/intel) + system bins
toolchain_dirs() {
  printf '%s\n' \
    "${HOME}/.vrooli/bin" \
    "${HOME}/.local/bin" \
    "${HOME}/go/bin" \
    "/usr/local/go/bin" \
    "/opt/homebrew/bin" \
    "/usr/local/bin"
}

# locate_offpath <cmd> — echo the directory of executable <cmd> found in a known
# toolchain dir but not on PATH, or nothing (return 1). bash-3.2-safe: while-read
# over toolchain_dirs, no mapfile.
locate_offpath() {
  local bin="$1" dir
  while IFS= read -r dir; do
    [ -n "$dir" ] || continue
    if [ -x "${dir}/${bin}" ]; then
      printf '%s' "$dir"
      return 0
    fi
  done < <(toolchain_dirs)
  return 1
}

# The toolchains the node must have to build from source (agent + CLI) and to run
# jobs (scenario UIs need pnpm). setup owns installing them; this list is only what
# the guard verifies.
readonly BUILD_TOOLCHAINS="go pnpm"

step_toolchain_guard() {
  # Fail fast and legibly when `vrooli setup` did not deliver a build toolchain,
  # instead of letting the build steps die with a confusing "go: command not
  # found". Also repairs the non-interactive-SSH PATH gap: a tool present only in a
  # known install dir is recovered onto PATH for the rest of this run.
  step_start toolchain "verify build toolchains resolve"
  if [ "$PREBUILT_MODE" -eq 1 ]; then
    step_skip "prebuilt binaries received; no node-side source build toolchain required"
    return
  fi
  local recovered="" absent="" tool dir
  for tool in $BUILD_TOOLCHAINS; do
    if have "$tool"; then
      continue
    fi
    if dir="$(locate_offpath "$tool")"; then
      case ":${PATH}:" in
        *":${dir}:"*) ;;
        *) PATH="${dir}:${PATH}"; export PATH ;;
      esac
      recovered="${recovered:+${recovered}, }${tool} (${dir})"
    else
      absent="${absent:+${absent} }${tool}"
    fi
  done
  if [ -n "$absent" ]; then
    # Name the gap AND the setup authority that should have closed it, so the
    # operator fixes the real cause (often: setup skipped root-required items for
    # privilege) rather than chasing a downstream build error.
    fail 1 "build toolchain(s) missing after setup: ${absent} — the node's 'vrooli setup' toolchain requirements did not deliver them (checked PATH and known install dirs: \$HOME/.vrooli/bin, \$HOME/.local/bin, \$HOME/go/bin, /usr/local/go/bin, /opt/homebrew/bin, /usr/local/bin). Run 'vrooli setup status' on this node to see why (e.g. a group skipped for privilege) then re-run; use --skip-setup only if you provision these another way."
  fi
  if [ -n "$recovered" ]; then
    step_ok "build toolchains resolve — recovered off-PATH and amended PATH: ${recovered}"
  else
    step_ok "build toolchains resolve on PATH"
  fi
}

step_build_agent() {
  step_start build-agent "prepare node-agent"
  if [ -n "$AGENT_BIN_OVERRIDE" ]; then
    [ -x "$AGENT_BIN_OVERRIDE" ] || fail 1 "--agent-bin ${AGENT_BIN_OVERRIDE} is not executable"
    AGENT_BIN="$AGENT_BIN_OVERRIDE"
    step_skip "received prebuilt ${AGENT_BIN}; no node-side build"
    return
  fi
  make -C "${CHECKOUT_DIR}/scenarios/vrooli-bridge/agent" build >&2
  AGENT_BIN="${CHECKOUT_DIR}/scenarios/vrooli-bridge/agent/bin/vrooli-bridge-agent"
  [ -x "$AGENT_BIN" ] || fail 1 "agent build produced no binary at ${AGENT_BIN}"
  step_ok "built ${AGENT_BIN}"
}

step_build_cli() {
  step_start build-cli "prepare vrooli-bridge CLI"
  if [ -n "$BRIDGE_CLI_OVERRIDE" ]; then
    [ -x "$BRIDGE_CLI_OVERRIDE" ] || fail 1 "--bridge-cli ${BRIDGE_CLI_OVERRIDE} is not executable"
    BRIDGE_CLI="$BRIDGE_CLI_OVERRIDE"
    step_skip "received prebuilt ${BRIDGE_CLI}; no node-side build"
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
  local cfg=() line
  # bash-3.2-safe array fill (macOS stock bash has no mapfile/readarray).
  while IFS= read -r line; do cfg+=("$line"); done < <(agent_service_args)

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
  # A direct enable may be permitted (root, or a self-linger the host's polkit
  # allows); try it first. Otherwise elevate with NON-INTERACTIVE sudo only — an
  # interactive password prompt would hang a headless onboarding run, so we degrade
  # loudly instead of blocking.
  if loginctl enable-linger "$target_user" >&2 2>/dev/null; then
    step_ok "enabled linger for ${target_user}"
    return
  fi
  if have_passwordless_sudo && as_root loginctl enable-linger "$target_user" >&2; then
    step_ok "enabled linger for ${target_user} (via sudo)"
    return
  fi
  fail 1 "could not enable linger for ${target_user} without an interactive password prompt (needed so the agent survives logout/reboot): provision passwordless sudo (onboard --provision-sudo) or run 'sudo loginctl enable-linger ${target_user}' as an admin and re-run"
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

  local cfg=() line
  # bash-3.2-safe array fill (macOS stock bash has no mapfile/readarray).
  while IFS= read -r line; do cfg+=("$line"); done < <(agent_service_args)

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
# Working-tree content digest folded into the setup sentinel key (empty in pinned
# mode); step_clone sets it in working-tree source mode.
SETUP_DIGEST_KEY=""

main() {
  marker run-start "" "vrooli-bridge node bootstrap"
  log "vrooli-bridge bootstrap: node=${NODE_NAME} cp=${CONTROL_PLANE_URL} checkout=${CHECKOUT_DIR} state=${STATE_DIR}"
  step_detect_os
  step_receive_artifacts
  step_prereqs
  step_clone
  # --work-dir defaults to the effective checkout. Resolve that default only
  # after step_clone because working-tree mode replaces CHECKOUT_DIR with the
  # control plane's pre-synced source directory.
  [ -n "$WORK_DIR" ] || WORK_DIR="$CHECKOUT_DIR"
  step_setup
  step_toolchain_guard
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
