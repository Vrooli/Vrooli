#!/usr/bin/env bats
# Tests for Resource Data Injection Engine (engine.sh)

# Load test setup
# shellcheck disable=SC1091
source "$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)/../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Set up test environment for injection engine
    export TEST_SCENARIO_DIR="${VROOLI_TEST_TMPDIR}/test-scenario"
    export TEST_SERVICE_JSON="${TEST_SCENARIO_DIR}/.vrooli/service.json"
    
    # Create test scenario directory structure
    mkdir -p "${TEST_SCENARIO_DIR}/.vrooli"
    mkdir -p "${TEST_SCENARIO_DIR}/initialization"
    
    # Create test service.json
    cat > "${TEST_SERVICE_JSON}" << 'EOF'
{
  "service": {
    "name": "test-scenario",
    "displayName": "Test Scenario",
    "description": "A test scenario for injection engine testing",
    "version": "1.0.0"
  },
  "resources": {
    "n8n": {
      "enabled": true,
      "initialization": {
        "workflows": [
          {
            "name": "test-workflow",
            "file": "workflows/test-workflow.json",
            "enabled": true
          }
        ]
      }
    }
  },
  "deployment": {
    "initialization": {
      "phases": [
        {
          "name": "setup",
          "parallel": false,
          "tasks": [
            {
              "name": "test-task",
              "type": "script",
              "script": "scripts/test.sh",
              "timeout": "30s"
            }
          ]
        }
      ]
    }
  }
}
EOF

    # Create test files referenced in service.json
    mkdir -p "${TEST_SCENARIO_DIR}/workflows"
    echo '{"test": "workflow"}' > "${TEST_SCENARIO_DIR}/workflows/test-workflow.json"
    
    mkdir -p "${TEST_SCENARIO_DIR}/scripts"
    cat > "${TEST_SCENARIO_DIR}/scripts/test.sh" << 'EOF'
#!/usr/bin/env bash
echo "Test script executed"
exit 0
EOF
    chmod +x "${TEST_SCENARIO_DIR}/scripts/test.sh"
}

teardown() {
    vrooli_cleanup_test
}

@test "engine::validate_scenario_dir validates valid scenario directory" {
    # Source the engine script
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    run injection::validate_scenario_dir "${TEST_SCENARIO_DIR}"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Scenario directory and service.json are valid"* ]]
}

@test "engine::validate_scenario_dir fails for non-existent directory" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    run injection::validate_scenario_dir "/non/existent/directory"
    [ "$status" -ne 0 ]
    [[ "$output" == *"Scenario directory not found"* ]]
}

@test "engine::validate_scenario_dir fails for missing service.json" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    # Create directory without service.json
    local test_dir="${VROOLI_TEST_TMPDIR}/no-service-json"
    mkdir -p "${test_dir}"
    
    run injection::validate_scenario_dir "${test_dir}"
    assert_failure
    assert_output --partial ".vrooli/service.json not found"
}

@test "engine::get_service_json returns valid JSON" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    run injection::get_service_json "${TEST_SCENARIO_DIR}"
    assert_success
    
    # Verify output is valid JSON and contains expected fields
    echo "$output" | jq -e '.service.name == "test-scenario"'
}

@test "engine::get_scenario_metadata extracts correct metadata" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    run injection::get_scenario_metadata "${TEST_SCENARIO_DIR}"
    assert_success
    
    # Parse and validate metadata
    local metadata="$output"
    [ "$(echo "$metadata" | jq -r '.name')" = "test-scenario" ]
    [ "$(echo "$metadata" | jq -r '.description')" = "A test scenario for injection engine testing" ]
    [ "$(echo "$metadata" | jq -r '.version')" = "1.0.0" ]
}

@test "engine::get_deployment_phases returns phases array" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    run injection::get_deployment_phases "${TEST_SCENARIO_DIR}"
    assert_success
    
    # Verify phases array structure
    echo "$output" | jq -e '. | length == 1'
    echo "$output" | jq -e '.[0].name == "setup"'
    echo "$output" | jq -e '.[0].tasks | length == 1'
}

@test "engine::get_resource_configs returns resources configuration" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    run injection::get_resource_configs "${TEST_SCENARIO_DIR}"
    assert_success
    
    # Verify resource configs structure
    echo "$output" | jq -e '.n8n.enabled == true'
    echo "$output" | jq -e '.n8n.initialization.workflows | length == 1'
}

@test "engine::execute_script_task executes valid script" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    # Create task configuration
    local task_config='{"name": "test-task", "type": "script", "script": "scripts/test.sh", "timeout": "30s"}'
    
    run injection::execute_script_task "${TEST_SCENARIO_DIR}" "$task_config"
    assert_success
    assert_output --partial "Script executed successfully"
}

@test "engine::execute_script_task fails for non-existent script" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    local task_config='{"name": "bad-task", "type": "script", "script": "scripts/nonexistent.sh", "timeout": "30s"}'
    
    run injection::execute_script_task "${TEST_SCENARIO_DIR}" "$task_config"
    assert_failure
    assert_output --partial "Script not found"
}

@test "engine::resolve_secret finds environment variable" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    export TEST_SECRET="test_value_from_env"
    
    run resolve_secret "TEST_SECRET"
    assert_success
    assert_output "test_value_from_env"
    
    unset TEST_SECRET
}

@test "engine::resolve_secret returns error for non-existent secret" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    run resolve_secret "NON_EXISTENT_SECRET"
    assert_failure
}

@test "engine::substitute_secrets_in_json replaces placeholders" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    export TEST_API_KEY="secret_api_key_123"
    
    local json_input='{"api_key": "{{TEST_API_KEY}}", "other": "value"}'
    
    run substitute_secrets_in_json "$json_input"
    assert_success
    assert_output --partial '"api_key": "secret_api_key_123"'
    assert_output --partial '"other": "value"'
    
    unset TEST_API_KEY
}

@test "engine::dry_run flag prevents actual execution" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    export DRY_RUN="yes"
    
    local task_config='{"name": "test-task", "type": "script", "script": "scripts/test.sh", "timeout": "30s"}'
    
    run injection::execute_script_task "${TEST_SCENARIO_DIR}" "$task_config"
    assert_success
    assert_output --partial "[DRY RUN]"
    assert_output --partial "Would execute task"
}

@test "engine::map_n8n_config converts service.json format to adapter format" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    local resource_config='{"initialization": {"workflows": [{"name": "test", "file": "workflows/test.json", "enabled": true}]}}'
    
    run injection::map_n8n_config "$resource_config" "${TEST_SCENARIO_DIR}"
    assert_success
    
    # Verify adapter format output
    echo "$output" | jq -e '.workflows | length == 1'
    echo "$output" | jq -e '.workflows[0].name == "test"'
    echo "$output" | jq -e '.workflows[0].file' | grep -q "${TEST_SCENARIO_DIR}/workflows/test.json"
}

@test "engine::inject_scenario_from_dir processes complete scenario" {
    source "${BATS_TEST_DIRNAME}/engine.sh"
    
    export DRY_RUN="yes"  # Use dry run to avoid actual resource calls
    
    run injection::inject_scenario_from_dir "${TEST_SCENARIO_DIR}"
    assert_success
    assert_output --partial "Injecting Scenario: test-scenario"
    assert_output --partial "Executing Phase: setup"
}