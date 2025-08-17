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

# Normalize the JSON structure to handle both formats
# Current format: array with Name, Enabled, Running fields
# Legacy format: nested resources object
NORMALIZED=$(echo "$INPUT_JSON" | jq -c '
	if type=="object" and has("resources") then
	  # Legacy nested format: .resources.category.resource_name
	  (.resources | to_entries[] as $cat | $cat.value | to_entries[] | {name: .key, enabled: (.value.enabled // false), running: false, status: ("unknown")} )
	elif type=="array" then
	  # Current format: array of objects with Name, Enabled, Running fields
	  map({
		name: (.Name // .name // ""),
		enabled: (.Enabled // .enabled // false),
		running: (.Running // .running // false),
		status: (.Status // .status // (if (.Running // .running // false) then "running" else "unknown" end))
	  } | select(.name != ""))
	else
	  []
	end
')

# Merge live runtime status if available
# Use robust error handling to prevent jq parsing issues
if command -v vrooli >/dev/null 2>&1; then
	# Try to get live status with timeout and validation
	if LIVE_JSON=$(timeout 10 vrooli resource status --format json 2>/dev/null) && \
	   [[ -n "${LIVE_JSON:-}" ]] && \
	   echo "$LIVE_JSON" | jq empty >/dev/null 2>&1; then
		
		# Normalize live status to match our format
		LIVE_NORMALIZED=$(echo "$LIVE_JSON" | jq -c '
			map({
				name: (.Name // .name // ""),
				enabled: (.Enabled // .enabled // false),
				running: (.Running // .running // false),
				status: (.Status // .status // (if (.Running // .running // false) then "running" else "unknown" end))
			} | select(.name != ""))
		')
		
		# Merge live status with base data using a simple, robust approach
		if [[ -n "${LIVE_NORMALIZED:-}" ]] && echo "$LIVE_NORMALIZED" | jq empty >/dev/null 2>&1; then
			NORMALIZED=$(echo "$NORMALIZED" | jq -c --argjson live "$LIVE_NORMALIZED" '
				# Create lookup table from live data
				def live_lookup:
					reduce .[] as $item ({}; .[$item.name] = $item);
				
				($live | live_lookup) as $live_map |
				map(
					if .name and $live_map[.name] then
						# Update with live data, preserving original enabled status
						. + {
							running: ($live_map[.name].running // .running),
							status: ($live_map[.name].status // .status)
						}
					else
						.
					end
				)
			' 2>/dev/null || echo "$NORMALIZED")
		fi
	fi
fi

# Process the normalized data to build priority buckets
# Use separate jq commands for each priority level to avoid complex expressions
echo "$NORMALIZED" | jq -r 'map(select(.enabled == true and (.running == false or .status == "stopped" or .status == "not_running"))) | .[].name' 2>/dev/null || true
echo "$NORMALIZED" | jq -r 'map(select(.enabled == true and (.running == true or .status == "running"))) | .[].name' 2>/dev/null || true
echo "$NORMALIZED" | jq -r 'map(select(.enabled == false)) | .[].name' 2>/dev/null || true
