#!/usr/bin/env bash

# Helper: Print resources list in JSON
# Prefers: vrooli resource list --format json

set -euo pipefail

# Allow override via env
CMD="${RESOURCES_JSON_CMD:-vrooli resource list --format json}"

# 1) Preferred: use CLI if available and returns JSON
if command -v vrooli >/dev/null 2>&1; then
	if OUT=$(timeout 10 bash -c "$CMD" 2>/dev/null) && [[ -n "${OUT:-}" ]] && echo "$OUT" | jq empty >/dev/null 2>&1; then
		printf '%s\n' "$OUT"
		exit 0
	fi
fi

# Resolve repo root
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CONFIG_PATH="${APP_ROOT}/.vrooli/service.json"
RES_DIR="${APP_ROOT}/resources"

# 2) Fallback: derive from service.json (authoritative enable flags)
if [[ -f "$CONFIG_PATH" ]]; then
	jq -r '
	  .resources | to_entries[] as $cat |
	  $cat.value | to_entries[] |
	  {name: .key, category: $cat.key, enabled: (.value.enabled // false), running: false}
	' "$CONFIG_PATH" | jq -s '.'
	exit 0
fi

# 3) Last-resort: scan resources for valid resource dirs (with manage.sh or cli.sh)
if [[ -d "$RES_DIR" ]]; then
	# Find directories that contain manage.sh or cli.sh directly inside
	mapfile -t dirs < <(find "$RES_DIR" -mindepth 2 -maxdepth 2 -type f \( -name 'manage.sh' -o -name 'cli.sh' \) -printf '%h\n' 2>/dev/null | sort -u)
	if ((${#dirs[@]})); then
		# Build minimal JSON with enabled false, running false
		{
			for d in "${dirs[@]}"; do
				name="$(basename "$d")"
				category="$(basename "$(dirname "$d")")"
				printf '{"name":"%s","category":"%s","enabled":false,"running":false}\n' "$name" "$category"
			done
		} | jq -s '.' 2>/dev/null || echo "[]"
		exit 0
	fi
fi

# No data available
echo "[]" 