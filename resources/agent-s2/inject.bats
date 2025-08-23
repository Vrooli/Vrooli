#!/usr/bin/env bats
# Tests for Agent S2 inject.sh script

# Get the script directory and source var.sh first
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get proper directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "agent-s2"
    
    # Set up directories and mock path
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    export MOCK_DIR="${SCRIPT_DIR}/../../../__test/fixtures/mocks"
    
    # Load all dependencies once (expensive operations)
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
    
    # Source Agent-S2 configuration
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
    
    # Load the agent-s2 mock once
    if [[ -f "$MOCK_DIR/agent-s2.sh" ]]; then
        # shellcheck disable=SC1091
        source "$MOCK_DIR/agent-s2.sh"
    fi
    
    # Source inject.sh
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/inject.sh"
    
    # Export key functions for BATS subshells
    export -f inject::main
    export -f inject::usage
    export -f inject::check_accessibility
    export -f inject::validate_config
    export -f inject::validate_profiles
    export -f inject::validate_workflows
    export -f inject::inject_data
    export -f inject::inject_profiles
    export -f inject::inject_workflows
    export -f inject::inject_proxies
    export -f inject::apply_configurations
    export -f inject::check_status
    export -f inject::execute_rollback
    export -f inject::add_rollback_action
    export -f inject::create_profile
    export -f inject::install_workflow
    export -f inject::configure_proxy
    
    # Export log functions once
    log::header() { echo "=== $* ==="; }
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    log::debug() { echo "[DEBUG] $*"; }
    log::warn() { echo "[WARN] $*" >&2; }
    export -f log::header log::info log::error log::success log::warning log::debug log::warn
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Reset mock state to clean slate for each test
    if declare -f mock::agents2::reset >/dev/null 2>&1; then
        mock::agents2::reset
    fi
    
    # Set test-specific environment variables
    export AGENT_S2_HOST="http://localhost:8006"
    export AGENT_S2_DATA_DIR="${BATS_TEST_TMPDIR}/agent-s2"
    export AGENT_S2_PROFILES_DIR="${AGENT_S2_DATA_DIR}/profiles"
    export AGENT_S2_WORKFLOWS_DIR="${AGENT_S2_DATA_DIR}/workflows"
    
    # Create test directories
    mkdir -p "$AGENT_S2_DATA_DIR"
    mkdir -p "$AGENT_S2_PROFILES_DIR"
    mkdir -p "$AGENT_S2_WORKFLOWS_DIR"
    
    # Reset rollback actions
    AGENT_S2_ROLLBACK_ACTIONS=()
}

# BATS teardown function - runs after each test
teardown() {
    # Clean up test directories
    trash::safe_remove "${BATS_TEST_TMPDIR}/agent-s2" --test-cleanup 2>/dev/null || true
    
    vrooli_cleanup_test
}

# ============================================================================
# Script Loading Tests
# ============================================================================

@test "inject.sh loads without errors" {
    # Should load successfully in setup_file
    [ "$?" -eq 0 ]
}

@test "inject.sh defines required functions" {
    # Functions should be available from setup_file
    declare -f inject::main >/dev/null
    declare -f inject::validate_config >/dev/null
    declare -f inject::inject_data >/dev/null
    declare -f inject::check_status >/dev/null
}

# ============================================================================
# Usage Tests
# ============================================================================

@test "inject::usage displays help text" {
    run inject::usage
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Agent-S2 Data Injection Adapter"* ]]
    [[ "$output" == *"--validate"* ]]
    [[ "$output" == *"--inject"* ]]
    [[ "$output" == *"--status"* ]]
    [[ "$output" == *"--rollback"* ]]
}

# ============================================================================
# Configuration Validation Tests
# ============================================================================

@test "inject::validate_config accepts valid configuration" {
    local config='{"profiles": [{"name": "test"}]}'
    
    run inject::validate_config "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Agent-S2 injection configuration is valid"* ]]
}

@test "inject::validate_config rejects invalid JSON" {
    local config='{"profiles": invalid}'
    
    run inject::validate_config "$config"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Invalid JSON"* ]]
}

@test "inject::validate_config requires at least one injection type" {
    local config='{}'
    
    run inject::validate_config "$config"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"must have 'profiles', 'workflows', 'proxy_configs', or 'configurations'"* ]]
}

# ============================================================================
# Profile Validation Tests
# ============================================================================

@test "inject::validate_profiles accepts valid profiles" {
    local profiles='[{"name": "stealth_profile", "user_agent": "Mozilla/5.0"}]'
    
    run inject::validate_profiles "$profiles"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"All profile configurations are valid"* ]]
}

@test "inject::validate_profiles rejects profiles without name" {
    local profiles='[{"user_agent": "Mozilla/5.0"}]'
    
    run inject::validate_profiles "$profiles"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"missing required 'name' field"* ]]
}

@test "inject::validate_profiles rejects non-array input" {
    local profiles='{"name": "test"}'
    
    run inject::validate_profiles "$profiles"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"must be an array"* ]]
}

# ============================================================================
# Workflow Validation Tests
# ============================================================================

@test "inject::validate_workflows accepts valid workflows with existing files" {
    # Create a test workflow file
    local test_file="${var_ROOT_DIR}/test-workflow.json"
    echo '{"steps": []}' > "$test_file"
    
    local workflows='[{"name": "test_workflow", "file": "test-workflow.json"}]'
    
    run inject::validate_workflows "$workflows"
    
    # Clean up
    trash::safe_remove "$test_file" --test-cleanup
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"All workflow configurations are valid"* ]]
}

@test "inject::validate_workflows rejects workflows without name" {
    local workflows='[{"file": "workflow.json"}]'
    
    run inject::validate_workflows "$workflows"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"missing required 'name' field"* ]]
}

@test "inject::validate_workflows rejects workflows without file" {
    local workflows='[{"name": "test_workflow"}]'
    
    run inject::validate_workflows "$workflows"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"missing required 'file' field"* ]]
}

@test "inject::validate_workflows rejects workflows with non-existent file" {
    local workflows='[{"name": "test_workflow", "file": "non-existent.json"}]'
    
    run inject::validate_workflows "$workflows"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Workflow file not found"* ]]
}

# ============================================================================
# Profile Creation Tests
# ============================================================================

@test "inject::create_profile creates profile file" {
    local profile='{"name": "test_profile", "user_agent": "Mozilla/5.0"}'
    
    run inject::create_profile "$profile"
    
    [ "$status" -eq 0 ]
    [ -f "${AGENT_S2_PROFILES_DIR}/test_profile.json" ]
    
    # Verify content
    local saved_content
    saved_content=$(cat "${AGENT_S2_PROFILES_DIR}/test_profile.json")
    [[ "$saved_content" == "$profile" ]]
}

@test "inject::create_profile adds rollback action" {
    local profile='{"name": "rollback_test", "user_agent": "Mozilla/5.0"}'
    
    # Clear rollback actions
    AGENT_S2_ROLLBACK_ACTIONS=()
    
    inject::create_profile "$profile"
    
    # Check rollback action was added
    [ "${#AGENT_S2_ROLLBACK_ACTIONS[@]}" -eq 1 ]
    [[ "${AGENT_S2_ROLLBACK_ACTIONS[0]}" == *"rollback_test"* ]]
}

# ============================================================================
# Workflow Installation Tests
# ============================================================================

@test "inject::install_workflow installs workflow file" {
    # Create a test workflow file
    local test_file="${var_ROOT_DIR}/test-workflow.json"
    echo '{"steps": ["step1", "step2"]}' > "$test_file"
    
    local workflow='{"name": "test_workflow", "file": "test-workflow.json", "autoload": false}'
    
    run inject::install_workflow "$workflow"
    
    # Clean up source file
    trash::safe_remove "$test_file" --test-cleanup
    
    [ "$status" -eq 0 ]
    [ -f "${AGENT_S2_WORKFLOWS_DIR}/test_workflow.json" ]
    [ -f "${AGENT_S2_WORKFLOWS_DIR}/test_workflow.meta.json" ]
    
    # Verify content
    local saved_content
    saved_content=$(cat "${AGENT_S2_WORKFLOWS_DIR}/test_workflow.json")
    [[ "$saved_content" == '{"steps": ["step1", "step2"]}' ]]
}

@test "inject::install_workflow creates metadata file" {
    # Create a test workflow file
    local test_file="${var_ROOT_DIR}/test-workflow.json"
    echo '{"steps": []}' > "$test_file"
    
    local workflow='{"name": "meta_test", "file": "test-workflow.json", "autoload": true}'
    
    inject::install_workflow "$workflow"
    
    # Clean up source file
    trash::safe_remove "$test_file" --test-cleanup
    
    [ -f "${AGENT_S2_WORKFLOWS_DIR}/meta_test.meta.json" ]
    
    # Check metadata content
    local metadata
    metadata=$(cat "${AGENT_S2_WORKFLOWS_DIR}/meta_test.meta.json")
    [[ "$metadata" == *'"name": "meta_test"'* ]]
    [[ "$metadata" == *'"autoload": true'* ]]
}

# ============================================================================
# Proxy Configuration Tests
# ============================================================================

@test "inject::configure_proxy saves proxy configuration" {
    local proxy='{"name": "test_proxy", "type": "http", "host": "proxy.example.com", "port": 8080}'
    
    run inject::configure_proxy "$proxy"
    
    [ "$status" -eq 0 ]
    [ -f "${AGENT_S2_DATA_DIR}/proxies/test_proxy.json" ]
    
    # Verify content
    local saved_content
    saved_content=$(cat "${AGENT_S2_DATA_DIR}/proxies/test_proxy.json")
    [[ "$saved_content" == "$proxy" ]]
}

# ============================================================================
# Configuration Application Tests
# ============================================================================

@test "inject::apply_configurations saves configuration file" {
    local configurations='[{"key": "headless", "value": false}, {"key": "stealth_mode", "value": true}]'
    
    run inject::apply_configurations "$configurations"
    
    [ "$status" -eq 0 ]
    [ -f "${AGENT_S2_DATA_DIR}/config.json" ]
    
    # Verify content
    local saved_content
    saved_content=$(cat "${AGENT_S2_DATA_DIR}/config.json")
    [[ "$saved_content" == "$configurations" ]]
}

# ============================================================================
# Rollback Tests
# ============================================================================

@test "inject::execute_rollback executes rollback actions in reverse order" {
    # Clear rollback actions
    AGENT_S2_ROLLBACK_ACTIONS=()
    
    # Create test files
    touch "${BATS_TEST_TMPDIR}/file1.txt"
    touch "${BATS_TEST_TMPDIR}/file2.txt"
    
    # Add rollback actions
    inject::add_rollback_action "Remove file1" "trash::safe_remove '${BATS_TEST_TMPDIR}/file1.txt' --test-cleanup"
    inject::add_rollback_action "Remove file2" "trash::safe_remove '${BATS_TEST_TMPDIR}/file2.txt' --test-cleanup"
    
    # Files should exist before rollback
    [ -f "${BATS_TEST_TMPDIR}/file1.txt" ]
    [ -f "${BATS_TEST_TMPDIR}/file2.txt" ]
    
    run inject::execute_rollback
    
    [ "$status" -eq 0 ]
    # Files should be removed after rollback
    [ ! -f "${BATS_TEST_TMPDIR}/file1.txt" ]
    [ ! -f "${BATS_TEST_TMPDIR}/file2.txt" ]
    
    # Rollback actions should be cleared
    [ "${#AGENT_S2_ROLLBACK_ACTIONS[@]}" -eq 0 ]
}

@test "inject::execute_rollback handles empty rollback list" {
    # Clear rollback actions
    AGENT_S2_ROLLBACK_ACTIONS=()
    
    run inject::execute_rollback
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"No Agent-S2 rollback actions to execute"* ]]
}

# ============================================================================
# Main Function Tests
# ============================================================================

@test "inject::main requires configuration JSON" {
    run inject::main "--validate"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Configuration JSON required"* ]]
}

@test "inject::main handles validate action" {
    local config='{"profiles": [{"name": "test"}]}'
    
    run inject::main "--validate" "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Agent-S2 injection configuration is valid"* ]]
}

@test "inject::main handles unknown action" {
    run inject::main "--unknown" '{"profiles": []}'
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown action: --unknown"* ]]
}

@test "inject::main handles help action" {
    run inject::main "--help" "dummy"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Agent-S2 Data Injection Adapter"* ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "inject::inject_data processes profiles successfully" {
    local config='{"profiles": [{"name": "integration_test", "user_agent": "Test/1.0"}]}'
    
    run inject::inject_data "$config"
    
    [ "$status" -eq 0 ]
    [ -f "${AGENT_S2_PROFILES_DIR}/integration_test.json" ]
    [[ "$output" == *"Agent-S2 data injection completed"* ]]
}

@test "inject::inject_data processes configurations successfully" {
    local config='{"configurations": [{"key": "test_key", "value": "test_value"}]}'
    
    run inject::inject_data "$config"
    
    [ "$status" -eq 0 ]
    [ -f "${AGENT_S2_DATA_DIR}/config.json" ]
}

@test "inject::check_status reports injection status" {
    # Create some test data
    mkdir -p "$AGENT_S2_PROFILES_DIR"
    echo '{"name": "test"}' > "${AGENT_S2_PROFILES_DIR}/test.json"
    
    mkdir -p "$AGENT_S2_WORKFLOWS_DIR"
    echo '{"steps": []}' > "${AGENT_S2_WORKFLOWS_DIR}/workflow.json"
    
    local config='{}'
    
    run inject::check_status "$config"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Found 1 browser profiles"* ]]
    [[ "$output" == *"Found 1 workflows"* ]]
}