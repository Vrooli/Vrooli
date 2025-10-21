#!/usr/bin/env bash
set -euo pipefail

# Privacy Terms Generator - Comprehensive Test Runner
# Executes all test phases in order

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SCENARIO_NAME="privacy-terms-generator"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $*"
}

log_success() {
    echo -e "${GREEN}✓${NC} $*"
}

log_error() {
    echo -e "${RED}✗${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $*"
}

# Test phase runner
run_phase() {
    local phase_name=$1
    local phase_script="$SCRIPT_DIR/phases/test-${phase_name}.sh"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if [[ ! -f "$phase_script" ]]; then
        log_warning "Phase script not found: $phase_script"
        SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
        return 0
    fi

    log_info "Running phase: $phase_name"

    if bash "$phase_script"; then
        log_success "Phase $phase_name passed"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log_error "Phase $phase_name failed"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Main test execution
main() {
    log_info "Starting comprehensive test suite for $SCENARIO_NAME"
    echo ""

    # Run test phases in order
    run_phase "structure" || true
    run_phase "dependencies" || true
    run_phase "unit" || true
    run_phase "integration" || true
    run_phase "business" || true
    run_phase "performance" || true

    # Summary
    echo ""
    echo "================================"
    echo "Test Summary"
    echo "================================"
    echo "Total:    $TOTAL_TESTS"
    echo -e "Passed:   ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed:   ${RED}$FAILED_TESTS${NC}"
    echo -e "Skipped:  ${YELLOW}$SKIPPED_TESTS${NC}"
    echo "================================"

    if [[ $FAILED_TESTS -eq 0 ]]; then
        log_success "All available tests passed!"
        exit 0
    else
        log_error "$FAILED_TESTS test(s) failed"
        exit 1
    fi
}

main "$@"
