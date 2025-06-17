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
# Global time-out defaults (override with env if desired)
###############################################################################
DEFAULT_TIMEOUT_MS=${DEFAULT_TIMEOUT_MS:-600000}   # 10 min
MAX_TIMEOUT_MS=${MAX_TIMEOUT_MS:-600000}           # 10 min
MCP_TIMEOUT_MS=${MCP_TIMEOUT_MS:-600000}           # 10 min

export BASH_DEFAULT_TIMEOUT_MS="$DEFAULT_TIMEOUT_MS"
export BASH_MAX_TIMEOUT_MS="$MAX_TIMEOUT_MS"
export MCP_TOOL_TIMEOUT="$MCP_TIMEOUT_MS"

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
  local file=$1 tmp; tmp=$(mktemp)

  jq  --argjson need "$(printf '%s\n' "${NEEDED_TOOLS[@]}" | jq -R . | jq -s '.')" \
      --arg defTm "$BASH_DEFAULT_TIMEOUT_MS" \
      --arg maxTm "$BASH_MAX_TIMEOUT_MS" \
      --arg mcpTm "$MCP_TOOL_TIMEOUT" '
    # ----- permissions block (existing logic) -----
    (.permissions //= {}) |
    (.permissions.allow //= []) |
    (.permissions.allow |= ((. + $need) | sort | unique)) |
    # ----- env block for timeout overrides --------
    (.env //= {}) |
    (.env.BASH_DEFAULT_TIMEOUT_MS = $defTm) |
    (.env.BASH_MAX_TIMEOUT_MS     = $maxTm) |
    (.env.MCP_TOOL_TIMEOUT        = $mcpTm)
  ' "$file" > "$tmp" && mv "$tmp" "$file"
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
# Core batch runner
###############################################################################
run_task() {
  local prompt_raw=$1 turns_total=$2
  [[ $turns_total =~ ^[0-9]+$ ]] || { echo "Turns must be numeric"; exit 1; }

  local date_str; date_str=$(date +%F)
  local prompt="[$date_str] $prompt_raw"

  echo -e "\n🛠️  \"$prompt_raw\" ($turns_total turns)"
  local remain=$turns_total session=""

  while (( remain > 0 )); do
    local t=$(( remain < BATCH_SIZE ? remain : BATCH_SIZE ))
    echo "➡️  Batch $t  (left $remain)"

    local sanitized_prompt
    sanitized_prompt=$(printf '%s' "$prompt_raw" | LC_ALL=C tr -cd 'A-Za-z0-9_.\- ' | tr ' ' '_')
    local out_file="$log_dir/$(date +%H%M%S_%3N)_${sanitized_prompt}.json"

    # ─────────────────────── call Claude & capture exit code ──────────────────
    set +e
    "$CLAUDE" "$PROMPT_FLAG" "$prompt" \
      --max-turns "$t" \
      --output-format stream-json \
      --verbose \
      "${ALLOWED_TOOLS_ARGS[@]}" \
      ${session:+--resume "$session"} | tee "$out_file"
    rc=${PIPESTATUS[0]}   # exit of Claude (not tee)
    set -e
    # ──────────────────────────────────────────────────────────────────────────

    # ◆ Error handling / success & max-turn detection
    if (( rc != 0 )); then
      if [[ ! -s "$out_file" ]]; then
        echo "❌  Claude CLI failed (exit $rc) and no log was written" >&2
        exit $rc
      fi

      # Pull both subtype and is_error for robustness
      read -r last_subtype last_is_error < <(
        tail -n 1 "$out_file" \
          | jq -r '[.subtype // ""]; (.is_error // false)' 2>/dev/null
      )

      # Continue if the run ended cleanly
      if [[ $last_subtype == "error_max_turns" \
            || $last_subtype == "success" \
            || $last_is_error == "false" ]]; then
        echo "ℹ️  Claude finished cleanly (${last_subtype:-no-subtype}); continuing…" >&2
      else
        echo "❌  Claude CLI failed (exit $rc); last subtype: '$last_subtype'" >&2
        exit $rc
      fi
    fi

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
