#!/bin/bash
# Business logic test phase
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running business logic tests"

# Run business tests with Jest
if npm test -- --testPathPattern=business 2>&1; then
    log::success "Business logic tests passed"
else
    testing::phase::add_error "Business logic tests failed"
fi

# End with summary
testing::phase::end_with_summary "Business logic tests completed"
