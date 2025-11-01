#!/bin/bash
# Performance test phase
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR/api"

log::info "Running performance tests..."

# Run performance tests with verbose output
if go test -v -run "Performance" -timeout 120s 2>&1 | tee /dev/stderr; then
    log::success "Performance tests passed"
    testing::phase::add_test passed
else
    testing::phase::add_error "Performance tests failed"
    testing::phase::add_test failed
fi

# End with summary
testing::phase::end_with_summary "Performance tests completed"
