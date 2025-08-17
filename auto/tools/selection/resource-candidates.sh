#!/usr/bin/env bash

# Helper: Select candidate resources for improvement
# Input: (optional) path to JSON file or '-' for stdin; otherwise uses resource-list.sh
# Output: newline-separated resource names ordered by priority

set -euo pipefail

# Load input JSON
INPUT_ARG="${1:-}"
if [[ -n "${INPUT_ARG}" && -f "${INPUT_ARG}" ]]; then
	INPUT_JSON="$(cat "${INPUT_ARG}")"
elif [[ "${INPUT_ARG:-}" == "-" || "${INPUT_ARG:-}" == "/dev/stdin" ]]; then
	INPUT_JSON="$(cat)"
else
	INPUT_JSON=$("$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/resource-list.sh" 2>/dev/null || echo '[]')
fi

# Validate JSON; fall back to empty array if invalid
if ! echo "$INPUT_JSON" | jq empty >/dev/null 2>&1; then
	INPUT_JSON='[]'
fi

# Normalize to array of {name, enabled, running, status}
NORMALIZED=$(echo "$INPUT_JSON" | jq -c '
	if type=="object" and has("resources") then
	  (.resources | to_entries[] as $cat | $cat.value | to_entries[] | {name: .key, enabled: (.value.enabled // false), running: false, status: ("unknown")} )
	else
	  .
	end
')

# Optionally merge live runtime status if available
# Expecting JSON array of {name, running, status}
if command -v vrooli >/dev/null 2>&1; then
	if LIVE=$(timeout 10 vrooli resource status --format json 2>/dev/null) && [[ -n "${LIVE:-}" ]] && echo "$LIVE" | jq empty >/dev/null 2>&1; then
		NORMALIZED=$(jq -cn --argjson base "$NORMALIZED" --argjson live "$LIVE" '
			# index live by name
			def toIndex:
			  reduce .[] as $i ({}; .[$i.name] = {running: ($i.running // false), status: ($i.status // (if ($i.running // false) then "running" else "unknown" end))});
			($live | (if type=="array" then . else [] end) | toIndex) as $L |
			($base | (if type=="array" then . else [] end) | map(
				if .name and ($L[.name]) then .running = ($L[.name].running // .running) | .status = ($L[.name].status // .status) else . end
			))
		')
	fi
fi

# Build priority buckets and preserve order with stable de-duplication
# 1) enabled && not running
# 2) enabled && running
# 3) disabled (or missing enabled)

echo "$NORMALIZED" | jq -r '
	def dedup:
	  reduce .[] as $x ([]; if index($x) then . else . + [$x] end);
	
	(if type=="array" then . else . end) as $all |
	($all | map(select(.name? != null))) as $r |
	(
	  [ $r[] | select((.enabled == true) and ((.running == false) or (.status == "stopped") or (.status == "not_running"))) | .name ] +
	  [ $r[] | select((.enabled == true) and ((.running == true) or (.status == "running"))) | .name ] +
	  [ $r[] | select(((.enabled // false) == false)) | .name ]
	) | dedup[]
' 2>/dev/null || true 