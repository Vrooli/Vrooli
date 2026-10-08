#!/usr/bin/env bash
set -euo pipefail

# Capture production landing evidence through one page-bound Playwright
# recording. The old implementation rendered assertions in a separate
# headless Chrome tab and recorded an unrelated X11 display; that can produce
# valid DOM plus black or first-run-dialog video. Historical launch flags are
# not known, so this script does not claim to reproduce them.

base_url="${1:-${LPBS_CAPTURE_BASE_URL:-http://127.0.0.1:23224}}"
output_dir="${2:-${LPBS_CAPTURE_OUTPUT_DIR:-.vrooli/artifacts/lpbs-evidence}}"
variant_slug="${3:-${LPBS_CAPTURE_VARIANT:-control}}"

command -v node >/dev/null || {
  echo "capture failed: required command is missing: node" >&2
  exit 1
}

# Use the already-approved repo-owned Playwright module when it is present.
# This does not install or resolve a package from the network.
repo_root="$(cd "$(dirname "$0")/../../.." && pwd)"
approved_playwright_module="$repo_root/node_modules/.pnpm/playwright@1.62.1/node_modules/playwright/index.mjs"
if [[ -z "${LPBS_PLAYWRIGHT_MODULE:-}" && -f "$approved_playwright_module" ]]; then
  export LPBS_PLAYWRIGHT_MODULE="$approved_playwright_module"
fi

capture_module="$(dirname "$0")/capture-public-evidence.mjs"
args=(
  --base-url "$base_url"
  --output-dir "$output_dir"
  --variant "$variant_slug"
  --route "${LPBS_EXPECTED_ROUTE:-/}"
  --revision "${LPBS_EXPECTED_REVISION:-}"
  --width "${LPBS_CAPTURE_WIDTH:-1440}"
  --height "${LPBS_CAPTURE_HEIGHT:-1000}"
  --dpr "${LPBS_CAPTURE_DPR:-1}"
  --record-ms "${LPBS_CAPTURE_RECORD_MS:-1200}"
  --sample-fps "${LPBS_CAPTURE_SAMPLE_FPS:-4}"
)

if [[ -n "${LPBS_DETAIL_ROUTE:-}" ]]; then
  args+=(--detail-route "$LPBS_DETAIL_ROUTE")
fi

# The module always writes capture-receipt.json, including on failure. Keep
# that receipt and its rejection reasons for diagnosis; do not suppress the
# exit status or delete failed media.
exec node "$capture_module" "${args[@]}"
