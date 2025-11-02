#!/bin/bash
# Integration phase – exercises core API endpoints against a running instance.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl is required for integration checks"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

API_BASE=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME" || true)
if [ -z "$API_BASE" ]; then
  testing::phase::add_error "Unable to resolve API endpoint for $TESTING_PHASE_SCENARIO_NAME"
  testing::phase::end_with_summary "Integration tests incomplete"
fi

TMP_IMAGE=$(mktemp -t image-tools-integration-XXXXXX.jpg)
cleanup_artifacts() {
  rm -f "$TMP_IMAGE"
}
testing::phase::register_cleanup cleanup_artifacts

echo "/9j/4AAQSkZJRgABAQEAYABgAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/2wBDAQkJCQwLDBgNDRgyIRwhMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjL/wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAr/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFQEBAQAAAAAAAAAAAAAAAAAAAAX/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIRAxEAPwCwAA8A/9k=" | base64 -d > "$TMP_IMAGE"

check_health() {
  set -euo pipefail
  local response
  response=$(curl -sSf "$API_BASE/health")
  grep -q '"status"' <<<"$response" && grep -qi healthy <<<"$response"
}

check_plugins() {
  set -euo pipefail
  local response
  response=$(curl -sSf "$API_BASE/api/v1/plugins")
  grep -q 'jpeg-optimizer' <<<"$response" && grep -q 'png-optimizer' <<<"$response"
}

check_compress() {
  set -euo pipefail
  local status
  status=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE/api/v1/image/compress" \
    -F "image=@$TMP_IMAGE" \
    -F "quality=80")
  [ "$status" = "200" ]
}

check_resize() {
  set -euo pipefail
  local status
  status=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE/api/v1/image/resize" \
    -F "image=@$TMP_IMAGE" \
    -F "width=50" \
    -F "height=50" \
    -F "maintain_aspect=true")
  [ "$status" = "200" ]
}

check_metadata() {
  set -euo pipefail
  local status
  status=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE/api/v1/image/metadata" \
    -F "image=@$TMP_IMAGE" \
    -F "strip=true")
  [ "$status" = "200" ]
}

check_presets() {
  set -euo pipefail
  local response
  response=$(curl -sSf "$API_BASE/api/v1/presets")
  grep -q 'web-optimized' <<<"$response"
}

testing::phase::check "API health endpoint responds" check_health
testing::phase::check "Plugins endpoint lists core plugins" check_plugins
testing::phase::check "Compression endpoint accepts uploads" check_compress
testing::phase::check "Resize endpoint processes requests" check_resize
testing::phase::check "Metadata endpoint strips data" check_metadata
testing::phase::check "Preset catalogue available" check_presets

testing::phase::end_with_summary "Integration validation completed"
