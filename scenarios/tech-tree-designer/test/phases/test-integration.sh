#!/bin/bash
# Integration testing phase for tech-tree-designer
# Tests complete workflows and data flow between components

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Run Go integration tests
testing::phase::log "Running Go integration tests..."
cd api
go test -tags testing -run "^TestIntegration_" -v -timeout 60s

if [ $? -ne 0 ]; then
    testing::phase::error "Integration tests failed"
    testing::phase::end_with_summary "Integration tests failed" 1
fi

testing::phase::end_with_summary "Integration tests completed"
