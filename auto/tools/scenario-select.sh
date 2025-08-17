#!/usr/bin/env bash

set -euo pipefail

SCENARIO="${1:-}"
if [[ -z "$SCENARIO" ]]; then echo "Usage: $0 <scenario-name>" >&2; exit 1; fi

AUTO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
DATA_DIR="${AUTO_DIR}/data/scenario-improvement"
EVENTS_FILE="${SCENARIO_EVENTS_JSONL:-${DATA_DIR}/events.ndjson}"
mkdir -p "$(dirname "$EVENTS_FILE")" 2>/dev/null || true

TS=$(date -Is)
EVENT=$(printf '{"type":"scenario_choice","ts":"%s","scenario":"%s"}' "$TS" "$SCENARIO")

# Append atomically if possible
if command -v flock >/dev/null 2>&1; then
	{
		flock -x 200
		printf '%s\n' "$EVENT" >&200
	} 200>>"$EVENTS_FILE"
else
	printf '%s\n' "$EVENT" >> "$EVENTS_FILE"
fi

echo "recorded: $SCENARIO" 