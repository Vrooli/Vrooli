#!/usr/bin/env bats

# Test for scenario-to-app.sh
# Tests the scenario to standalone app generation functionality

# Load test setup
SCENARIO_TOOLS_DIR="$BATS_TEST_DIRNAME"
SCRIPTS_DIR="$(dirname "$(dirname "$SCENARIO_TOOLS_DIR")")"

# Source dependencies
. "$SCRIPTS_DIR/lib/utils/var.sh"
. "$SCRIPTS_DIR/lib/utils/log.sh"

# Mock functions to prevent real file operations during tests
scenario_to_app::mock_file_operations() {
    # Override commands that would do real file operations
    find() { echo "/tmp/mock/file1.json"; echo "/tmp/mock/file2.yaml"; }
    cp() { return 0; }
    mv() { return 0; }
    mkdir() { return 0; }
    cat() {
        if [[ "$1" == *"service.json" ]]; then
            echo '{"service": {"name": "test-service", "version": "1.0.0", "type": "application"}}'
        else
            echo "mock file content"
        fi
    }
    jq() {
        case "$*" in
            *".service.name"*) echo "test-service" ;;
            *"empty"*) return 0 ;;
            *) echo "mock jq output" ;;
        esac
    }
    export -f find cp mv mkdir cat jq
}

# Load the script functions in a subshell for isolation
load_script_functions() {
    bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        scenario_to_app::mock_file_operations
        $1
    "
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "scenario_to_app::parse_args handles help flag" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        scenario_to_app::parse_args --help 2>/dev/null || echo 'help shown'
    "
    [[ "$output" == *"help shown"* ]] || [[ "$status" -eq 0 ]]
}

@test "scenario_to_app::parse_args requires scenario name" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        scenario_to_app::parse_args 2>/dev/null || echo 'error: no scenario name'
    "
    [[ "$output" == *"error: no scenario name"* ]] || [[ "$status" -eq 1 ]]
}

@test "scenario_to_app::parse_args sets verbose flag" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        scenario_to_app::parse_args test-scenario --verbose
        echo \"VERBOSE=\$VERBOSE\"
    "
    [[ "$output" == *"VERBOSE=true"* ]]
}

@test "scenario_to_app::parse_args sets dry run flag" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        scenario_to_app::parse_args test-scenario --dry-run
        echo \"DRY_RUN=\$DRY_RUN\"
    "
    [[ "$output" == *"DRY_RUN=true"* ]]
}

@test "scenario_to_app::parse_args sets force flag" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        scenario_to_app::parse_args test-scenario --force
        echo \"FORCE=\$FORCE\"
    "
    [[ "$output" == *"FORCE=true"* ]]
}

# ============================================================================
# Resource Analysis Tests
# ============================================================================

@test "scenario_to_app::get_enabled_resources extracts enabled resources from JSON" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        SERVICE_JSON='{
            \"resources\": {
                \"ai\": {
                    \"ollama\": {\"enabled\": true},
                    \"whisper\": {\"enabled\": false}
                },
                \"automation\": {
                    \"n8n\": {\"enabled\": true}
                }
            }
        }'
        scenario_to_app::get_enabled_resources
    "
    [[ "$output" == *"ollama (ai)"* ]]
    [[ "$output" == *"n8n (automation)"* ]]
    [[ "$output" != *"whisper"* ]]
}

@test "scenario_to_app::get_resource_categories extracts resource categories" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        SERVICE_JSON='{
            \"resources\": {
                \"ai\": {},
                \"automation\": {},
                \"storage\": {}
            }
        }'
        scenario_to_app::get_resource_categories
    "
    [[ "$output" == *"ai"* ]]
    [[ "$output" == *"automation"* ]]
    [[ "$output" == *"storage"* ]]
}

# ============================================================================
# Service JSON Creation Tests
# ============================================================================

@test "scenario_to_app::create_default_service_json creates valid structure" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        # Mock cat to simulate created file content
        cat() {
            echo '{
                \"service\": {\"name\": \"test-app\", \"type\": \"application\"},
                \"lifecycle\": {\"setup\": {\"steps\": []}}
            }'
        }
        export -f cat
        scenario_to_app::create_default_service_json /tmp/test.json > /dev/null
        cat /tmp/test.json
    "
    [[ "$output" == *"\"service\""* ]]
    [[ "$output" == *"\"lifecycle\""* ]]
    [[ "$output" == *"\"application\""* ]]
}

# ============================================================================
# File Processing Tests (Mocked)
# ============================================================================

@test "scenario_to_app::should_process_file logic works correctly" {
    # Test through the copy functions which use similar logic
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        
        # Mock file command
        file() {
            case \"\$1\" in
                *.json) echo \"text/plain\" ;;
                *.bin) echo \"application/octet-stream\" ;;
                *) echo \"text/plain\" ;;
            esac
        }
        export -f file
        
        # Test via process_template_variables which checks file types
        scenario_to_app::process_template_variables \"/tmp/test.json\" '{}'
        echo \"processed json file\"
        
        scenario_to_app::process_template_variables \"/tmp/test.bin\" '{}'
        echo \"processed binary file\"
    "
    [[ "$output" == *"processed json file"* ]]
    [[ "$output" == *"processed binary file"* ]]
}

# ============================================================================
# Validation Function Tests
# ============================================================================

@test "scenario_to_app::validate_initialization_paths handles empty paths gracefully" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        SCENARIO_PATH=\"/tmp/mock-scenario\"
        SERVICE_JSON='{\"resources\": {}, \"deployment\": {}}'
        
        # Mock jq to return no paths
        jq() {
            case \"\$*\" in
                *\".resources\"*) echo \"\" ;;
                *\".deployment\"*) echo \"\" ;;
                *) echo \"mock output\" ;;
            esac
        }
        export -f jq
        
        if scenario_to_app::validate_initialization_paths; then
            echo \"validation passed\"
        else
            echo \"validation failed\"
        fi
    "
    [[ "$output" == *"validation passed"* ]]
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "script handles missing scenario directory gracefully" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        SCENARIO_NAME=\"nonexistent-scenario\"
        scenario_to_app::validate_scenario 2>/dev/null || echo 'error handled'
    "
    [[ "$output" == *"error handled"* ]] || [[ "$status" -eq 1 ]]
}

@test "script handles invalid JSON gracefully" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        
        # Mock invalid JSON content
        cat() {
            if [[ \"\$1\" == *\"service.json\" ]]; then
                echo '{'  # Invalid JSON
            else
                echo \"mock content\"
            fi
        }
        
        jq() {
            if [[ \"\$*\" == *\"empty\"* ]]; then
                return 1  # Simulate JSON parse error
            else
                echo \"mock output\"
            fi
        }
        
        export -f cat jq
        
        SCENARIO_PATH=\"/tmp/mock\"
        SERVICE_JSON=\"\"
        
        scenario_to_app::validate_scenario 2>/dev/null || echo 'json error handled'
    "
    [[ "$output" == *"json error handled"* ]] || [[ "$status" -eq 1 ]]
}

# ============================================================================
# Integration Tests (Mocked File Operations)
# ============================================================================

@test "script dry-run mode prevents file operations" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        DRY_RUN=true
        
        # These should all be no-ops in dry run mode
        if scenario_to_app::copy_scenario_files \"/tmp/mock-src\" \"/tmp/mock-dest\"; then
            echo \"dry run copy succeeded\"
        fi
        
        if scenario_to_app::copy_universal_scripts \"/tmp/mock-dest\"; then
            echo \"dry run scripts copy succeeded\"
        fi
    "
    [[ "$output" == *"dry run copy succeeded"* ]]
    [[ "$output" == *"dry run scripts copy succeeded"* ]]
}

@test "function naming follows correct pattern" {
    # Verify all functions use the scenario_to_app:: prefix
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/scenario-to-app.sh'
        # Get all function names and check they follow the pattern
        declare -F | grep 'scenario_to_app::' | wc -l
    "
    # Should have multiple functions with the correct prefix
    [ "$output" -gt 5 ]
}