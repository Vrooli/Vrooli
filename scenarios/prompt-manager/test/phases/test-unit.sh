#!/bin/bash
# Unit test phase - uses centralized testing library
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 60-second target
testing::phase::init --target-time "60s"

# Source centralized testing library
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# Run unit tests with standardized parameters
# Note: Low coverage threshold (10%) is intentional for this database-heavy scenario
# Most handlers require database connectivity and are covered by integration tests
# Database-independent code (models, helpers, validation) achieves 100% coverage
if testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 15 \
    --coverage-error 10; then
    log::success "All unit tests passed"
else
    testing::phase::add_error "Some unit tests failed"
fi

# End with summary
testing::phase::end_with_summary "Unit tests completed"
