#!/usr/bin/env bash

set -euo pipefail

AUTO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
TASK_DATA_DIR="${AUTO_DIR}/data/scenario-improvement"
EVENTS_FILE="${SCENARIO_EVENTS_JSONL:-${TASK_DATA_DIR}/events.ndjson}"
N="${1:-5}"

if [[ ! -f "$EVENTS_FILE" ]]; then echo "[]"; exit 0; fi

if ! command -v jq >/dev/null 2>&1; then
	echo "[]"; exit 0
fi

# Extract recent scenario_choice events and dedupe preserving order
tail -n 50000 "$EVENTS_FILE" \
	| jq -rc 'select(.type=="scenario_choice") | .scenario' \
	| awk '!seen[$0]++' \
	| head -n "$N" \
	| jq -R . \
	| jq -s '.' 