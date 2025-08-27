#!/usr/bin/env bash
# Vault Resource Test Runner - Main entry point for Vault resource validation
# Follows v2.0 CLI contract testing standards with phase-based approach
#
# Test Phases:
#   smoke       - Quick health validation (30s max)
#   unit        - Library function tests (60s max)
#   integration - Full end-to-end testing (120s max)

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
VAULT_CLI_DIR="${APP_ROOT}/resources/vault"
TEST_DIR="${VAULT_CLI_DIR}/test"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"

show_usage() {
    cat << 'EOF'
Vault Resource Test Runner - Validate Vault service functionality

Usage: ./run-tests.sh [OPTIONS] [PHASES...]

OPTIONS:
  --verbose         Show detailed output for each test
  --no-summary      Skip test summary
  --continue        Continue on test failures (don't exit early)
  --help            Show this help

PHASES:
  smoke            Quick health validation (30s)
  unit             Library function tests (60s)
  integration      Full end-to-end testing (120s)
  all              Run all test phases (default)

EXAMPLES:
  ./run-tests.sh                    # Run all tests
  ./run-tests.sh smoke              # Quick health check
  ./run-tests.sh smoke integration  # Smoke + integration only
  ./run-tests.sh --verbose all      # All tests with detailed output

NOTE: These tests validate Vault as a RESOURCE (health, connectivity).
      For secret management testing, use: resource-vault test secrets
EOF
}

# Test phase functions
run_smoke_tests() {
    log::info "=== VAULT RESOURCE SMOKE TESTS ==="
    if bash "${TEST_DIR}/phases/test-smoke.sh"; then
        return 0
    else
        return 1
    fi
}

run_unit_tests() {
    log::info "=== VAULT RESOURCE UNIT TESTS ==="
    if bash "${TEST_DIR}/phases/test-unit.sh"; then
        return 0
    else
        return 1
    fi
}

run_integration_tests() {
    log::info "=== VAULT RESOURCE INTEGRATION TESTS ==="
    if bash "${TEST_DIR}/phases/test-integration.sh"; then
        return 0
    else
        return 1
    fi
}

# Main test runner
main() {
    local phases=()
    local verbose=false
    local no_summary=false
    local continue_on_failure=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose)
                verbose=true
                shift
                ;;
            --no-summary)
                no_summary=true
                shift
                ;;
            --continue)
                continue_on_failure=true
                shift
                ;;
            --help|-h)
                show_usage
                return 0
                ;;
            all)
                phases=("smoke" "unit" "integration")
                shift
                ;;
            smoke|unit|integration)
                phases+=("$1")
                shift
                ;;
            *)
                log::error "Unknown option: $1"
                show_usage
                return 1
                ;;
        esac
    done
    
    # Default to all phases if none specified
    if [[ ${#phases[@]} -eq 0 ]]; then
        phases=("smoke" "unit" "integration")
    fi
    
    # Set verbose logging if requested
    if [[ "$verbose" == "true" ]]; then
        export LOG_LEVEL="DEBUG"
        export VAULT_TEST_VERBOSE="true"
    fi
    
    log::info "🧪 Starting Vault Resource Validation Tests"
    log::info "Phases: ${phases[*]}"
    echo ""
    
    local start_time=$(date +%s)
    local phase_results=()
    local overall_status=0
    
    # Run each phase
    for phase in "${phases[@]}"; do
        local phase_start=$(date +%s)
        local phase_status=0
        
        case "$phase" in
            smoke)
                run_smoke_tests || phase_status=1
                ;;
            unit)
                run_unit_tests || phase_status=1
                ;;
            integration)
                run_integration_tests || phase_status=1
                ;;
            *)
                log::error "Unknown test phase: $phase"
                phase_status=1
                ;;
        esac
        
        local phase_duration=$(($(date +%s) - phase_start))
        
        if [[ $phase_status -eq 0 ]]; then
            phase_results+=("✅ $phase (${phase_duration}s)")
        else
            phase_results+=("❌ $phase (${phase_duration}s)")
            overall_status=1
            
            if [[ "$continue_on_failure" != "true" ]]; then
                log::error "Test phase '$phase' failed. Stopping execution."
                break
            fi
        fi
        
        echo ""
    done
    
    local total_duration=$(($(date +%s) - start_time))
    
    # Show summary unless disabled
    if [[ "$no_summary" != "true" ]]; then
        echo "================================"
        log::info "🧪 Vault Resource Test Summary"
        echo "================================"
        
        for result in "${phase_results[@]}"; do
            echo "$result"
        done
        
        echo ""
        echo "Total Duration: ${total_duration}s"
        echo "Timestamp: $(date)"
        
        if [[ $overall_status -eq 0 ]]; then
            echo ""
            log::success "🎉 ALL VAULT RESOURCE TESTS PASSED"
            echo "Vault service is healthy and fully functional"
        else
            echo ""
            log::error "💥 SOME VAULT RESOURCE TESTS FAILED"
            echo "Vault service needs attention before use"
        fi
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi