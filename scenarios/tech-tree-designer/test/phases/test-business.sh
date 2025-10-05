#!/bin/bash
# Business logic testing phase for tech-tree-designer
# Tests business rules, validations, and domain logic

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Run Go business logic tests
testing::phase::log "Running business logic tests..."
cd api
go test -tags testing -run "^TestBusiness_" -v -timeout 45s

if [ $? -ne 0 ]; then
    testing::phase::error "Business logic tests failed"
    testing::phase::end_with_summary "Business logic tests failed" 1
fi

testing::phase::end_with_summary "Business logic tests completed"
