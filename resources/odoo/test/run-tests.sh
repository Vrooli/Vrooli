#!/usr/bin/env bash
################################################################################
# Odoo Resource - Main Test Runner
# Orchestrates all test phases according to v2.0 universal contract
################################################################################

set -euo pipefail

# Determine paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source common utilities
# Note: test.sh might not exist, so we'll handle that gracefully
if [[ -f "$APP_ROOT/scripts/lib/utils/test.sh" ]]; then
    source "$APP_ROOT/scripts/lib/utils/test.sh"
fi
source "$APP_ROOT/scripts/lib/utils/var.sh"
source "$APP_ROOT/scripts/lib/utils/log.sh"

# Test configuration
readonly TEST_TIMEOUT_SMOKE=30
readonly TEST_TIMEOUT_INTEGRATION=120
readonly TEST_TIMEOUT_UNIT=60
readonly TEST_TIMEOUT_ALL=300

# Initialize test results
TEST_RESULTS=()
TOTAL_TESTS=0
PASSED_TESTS=0

################################################################################
# Run specific test phase
################################################################################
run_test_phase() {
    local phase="$1"
    local timeout="${2:-60}"
    local test_script="$SCRIPT_DIR/phases/test-${phase}.sh"
    
    if [[ ! -f "$test_script" ]]; then
        log::warning "Test phase '$phase' not found: $test_script"
        return 2
    fi
    
    log::info "Running $phase tests (timeout: ${timeout}s)..."
    
    if timeout "$timeout" bash "$test_script"; then
        log::success "$phase tests passed"
        TEST_RESULTS+=("${phase}:passed")
        ((PASSED_TESTS++))
        return 0
    else
        local exit_code=$?
        log::error "$phase tests failed (exit code: $exit_code)"
        TEST_RESULTS+=("${phase}:failed")
        return $exit_code
    fi
}

################################################################################
# Main test execution
################################################################################
main() {
    local test_phase="${1:-all}"
    
    log::info "Starting Odoo resource tests: $test_phase"
    
    case "$test_phase" in
        smoke)
            ((TOTAL_TESTS++))
            run_test_phase "smoke" "$TEST_TIMEOUT_SMOKE"
            ;;
            
        integration)
            ((TOTAL_TESTS++))
            run_test_phase "integration" "$TEST_TIMEOUT_INTEGRATION"
            ;;
            
        unit)
            ((TOTAL_TESTS++))
            run_test_phase "unit" "$TEST_TIMEOUT_UNIT"
            ;;
            
        all)
            # Run all test phases in order
            for phase in smoke unit integration; do
                ((TOTAL_TESTS++))
                case "$phase" in
                    smoke) timeout="$TEST_TIMEOUT_SMOKE" ;;
                    integration) timeout="$TEST_TIMEOUT_INTEGRATION" ;;
                    unit) timeout="$TEST_TIMEOUT_UNIT" ;;
                esac
                run_test_phase "$phase" "$timeout" || true
            done
            ;;
            
        *)
            log::error "Unknown test phase: $test_phase"
            echo "Usage: $0 {smoke|integration|unit|all}"
            exit 1
            ;;
    esac
    
    # Print summary
    echo ""
    log::info "Test Summary: $PASSED_TESTS/$TOTAL_TESTS passed"
    
    # Determine exit code
    if [[ $PASSED_TESTS -eq $TOTAL_TESTS ]]; then
        log::success "All tests passed!"
        exit 0
    elif [[ $PASSED_TESTS -gt 0 ]]; then
        log::warning "Some tests failed"
        exit 1
    else
        log::error "All tests failed"
        exit 2
    fi
}

# Execute main function
main "$@"