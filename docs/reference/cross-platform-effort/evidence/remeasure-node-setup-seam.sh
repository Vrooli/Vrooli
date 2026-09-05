#!/usr/bin/env bash
set -euo pipefail

# Re-measures only the node-setup seam. It intentionally does not claim a
# repository-wide health result; callers can compare these values with the
# captured baseline record.
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/../../../.." && pwd)"
cd "$repo_root"

count_bindings() {
  local filter="$1"
  local total=0
  local manifest value
  while IFS= read -r manifest; do
    value="$(jq -r "$filter" "$manifest" 2>/dev/null || printf '0')"
    [[ "$value" =~ ^[0-9]+$ ]] || value=0
    total=$((total + value))
  done < <(find scenarios -mindepth 3 -maxdepth 3 -path '*/cli/manifest.json' -type f -print | sort)
  printf '%s' "$total"
}

peer_dir="${VROOLI_PEER_DIR:-${HOME}/.vrooli/peers}"
peer_records=0
if [[ -d "$peer_dir" ]]; then
  peer_records="$(find "$peer_dir" -maxdepth 1 -type f -print | wc -l | tr -d ' ')"
fi

manifest_commands="$(count_bindings '[.. | objects | select(has("binding"))] | length')"
connect_rpc_commands="$(count_bindings '[.. | objects | select(.binding?.kind == "connect-rpc")] | length')"
onboarding_manifest="absent"
[[ -f scenarios/vrooli-onboarding/cli/manifest.json ]] && onboarding_manifest="present"
bridge_rcl="$(rg -l '@vrooli/react-component-library' scenarios/vrooli-bridge/{ui,api} --glob '*.{ts,tsx,go}' 2>/dev/null | wc -l | tr -d ' ')"
onboarding_rcl="$(rg -l '@vrooli/react-component-library' scenarios/vrooli-onboarding --glob '*.{ts,tsx,go}' 2>/dev/null | wc -l | tr -d ' ')"
adapter_assets="$(find scenarios/react-component-library/catalog/assets/adapters -maxdepth 1 -type f -name '*.json' 2>/dev/null | wc -l | tr -d ' ')"
port_lookup="unavailable"
if command -v curl >/dev/null 2>&1; then
  port_lookup="$(curl -fsS --max-time 2 http://127.0.0.1:8092/metrics/processes 2>/dev/null \
    | jq -c '.data.port_lookup // .port_lookup' 2>/dev/null || printf 'unavailable')"
fi

printf 'peer_records=%s\n' "$peer_records"
printf 'manifest_commands=%s\n' "$manifest_commands"
printf 'connect_rpc_commands=%s\n' "$connect_rpc_commands"
printf 'onboarding_manifest=%s\n' "$onboarding_manifest"
printf 'bridge_rcl_imports=%s\n' "$bridge_rcl"
printf 'onboarding_rcl_imports=%s\n' "$onboarding_rcl"
printf 'catalog_adapter_assets=%s\n' "$adapter_assets"
printf 'port_lookup=%s\n' "$port_lookup"
