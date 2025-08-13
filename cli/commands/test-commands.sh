#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Test Management Commands
# 
# Provides unified interface to the Vrooli testing infrastructure located in
# scripts/__test-revised/. This connects the CLI to the convention-based
# testing system designed for Vrooli's scenario-first architecture.
#
# Usage:
#   vrooli test [type] [options]
#
################################################################################

set -euo pipefail

# Get CLI directory and project root
CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="$(cd "$CLI_DIR/../.." && pwd)"

# Source utilities
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Path to the revised testing infrastructure
TEST_INFRASTRUCTURE_DIR="$VROOLI_ROOT/scripts/__test-revised"
TEST_ORCHESTRATOR="$TEST_INFRASTRUCTURE_DIR/run-tests.sh"

# Verify test infrastructure exists
check_test_infrastructure() {
    if [[ ! -f "$TEST_ORCHESTRATOR" ]]; then
        log::error "Test infrastructure not found at: $TEST_ORCHESTRATOR"
        echo "The revised testing infrastructure should be available in scripts/__test-revised/"
        return 1
    fi
    
    if [[ ! -x "$TEST_ORCHESTRATOR" ]]; then
        log::warning "Test orchestrator not executable, fixing permissions..."
        chmod +x "$TEST_ORCHESTRATOR" || {
            log::error "Cannot make test orchestrator executable"
            return 1
        }
    fi
    
    return 0
}

# Show help for test commands
show_test_help() {
    cat << EOF
🧪 Vrooli Test Commands - Connected to revised testing infrastructure

USAGE:
    vrooli test [type] [options]

TEST TYPES:
    all          Run all test suites (default)
    static       Run static analysis (shellcheck, bash -n) on all scripts
    resources    Run resource validation and mock testing
    scenarios    Run scenario validation and integration tests
    bats         Run all BATS test files with caching

OPTIONS:
    --verbose      Show detailed output for all test phases
    --parallel     Run tests in parallel where possible
    --no-cache     Skip caching optimizations - force re-run all tests
    --dry-run      Show what tests would be run without executing
    --clear-cache  Clear test cache before running
    --timeout N    Set timeout in minutes (default: 15)

EXAMPLES:
    vrooli test                         # Run all shell tests (static, resources, scenarios, bats)
    vrooli test static --verbose        # Run static analysis with detailed output
    vrooli test resources --dry-run     # Show what resource tests would run
    vrooli test scenarios --no-cache    # Force re-run scenario validation
    vrooli test bats --parallel         # Run BATS tests in parallel

INFRASTRUCTURE:
    The test commands connect to the convention-based testing infrastructure
    located in scripts/__test-revised/. This system automatically discovers
    and tests resources, validates scenarios, and runs comprehensive analysis
    focused on Vrooli's scenario-first architecture.
    
    Test results are cached intelligently to avoid unnecessary reruns.
    Each test phase can be run independently or as part of the complete suite.

For more details on the testing architecture:
    https://docs.vrooli.com/testing
EOF
}


# Main test command handler
main() {
    # Check that test infrastructure is available
    if ! check_test_infrastructure; then
        return 1
    fi
    
    # Parse arguments to separate test type from options
    local test_type="all"
    local args=()
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            # These are test types
            all|static|resources|scenarios|bats|help|--help|-h)
                test_type="$1"
                shift
                # Collect remaining arguments
                args+=("$@")
                break
                ;;
            # These are options - collect them
            --*)
                args+=("$1")
                shift
                ;;
            # Unknown argument - treat as test type
            *)
                test_type="$1"
                shift
                args+=("$@")
                break
                ;;
        esac
    done
    
    case "$test_type" in
        # All test suites (revised infrastructure)
        all)
            log::info "🧪 Running complete test suite via revised infrastructure..."
            if [[ ${#args[@]} -eq 0 ]]; then
                exec "$TEST_ORCHESTRATOR" all
            else
                exec "$TEST_ORCHESTRATOR" all "${args[@]}"
            fi
            ;;
        static)
            log::info "🔍 Running static analysis tests..."
            if [[ ${#args[@]} -eq 0 ]]; then
                exec "$TEST_ORCHESTRATOR" static
            else
                exec "$TEST_ORCHESTRATOR" static "${args[@]}"
            fi
            ;;
        resources)
            log::info "⚡ Running resource validation tests..."
            if [[ ${#args[@]} -eq 0 ]]; then
                exec "$TEST_ORCHESTRATOR" resources
            else
                exec "$TEST_ORCHESTRATOR" resources "${args[@]}"
            fi
            ;;
        scenarios)
            log::info "🎬 Running scenario validation tests..."
            if [[ ${#args[@]} -eq 0 ]]; then
                exec "$TEST_ORCHESTRATOR" scenarios
            else
                exec "$TEST_ORCHESTRATOR" scenarios "${args[@]}"
            fi
            ;;
        bats)
            log::info "🦇 Running BATS test suite..."
            if [[ ${#args[@]} -eq 0 ]]; then
                exec "$TEST_ORCHESTRATOR" bats
            else
                exec "$TEST_ORCHESTRATOR" bats "${args[@]}"
            fi
            ;;
            
        # Help and unknown commands
        help|--help|-h)
            show_test_help
            ;;
        *)
            log::error "Unknown test type: $test_type"
            echo ""
            echo "Run 'vrooli test --help' for usage information"
            return 1
            ;;
    esac
}

# Export functions for testing (if needed)
export -f check_test_infrastructure

# Execute main function with all arguments
main "$@"