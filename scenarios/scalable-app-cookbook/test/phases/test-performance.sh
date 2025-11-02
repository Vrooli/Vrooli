#!/bin/bash
# Captures lightweight performance baselines for critical endpoints.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s" --require-runtime

scenario_name="$TESTING_PHASE_SCENARIO_NAME"
api_base=""

if ! api_base=$(testing::connectivity::get_api_url "$scenario_name"); then
  testing::phase::add_error "Unable to discover API URL; skipping performance benchmarks"
  testing::phase::end_with_summary "Performance benchmarks incomplete"
fi

if ! command -v python3 >/dev/null 2>&1; then
  testing::phase::add_warning "python3 not available; performance checks skipped"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Performance benchmarks skipped"
fi

performance_check() {
  local label="$1"
  local endpoint="$2"
  local threshold="$3"

  testing::phase::check "$label" python3 - "$api_base$endpoint" "$threshold" <<'PY'
import json
import sys
import time
import urllib.request

url = sys.argv[1]
threshold = float(sys.argv[2])

start = time.time()
with urllib.request.urlopen(url, timeout=15) as response:
    response.read()
duration = time.time() - start

print(f"duration={duration:.3f}s")

sys.exit(0 if duration <= threshold else 1)
PY
}

performance_check "Health responds within 1s" "/health" "1.0"
performance_check "Pattern search responds within 2s" "/api/v1/patterns/search?limit=10" "2.0"
performance_check "Stats endpoint responds within 2s" "/api/v1/patterns/stats" "2.0"

testing::phase::end_with_summary "Performance benchmarks completed"
