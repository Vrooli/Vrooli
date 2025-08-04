#!/usr/bin/env bash
# Resource Integration Test Runner
# This runner is called by scripts/resources/index.sh after resource discovery and filtering
# It receives a list of healthy local resources and runs appropriate tests for each

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Script directory and paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$(dirname "$SCRIPT_DIR")"
VROOLI_ROOT="$(cd "$RESOURCES_DIR/../.." && pwd)"

# Source the main resources script to get utility functions
# shellcheck disable=SC1091
source "$RESOURCES_DIR/index.sh"

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
# Logging functions
#######################################
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

log_header() {
    echo
    echo -e "${BLUE}=== $* ===${NC}"
    echo
}

#######################################
# Parse command line arguments
#######################################
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --verbose|-v)
                VERBOSE="true"
                shift
                ;;
            --help|-h)
                show_help
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
show_help() {
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
test_resource() {
    local resource="$1"
    
    log_header "Testing: $resource"
    
    # Get resource category
    local category
    category=$(resources::get_category "$resource")
    if [[ -z "$category" ]]; then
        log_error "$resource: Unable to determine resource category"
        return 1
    fi
    
    # Check for standardized integration test
    local integration_test="$RESOURCES_DIR/$category/$resource/test/integration-test.sh"
    [[ "$VERBOSE" == "true" ]] && log_info "$resource: Looking for test at: $integration_test"
    
    if [[ -f "$integration_test" ]]; then
        log_info "$resource: Running integration tests..."
        # Temporarily disable exit-on-error for integration test execution
        # since we want to handle the return codes (0, 1, 2) explicitly
        set +e
        timeout 120 bash "$integration_test"
        local test_result=$?
        set -e
        
        case $test_result in
            0)
                log_success "✅ $resource: Integration tests passed"
                return 0
                ;;
            1)
                log_error "❌ $resource: Integration tests failed"
                return 1
                ;;
            2)
                log_warning "⏭️  $resource: Integration tests skipped"
                return 2
                ;;
            124)
                log_error "❌ $resource: Integration tests timed out (120s)"
                return 1
                ;;
            *)
                log_error "❌ $resource: Integration tests returned unexpected exit code: $test_result"
                return 1
                ;;
        esac
    else
        log_info "$resource: No integration tests found at: $integration_test"
        log_warning "⏭️  $resource: Skipped (no tests available)"
        return 2
    fi
}

#######################################
# Main execution
#######################################
main() {
    parse_args "$@"
    
    log_header "Resource Integration Test Runner"
    
    # Check if we have resources to test
    if [[ -z "${HEALTHY_RESOURCES_STR:-}" ]]; then
        log_error "No resources provided via HEALTHY_RESOURCES_STR environment variable"
        log_info "This script is meant to be called by scripts/resources/index.sh"
        exit 1
    fi
    
    # Convert space-separated string to array
    local -a resources_array
    read -ra resources_array <<< "$HEALTHY_RESOURCES_STR"
    
    if [[ ${#resources_array[@]} -eq 0 ]]; then
        log_warning "No healthy resources to test"
        exit 0
    fi
    
    log_info "Testing ${#resources_array[@]} healthy resources: ${resources_array[*]}"
    [[ "$VERBOSE" == "true" ]] && log_info "Debug: About to start resource loop"
    echo
    
    # Run tests for each resource
    for resource in "${resources_array[@]}"; do
        [[ "$VERBOSE" == "true" ]] && log_info "Debug: Processing resource: $resource"
        TESTS_RUN=$((TESTS_RUN + 1))
        [[ "$VERBOSE" == "true" ]] && log_info "Debug: Tests run counter: $TESTS_RUN"
        [[ "$VERBOSE" == "true" ]] && log_info "Debug: About to call test_resource"
        
        # Temporarily disable exit-on-error for test_resource calls
        # since we want to handle the return codes (0, 1, 2) explicitly
        set +e
        test_resource "$resource"
        local result=$?
        set -e
        [[ "$VERBOSE" == "true" ]] && log_info "Debug: test_resource returned: $result"
        
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
    log_header "Test Results Summary"
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Total Resources Tested: $TESTS_RUN"
    echo "  Passed:                 $TESTS_PASSED ✅"
    echo "  Failed:                 $TESTS_FAILED ❌"
    echo "  Skipped (no tests):     $TESTS_SKIPPED ⏭️"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ ${#FAILED_TESTS[@]} -gt 0 ]]; then
        echo
        log_error "Failed Resources:"
        for failed_resource in "${FAILED_TESTS[@]}"; do
            echo "  ❌ $failed_resource"
        done
    fi
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log_success "🎉 All resource tests passed or were skipped!"
        exit 0
    else
        log_error "💥 $TESTS_FAILED resource test(s) failed"
        exit 1
    fi
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi