#!/bin/bash
# OpenRouter benchmark functionality tests

# Set error handling
set +e  # Don't exit on error for tests

# Get directories
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Source test framework
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true

# Initialize test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name="$1"
    local command="$2"
    local expected_exit="${3:-0}"
    
    echo -n "Testing $test_name... "
    
    # Run command and capture output
    output=$(eval "$command" 2>&1)
    exit_code=$?
    
    if [[ $exit_code -eq $expected_exit ]]; then
        echo -e "\033[32m✓\033[0m"
        ((TESTS_PASSED++))
    else
        echo -e "\033[31m✗\033[0m"
        echo "  Expected exit code: $expected_exit, got: $exit_code"
        echo "  Output: $output"
        ((TESTS_FAILED++))
    fi
}

# Run benchmark tests
log::info "Running OpenRouter benchmark tests..."

# Test benchmark commands exist
run_test "Benchmark help" "resource-openrouter help | grep -q benchmark"
run_test "Benchmark list (empty)" "resource-openrouter benchmark list"
run_test "Benchmark single model" "resource-openrouter benchmark test 'openai/gpt-3.5-turbo' | grep -q 'success'"
run_test "Benchmark comparison" "resource-openrouter benchmark compare >/dev/null 2>&1"

# Test benchmark data was created
run_test "Benchmark data directory created" "test -d \${VROOLI_ROOT:-\$HOME/Vrooli}/data/openrouter/benchmarks"
run_test "Benchmark results file exists" "ls \${VROOLI_ROOT:-\$HOME/Vrooli}/data/openrouter/benchmarks/*.json >/dev/null 2>&1"

# Clean up test data
rm -rf "${VROOLI_ROOT:-$HOME/Vrooli}/data/openrouter/benchmarks"/*.json 2>/dev/null

# Display results
echo ""
log::info "Test Results: \033[32m$TESTS_PASSED passed\033[0m, \033[31m$TESTS_FAILED failed\033[0m"

# Exit with appropriate code
if [[ $TESTS_FAILED -gt 0 ]]; then
    exit 1
fi

exit 0