#!/usr/bin/env bash
# BATS Execution Phase - Vrooli Testing Infrastructure
# 
# Executes all BATS tests with caching:
# - Discovers all .bats test files
# - Runs BATS tests with intelligent caching
# - Uses existing BATS infrastructure and helpers
# - Provides detailed reporting and parallel execution
# - Integrates with existing test setup and teardown

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_ROOT="${PROJECT_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"
# shellcheck disable=SC1091  
source "$SCRIPT_DIR/shared/test-helpers.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/cache.bash"

show_usage() {
    cat << 'EOF'
BATS Execution Phase - Run all BATS tests with caching optimization

Usage: ./test-bats.sh [OPTIONS]

OPTIONS:
  --verbose      Show detailed output for each test file
  --parallel     Run BATS tests in parallel (when possible)  
  --no-cache     Skip caching optimizations - force re-run all tests
  --dry-run      Show what tests would be run without executing
  --clear-cache  Clear BATS test cache before running
  --filter PATTERN  Only run tests matching pattern (file name or path)
  --help         Show this help

WHAT THIS PHASE TESTS:
  1. Discovers all .bats test files in the project
  2. Runs BATS tests with intelligent caching based on file modification time
  3. Uses existing BATS test infrastructure and setup/teardown
  4. Provides parallel execution for independent test files
  5. Integrates with existing mocks and test helpers
  6. Reports detailed results with timing information

EXAMPLES:
  ./test-bats.sh                           # Run all BATS tests
  ./test-bats.sh --verbose --parallel      # Run in parallel with details
  ./test-bats.sh --filter "resource"       # Only run tests with "resource" in path
  ./test-bats.sh --no-cache                # Force re-run all tests
  ./test-bats.sh --clear-cache             # Clear cache and re-run
EOF
}

# Parse command line arguments
CLEAR_CACHE=""
TEST_FILTER=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --clear-cache)
            CLEAR_CACHE="true"
            shift
            ;;
        --filter)
            TEST_FILTER="$2"
            shift 2
            ;;
        --help)
            show_usage
            exit 0
            ;;
        --verbose|--parallel|--no-cache|--dry-run)
            # These are handled by the main orchestrator
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Clear cache if requested
if [[ -n "$CLEAR_CACHE" ]]; then
    clear_cache "bats"
fi

# Check BATS availability and version
check_bats_availability() {
    log_info "🔧 Checking BATS availability..."
    
    if ! command -v bats >/dev/null 2>&1; then
        log_error "BATS not found - cannot run BATS tests"
        log_info "Install BATS:"
        log_info "  npm install -g bats                    # Via npm"
        log_info "  brew install bats-core                 # Via Homebrew"
        log_info "  apt-get install bats                   # Via apt"
        log_info "  git clone https://github.com/bats-core/bats-core.git && ./bats-core/install.sh /usr/local"
        return 1
    fi
    
    local bats_version
    bats_version=$(bats --version 2>/dev/null | head -n1 || echo "unknown version")
    is_verbose && log_success "BATS available: $bats_version"
    
    # Check for minimum version requirements
    if bats --version 2>/dev/null | grep -q "Bats 1\.[0-9]\+\.[0-9]\+"; then
        is_verbose && log_success "BATS version is compatible (1.x series)"
    else
        log_warning "BATS version may not be fully compatible - recommend 1.x series"
    fi
    
    return 0
}

# Discover BATS test files
discover_bats_tests() {
    log_info "🔍 Discovering BATS test files..."
    
    local test_files=()
    
    # Search for .bats files in common test locations
    local search_dirs=(
        "$PROJECT_ROOT/scripts/__test"
        "$PROJECT_ROOT/packages/*/src/__test"
        "$PROJECT_ROOT/test"
        "$PROJECT_ROOT/tests"
    )
    
    for search_dir in "${search_dirs[@]}"; do
        # Use glob expansion to handle wildcards in package paths
        for expanded_dir in $search_dir; do
            if [[ -d "$expanded_dir" ]]; then
                while IFS= read -r -d '' test_file; do
                    # Apply filter if specified
                    if [[ -n "$TEST_FILTER" ]]; then
                        if [[ "$test_file" =~ $TEST_FILTER ]]; then
                            test_files+=("$test_file")
                        fi
                    else
                        test_files+=("$test_file")
                    fi
                done < <(find "$expanded_dir" -type f -name "*.bats" -print0 2>/dev/null)
            fi
        done
    done
    
    # Remove duplicates and sort
    local unique_tests
    mapfile -t unique_tests < <(printf '%s\n' "${test_files[@]}" | sort -u)
    
    local total_tests=${#unique_tests[@]}
    
    if [[ $total_tests -eq 0 ]]; then
        if [[ -n "$TEST_FILTER" ]]; then
            log_warning "No BATS test files found matching filter: $TEST_FILTER"
        else
            log_warning "No BATS test files found in project"
        fi
        return 1
    fi
    
    log_info "📋 Found $total_tests BATS test files to execute"
    
    # Show breakdown by directory if verbose
    if is_verbose; then
        log_info "📁 BATS tests by directory:"
        printf '%s\n' "${unique_tests[@]}" | \
            sed "s|$PROJECT_ROOT/||" | \
            cut -d'/' -f1-2 | \
            sort | uniq -c | \
            while read -r count dir; do
                log_info "  $dir: $count test files"
            done
    fi
    
    # Store the discovered tests globally
    DISCOVERED_BATS_TESTS=("${unique_tests[@]}")
    
    return 0
}

# Run a single BATS test file with caching
run_bats_file() {
    local test_file="$1"
    local test_name
    test_name=$(relative_path "$test_file")
    
    # Check if we should skip due to caching
    if should_skip_test "$test_file" "bats"; then
        return 0
    fi
    
    log_test_start "$test_name"
    
    if is_dry_run; then
        log_info "[DRY RUN] Would run BATS test: $test_file"
        cache_test_result "$test_file" "bats" "skipped" "dry run"
        increment_test_counter "skipped"
        return 0
    fi
    
    # Prepare BATS environment
    local bats_args=()
    
    # Add verbose flag if requested
    if is_verbose; then
        bats_args+=("--verbose-run")
    fi
    
    # Add timing information
    bats_args+=("--timing")
    
    # Set up environment for the test
    export PROJECT_ROOT
    export BATS_TEST_DESCRIPTION="$test_name"
    
    # Run the BATS test
    local test_result="failed"
    local error_message=""
    local start_time
    start_time=$(date +%s)
    
    if timeout 300 bats "${bats_args[@]}" "$test_file" >/dev/null 2>&1; then
        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        test_result="passed"
        log_test_pass "$test_name" "(${duration}s)"
        increment_test_counter "passed"
    else
        local exit_code=$?
        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        if [[ $exit_code -eq 124 ]]; then
            error_message="timeout after 300s"
        else
            error_message="exit code: $exit_code"
        fi
        
        test_result="failed"
        log_test_fail "$test_name" "$error_message (${duration}s)"
        increment_test_counter "failed"
    fi
    
    # Cache the result
    cache_test_result "$test_file" "bats" "$test_result" "$error_message"
    
    # Clean up environment
    unset BATS_TEST_DESCRIPTION
    
    # Return appropriate exit code
    case "$test_result" in
        "passed"|"skipped")
            return 0
            ;;
        "failed")
            return 1
            ;;
    esac
}

# Run BATS tests in parallel
run_bats_parallel() {
    local test_files=("$@")
    local max_parallel="${BATS_PARALLEL_JOBS:-4}"
    
    log_info "🚀 Running BATS tests in parallel (max $max_parallel jobs)"
    
    # Create temporary directory for parallel job tracking
    local temp_dir
    temp_dir=$(mktemp -d)
    trap "rm -rf '$temp_dir'" EXIT
    
    local job_count=0
    local total_files=${#test_files[@]}
    local current=0
    
    for test_file in "${test_files[@]}"; do
        ((current++))
        
        # Wait for available slot if at max parallel jobs
        while [[ $job_count -ge $max_parallel ]]; do
            # Check for completed jobs
            for job_file in "$temp_dir"/*.job; do
                [[ -f "$job_file" ]] || continue
                
                local job_pid
                job_pid=$(basename "$job_file" .job)
                
                if ! kill -0 "$job_pid" 2>/dev/null; then
                    # Job completed, clean up
                    rm -f "$job_file"
                    ((job_count--))
                fi
            done
            
            # Small delay to avoid busy waiting
            sleep 0.1
        done
        
        # Start new job
        (
            run_bats_file "$test_file"
            local exit_code=$?
            echo "$exit_code" > "$temp_dir/result.$$"
            exit $exit_code
        ) &
        
        local job_pid=$!
        echo "$job_pid" > "$temp_dir/$job_pid.job"
        ((job_count++))
        
        log_progress "$current" "$total_files" "Starting BATS tests"
    done
    
    # Wait for all remaining jobs to complete
    log_info "⏳ Waiting for all BATS tests to complete..."
    wait
    
    # Collect results
    local parallel_failures=0
    for result_file in "$temp_dir"/result.*; do
        [[ -f "$result_file" ]] || continue
        local result_code
        result_code=$(<"$result_file")
        if [[ $result_code -ne 0 ]]; then
            ((parallel_failures++))
        fi
    done
    
    return $parallel_failures
}

# Run BATS tests sequentially
run_bats_sequential() {
    local test_files=("$@")
    local total_files=${#test_files[@]}
    local current=0
    local failures=0
    
    log_info "📝 Running BATS tests sequentially"
    
    for test_file in "${test_files[@]}"; do
        ((current++))
        log_progress "$current" "$total_files" "Running BATS tests"
        
        if ! run_bats_file "$test_file"; then
            ((failures++))
        fi
    done
    
    return $failures
}

# Main BATS execution function
run_bats_tests() {
    log_section "BATS Execution Phase"
    
    reset_test_counters
    reset_cache_stats
    
    # Load cache for this phase
    load_cache "bats"
    
    # Check BATS availability
    if ! check_bats_availability; then
        print_test_summary
        return 1
    fi
    
    # Discover BATS test files
    if ! discover_bats_tests; then
        print_test_summary
        return 1
    fi
    
    local test_files=("${DISCOVERED_BATS_TESTS[@]}")
    local total_files=${#test_files[@]}
    
    # Show filter information if used
    if [[ -n "$TEST_FILTER" ]]; then
        log_info "🔍 Using filter: $TEST_FILTER"
        log_info "📋 Matching files: $total_files"
    fi
    
    # Run tests based on parallel/sequential preference
    local execution_failures=0
    
    if [[ -n "${PARALLEL:-}" ]] && [[ $total_files -gt 1 ]]; then
        if ! run_bats_parallel "${test_files[@]}"; then
            execution_failures=$?
        fi
    else
        if ! run_bats_sequential "${test_files[@]}"; then
            execution_failures=$?
        fi
    fi
    
    # Save cache before printing results
    save_cache
    
    # Print final results
    print_test_summary
    
    # Show cache statistics if verbose
    if is_verbose; then
        log_info ""
        show_cache_stats
    fi
    
    # Additional BATS-specific summary
    log_info ""
    log_info "📊 BATS Execution Summary:"
    log_info "  Test files discovered: $total_files"
    log_info "  Execution failures: $execution_failures"
    if [[ -n "$TEST_FILTER" ]]; then
        log_info "  Filter applied: $TEST_FILTER"
    fi
    if [[ -n "${PARALLEL:-}" ]]; then
        log_info "  Execution mode: Parallel"
    else
        log_info "  Execution mode: Sequential"  
    fi
    
    # Return success if no failures
    local counters
    read -r total passed failed skipped <<< "$(get_test_counters)"
    
    if [[ $failed -eq 0 ]] && [[ $execution_failures -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

# Global array to store discovered tests
declare -a DISCOVERED_BATS_TESTS=()

# Main execution
main() {
    if is_dry_run; then
        log_info "🧪 [DRY RUN] BATS Execution Phase"
        log_info "Would discover all .bats files in:"
        log_info "  - $PROJECT_ROOT/scripts/__test"
        log_info "  - $PROJECT_ROOT/packages/*/src/__test"
        log_info "  - $PROJECT_ROOT/test"
        log_info "  - $PROJECT_ROOT/tests"
        if [[ -n "$TEST_FILTER" ]]; then
            log_info "Would apply filter: $TEST_FILTER"
        fi
        if [[ -n "${PARALLEL:-}" ]]; then
            log_info "Would run tests in parallel"
        fi
        
        # Still discover tests to show what would be run
        if discover_bats_tests; then
            local test_count=${#DISCOVERED_BATS_TESTS[@]}
            log_info "Found $test_count BATS test files that would be executed"
        fi
        
        return 0
    fi
    
    # Run the BATS tests
    if run_bats_tests; then
        log_success "✅ BATS execution phase completed successfully"
        return 0
    else
        log_error "❌ BATS execution phase completed with failures"
        return 1
    fi
}

# Export functions for testing
export -f run_bats_tests check_bats_availability discover_bats_tests
export -f run_bats_file run_bats_parallel run_bats_sequential

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi