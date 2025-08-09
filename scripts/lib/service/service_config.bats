#!/usr/bin/env bats
# Tests for service_config.sh - Service configuration management

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

setup() {
    vrooli_setup_unit_test
    
    # Source the service config utilities
    source "${BATS_TEST_DIRNAME}/service_config.sh"
    
    # Create mock project structure
    export MOCK_PROJECT_ROOT="${MOCK_TMP_DIR}/project"
    mkdir -p "${MOCK_PROJECT_ROOT}/.vrooli"
    
    # Override var variables for testing
    export var_VROOLI_CONFIG_DIR="${MOCK_PROJECT_ROOT}/.vrooli"
    export var_SERVICE_JSON_FILE="${MOCK_PROJECT_ROOT}/.vrooli/service.json"
}

teardown() {
    vrooli_cleanup_test
}

@test "service_config::has_inheritance detects inheritance" {
    # Create service.json with inheritance
    local service_json="${MOCK_TMP_DIR}/service.json"
    cat > "$service_json" <<EOF
{
  "version": "1.0.0",
  "inheritance": {
    "extends": "base.json"
  }
}
EOF
    
    run service_config::has_inheritance "$service_json"
    assert_success
}

@test "service_config::has_inheritance detects no inheritance" {
    # Create service.json without inheritance
    local service_json="${MOCK_TMP_DIR}/service.json"
    cat > "$service_json" <<EOF
{
  "version": "1.0.0",
  "service": "test"
}
EOF
    
    run service_config::has_inheritance "$service_json"
    assert_failure
}

@test "service_config::merge_configs merges JSON objects" {
    local base='{"a": 1, "b": {"x": 10}}'
    local override='{"b": {"y": 20}, "c": 3}'
    
    run service_config::merge_configs "$base" "$override"
    assert_success
    
    # Check that merged result contains expected values
    local result="$output"
    assert [ "$(echo "$result" | jq -r '.a')" = "1" ]
    assert [ "$(echo "$result" | jq -r '.b.x')" = "10" ]
    assert [ "$(echo "$result" | jq -r '.b.y')" = "20" ]
    assert [ "$(echo "$result" | jq -r '.c')" = "3" ]
}

@test "service_config::load_and_process loads simple config" {
    # Create simple service.json
    local service_json="${MOCK_TMP_DIR}/service.json"
    cat > "$service_json" <<EOF
{
  "version": "1.0.0",
  "service": "test"
}
EOF
    
    run service_config::load_and_process "$service_json"
    assert_success
    assert_output --partial '"version": "1.0.0"'
}

@test "service_config::load_and_process handles missing file" {
    run service_config::load_and_process "/nonexistent/service.json"
    assert_failure
    assert_output --partial "not found"
}

@test "service_config::validate checks required fields" {
    # Create invalid service.json (missing version)
    local service_json="${MOCK_TMP_DIR}/service.json"
    cat > "$service_json" <<EOF
{
  "service": "test"
}
EOF
    
    run service_config::validate "$service_json"
    assert_failure
    assert_output --partial "Missing required field 'version'"
}

@test "service_config::validate accepts valid config" {
    # Create valid service.json
    local service_json="${MOCK_TMP_DIR}/service.json"
    cat > "$service_json" <<EOF
{
  "version": "1.0.0",
  "service": "test"
}
EOF
    
    run service_config::validate "$service_json"
    assert_success
    assert_output --partial "valid"
}

@test "service_config::validate handles invalid JSON" {
    # Create invalid JSON
    local service_json="${MOCK_TMP_DIR}/service.json"
    cat > "$service_json" <<EOF
{
  "version": "1.0.0"
  "service": "test"  # Missing comma
}
EOF
    
    run service_config::validate "$service_json"
    assert_failure
    assert_output --partial "Invalid JSON"
}

@test "service_config::export_resource_urls exports environment variables" {
    # Create service.json
    local service_json="${MOCK_TMP_DIR}/service.json"
    cat > "$service_json" <<EOF
{
  "version": "1.0.0",
  "service": "test"
}
EOF
    
    # Mock RESOURCE_PORTS array
    declare -gA RESOURCE_PORTS=(["ollama"]="11434" ["n8n"]="5678")
    
    run service_config::export_resource_urls "$service_json"
    assert_success
    
    # Check that variables were exported
    [ "$OLLAMA_URL" = "http://localhost:11434" ]
    [ "$N8N_URL" = "http://localhost:5678" ]
}