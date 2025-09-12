#!/usr/bin/env bash
# SearXNG Resource Test Implementation
# Provides test functionality for v2.0 contract compliance

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SEARXNG_CLI_DIR="${APP_ROOT}/resources/searxng"
TEST_DIR="${SEARXNG_CLI_DIR}/test"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Run smoke tests - quick health validation
# Globals: None
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
searxng::test::smoke() {
    log::info "Running smoke test..."
    if [[ -f "${TEST_DIR}/phases/test-smoke.sh" ]]; then
        bash "${TEST_DIR}/phases/test-smoke.sh"
    else
        log::error "Smoke test script not found"
        return 1
    fi
}

#######################################
# Run integration tests - full functionality
# Globals: None
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
searxng::test::integration() {
    log::info "Running integration test..."
    if [[ -f "${TEST_DIR}/phases/test-integration.sh" ]]; then
        bash "${TEST_DIR}/phases/test-integration.sh"
    else
        log::error "Integration test script not found"
        return 1
    fi
}

#######################################
# Run unit tests - library functions
# Globals: None
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
searxng::test::unit() {
    log::info "Running unit test..."
    if [[ -f "${TEST_DIR}/phases/test-unit.sh" ]]; then
        bash "${TEST_DIR}/phases/test-unit.sh"
    else
        log::error "Unit test script not found"
        return 1
    fi
}

#######################################
# Run all tests
# Globals: None
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
searxng::test::all() {
    log::info "Running all tests..."
    local overall_status=0
    
    # Run smoke test
    if ! searxng::test::smoke; then
        overall_status=1
    fi
    
    # Run unit test
    if ! searxng::test::unit; then
        overall_status=1
    fi
    
    # Run integration test
    if ! searxng::test::integration; then
        overall_status=1
    fi
    
    if [[ $overall_status -eq 0 ]]; then
        log::success "All tests passed!"
    else
        log::error "Some tests failed"
    fi
    
    return $overall_status
}

#######################################
# Main test handler
# Globals: None
# Arguments: $@ - all arguments passed from CLI
# Returns: Appropriate exit code
#######################################
searxng::test() {
    # Extract the phase from the CLI framework
    # The phase is passed as the last argument in the format test::phase
    local phase=""
    for arg in "$@"; do
        case "$arg" in
            smoke|integration|unit|all)
                phase="$arg"
                break
                ;;
        esac
    done
    
    # If no phase found, check if it's in the CLI_CURRENT_COMMAND
    if [[ -z "$phase" ]] && [[ -n "${CLI_CURRENT_COMMAND:-}" ]]; then
        phase="${CLI_CURRENT_COMMAND##*::}"
    fi
    
    # Default to all if no phase specified
    phase="${phase:-all}"
    
    case "$phase" in
        smoke)
            searxng::test::smoke
            ;;
        integration)
            searxng::test::integration
            ;;
        unit)
            searxng::test::unit
            ;;
        all)
            searxng::test::all
            ;;
        *)
            log::error "Unknown test phase: $phase"
            log::info "Valid phases: smoke, integration, unit, all"
            return 1
            ;;
    esac
}