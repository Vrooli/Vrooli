#!/usr/bin/env bash
################################################################################
# PostGIS Test Runner
# Main test orchestrator for all test phases
################################################################################

set -euo pipefail

# Determine script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "$SCRIPT_DIR/../../../" && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Test utility functions
test::header() { echo -e "\n=== $* ==="; }
test::success() { echo -e "✅ $*"; }
test::error() { echo -e "❌ $*"; }
log::header() { test::header "$@"; }
log::success() { test::success "$@"; }
log::error() { test::error "$@"; }
log::warning() { echo -e "⚠️  $*"; }
log::info() { echo -e "ℹ️  $*"; }
log::banner() { echo -e "\n╔════════════════════════════════════╗\n║ $* ║\n╚════════════════════════════════════╝"; }

# Test configuration
TEST_TIMEOUT="${TEST_TIMEOUT:-300}"
TEST_PHASES_DIR="${SCRIPT_DIR}/phases"

# Test phase definitions
declare -A TEST_PHASES=(
    ["smoke"]="test-smoke.sh"
    ["integration"]="test-integration.sh"
    ["unit"]="test-unit.sh"
)

# Run specific test phase
run_test_phase() {
    local phase="$1"
    local script="${TEST_PHASES[$phase]}"
    local script_path="${TEST_PHASES_DIR}/${script}"
    
    if [[ ! -f "$script_path" ]]; then
        log::error "Test script not found: $script_path"
        return 1
    fi
    
    log::header "Running $phase tests"
    
    if timeout "$TEST_TIMEOUT" bash "$script_path"; then
        log::success "$phase tests passed"
        return 0
    else
        log::error "$phase tests failed"
        return 1
    fi
}

# Main test execution
main() {
    local phase="${1:-all}"
    local exit_code=0
    
    log::banner "PostGIS Test Suite"
    
    # Check if PostGIS is running
    if ! vrooli resource postgis status &>/dev/null; then
        log::warning "PostGIS not running. Starting it for tests..."
        vrooli resource postgis manage start --wait || {
            log::error "Failed to start PostGIS"
            exit 1
        }
    fi
    
    case "$phase" in
        smoke)
            run_test_phase "smoke" || exit_code=$?
            ;;
        integration)
            run_test_phase "integration" || exit_code=$?
            ;;
        unit)
            run_test_phase "unit" || exit_code=$?
            ;;
        all)
            # Run all phases in order
            for test_phase in smoke unit integration; do
                if [[ ${TEST_PHASES[$test_phase]+isset} ]]; then
                    run_test_phase "$test_phase" || exit_code=$?
                    # Continue even if a phase fails to get full results
                fi
            done
            ;;
        *)
            log::error "Unknown test phase: $phase"
            log::info "Available phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
    
    # Summary
    echo
    if [[ $exit_code -eq 0 ]]; then
        log::success "All tests passed!"
    else
        log::error "Some tests failed"
    fi
    
    exit $exit_code
}

# Handle script arguments
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi