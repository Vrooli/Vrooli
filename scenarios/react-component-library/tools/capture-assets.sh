#!/usr/bin/env bash
set -euo pipefail

# Repeatable visual capture instrument for the bounded preview inventory.
# Usage: capture-assets.sh [asset ...]
# Output: coverage/visual/baseline/<asset>/<viewport>-<theme>.jpg

scenario_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output_root="${RCL_CAPTURE_OUTPUT_ROOT:-${scenario_root}/coverage/visual/baseline}"
assets=(
  data-display.code-block
  data-display.data-table
  data-display.description-list
  data-display.tree-view
  forms.input
  forms.select
  navigation.page
  overlays.dialog
  overlays.inspector-panel
  overlays.popover
  manipulation.split-pane
  navigation.master-detail
  templates.dashboard-page
  preview.preview-dock
  preview.story-palette
  preview.inspector-drawer
  preview.canvas-frame
)

if (( $# > 0 )); then
  assets=("$@")
fi

matrix=(
  "mobile:390:844"
  "tablet:834:1112"
  "wide:1440:900"
  "ultrawide:2560:1440"
)

api_port="${API_PORT:-17193}"
component_index="$(mktemp)"
trap 'rm -f "${component_index}"' EXIT
curl --fail --silent --show-error \
  -X POST "http://127.0.0.1:${api_port}/vrooli.react_component_library.v1.components.ComponentsService/ListComponents" \
  -H 'content-type: application/json' \
  -d '{"limit":500}' >"${component_index}"

for asset in "${assets[@]}"; do
  component_id="$(jq -r --arg catalog_id "${asset}" '.components[] | select(.catalogId == $catalog_id) | .id' "${component_index}" | head -1)"
  if [[ -z "${component_id}" || "${component_id}" == "null" ]]; then
    echo "catalog asset is not indexed: ${asset}" >&2
    exit 1
  fi
  for item in "${matrix[@]}"; do
    IFS=: read -r viewport width height <<<"${item}"
    for theme in light dark; do
      destination="${output_root}/${asset}"
      mkdir -p "${destination}"
      echo "capture asset=${asset} viewport=${viewport} theme=${theme}" >&2
      browser-automation-studio capture \
        --url "scenario=react-component-library,path=/assets/${component_id}?theme=${theme}" \
        --capture screenshot \
        --width "${width}" \
        --height "${height}" \
        --wait-for 6000 \
        --label "${asset} ${viewport} ${theme}" \
        --out "${destination}" \
        --json \
        >"${destination}/${viewport}-${theme}.json"
      artifact="$(jq -r '.primary_artifact_path // empty' "${destination}/${viewport}-${theme}.json")"
      if [[ -z "${artifact}" ]]; then
        echo "capture produced no screenshot for ${asset} ${viewport} ${theme}" >&2
        exit 1
      fi
      cp "${artifact}" "${destination}/${viewport}-${theme}.jpg"
    done
  done
done
