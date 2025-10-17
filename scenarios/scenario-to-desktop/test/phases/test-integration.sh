#!/bin/bash
# Integration tests for scenario-to-desktop

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "📦 Running integration tests..."

# Test API integration
if [[ -f "api/main_test.go" ]]; then
    echo "🧪 Testing API endpoints integration..."
    cd api
    go test -v -run "TestGenerate|TestBuild|TestPackage" -timeout 60s
    local test_result=$?
    cd ..

    if [[ $test_result -ne 0 ]]; then
        testing::phase::end_with_error "API integration tests failed"
    fi
fi

testing::phase::end_with_summary "Integration tests completed"
