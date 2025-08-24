#!/usr/bin/env bash

set -euo pipefail

SCENARIO="${1:-}"
if [[ -z "$SCENARIO" ]]; then echo "Usage: $0 <scenario-name>" >&2; exit 1; fi

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
AUTO_DIR="${APP_ROOT}/auto"
BASE_DIR="${APP_ROOT}"
SCEN_DIR="${BASE_DIR}/scripts/scenarios/core"
DATA_DIR="${AUTO_DIR}/data/scenario-improvement"
EVENTS_FILE="${SCENARIO_EVENTS_JSONL:-${DATA_DIR}/events.ndjson}"
mkdir -p "$(dirname "$EVENTS_FILE")" 2>/dev/null || true

# Validate scenario exists
if [[ ! -d "${SCEN_DIR}/${SCENARIO}" ]]; then
	echo "ERROR: Scenario not found: ${SCENARIO}" >&2
	exit 2
fi

TS=$(date -Is)
if command -v jq >/dev/null 2>&1; then
	EVENT=$(jq -c -n --arg ts "$TS" --arg s "$SCENARIO" '{type:"scenario_choice", ts:$ts, scenario:$s}')
else
	# Fallback: naive JSON (may break with special chars)
	EVENT=$(printf '{"type":"scenario_choice","ts":"%s","scenario":"%s"}' "$TS" "$SCENARIO")
fi

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