#!/usr/bin/env bats
# Tests for Schema Validator (schema-validator.sh)

# Load test setup
# shellcheck disable=SC1091
source "$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)/../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Set up test environment for schema validator
    export TEST_VALID_CONFIG="${VROOLI_TEST_TMPDIR}/valid-scenarios.json"
    export TEST_INVALID_JSON="${VROOLI_TEST_TMPDIR}/invalid.json"
    export TEST_MISSING_SCENARIOS="${VROOLI_TEST_TMPDIR}/missing-scenarios.json"
    export TEST_INVALID_STRUCTURE="${VROOLI_TEST_TMPDIR}/invalid-structure.json"
    export TEST_CIRCULAR_DEPS="${VROOLI_TEST_TMPDIR}/circular-deps.json"
    export TEST_SCHEMA_FILE="${VROOLI_TEST_TMPDIR}/test-schema.json"
    
    # Create valid scenarios configuration
    cat > "${TEST_VALID_CONFIG}" << 'EOF'
{
  "version": "1.0.0",
  "scenarios": {
    "test-scenario": {
      "description": "A test scenario for validation testing",
      "version": "1.0.0",
      "resources": {
        "n8n": {
          "workflows": [
            {
              "name": "test-workflow",
              "file": "workflows/test.json",
              "enabled": true
            }
          ]
        }
      }
    },
    "another-scenario": {
      "description": "Another test scenario",
      "version": "2.1.0",
      "dependencies": ["test-scenario"],
      "resources": {
        "postgres": {
          "data": [
            {
              "type": "schema",
              "file": "sql/schema.sql"
            }
          ]
        }
      }
    }
  },
  "active": ["test-scenario", "another-scenario"]
}
EOF

    # Create invalid JSON file
    cat > "${TEST_INVALID_JSON}" << 'EOF'
{
  "scenarios": {
    "broken": {
      "description": "This has invalid JSON syntax"
      // Invalid comment
    }
  }
EOF

    # Create file missing scenarios section
    cat > "${TEST_MISSING_SCENARIOS}" << 'EOF'
{
  "version": "1.0.0",
  "active": ["test"]
}
EOF

    # Create file with invalid scenario structure
    cat > "${TEST_INVALID_STRUCTURE}" << 'EOF'
{
  "scenarios": {
    "invalid-scenario": {
      "description": "Missing required fields",
      "resources": {}
    }
  }
}
EOF

    # Create file with circular dependencies
    cat > "${TEST_CIRCULAR_DEPS}" << 'EOF'
{
  "scenarios": {
    "scenario-a": {
      "description": "Scenario A",
      "version": "1.0.0",
      "dependencies": ["scenario-b"],
      "resources": {}
    },
    "scenario-b": {
      "description": "Scenario B",
      "version": "1.0.0",
      "dependencies": ["scenario-a"],
      "resources": {}
    }
  }
}
EOF

    # Create test schema file
    cat > "${TEST_SCHEMA_FILE}" << 'EOF'
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "scenarios": {
      "type": "object"
    }
  },
  "required": ["scenarios"]
}
EOF
}

teardown() {
    vrooli_cleanup_test
}

@test "schema_validator::validate_json_syntax accepts valid JSON" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    run validator::validate_json_syntax "${TEST_VALID_CONFIG}"
    assert_success
}

@test "schema_validator::validate_json_syntax rejects invalid JSON" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    run validator::validate_json_syntax "${TEST_INVALID_JSON}"
    assert_failure
    assert_output --partial "Invalid JSON syntax"
}

@test "schema_validator::validate_json_syntax fails for non-existent file" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    run validator::validate_json_syntax "/non/existent/file.json"
    assert_failure
    assert_output --partial "File not found"
}

@test "schema_validator::validate_structure accepts valid structure" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    run validator::validate_structure "${TEST_VALID_CONFIG}"
    assert_success
    assert_output --partial "Configuration structure is valid"
}

@test "schema_validator::validate_structure rejects missing scenarios" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    run validator::validate_structure "${TEST_MISSING_SCENARIOS}"
    assert_failure
    assert_output --partial "Missing required 'scenarios' property"
}

@test "schema_validator::validate_structure rejects invalid scenario structure" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    run validator::validate_structure "${TEST_INVALID_STRUCTURE}"
    assert_failure
    assert_output --partial "missing required 'version' property"
}

@test "schema_validator::validate_structure validates version format" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    # Create config with invalid version format
    local invalid_version="${VROOLI_TEST_TMPDIR}/invalid-version.json"
    cat > "${invalid_version}" << 'EOF'
{
  "scenarios": {
    "test": {
      "description": "Test",
      "version": "invalid-version",
      "resources": {}
    }
  }
}
EOF
    
    run validator::validate_structure "${invalid_version}"
    assert_failure
    assert_output --partial "invalid version format"
}

@test "schema_validator::check_dependencies_cycle detects circular dependencies" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    run validator::check_dependencies_cycle "scenario-a" "${TEST_CIRCULAR_DEPS}"
    assert_failure
    assert_output --partial "Circular dependency detected"
}

@test "schema_validator::check_dependencies_cycle accepts valid dependencies" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    run validator::check_dependencies_cycle "another-scenario" "${TEST_VALID_CONFIG}"
    assert_success
}

@test "schema_validator::check_dependencies_cycle detects non-existent dependencies" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    # Create config with non-existent dependency
    local bad_deps="${VROOLI_TEST_TMPDIR}/bad-deps.json"
    cat > "${bad_deps}" << 'EOF'
{
  "scenarios": {
    "test": {
      "description": "Test",
      "version": "1.0.0",
      "dependencies": ["non-existent"],
      "resources": {}
    }
  }
}
EOF
    
    run validator::check_dependencies_cycle "test" "${bad_deps}"
    assert_failure
    assert_output --partial "depends on non-existent scenario"
}

@test "schema_validator::validate_config performs complete validation" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    run validator::validate_config "${TEST_VALID_CONFIG}"
    assert_success
    assert_output --partial "Configuration file is valid"
}

@test "schema_validator::validate_config fails for invalid files" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    run validator::validate_config "${TEST_INVALID_JSON}"
    assert_failure
}

@test "schema_validator::check_schema validates schema file syntax" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    # Mock the SCHEMA_FILE variable to point to our test schema
    SCHEMA_FILE="${TEST_SCHEMA_FILE}"
    
    run validator::check_schema
    assert_success
    assert_output --partial "Schema file is valid"
}

@test "schema_validator::check_dependencies verifies jq availability" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    # Mock system::is_command to simulate jq being available
    system::is_command() {
        [[ "$1" == "jq" ]]
    }
    export -f system::is_command
    
    run validator::check_dependencies
    assert_success
}

@test "schema_validator::check_dependencies fails when jq unavailable" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    # Mock system::is_command to simulate jq being unavailable
    system::is_command() {
        [[ "$1" != "jq" ]]
    }
    export -f system::is_command
    
    run validator::check_dependencies
    assert_failure
    assert_output --partial "jq command not available"
}

@test "schema_validator::parse_arguments handles action parameter" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    # Test that parsing arguments works
    validator::parse_arguments --action validate --config-file "${TEST_VALID_CONFIG}"
    
    [ "${ACTION}" = "validate" ]
    [ "${CONFIG_FILE}" = "${TEST_VALID_CONFIG}" ]
}

@test "schema_validator::parse_arguments sets defaults" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    validator::parse_arguments
    
    [ "${ACTION}" = "validate" ]
    [ "${VERBOSE}" = "no" ]
}

@test "schema_validator::main executes validate action" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    # Mock check_dependencies to return success
    validator::check_dependencies() { return 0; }
    export -f validator::check_dependencies
    
    run validator::main --action validate --config-file "${TEST_VALID_CONFIG}"
    assert_success
}

@test "schema_validator::main executes check-schema action" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    # Mock check_dependencies and set test schema
    validator::check_dependencies() { return 0; }
    export -f validator::check_dependencies
    SCHEMA_FILE="${TEST_SCHEMA_FILE}"
    
    run validator::main --action check-schema
    assert_success
    assert_output --partial "Schema file is valid"
}

@test "schema_validator::main handles unknown action" {
    source "${BATS_TEST_DIRNAME}/schema-validator.sh"
    
    validator::check_dependencies() { return 0; }
    export -f validator::check_dependencies
    
    run validator::main --action unknown-action
    assert_failure
    assert_output --partial "Unknown action"
}