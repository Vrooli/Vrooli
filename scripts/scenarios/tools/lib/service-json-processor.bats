#!/usr/bin/env bats

################################################################################
# Test Suite for Service JSON Processor
#
# Comprehensive tests for the service-json-processor.sh module, covering:
# - JSON validation and parsing
# - Resource extraction and filtering
# - File reference resolution
# - Error handling and edge cases
# - Resource conflict detection
#
################################################################################

# Set up test environment
setup() {
    # Load the module under test
    SCRIPT_DIR="$(cd "$(dirname "${BATS_TEST_FILENAME}")" && pwd)"
    source "${SCRIPT_DIR}/service-json-processor.sh"
    
    # Create temporary directory for test files
    TEST_TEMP_DIR="$(mktemp -d)"
    
    # Create valid service.json test content
    VALID_SERVICE_JSON='{
        "service": {
            "name": "test-service",
            "version": "1.0.0",
            "displayName": "Test Service",
            "description": "A test service",
            "type": "application"
        },
        "resources": {
            "storage": {
                "postgres": {
                    "enabled": true,
                    "required": true,
                    "type": "postgresql",
                    "port": 5432,
                    "initialization": {
                        "data": [
                            {
                                "type": "schema",
                                "file": "database/schema.sql"
                            },
                            {
                                "type": "seed", 
                                "file": "database/seed.sql"
                            }
                        ]
                    }
                },
                "redis": {
                    "enabled": false,
                    "required": false,
                    "port": 6379
                }
            },
            "automation": {
                "windmill": {
                    "enabled": true,
                    "required": true,
                    "initialization": {
                        "scripts": [
                            {
                                "name": "process-data",
                                "file": "scripts/process-data.ts"
                            }
                        ],
                        "apps": [
                            {
                                "name": "dashboard",
                                "file": "ui/dashboard.json"
                            }
                        ]
                    }
                },
                "comfyui": {
                    "enabled": true,
                    "required": false,
                    "initialization": {
                        "workflows": [
                            {
                                "name": "image-gen",
                                "file": "workflows/image-generation.json"
                            }
                        ]
                    }
                }
            },
            "ai": {
                "ollama": {
                    "enabled": true,
                    "required": true,
                    "models": ["llama3.1:8b"]
                }
            }
        }
    }'
    
    # Create invalid JSON content
    INVALID_SERVICE_JSON='{
        "service": {
            "name": "test-service",
            "invalid": json syntax here
        }
    }'
    
    # Create minimal valid JSON
    MINIMAL_SERVICE_JSON='{
        "service": {
            "name": "minimal-service",
            "version": "1.0.0"
        },
        "resources": {}
    }'
    
    # Create JSON with port conflicts
    CONFLICT_SERVICE_JSON='{
        "service": {
            "name": "conflict-service",
            "version": "1.0.0"
        },
        "resources": {
            "storage": {
                "postgres": {
                    "enabled": true,
                    "port": 5432
                },
                "custom-db": {
                    "enabled": true,
                    "port": 5432
                }
            }
        }
    }'
    
    # Write test files
    echo "$VALID_SERVICE_JSON" > "$TEST_TEMP_DIR/valid-service.json"
    echo "$INVALID_SERVICE_JSON" > "$TEST_TEMP_DIR/invalid-service.json"
    echo "$MINIMAL_SERVICE_JSON" > "$TEST_TEMP_DIR/minimal-service.json"
    echo "$CONFLICT_SERVICE_JSON" > "$TEST_TEMP_DIR/conflict-service.json"
}

# Cleanup after each test
teardown() {
    rm -rf "$TEST_TEMP_DIR"
}

################################################################################
# Dependency and Basic Validation Tests
################################################################################

@test "sjp_check_dependencies: jq is available" {
    run sjp_check_dependencies
    [ "$status" -eq 0 ]
}

@test "sjp_validate_json_syntax: valid JSON passes" {
    run sjp_validate_json_syntax "$VALID_SERVICE_JSON"
    [ "$status" -eq 0 ]
}

@test "sjp_validate_json_syntax: invalid JSON fails" {
    run sjp_validate_json_syntax "$INVALID_SERVICE_JSON"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Invalid JSON syntax" ]]
}

@test "sjp_validate_json_syntax: empty content fails" {
    run sjp_validate_json_syntax ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Empty JSON content" ]]
}

@test "sjp_validate_json_file: valid file passes" {
    run sjp_validate_json_file "$TEST_TEMP_DIR/valid-service.json"
    [ "$status" -eq 0 ]
}

@test "sjp_validate_json_file: invalid file fails" {
    run sjp_validate_json_file "$TEST_TEMP_DIR/invalid-service.json"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Invalid JSON in file" ]]
}

@test "sjp_validate_json_file: missing file fails" {
    run sjp_validate_json_file "$TEST_TEMP_DIR/nonexistent.json"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: File not found" ]]
}

################################################################################
# Service Information Extraction Tests
################################################################################

@test "sjp_get_service_info: extracts service name" {
    run sjp_get_service_info "$VALID_SERVICE_JSON" "name"
    [ "$status" -eq 0 ]
    [ "$output" = "test-service" ]
}

@test "sjp_get_service_info: extracts service version" {
    run sjp_get_service_info "$VALID_SERVICE_JSON" "version"
    [ "$status" -eq 0 ]
    [ "$output" = "1.0.0" ]
}

@test "sjp_get_service_info: extracts display name" {
    run sjp_get_service_info "$VALID_SERVICE_JSON" "displayName"
    [ "$status" -eq 0 ]
    [ "$output" = "Test Service" ]
}

@test "sjp_get_service_info: extracts description" {
    run sjp_get_service_info "$VALID_SERVICE_JSON" "description"
    [ "$status" -eq 0 ]
    [ "$output" = "A test service" ]
}

@test "sjp_get_service_info: extracts service type" {
    run sjp_get_service_info "$VALID_SERVICE_JSON" "type"
    [ "$status" -eq 0 ]
    [ "$output" = "application" ]
}

@test "sjp_get_service_info: unknown field fails" {
    run sjp_get_service_info "$VALID_SERVICE_JSON" "unknown"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Unknown service field" ]]
}

@test "sjp_get_service_info: missing field fails" {
    run sjp_get_service_info "$MINIMAL_SERVICE_JSON" "description"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Service field 'description' not found" ]]
}

################################################################################
# Resource Category and Discovery Tests
################################################################################

@test "sjp_get_resource_categories: extracts all categories" {
    run sjp_get_resource_categories "$VALID_SERVICE_JSON"
    [ "$status" -eq 0 ]
    # Should contain storage, automation, ai
    [[ "$output" =~ "storage" ]]
    [[ "$output" =~ "automation" ]]
    [[ "$output" =~ "ai" ]]
}

@test "sjp_get_resource_categories: handles empty resources" {
    run sjp_get_resource_categories "$MINIMAL_SERVICE_JSON"
    [ "$status" -eq 0 ]
    [ "$output" = "" ]
}

@test "sjp_get_resources_in_category: extracts storage resources" {
    run sjp_get_resources_in_category "$VALID_SERVICE_JSON" "storage"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgres" ]]
    [[ "$output" =~ "redis" ]]
}

@test "sjp_get_resources_in_category: extracts automation resources" {
    run sjp_get_resources_in_category "$VALID_SERVICE_JSON" "automation"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "windmill" ]]
    [[ "$output" =~ "comfyui" ]]
}

@test "sjp_get_resources_in_category: handles nonexistent category" {
    run sjp_get_resources_in_category "$VALID_SERVICE_JSON" "nonexistent"
    [ "$status" -eq 0 ]
    [ "$output" = "" ]
}

################################################################################
# Resource Property and Condition Tests
################################################################################

@test "sjp_resource_has_property: detects required property" {
    run sjp_resource_has_property "$VALID_SERVICE_JSON" "storage" "postgres" "required"
    [ "$status" -eq 0 ]
}

@test "sjp_resource_has_property: detects enabled property" {
    run sjp_resource_has_property "$VALID_SERVICE_JSON" "storage" "postgres" "enabled"
    [ "$status" -eq 0 ]
}

@test "sjp_resource_has_property: false for disabled resource" {
    run sjp_resource_has_property "$VALID_SERVICE_JSON" "storage" "redis" "required"
    [ "$status" -eq 1 ]
}

@test "sjp_get_resources_by_condition: finds required resources" {
    run sjp_get_resources_by_condition "$VALID_SERVICE_JSON" "required == true"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "storage.postgres" ]]
    [[ "$output" =~ "automation.windmill" ]]
    [[ "$output" =~ "ai.ollama" ]]
}

@test "sjp_get_resources_by_condition: finds enabled resources" {
    run sjp_get_resources_by_condition "$VALID_SERVICE_JSON" "enabled == true"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "storage.postgres" ]]
    [[ "$output" =~ "automation.windmill" ]]
    [[ "$output" =~ "automation.comfyui" ]]
    [[ "$output" =~ "ai.ollama" ]]
}

@test "sjp_get_resources_by_condition: finds resources with initialization" {
    run sjp_get_resources_by_condition "$VALID_SERVICE_JSON" "initialization != null"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "storage.postgres" ]]
    [[ "$output" =~ "automation.windmill" ]]
    [[ "$output" =~ "automation.comfyui" ]]
    # Should not include ai.ollama (no initialization) or storage.redis
}

################################################################################
# File Reference Extraction Tests
################################################################################

@test "sjp_get_data_files: extracts all data files" {
    run sjp_get_data_files "$VALID_SERVICE_JSON"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "database/schema.sql" ]]
    [[ "$output" =~ "database/seed.sql" ]]
}

@test "sjp_get_data_files: extracts specific resource data files" {
    run sjp_get_data_files "$VALID_SERVICE_JSON" "storage" "postgres"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "database/schema.sql" ]]
    [[ "$output" =~ "database/seed.sql" ]]
}

@test "sjp_get_workflow_files: extracts workflow files" {
    run sjp_get_workflow_files "$VALID_SERVICE_JSON"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "workflows/image-generation.json" ]]
}

@test "sjp_get_script_files: extracts script files" {
    run sjp_get_script_files "$VALID_SERVICE_JSON"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "scripts/process-data.ts" ]]
}

@test "sjp_get_app_files: extracts app files" {
    run sjp_get_app_files "$VALID_SERVICE_JSON"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "ui/dashboard.json" ]]
}

@test "sjp_get_all_referenced_files: extracts all file types" {
    run sjp_get_all_referenced_files "$VALID_SERVICE_JSON"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "database/schema.sql" ]]
    [[ "$output" =~ "database/seed.sql" ]]
    [[ "$output" =~ "workflows/image-generation.json" ]]
    [[ "$output" =~ "scripts/process-data.ts" ]]
    [[ "$output" =~ "ui/dashboard.json" ]]
}

@test "sjp_get_data_files: handles missing initialization" {
    run sjp_get_data_files "$MINIMAL_SERVICE_JSON"
    [ "$status" -eq 0 ]
    [ "$output" = "" ]
}

################################################################################
# Resource Analysis Tests
################################################################################

@test "sjp_get_resource_summary: generates complete summary" {
    run sjp_get_resource_summary "$VALID_SERVICE_JSON" "storage" "postgres"
    [ "$status" -eq 0 ]
    
    # Parse JSON output and verify fields
    local summary="$output"
    local category=$(echo "$summary" | jq -r '.category')
    local resource=$(echo "$summary" | jq -r '.resource')
    local enabled=$(echo "$summary" | jq -r '.enabled')
    local required=$(echo "$summary" | jq -r '.required')
    local has_init=$(echo "$summary" | jq -r '.has_initialization')
    
    [ "$category" = "storage" ]
    [ "$resource" = "postgres" ]
    [ "$enabled" = "true" ]
    [ "$required" = "true" ]
    [ "$has_init" = "true" ]
    
    # Check initialization types
    [[ "$summary" =~ "data" ]]
}

@test "sjp_get_resource_summary: handles resource without initialization" {
    run sjp_get_resource_summary "$VALID_SERVICE_JSON" "ai" "ollama"
    [ "$status" -eq 0 ]
    
    local has_init=$(echo "$output" | jq -r '.has_initialization')
    [ "$has_init" = "false" ]
}

@test "sjp_check_resource_conflicts: detects port conflicts" {
    run sjp_check_resource_conflicts "$CONFLICT_SERVICE_JSON"
    [ "$status" -eq 0 ]
    
    # Should detect port 5432 conflict
    [[ "$output" =~ "port_conflict" ]]
    [[ "$output" =~ "5432" ]]
}

@test "sjp_check_resource_conflicts: no conflicts in valid config" {
    run sjp_check_resource_conflicts "$VALID_SERVICE_JSON"
    [ "$status" -eq 0 ]
    
    # Should return empty array
    [ "$output" = "[]" ]
}

################################################################################
# Error Handling and Edge Cases
################################################################################

@test "sjp_validate_json_syntax: handles null input gracefully" {
    run sjp_validate_json_syntax
    [ "$status" -eq 1 ]
}

@test "sjp_get_service_info: handles invalid JSON" {
    run sjp_get_service_info "$INVALID_SERVICE_JSON" "name"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Invalid JSON syntax" ]]
}

@test "sjp_get_resource_categories: handles invalid JSON" {
    run sjp_get_resource_categories "$INVALID_SERVICE_JSON"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Invalid JSON syntax" ]]
}

@test "sjp_get_data_files: handles JSON without resources" {
    local no_resources_json='{"service": {"name": "test", "version": "1.0.0"}}'
    run sjp_get_data_files "$no_resources_json"
    [ "$status" -eq 0 ]
    [ "$output" = "" ]
}

@test "sjp_get_resources_by_condition: handles complex conditions" {
    run sjp_get_resources_by_condition "$VALID_SERVICE_JSON" "enabled == true and required == true"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "storage.postgres" ]]
    [[ "$output" =~ "automation.windmill" ]]
    [[ "$output" =~ "ai.ollama" ]]
    # Should not include automation.comfyui (required == false)
}

################################################################################
# CLI Interface Tests
################################################################################

@test "main: check-deps command works" {
    run main check-deps
    [ "$status" -eq 0 ]
}

@test "main: validate-file command works" {
    run main validate-file "$TEST_TEMP_DIR/valid-service.json"
    [ "$status" -eq 0 ]
}

@test "main: validate-file command detects invalid files" {
    run main validate-file "$TEST_TEMP_DIR/invalid-service.json"
    [ "$status" -eq 1 ]
}

@test "main: get-service-info command works" {
    run main get-service-info "$TEST_TEMP_DIR/valid-service.json" "name"
    [ "$status" -eq 0 ]
    [ "$output" = "test-service" ]
}

@test "main: get-categories command works" {
    run main get-categories "$TEST_TEMP_DIR/valid-service.json"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "storage" ]]
    [[ "$output" =~ "automation" ]]
}

@test "main: get-resources command works" {
    run main get-resources "$TEST_TEMP_DIR/valid-service.json" "storage"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "postgres" ]]
}

@test "main: get-files command works for all types" {
    for file_type in data workflows scripts apps all; do
        run main get-files "$TEST_TEMP_DIR/valid-service.json" "$file_type"
        [ "$status" -eq 0 ]
    done
}

@test "main: check-conflicts command works" {
    run main check-conflicts "$TEST_TEMP_DIR/conflict-service.json"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "port_conflict" ]]
}

@test "main: version command works" {
    run main version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Service JSON Processor" ]]
    [[ "$output" =~ "version" ]]
}

@test "main: unknown command fails" {
    run main unknown-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
}

@test "main: no arguments shows help" {
    run main
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
    [[ "$output" =~ "Commands:" ]]
}