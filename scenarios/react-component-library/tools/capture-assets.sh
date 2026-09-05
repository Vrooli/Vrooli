#!/usr/bin/env bash
set -euo pipefail

# Repeatable visual capture instrument for the bounded preview inventory.
# Usage: capture-assets.sh [asset ...]
# Output: coverage/visual/baseline/<asset>/<viewport>-<theme>.jpg
# The capture path is intentionally bounded: readiness is a DOM signal and
# matrix entries run through a small worker pool. Set RCL_CHANGED_ONLY=1 to
# skip assets whose catalog/source revision is unchanged since the last run.

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
wait_for="${RCL_CAPTURE_WAIT_FOR:-#root[data-experience-state=ready][data-rcl-theme] }"
wait_for="${wait_for% }"
workers="${RCL_CAPTURE_WORKERS:-4}"
if ! [[ "${workers}" =~ ^[1-9][0-9]*$ ]]; then
  echo "RCL_CAPTURE_WORKERS must be a positive integer" >&2
  exit 1
fi
changed_only="${RCL_CHANGED_ONLY:-0}"
revision_file="${RCL_CAPTURE_REVISION_FILE:-${output_root}/.revisions.json}"
component_index="$(mktemp)"
trap 'rm -f "${component_index}"' EXIT
curl --fail --silent --show-error \
  -X POST "http://127.0.0.1:${api_port}/vrooli.react_component_library.v1.components.ComponentsService/ListComponents" \
  -H 'content-type: application/json' \
  -d '{"limit":500}' >"${component_index}"

declare -A revisions=()
if [[ -f "${revision_file}" ]]; then
  while IFS=$'\t' read -r asset revision; do
    [[ -n "${asset}" && -n "${revision}" ]] && revisions["${asset}"]="${revision}"
  done < <(jq -r '.[] | [.asset, .revision] | @tsv' "${revision_file}" 2>/dev/null || true)
fi

declare -A next_revisions=()
for asset in "${!revisions[@]}"; do next_revisions["${asset}"]="${revisions[${asset}]}"; done
for asset in "${assets[@]}"; do
  component_id="$(jq -r --arg catalog_id "${asset}" '.components[] | select(.catalogId == $catalog_id) | .id' "${component_index}" | head -1)"
  if [[ -z "${component_id}" || "${component_id}" == "null" ]]; then
    echo "catalog asset is not indexed: ${asset}" >&2
    exit 1
  fi
  source_path="$(jq -r --arg catalog_id "${asset}" '.components[] | select(.catalogId == $catalog_id) | .sourcePath' "${component_index}" | head -1)"
  manifest_path="$(jq -r --arg catalog_id "${asset}" '.components[] | select(.catalogId == $catalog_id) | .manifestPath' "${component_index}" | head -1)"
  catalog_path=""
  while IFS= read -r candidate; do
    if jq -e --arg id "${asset}" '.asset.id == $id' "${candidate}" >/dev/null 2>&1; then
      catalog_path="${candidate}"
      break
    fi
  done < <(find "${scenario_root}/catalog/assets" -type f -name '*.json' -print)
  if [[ -z "${catalog_path}" ]]; then
    echo "catalog asset declaration is missing: ${asset}" >&2
    exit 1
  fi
  source_path="${scenario_root}/library/${source_path}"
  manifest_path="${scenario_root}/library/${manifest_path}"
  revision="$(cat "${catalog_path}" "${manifest_path}" "${source_path}" | sha256sum | awk '{print $1}')"
  next_revisions["${asset}"]="${revision}"
  if [[ "${changed_only}" == "1" && "${revisions[${asset}]:-}" == "${revision}" ]]; then
    echo "skip unchanged asset=${asset} revision=${revision}" >&2
    continue
  fi
  pids=()
  running=0
  for item in "${matrix[@]}"; do
    IFS=: read -r viewport width height <<<"${item}"
    for theme in light dark; do
      destination="${output_root}/${asset}"
      mkdir -p "${destination}"
      echo "capture asset=${asset} viewport=${viewport} theme=${theme}" >&2
      (
      browser-automation-studio capture \
        --url "http://127.0.0.1:${api_port}/preview/${component_id}/harness.html?theme=${theme}&motion=reduce&seed=rcl-calibration-v1&kit=vrooli-default" \
        --capture screenshot \
        --width "${width}" \
        --height "${height}" \
        --wait-for "${wait_for}" \
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
      ) &
      pids+=("$!")
      ((running+=1))
      if (( running >= workers )); then
        for pid in "${pids[@]}"; do wait "${pid}"; done
        pids=()
        running=0
      fi
    done
  done
  for pid in "${pids[@]}"; do wait "${pid}"; done
done

mkdir -p "$(dirname "${revision_file}")"
{
  printf '[\n'
  first=1
  for asset in "${!next_revisions[@]}"; do
    [[ "${first}" == 0 ]] && printf ',\n'
    first=0
    printf '  %s\n' "$(jq -cn --arg asset "${asset}" --arg revision "${next_revisions[${asset}]}" '{asset:$asset,revision:$revision}')"
  done
  printf ']\n'
} >"${revision_file}"
