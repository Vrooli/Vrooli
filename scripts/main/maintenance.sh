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
# Config
###############################################################################
BATCH_SIZE=${BATCH_SIZE:-50}
COOLDOWN_SECONDS=${COOLDOWN_SECONDS:-5}
COOLDOWN_BETWEEN_TASKS=${COOLDOWN_BETWEEN_TASKS:-10}
NEEDED_TOOLS=(
  "Bash:*"            # colon form seems to work more reliably than *(*) in 0.2.1xx
  "Edit" "Write" "Glob" "Grep" "LS" "ReadFile"
  "MultiEdit" "Git:*" "WebSearch" "WebFetch"
)
ALLOWED_TOOLS=$(IFS=','; echo "${NEEDED_TOOLS[*]}")

log_dir="logs/$(date +%F)"
mkdir -p "$log_dir"
trap 'echo -e "\n🔴  Aborted"; exit 130' INT

###############################################################################
# Discover all settings files that could block us
###############################################################################
settings_files=()
for f in \
  ".claude/settings.local.json" \
  ".claude/settings.json" \
  "$HOME/.claude/settings.json"
do
  [[ -f $f ]] || continue
  settings_files+=("$f")
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
  local file=$1; shift
  tmp=$(mktemp)
  jq --argjson need "$(printf '%s\n' "${NEEDED_TOOLS[@]}" | jq -R . | jq -s .)" '
      (.permissions //= {}) |
      (.permissions.allow //= []) |
      (.permissions.allow |= (. + $need | unique))
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
# Wrapper that retries once with --dangerously-skip-permissions
###############################################################################
run_claude() {
  local allow_flag=( --allowedTools "$ALLOWED_TOOLS" )
  "$CLAUDE" "${allow_flag[@]}" "$@" || {
    echo "⚠️  CLI ignored --allowedTools, retrying with --dangerously-skip-permissions" >&2
    "$CLAUDE" --dangerously-skip-permissions "$@"
  }
}

###############################################################################
# Core batch runner
###############################################################################
run_task() {
  local prompt=$1 turns_total=$2
  [[ $turns_total =~ ^[0-9]+$ ]] || { echo "Turns must be numeric"; exit 1; }

  echo -e "\n🛠️  \"$prompt\" ($turns_total turns)"
  local remain=$turns_total session=""
  while (( remain > 0 )); do
    local t=$(( remain < BATCH_SIZE ? remain : BATCH_SIZE ))
    echo "➡️  Batch $t  (left $remain)"

    out_file="$log_dir/$(date +%H%M%S)_${prompt// /_}.json"
    run_claude "$PROMPT_FLAG" "$prompt" \
       --max-turns "$t" --output-format stream-json --verbose \
       ${session:+--resume "$session"} \
       | tee "$out_file"

    session=$(grep -oP '"sessionId"\s*:\s*"\K[^"]+' "$out_file" | tail -1 || true)
    (( remain -= t ))
    (( remain )) && { sleep "$COOLDOWN_SECONDS"; }
  done
}

###############################################################################
# Parse PROMPT/TURNS pairs
###############################################################################
[[ $# -ge 2 && $(( $# % 2 )) -eq 0 ]] || {
  echo "Usage: $0 \"PROMPT1\" TURNS1 [\"PROMPT2\" TURNS2 ...]" >&2; exit 1;
}
while (( $# )); do
  run_task "$1" "$2"
  shift 2
  (( $# )) && sleep "$COOLDOWN_BETWEEN_TASKS"
done
