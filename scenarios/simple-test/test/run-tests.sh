#!/bin/bash
# Main test runner for simple-test scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCENARIO_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "$var_LOG_FILE"

# Test phases to run
PHASES=(
    "test-dependencies"
    "test-structure"
    "test-unit"
    "test-integration"
    "test-performance"
)

# Track results
TOTAL_PHASES=${#PHASES[@]}
PASSED_PHASES=0
FAILED_PHASES=0

log::info "🧪 Starting simple-test test suite"
log::info "Running $TOTAL_PHASES test phases"
echo ""

# Run each phase
for phase in "${PHASES[@]}"; do
    log::info "Running phase: $phase"

    if bash "$SCENARIO_DIR/test/phases/${phase}.sh"; then
        log::success "✓ Phase passed: $phase"
        PASSED_PHASES=$((PASSED_PHASES + 1))
    else
        log::error "✗ Phase failed: $phase"
        FAILED_PHASES=$((FAILED_PHASES + 1))
    fi

    echo ""
done

# Summary
log::info "📊 Test Summary"
echo "=============================================="
echo "Total Phases:  $TOTAL_PHASES"
echo "Passed:        $PASSED_PHASES"
echo "Failed:        $FAILED_PHASES"
echo ""

if [ $FAILED_PHASES -eq 0 ]; then
    log::success "🎉 All test phases passed!"
    exit 0
else
    log::error "❌ $FAILED_PHASES phase(s) failed"
    exit 1
fi
