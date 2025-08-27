#!/usr/bin/env bash
# ============================================================================
# Universal Contract Test Framework
# ============================================================================
#
# Test framework implementation for validating resources against the 
# universal contract (v2.0). This replaces the category-based validation
# with a unified approach that all resources must pass.
#
# This file provides the actual test implementations that resources
# can use in their lib/test.sh files.
# ============================================================================

set -euo pipefail

# Setup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../../.." && builtin pwd)}"
CONTRACTS_DIR="${APP_ROOT}/scripts/resources/contracts"
RESOURCES_DIR="${APP_ROOT}/resources"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}" 2>/dev/null || {
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::warning() { echo "[WARNING] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
}

# Test state
declare -g TEST_RESOURCE_DIR=""
declare -g TEST_RESOURCE_NAME=""
declare -g TEST_RESULTS=()
declare -g TEST_FAILURES=()

#######################################
# Initialize test framework for a resource
# Args: $1 - resource directory path
#######################################
test::init() {
    local resource_dir="${1:-$(pwd)}"
    TEST_RESOURCE_DIR="$resource_dir"
    TEST_RESOURCE_NAME="$(basename "$resource_dir")"
    TEST_RESULTS=()
    TEST_FAILURES=()
    
    log::info "Initializing tests for: $TEST_RESOURCE_NAME"
}

#######################################
# Run all test suites
# Returns: 0 if all pass, 1 if any fail
#######################################
test::all() {
    log::info "Running all test suites for $TEST_RESOURCE_NAME"
    
    local all_passed=true
    
    # Run test suites in order
    if ! test::smoke; then
        all_passed=false
    fi
    
    if ! test::unit; then
        all_passed=false
    fi
    
    if ! test::integration; then
        all_passed=false
    fi
    
    # Report results
    test::report
    
    [[ "$all_passed" == "true" ]] && return 0 || return 1
}

#######################################
# Smoke test - quick health check
# Returns: 0 if pass, 1 if fail
#######################################
test::smoke() {
    log::info "Running smoke tests..."
    
    local passed=true
    
    # Check CLI exists and is executable
    if [[ -f "$TEST_RESOURCE_DIR/cli.sh" ]] && [[ -x "$TEST_RESOURCE_DIR/cli.sh" ]]; then
        TEST_RESULTS+=("PASS: cli.sh exists and is executable")
    else
        TEST_FAILURES+=("FAIL: cli.sh not found or not executable")
        passed=false
    fi
    
    # Check help command works
    if "$TEST_RESOURCE_DIR/cli.sh" help >/dev/null 2>&1; then
        TEST_RESULTS+=("PASS: help command works")
    else
        TEST_FAILURES+=("FAIL: help command failed")
        passed=false
    fi
    
    # Check status command exists
    if "$TEST_RESOURCE_DIR/cli.sh" status >/dev/null 2>&1 || [ $? -eq 2 ]; then
        TEST_RESULTS+=("PASS: status command responds")
    else
        TEST_FAILURES+=("FAIL: status command error")
        passed=false
    fi
    
    [[ "$passed" == "true" ]] && return 0 || return 1
}

#######################################
# Unit tests - test individual functions
# Returns: 0 if pass, 1 if fail, 2 if no tests
#######################################
test::unit() {
    log::info "Running unit tests..."
    
    # Check if resource has unit tests
    if [[ -f "$TEST_RESOURCE_DIR/lib/test.sh" ]]; then
        # Source and run unit tests
        # shellcheck disable=SC1090
        source "$TEST_RESOURCE_DIR/lib/test.sh"
        
        # Look for test functions
        local test_functions=()
        while IFS= read -r func; do
            test_functions+=("$func")
        done < <(declare -F | grep "test_" | awk '{print $3}')
        
        if [[ ${#test_functions[@]} -eq 0 ]]; then
            TEST_RESULTS+=("SKIP: No unit test functions found")
            return 2
        fi
        
        local passed=true
        for test_func in "${test_functions[@]}"; do
            if $test_func 2>/dev/null; then
                TEST_RESULTS+=("PASS: $test_func")
            else
                TEST_FAILURES+=("FAIL: $test_func")
                passed=false
            fi
        done
        
        [[ "$passed" == "true" ]] && return 0 || return 1
    else
        TEST_RESULTS+=("SKIP: No unit tests (lib/test.sh not found)")
        return 2
    fi
}

#######################################
# Integration tests - test resource integration
# Returns: 0 if pass, 1 if fail, 2 if no tests
#######################################
test::integration() {
    log::info "Running integration tests..."
    
    # Check for integration test file
    if [[ -f "$TEST_RESOURCE_DIR/test/integration-test.sh" ]]; then
        if bash "$TEST_RESOURCE_DIR/test/integration-test.sh" 2>/dev/null; then
            TEST_RESULTS+=("PASS: Integration tests passed")
            return 0
        else
            TEST_FAILURES+=("FAIL: Integration tests failed")
            return 1
        fi
    else
        TEST_RESULTS+=("SKIP: No integration tests found")
        return 2
    fi
}

#######################################
# Test command structure compliance
# Returns: 0 if compliant, 1 if not
#######################################
test::command_structure() {
    log::info "Testing command structure compliance..."
    
    local cli_file="$TEST_RESOURCE_DIR/cli.sh"
    if [[ ! -f "$cli_file" ]]; then
        TEST_FAILURES+=("FAIL: cli.sh not found")
        return 1
    fi
    
    # Required commands per v2.0 contract
    local required_commands=(
        "help"
        "status"
        "logs"
        "manage"
        "test"
        "content"
    )
    
    local passed=true
    for cmd in "${required_commands[@]}"; do
        if grep -q "cli::register_command.*[\"']${cmd}[\"']" "$cli_file" 2>/dev/null; then
            TEST_RESULTS+=("PASS: Command registered: $cmd")
        else
            TEST_FAILURES+=("FAIL: Missing command: $cmd")
            passed=false
        fi
    done
    
    [[ "$passed" == "true" ]] && return 0 || return 1
}

#######################################
# Test file structure compliance
# Returns: 0 if compliant, 1 if not
#######################################
test::file_structure() {
    log::info "Testing file structure compliance..."
    
    # Required files per v2.0 contract
    local required_files=(
        "cli.sh"
        "lib/core.sh"
        "lib/test.sh"
        "config/defaults.sh"
    )
    
    local passed=true
    for file in "${required_files[@]}"; do
        if [[ -f "$TEST_RESOURCE_DIR/$file" ]]; then
            TEST_RESULTS+=("PASS: Required file exists: $file")
        else
            TEST_FAILURES+=("FAIL: Missing required file: $file")
            passed=false
        fi
    done
    
    # Check for deprecated files
    local deprecated_files=(
        "manage.sh"
        "manage.bats"
        "inject.sh"
    )
    
    for file in "${deprecated_files[@]}"; do
        if [[ -f "$TEST_RESOURCE_DIR/$file" ]]; then
            TEST_FAILURES+=("WARN: Deprecated file found: $file")
        fi
    done
    
    [[ "$passed" == "true" ]] && return 0 || return 1
}

#######################################
# Test manage subcommands
# Returns: 0 if all work, 1 if any fail
#######################################
test::manage_commands() {
    log::info "Testing manage subcommands..."
    
    local cli="$TEST_RESOURCE_DIR/cli.sh"
    if [[ ! -x "$cli" ]]; then
        TEST_FAILURES+=("FAIL: cli.sh not executable")
        return 1
    fi
    
    local passed=true
    
    # Test manage subcommands exist (don't actually run them)
    for subcmd in install uninstall start stop restart; do
        if "$cli" manage "$subcmd" --help 2>/dev/null || \
           "$cli" manage "$subcmd" --dry-run 2>/dev/null || \
           grep -q "manage.*$subcmd" "$cli" 2>/dev/null; then
            TEST_RESULTS+=("PASS: manage $subcmd available")
        else
            TEST_FAILURES+=("FAIL: manage $subcmd not found")
            passed=false
        fi
    done
    
    [[ "$passed" == "true" ]] && return 0 || return 1
}

#######################################
# Test content management commands
# Returns: 0 if all work, 1 if any fail
#######################################
test::content_commands() {
    log::info "Testing content commands..."
    
    local cli="$TEST_RESOURCE_DIR/cli.sh"
    if [[ ! -x "$cli" ]]; then
        TEST_FAILURES+=("FAIL: cli.sh not executable")
        return 1
    fi
    
    local passed=true
    
    # Test content subcommands
    for subcmd in add list get remove; do
        if "$cli" content "$subcmd" --help 2>/dev/null || \
           grep -q "content.*$subcmd" "$cli" 2>/dev/null; then
            TEST_RESULTS+=("PASS: content $subcmd available")
        else
            TEST_FAILURES+=("FAIL: content $subcmd not found")
            passed=false
        fi
    done
    
    [[ "$passed" == "true" ]] && return 0 || return 1
}

#######################################
# Test exit codes match contract
# Returns: 0 if compliant, 1 if not
#######################################
test::exit_codes() {
    log::info "Testing exit code compliance..."
    
    local cli="$TEST_RESOURCE_DIR/cli.sh"
    if [[ ! -x "$cli" ]]; then
        TEST_FAILURES+=("FAIL: cli.sh not executable")
        return 1
    fi
    
    # Test help command returns 0
    if "$cli" help >/dev/null 2>&1; then
        TEST_RESULTS+=("PASS: help returns 0")
    else
        TEST_FAILURES+=("FAIL: help didn't return 0")
    fi
    
    # Test status returns 0, 1, or 2
    local status_code
    "$cli" status >/dev/null 2>&1 || status_code=$?
    if [[ $status_code -le 2 ]]; then
        TEST_RESULTS+=("PASS: status returns valid code ($status_code)")
    else
        TEST_FAILURES+=("FAIL: status returned invalid code ($status_code)")
    fi
    
    return 0
}

#######################################
# Generate test report
#######################################
test::report() {
    local total_tests=$((${#TEST_RESULTS[@]} + ${#TEST_FAILURES[@]}))
    local passed=${#TEST_RESULTS[@]}
    local failed=${#TEST_FAILURES[@]}
    
    echo
    echo "═══════════════════════════════════════════════════════════════════"
    echo "Test Results for: $TEST_RESOURCE_NAME"
    echo "═══════════════════════════════════════════════════════════════════"
    echo "Total: $total_tests | Passed: $passed | Failed: $failed"
    echo
    
    if [[ ${#TEST_RESULTS[@]} -gt 0 ]]; then
        echo "Passed Tests:"
        for result in "${TEST_RESULTS[@]}"; do
            echo "  ✅ $result"
        done
        echo
    fi
    
    if [[ ${#TEST_FAILURES[@]} -gt 0 ]]; then
        echo "Failed Tests:"
        for failure in "${TEST_FAILURES[@]}"; do
            if [[ "$failure" == WARN:* ]]; then
                echo "  ⚠️  ${failure#WARN: }"
            else
                echo "  ❌ ${failure#FAIL: }"
            fi
        done
        echo
    fi
    
    echo "═══════════════════════════════════════════════════════════════════"
    
    if [[ $failed -eq 0 ]]; then
        log::success "All tests passed! ✅"
    else
        log::error "Some tests failed ❌"
    fi
}

#######################################
# Helper: Check if command exists in CLI
# Args: $1 - command name
# Returns: 0 if exists, 1 if not
#######################################
test::command_exists() {
    local cmd="$1"
    local cli="$TEST_RESOURCE_DIR/cli.sh"
    
    grep -q "cli::register_command.*[\"']${cmd}[\"']" "$cli" 2>/dev/null
}

#######################################
# Helper: Run command and check exit code
# Args: $1 - command, $2 - expected exit codes (regex)
# Returns: 0 if matches, 1 if not
#######################################
test::check_exit_code() {
    local cmd="$1"
    local expected="$2"
    local cli="$TEST_RESOURCE_DIR/cli.sh"
    
    local exit_code=0
    timeout 10 "$cli" $cmd >/dev/null 2>&1 || exit_code=$?
    
    [[ "$exit_code" =~ $expected ]]
}

# Export functions for use in resource tests
export -f test::init
export -f test::all
export -f test::smoke
export -f test::unit
export -f test::integration
export -f test::command_structure
export -f test::file_structure
export -f test::manage_commands
export -f test::content_commands
export -f test::exit_codes
export -f test::report
export -f test::command_exists
export -f test::check_exit_code