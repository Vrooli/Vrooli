#!/usr/bin/env bats
################################################################################
# BATS tests for Vrooli Orchestrator dependency setup
################################################################################

# Test helper setup
setup() {
    # Load test helpers
    local test_dir="${BATS_TEST_DIRNAME}/../../../scripts/__test/helpers"
    
    if [[ -f "$test_dir/bats-support/load.bash" ]]; then
        load "$test_dir/bats-support/load.bash"
    fi
    
    if [[ -f "$test_dir/bats-assert/load.bash" ]]; then
        load "$test_dir/bats-assert/load.bash"
    fi
    
    # Source the orchestrator dependency script
    source "$BATS_TEST_DIRNAME/orchestrator.sh"
    
    # Setup test environment
    export VROOLI_ORCHESTRATOR_HOME="$BATS_TMPDIR/test-orchestrator"
    export SUDO_MODE="skip"  # Avoid sudo during tests
}

# Test cleanup
teardown() {
    # Clean up test orchestrator home
    [[ -d "$VROOLI_ORCHESTRATOR_HOME" ]] && rm -rf "$VROOLI_ORCHESTRATOR_HOME"
}

@test "orchestrator::setup creates home directory" {
    run orchestrator::setup
    
    assert_success
    assert [ -d "$VROOLI_ORCHESTRATOR_HOME" ]
    assert [ -d "$VROOLI_ORCHESTRATOR_HOME/logs" ]
    assert [ -d "$VROOLI_ORCHESTRATOR_HOME/sockets" ]
    assert [ -d "$VROOLI_ORCHESTRATOR_HOME/backups" ]
}

@test "orchestrator::setup creates initial registry file" {
    run orchestrator::setup
    
    assert_success
    assert [ -f "$VROOLI_ORCHESTRATOR_HOME/processes.json" ]
    
    # Verify JSON structure
    run jq '.version' "$VROOLI_ORCHESTRATOR_HOME/processes.json"
    assert_success
    assert_output '"1.0.0"'
    
    run jq '.processes' "$VROOLI_ORCHESTRATOR_HOME/processes.json"
    assert_success
    assert_output '{}'
}

@test "orchestrator::check validates system dependencies" {
    # Setup orchestrator first
    orchestrator::setup
    
    run orchestrator::check
    
    # Should succeed if all dependencies are available
    # Note: This might fail in CI environments without all tools
    if command -v jq >/dev/null 2>&1 && command -v curl >/dev/null 2>&1; then
        assert_success
    else
        # It's okay to skip this test if dependencies aren't available
        skip "System dependencies not available in test environment"
    fi
}

@test "orchestrator::check detects missing home directory" {
    # Don't setup orchestrator, just check
    run orchestrator::check
    
    assert_failure
    assert_output --partial "Orchestrator home not found"
}

@test "system commands are available" {
    local commands=("ps" "mkfifo")
    
    for cmd in "${commands[@]}"; do
        run command -v "$cmd"
        assert_success
    done
}

@test "orchestrator scripts exist and are executable" {
    local script_dir
    script_dir="$(cd "$BATS_TEST_DIRNAME/../../scenarios/tools" && pwd)"
    
    local scripts=(
        "vrooli-orchestrator.sh"
        "orchestrator-client.sh" 
        "orchestrator-ctl.sh"
        "multi-app-runner.sh"
    )
    
    for script in "${scripts[@]}"; do
        local script_path="$script_dir/$script"
        assert [ -f "$script_path" ]
        assert [ -x "$script_path" ]
    done
}

@test "orchestrator home permissions are correct" {
    orchestrator::setup
    
    # Check directory permissions (should be readable/writable by owner)
    local perm
    perm=$(stat -c "%a" "$VROOLI_ORCHESTRATOR_HOME")
    assert_equal "$perm" "755"
    
    # Check subdirectory permissions
    for subdir in logs sockets backups; do
        perm=$(stat -c "%a" "$VROOLI_ORCHESTRATOR_HOME/$subdir")
        assert_equal "$perm" "755"
    done
}

@test "registry file contains valid JSON" {
    orchestrator::setup
    
    local registry="$VROOLI_ORCHESTRATOR_HOME/processes.json"
    
    # Validate JSON syntax
    run jq '.' "$registry"
    assert_success
    
    # Check required fields
    run jq -e '.version' "$registry"
    assert_success
    
    run jq -e '.processes' "$registry"
    assert_success
    
    run jq -e '.metadata' "$registry"
    assert_success
}

@test "setup is idempotent" {
    # Run setup twice
    run orchestrator::setup
    assert_success
    
    run orchestrator::setup
    assert_success
    
    # Directory should still exist and be valid
    assert [ -d "$VROOLI_ORCHESTRATOR_HOME" ]
    assert [ -f "$VROOLI_ORCHESTRATOR_HOME/processes.json" ]
    
    # Registry should still be valid JSON
    run jq '.' "$VROOLI_ORCHESTRATOR_HOME/processes.json"
    assert_success
}

@test "environment variables are respected" {
    local custom_home="$BATS_TMPDIR/custom-orchestrator"
    export VROOLI_ORCHESTRATOR_HOME="$custom_home"
    
    run orchestrator::setup
    assert_success
    
    # Should create custom directory
    assert [ -d "$custom_home" ]
    assert [ -f "$custom_home/processes.json" ]
    
    # Clean up
    rm -rf "$custom_home"
}