#!/usr/bin/env bats

# Test script for index.sh (resource management orchestrator)
# Uses mocks to avoid real installations and network calls

# Load test helpers
load "../__test/fixtures/setup.bash"

# Setup test environment
setup() {
    # Use the standard Vrooli test setup
    vrooli_setup_unit_test
    
    # Set up test directory
    export RESOURCES_DIR="$BATS_TEST_DIRNAME"
    export MOCK_RESPONSES_DIR="$VROOLI_TEST_TMPDIR/mock_responses"
    mkdir -p "$MOCK_RESPONSES_DIR"
    
    # Mock external dependencies to prevent real actions
    mock::service::set_state "ollama" "active"
    mock::command::set_output "jq" '{"enabled": true}' 0
}

teardown() {
    # Use standard Vrooli test cleanup
    vrooli_cleanup_test
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "index::parse_arguments sets correct defaults" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && resources::parse_arguments && echo \"\$ACTION|\$RESOURCES_INPUT|\$FORCE\"")
    [[ "$output" == "install|none|no" ]]
}

@test "index::parse_arguments handles --help flag" {
    run bash -c "source '$RESOURCES_DIR/index.sh' && resources::parse_arguments --help"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Manages local development resources" ]]
}

@test "index::parse_arguments parses custom action" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && resources::parse_arguments --action status && echo \"\$ACTION\"")
    [[ "$output" == "status" ]]
}

@test "index::parse_arguments parses custom resources" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && resources::parse_arguments --resources ollama,n8n && echo \"\$RESOURCES_INPUT\"")
    [[ "$output" == "ollama,n8n" ]]
}

# ============================================================================
# Resource Resolution Tests
# ============================================================================

@test "index::resolve_list handles 'all' keyword" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && export RESOURCES_INPUT='all' && resources::resolve_list")
    [[ "$output" =~ "ollama" ]]
    [[ "$output" =~ "n8n" ]]
    [[ "$output" =~ "browserless" ]]
}

@test "index::resolve_list handles 'ai-only' keyword" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && export RESOURCES_INPUT='ai-only' && resources::resolve_list")
    [[ "$output" =~ "ollama" ]]
    [[ "$output" =~ "whisper" ]]
    [[ ! "$output" =~ "n8n" ]]
}

@test "index::resolve_list handles 'none' keyword" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && export RESOURCES_INPUT='none' && resources::resolve_list")
    [[ "$output" == "" ]]
}

@test "index::resolve_list handles comma-separated list" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && export RESOURCES_INPUT='ollama,n8n' && resources::resolve_list")
    [[ "$output" =~ "ollama" ]]
    [[ "$output" =~ "n8n" ]]
    [[ ! "$output" =~ "browserless" ]]
}

@test "index::resolve_list warns about unknown resources" {
    run bash -c "source '$RESOURCES_DIR/index.sh' && export RESOURCES_INPUT='unknown-resource' && resources::resolve_list 2>&1"
    [[ "$output" =~ "Unknown resource: unknown-resource" ]]
}

# ============================================================================
# Resource Category Tests
# ============================================================================

@test "index::get_category returns correct category for ollama" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && resources::get_category 'ollama'")
    [[ "$output" == "ai" ]]
}

@test "index::get_category returns correct category for n8n" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && resources::get_category 'n8n'")
    [[ "$output" == "automation" ]]
}

@test "index::get_category returns correct category for browserless" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && resources::get_category 'browserless'")
    [[ "$output" == "agents" ]]
}

@test "index::get_category returns 'unknown' for invalid resource" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && resources::get_category 'invalid'")
    [[ "$output" == "unknown" ]]
}

# ============================================================================
# Script Path Tests
# ============================================================================

@test "index::get_script_path returns correct path for ollama" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && resources::get_script_path 'ollama'")
    [[ "$output" =~ "/ai/ollama/manage.sh" ]]
}

@test "index::get_script_path returns correct path for n8n" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && resources::get_script_path 'n8n'")
    [[ "$output" =~ "/automation/n8n/manage.sh" ]]
}

# ============================================================================
# API-based Resource Detection Tests
# ============================================================================

@test "index::is_api_based returns true for known API services" {
    bash -c "source '$RESOURCES_DIR/index.sh' && resources::is_api_based 'openrouter'"
    [ "$?" -eq 0 ]
}

@test "index::is_api_based returns false for local services" {
    bash -c "source '$RESOURCES_DIR/index.sh' && resources::is_api_based 'ollama'"
    [ "$?" -eq 1 ]
}

# ============================================================================
# Interface Validation Tests
# ============================================================================

@test "index::run_interface_validation handles empty resource list" {
    run bash -c "source '$RESOURCES_DIR/index.sh' && resources::run_interface_validation"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No resources to validate" ]]
}

# Note: More comprehensive interface validation tests would require
# creating temporary mock scripts, which is beyond the scope of safe testing

# ============================================================================
# Configuration Tests (Safe - No Real File Creation)
# ============================================================================

@test "index::update_scenario_template handles missing template file safely" {
    run bash -c "source '$RESOURCES_DIR/index.sh' && resources::update_scenario_template '/nonexistent/file'"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Template file not found" ]]
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "index::execute_action handles missing script gracefully" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && resources::execute_action 'nonexistent-resource' 'status' 2>&1")
    [[ "$output" =~ "Resource script not found" ]]
}

@test "index::execute_action handles API-based resources correctly" {
    output=$(bash -c "source '$RESOURCES_DIR/index.sh' && resources::execute_action 'openrouter' 'install' 2>&1")
    [[ "$output" =~ "API-based service" ]]
}

# ============================================================================
# Usage Display Tests
# ============================================================================

@test "index::usage displays correct information" {
    run bash -c "source '$RESOURCES_DIR/index.sh' && resources::usage"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Manages local development resources" ]]
    [[ "$output" =~ "Examples:" ]]
    [[ "$output" =~ "Resource Categories:" ]]
}

# ============================================================================
# List Available Resources Tests
# ============================================================================

@test "index::list_available displays resource categories" {
    run bash -c "source '$RESOURCES_DIR/index.sh' && resources::list_available 2>&1"
    [[ "$output" =~ "Available Resources" ]]
    [[ "$output" =~ "Category: ai" ]]
    [[ "$output" =~ "Category: automation" ]]
}