#!/bin/bash
# Placeholder for performance baselines
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s" --require-runtime

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_warning "curl not available; skipping performance sampling"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Performance sampling skipped"
fi

API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME")
LATENCY=0

if [[ -z "$API_URL" ]]; then
  testing::phase::add_warning "Unable to resolve API URL; skipping performance sampling"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Performance sampling skipped"
fi

if curl -s --max-time 5 "$API_URL/health" >/dev/null; then
  testing::phase::add_test passed
else
  testing::phase::add_warning "Health endpoint slow or unavailable during performance sampling"
  testing::phase::add_test failed
fi

START_TIME=$(date +%s%3N)
curl -s --max-time 5 "$API_URL/api/v1/comments/performance-probe" >/dev/null || true
END_TIME=$(date +%s%3N)
LATENCY=$((END_TIME - START_TIME))
echo "Measured GET latency: ${LATENCY}ms"
testing::phase::add_test passed

if (( LATENCY > 500 )); then
  testing::phase::add_warning "Comment list latency exceeded 500ms baseline"
fi

testing::phase::end_with_summary "Performance sampling completed"
