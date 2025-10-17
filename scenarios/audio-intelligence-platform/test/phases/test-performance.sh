#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "=== Running performance tests ==="
cd api && go test -v -tags=testing -run "Performance|Benchmark" -timeout 180s -short=false

testing::phase::end_with_summary "Performance tests completed"