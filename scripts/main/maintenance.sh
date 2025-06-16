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
# Configurable knobs
###############################################################################
BATCH_SIZE=50       # turns per Claude invocation
COOLDOWN_SECONDS=5  # pause between batches
COOLDOWN_BETWEEN_TASKS=10   # pause between distinct tasks
NEEDED_TOOLS=(
  "Bash(*)" "Write(*)" "Edit" "Glob" "Grep" "LS" "ReadFile" "FileWrite"
  "MultiEdit" "Git(*)" "WebSearch" "WebFetch"
)
ALLOWED_TOOLS=$(IFS=','; echo "${NEEDED_TOOLS[*]}")

###############################################################################
# Basic checks & helpers
###############################################################################
usage() {
  echo "Usage: $0 \"PROMPT_1\" TURNS_1 [\"PROMPT_2\" TURNS_2 ...]" >&2
  exit 1
}

[[ $# -ge 2 && $(( $# % 2 )) -eq 0 ]] || usage

if [[ "$(id -u)" -eq 0 ]]; then
  echo "❌  Refusing to run as root. Please switch to an unprivileged user." >&2
  exit 1
fi

###############################################################################
# Locate or create Claude settings file
###############################################################################
if [[ -f ".claude/settings.json" ]]; then
  SETTINGS_FILE=".claude/settings.json"
else
  SETTINGS_FILE="$HOME/.claude/settings.json"
fi
mkdir -p "$(dirname "$SETTINGS_FILE")"
[[ -s "$SETTINGS_FILE" ]] || echo '{ "permissions": { "allow": [] } }' > "$SETTINGS_FILE"

# Ensure required tools are whitelisted
tmp=$(mktemp)
jq --argjson need "$(printf '%s\n' "${NEEDED_TOOLS[@]}" | jq -R . | jq -s .)" '
  (.permissions //= {}) |
  (.permissions.allow //= []) |
  (.permissions.allow |= (. + $need | unique))
' "$SETTINGS_FILE" > "$tmp" && mv "$tmp" "$SETTINGS_FILE"

###############################################################################
# Pick Claude CLI
###############################################################################
if command -v claude-code >/dev/null 2>&1; then
  CLAUDE=claude-code; PROMPT_FLAG=--prompt
elif command -v claude >/dev/null 2>&1; then
  CLAUDE=claude;      PROMPT_FLAG=-p
else
  echo "❌  Neither 'claude-code' nor 'claude' found on PATH." >&2
  exit 1
fi

###############################################################################
# Core runner
###############################################################################
run_task() {
  local TASK_PROMPT=$1
  local TOTAL_TURNS=$2

  [[ $TOTAL_TURNS =~ ^[0-9]+$ ]] || { echo "Turns must be numeric"; exit 1; }

  echo -e "\n🛠️  Starting task: \"$TASK_PROMPT\" (up to $TOTAL_TURNS turns)"
  local remaining=$TOTAL_TURNS
  local session=""

  while (( remaining > 0 )); do
    local turns=$(( remaining < BATCH_SIZE ? remaining : BATCH_SIZE ))
    echo "➡️  Running batch of $turns turns (remaining: $remaining)…"

    cmd=(
      "$CLAUDE" "$PROMPT_FLAG" "$TASK_PROMPT"
      --max-turns "$turns"
      --verbose
      --output-format stream-json
      --allowedTools "$ALLOWED_TOOLS"
    )
    [[ -n $session ]] && cmd+=( --resume "$session" )

    output="$("${cmd[@]}" 2>&1 | tee /dev/tty)"
    session=$(grep -oP '"sessionId"\s*:\s*"\K[^"]+' <<<"$output" | tail -n1 || true)

    (( remaining -= turns ))
    [[ $remaining -gt 0 ]] && { echo "💤  Cooling down $COOLDOWN_SECONDS s…"; sleep "$COOLDOWN_SECONDS"; }
  done

  echo "✅  Finished task \"$TASK_PROMPT\" (session id: ${session:-<none>})"
}

###############################################################################
# Main loop – iterate through pairs
###############################################################################
while (( $# )); do
  PROMPT=$1; shift
  TURNS=$1; shift
  run_task "$PROMPT" "$TURNS"

  # Optional cool-off between distinct tasks
  (( $# )) && { echo "🌙  Waiting $COOLDOWN_BETWEEN_TASKS s before next task…"; sleep "$COOLDOWN_BETWEEN_TASKS"; }
done
