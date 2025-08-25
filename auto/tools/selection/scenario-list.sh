#!/usr/bin/env bash

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
AUTO_DIR="${APP_ROOT}/auto"
BASE_DIR="${APP_ROOT}"
SCEN_DIR="${1:-${BASE_DIR}/scenarios}"

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