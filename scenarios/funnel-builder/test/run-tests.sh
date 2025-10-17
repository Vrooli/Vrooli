#!/bin/bash
# Test orchestrator for funnel-builder scenario
# Runs all test phases in sequence

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
SCENARIO_NAME="funnel-builder"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[1;34m'
NC='\033[0m'

# Test results tracking
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

run_test_phase() {
    local phase_name="$1"
    local phase_script="$2"

    if [ ! -f "$phase_script" ]; then
        log_warning "Phase script not found: $phase_script (skipping)"
        return 0
    fi

    if [ ! -x "$phase_script" ]; then
        chmod +x "$phase_script"
    fi

    echo ""
    log_info "Running $phase_name..."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    if bash "$phase_script"; then
        log_success "$phase_name passed"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        log_error "$phase_name failed"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi

    TESTS_RUN=$((TESTS_RUN + 1))
}

# Header
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  🧪 Funnel Builder Test Suite             ║${NC}"
echo -e "${BLUE}╔════════════════════════════════════════════╗${NC}"
echo ""

# Check if scenario is running
log_info "Checking scenario status..."
if ! vrooli scenario status "$SCENARIO_NAME" &>/dev/null; then
    log_warning "Scenario is not running. Some tests may fail."
    echo "Run 'make run' to start the scenario before testing."
fi

# Run test phases in order
run_test_phase "Unit Tests" "$SCRIPT_DIR/phases/test-unit.sh"
run_test_phase "Integration Tests" "$SCRIPT_DIR/phases/test-integration.sh"
run_test_phase "API Tests" "$SCRIPT_DIR/phases/test-api.sh"
run_test_phase "CLI Tests" "$SCRIPT_DIR/phases/test-cli.sh"

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${BLUE}📊 Test Summary${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Total phases: $TESTS_RUN"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
fi
echo ""

# Exit with appropriate code
if [ $TESTS_FAILED -gt 0 ]; then
    log_error "Some tests failed"
    exit 1
else
    log_success "All tests passed!"
    exit 0
fi
