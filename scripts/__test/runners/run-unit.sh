#!/usr/bin/env bash
# Unit Test Runner - Run only BATS unit tests
# Focused runner for unit tests with mocking support

set -euo pipefail

# Script directory
RUNNER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="$(dirname "$RUNNER_DIR")"

# Load shared utilities
source "$TEST_ROOT/shared/logging.bash"
source "$TEST_ROOT/shared/utils.bash"
source "$TEST_ROOT/shared/config-simple.bash"
source "$TEST_ROOT/shared/test-isolation.bash"
source "$TEST_ROOT/shared/port-manager.bash"

# Source trash system for safe removal
# shellcheck disable=SC1091
source "${TEST_ROOT}/../lib/system/trash.sh" 2>/dev/null || true

# Configuration
PARALLEL_ENABLED="${VROOLI_TEST_PARALLEL:-false}"
MAX_PARALLEL_JOBS="${VROOLI_TEST_MAX_PARALLEL:-4}"
VERBOSE="${VROOLI_TEST_VERBOSE:-false}"
FAIL_FAST="${VROOLI_TEST_FAIL_FAST:-false}"
TIMEOUT="${VROOLI_TEST_UNIT_TIMEOUT:-30}"

# Test tracking
declare -a TEST_RESULTS=()
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

#######################################
# Print usage information
#######################################
print_usage() {
    cat << EOF
Usage: $0 [OPTIONS] [TEST_FILES...]

Run BATS unit tests with mocking support.

OPTIONS:
    --parallel         Enable parallel execution
    --jobs N          Number of parallel jobs (default: 4)
    --verbose         Enable verbose output
    --fail-fast       Stop on first failure
    --timeout N       Test timeout in seconds (default: 30)
    --filter PATTERN  Filter tests by pattern
    --list            List tests without running
    -h, --help        Show this help message

EXAMPLES:
    $0                           # Run all unit tests
    $0 test_file.bats           # Run specific test file
    $0 --parallel --jobs 8      # Run with 8 parallel jobs
    $0 --filter "service"       # Run only service-related tests

EOF
}

#######################################
# Parse command line arguments
#######################################
parse_args() {
    local test_files=()
    local filter_pattern=""
    local list_only=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --parallel)
                PARALLEL_ENABLED="true"
                shift
                ;;
            --jobs)
                MAX_PARALLEL_JOBS="$2"
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
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --filter)
                filter_pattern="$2"
                shift 2
                ;;
            --list)
                list_only=true
                shift
                ;;
            -h|--help)
                print_usage
                exit 0
                ;;
            *.bats)
                test_files+=("$1")
                shift
                ;;
            *)
                vrooli_log_error "UNIT" "Unknown option: $1"
                print_usage
                exit 1
                ;;
        esac
    done
    
    # Export for use in other functions
    export VROOLI_TEST_FILES=("${test_files[@]}")
    export VROOLI_TEST_FILTER="$filter_pattern"
    export VROOLI_TEST_LIST_ONLY="$list_only"
}

#######################################
# Discover unit test files
#######################################
discover_tests() {
    local -a discovered_tests=()
    
    # If specific files provided, use those
    if [[ ${#VROOLI_TEST_FILES[@]} -gt 0 ]]; then
        for test_file in "${VROOLI_TEST_FILES[@]}"; do
            if [[ -f "$test_file" ]]; then
                discovered_tests+=("$test_file")
            else
                vrooli_log_warn "UNIT" "Test file not found: $test_file"
            fi
        done
    else
        # Discover all unit tests
        local test_dirs=(
            "$TEST_ROOT/fixtures/tests"
            "$TEST_ROOT/unit"
        )
        
        for dir in "${test_dirs[@]}"; do
            if [[ -d "$dir" ]]; then
                while IFS= read -r test_file; do
                    # Apply filter if specified
                    if [[ -n "$VROOLI_TEST_FILTER" ]]; then
                        if [[ ! "$test_file" =~ $VROOLI_TEST_FILTER ]]; then
                            continue
                        fi
                    fi
                    discovered_tests+=("$test_file")
                done < <(find "$dir" -name "*.bats" -type f 2>/dev/null | sort)
            fi
        done
    fi
    
    TOTAL_TESTS=${#discovered_tests[@]}
    
    if [[ "$VROOLI_TEST_LIST_ONLY" == "true" ]]; then
        vrooli_log_info "UNIT" "Unit tests found: $TOTAL_TESTS"
        for test in "${discovered_tests[@]}"; do
            echo "  $(basename "$test")"
        done
        exit 0
    fi
    
    vrooli_log_info "UNIT" "Found $TOTAL_TESTS unit test files"
    
    # Export for parallel execution
    export UNIT_TEST_FILES=("${discovered_tests[@]}")
}

#######################################
# Run a single test file
#######################################
run_single_test() {
    local test_file="$1"
    local test_name
    test_name=$(basename "$test_file" .bats)
    
    if [[ "$VERBOSE" == "true" ]]; then
        vrooli_log_info "UNIT" "Running: $test_name"
    fi
    
    # Create isolated environment for this test
    local test_namespace
    test_namespace=$(vrooli_isolation_init "$test_name")
    
    # Set up test-specific environment
    export BATS_TEST_FILENAME="$test_file"
    export VROOLI_TEST_MODE="unit"
    
    # Run the test with timeout
    local output_file="/tmp/${test_namespace}_output.txt"
    local exit_code=0
    
    if timeout "$TIMEOUT" bats "$test_file" > "$output_file" 2>&1; then
        exit_code=0
        if [[ "$VERBOSE" == "true" ]]; then
            cat "$output_file"
        fi
    else
        exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            vrooli_log_error "UNIT" "Test timeout: $test_name (>${TIMEOUT}s)"
        else
            vrooli_log_error "UNIT" "Test failed: $test_name (exit: $exit_code)"
        fi
        
        if [[ "$VERBOSE" == "true" || "$FAIL_FAST" == "true" ]]; then
            cat "$output_file"
        fi
    fi
    
    # Clean up
    rm -f "$output_file"
    vrooli_isolation_cleanup
    
    return $exit_code
}

#######################################
# Run tests in parallel
#######################################
run_parallel_tests() {
    vrooli_log_info "UNIT" "Running tests in parallel (max $MAX_PARALLEL_JOBS jobs)"
    
    # Create results directory
    local results_dir="/tmp/vrooli_unit_test_results_$$"
    mkdir -p "$results_dir"
    
    # Run tests in parallel using xargs
    printf '%s\n' "${UNIT_TEST_FILES[@]}" | \
        xargs -P "$MAX_PARALLEL_JOBS" -I {} bash -c '
            test_file="$1"
            results_dir="$2"
            timeout="$3"
            verbose="$4"
            
            test_name=$(basename "$test_file" .bats)
            result_file="$results_dir/$test_name.result"
            
            # Run test and capture result
            if timeout "$timeout" bats "$test_file" > "$result_file.out" 2>&1; then
                echo "PASS" > "$result_file"
                [[ "$verbose" == "true" ]] && echo "✅ $test_name"
            else
                exit_code=$?
                echo "FAIL:$exit_code" > "$result_file"
                [[ "$verbose" == "true" ]] && echo "❌ $test_name"
                
                # Save failure output
                cp "$result_file.out" "$result_file.error"
            fi
        ' _ {} "$results_dir" "$TIMEOUT" "$VERBOSE"
    
    # Collect results
    for test_file in "${UNIT_TEST_FILES[@]}"; do
        local test_name
        test_name=$(basename "$test_file" .bats)
        local result_file="$results_dir/$test_name.result"
        
        if [[ -f "$result_file" ]]; then
            local result
            result=$(cat "$result_file")
            
            if [[ "$result" == "PASS" ]]; then
                ((PASSED_TESTS++))
            else
                ((FAILED_TESTS++))
                
                # Show error if verbose or fail-fast
                if [[ "$VERBOSE" == "true" || "$FAIL_FAST" == "true" ]]; then
                    if [[ -f "$result_file.error" ]]; then
                        vrooli_log_error "UNIT" "Error output for $test_name:"
                        cat "$result_file.error"
                    fi
                fi
                
                # Stop if fail-fast
                if [[ "$FAIL_FAST" == "true" ]]; then
                    vrooli_log_error "UNIT" "Stopping due to test failure (fail-fast mode)"
                    if command -v trash::safe_remove >/dev/null 2>&1; then
                        trash::safe_remove "$results_dir" --no-confirm
                    else
                        rm -rf "$results_dir"
                    fi
                    return 1
                fi
            fi
        else
            vrooli_log_warn "UNIT" "No result for test: $test_name"
            ((FAILED_TESTS++))
        fi
    done
    
    # Clean up results directory
    if command -v trash::safe_remove >/dev/null 2>&1; then
        trash::safe_remove "$results_dir" --no-confirm
    else
        rm -rf "$results_dir"
    fi
    
    return 0
}

#######################################
# Run tests sequentially
#######################################
run_sequential_tests() {
    vrooli_log_info "UNIT" "Running tests sequentially"
    
    for test_file in "${UNIT_TEST_FILES[@]}"; do
        if run_single_test "$test_file"; then
            ((PASSED_TESTS++))
            vrooli_log_success "✅ $(basename "$test_file" .bats)"
        else
            ((FAILED_TESTS++))
            vrooli_log_error "UNIT" "❌ $(basename "$test_file" .bats)"
            
            if [[ "$FAIL_FAST" == "true" ]]; then
                vrooli_log_error "Stopping due to test failure (fail-fast mode)"
                return 1
            fi
        fi
        
        # Progress update
        local completed=$((PASSED_TESTS + FAILED_TESTS))
        if [[ $((completed % 10)) -eq 0 && $completed -gt 0 ]]; then
            local progress=$((completed * 100 / TOTAL_TESTS))
            vrooli_log_info "UNIT" "Progress: $completed/$TOTAL_TESTS ($progress%)"
        fi
    done
    
    return 0
}

#######################################
# Generate test summary
#######################################
generate_summary() {
    local success_rate=0
    if [[ $TOTAL_TESTS -gt 0 ]]; then
        success_rate=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    fi
    
    vrooli_log_header "Unit Test Summary"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Total:    $TOTAL_TESTS"
    echo "  Passed:   $PASSED_TESTS ✅"
    echo "  Failed:   $FAILED_TESTS ❌"
    echo "  Success:  ${success_rate}%"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ $FAILED_TESTS -gt 0 ]]; then
        vrooli_log_error "UNIT" "Unit tests failed"
        return 1
    else
        vrooli_log_success "All unit tests passed!"
        return 0
    fi
}

#######################################
# Main execution
#######################################
main() {
    # Parse arguments
    parse_args "$@"
    
    # Initialize
    vrooli_log_header "🧪 Unit Test Runner"
    vrooli_config_init
    
    # Set up global test isolation
    vrooli_isolation_init "unit-test-suite"
    vrooli_isolation_setup_traps
    
    # Discover tests
    discover_tests
    
    if [[ $TOTAL_TESTS -eq 0 ]]; then
        vrooli_log_warn "UNIT" "No unit tests found"
        exit 0
    fi
    
    # Run tests
    if [[ "$PARALLEL_ENABLED" == "true" && $TOTAL_TESTS -gt 1 ]]; then
        run_parallel_tests
    else
        run_sequential_tests
    fi
    
    # Generate summary
    generate_summary
    exit $?
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi