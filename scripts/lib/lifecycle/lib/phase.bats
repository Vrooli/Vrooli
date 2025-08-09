#!/usr/bin/env bats
# Phase Management Module Tests
# Tests for phase execution and lifecycle management

bats_require_minimum_version 1.5.0

# Load test infrastructure (single entry point)
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test  # Basic mocks and environment
    
    # Source dependencies
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/config.sh"
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/targets.sh"
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/parallel.sh"
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/condition.sh"
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/phase.sh"
    
    # Set up test configuration
    export PHASE_CONFIG='{
        "steps": [
            {"name": "test_step_1", "command": "echo step1"},
            {"name": "test_step_2", "command": "echo step2"}
        ],
        "targets": {
            "default": {
                "env": {"TEST_VAR": "test_value"}
            }
        }
    }'
    export LIFECYCLE_PHASE="test"
}

teardown() {
    vrooli_cleanup_test     # Clean up resources
}

# Helper functions for testing  
test_step_function() {
    echo "Test step function called with: $*"
    return 0
}

failing_step_function() {
    echo "Failing step function"
    return 1
}

@test "phase::get_steps - extracts steps from configuration" {
    local steps
    steps=$(phase::get_steps)
    
    # Should be a JSON array with 2 steps
    local count
    count=$(echo "$steps" | jq 'length')
    [[ "$count" -eq 2 ]]
}

@test "phase::get_steps - handles missing steps" {
    export PHASE_CONFIG='{}'
    
    local steps
    steps=$(phase::get_steps)
    
    # Should return empty array
    [[ "$steps" == "[]" ]]
}

@test "phase::should_run_step - basic step filtering" {
    local step='{"name": "test_step", "command": "echo test"}'
    
    run phase::should_run_step "$step"
    assert_success
}

@test "phase::should_run_step - respects condition" {
    local step='{"name": "test_step", "command": "echo test", "condition": "false"}'
    
    run phase::should_run_step "$step"
    assert_failure
}

@test "phase::should_run_step - respects ONLY_STEP" {
    export ONLY_STEP="target_step"
    
    local step='{"name": "target_step", "command": "echo test"}'
    run phase::should_run_step "$step"
    assert_success
    
    local other_step='{"name": "other_step", "command": "echo test"}'
    run phase::should_run_step "$other_step"
    assert_failure
    
    unset ONLY_STEP
}

@test "phase::get_step_timeout - gets step timeout" {
    local step='{"name": "test_step", "timeout": 300}'
    
    local timeout
    timeout=$(phase::get_step_timeout "$step")
    [[ "$timeout" -eq 300 ]]
}

@test "phase::get_step_timeout - uses default timeout" {
    local step='{"name": "test_step"}'
    
    # Should use default (or return empty if no default set)
    local timeout
    timeout=$(phase::get_step_timeout "$step")
    # Either empty or a reasonable default
    [[ -z "$timeout" ]] || [[ "$timeout" =~ ^[0-9]+$ ]]
}

@test "phase::setup_phase_env - sets up environment from target" {
    export TARGET="default"
    
    run phase::setup_phase_env
    assert_success
    
    # Should have set the environment variable from target
    [[ "$TEST_VAR" == "test_value" ]]
}

@test "phase::setup_phase_env - handles missing target" {
    unset TARGET
    
    run phase::setup_phase_env
    assert_success
    
    # Should not fail, just not set any env vars
}

@test "phase::validate_phase_config - validates basic configuration" {
    run phase::validate_phase_config
    assert_success
}

@test "phase::validate_phase_config - validates steps format" {
    export PHASE_CONFIG='{"steps": "invalid"}'
    
    run phase::validate_phase_config
    assert_failure
}

@test "phase::has_parallel_steps - detects parallel steps" {
    export PHASE_CONFIG='{
        "steps": [
            {"name": "step1", "parallel": true},
            {"name": "step2"}
        ]
    }'
    
    run phase::has_parallel_steps
    assert_success
}

@test "phase::has_parallel_steps - detects no parallel steps" {
    export PHASE_CONFIG='{
        "steps": [
            {"name": "step1"},
            {"name": "step2"}
        ]
    }'
    
    run phase::has_parallel_steps
    assert_failure
}

@test "phase::get_phase_summary - returns phase summary" {
    local summary
    summary=$(phase::get_phase_summary)
    
    # Should contain step count and phase information
    echo "$summary" | jq -e '.step_count'
    echo "$summary" | jq -e '.phase'
}