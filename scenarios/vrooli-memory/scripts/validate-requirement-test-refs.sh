#!/usr/bin/env bash
set -euo pipefail

scenario=${1:?usage: validate-requirement-test-refs.sh <scenario>}
script_dir=$(cd "$(dirname "$0")" && pwd)
repo_root=$(cd "$script_dir/../../.." && pwd)
scenario_dir="$repo_root/scenarios/$scenario"

if [[ ! -d "$scenario_dir/requirements" ]]; then
	printf 'requirements directory not found: %s\n' "$scenario_dir/requirements" >&2
	exit 2
fi

failures=0
refs=0
while IFS=$'\t' read -r ref source_requirement; do
	[[ -n "$ref" ]] || continue
	refs=$((refs + 1))
	path=${ref%%::*}
	symbol=${ref##*::}
	file="$scenario_dir/$path"
	if [[ "$ref" != *::* || "$path" == "$symbol" ]]; then
		printf 'INVALID %s: test ref must be path::Symbol\n' "$source_requirement"
		failures=$((failures + 1))
		continue
	fi
	if [[ ! -f "$file" ]]; then
		printf 'MISSING %s: file %s\n' "$source_requirement" "$ref"
		failures=$((failures + 1))
		continue
	fi
	if ! rg -q "^func ${symbol}[[:space:]]*\\(" "$file"; then
		printf 'MISSING %s: symbol %s in %s\n' "$source_requirement" "$symbol" "$file"
		failures=$((failures + 1))
	fi
done < <(find "$scenario_dir/requirements" -name module.json -print0 | xargs -0 -r jq -r --arg scenario "$scenario" '.requirements[] | .id as $id | (.validation // [])[] | select(.type == "test" and (.ref // "") != "") | [.ref, ($scenario + ":" + $id)] | @tsv')

if (( failures > 0 )); then
	printf 'Requirement test-ref validation failed: %d dangling refs out of %d.\n' "$failures" "$refs" >&2
	exit 1
fi
printf 'Requirement test-ref validation passed: %d refs resolved for %s.\n' "$refs" "$scenario"
