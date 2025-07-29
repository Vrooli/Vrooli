#!/usr/bin/env bats
# Tests for n8n api.sh functions

# Setup for each test
setup() {
    # Set test environment
    export N8N_CUSTOM_PORT="5678"
    export N8N_CONTAINER_NAME="n8n-test"
    export N8N_BASE_URL="http://localhost:5678"
    export N8N_BASIC_AUTH_USER="admin"
    export N8N_BASIC_AUTH_PASSWORD="password123"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    N8N_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock system functions
    system::is_command() {
        case "$1" in
            "curl") return 0 ;;
            "jq") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    # Mock curl for API calls
    curl() {
        local url=""
        local method="GET"
        local data=""
        local auth=""
        
        # Parse curl arguments
        while [[ $# -gt 0 ]]; do
            case $1 in
                -X) method="$2"; shift 2 ;;
                -d) data="$2"; shift 2 ;;
                -u) auth="$2"; shift 2 ;;
                --data-raw) data="$2"; shift 2 ;;
                -H) shift 2 ;;  # Ignore headers
                -s|-f) shift ;;  # Ignore flags
                http*) url="$1"; shift ;;
                *) shift ;;
            esac
        done
        
        case "$url" in
            *"/api/v1/workflows"*)
                if [[ "$method" == "GET" ]]; then
                    echo '{"data":[{"id":"1","name":"Test Workflow","active":true}]}'
                elif [[ "$method" == "POST" ]]; then
                    echo '{"data":{"id":"2","name":"New Workflow","active":false}}'
                fi
                return 0
                ;;
            *"/api/v1/executions"*)
                echo '{"data":[{"id":"1","workflowId":"1","status":"success","startedAt":"2024-01-01T12:00:00Z"}]}'
                return 0
                ;;
            *"/api/v1/credentials"*)
                echo '{"data":[{"id":"1","name":"Test Credential","type":"httpBasicAuth"}]}'
                return 0
                ;;
            *"/healthz"*)
                echo '{"status":"ok"}'
                return 0
                ;;
            *)
                return 1
                ;;
        esac
    }
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".data"*)
                echo '[{"id":"1","name":"Test"}]'
                ;;
            *".status"*)
                echo "ok"
                ;;
            *".id"*)
                echo "1"
                ;;
            *".name"*)
                echo "Test Workflow"
                ;;
            *) echo "{}" ;;
        esac
    }
    
    # Mock log functions
    log::info() {
        echo "INFO: $1"
        return 0
    }
    
    log::error() {
        echo "ERROR: $1"
        return 0
    }
    
    log::warn() {
        echo "WARN: $1"
        return 0
    }
    
    log::success() {
        echo "SUCCESS: $1"
        return 0
    }
    
    # Mock n8n functions
    n8n::is_running() { return 0; }
    n8n::get_api_url() { echo "$N8N_BASE_URL/api/v1"; }
    n8n::get_auth_header() { echo "admin:password123"; }
    
    # Load configuration and messages
    source "${N8N_DIR}/config/defaults.sh"
    source "${N8N_DIR}/config/messages.sh"
    n8n::export_config
    n8n::export_messages
    
    # Load the functions to test
    source "${N8N_DIR}/lib/api.sh"
}

# Test API health check
@test "n8n::api_health_check verifies API availability" {
    result=$(n8n::api_health_check)
    
    [[ "$result" =~ "API health:" ]]
    [[ "$result" =~ "ok" ]]
}

# Test API health check with service down
@test "n8n::api_health_check handles service unavailable" {
    # Override curl to fail
    curl() {
        return 1
    }
    
    run n8n::api_health_check
    [ "$status" -eq 1 ]
}

# Test workflow listing
@test "n8n::list_workflows returns workflow list" {
    result=$(n8n::list_workflows)
    
    [[ "$result" =~ "Workflows:" ]]
    [[ "$result" =~ "Test Workflow" ]]
    [[ "$result" =~ "active" ]]
}

# Test workflow listing with API error
@test "n8n::list_workflows handles API error" {
    # Override curl to fail for workflows endpoint
    curl() {
        case "$*" in
            *"/workflows"*)
                return 1
                ;;
            *) return 0 ;;
        esac
    }
    
    run n8n::list_workflows
    [ "$status" -eq 1 ]
}

# Test workflow creation
@test "n8n::create_workflow creates new workflow" {
    local workflow_name="Test Creation"
    local workflow_data='{"name":"Test Creation","nodes":[],"connections":{}}'
    
    result=$(n8n::create_workflow "$workflow_name" "$workflow_data")
    
    [[ "$result" =~ "INFO: Creating workflow" ]]
    [[ "$result" =~ "Test Creation" ]]
    [[ "$result" =~ "SUCCESS:" ]]
}

# Test workflow activation
@test "n8n::activate_workflow activates workflow" {
    local workflow_id="1"
    
    result=$(n8n::activate_workflow "$workflow_id")
    
    [[ "$result" =~ "INFO: Activating workflow" ]]
    [[ "$result" =~ "$workflow_id" ]]
}

# Test workflow deactivation
@test "n8n::deactivate_workflow deactivates workflow" {
    local workflow_id="1"
    
    result=$(n8n::deactivate_workflow "$workflow_id")
    
    [[ "$result" =~ "INFO: Deactivating workflow" ]]
    [[ "$result" =~ "$workflow_id" ]]
}

# Test workflow execution
@test "n8n::execute_workflow executes workflow manually" {
    local workflow_id="1"
    
    result=$(n8n::execute_workflow "$workflow_id")
    
    [[ "$result" =~ "INFO: Executing workflow" ]]
    [[ "$result" =~ "$workflow_id" ]]
}

# Test workflow deletion
@test "n8n::delete_workflow deletes workflow" {
    local workflow_id="1"
    
    result=$(n8n::delete_workflow "$workflow_id")
    
    [[ "$result" =~ "INFO: Deleting workflow" ]]
    [[ "$result" =~ "$workflow_id" ]]
}

# Test execution history
@test "n8n::list_executions returns execution history" {
    result=$(n8n::list_executions)
    
    [[ "$result" =~ "Executions:" ]]
    [[ "$result" =~ "success" ]]
    [[ "$result" =~ "2024-01-01" ]]
}

# Test execution details
@test "n8n::get_execution_details returns execution information" {
    local execution_id="1"
    
    result=$(n8n::get_execution_details "$execution_id")
    
    [[ "$result" =~ "Execution details:" ]]
    [[ "$result" =~ "$execution_id" ]]
}

# Test execution retry
@test "n8n::retry_execution retries failed execution" {
    local execution_id="1"
    
    result=$(n8n::retry_execution "$execution_id")
    
    [[ "$result" =~ "INFO: Retrying execution" ]]
    [[ "$result" =~ "$execution_id" ]]
}

# Test credential listing
@test "n8n::list_credentials returns credential list" {
    result=$(n8n::list_credentials)
    
    [[ "$result" =~ "Credentials:" ]]
    [[ "$result" =~ "Test Credential" ]]
    [[ "$result" =~ "httpBasicAuth" ]]
}

# Test credential creation
@test "n8n::create_credential creates new credential" {
    local cred_name="Test Credential"
    local cred_type="httpBasicAuth"
    local cred_data='{"user":"test","password":"secret"}'
    
    result=$(n8n::create_credential "$cred_name" "$cred_type" "$cred_data")
    
    [[ "$result" =~ "INFO: Creating credential" ]]
    [[ "$result" =~ "Test Credential" ]]
}

# Test credential update
@test "n8n::update_credential updates existing credential" {
    local cred_id="1"
    local cred_data='{"user":"updated","password":"newsecret"}'
    
    result=$(n8n::update_credential "$cred_id" "$cred_data")
    
    [[ "$result" =~ "INFO: Updating credential" ]]
    [[ "$result" =~ "$cred_id" ]]
}

# Test credential deletion
@test "n8n::delete_credential deletes credential" {
    local cred_id="1"
    
    result=$(n8n::delete_credential "$cred_id")
    
    [[ "$result" =~ "INFO: Deleting credential" ]]
    [[ "$result" =~ "$cred_id" ]]
}

# Test webhook trigger
@test "n8n::trigger_webhook triggers workflow via webhook" {
    local webhook_url="http://localhost:5678/webhook/test"
    local payload='{"test":"data"}'
    
    result=$(n8n::trigger_webhook "$webhook_url" "$payload")
    
    [[ "$result" =~ "INFO: Triggering webhook" ]]
    [[ "$result" =~ "test" ]]
}

# Test API authentication
@test "n8n::test_api_auth tests API authentication" {
    result=$(n8n::test_api_auth)
    
    [[ "$result" =~ "Testing API authentication" ]]
    [[ "$result" =~ "SUCCESS:" ]] || [[ "$result" =~ "authenticated" ]]
}

# Test API authentication failure
@test "n8n::test_api_auth handles authentication failure" {
    # Override auth to use invalid credentials
    n8n::get_auth_header() {
        echo "invalid:credentials"
    }
    
    # Mock curl to return 401
    curl() {
        return 22  # HTTP error
    }
    
    run n8n::test_api_auth
    [ "$status" -eq 1 ]
}

# Test workflow import
@test "n8n::import_workflow imports workflow from file" {
    local workflow_file="/tmp/test_workflow.json"
    echo '{"name":"Imported Workflow","nodes":[],"connections":{}}' > "$workflow_file"
    
    result=$(n8n::import_workflow "$workflow_file")
    
    [[ "$result" =~ "INFO: Importing workflow" ]]
    [[ "$result" =~ "Imported Workflow" ]]
    
    rm -f "$workflow_file"
}

# Test workflow export
@test "n8n::export_workflow exports workflow to file" {
    local workflow_id="1"
    local output_file="/tmp/exported_workflow.json"
    
    result=$(n8n::export_workflow "$workflow_id" "$output_file")
    
    [[ "$result" =~ "INFO: Exporting workflow" ]]
    [[ "$result" =~ "$workflow_id" ]]
    
    rm -f "$output_file"
}

# Test bulk operations
@test "n8n::bulk_activate_workflows activates multiple workflows" {
    local workflow_ids="1,2,3"
    
    result=$(n8n::bulk_activate_workflows "$workflow_ids")
    
    [[ "$result" =~ "INFO: Bulk activating workflows" ]]
    [[ "$result" =~ "1,2,3" ]]
}

# Test bulk operations
@test "n8n::bulk_deactivate_workflows deactivates multiple workflows" {
    local workflow_ids="1,2,3"
    
    result=$(n8n::bulk_deactivate_workflows "$workflow_ids")
    
    [[ "$result" =~ "INFO: Bulk deactivating workflows" ]]
    [[ "$result" =~ "1,2,3" ]]
}

# Test API endpoint availability
@test "n8n::check_api_endpoints verifies API endpoint availability" {
    result=$(n8n::check_api_endpoints)
    
    [[ "$result" =~ "API endpoints:" ]]
    [[ "$result" =~ "/workflows" ]]
    [[ "$result" =~ "/executions" ]]
    [[ "$result" =~ "/credentials" ]]
}

# Test API rate limiting
@test "n8n::check_api_rate_limit checks rate limiting" {
    result=$(n8n::check_api_rate_limit)
    
    [[ "$result" =~ "Rate limit check:" ]]
}

# Test API version
@test "n8n::get_api_version returns API version information" {
    result=$(n8n::get_api_version)
    
    [[ "$result" =~ "API version:" ]] || [[ "$result" =~ "v1" ]]
}

# Test statistics collection
@test "n8n::get_statistics collects n8n statistics" {
    result=$(n8n::get_statistics)
    
    [[ "$result" =~ "Statistics:" ]]
    [[ "$result" =~ "workflows" ]] || [[ "$result" =~ "executions" ]]
}

# Test backup operations
@test "n8n::backup_workflows backs up all workflows" {
    local backup_dir="/tmp/n8n_backup"
    mkdir -p "$backup_dir"
    
    result=$(n8n::backup_workflows "$backup_dir")
    
    [[ "$result" =~ "INFO: Backing up workflows" ]]
    [[ "$result" =~ "backup" ]]
    
    rm -rf "$backup_dir"
}

# Test restore operations
@test "n8n::restore_workflows restores workflows from backup" {
    local backup_dir="/tmp/n8n_backup"
    mkdir -p "$backup_dir"
    echo '{"workflows":[]}' > "$backup_dir/workflows.json"
    
    result=$(n8n::restore_workflows "$backup_dir")
    
    [[ "$result" =~ "INFO: Restoring workflows" ]]
    [[ "$result" =~ "backup" ]]
    
    rm -rf "$backup_dir"
}