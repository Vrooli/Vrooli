#!/usr/bin/env bats
# Basic Test Template - Simple unit tests (30 lines)
# Usage: Copy this file and replace TODO sections

bats_require_minimum_version 1.5.0

# Load test infrastructure (single entry point)
source "$(dirname "$0")/../setup.bash"

setup() {
    vrooli_setup_unit_test  # Basic mocks and environment
}

teardown() {
    vrooli_cleanup_test     # Clean up resources
}

# Replace with your test
@test "TODO: describe what you're testing" {
    # Arrange - Set up test data
    local input="test data"
    local expected="expected result"
    
    # Act - Run the command/function
    run echo "$input"
    
    # Assert - Verify results
    assert_success
    assert_output_contains "test"
}

# Template: 30 lines | Copy and modify for your tests