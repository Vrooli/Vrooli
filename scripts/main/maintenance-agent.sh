#!/usr/bin/env bash
# maintenance-agent.sh ---------------------------------------------------------
# Run long Claude Code maintenance tasks safely, in batches – now supports
# multiple sequential tasks:
#
#   ./maintenance-agent.sh "Task 1" 100 "Task 2" 100 "Task 3" 50
#
# Each prompt/turn-limit pair is executed in order, re-using the same CLI
# settings but starting a fresh Claude session for every task.
#
# DEPENDENCIES
#   • claude or claude-code CLI on PATH
#   • jq • bash • sleep (coreutils)
# ------------------------------------------------------------------------------
set -euo pipefail

###############################################################################
# Privilege handling
###############################################################################
# If run with sudo/root, drop privileges to FALLBACK_USER instead of exiting.
FALLBACK_USER=${FALLBACK_USER:-matt}

if [[ $EUID -eq 0 ]]; then
  echo "⚠️  Detected root privileges. Re-executing as '${FALLBACK_USER}'..."
  if id -u "$FALLBACK_USER" &>/dev/null; then
    if command -v sudo &>/dev/null; then
      # Re-exec the same script under the fallback user, preserving all args.
      exec sudo -u "$FALLBACK_USER" -- "$(realpath "$0")" "$@"
    elif command -v su &>/dev/null; then
      # Fallback if sudo is unavailable.
      exec su -s /bin/bash - "$FALLBACK_USER" -c "$(printf '%q ' "$(realpath "$0")" "$@")"
    else
      echo "❌  Neither sudo nor su available for privilege drop." >&2
      exit 1
    fi
  else
    echo "❌  Fallback user '${FALLBACK_USER}' does not exist." >&2
    exit 1
  fi
fi

# Ensure $HOME is correct after any privilege drop.
if [[ -z "${HOME:-}" || "$HOME" == "/root" ]]; then
  HOME=$(getent passwd "$USER" | cut -d: -f6)
  export HOME
fi

###############################################################################
# Config
###############################################################################
BATCH_SIZE=${BATCH_SIZE:-50}
COOLDOWN_SECONDS=${COOLDOWN_SECONDS:-5}
COOLDOWN_BETWEEN_TASKS=${COOLDOWN_BETWEEN_TASKS:-10}
# Tools Claude is allowed to use (NO “Task”!  Bash(*) gives it full shell access)
NEEDED_TOOLS=(
  "Bash" "Edit" "Write" "Glob" "Grep" "LS" "ReadFile"
  "MultiEdit" "Git" "WebSearch" "WebFetch"
)

# Build --allowedTools arguments from NEEDED_TOOLS
ALLOWED_TOOLS_ARGS=()
for tool in "${NEEDED_TOOLS[@]}"; do
  if [[ $tool == "Bash" ]]; then
    ALLOWED_TOOLS_ARGS+=(--allowedTools "Bash(*)")
  else
    ALLOWED_TOOLS_ARGS+=(--allowedTools "$tool")
  fi
done

# Log directory per day
log_dir="logs/$(date +%F)"
mkdir -p "$log_dir"
trap 'echo -e "\n🔴  Aborted"; kill 0; exit 130' INT

###############################################################################
# Discover all settings files that could block us
###############################################################################
settings_files=()
for f in \
  ".claude/settings.local.json" \
  ".claude/settings.json" \
  "$HOME/.claude/settings.json"
do
  [[ -f $f ]] && settings_files+=("$f")
done

# If none exist yet, create local settings (highest precedence)
if [[ ${#settings_files[@]} -eq 0 ]]; then
  mkdir -p .claude
  echo '{ "permissions": { "allow": [] } }' > .claude/settings.local.json
  settings_files=( ".claude/settings.local.json" )
fi

###############################################################################
# Helper: patch one file in-place
###############################################################################
patch_settings() {
  local file=$1
  local tmp; tmp=$(mktemp)

  jq --argjson need "$(printf '%s\n' "${NEEDED_TOOLS[@]}" \
                      | jq -R . | jq -s '.')" '
    (.permissions //= {}) |
    (.permissions.allow //= []) |
    # --------------------------------------------------
    # keep the whole pipeline *inside* |= so only the
    # array is sorted/uniqued, not the outer object
    # --------------------------------------------------
    (.permissions.allow |= ((. + $need) | sort | unique))
  ' "$file" > "$tmp" &&
  mv "$tmp" "$file"
}

for f in "${settings_files[@]}"; do
  patch_settings "$f"
done

###############################################################################
# Pick CLI
###############################################################################
if command -v claude-code &>/dev/null; then
  CLAUDE=claude-code; PROMPT_FLAG=--prompt
elif command -v claude &>/dev/null; then
  CLAUDE=claude; PROMPT_FLAG=-p
else
  echo "❌  Claude CLI not found" >&2; exit 1
fi

###############################################################################
# Wrapper: log and continue on non-zero exit (no unsafe retry)
###############################################################################
run_claude() {
  if ! "$CLAUDE" "$@"; then
    echo "⚠️  Claude exited non-zero (possibly max-turns, so we shouldn't quit) - continuing…" >&2
  fi
  return 0               # ← never let a CLI error abort the script
}

###############################################################################
# Core batch runner
###############################################################################
run_task() {
  local prompt_raw=$1 turns_total=$2
  [[ $turns_total =~ ^[0-9]+$ ]] || { echo "Turns must be numeric"; exit 1; }

  # Inject current date into the prompt
  local date_str
  date_str=$(date +%F)
  local prompt="[$date_str] $prompt_raw"

  echo -e "\n🛠️  \"$prompt_raw\" ($turns_total turns)"
  local remain=$turns_total session=""
  while (( remain > 0 )); do
    local t=$(( remain < BATCH_SIZE ? remain : BATCH_SIZE ))
    echo "➡️  Batch $t  (left $remain)"

    # Sanitize prompt for filename and add sub-second timestamp
    local sanitized_prompt
    sanitized_prompt=$(
        printf '%s' "$prompt_raw" \
            | LC_ALL=C tr -cd 'A-Za-z0-9_.\- ' \
            | tr ' ' '_'
    )
    local out_file="$log_dir/$(date +%H%M%S_%3N)_${sanitized_prompt}.json"

    # ──────────────────────────────────────────────────────────────────────
    # If Claude stops because of max-turns, we *expect* a non-zero exit.
    # `|| true` swallows it so the loop keeps ticking.
    # ──────────────────────────────────────────────────────────────────────
    ( run_claude "$PROMPT_FLAG" "$prompt" \
        --max-turns "$t" \
        --output-format stream-json \
        --verbose \
        "${ALLOWED_TOOLS_ARGS[@]}" \
        ${session:+--resume "$session"} ) \
      | tee "$out_file" || true
    # ──────────────────────────────────────────────────────────────────────

    # Extract session-id for resume
    session=$(jq -r '(.sessionId // .session_id // empty)' "$out_file" | tail -1 || true)
    (( remain -= t ))
    (( remain )) && sleep "$COOLDOWN_SECONDS"
  done
}

###############################################################################
# Parse PROMPT/TURNS pairs
###############################################################################
if [[ $# -lt 2 || $(( $# % 2 )) -ne 0 ]]; then
  echo "Usage: $0 \"PROMPT1\" TURNS1 [\"PROMPT2\" TURNS2 ...]" >&2
  exit 1
fi

while (( $# )); do
  run_task "$1" "$2"
  shift 2
  (( $# )) && sleep "$COOLDOWN_BETWEEN_TASKS"
done
