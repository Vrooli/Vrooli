#!/usr/bin/env bash
# Run Shell Tests - Run only shell (resource) BATS tests
# Dedicated runner for shell script tests in the resources directory

set -euo pipefail

# Enable job control to handle signals properly
set -m

# Script directory
RUNNER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ROOT="$(dirname "$RUNNER_DIR")"
SCRIPTS_DIR="$(dirname "$TEST_ROOT")"

# Load shared utilities (but skip test isolation to avoid EXIT trap issues)
source "$TEST_ROOT/shared/logging.bash"
source "$TEST_ROOT/shared/utils.bash"
source "$TEST_ROOT/shared/config-simple.bash"
# NOTE: test-isolation.bash is NOT loaded to prevent EXIT trap interference

# Configuration
VERBOSE="${VROOLI_TEST_VERBOSE:-false}"
FAIL_FAST="${VROOLI_TEST_FAIL_FAST:-false}"
TIMEOUT="${VROOLI_TEST_TIMEOUT:-60}"
USE_CACHE="${VROOLI_TEST_USE_CACHE:-true}"
CACHE_TTL="${VROOLI_TEST_CACHE_TTL:-120}"
FORCE_RUN="${VROOLI_TEST_FORCE_RUN:-false}"
CACHE_ONLY="${VROOLI_TEST_CACHE_ONLY:-false}"

# Ensure VROOLI_LOG_LEVEL is set for child processes
export VROOLI_LOG_LEVEL="${VROOLI_LOG_LEVEL:-INFO}"

# Source cache module if available
CACHE_SCRIPT="$TEST_ROOT/cache/test-cache.sh"
if [[ -f "$CACHE_SCRIPT" && "$USE_CACHE" == "true" ]]; then
    source "$CACHE_SCRIPT"
    export VROOLI_TEST_CACHE_ENABLED="$USE_CACHE"
    export VROOLI_TEST_CACHE_TTL="$CACHE_TTL"
else
    USE_CACHE="false"
    [[ ! -f "$CACHE_SCRIPT" ]] && vrooli_log_debug "Cache script not found, caching disabled"
fi

# Flag to track if we should exit early due to interrupt
INTERRUPTED=false

# Signal handler for immediate exit on Ctrl+C
handle_interrupt() {
    echo ""
    vrooli_log_warn "Test execution interrupted by user (Ctrl+C)"
    INTERRUPTED=true
    
    # Kill any running background processes (timeout/bats)
    local jobs_count
    jobs_count=$(jobs -r | wc -l)
    if [[ $jobs_count -gt 0 ]]; then
        echo "Stopping running tests..."
        # Kill all jobs in the current process group
        jobs -p | xargs -r kill 2>/dev/null || true
        sleep 0.5
        # Force kill if still running
        jobs -p | xargs -r kill -9 2>/dev/null || true
    fi
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Test execution stopped by user"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    exit 130
}

# Set up signal trap for immediate interruption
trap handle_interrupt SIGINT SIGTERM

#######################################
# Print usage information
#######################################
print_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Run shell BATS tests from the resources directory.

OPTIONS:
    --verbose         Enable verbose output
    --fail-fast       Stop on first failure
    --timeout N       Test timeout in seconds (default: 60)
    --dry-run         Show what would be run without executing
    --no-cache        Disable test result caching
    --cache-ttl N     Cache TTL in minutes (default: 120)
    --force           Force run all tests, ignoring cache
    --cache-only      Show what would be skipped (dry run with cache)
    --clear-cache     Clear test cache before running
    -h, --help        Show this help message

EXAMPLES:
    $0                          # Run all shell tests (skip recently passed)
    $0 --verbose                # Run with verbose output
    $0 --fail-fast              # Stop on first failure
    $0 --force                  # Force run all tests
    $0 --no-cache               # Run without cache
    $0 --cache-only             # Preview what would be skipped
EOF
}

#######################################
# Parse command line arguments
#######################################
parse_args() {
    local dry_run=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
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
            --dry-run)
                dry_run=true
                shift
                ;;
            --no-cache)
                USE_CACHE="false"
                shift
                ;;
            --cache-ttl)
                CACHE_TTL="$2"
                shift 2
                ;;
            --force)
                FORCE_RUN="true"
                shift
                ;;
            --cache-only)
                CACHE_ONLY="true"
                shift
                ;;
            --clear-cache)
                if [[ "$USE_CACHE" == "true" ]] && declare -f cache::clear &>/dev/null; then
                    cache::clear
                    vrooli_log_success "Cache cleared"
                fi
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
    
    export VROOLI_TEST_DRY_RUN="$dry_run"
}

#######################################
# Main execution
#######################################
main() {
    # Parse arguments
    parse_args "$@"
    
    vrooli_log_header "🐚 Shell Test Runner"
    
    # Initialize configuration
    vrooli_config_init
    
    # Create a simple test namespace for identification (no traps)
    export TEST_NAMESPACE="vrooli_shell_tests_$$_$(date +%s)"
    vrooli_log_info "Test namespace: $TEST_NAMESPACE"
    
    # Discover shell test files from ALL of scripts/
    local -a shell_tests=()
    while IFS= read -r test_file; do
        shell_tests+=("$test_file")
    done < <(find "$SCRIPTS_DIR" -name "*.bats" -type f 2>/dev/null | sort)
    
    local total_tests=${#shell_tests[@]}
    
    # Count tests by top-level directory in scripts/
    local helpers_tests=$(find "$SCRIPTS_DIR/helpers" -name "*.bats" -type f 2>/dev/null | wc -l)
    local resources_tests=$(find "$SCRIPTS_DIR/resources" -name "*.bats" -type f 2>/dev/null | wc -l)
    local main_tests=$(find "$SCRIPTS_DIR/main" -name "*.bats" -type f 2>/dev/null | wc -l)
    local test_tests=$(find "$SCRIPTS_DIR/__test" -name "*.bats" -type f 2>/dev/null | wc -l)
    local package_tests=$(find "$SCRIPTS_DIR/package" -name "*.bats" -type f 2>/dev/null | wc -l)
    local other_tests=$(find "$SCRIPTS_DIR" -maxdepth 1 -name "*.bats" -type f 2>/dev/null | wc -l)
    
    # Also get resource subcategory breakdown for detail
    local ai_tests=$(find "$SCRIPTS_DIR/resources/ai" -name "*.bats" -type f 2>/dev/null | wc -l)
    local agents_tests=$(find "$SCRIPTS_DIR/resources/agents" -name "*.bats" -type f 2>/dev/null | wc -l)
    local automation_tests=$(find "$SCRIPTS_DIR/resources/automation" -name "*.bats" -type f 2>/dev/null | wc -l)
    local execution_tests=$(find "$SCRIPTS_DIR/resources/execution" -name "*.bats" -type f 2>/dev/null | wc -l)
    local search_tests=$(find "$SCRIPTS_DIR/resources/search" -name "*.bats" -type f 2>/dev/null | wc -l)
    local storage_tests=$(find "$SCRIPTS_DIR/resources/storage" -name "*.bats" -type f 2>/dev/null | wc -l)
    
    # Display test discovery summary
    vrooli_log_header "📊 Test Discovery Summary"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  TOTAL SHELL TESTS FOUND: $total_tests"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  By Top-Level Directory:"
    echo "    helpers/:    $helpers_tests tests"
    echo "    resources/:  $resources_tests tests"
    if [[ $resources_tests -gt 0 ]]; then
        echo "      ├─ AI:         $ai_tests"
        echo "      ├─ Agents:     $agents_tests"
        echo "      ├─ Automation: $automation_tests"
        echo "      ├─ Execution:  $execution_tests"
        echo "      ├─ Search:     $search_tests"
        echo "      └─ Storage:    $storage_tests"
    fi
    echo "    main/:       $main_tests tests"
    echo "    __test/:     $test_tests tests"
    echo "    package/:    $package_tests tests"
    echo "    Other:       $other_tests tests"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    
    if [[ $total_tests -eq 0 ]]; then
        vrooli_log_warn "No shell tests found"
        exit 0
    fi
    
    if [[ "$VROOLI_TEST_DRY_RUN" == "true" ]]; then
        vrooli_log_info "Dry run mode - would run these tests:"
        for test in "${shell_tests[@]}"; do
            echo "  $(basename "$test")"
        done
        exit 0
    fi
    
    # Run tests sequentially with proper error handling
    local passed=0
    local failed=0
    local skipped=0
    local -a skipped_tests=()
    local time_saved=0
    
    # Disable exit on error for test execution
    set +e
    
    for test_file in "${shell_tests[@]}"; do
        # Check for interruption before starting each test
        if [[ "$INTERRUPTED" == "true" ]]; then
            break
        fi
        
        local test_name
        test_name=$(basename "$test_file" .bats)
        
        # Check cache if enabled and not forcing
        if [[ "$USE_CACHE" == "true" && "$FORCE_RUN" != "true" ]] && declare -f cache::is_valid &>/dev/null; then
            if cache::is_valid "$test_file" "$CACHE_TTL"; then
                ((skipped++))
                skipped_tests+=("$test_name")
                time_saved=$((time_saved + 2))  # Estimate 2 seconds per test
                
                if [[ "$CACHE_ONLY" == "true" ]]; then
                    vrooli_log_info "Would skip (cached): $test_name ⏭️"
                elif [[ "$VERBOSE" == "true" ]]; then
                    vrooli_log_info "Skipping (cached): $test_name ⏭️"
                fi
                continue
            fi
        fi
        
        # If cache-only mode, just show what would run
        if [[ "$CACHE_ONLY" == "true" ]]; then
            vrooli_log_info "Would run: $test_name 🏃"
            continue
        fi
        
        if [[ "$VERBOSE" == "true" ]]; then
            vrooli_log_info "Running: $test_name"
        fi
        
        # Run test directly to preserve colors, but capture stderr to temp file for interrupt detection
        local temp_stderr=$(mktemp)
        local test_start_time=$(date +%s)
        timeout "$TIMEOUT" bats "$test_file" 2> >(tee "$temp_stderr" >&2)
        local test_exit_code=$?
        local test_duration=$(($(date +%s) - test_start_time))
        
        # Check for interrupt indicators if exit code is ambiguous
        if [[ $test_exit_code -eq 1 ]]; then
            if grep -q "Received SIGINT" "$temp_stderr" 2>/dev/null; then
                rm -f "$temp_stderr"
                vrooli_log_warn "Test interrupted by user (Ctrl+C)"
                INTERRUPTED=true
                break
            fi
        fi
        rm -f "$temp_stderr"
        
        # If interrupted flag was set by signal handler, break immediately
        if [[ "$INTERRUPTED" == "true" ]]; then
            vrooli_log_warn "Test interrupted by user (Ctrl+C)"
            break
        fi
        
        # Check if test was killed by signal or interrupted
        # Exit codes: 0=success, 1=failure, 124=timeout, 130=SIGINT, 143=SIGTERM
        if [[ $test_exit_code -eq 130 || $test_exit_code -eq 143 ]]; then
            vrooli_log_warn "Test interrupted by user (Ctrl+C)"
            INTERRUPTED=true
            break
        elif [[ $test_exit_code -eq 124 ]]; then
            ((failed++))
            vrooli_log_error "❌ $test_name (timeout after ${TIMEOUT}s)"
        elif [[ $test_exit_code -eq 0 ]]; then
            ((passed++))
            if [[ "$VERBOSE" == "true" ]]; then
                vrooli_log_success "✅ $test_name"
            fi
            # Store successful result in cache
            if [[ "$USE_CACHE" == "true" ]] && declare -f cache::store_result &>/dev/null; then
                cache::store_result "$test_file" "passed" "$test_duration"
            fi
        else
            ((failed++))
            vrooli_log_error "❌ $test_name (exit code: $test_exit_code)"
            # Store failed result in cache (to avoid re-running failed tests)
            if [[ "$USE_CACHE" == "true" ]] && declare -f cache::store_result &>/dev/null; then
                cache::store_result "$test_file" "failed" "$test_duration"
            fi
        fi
        
        # Check fail-fast condition (but not for interrupted tests)
        if [[ $test_exit_code -ne 0 && $test_exit_code -ne 130 && $test_exit_code -ne 143 && "$FAIL_FAST" == "true" && "$INTERRUPTED" != "true" ]]; then
            vrooli_log_error "Stopping due to test failure (fail-fast mode)"
            break
        fi
    done
    
    # Re-enable exit on error
    set -e
    
    # Report results
    if [[ "$CACHE_ONLY" == "true" ]]; then
        vrooli_log_header "📊 Cache Analysis Results"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "  Total Tests:      $total_tests"
        echo "  Would Skip:       $skipped ⏭️ (cached)"
        echo "  Would Run:        $((total_tests - skipped)) 🏃"
        echo "  Time Saved:       ~${time_saved}s"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        if [[ $skipped -gt 0 && "$VERBOSE" == "true" ]]; then
            echo ""
            echo "Tests that would be skipped:"
            for test in "${skipped_tests[@]}"; do
                echo "  - $test"
            done
        fi
    else
        vrooli_log_header "📊 Shell Test Results"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "  Total Tests:      $total_tests"
        echo "  Passed:           $passed ✅"
        echo "  Failed:           $failed ❌"
        if [[ $skipped -gt 0 ]]; then
            echo "  Skipped (cached): $skipped ⏭️"
            echo "  Time Saved:       ~${time_saved}s"
        fi
        if [[ "$INTERRUPTED" == "true" ]]; then
            local interrupted_count=$((total_tests - passed - failed - skipped))
            echo "  Interrupted:      $interrupted_count ⏸️"
        fi
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        # Show cache statistics if verbose and cache is enabled
        if [[ "$VERBOSE" == "true" && "$USE_CACHE" == "true" ]] && declare -f cache::stats &>/dev/null; then
            echo ""
            cache::stats
        fi
    fi
    
    # Exit with appropriate code
    if [[ "$CACHE_ONLY" == "true" ]]; then
        exit 0  # Cache analysis always succeeds
    elif [[ "$INTERRUPTED" == "true" ]]; then
        vrooli_log_warn "Shell tests interrupted by user"
        exit 130  # Standard exit code for SIGINT
    elif [[ $failed -gt 0 ]]; then
        vrooli_log_error "Shell tests failed with $failed failures"
        exit 1
    else
        if [[ $skipped -gt 0 ]]; then
            vrooli_log_success "🎉 All tests passed! ($skipped cached, ~${time_saved}s saved)"
        else
            vrooli_log_success "🎉 All shell tests passed!"
        fi
        exit 0
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi