#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

collect_go_module_dirs() {
  local manifest_path root_dir module_dir

  while IFS= read -r manifest_path; do
    root_dir="$(dirname "$(dirname "$manifest_path")")"
    module_dir="$(jq -r '
      select(.cli.enabled == true and .cli.adapter.kind == "go_module")
      | .cli.adapter.module_dir // empty
    ' "$manifest_path" 2>/dev/null || true)"
    [[ -n "$module_dir" ]] || continue
    printf '%s\n' "$root_dir/$module_dir"
  done < <(rg --files scenarios -g '.vrooli/service.json' | sort)

  while IFS= read -r manifest_path; do
    root_dir="$(dirname "$manifest_path")"
    module_dir="$(jq -r '
      select(.cli.enabled == true and .cli.adapter.kind == "go_module")
      | .cli.adapter.module_dir // empty
    ' "$manifest_path" 2>/dev/null || true)"
    [[ -n "$module_dir" ]] || continue
    printf '%s\n' "$root_dir/$module_dir"
  done < <(rg --files resources -g 'resource.json' | sort)
}

mapfile -t modules < <(collect_go_module_dirs | sort -u)

if [ "${#modules[@]}" -eq 0 ]; then
  echo "validate-go-cli-consumers: no manifest-declared go_module CLIs found"
  exit 0
fi

for module_dir in "${modules[@]}"; do
  if [ ! -f "$module_dir/go.mod" ]; then
    echo "validate-go-cli-consumers: missing $module_dir/go.mod for manifest-declared go_module CLI" >&2
    exit 1
  fi
  echo "validate-go-cli-consumers: $module_dir"
  (
    cd "$module_dir"
    GOWORK=off go build ./...
  )
done
