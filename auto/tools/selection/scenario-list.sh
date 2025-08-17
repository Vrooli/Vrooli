#!/usr/bin/env bash

set -euo pipefail

AUTO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
BASE_DIR="$(cd "$AUTO_DIR"/.. && pwd)"
SCEN_DIR="${1:-${BASE_DIR}/scripts/scenarios/core}"

if [[ ! -d "$SCEN_DIR" ]]; then
	echo "[]"
	exit 0
fi

mapfile -t scenarios < <(find "$SCEN_DIR" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort)

# Output JSON array of names, robustly escaped if jq exists
if command -v jq >/dev/null 2>&1; then
	printf '%s\n' "${scenarios[@]}" | jq -R . | jq -s '.'
else
	{
		printf "["
		for ((i=0; i<${#scenarios[@]}; i++)); do
			name="${scenarios[$i]}"
			printf '"%s"' "$name"
			if (( i < ${#scenarios[@]}-1 )); then printf ","; fi
		done
		printf "]\n"
	}
fi 