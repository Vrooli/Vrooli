#!/usr/bin/env bash
# Run tests for Huginn resource management scripts
set -euo pipefail

# Get the directory of this script using unique directory variable
HUGINN_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$HUGINN_LIB_DIR"

# Colors for output
readonly COLOR_RED='\033[0;31m'
readonly COLOR_GREEN='\033[0;32m'
readonly COLOR_YELLOW='\033[1;33m'
readonly COLOR_BLUE='\033[0;34m'
readonly COLOR_RESET='\033[0m'

# Test results
TESTS_PASSED=0
TESTS_FAILED=0
FAILED_TESTS=()

# Print colored output
print_header() {
    echo -e "\n${COLOR_BLUE}=== $1 ===${COLOR_RESET}\n"
}

print_success() {
    echo -e "${COLOR_GREEN}✓ $1${COLOR_RESET}"
}

print_failure() {
    echo -e "${COLOR_RED}✗ $1${COLOR_RESET}"
}

print_info() {
    echo -e "${COLOR_YELLOW}→ $1${COLOR_RESET}"
}

# Check if bats is installed
check_bats() {
    if ! command -v bats >/dev/null 2>&1; then
        echo -e "${COLOR_RED}Error: bats is not installed${COLOR_RESET}"
        echo "Please install bats-core:"
        echo "  Ubuntu/Debian: sudo apt-get install bats"
        echo "  macOS: brew install bats-core"
        echo "  Or visit: https://github.com/bats-core/bats-core"
        exit 1
    fi
}

# Run a single test file
run_test() {
    local test_file="$1"
    local test_name="$(basename "$test_file" .bats)"
    
    print_info "Running $test_name tests..."
    
    if bats "$test_file"; then
        print_success "$test_name tests passed"
        ((TESTS_PASSED++))
    else
        print_failure "$test_name tests failed"
        ((TESTS_FAILED++))
        FAILED_TESTS+=("$test_name")
    fi
}

# Run all tests
run_all_tests() {
    print_header "Huginn Resource Management Tests"
    
    # Check for bats
    check_bats
    
    # Export test mode to prevent actual Docker operations
    export HUGINN_TEST_MODE=true
    
    # Run manage.sh tests
    if [[ -f "manage.bats" ]]; then
        run_test "manage.bats"
    else
        print_info "Skipping manage.bats (not found)"
    fi
    
    # Run config tests
    for test_file in config/*.bats; do
        if [[ -f "$test_file" ]]; then
            run_test "$test_file"
        fi
    done
    
    # Run lib tests
    for test_file in lib/*.bats; do
        if [[ -f "$test_file" ]]; then
            run_test "$test_file"
        fi
    done
    
    # Print summary
    print_header "Test Summary"
    
    local total=$((TESTS_PASSED + TESTS_FAILED))
    echo "Total test files: $total"
    echo -e "Passed: ${COLOR_GREEN}$TESTS_PASSED${COLOR_RESET}"
    echo -e "Failed: ${COLOR_RED}$TESTS_FAILED${COLOR_RESET}"
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "\n${COLOR_RED}Failed tests:${COLOR_RESET}"
        for failed in "${FAILED_TESTS[@]}"; do
            echo "  - $failed"
        done
        exit 1
    else
        echo -e "\n${COLOR_GREEN}All tests passed!${COLOR_RESET}"
    fi
}

# Handle command line arguments
case "${1:-all}" in
    all)
        run_all_tests
        ;;
    manage)
        run_test "manage.bats"
        ;;
    config)
        for test_file in config/*.bats; do
            [[ -f "$test_file" ]] && run_test "$test_file"
        done
        ;;
    lib)
        for test_file in lib/*.bats; do
            [[ -f "$test_file" ]] && run_test "$test_file"
        done
        ;;
    *)
        # Try to run specific test file
        if [[ -f "$1" ]]; then
            run_test "$1"
        elif [[ -f "$1.bats" ]]; then
            run_test "$1.bats"
        elif [[ -f "lib/$1.bats" ]]; then
            run_test "lib/$1.bats"
        elif [[ -f "config/$1.bats" ]]; then
            run_test "config/$1.bats"
        else
            echo "Unknown test target: $1"
            echo "Usage: $0 [all|manage|config|lib|<test-file>]"
            exit 1
        fi
        ;;
esac