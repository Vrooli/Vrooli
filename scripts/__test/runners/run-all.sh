#!/usr/bin/env bash
# Unified Test Runner - Run all tests with smart defaults
# Provides a single entry point for running the entire test suite

set -euo pipefail

# Script directory
RUNNER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="$(dirname "$RUNNER_DIR")"
SCRIPTS_DIR="$(dirname "$TEST_ROOT")"

# Load shared utilities
source "$TEST_ROOT/shared/logging.bash"
source "$TEST_ROOT/shared/utils.bash"
source "$TEST_ROOT/shared/config-simple.bash"
source "$TEST_ROOT/shared/test-isolation.bash"

# Configuration
PARALLEL_ENABLED="${VROOLI_TEST_PARALLEL:-true}"
MAX_PARALLEL_JOBS="${VROOLI_TEST_MAX_PARALLEL:-4}"
TIMEOUT_MULTIPLIER="${VROOLI_TEST_TIMEOUT_MULTIPLIER:-1}"
VERBOSE="${VROOLI_TEST_VERBOSE:-false}"
FAIL_FAST="${VROOLI_TEST_FAIL_FAST:-false}"
COVERAGE="${VROOLI_TEST_COVERAGE:-false}"

# Ensure VROOLI_LOG_LEVEL is set for child processes
export VROOLI_LOG_LEVEL="${VROOLI_LOG_LEVEL:-INFO}"

# Test statistics
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0
START_TIME=$(date +%s)

#######################################
# Print usage information
#######################################
print_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Run all Vrooli tests with smart defaults and optimizations.

OPTIONS:
    --parallel          Enable parallel execution (default: true)
    --jobs N           Number of parallel jobs (default: 4)
    --timeout-mult N   Timeout multiplier for slow tests (default: 1)
    --verbose          Enable verbose output
    --fail-fast        Stop on first failure
    --coverage         Generate coverage report
    --filter PATTERN   Filter tests by pattern
    --dry-run          Show what would be run without executing
    -h, --help         Show this help message

EXAMPLES:
    $0                          # Run all tests with defaults
    $0 --parallel --jobs 8      # Run with 8 parallel jobs
    $0 --filter "ollama"        # Run only Ollama-related tests
    $0 --fail-fast --verbose    # Stop on first failure with verbose output

ENVIRONMENT VARIABLES:
    VROOLI_TEST_PARALLEL        Enable parallel execution
    VROOLI_TEST_MAX_PARALLEL    Number of parallel jobs
    VROOLI_TEST_TIMEOUT_MULTIPLIER  Timeout multiplier
    VROOLI_TEST_VERBOSE         Verbose output
    VROOLI_TEST_FAIL_FAST       Stop on first failure
    
EOF
}

#######################################
# Parse command line arguments
#######################################
parse_args() {
    local filter_pattern=""
    local dry_run=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --parallel)
                PARALLEL_ENABLED="true"
                shift
                ;;
            --no-parallel)
                PARALLEL_ENABLED="false"
                shift
                ;;
            --jobs)
                MAX_PARALLEL_JOBS="$2"
                shift 2
                ;;
            --timeout-mult)
                TIMEOUT_MULTIPLIER="$2"
                shift 2
                ;;
            --verbose|-v)
                VERBOSE="true"
                shift
                ;;
            --fail-fast)
                FAIL_FAST="true"
                shift
                ;;
            --coverage)
                COVERAGE="true"
                shift
                ;;
            --filter)
                filter_pattern="$2"
                shift 2
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            -h|--help)
                print_usage
                exit 0
                ;;
            *)
                vrooli_log_error "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    # Export for child processes
    export VROOLI_TEST_FILTER="$filter_pattern"
    export VROOLI_TEST_DRY_RUN="$dry_run"
}

#######################################
# Initialize test environment
#######################################
init_test_environment() {
    vrooli_log_header "🚀 Vrooli Test Suite - All Tests"
    
    # Initialize configuration
    vrooli_config_init
    
    # Initialize test isolation
    local namespace
    namespace=$(vrooli_isolation_init "all-tests")
    vrooli_log_info "Test namespace: $namespace"
    
    # Set up cleanup traps
    vrooli_isolation_setup_traps
    
    # Show configuration
    vrooli_log_info "Configuration:"
    vrooli_log_info "  Parallel: $PARALLEL_ENABLED (max $MAX_PARALLEL_JOBS jobs)"
    vrooli_log_info "  Timeout multiplier: ${TIMEOUT_MULTIPLIER}x"
    vrooli_log_info "  Verbose: $VERBOSE"
    vrooli_log_info "  Fail fast: $FAIL_FAST"
    vrooli_log_info "  Coverage: $COVERAGE"
    
    if [[ -n "${VROOLI_TEST_FILTER:-}" ]]; then
        vrooli_log_info "  Filter: $VROOLI_TEST_FILTER"
    fi
}

#######################################
# Discover all test files
#######################################
discover_tests() {
    local -a unit_tests=()
    local -a integration_tests=()
    local -a shell_tests=()
    
    vrooli_log_info "Discovering test files..."
    
    # Find BATS unit tests
    while IFS= read -r test_file; do
        # Apply filter if specified
        if [[ -n "${VROOLI_TEST_FILTER:-}" ]]; then
            if [[ ! "$test_file" =~ $VROOLI_TEST_FILTER ]]; then
                continue
            fi
        fi
        unit_tests+=("$test_file")
    done < <(find "$TEST_ROOT/fixtures/tests" -name "*.bats" -type f 2>/dev/null | sort)
    
    # Find integration tests
    if [[ -d "$TEST_ROOT/integration/services" ]]; then
        while IFS= read -r test_file; do
            if [[ -n "${VROOLI_TEST_FILTER:-}" ]]; then
                if [[ ! "$test_file" =~ $VROOLI_TEST_FILTER ]]; then
                    continue
                fi
            fi
            integration_tests+=("$test_file")
        done < <(find "$TEST_ROOT/integration/services" -name "*.sh" -type f 2>/dev/null | sort)
    fi
    
    # Find shell script BATS tests in resources
    while IFS= read -r test_file; do
        if [[ -n "${VROOLI_TEST_FILTER:-}" ]]; then
            if [[ ! "$test_file" =~ $VROOLI_TEST_FILTER ]]; then
                continue
            fi
        fi
        shell_tests+=("$test_file")
    done < <(find "$SCRIPTS_DIR/resources" -name "*.bats" -type f 2>/dev/null | sort)
    
    # Report discovery results
    TOTAL_TESTS=$((${#unit_tests[@]} + ${#integration_tests[@]} + ${#shell_tests[@]}))
    
    # Count shell tests by category for better visibility
    local ai_tests=$(find "$SCRIPTS_DIR/resources/ai" -name "*.bats" -type f 2>/dev/null | wc -l)
    local agents_tests=$(find "$SCRIPTS_DIR/resources/agents" -name "*.bats" -type f 2>/dev/null | wc -l)
    local automation_tests=$(find "$SCRIPTS_DIR/resources/automation" -name "*.bats" -type f 2>/dev/null | wc -l)
    local execution_tests=$(find "$SCRIPTS_DIR/resources/execution" -name "*.bats" -type f 2>/dev/null | wc -l)
    local search_tests=$(find "$SCRIPTS_DIR/resources/search" -name "*.bats" -type f 2>/dev/null | wc -l)
    local storage_tests=$(find "$SCRIPTS_DIR/resources/storage" -name "*.bats" -type f 2>/dev/null | wc -l)
    local other_tests=$(find "$SCRIPTS_DIR/resources" -maxdepth 1 -name "*.bats" -type f 2>/dev/null | wc -l)
    
    vrooli_log_header "📊 Test Discovery Summary"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  TOTAL TESTS FOUND: $TOTAL_TESTS"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Test Types:"
    echo "    Unit tests:        ${#unit_tests[@]}"
    echo "    Integration tests: ${#integration_tests[@]}"
    echo "    Shell tests:       ${#shell_tests[@]}"
    echo ""
    echo "  Shell Tests by Category:"
    echo "    AI:         $ai_tests"
    echo "    Agents:     $agents_tests"
    echo "    Automation: $automation_tests"
    echo "    Execution:  $execution_tests"
    echo "    Search:     $search_tests"
    echo "    Storage:    $storage_tests"
    echo "    Other:      $other_tests"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    
    # Export test arrays for use in other functions
    export UNIT_TESTS=("${unit_tests[@]}")
    export INTEGRATION_TESTS=("${integration_tests[@]}")
    export SHELL_TESTS=("${shell_tests[@]}")
}

#######################################
# Run unit tests
#######################################
run_unit_tests() {
    if [[ ${#UNIT_TESTS[@]} -eq 0 ]]; then
        vrooli_log_info "No unit tests to run"
        return 0
    fi
    
    vrooli_log_header "📦 Running Unit Tests (${#UNIT_TESTS[@]} files)"
    
    if [[ "${VROOLI_TEST_DRY_RUN:-false}" == "true" ]]; then
        for test in "${UNIT_TESTS[@]}"; do
            echo "Would run: $test"
        done
        return 0
    fi
    
    # Run unit tests using the dedicated runner
    # Build command arguments
    local cmd_args=()
    [[ "$PARALLEL_ENABLED" == "true" ]] && cmd_args+=(--parallel)
    [[ -n "$MAX_PARALLEL_JOBS" ]] && [[ "$PARALLEL_ENABLED" == "true" ]] && cmd_args+=(--jobs "$MAX_PARALLEL_JOBS")
    [[ "$VERBOSE" == "true" ]] && cmd_args+=(--verbose)
    [[ "$FAIL_FAST" == "true" ]] && cmd_args+=(--fail-fast)
    
    # Call run-unit.sh with proper array handling
    if [[ ${#cmd_args[@]} -gt 0 ]]; then
        if "$RUNNER_DIR/run-unit.sh" "${cmd_args[@]}" "${UNIT_TESTS[@]}"; then
            vrooli_log_success "Unit tests completed successfully"
            return 0
        else
            vrooli_log_error "Unit tests failed"
            return 1
        fi
    else
        if "$RUNNER_DIR/run-unit.sh" "${UNIT_TESTS[@]}"; then
            vrooli_log_success "Unit tests completed successfully"
            return 0
        else
            vrooli_log_error "Unit tests failed"
            return 1
        fi
    fi
}

#######################################
# Run integration tests
#######################################
run_integration_tests() {
    if [[ ${#INTEGRATION_TESTS[@]} -eq 0 ]]; then
        vrooli_log_info "No integration tests to run"
        return 0
    fi
    
    vrooli_log_header "🔌 Running Integration Tests (${#INTEGRATION_TESTS[@]} files)"
    
    if [[ "${VROOLI_TEST_DRY_RUN:-false}" == "true" ]]; then
        for test in "${INTEGRATION_TESTS[@]}"; do
            echo "Would run: $test"
        done
        return 0
    fi
    
    # Run integration tests using the dedicated runner
    if "$RUNNER_DIR/run-integration.sh" \
        ${VERBOSE:+--verbose} \
        ${FAIL_FAST:+--fail-fast}; then
        vrooli_log_success "Integration tests completed successfully"
        return 0
    else
        vrooli_log_error "Integration tests failed"
        return 1
    fi
}

#######################################
# Run shell script tests
#######################################
run_shell_tests() {
    if [[ ${#SHELL_TESTS[@]} -eq 0 ]]; then
        vrooli_log_info "No shell tests to run"
        return 0
    fi
    
    vrooli_log_header "🐚 Running Shell Tests (${#SHELL_TESTS[@]} files)"
    
    if [[ "${VROOLI_TEST_DRY_RUN:-false}" == "true" ]]; then
        for test in "${SHELL_TESTS[@]}"; do
            echo "Would run: $test"
        done
        return 0
    fi
    
    # Use run-unit.sh for BATS files (shell tests are unit tests for shell scripts)
    # Build command arguments
    local cmd_args=()
    [[ "$PARALLEL_ENABLED" == "true" ]] && cmd_args+=(--parallel)
    [[ -n "$MAX_PARALLEL_JOBS" ]] && [[ "$PARALLEL_ENABLED" == "true" ]] && cmd_args+=(--jobs "$MAX_PARALLEL_JOBS")
    [[ "$VERBOSE" == "true" ]] && cmd_args+=(--verbose)
    [[ "$FAIL_FAST" == "true" ]] && cmd_args+=(--fail-fast)
    cmd_args+=(--timeout "$((60 * TIMEOUT_MULTIPLIER))")
    
    if "$RUNNER_DIR/run-unit.sh" "${cmd_args[@]}" "${SHELL_TESTS[@]}"; then
        vrooli_log_success "Shell tests completed successfully"
        return 0
    else
        vrooli_log_error "Shell tests failed"
        return 1
    fi
}

#######################################
# Generate test report
#######################################
generate_report() {
    local end_time=$(date +%s)
    local duration=$((end_time - START_TIME))
    local minutes=$((duration / 60))
    local seconds=$((duration % 60))
    
    vrooli_log_header "📊 Test Results Summary"
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Total Tests:    $TOTAL_TESTS"
    echo "  Passed:         $PASSED_TESTS ✅"
    echo "  Failed:         $FAILED_TESTS ❌"
    echo "  Skipped:        $SKIPPED_TESTS ⏭️"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Duration:       ${minutes}m ${seconds}s"
    echo "  Parallel:       $PARALLEL_ENABLED"
    if [[ "$PARALLEL_ENABLED" == "true" ]]; then
        echo "  Jobs:           $MAX_PARALLEL_JOBS"
    fi
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Generate coverage report if requested
    if [[ "$COVERAGE" == "true" ]]; then
        generate_coverage_report
    fi
    
    # Show isolation statistics
    echo ""
    vrooli_isolation_stats
}

#######################################
# Generate coverage report
#######################################
generate_coverage_report() {
    vrooli_log_info "Generating coverage report..."
    
    # TODO: Implement actual coverage generation
    # This is a placeholder for future implementation
    vrooli_log_warn "Coverage report generation not yet implemented"
}

#######################################
# Main execution
#######################################
main() {
    # Parse arguments
    parse_args "$@"
    
    # Initialize environment
    init_test_environment
    
    # Discover tests
    discover_tests
    
    if [[ $TOTAL_TESTS -eq 0 ]]; then
        vrooli_log_warn "No tests found matching criteria"
        exit 0
    fi
    
    # Track failures
    local failures=0
    
    # Run test suites
    if ! run_unit_tests; then
        ((failures++))
        if [[ "$FAIL_FAST" == "true" ]]; then
            vrooli_log_error "Stopping due to unit test failures (fail-fast mode)"
            exit 1
        fi
    fi
    
    if ! run_integration_tests; then
        ((failures++))
        if [[ "$FAIL_FAST" == "true" ]]; then
            vrooli_log_error "Stopping due to integration test failures (fail-fast mode)"
            exit 1
        fi
    fi
    
    if ! run_shell_tests; then
        ((failures++))
        if [[ "$FAIL_FAST" == "true" ]]; then
            vrooli_log_error "Stopping due to shell test failures (fail-fast mode)"
            exit 1
        fi
    fi
    
    # Update statistics (simplified for now)
    if [[ $failures -eq 0 ]]; then
        PASSED_TESTS=$TOTAL_TESTS
        FAILED_TESTS=0
    else
        # This is simplified - in reality we'd track per-test results
        FAILED_TESTS=$failures
        PASSED_TESTS=$((TOTAL_TESTS - failures))
    fi
    
    # Generate report
    generate_report
    
    # Exit with appropriate code
    if [[ $FAILED_TESTS -gt 0 ]]; then
        vrooli_log_error "Test suite failed with $FAILED_TESTS failures"
        exit 1
    else
        vrooli_log_success "🎉 All tests passed!"
        exit 0
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi