#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "=== Running integration tests ==="
# Integration tests are included in main_test.go with database setup
cd api && go test -v -tags=testing -run "TestUpload|TestAnalyze|TestSearch" -timeout 120s

testing::phase::end_with_summary "Integration tests completed"