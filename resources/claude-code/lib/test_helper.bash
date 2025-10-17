#!/usr/bin/env bash
# Test helper functions for Claude Code BATS tests

# Helper function to test without using 'run' for better variable control
test_without_run() {
    local func="$1"
    shift
    local output
    local status
    
    output=$("$func" "$@" 2>&1) || status=$?
    status=${status:-0}
    
    echo "$output"
    return $status
}

# Helper to test functions in a subshell with proper context
test_in_subshell() {
    local test_code="$1"
    local output
    local status=0
    
    output=$(eval "$test_code" 2>&1) || status=$?
    
    printf '%s' "$output"
    return $status
}

# Export test mode to allow source files to detect test environment
export TEST_MODE=true