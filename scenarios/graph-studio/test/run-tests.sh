#!/usr/bin/env bash
# Graph Studio Test Runner
# Executes all test phases in order

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Track results
TESTS_PASSED=0
TESTS_FAILED=0
PHASES_PASSED=0
PHASES_FAILED=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

run_phase() {
    local phase_name="$1"
    local phase_script="$2"

    log_info "Running phase: $phase_name"

    if [[ ! -f "$phase_script" ]]; then
        log_warning "Phase script not found: $phase_script"
        return 0
    fi

    if bash "$phase_script"; then
        log_success "Phase passed: $phase_name"
        ((PHASES_PASSED++))
        return 0
    else
        log_error "Phase failed: $phase_name"
        ((PHASES_FAILED++))
        return 1
    fi
}

# Main execution
main() {
    log_info "Starting Graph Studio test suite"
    log_info "Scenario root: $SCENARIO_ROOT"
    echo ""

    # Run test phases in order
    run_phase "Unit Tests" "$SCRIPT_DIR/phases/test-unit.sh" || true
    run_phase "Integration Tests" "$SCRIPT_DIR/phases/test-integration.sh" || true
    run_phase "API Tests" "$SCRIPT_DIR/phases/test-api.sh" || true
    run_phase "CLI Tests" "$SCRIPT_DIR/phases/test-cli.sh" || true
    run_phase "UI Tests" "$SCRIPT_DIR/phases/test-ui.sh" || true

    # Summary
    echo ""
    echo "========================================="
    echo "Test Summary"
    echo "========================================="
    echo -e "Phases Passed: ${GREEN}$PHASES_PASSED${NC}"
    echo -e "Phases Failed: ${RED}$PHASES_FAILED${NC}"
    echo ""

    if [[ $PHASES_FAILED -eq 0 ]]; then
        log_success "All test phases passed!"
        return 0
    else
        log_error "Some test phases failed"
        return 1
    fi
}

main "$@"
