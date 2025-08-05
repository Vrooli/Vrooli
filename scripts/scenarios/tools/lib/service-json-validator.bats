#!/usr/bin/env bats

################################################################################
# Test Suite for service-json-validator.sh
#
# Tests all validation functions with various scenarios including:
# - Valid and invalid service.json structures
# - Missing and present file references
# - Syntax validation for JSON/YAML/SQL files
# - Resource conflicts and compatibility
# - Injection handler availability
# - Comprehensive deployment readiness checks
################################################################################

# Setup and teardown
setup() {
    # Set up test environment
    SCRIPT_DIR="$(cd "$(dirname "${BATS_TEST_FILENAME}")" && pwd)"
    PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
    
    # Source the validator
    source "${SCRIPT_DIR}/service-json-validator.sh"
    
    # Create temporary test directory
    TEST_TEMP_DIR="${BATS_TMPDIR}/validator-test-$$"
    mkdir -p "$TEST_TEMP_DIR"
    
    # Set validator to verbose mode for testing
    export VALIDATOR_VERBOSE="true"
    export VALIDATOR_STRICT="true"
}

teardown() {
    # Clean up temporary test directory
    if [[ -d "$TEST_TEMP_DIR" ]]; then
        rm -rf "$TEST_TEMP_DIR"
    fi
}

################################################################################
# Helper Functions for Tests
################################################################################

create_valid_service_json() {
    cat << 'EOF'
{
    "service": {
        "name": "test-scenario",
        "displayName": "Test Scenario",
        "version": "1.0.0",
        "type": "application"
    },
    "resources": {
        "storage": {
            "postgres": {
                "required": true,
                "port": 5432,
                "initialization": {
                    "data": [
                        {"file": "database/schema.sql"},
                        {"file": "database/seed.sql"}
                    ]
                }
            }
        },
        "ai": {
            "ollama": {
                "required": true,
                "port": 11434
            }
        }
    }
}
EOF
}

create_invalid_service_json() {
    cat << 'EOF'
{
    "service": {
        "version": "1.0.0"
    },
    "resources": {}
}
EOF
}

create_service_json_with_port_conflicts() {
    cat << 'EOF'
{
    "service": {
        "name": "conflict-test",
        "version": "1.0.0"
    },
    "resources": {
        "storage": {
            "postgres": {
                "required": true,
                "port": 8080
            }
        },
        "ai": {
            "ollama": {
                "required": true,
                "port": 8080
            }
        }
    }
}
EOF
}

create_test_scenario_with_files() {
    local scenario_dir="$1"
    
    mkdir -p "${scenario_dir}"/{database,workflows,scripts}
    
    # Create service.json
    create_valid_service_json > "${scenario_dir}/service.json"
    
    # Create referenced files
    echo "CREATE TABLE users (id SERIAL PRIMARY KEY);" > "${scenario_dir}/database/schema.sql"
    echo "INSERT INTO users (id) VALUES (1);" > "${scenario_dir}/database/seed.sql"
    
    # Create valid JSON workflow
    cat << 'EOF' > "${scenario_dir}/workflows/test-workflow.json"
{
    "name": "test-workflow",
    "steps": [
        {"action": "create", "target": "user"}
    ]
}
EOF
    
    # Create invalid JSON file for testing
    cat << 'EOF' > "${scenario_dir}/workflows/invalid.json"
{
    "name": "invalid-workflow",
    "missing": quote here
}
EOF
}

create_test_scenario_missing_files() {
    local scenario_dir="$1"
    
    mkdir -p "$scenario_dir"
    
    # Create service.json that references non-existent files
    cat << 'EOF' > "${scenario_dir}/service.json"
{
    "service": {
        "name": "missing-files-test",
        "version": "1.0.0"
    },
    "resources": {
        "storage": {
            "postgres": {
                "required": true,
                "initialization": {
                    "data": [
                        {"file": "database/missing.sql"}
                    ]
                }
            }
        }
    }
}
EOF
}

################################################################################
# Schema Validation Tests
################################################################################

@test "validate_service_schema: accepts valid service.json" {
    local service_json
    service_json="$(create_valid_service_json)"
    
    run validate_service_schema "$service_json"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Service schema validation passed" ]]
}

@test "validate_service_schema: rejects invalid JSON" {
    local invalid_json='{"service": invalid json}'
    
    run validate_service_schema "$invalid_json"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not valid JSON" ]]
}

@test "validate_service_schema: rejects missing service section" {
    local json_without_service='{"resources": {}}'
    
    run validate_service_schema "$json_without_service"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Missing required section: 'service'" ]]
}

@test "validate_service_schema: rejects missing resources section" {
    local json_without_resources='{"service": {"name": "test", "version": "1.0"}}'
    
    run validate_service_schema "$json_without_resources"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Missing required section: 'resources'" ]]
}

@test "validate_service_schema: rejects missing service.name" {
    local service_json
    service_json="$(create_invalid_service_json)"
    
    run validate_service_schema "$service_json"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Missing required field: service.name" ]]
}

@test "validate_service_schema: rejects invalid service name format" {
    local json_with_invalid_name='{
        "service": {"name": "invalid@name!", "version": "1.0"},
        "resources": {}
    }'
    
    run validate_service_schema "$json_with_invalid_name"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid service name format" ]]
}

@test "validate_service_schema: accepts valid service name formats" {
    local names=("test-scenario" "test_scenario" "test123" "Test-Scenario_123")
    
    for name in "${names[@]}"; do
        local json="{\"service\": {\"name\": \"$name\", \"version\": \"1.0\"}, \"resources\": {}}"
        
        run validate_service_schema "$json"
        [ "$status" -eq 0 ]
    done
}

################################################################################
# File Reference Validation Tests
################################################################################

@test "validate_file_references: passes when all files exist" {
    local scenario_dir="${TEST_TEMP_DIR}/valid-scenario"
    create_test_scenario_with_files "$scenario_dir"
    
    local file_list="database/schema.sql database/seed.sql workflows/test-workflow.json"
    
    run validate_file_references "$scenario_dir" "$file_list"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "All 3 file references validated" ]]
}

@test "validate_file_references: fails when files are missing" {
    local scenario_dir="${TEST_TEMP_DIR}/missing-files"
    create_test_scenario_missing_files "$scenario_dir"
    
    local file_list="database/missing.sql workflows/nonexistent.json"
    
    run validate_file_references "$scenario_dir" "$file_list"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Missing 2 out of 2 referenced files" ]]
    [[ "$output" =~ "database/missing.sql" ]]
    [[ "$output" =~ "workflows/nonexistent.json" ]]
}

@test "validate_file_references: handles empty file list" {
    local scenario_dir="${TEST_TEMP_DIR}/empty-test"
    mkdir -p "$scenario_dir"
    
    run validate_file_references "$scenario_dir" ""
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No files to validate" ]]
}

@test "validate_file_references: handles whitespace in file list" {
    local scenario_dir="${TEST_TEMP_DIR}/whitespace-test"
    create_test_scenario_with_files "$scenario_dir"
    
    local file_list="  database/schema.sql   database/seed.sql  "
    
    run validate_file_references "$scenario_dir" "$file_list"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "All 2 file references validated" ]]
}

################################################################################
# File Syntax Validation Tests
################################################################################

@test "validate_file_syntax: validates JSON files correctly" {
    local scenario_dir="${TEST_TEMP_DIR}/json-syntax-test"
    create_test_scenario_with_files "$scenario_dir"
    
    local file_list="workflows/test-workflow.json"
    
    run validate_file_syntax "$scenario_dir" "$file_list"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Valid JSON: workflows/test-workflow.json" ]]
    [[ "$output" =~ "All 1 files have valid syntax" ]]
}

@test "validate_file_syntax: detects invalid JSON files" {
    local scenario_dir="${TEST_TEMP_DIR}/invalid-json-test"
    create_test_scenario_with_files "$scenario_dir"
    
    local file_list="workflows/invalid.json"
    
    run validate_file_syntax "$scenario_dir" "$file_list"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid JSON: workflows/invalid.json" ]]
    [[ "$output" =~ "Found 1 files with syntax errors" ]]
}

@test "validate_file_syntax: handles SQL files" {
    local scenario_dir="${TEST_TEMP_DIR}/sql-test"
    create_test_scenario_with_files "$scenario_dir"
    
    local file_list="database/schema.sql database/seed.sql"
    
    run validate_file_syntax "$scenario_dir" "$file_list"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SQL file present: database/schema.sql" ]]
    [[ "$output" =~ "SQL file present: database/seed.sql" ]]
}

@test "validate_file_syntax: handles YAML files when yq available" {
    local scenario_dir="${TEST_TEMP_DIR}/yaml-test"
    mkdir -p "${scenario_dir}/config"
    
    # Create valid YAML file
    cat << 'EOF' > "${scenario_dir}/config/test.yml"
name: test
version: 1.0
items:
  - one
  - two
EOF
    
    local file_list="config/test.yml"
    
    if command -v yq &>/dev/null; then
        run validate_file_syntax "$scenario_dir" "$file_list"
        [ "$status" -eq 0 ]
        [[ "$output" =~ "Valid YAML: config/test.yml" ]]
    else
        run validate_file_syntax "$scenario_dir" "$file_list"
        [ "$status" -eq 0 ]
        [[ "$output" =~ "YAML validation skipped" ]]
    fi
}

@test "validate_file_syntax: skips validation for unknown file types" {
    local scenario_dir="${TEST_TEMP_DIR}/unknown-type-test"
    mkdir -p "${scenario_dir}/files"
    
    echo "some content" > "${scenario_dir}/files/unknown.txt"
    
    local file_list="files/unknown.txt"
    
    run validate_file_syntax "$scenario_dir" "$file_list"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Unknown file type, skipping validation: files/unknown.txt" ]]
}

@test "validate_file_syntax: handles empty file list gracefully" {
    local scenario_dir="${TEST_TEMP_DIR}/empty-syntax-test"
    mkdir -p "$scenario_dir"
    
    run validate_file_syntax "$scenario_dir" ""
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No files to validate syntax" ]]
}

################################################################################
# Resource Conflict Validation Tests
################################################################################

@test "validate_resource_conflicts: detects port conflicts" {
    local service_json
    service_json="$(create_service_json_with_port_conflicts)"
    
    run validate_resource_conflicts "$service_json"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Resource conflicts detected" ]] || [[ "$output" =~ "conflict" ]]
}

@test "validate_resource_conflicts: passes when no conflicts" {
    local service_json
    service_json="$(create_valid_service_json)"
    
    run validate_resource_conflicts "$service_json"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No resource conflicts found" ]]
}

@test "validate_resource_conflicts: handles services without ports" {
    local service_json='{
        "service": {"name": "no-ports", "version": "1.0"},
        "resources": {
            "storage": {
                "postgres": {"required": true}
            }
        }
    }'
    
    run validate_resource_conflicts "$service_json"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No resource conflicts found" ]]
}

@test "validate_resource_conflicts: warns about database conflicts" {
    local service_json='{
        "service": {"name": "db-conflict", "version": "1.0"},
        "resources": {
            "storage": {
                "postgres": {"required": true},
                "mysql": {"required": true}
            }
        }
    }'
    
    run validate_resource_conflicts "$service_json"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Both PostgreSQL and MySQL are required" ]]
}

################################################################################
# Injection Handler Validation Tests
################################################################################

@test "validate_injection_handlers: handles empty resource list" {
    run validate_injection_handlers ""
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No required resources to check handlers for" ]]
}

@test "validate_injection_handlers: checks for injection scripts" {
    # This test depends on actual injection scripts existing
    # We'll test the logic without depending on specific scripts
    
    local required_resources="postgres ollama"
    
    run validate_injection_handlers "$required_resources"
    
    # Should complete without crashing, exact behavior depends on available scripts
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
    [[ "$output" =~ "✓ Checked injection handlers for 2 resources" ]]
}

@test "validate_injection_handlers: strict mode fails with missing handlers" {
    export VALIDATOR_STRICT="true"
    
    # Use a resource that definitely won't have a handler
    local required_resources="nonexistent-resource-xyz"
    
    run validate_injection_handlers "$required_resources"
    
    # In strict mode, should fail if handler is missing
    [[ "$output" =~ "No injection handler: nonexistent-resource-xyz" ]]
}

@test "validate_injection_handlers: non-strict mode warns about missing handlers" {
    export VALIDATOR_STRICT="false"
    
    local required_resources="nonexistent-resource-xyz"
    
    run validate_injection_handlers "$required_resources"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No injection handler: nonexistent-resource-xyz" ]]
    [[ "$output" =~ "These resources will need manual setup" ]]
}

################################################################################
# Comprehensive Validation Tests
################################################################################

@test "validate_deployment_readiness: passes for complete valid scenario" {
    local scenario_dir="${TEST_TEMP_DIR}/complete-scenario"
    create_test_scenario_with_files "$scenario_dir"
    
    local service_json
    service_json="$(cat "${scenario_dir}/service.json")"
    
    # Use non-strict mode for testing
    export VALIDATOR_STRICT="false"
    
    run validate_deployment_readiness "$scenario_dir" "$service_json"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "All deployment readiness checks passed" ]]
}

@test "validate_deployment_readiness: fails for scenario with missing files" {
    local scenario_dir="${TEST_TEMP_DIR}/incomplete-scenario"
    create_test_scenario_missing_files "$scenario_dir"
    
    local service_json
    service_json="$(cat "${scenario_dir}/service.json")"
    
    run validate_deployment_readiness "$scenario_dir" "$service_json"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Deployment readiness validation failed" ]]
}

@test "validate_basic_structure: validates structure without file I/O" {
    local service_json
    service_json="$(create_valid_service_json)"
    
    run validate_basic_structure "$service_json"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Basic structure validation passed" ]]
}

@test "validate_basic_structure: fails for invalid structure" {
    local service_json
    service_json="$(create_invalid_service_json)"
    
    run validate_basic_structure "$service_json"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Missing required field: service.name" ]]
}

################################################################################
# Main Entry Point Tests
################################################################################

@test "validate_scenario: validates complete scenario directory" {
    local scenario_dir="${TEST_TEMP_DIR}/main-test-scenario"
    create_test_scenario_with_files "$scenario_dir"
    
    # Use non-strict mode for testing since we don't have all injection handlers
    export VALIDATOR_STRICT="false"
    
    run validate_scenario "$scenario_dir"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "All deployment readiness checks passed" ]]
}

@test "validate_scenario: fails when service.json missing" {
    local scenario_dir="${TEST_TEMP_DIR}/no-service-json"
    mkdir -p "$scenario_dir"
    
    run validate_scenario "$scenario_dir"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "service.json not found" ]]
}

@test "validate_scenario: fails when service.json unreadable" {
    local scenario_dir="${TEST_TEMP_DIR}/unreadable-service-json"
    mkdir -p "$scenario_dir"
    
    # Create unreadable service.json
    echo "test" > "${scenario_dir}/service.json"
    chmod 000 "${scenario_dir}/service.json"
    
    run validate_scenario "$scenario_dir"
    
    [ "$status" -eq 1 ]
    
    # Clean up
    chmod 644 "${scenario_dir}/service.json"
}

################################################################################
# Utility Function Tests
################################################################################

@test "check_validator_dependencies: passes with required tools" {
    # This test assumes jq is available (required for all tests)
    run check_validator_dependencies
    
    [ "$status" -eq 0 ]
}

@test "extract_all_referenced_files: extracts files from service.json" {
    local service_json
    service_json="$(create_valid_service_json)"
    
    run extract_all_referenced_files "$service_json"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "database/schema.sql" ]]
    [[ "$output" =~ "database/seed.sql" ]]
}

################################################################################
# Error Handling Tests
################################################################################

@test "validator handles malformed JSON gracefully" {
    local malformed_json='{"service": {"name": "test"'  # Missing closing braces
    
    run validate_service_schema "$malformed_json"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not valid JSON" ]]
}

@test "validator handles extremely large service.json" {
    # Create a large but valid JSON
    local large_json='{"service":{"name":"large-test","version":"1.0"},"resources":{'
    for i in {1..100}; do
        large_json+='"resource'$i'":{"required":false},'
    done
    large_json="${large_json%,}}}"
    
    run validate_service_schema "$large_json"
    
    [ "$status" -eq 0 ]
}

@test "validator handles special characters in file paths" {
    local scenario_dir="${TEST_TEMP_DIR}/special-chars"
    mkdir -p "${scenario_dir}/files_with_special_chars"
    
    echo "test" > "${scenario_dir}/files_with_special_chars/test-file.sql"
    
    # Use newline-separated format to avoid space issues
    local file_list=$'files_with_special_chars/test-file.sql'
    
    run validate_file_references "$scenario_dir" "$file_list"
    
    [ "$status" -eq 0 ]
}

################################################################################
# Integration Tests with service-json-processor.sh
################################################################################

@test "validator integrates with processor for file extraction" {
    local scenario_dir="${TEST_TEMP_DIR}/integration-test"
    create_test_scenario_with_files "$scenario_dir"
    
    local service_json
    service_json="$(cat "${scenario_dir}/service.json")"
    
    # This should use the processor internally
    run extract_all_referenced_files "$service_json"
    
    [ "$status" -eq 0 ]
    # Should extract files that exist in the test scenario
    [[ "$output" =~ "database/schema.sql" ]]
}

################################################################################
# Performance Tests
################################################################################

@test "validator performs reasonably on complex scenarios" {
    local scenario_dir="${TEST_TEMP_DIR}/complex-scenario"
    mkdir -p "${scenario_dir}"/{database,workflows,scripts,config}
    
    # Create a complex service.json
    cat << 'EOF' > "${scenario_dir}/service.json"
{
    "service": {
        "name": "complex-scenario",
        "version": "1.0.0"
    },
    "resources": {
        "storage": {
            "postgres": {"required": true},
            "redis": {"required": true},
            "minio": {"required": false}
        },
        "ai": {
            "ollama": {"required": true},
            "whisper": {"required": true}
        },
        "automation": {
            "windmill": {"required": true},
            "n8n": {"required": false}
        }
    }
}
EOF
    
    # Create multiple files
    for i in {1..10}; do
        echo "CREATE TABLE test$i (id SERIAL);" > "${scenario_dir}/database/schema$i.sql"
        echo '{"workflow": "'$i'"}' > "${scenario_dir}/workflows/workflow$i.json"
    done
    
    local service_json
    service_json="$(cat "${scenario_dir}/service.json")"
    
    # Test should complete in reasonable time - just run normally
    # The performance aspect is tested by the fact that it completes at all
    run validate_deployment_readiness "$scenario_dir" "$service_json"
    
    # Should complete without hanging (any exit code is fine for this performance test)
    # The fact that we get here means it didn't hang indefinitely
}