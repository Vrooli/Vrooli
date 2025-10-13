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
# shellcheck disable=SC2154  # var_LOG_FILE is defined in var.sh
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
TEST_TIMEOUT="${TEST_TIMEOUT:-600}"  # Increased to 10 minutes for comprehensive integration tests
TEST_PHASES_DIR="${SCRIPT_DIR}/phases"

# Test phase definitions
# Core test phases (P0/P1 requirements)
declare -A TEST_PHASES=(
    ["smoke"]="test-smoke.sh"
    ["integration"]="test-integration.sh"
    ["unit"]="test-unit.sh"
)

# Optional test phases (P2 features)
declare -A OPTIONAL_TEST_PHASES=(
    ["geocoding"]="test-geocoding.sh"
    ["spatial"]="test-spatial.sh"
    ["visualization"]="test-visualization.sh"
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
        geocoding|spatial|visualization)
            # Optional P2 feature tests
            if [[ -f "${TEST_PHASES_DIR}/${OPTIONAL_TEST_PHASES[$phase]}" ]]; then
                log::header "Running optional $phase tests (P2 features)"
                if timeout "$TEST_TIMEOUT" bash "${TEST_PHASES_DIR}/${OPTIONAL_TEST_PHASES[$phase]}"; then
                    log::success "$phase tests passed"
                else
                    log::error "$phase tests failed"
                    exit_code=1
                fi
            else
                log::error "Optional test not found: $phase"
                exit_code=1
            fi
            ;;
        all)
            # Run core test phases in order
            for test_phase in smoke unit integration; do
                if [[ ${TEST_PHASES[$test_phase]+isset} ]]; then
                    run_test_phase "$test_phase" || exit_code=$?
                    # Continue even if a phase fails to get full results
                fi
            done
            ;;
        extended)
            # Run core + optional tests
            log::header "Running extended test suite (core + P2 features)"
            for test_phase in smoke unit integration; do
                if [[ ${TEST_PHASES[$test_phase]+isset} ]]; then
                    run_test_phase "$test_phase" || exit_code=$?
                fi
            done
            for test_phase in geocoding spatial visualization; do
                if [[ -f "${TEST_PHASES_DIR}/${OPTIONAL_TEST_PHASES[$test_phase]}" ]]; then
                    log::header "Running optional $test_phase tests"
                    timeout "$TEST_TIMEOUT" bash "${TEST_PHASES_DIR}/${OPTIONAL_TEST_PHASES[$test_phase]}" || {
                        log::warning "$test_phase tests failed (non-critical P2 feature)"
                    }
                fi
            done
            ;;
        *)
            log::error "Unknown test phase: $phase"
            log::info "Core phases: smoke, integration, unit, all"
            log::info "Optional P2 phases: geocoding, spatial, visualization, extended"
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