#!/usr/bin/env bash
# maintenance-agent.sh — Run long-running Claude Code maintenance tasks in batches
#
# Usage:
#   ./maintenance-agent.sh \
#     "Do maintenance task TEST_QUALITY in the shared package" \
#     TOTAL_TURNS
#
# Example:
#   ./maintenance-agent.sh \
#     "Do maintenance task SECURITY_AUDIT in shared package" \
#     1000
#
# This will:
#   - Refuse to run as root (for security)
#   - Run Claude Code in non-interactive, streaming mode
#   - Process turns in batches (default 50), with a brief cooldown
#   - Grant permissions for Bash and Write tools using --allowedTools
#     (required even as root) :contentReference[oaicite:1]{index=1}
#   - Use --resume to continue the session across batches :contentReference[oaicite:2]{index=2}
#
# Adjust BATCH_SIZE and COOLDOWN_SECONDS as needed.

set -euo pipefail

TASK=${1:-}
TOTAL_TURNS=${2:-}
BATCH_SIZE=50
COOLDOWN_SECONDS=5

if [[ -z $TASK || -z $TOTAL_TURNS ]]; then
  echo "Usage: $0 \"<task prompt>\" TOTAL_TURNS" >&2
  exit 1
fi

if [[ $(id -u) -eq 0 ]]; then
  echo "❌ Refusing to run as root. Please switch to a regular user." >&2
  exit 1
fi

remaining=$TOTAL_TURNS
session_id=""

while (( remaining > 0 )); do
  turns=$((remaining < BATCH_SIZE ? remaining : BATCH_SIZE))
  echo "➡️ Executing batch of $turns turns (remaining: $remaining)..."

  # Construct command
  cmd=(claude -p "$TASK"
       --max-turns "$turns"
       --verbose
       --output-format stream-json
       --allowedTools "Bash(*),Write"
  )

  # Resume session if present
  if [[ -n $session_id ]]; then
    cmd+=(--resume "$session_id")
  fi

  # Run Claude Code batch
  session_output="$("${cmd[@]}" 2>&1 | tee /dev/tty)"
  # Extract new session-id if emitted (JSON field "sessionId":)
  new_session_id=$(grep -oP '"sessionId"\s*:\s*"\K[^"]+' <<<"$session_output" | tail -n1 || true)
  if [[ -n $new_session_id ]]; then
    session_id=$new_session_id
  fi

  ((remaining -= turns))
  echo "💤 Cooling down for $COOLDOWN_SECONDS seconds..."
  sleep $COOLDOWN_SECONDS
done

echo "✅ Completed $TOTAL_TURNS turns. Final session ID: $session_id"
