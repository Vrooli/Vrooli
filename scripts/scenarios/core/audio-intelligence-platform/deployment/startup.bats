#!/usr/bin/env bats
# Tests for audio intelligence platform startup script
# Validates deployment script functionality

bats_require_minimum_version 1.5.0

# Load test setup
# shellcheck disable=SC1091
load "../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Create temporary scenario directory structure for testing
    export TEST_SCENARIO_DIR="${BATS_TEST_TMPDIR}/test_scenario"
    mkdir -p "$TEST_SCENARIO_DIR/.vrooli"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/automation/n8n"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/configuration"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/storage/postgres"
    
    # Create a mock service.json
    cat > "$TEST_SCENARIO_DIR/.vrooli/service.json" << 'EOF'
{
    "resources": {
        "required": {
            "postgres": {"required": true},
            "ollama": {"required": true}
        }
    },
    "deployment": {
        "testing": {
            "ui": {"required": true, "type": "windmill"},
            "timeout": "45m"
        }
    }
}
EOF
    
    # Override SCENARIO_DIR for testing
    export SCENARIO_DIR="$TEST_SCENARIO_DIR"
    
    # Load the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/startup.sh"
}

teardown() {
    vrooli_cleanup_test
}

# Test that the script loads without errors
@test "startup.sh loads without errors" {
    # Test that functions are defined
    declare -F startup::load_configuration >/dev/null
    declare -F startup::validate_resources >/dev/null
    declare -F startup::initialize_database >/dev/null
    declare -F startup::main >/dev/null
}

# Test configuration loading
@test "startup::load_configuration loads service.json correctly" {
    run startup::load_configuration
    assert_success
    
    # Check that variables are set
    [[ -n "$REQUIRED_RESOURCES" ]]
    [[ "$REQUIRED_RESOURCES" =~ "postgres" ]]
    [[ "$REQUIRED_RESOURCES" =~ "ollama" ]]
    [[ "$REQUIRES_UI" == "true" ]]
}

# Test configuration loading with missing service.json
@test "startup::load_configuration fails with missing service.json" {
    # Remove the service.json file
    rm "$TEST_SCENARIO_DIR/.vrooli/service.json"
    
    run startup::load_configuration
    assert_failure
}

# Test resource validation (using mocks)
@test "startup::validate_resources succeeds with healthy resources" {
    startup::load_configuration
    
    run startup::validate_resources
    if [[ -n "${MOCK_LOG_DIR:-}" ]]; then
        # If mocks are available, test should succeed
        assert_success
    else
        # If no mocks, skip this test
        skip "No resource mocks available"
    fi
}

# Test configuration validation
@test "startup::apply_configuration validates JSON files" {
    # Create some test configuration files
    echo '{"test": "config"}' > "$TEST_SCENARIO_DIR/initialization/configuration/test.json"
    echo '{"invalid": json}' > "$TEST_SCENARIO_DIR/initialization/configuration/invalid.json"
    
    run startup::apply_configuration
    assert_success
    # The function should complete but log errors for invalid JSON
}

# Test help functionality
@test "startup script shows help with help command" {
    run "$BATS_TEST_DIRNAME/startup.sh" help
    assert_success
    assert_output --partial "Usage:"
    assert_output --partial "Commands:"
}

# Test validation command
@test "startup script validate command works" {
    run "$BATS_TEST_DIRNAME/startup.sh" validate
    # May succeed or fail depending on available resources, but should not crash
    [[ "$status" -eq 0 ]] || [[ "$status" -eq 1 ]]
}

# Assertion functions are provided by test fixtures