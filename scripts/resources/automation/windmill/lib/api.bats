#!/usr/bin/env bats

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "windmill"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    export SETUP_FILE_WINDMILL_DIR="$(dirname "${BATS_TEST_DIRNAME}")" 
    export SETUP_FILE_CONFIG_DIR="$(dirname "${BATS_TEST_DIRNAME}")/config"
    export SETUP_FILE_LIB_DIR="$(dirname "${BATS_TEST_DIRNAME}")/lib"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    WINDMILL_DIR="${SETUP_FILE_WINDMILL_DIR}"
    CONFIG_DIR="${SETUP_FILE_CONFIG_DIR}"
    LIB_DIR="${SETUP_FILE_LIB_DIR}"
    
    # Set test environment BEFORE sourcing config files to avoid readonly conflicts
    export WINDMILL_PORT="5681"
    export WINDMILL_BASE_URL="http://localhost:5681"
    export WINDMILL_API_TOKEN="wm_abc123def456"
    export WORKSPACE_NAME="demo"
    export SCRIPT_PATH="/tmp/test-script.py"
    export SCRIPT_CONTENT='print("Hello from Windmill!")'
    export JOB_ID="job_123456"
    export FLOW_ID="flow_789012"
    export YES="no"
    
    # Create test script file
    echo "$SCRIPT_CONTENT" > "$SCRIPT_PATH"
    
    # Mock resources functions that are called during config loading
    resources::get_default_port() {
        case "$1" in
            "windmill") echo "5681" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Now source the config files
    source "${WINDMILL_DIR}/config/defaults.sh"
    source "${WINDMILL_DIR}/config/messages.sh"
    
    # Export config and messages
    windmill::export_config
    windmill::export_messages
    
    # Load the functions to test
    source "${WINDMILL_DIR}/lib/api.sh"
    
    # Mock basic curl function
    curl() {
        case "$*" in
            *"health"*) echo '{"status":"healthy"}';;
            *) echo '{"success":true}';;
        esac
        return 0
    }
    export -f curl
    
    # Mock curl for API calls
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".id"*) echo "job_123456" ;;
            *".hash"*) echo "script_hash_123" ;;
            *".path"*) echo "u/admin/test_script" ;;
            *".result"*) echo "Hello from Windmill!" ;;
            *".success"*) echo "true" ;;
            *".type"*) echo "CompletedJob" ;;
            *".version"*) echo "1.0.0" ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock script interpreters
    python3() {
        echo "Hello from Windmill!"
        return 0
    }
    
    node() {
        echo "Hello from Node.js!"
        return 0
    }
    
    go() {
        case "$1" in
            "run") echo "Hello from Go!" ;;
            *) echo "GO: $*" ;;
        esac
        return 0
    }
    
    # Mock log functions
    
    # Mock Windmill utility functions
    windmill::is_running() { return 0; }
    windmill::is_healthy() { return 0; }
    
    # Load configuration and messages
    source "${WINDMILL_DIR}/config/defaults.sh"
    source "${WINDMILL_DIR}/config/messages.sh"
    windmill::export_config
    windmill::export_messages
    
    # Load the functions to test
    source "${WINDMILL_DIR}/lib/api.sh"
}

# Cleanup after each test
teardown() {
    trash::safe_remove "$SCRIPT_PATH" --test-cleanup
    vrooli_cleanup_test
}

# Test API connectivity
@test "windmill::test_api_connection tests API connectivity" {
    result=$(windmill::test_api_connection)
    
    [[ "$result" =~ "API" ]]
    [[ "$result" =~ "accessible" ]] || [[ "$result" =~ "version" ]]
    [[ "$result" =~ "1.0.0" ]]
}

# Test API connectivity failure
@test "windmill::test_api_connection handles API failure" {
    # Override curl to fail
    curl() {
        return 1
    }
    
    run windmill::test_api_connection
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "failed" ]]
}

# Test script creation
@test "windmill::create_script creates new script" {
    result=$(windmill::create_script "$SCRIPT_PATH" "python3" "Test script" "A test Python script")
    
    [[ "$result" =~ "script" ]]
    [[ "$result" =~ "created" ]]
    [[ "$result" =~ "script_hash_123" ]]
}

# Test script creation with missing file
@test "windmill::create_script handles missing script file" {
    run windmill::create_script "/nonexistent/script.py" "python3" "Test" "Description"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "not found" ]]
}

# Test script execution
@test "windmill::run_script executes script and returns result" {
    result=$(windmill::run_script "script_hash_123" '{"param1": "value1"}')
    
    [[ "$result" =~ "job" ]]
    [[ "$result" =~ "job_123456" ]]
    [[ "$result" =~ "Hello from Windmill!" ]]
}

# Test script execution with parameters
@test "windmill::run_script_with_params executes script with parameters" {
    result=$(windmill::run_script_with_params "script_hash_123" "param1=value1" "param2=value2")
    
    [[ "$result" =~ "job" ]]
    [[ "$result" =~ "parameters" ]] || [[ "$result" =~ "job_123456" ]]
}

# Test job status monitoring
@test "windmill::get_job_status retrieves job status" {
    result=$(windmill::get_job_status "$JOB_ID")
    
    [[ "$result" =~ "$JOB_ID" ]]
    [[ "$result" =~ "CompletedJob" ]]
    [[ "$result" =~ "success" ]]
}

# Test job result retrieval
@test "windmill::get_job_result retrieves job execution result" {
    result=$(windmill::get_job_result "$JOB_ID")
    
    [[ "$result" =~ "Hello from Windmill!" ]]
}

# Test job cancellation
@test "windmill::cancel_job cancels running job" {
    result=$(windmill::cancel_job "$JOB_ID")
    
    [[ "$result" =~ "cancel" ]] || [[ "$result" =~ "stopped" ]]
    [[ "$result" =~ "$JOB_ID" ]]
}

# Test script listing
@test "windmill::list_scripts shows available scripts" {
    result=$(windmill::list_scripts)
    
    [[ "$result" =~ "script" ]]
    [[ "$result" =~ "u/admin/test_script" ]]
    [[ "$result" =~ "python3" ]]
}

# Test script deletion
@test "windmill::delete_script removes script" {
    export YES="yes"
    
    result=$(windmill::delete_script "script_hash_123")
    
    [[ "$result" =~ "delete" ]] || [[ "$result" =~ "removed" ]]
    [[ "$result" =~ "script_hash_123" ]]
}

# Test script update
@test "windmill::update_script updates existing script" {
    result=$(windmill::update_script "script_hash_123" "$SCRIPT_PATH" "Updated script")
    
    [[ "$result" =~ "update" ]] || [[ "$result" =~ "modified" ]]
    [[ "$result" =~ "script_hash_123" ]]
}

# Test flow creation
@test "windmill::create_flow creates new flow" {
    local flow_definition='{"modules":[{"id":"a","value":{"type":"script","path":"u/admin/test_script"}}]}'
    
    result=$(windmill::create_flow "test_flow" "$flow_definition" "Test flow" "A test flow")
    
    [[ "$result" =~ "flow" ]]
    [[ "$result" =~ "created" ]]
    [[ "$result" =~ "u/admin/test_flow" ]]
}

# Test flow execution
@test "windmill::run_flow executes flow" {
    result=$(windmill::run_flow "u/admin/test_flow" '{"input": "test"}')
    
    [[ "$result" =~ "flow" ]] || [[ "$result" =~ "job" ]]
    [[ "$result" =~ "job_123456" ]]
}

# Test flow listing
@test "windmill::list_flows shows available flows" {
    result=$(windmill::list_flows)
    
    [[ "$result" =~ "flow" ]]
    [[ "$result" =~ "u/admin/test_flow" ]]
}

# Test app creation
@test "windmill::create_app creates new app" {
    local app_definition='{"grid":[{"id":"a","data":{"type":"text","text":"Hello World"}}]}'
    
    result=$(windmill::create_app "test_app" "$app_definition" "Test app")
    
    [[ "$result" =~ "app" ]]
    [[ "$result" =~ "created" ]]
    [[ "$result" =~ "u/admin/test_app" ]]
}

# Test app listing
@test "windmill::list_apps shows available apps" {
    result=$(windmill::list_apps)
    
    [[ "$result" =~ "app" ]]
    [[ "$result" =~ "u/admin/test_app" ]]
}

# Test variable creation
@test "windmill::create_variable creates workspace variable" {
    result=$(windmill::create_variable "test_var" "test_value" "Test variable" false)
    
    [[ "$result" =~ "variable" ]]
    [[ "$result" =~ "created" ]]
    [[ "$result" =~ "u/admin/test_var" ]]
}

# Test secret variable creation
@test "windmill::create_secret creates secret variable" {
    result=$(windmill::create_secret "secret_var" "secret_value" "Secret variable")
    
    [[ "$result" =~ "secret" ]] || [[ "$result" =~ "variable" ]]
    [[ "$result" =~ "created" ]]
}

# Test variable listing
@test "windmill::list_variables shows workspace variables" {
    result=$(windmill::list_variables)
    
    [[ "$result" =~ "variable" ]]
    [[ "$result" =~ "u/admin/test_var" ]]
}

# Test resource creation
@test "windmill::create_resource creates workspace resource" {
    local resource_config='{"host":"localhost","port":5432,"database":"test","user":"user","password":"pass"}'
    
    result=$(windmill::create_resource "test_db" "postgresql" "$resource_config" "Test database")
    
    [[ "$result" =~ "resource" ]]
    [[ "$result" =~ "created" ]]
    [[ "$result" =~ "u/admin/test_resource" ]]
}

# Test resource listing
@test "windmill::list_resources shows workspace resources" {
    result=$(windmill::list_resources)
    
    [[ "$result" =~ "resource" ]]
    [[ "$result" =~ "u/admin/test_resource" ]]
    [[ "$result" =~ "postgresql" ]]
}

# Test schedule creation
@test "windmill::create_schedule creates job schedule" {
    result=$(windmill::create_schedule "test_schedule" "script_hash_123" "0 */6 * * *" "UTC")
    
    [[ "$result" =~ "schedule" ]]
    [[ "$result" =~ "created" ]]
    [[ "$result" =~ "0 */6 * * *" ]]
}

# Test schedule listing
@test "windmill::list_schedules shows job schedules" {
    result=$(windmill::list_schedules)
    
    [[ "$result" =~ "schedule" ]]
    [[ "$result" =~ "u/admin/test_schedule" ]]
    [[ "$result" =~ "0 */6 * * *" ]]
}

# Test schedule enable/disable
@test "windmill::enable_schedule enables job schedule" {
    result=$(windmill::enable_schedule "u/admin/test_schedule")
    
    [[ "$result" =~ "enable" ]] || [[ "$result" =~ "activated" ]]
    [[ "$result" =~ "u/admin/test_schedule" ]]
}

@test "windmill::disable_schedule disables job schedule" {
    result=$(windmill::disable_schedule "u/admin/test_schedule")
    
    [[ "$result" =~ "disable" ]] || [[ "$result" =~ "deactivated" ]]
    [[ "$result" =~ "u/admin/test_schedule" ]]
}

# Test job history
@test "windmill::get_job_history retrieves job execution history" {
    result=$(windmill::get_job_history)
    
    [[ "$result" =~ "job" ]]
    [[ "$result" =~ "job_123456" ]]
    [[ "$result" =~ "CompletedJob" ]]
}

# Test job history filtering
@test "windmill::get_job_history_by_script filters job history by script" {
    result=$(windmill::get_job_history_by_script "script_hash_123")
    
    [[ "$result" =~ "job" ]]
    [[ "$result" =~ "script_hash_123" ]]
}

# Test workspace listing
@test "windmill::list_workspaces shows available workspaces" {
    result=$(windmill::list_workspaces)
    
    [[ "$result" =~ "workspace" ]]
    [[ "$result" =~ "Demo Workspace" ]]
    [[ "$result" =~ "Production" ]]
}

# Test user info
@test "windmill::get_user_info retrieves current user information" {
    result=$(windmill::get_user_info)
    
    [[ "$result" =~ "admin@test.com" ]]
    [[ "$result" =~ "admin" ]]
    [[ "$result" =~ "is_admin" ]]
}

# Test script validation
@test "windmill::validate_script validates script syntax" {
    result=$(windmill::validate_script "$SCRIPT_PATH" "python3")
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "syntax" ]]
}

# Test script validation with invalid syntax
@test "windmill::validate_script detects syntax errors" {
    echo "invalid python syntax (" > "/tmp/invalid.py"
    
    run windmill::validate_script "/tmp/invalid.py" "python3"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "syntax" ]] || [[ "$output" =~ "error" ]]
    
    trash::safe_remove "/tmp/invalid.py" --test-cleanup
}

# Test batch script execution
@test "windmill::run_batch_scripts executes multiple scripts" {
    local scripts=("script_hash_123" "script_hash_456")
    
    result=$(windmill::run_batch_scripts "${scripts[@]}")
    
    [[ "$result" =~ "batch" ]] || [[ "$result" =~ "multiple" ]]
    [[ "$result" =~ "script_hash_123" ]]
}

# Test script dependency analysis
@test "windmill::analyze_script_dependencies analyzes script dependencies" {
    result=$(windmill::analyze_script_dependencies "script_hash_123")
    
    [[ "$result" =~ "dependencies" ]] || [[ "$result" =~ "imports" ]]
}

# Test API rate limiting
@test "windmill::check_api_rate_limit checks API rate limits" {
    result=$(windmill::check_api_rate_limit)
    
    [[ "$result" =~ "rate" ]] || [[ "$result" =~ "limit" ]]
}

# Test API metrics
@test "windmill::get_api_metrics retrieves API usage metrics" {
    result=$(windmill::get_api_metrics)
    
    [[ "$result" =~ "metrics" ]] || [[ "$result" =~ "usage" ]]
}

# Test bulk operations
@test "windmill::bulk_import_scripts imports multiple scripts" {
    local scripts_dir="/tmp/scripts"
    mkdir -p "$scripts_dir"
    echo 'print("Script 1")' > "$scripts_dir/script1.py"
    echo 'print("Script 2")' > "$scripts_dir/script2.py"
    
    result=$(windmill::bulk_import_scripts "$scripts_dir")
    
    [[ "$result" =~ "bulk" ]] || [[ "$result" =~ "import" ]]
    [[ "$result" =~ "2" ]] || [[ "$result" =~ "scripts" ]]
    
    trash::safe_remove "$scripts_dir" --test-cleanup
}

# Test export operations
@test "windmill::export_workspace exports workspace data" {
    result=$(windmill::export_workspace "/tmp/workspace_export.json")
    
    [[ "$result" =~ "export" ]]
    [[ "$result" =~ "/tmp/workspace_export.json" ]]
}

# Test import operations
@test "windmill::import_workspace imports workspace data" {
    echo '{"scripts":[],"flows":[],"apps":[]}' > "/tmp/workspace_import.json"
    
    result=$(windmill::import_workspace "/tmp/workspace_import.json")
    
    [[ "$result" =~ "import" ]]
    [[ "$result" =~ "/tmp/workspace_import.json" ]]
    
    trash::safe_remove "/tmp/workspace_import.json" --test-cleanup
}

# Test API endpoint discovery
@test "windmill::discover_api_endpoints discovers available API endpoints" {
    result=$(windmill::discover_api_endpoints)
    
    [[ "$result" =~ "endpoint" ]] || [[ "$result" =~ "API" ]]
}

# Test webhook creation
@test "windmill::create_webhook creates webhook endpoint" {
    result=$(windmill::create_webhook "test_webhook" "script_hash_123")
    
    [[ "$result" =~ "webhook" ]]
    [[ "$result" =~ "created" ]]
    [[ "$result" =~ "test_webhook" ]]
}

# Test webhook testing
@test "windmill::test_webhook tests webhook functionality" {
    result=$(windmill::test_webhook "test_webhook" '{"test": "data"}')
    
    [[ "$result" =~ "webhook" ]]
    [[ "$result" =~ "test" ]]
}
