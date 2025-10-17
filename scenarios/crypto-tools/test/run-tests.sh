#!/bin/bash

# Master test runner for crypto-tools scenario
# Orchestrates phased testing approach

set -euo pipefail

# Determine app root
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../.." && pwd)}"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Get scenario directory
SCENARIO_DIR="$(cd "${BASH_SOURCE[0]%/*}/.." && pwd)"
SCENARIO_NAME="$(basename "$SCENARIO_DIR")"

log::info "Running test suite for ${SCENARIO_NAME}"

# Default: run all phases
PHASES="${1:-all}"

# Test phase execution order
declare -a TEST_PHASES=(
    "test-structure.sh"
    "test-dependencies.sh"
    "test-unit.sh"
    "test-integration.sh"
    "test-business.sh"
)

# Track results
PASSED=0
FAILED=0
SKIPPED=0

# Execute test phases
for phase in "${TEST_PHASES[@]}"; do
    phase_path="${SCENARIO_DIR}/test/phases/${phase}"

    if [[ ! -x "$phase_path" ]]; then
        log::warn "Phase ${phase} not executable or missing, skipping"
        ((SKIPPED++))
        continue
    fi

    log::info "Running phase: ${phase}"

    if bash "$phase_path"; then
        log::success "✅ Phase ${phase} passed"
        ((PASSED++))
    else
        log::error "❌ Phase ${phase} failed"
        ((FAILED++))
    fi
done

# Summary
log::info ""
log::info "Test Summary:"
log::info "  Passed:  ${PASSED}"
log::info "  Failed:  ${FAILED}"
log::info "  Skipped: ${SKIPPED}"

# Exit with appropriate code
if [[ $FAILED -gt 0 ]]; then
    exit 1
fi

exit 0
