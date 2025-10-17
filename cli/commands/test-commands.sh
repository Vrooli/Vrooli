#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Test Management Commands
# 
# Provides unified interface to the Vrooli testing infrastructure located in
# __test/. This connects the CLI to the convention-based
# testing system designed for Vrooli's scenario-first architecture.
#
# Usage:
#   vrooli test [type] [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
CLI_DIR="${APP_ROOT}/cli/commands"
VROOLI_ROOT="$APP_ROOT"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Path to the testing infrastructure
TEST_INFRASTRUCTURE_DIR="$VROOLI_ROOT/__test"
TEST_ORCHESTRATOR="$TEST_INFRASTRUCTURE_DIR/run-tests.sh"

# Verify test infrastructure exists
check_test_infrastructure() {
    if [[ ! -f "$TEST_ORCHESTRATOR" ]]; then
        log::error "Test infrastructure not found at: $TEST_ORCHESTRATOR"
        echo "The testing infrastructure should be available in __test/"
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
    static       Run static analysis (shellcheck, TypeScript, Python, Go) on all files
    structure    Run file/directory structure and configuration validation
    integration  Run integration tests (resource mocks, app testing, etc.)
    unit         Run all unit tests (BATS) with caching
    docs         Run documentation validation (markdown syntax & links)

OPTIONS:
    --verbose      Show detailed output for all test phases
    --parallel     Run tests in parallel where possible
    --no-cache     Skip caching optimizations - force re-run all tests
    --dry-run      Show what tests would be run without executing
    --clear-cache  Clear test cache before running
    --timeout N    Set timeout in seconds (default: 900)

EXAMPLES:
    vrooli test                         # Run all tests (900s timeout)
    vrooli test --timeout 1800          # Run all tests with 30 minute timeout
    vrooli test static --verbose        # Run static analysis with detailed output
    vrooli test structure --dry-run     # Show what structure tests would run
    vrooli test integration --no-cache  # Force re-run integration tests
    vrooli test unit --parallel         # Run unit tests in parallel
    vrooli test docs --verbose          # Run documentation validation with details

INFRASTRUCTURE:
    The test commands connect to the test-type-based testing infrastructure
    located in __test/. This system provides consistent, focused testing
    where each phase has a single responsibility (static analysis, structure
    validation, integration testing, unit testing, documentation validation).
    
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
    
    # First, filter shell artifacts from arguments
    local clean_args=()
    for arg in "$@"; do
        case "$arg" in
            # Skip shell redirection operators
            "2>&1"|"1>&2"|"&>"|">&"|"2>"|"1>"|">"|">>"|"<"|"<<")
                continue
                ;;
            # Filter suspicious single digits (likely shell artifacts)
            [0-9])
                # If this is a standalone digit at the end, it's likely a shell artifact
                local next_arg=""
                local found_next=false
                for check_arg in "${clean_args[@]}" "$@"; do
                    if [[ "$found_next" == "true" ]]; then
                        next_arg="$check_arg"
                        break
                    fi
                    [[ "$check_arg" == "$arg" ]] && found_next=true
                done
                # If no next arg or next arg is a shell operator, skip this digit
                if [[ -z "$next_arg" ]] || [[ "$next_arg" =~ ^[\>\<\&\|] ]]; then
                    continue
                fi
                ;;
        esac
        clean_args+=("$arg")
    done
    
    # Parse filtered arguments to separate test type from options
    local test_type="all"
    local args=()
    
    set -- "${clean_args[@]}"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            # These are test types
            all|static|structure|integration|unit|docs|help|--help|-h)
                test_type="$1"
                shift
                # Collect remaining arguments
                args+=("$@")
                break
                ;;
            # Handle timeout option with value
            --timeout)
                args+=("$1")
                shift
                if [[ $# -gt 0 ]]; then
                    args+=("$1")
                    shift
                fi
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
        structure)
            log::info "🏗️ Running structure validation tests..."
            if [[ ${#args[@]} -eq 0 ]]; then
                exec "$TEST_ORCHESTRATOR" structure
            else
                exec "$TEST_ORCHESTRATOR" structure "${args[@]}"
            fi
            ;;
        integration)
            log::info "⚡ Running integration tests..."
            if [[ ${#args[@]} -eq 0 ]]; then
                exec "$TEST_ORCHESTRATOR" integration
            else
                exec "$TEST_ORCHESTRATOR" integration "${args[@]}"
            fi
            ;;
        unit)
            log::info "🧪 Running unit test suite..."
            if [[ ${#args[@]} -eq 0 ]]; then
                exec "$TEST_ORCHESTRATOR" unit
            else
                exec "$TEST_ORCHESTRATOR" unit "${args[@]}"
            fi
            ;;
        docs)
            log::info "📚 Running documentation validation..."
            if [[ ${#args[@]} -eq 0 ]]; then
                exec "$TEST_ORCHESTRATOR" docs
            else
                exec "$TEST_ORCHESTRATOR" docs "${args[@]}"
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