#!/usr/bin/env bats
# Tests for Claude Code inject.sh script
bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure
source "${APP_ROOT}/__test/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "claude-code"
    
    # Load Claude Code specific configuration once per file
    SCRIPT_DIR="${APP_ROOT}/resources/claude-code"
    
    # Load configuration if available
    if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
        source "${SCRIPT_DIR}/config/defaults.sh"
    fi
    
    # Load all dependencies needed by inject.sh using proper var_ variables
    # shellcheck disable=SC1091
    source "${APP_ROOT}/lib/utils/var.sh" || true
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh" || true
    
    # Now load inject.sh with all dependencies available
    source "${SCRIPT_DIR}/inject.sh"
    
    # Export key functions for BATS subshells
    export -f claude_code_inject::main
    export -f claude_code_inject::usage
    export -f claude_code_inject::validate_config
    export -f claude_code_inject::inject_data
    export -f claude_code_inject::check_status
    export -f claude_code_inject::check_accessibility
    export -f claude_code_inject::validate_templates
    export -f claude_code_inject::install_template
    export -f claude_code_inject::install_prompt
    export -f claude_code_inject::install_session
    
    # Create test fixture directories
    export TEST_CLAUDE_CODE_DATA_DIR="${BATS_TMPDIR}/claude-code-test"
    export CLAUDE_CODE_DATA_DIR="$TEST_CLAUDE_CODE_DATA_DIR"
    export CLAUDE_CODE_TEMPLATES_DIR="${CLAUDE_CODE_DATA_DIR}/templates"
    export CLAUDE_CODE_PROMPTS_DIR="${CLAUDE_CODE_DATA_DIR}/prompts"
    export CLAUDE_CODE_SESSIONS_DIR="${CLAUDE_CODE_DATA_DIR}/sessions"
}

setup() {
    # Reset mock state for each test
    mock::claude::reset >/dev/null 2>&1 || true
    
    # Clean up test directories for each test
    trash::safe_remove "$TEST_CLAUDE_CODE_DATA_DIR" --test-cleanup
    mkdir -p "$TEST_CLAUDE_CODE_DATA_DIR"
}

teardown() {
    # Clean up test directories
    trash::safe_remove "$TEST_CLAUDE_CODE_DATA_DIR" --test-cleanup
}

teardown_file() {
    # Clean up any remaining test artifacts
    trash::safe_remove "${BATS_TMPDIR}/claude-code-test" --test-cleanup
}

#######################################
# Configuration Validation Tests
#######################################

@test "claude_code_inject::validate_config - should reject invalid JSON" {
    run claude_code_inject::validate_config "invalid json"
    
    assert_failure
    assert_output --partial "Invalid JSON"
}

@test "claude_code_inject::validate_config - should require at least one injection type" {
    local config='{"other": "data"}'
    
    run claude_code_inject::validate_config "$config"
    
    assert_failure
    assert_output --partial "must have 'templates', 'prompts', 'sessions', 'api_keys', or 'configurations'"
}

@test "claude_code_inject::validate_config - should validate templates configuration" {
    local config='{
        "templates": [
            {
                "name": "test-template",
                "file": "README.md",
                "description": "Test template"
            }
        ]
    }'
    
    run claude_code_inject::validate_config "$config"
    
    assert_success
    assert_output --partial "All template configurations are valid"
}

@test "claude_code_inject::validate_config - should reject template without name" {
    local config='{
        "templates": [
            {
                "file": "README.md"
            }
        ]
    }'
    
    run claude_code_inject::validate_config "$config"
    
    assert_failure
    assert_output --partial "missing required 'name' field"
}

@test "claude_code_inject::validate_config - should reject template without file" {
    local config='{
        "templates": [
            {
                "name": "test-template"
            }
        ]
    }'
    
    run claude_code_inject::validate_config "$config"
    
    assert_failure
    assert_output --partial "missing required 'file' field"
}

@test "claude_code_inject::validate_config - should reject template with non-existent file" {
    local config='{
        "templates": [
            {
                "name": "test-template",
                "file": "non-existent-file.js"
            }
        ]
    }'
    
    run claude_code_inject::validate_config "$config"
    
    assert_failure
    assert_output --partial "Template file not found"
}

#######################################
# Template Installation Tests
#######################################

@test "claude_code_inject::install_template - should install template successfully" {
    # Create a test template file
    local test_file="${var_ROOT_DIR}/test-template.js"
    echo "console.log('test template');" > "$test_file"
    
    local template_config='{
        "name": "test-template",
        "file": "test-template.js",
        "description": "Test template",
        "tags": ["test"]
    }'
    
    run claude_code_inject::install_template "$template_config"
    
    assert_success
    assert_output --partial "Installing template: test-template"
    assert_output --partial "Installed template: test-template"
    
    # Verify template was installed
    assert [ -f "${CLAUDE_CODE_TEMPLATES_DIR}/test-template.js" ]
    assert [ -f "${CLAUDE_CODE_TEMPLATES_DIR}/test-template.meta.json" ]
    
    # Clean up
    trash::safe_remove "$test_file" --test-cleanup
}

#######################################
# Accessibility Tests
#######################################

@test "claude_code_inject::check_accessibility - should return success when claude command exists" {
    # Mock claude command as available
    mock::claude::scenario::setup_healthy
    
    # Create a temporary claude command in PATH
    local temp_bin="${BATS_TMPDIR}/bin"
    mkdir -p "$temp_bin"
    echo '#!/bin/bash' > "${temp_bin}/claude"
    chmod +x "${temp_bin}/claude"
    export PATH="${temp_bin}:${PATH}"
    
    run claude_code_inject::check_accessibility
    
    assert_success
    
    # Clean up
    trash::safe_remove "$temp_bin" --test-cleanup
}

@test "claude_code_inject::check_accessibility - should return success when data directory exists" {
    # Create data directory
    mkdir -p "$CLAUDE_CODE_DATA_DIR"
    
    run claude_code_inject::check_accessibility
    
    assert_success
}

@test "claude_code_inject::check_accessibility - should return failure when neither command nor directory exists" {
    # Ensure data directory doesn't exist
    trash::safe_remove "$CLAUDE_CODE_DATA_DIR" --test-cleanup
    
    run claude_code_inject::check_accessibility
    
    assert_failure
    assert_output --partial "Claude Code CLI not found"
}

#######################################
# Status Check Tests
#######################################

@test "claude_code_inject::check_status - should report installation status" {
    local config='{"templates": []}'
    
    run claude_code_inject::check_status "$config"
    
    assert_success
    assert_output --partial "Checking Claude Code injection status"
}

#######################################
# Usage and Main Function Tests
#######################################

@test "claude_code_inject::usage - should display usage information" {
    run claude_code_inject::usage
    
    assert_success
    assert_output --partial "Claude Code Data Injection Adapter"
    assert_output --partial "USAGE:"
    assert_output --partial "--validate"
    assert_output --partial "--inject"
    assert_output --partial "--status"
    assert_output --partial "--rollback"
}

@test "claude_code_inject::main - should validate configuration" {
    local config='{"templates": []}'
    
    run claude_code_inject::main "--validate" "$config"
    
    assert_success
}

@test "claude_code_inject::main - should require configuration parameter" {
    run claude_code_inject::main "--validate"
    
    assert_failure
    assert_output --partial "Configuration JSON required"
}

@test "claude_code_inject::main - should handle unknown action" {
    local config='{"templates": []}'
    
    run claude_code_inject::main "--unknown" "$config"
    
    assert_failure
    assert_output --partial "Unknown action: --unknown"
}

@test "claude_code_inject::main - should show help" {
    run claude_code_inject::main "--help"
    
    assert_success
    assert_output --partial "Claude Code Data Injection Adapter"
}