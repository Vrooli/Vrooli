#!/usr/bin/env bats
# Configuration Module Tests
# Tests for service.json loading and configuration validation

bats_require_minimum_version 1.5.0

# Load test infrastructure (single entry point)
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test  # Basic mocks and environment
    
    # Source the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/config.sh"
    
    # Create a test service.json
    export TEST_SERVICE_JSON="${BATS_TEST_TMPDIR}/service.json"
    cat > "$TEST_SERVICE_JSON" <<EOF
{
  "lifecycle": {
    "phases": {
      "setup": {
        "steps": [
          {"name": "test_step", "command": "echo test"}
        ],
        "targets": {
          "default": {
            "env": {"TEST_VAR": "test_value"}
          }
        }
      }
    }
  },
  "defaults": {
    "timeout": 300,
    "shell": "/bin/bash"
  }
}
EOF
}

teardown() {
    vrooli_cleanup_test     # Clean up resources
}

@test "config::load - loads valid service.json" {
    config::load "$TEST_SERVICE_JSON"
    
    # Check that SERVICE_JSON is populated
    [[ -n "$SERVICE_JSON" ]]
}

@test "config::load - fails with invalid JSON" {
    local invalid_json="${BATS_TEST_TMPDIR}/invalid.json"
    echo "invalid json content" > "$invalid_json"
    
    run config::load "$invalid_json"
    assert_failure
}

@test "config::load - fails with non-existent file" {
    run config::load "/nonexistent/file.json"
    assert_failure
}

@test "config::get_lifecycle - extracts lifecycle configuration" {
    config::load "$TEST_SERVICE_JSON"
    
    local lifecycle
    lifecycle=$(config::get_lifecycle)
    
    # Should contain phases
    echo "$lifecycle" | jq -e '.phases' >/dev/null
}

@test "config::get_defaults - extracts defaults configuration" {
    config::load "$TEST_SERVICE_JSON"
    
    local defaults
    defaults=$(config::get_defaults)
    
    # Should contain timeout
    local timeout
    timeout=$(echo "$defaults" | jq -r '.timeout')
    [[ "$timeout" == "300" ]]
}

@test "config::get_phase - extracts specific phase configuration" {
    config::load "$TEST_SERVICE_JSON"
    config::set_phase "setup"
    
    local phase_config
    phase_config=$(config::get_phase "setup")
    
    # Should contain steps
    echo "$phase_config" | jq -e '.steps' >/dev/null
}

@test "config::validate_phase - validates existing phase" {
    config::load "$TEST_SERVICE_JSON"
    
    run config::validate_phase "setup"
    assert_success
}

@test "config::validate_phase - fails for non-existent phase" {
    config::load "$TEST_SERVICE_JSON"
    
    run config::validate_phase "nonexistent"
    assert_failure
}

@test "config::target_exists - checks target existence" {
    config::load "$TEST_SERVICE_JSON"
    config::set_phase "setup"
    
    run config::target_exists "default"
    assert_success
    
    run config::target_exists "nonexistent"
    assert_failure
}

@test "config::get_targets - extracts targets configuration" {
    config::load "$TEST_SERVICE_JSON"
    config::set_phase "setup"
    
    local targets
    targets=$(config::get_targets)
    
    # Should contain default target
    echo "$targets" | jq -e '.default' >/dev/null
}