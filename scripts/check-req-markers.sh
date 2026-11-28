#!/usr/bin/env bash
set -euo pipefail

root=${1:-.}
declare -A owners
declare -A files
violations=0

while IFS= read -r -d '' file; do
  if ! rg -q '\[REQ:[^]]+\]' "$file"; then
    continue
  fi
  normalized=$(sed -E 's/[[:space:]]*\/\/ *\[REQ:[^]]+\]//g; s/[[:space:]]+$//' "$file")
  hash=$(printf '%s\n' "$normalized" | sha256sum | cut -d' ' -f1)
  key="$hash"
  owners[$key]="${owners[$key]:-}${owners[$key]:+\n}$file"
  files[$key]="$file"
done < <(find "$root/scenarios" -type d \( -name node_modules -o -name dist -o -name coverage \) -prune -o -type f \( -name '*_test.go' -o -name '*.test.ts' -o -name '*.test.tsx' \) -print0)

for key in "${!owners[@]}"; do
  group=${owners[$key]}
  count=$(printf '%b\n' "$group" | wc -l)
  if (( count > 1 )); then
    echo "duplicate requirement evidence:"
    printf '%b\n' "$group"
    violations=$((violations + 1))
  fi
done

if (( violations > 0 )); then
  echo "$violations invalid duplicate requirement-evidence group(s)" >&2
  exit 1
fi
echo "requirement marker check: 0 violations"
