#!/usr/bin/env bash
# Resource Integration Test Runner
# This runner is called by scripts/resources/index.sh after resource discovery and filtering
# It receives a list of healthy local resources and runs appropriate tests for each

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Configuration
VERBOSE="${VERBOSE:-false}"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Failed tests for reporting
declare -a FAILED_TESTS=()

#######################################
# Parse command line arguments
#######################################
run::parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --verbose|-v)
                VERBOSE="true"
                shift
                ;;
            --help|-h)
                run::show_help
                exit 0
                ;;
            *)
                shift
                ;;
        esac
    done
}

#######################################
# Show help information
#######################################
run::show_help() {
    cat << EOF
Resource Integration Test Runner

This runner tests healthy local resources discovered by the resource system.

Usage: $0 [OPTIONS]

Options:
    --verbose, -v     Enable verbose output
    --help, -h        Show this help message

Environment Variables:
    HEALTHY_RESOURCES_STR    Space-separated list of healthy resources to test

Examples:
    HEALTHY_RESOURCES_STR="ollama whisper" $0
    HEALTHY_RESOURCES_STR="ollama" $0 --verbose
EOF
}

#######################################
# Test a single resource
# Arguments:
#   $1 - resource name
# Returns:
#   0 if tests pass, 1 if tests fail, 2 if no tests found
#######################################
run::test_resource() {
    local resource="$1"
    
    log::header "Testing: $resource"
    
    # Get resource category
    local category
    category=$(resources::get_category "$resource")
    if [[ -z "$category" ]]; then
        log::error "$resource: Unable to determine resource category"
        return 1
    fi
    
    # Check for standardized integration test  
    local integration_test="${var_SCRIPTS_RESOURCES_DIR}/$category/$resource/test/integration-test.sh"
    [[ "$VERBOSE" == "true" ]] && log::info "$resource: Looking for test at: $integration_test"
    
    if [[ -f "$integration_test" ]]; then
        log::info "$resource: Running integration tests..."
        # Temporarily disable exit-on-error for integration test execution
        # since we want to handle the return codes (0, 1, 2) explicitly
        set +e
        timeout 120 bash "$integration_test"
        local test_result=$?
        set -e
        
        case $test_result in
            0)
                log::success "✅ $resource: Integration tests passed"
                return 0
                ;;
            1)
                log::error "❌ $resource: Integration tests failed"
                return 1
                ;;
            2)
                log::warning "⏭️  $resource: Integration tests skipped"
                return 2
                ;;
            124)
                log::error "❌ $resource: Integration tests timed out (120s)"
                return 1
                ;;
            *)
                log::error "❌ $resource: Integration tests returned unexpected exit code: $test_result"
                return 1
                ;;
        esac
    else
        log::info "$resource: No integration tests found at: $integration_test"
        log::warning "⏭️  $resource: Skipped (no tests available)"
        return 2
    fi
}

#######################################
# Main execution
#######################################
run::main() {
    run::parse_args "$@"
    
    log::header "Resource Integration Test Runner"
    
    # Check if we have resources to test
    if [[ -z "${HEALTHY_RESOURCES_STR:-}" ]]; then
        log::error "No resources provided via HEALTHY_RESOURCES_STR environment variable"
        log::info "This script is meant to be called by scripts/resources/index.sh"
        exit 1
    fi
    
    # Convert space-separated string to array
    local -a resources_array
    read -ra resources_array <<< "$HEALTHY_RESOURCES_STR"
    
    if [[ ${#resources_array[@]} -eq 0 ]]; then
        log::warning "No healthy resources to test"
        exit 0
    fi
    
    log::info "Testing ${#resources_array[@]} healthy resources: ${resources_array[*]}"
    [[ "$VERBOSE" == "true" ]] && log::info "Debug: About to start resource loop"
    echo
    
    # Run tests for each resource
    for resource in "${resources_array[@]}"; do
        [[ "$VERBOSE" == "true" ]] && log::info "Debug: Processing resource: $resource"
        TESTS_RUN=$((TESTS_RUN + 1))
        [[ "$VERBOSE" == "true" ]] && log::info "Debug: Tests run counter: $TESTS_RUN"
        [[ "$VERBOSE" == "true" ]] && log::info "Debug: About to call test_resource"
        
        # Temporarily disable exit-on-error for test_resource calls
        # since we want to handle the return codes (0, 1, 2) explicitly
        set +e
        run::test_resource "$resource"
        local result=$?
        set -e
        [[ "$VERBOSE" == "true" ]] && log::info "Debug: test_resource returned: $result"
        
        case $result in
            0)
                TESTS_PASSED=$((TESTS_PASSED + 1))
                ;;
            1)
                TESTS_FAILED=$((TESTS_FAILED + 1))
                FAILED_TESTS+=("$resource")
                ;;
            2)
                TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
                ;;
        esac
    done
    
    # Generate final report
    echo
    log::header "Test Results Summary"
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Total Resources Tested: $TESTS_RUN"
    echo "  Passed:                 $TESTS_PASSED ✅"
    echo "  Failed:                 $TESTS_FAILED ❌"
    echo "  Skipped (no tests):     $TESTS_SKIPPED ⏭️"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ ${#FAILED_TESTS[@]} -gt 0 ]]; then
        echo
        log::error "Failed Resources:"
        for failed_resource in "${FAILED_TESTS[@]}"; do
            echo "  ❌ $failed_resource"
        done
    fi
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log::success "🎉 All resource tests passed or were skipped!"
        exit 0
    else
        log::error "💥 $TESTS_FAILED resource test(s) failed"
        exit 1
    fi
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run::main "$@"
fi