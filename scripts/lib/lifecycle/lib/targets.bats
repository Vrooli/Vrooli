#!/usr/bin/env bats
# Target Management Module Tests
# Tests for target configuration and inheritance

bats_require_minimum_version 1.5.0

# Load test infrastructure (single entry point)
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test  # Basic mocks and environment
    
    # Source the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/config.sh"
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/targets.sh"
    
    # Set up test phase configuration
    export PHASE_CONFIG='{
        "targets": {
            "default": {
                "steps": [{"name": "default_step", "command": "echo default"}],
                "env": {"DEFAULT_VAR": "default_value"}
            },
            "docker": {
                "extends": "default",
                "steps": [{"name": "docker_step", "command": "docker build"}],
                "env": {"DOCKER_VAR": "docker_value"}
            },
            "production": {
                "override": true,
                "steps": [{"name": "prod_step", "command": "deploy"}]
            }
        }
    }'
    export LIFECYCLE_PHASE="test"
}

teardown() {
    vrooli_cleanup_test     # Clean up resources
    targets::clear_cache
}

@test "targets::get_config - returns empty object for non-existent target" {
    local config
    config=$(targets::get_config "nonexistent")
    
    [[ "$config" == "{}" ]]
}

@test "targets::get_config - returns default target configuration" {
    local config
    config=$(targets::get_config "default")
    
    # Should contain steps
    echo "$config" | jq -e '.steps' >/dev/null
    # Should contain env
    echo "$config" | jq -e '.env' >/dev/null
}

@test "targets::get_config - handles inheritance with extends" {
    local config
    config=$(targets::get_config "docker")
    
    # Should have both default and docker steps (merged)
    local step_count
    step_count=$(echo "$config" | jq '.steps | length')
    [[ "$step_count" -eq 2 ]]
    
    # Should have both default and docker environment variables
    local default_var
    default_var=$(echo "$config" | jq -r '.env.DEFAULT_VAR')
    [[ "$default_var" == "default_value" ]]
    
    local docker_var
    docker_var=$(echo "$config" | jq -r '.env.DOCKER_VAR')
    [[ "$docker_var" == "docker_value" ]]
}

@test "targets::get_config - handles override strategy" {
    local config
    config=$(targets::get_config "production")
    
    # Should only have production step (override = true)
    local step_count
    step_count=$(echo "$config" | jq '.steps | length')
    [[ "$step_count" -eq 1 ]]
    
    local step_name
    step_name=$(echo "$config" | jq -r '.steps[0].name')
    [[ "$step_name" == "prod_step" ]]
}

@test "targets::list - lists all available targets" {
    local targets
    targets=$(targets::list)
    
    # Should contain all three targets
    echo "$targets" | grep -q "default"
    echo "$targets" | grep -q "docker"
    echo "$targets" | grep -q "production"
}

@test "targets::list - handles empty targets configuration" {
    export PHASE_CONFIG='{"targets": {}}'
    
    local targets
    targets=$(targets::list)
    
    # Should be empty
    [[ -z "$targets" ]]
}

@test "targets::validate - validates existing target" {
    run targets::validate "default"
    assert_success
}

@test "targets::validate - accepts any target when no targets defined" {
    export PHASE_CONFIG='{}'
    
    run targets::validate "any_target"
    assert_success
}

@test "targets::validate - fails for non-existent target" {
    run targets::validate "nonexistent"
    assert_failure
}

@test "targets::validate - falls back to default for missing target" {
    # Create config without docker target but with default
    export PHASE_CONFIG='{
        "targets": {
            "default": {"steps": []}
        }
    }'
    
    run targets::validate "missing_target"
    assert_success  # Should succeed because default exists
}

@test "targets::get_env - extracts environment variables" {
    local env
    env=$(targets::get_env "default")
    
    # Should contain the environment variable in key=value format
    echo "$env" | grep -q "DEFAULT_VAR=default_value"
}

@test "targets::get_default - extracts default values" {
    # Set up config with defaults
    export PHASE_CONFIG='{
        "targets": {
            "default": {
                "defaults": {
                    "timeout": 600,
                    "shell": "/bin/bash"
                }
            }
        }
    }'
    
    local timeout
    timeout=$(targets::get_default "default" "timeout")
    [[ "$timeout" == "600" ]]
    
    local shell
    shell=$(targets::get_default "default" "shell")
    [[ "$shell" == "/bin/bash" ]]
}

@test "targets::clear_cache - clears target cache" {
    # First call should populate cache
    targets::get_config "default"
    
    # Clear cache
    targets::clear_cache
    
    # Cache should be empty (we can't directly test this, but function should not fail)
    run targets::clear_cache
    assert_success
}