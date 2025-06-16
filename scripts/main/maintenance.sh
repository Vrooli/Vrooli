#!/usr/bin/env bash
# maintenance-agent.sh ---------------------------------------------------------
# Run long Claude Code maintenance tasks safely, in batches.
#
# USAGE
#   ./maintenance-agent.sh "Do maintenance task TEST_QUALITY in the shared package" 10000
#
# WHAT IT DOES
#   1. Verifies you are *not* root (sudo). Exits with guidance if you are.
#   2. Ensures $HOME/.claude/settings.json (or .claude/settings.json in cwd)
#      contains "Bash(*)" and "Write(*)" in permissions.allow, using jq.
#   3. Executes Claude Code in batches (default 50 turns) with a cooldown.
#      Continues the same session by passing --resume between batches.
#
# DEPENDENCIES
#   • claude or claude-code CLI on PATH
#   • jq • bash • sleep (coreutils)
#
# ------------------------------------------------------------------------------

set -euo pipefail

###############################################################################
# Configurable knobs
###############################################################################
BATCH_SIZE=50       # turns per Claude invocation
COOLDOWN_SECONDS=5  # pause between batches
NEEDED_TOOLS=(
  "Bash(*)"
  "Write(*)"
  "Edit"
  "Glob"
  "Grep"
  "LS"
  "ReadFile"
  "FileWrite"
  "MultiEdit"
  "Git(*)"
  "WebSearch"
  "WebFetch"
)
ALLOWED_TOOLS=$(IFS=',' ; echo "${NEEDED_TOOLS[*]}")


###############################################################################
# Argument parsing
###############################################################################
TASK_PROMPT=${1:-}
TOTAL_TURNS=${2:-}

if [[ -z "${TASK_PROMPT}" || -z "${TOTAL_TURNS}" ]]; then
  echo "Usage: $0 \"<task prompt>\" TOTAL_TURNS" >&2
  exit 1
fi

[[ $TOTAL_TURNS =~ ^[0-9]+$ ]] || { echo "Turns must be numeric"; exit 1; }

###############################################################################
# Root / sudo safety check
###############################################################################
if [[ "$(id -u)" -eq 0 ]]; then
  echo "❌  Refusing to run as root. Please switch to a regular user (e.g. with 'sudo -u matt -i')." >&2
  exit 1
fi

###############################################################################
# Locate settings file (project file preferred, else user file)
###############################################################################
if [[ -f ".claude/settings.json" ]]; then
  SETTINGS_FILE=".claude/settings.json"
else
  SETTINGS_FILE="$HOME/.claude/settings.json"
fi

# Ensure directory exists
mkdir -p "$(dirname "$SETTINGS_FILE")"

###############################################################################
# Bootstrap settings file if missing
###############################################################################
if [[ ! -s "$SETTINGS_FILE" ]]; then
  echo '{ "permissions": { "allow": [] } }' > "$SETTINGS_FILE"
fi

###############################################################################
# Patch allow-list with required tools via jq
###############################################################################
tmp=$(mktemp)
jq --argjson need "$(printf '%s\n' "${NEEDED_TOOLS[@]}" | jq -R . | jq -s .)" '
  (.permissions //= {}) |
  (.permissions.allow //= []) |
  (.permissions.allow |= ( . + $need | unique ))
' "$SETTINGS_FILE" > "$tmp"
mv "$tmp" "$SETTINGS_FILE"

###############################################################################
# Pick Claude CLI & prompt flag
###############################################################################
if command -v claude-code >/dev/null 2>&1; then
  CLAUDE=claude-code; PROMPT_FLAG=--prompt
elif command -v claude >/dev/null 2>&1;   then
  CLAUDE=claude;      PROMPT_FLAG=-p
else
  echo "❌  Neither 'claude-code' nor 'claude' found on PATH." >&2
  exit 1
fi

###############################################################################
# Batched execution loop
###############################################################################
remaining=$TOTAL_TURNS
session=""

while (( remaining > 0 )); do
  turns=$(( remaining < BATCH_SIZE ? remaining : BATCH_SIZE ))
  echo "➡️  Running batch of $turns turns (remaining: $remaining)…"
  cmd=( "$CLAUDE" $PROMPT_FLAG "$TASK_PROMPT"
        --max-turns "$turns"
        --verbose
        --output-format stream-json
        --allowedTools "$ALLOWED_TOOLS"
  )
  [[ -n $session ]] && cmd+=( --resume "$session" )

  # capture & tee so we can scrape the sessionId
  output="$("${cmd[@]}" 2>&1 | tee /dev/tty)"
  session=$(grep -oP '"sessionId"\s*:\s*"\K[^"]+' <<<"$output" | tail -n1 || true)

  (( remaining -= turns ))
  [[ $remaining -gt 0 ]] && { echo "💤  Cooling down for $COOLDOWN_SECONDS s…"; sleep "$COOLDOWN_SECONDS"; }
done

echo "✅  Completed $TOTAL_TURNS turns. Session id: ${session:-<none>}"
