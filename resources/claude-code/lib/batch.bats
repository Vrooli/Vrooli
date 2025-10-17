#!/usr/bin/env bats
# Tests for Claude Code batch.sh functions
bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure
source "${APP_ROOT}/__test/fixtures/setup.bash"

setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "claude-code-batch"
    
    # Load dependencies
    SCRIPT_DIR="${APP_ROOT}/resources/claude-code"
    
    # shellcheck disable=SC1091
    source "${APP_ROOT}/lib/utils/var.sh" || true
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh" || true
    
    # Load configuration if available
    if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
        source "${SCRIPT_DIR}/config/defaults.sh"
    fi
    
    # Load the batch functions
    source "${BATS_TEST_DIRNAME}/batch.sh"
    
    # Export functions for subshells
    export -f claude_code::batch_simple
    export -f claude_code::batch_automation
    export -f claude_code::batch_config
    export -f claude_code::batch_multi
    export -f claude_code::batch_parallel
    export -f claude_code::batch
}

setup() {
    # Reset mocks for each test
    mock::claude::reset >/dev/null 2>&1 || true
    
    # Create temporary test files
    export TEST_BATCH_DIR="${BATS_TMPDIR}/batch-test"
    mkdir -p "$TEST_BATCH_DIR"
}

teardown() {
    # Clean up test files
    trash::safe_remove "$TEST_BATCH_DIR" --test-cleanup
}

#######################################
# Batch Simple Tests
#######################################

@test "claude_code::batch_simple - should require prompt and total turns" {
    run claude_code::batch_simple "" ""
    
    assert_failure
    assert_output --partial "Prompt and total turns are required"
}

@test "claude_code::batch_simple - should validate total turns as number" {
    run claude_code::batch_simple "test prompt" "not_a_number"
    
    assert_failure
    assert_output --partial "Total turns must be a number"
}

@test "claude_code::batch_simple - should execute with valid parameters" {
    # Mock successful execution
    mock::claude::scenario::setup_development
    
    run claude_code::batch_simple "Create a function" "5" "10" "Read,Write"
    
    assert_success
    assert_output --partial "Starting batch processing"
}

@test "claude_code::batch_simple - should use default batch size" {
    # Mock successful execution
    mock::claude::scenario::setup_development
    
    run claude_code::batch_simple "test prompt" "5"
    
    assert_success
    # Should use default batch size of 50
    assert_output --partial "batch size: 50"
}

#######################################
# Batch Automation Tests
#######################################

@test "claude_code::batch_automation - should execute automation batch" {
    # Mock successful execution
    mock::claude::scenario::setup_development
    
    run claude_code::batch_automation "Automate testing" "10" "20"
    
    assert_success
    assert_output --partial "batch automation"
}

@test "claude_code::batch_automation - should handle authentication errors" {
    # Mock authentication error
    mock::claude::scenario::setup_auth_required
    
    run claude_code::batch_automation "test prompt" "5" "10"
    
    assert_failure
    assert_output --partial "Authentication required"
}

#######################################
# Batch Config Tests
#######################################

@test "claude_code::batch_config - should require config file" {
    run claude_code::batch_config ""
    
    assert_failure
    assert_output --partial "Configuration file is required"
}

@test "claude_code::batch_config - should validate config file exists" {
    run claude_code::batch_config "/nonexistent/config.json"
    
    assert_failure
    assert_output --partial "Configuration file not found"
}

@test "claude_code::batch_config - should process valid config file" {
    local config_file="${TEST_BATCH_DIR}/batch_config.json"
    cat > "$config_file" << 'EOF'
{
    "batches": [
        {
            "prompt": "Create a function",
            "turns": 5,
            "tools": ["Read", "Write"]
        }
    ]
}
EOF
    
    # Mock successful execution
    mock::claude::scenario::setup_development
    
    run claude_code::batch_config "$config_file"
    
    assert_success
    assert_output --partial "Processing configuration"
}

#######################################
# Batch Multi Tests
#######################################

@test "claude_code::batch_multi - should handle multiple prompts" {
    # Mock successful execution
    mock::claude::scenario::setup_development
    
    run claude_code::batch_multi "Create function,Write tests,Debug code" "5"
    
    assert_success
    assert_output --partial "Processing multiple prompts"
}

@test "claude_code::batch_multi - should require prompts" {
    run claude_code::batch_multi "" "5"
    
    assert_failure
    assert_output --partial "At least one prompt is required"
}

#######################################
# Batch Parallel Tests
#######################################

@test "claude_code::batch_parallel - should execute parallel batch" {
    # Mock successful execution
    mock::claude::scenario::setup_development
    
    run claude_code::batch_parallel "2" "Create function,Write tests" "5"
    
    assert_success
    assert_output --partial "Parallel batch execution"
}

@test "claude_code::batch_parallel - should validate worker count" {
    run claude_code::batch_parallel "not_a_number" "test prompt" "5"
    
    assert_failure
    assert_output --partial "Worker count must be a valid number"
}

@test "claude_code::batch_parallel - should require prompts" {
    run claude_code::batch_parallel "2" "" "5"
    
    assert_failure
    assert_output --partial "At least one prompt is required"
}

#######################################
# Legacy Batch Function Tests
#######################################

@test "claude_code::batch - should execute legacy batch function" {
    # Mock successful execution
    mock::claude::scenario::setup_development
    
    run claude_code::batch
    
    assert_success
    assert_output --partial "batch mode"
}