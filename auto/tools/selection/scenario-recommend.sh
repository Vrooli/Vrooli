#!/usr/bin/env bash

set -euo pipefail

AUTO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
BASE_DIR="$(cd "$AUTO_DIR"/.. && pwd)"
TOOLS_DIR="${AUTO_DIR}/tools/selection"
DATA_DIR="${AUTO_DIR}/data/scenario-improvement"
EVENTS_FILE="${SCENARIO_EVENTS_JSONL:-${DATA_DIR}/events.ndjson}"
MODEL="${OLLAMA_SUMMARY_MODEL:-llama3.2:3b}"
K="${K:-5}"
GOALS="${GOALS:-}" # comma-separated
COOLDOWN_SIZE="${MAX_CONCURRENT_WORKERS:-5}"

# Get full scenario list
SCENARIOS_JSON=$("${TOOLS_DIR}/scenario-list.sh")

# Get recent unique choices (up to cooldown size)
RECENT_JSON=$("${TOOLS_DIR}/scenario-recent.sh" "$COOLDOWN_SIZE" 2>/dev/null || echo '[]')

# Build filtered list via jq if available
FILTERED="$SCENARIOS_JSON"
if command -v jq >/dev/null 2>&1; then
	if ! FILTERED=$(jq -c --argjson all "$SCENARIOS_JSON" --argjson recent "$RECENT_JSON" '($all - $recent)' <<<"{}" 2>/dev/null); then
		FILTERED="$SCENARIOS_JSON"
	fi
fi

# If Ollama is available, ask it to rank
if command -v resource-ollama >/dev/null 2>&1; then
	PROMPT=$(mktemp)
	{
		printf "You are helping select scenarios to improve next.\n"
		printf "Goals (optional): %s\n" "$GOALS"
		printf "Exclude recently picked: %s\n" "$RECENT_JSON"
		printf "Candidates (JSON array of names): %s\n" "$FILTERED"
		printf "Return a JSON array of up to %d scenario names in order of priority, no explanations.\n" "$K"
	} > "$PROMPT"
	OUT=$(resource-ollama generate "$MODEL" "$(cat "$PROMPT")" 2>/dev/null || echo "[]")
	rm -f "$PROMPT"
	# Try to extract JSON array
	if command -v jq >/dev/null 2>&1 && echo "$OUT" | jq -e . >/dev/null 2>&1; then
		echo "$OUT" | jq -c .
		exit 0
	fi
fi

# Fallback: random sample from filtered
if command -v shuf >/dev/null 2>&1; then
	echo "$FILTERED" | jq -r '.[]' 2>/dev/null | shuf | head -n "$K" | jq -R . | jq -s '.' 2>/dev/null || echo "$FILTERED"
else
	echo "$FILTERED" | (command -v jq >/dev/null 2>&1 && jq -c '.[:'"$K"']' || cat)
fi 