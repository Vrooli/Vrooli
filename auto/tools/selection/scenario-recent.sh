#!/usr/bin/env bash

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
AUTO_DIR="${APP_ROOT}/auto"
TASK_DATA_DIR="${AUTO_DIR}/data/scenario-improvement"
EVENTS_FILE="${SCENARIO_EVENTS_JSONL:-${TASK_DATA_DIR}/events.ndjson}"
N="${1:-5}"

if [[ ! -f "$EVENTS_FILE" ]]; then echo "[]"; exit 0; fi

if ! command -v jq >/dev/null 2>&1; then
	echo "[]"; exit 0
fi

# Extract recent scenario_choice events and dedupe preserving order
# Guard tail and jq failures
set +e
TAIL_OUT=$(tail -n 50000 "$EVENTS_FILE" 2>/dev/null)
if [[ -z "${TAIL_OUT:-}" ]]; then echo "[]"; exit 0; fi
printf '%s\n' "$TAIL_OUT" \
	| jq -rc 'select(.type=="scenario_choice") | .scenario' 2>/dev/null \
	| awk '!seen[$0]++' \
	| head -n "$N" \
	| jq -R . \
	| jq -s '.'
exit 0 