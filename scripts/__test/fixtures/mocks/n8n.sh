#!/usr/bin/env bash
# N8N Workflow Automation Platform Mock Implementation
# 
# Provides comprehensive mock for n8n operations including:
# - CLI command interception (import, export, execute, update)
# - Docker container management simulation
# - REST API endpoint mocking
# - Webhook functionality simulation
# - Workflow and execution state management
# - Credential and authentication handling
#
# This mock follows the same standards as other updated mocks with:
# - Comprehensive state management
# - File-based persistence for BATS compatibility
# - Integration with centralized logging
# - Test helper functions

# Prevent duplicate loading
[[ -n "${N8N_MOCK_LOADED:-}" ]] && return 0
declare -g N8N_MOCK_LOADED=1

# Load dependencies
MOCK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$MOCK_DIR/logs.sh" ]] && source "$MOCK_DIR/logs.sh"
[[ -f "$MOCK_DIR/http.sh" ]] && source "$MOCK_DIR/http.sh"
[[ -f "$MOCK_DIR/docker.sh" ]] && source "$MOCK_DIR/docker.sh"

# Global configuration
declare -g N8N_MOCK_STATE_DIR="${N8N_MOCK_STATE_DIR:-/tmp/n8n-mock-state}"
declare -g N8N_MOCK_DEBUG="${N8N_MOCK_DEBUG:-}"

# Global state arrays
declare -gA N8N_MOCK_CONFIG=(              # N8N configuration
    [host]="localhost"
    [port]="5678"
    [base_url]="http://localhost:5678"
    [container_name]="n8n_test"
    [version]="1.25.0"
    [database_type]="sqlite"
    [server_status]="running"
    [api_key]="n8n_api_key_test123"
    [basic_auth_user]="admin"
    [basic_auth_password]="password"
    [encryption_key]="test_encryption_key"
    [webhook_url_base]="http://localhost:5678/webhook"
    [tunnel_enabled]="false"
    [connected]="true"
    [error_mode]=""
    [execution_mode]="regular"
)

declare -gA N8N_MOCK_WORKFLOWS=()          # Workflow storage: id -> workflow_json
declare -gA N8N_MOCK_EXECUTIONS=()         # Execution storage: execution_id -> execution_json
declare -gA N8N_MOCK_CREDENTIALS=()        # Credential storage: id -> credential_json
declare -gA N8N_MOCK_WORKFLOW_ACTIVE=()    # Workflow active status: id -> true/false
declare -gA N8N_MOCK_WEBHOOK_CALLS=()      # Webhook call tracking: webhook_id -> call_count
declare -gA N8N_MOCK_CLI_HISTORY=()        # CLI command history tracking

# Initialize state directory
mkdir -p "$N8N_MOCK_STATE_DIR"

# State persistence functions
mock::n8n::save_state() {
    # Ensure state directory exists
    mkdir -p "$N8N_MOCK_STATE_DIR"
    
    local state_file="$N8N_MOCK_STATE_DIR/n8n-state.sh"
    {
        echo "# N8N mock state - $(date)"
        
        # Save arrays using declare -p for proper restoration
        declare -p N8N_MOCK_CONFIG 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA N8N_MOCK_CONFIG=()"
        declare -p N8N_MOCK_WORKFLOWS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA N8N_MOCK_WORKFLOWS=()"
        declare -p N8N_MOCK_EXECUTIONS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA N8N_MOCK_EXECUTIONS=()"
        declare -p N8N_MOCK_CREDENTIALS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA N8N_MOCK_CREDENTIALS=()"
        declare -p N8N_MOCK_WORKFLOW_ACTIVE 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA N8N_MOCK_WORKFLOW_ACTIVE=()"
        declare -p N8N_MOCK_WEBHOOK_CALLS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA N8N_MOCK_WEBHOOK_CALLS=()"
        declare -p N8N_MOCK_CLI_HISTORY 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA N8N_MOCK_CLI_HISTORY=()"
    } > "$state_file"
    
    mock::log_state "n8n" "Saved n8n state to $state_file"
}

mock::n8n::load_state() {
    local state_file="$N8N_MOCK_STATE_DIR/n8n-state.sh"
    if [[ -f "$state_file" ]]; then
        source "$state_file"
        mock::log_state "n8n" "Loaded n8n state from $state_file"
    fi
}

# Automatically load state when sourced
mock::n8n::load_state

#######################################
# Helper functions for generating IDs and timestamps
#######################################
mock::n8n::generate_id() {
    local prefix="${1:-}"
    local timestamp=$(date +%s)
    local random=$(( RANDOM + RANDOM ))
    if [[ -n "$prefix" ]]; then
        echo "${prefix}_${timestamp}_${random}"
    else
        echo "${timestamp}_${random}"
    fi
}

mock::n8n::generate_execution_id() {
    echo "exec_$(mock::n8n::generate_id)"
}

mock::n8n::generate_workflow_id() {
    echo "wf_$(mock::n8n::generate_id)"
}

mock::n8n::generate_credential_id() {
    echo "cred_$(mock::n8n::generate_id)"
}

mock::n8n::current_iso_timestamp() {
    date -u +%Y-%m-%dT%H:%M:%S.000Z
}

#######################################
# Main n8n CLI interceptor
#######################################
n8n() {
    mock::log_and_verify "n8n" "$@"
    
    # Always reload state at the beginning to handle BATS subshells
    mock::n8n::load_state
    
    # Track CLI usage
    local cli_call="$(date +%s):$*"
    N8N_MOCK_CLI_HISTORY["${#N8N_MOCK_CLI_HISTORY[@]}"]="$cli_call"
    
    # Check if n8n is connected
    if [[ "${N8N_MOCK_CONFIG[connected]}" != "true" ]]; then
        echo "n8n: Connection failed - n8n server is not running" >&2
        mock::n8n::save_state
        return 1
    fi
    
    # Check for error injection
    if [[ -n "${N8N_MOCK_CONFIG[error_mode]}" ]]; then
        case "${N8N_MOCK_CONFIG[error_mode]}" in
            "connection_timeout")
                echo "n8n: Connection timeout - could not connect to n8n server" >&2
                mock::n8n::save_state
                return 1
                ;;
            "auth_failed")
                echo "n8n: Authentication failed - invalid credentials" >&2
                mock::n8n::save_state
                return 1
                ;;
            "database_error")
                echo "n8n: Database error - could not access database" >&2
                mock::n8n::save_state
                return 1
                ;;
        esac
    fi
    
    # Parse command and delegate
    local command="${1:-help}"
    local command_exit_code=0
    
    case "$command" in
        "--help"|"help"|"-h")
            mock::n8n::show_help
            command_exit_code=$?
            ;;
        "start")
            shift
            mock::n8n::cli_start "$@"
            command_exit_code=$?
            ;;
        "export:workflow")
            shift
            mock::n8n::cli_export_workflow "$@"
            command_exit_code=$?
            ;;
        "export:credentials")
            shift
            mock::n8n::cli_export_credentials "$@"
            command_exit_code=$?
            ;;
        "import:workflow")
            shift
            mock::n8n::cli_import_workflow "$@"
            command_exit_code=$?
            ;;
        "import:credentials")
            shift
            mock::n8n::cli_import_credentials "$@"
            command_exit_code=$?
            ;;
        "execute")
            shift
            mock::n8n::cli_execute "$@"
            command_exit_code=$?
            ;;
        "update:workflow")
            shift
            mock::n8n::cli_update_workflow "$@"
            command_exit_code=$?
            ;;
        "list:workflows")
            shift
            mock::n8n::cli_list_workflows "$@"
            command_exit_code=$?
            ;;
        "list:executions")
            shift
            mock::n8n::cli_list_executions "$@"
            command_exit_code=$?
            ;;
        "--version"|"version"|"-v")
            echo "n8n Workflow Automation Tool"
            echo "VERSION"
            echo "  n8n/${N8N_MOCK_CONFIG[version]} linux-x64 node-v18.17.0"
            command_exit_code=$?
            ;;
        *)
            echo "n8n: Unknown command '$command'" >&2
            echo "Use 'n8n --help' for available commands" >&2
            mock::n8n::save_state
            return 1
            ;;
    esac
    
    mock::n8n::save_state
    return $command_exit_code
}

#######################################
# CLI command implementations
#######################################
mock::n8n::show_help() {
    cat << 'EOF'
n8n Workflow Automation Tool

VERSION
  n8n/1.25.0 linux-x64 node-v18.17.0

USAGE
  $ n8n [COMMAND]

TOPICS
  db                Revert last database migration
  export            Export credentials and workflows
  import            Import credentials and workflows
  list              List workflows and executions
  update            Update workflows
  user-management   User management operations

COMMANDS
  execute           Execute workflow by ID
  executeBatch      Execute multiple workflows
  start             Start n8n server

Use n8n [COMMAND] --help for more information about a command.
EOF
}

mock::n8n::cli_start() {
    echo "Starting n8n process..."
    echo "n8n ready on ${N8N_MOCK_CONFIG[base_url]}"
    echo "Version: ${N8N_MOCK_CONFIG[version]}"
    echo "Database: ${N8N_MOCK_CONFIG[database_type]}"
    N8N_MOCK_CONFIG[server_status]="running"
}

mock::n8n::cli_export_workflow() {
    local workflow_id=""
    local output_file=""
    local backup=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --id=*)
                workflow_id="${1#--id=}"
                shift
                ;;
            --output=*)
                output_file="${1#--output=}"
                shift
                ;;
            --backup)
                backup=true
                shift
                ;;
            *)
                echo "n8n export:workflow: Unknown option $1" >&2
                mock::n8n::save_state
                return 1
                ;;
        esac
    done
    
    if [[ "$backup" == "true" ]]; then
        # Export all workflows
        local all_workflows="["
        local first=true
        for wf_id in "${!N8N_MOCK_WORKFLOWS[@]}"; do
            if [[ "$first" == "true" ]]; then
                first=false
            else
                all_workflows+=","
            fi
            all_workflows+="${N8N_MOCK_WORKFLOWS[$wf_id]}"
        done
        all_workflows+="]"
        
        if [[ -n "$output_file" ]]; then
            echo "$all_workflows" > "$output_file"
            echo "Exported all workflows to $output_file"
        else
            echo "$all_workflows"
        fi
    elif [[ -n "$workflow_id" ]]; then
        # Export specific workflow
        if [[ -z "${N8N_MOCK_WORKFLOWS[$workflow_id]:-}" ]]; then
            echo "n8n export:workflow: Workflow with ID '$workflow_id' not found" >&2
            mock::n8n::save_state
            return 1
        fi
        
        local workflow_data="${N8N_MOCK_WORKFLOWS[$workflow_id]}"
        if [[ -n "$output_file" ]]; then
            echo "$workflow_data" > "$output_file"
            echo "Exported workflow $workflow_id to $output_file"
        else
            echo "$workflow_data"
        fi
    else
        echo "n8n export:workflow: Either --id or --backup must be specified" >&2
        mock::n8n::save_state
        return 1
    fi
}

mock::n8n::cli_export_credentials() {
    local output_file=""
    local backup=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --output=*)
                output_file="${1#--output=}"
                shift
                ;;
            --backup)
                backup=true
                shift
                ;;
            *)
                echo "n8n export:credentials: Unknown option $1" >&2
                return 1
                ;;
        esac
    done
    
    # Export all credentials
    local all_credentials="["
    local first=true
    for cred_id in "${!N8N_MOCK_CREDENTIALS[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            all_credentials+=","
        fi
        all_credentials+="${N8N_MOCK_CREDENTIALS[$cred_id]}"
    done
    all_credentials+="]"
    
    if [[ -n "$output_file" ]]; then
        echo "$all_credentials" > "$output_file"
        echo "Exported credentials to $output_file"
    else
        echo "$all_credentials"
    fi
}

mock::n8n::cli_import_workflow() {
    local input_file=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --input=*)
                input_file="${1#--input=}"
                shift
                ;;
            *)
                echo "n8n import:workflow: Unknown option $1" >&2
                mock::n8n::save_state
                return 1
                ;;
        esac
    done
    
    if [[ -z "$input_file" ]]; then
        echo "n8n import:workflow: --input option is required" >&2
        mock::n8n::save_state
        return 1
    fi
    
    if [[ ! -f "$input_file" ]]; then
        echo "n8n import:workflow: File '$input_file' not found" >&2
        mock::n8n::save_state
        return 1
    fi
    
    # Read and parse workflow file
    local workflow_data
    workflow_data=$(cat "$input_file")
    
    # Generate new workflow ID
    local new_workflow_id
    new_workflow_id=$(mock::n8n::generate_workflow_id)
    
    # Store workflow
    N8N_MOCK_WORKFLOWS["$new_workflow_id"]="$workflow_data"
    N8N_MOCK_WORKFLOW_ACTIVE["$new_workflow_id"]="false"  # Deactivated during import
    
    echo "Imported workflow as ID: $new_workflow_id"
    echo "Workflow deactivated during import (use update:workflow --active=true to activate)"
}

mock::n8n::cli_import_credentials() {
    local input_file=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --input=*)
                input_file="${1#--input=}"
                shift
                ;;
            *)
                echo "n8n import:credentials: Unknown option $1" >&2
                return 1
                ;;
        esac
    done
    
    if [[ -z "$input_file" ]]; then
        echo "n8n import:credentials: --input option is required" >&2
        return 1
    fi
    
    if [[ ! -f "$input_file" ]]; then
        echo "n8n import:credentials: File '$input_file' not found" >&2
        return 1
    fi
    
    # Read credentials file
    local credentials_data
    credentials_data=$(cat "$input_file")
    
    # Generate new credential ID
    local new_credential_id
    new_credential_id=$(mock::n8n::generate_credential_id)
    
    # Store credential
    N8N_MOCK_CREDENTIALS["$new_credential_id"]="$credentials_data"
    
    echo "Imported credentials as ID: $new_credential_id"
}

mock::n8n::cli_execute() {
    local workflow_id=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --id=*)
                workflow_id="${1#--id=}"
                shift
                ;;
            *)
                workflow_id="$1"
                shift
                ;;
        esac
    done
    
    if [[ -z "$workflow_id" ]]; then
        echo "n8n execute: Workflow ID is required" >&2
        mock::n8n::save_state
        return 1
    fi
    
    if [[ -z "${N8N_MOCK_WORKFLOWS[$workflow_id]:-}" ]]; then
        echo "n8n execute: Workflow with ID '$workflow_id' not found" >&2
        mock::n8n::save_state
        return 1
    fi
    
    # Generate execution
    local execution_id
    execution_id=$(mock::n8n::generate_execution_id)
    
    local execution_data="{
        \"id\": \"$execution_id\",
        \"workflowId\": \"$workflow_id\",
        \"status\": \"success\",
        \"mode\": \"manual\",
        \"startedAt\": \"$(mock::n8n::current_iso_timestamp)\",
        \"finishedAt\": \"$(mock::n8n::current_iso_timestamp)\",
        \"executionTime\": 1.5
    }"
    
    N8N_MOCK_EXECUTIONS["$execution_id"]="$execution_data"
    
    echo "Execution started"
    echo "Execution ID: $execution_id"
    echo "Status: success"
}

mock::n8n::cli_update_workflow() {
    local workflow_id=""
    local active=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --id=*)
                workflow_id="${1#--id=}"
                shift
                ;;
            --active=*)
                active="${1#--active=}"
                shift
                ;;
            *)
                echo "n8n update:workflow: Unknown option $1" >&2
                mock::n8n::save_state
                return 1
                ;;
        esac
    done
    
    if [[ -z "$workflow_id" ]]; then
        echo "n8n update:workflow: --id option is required" >&2
        mock::n8n::save_state
        return 1
    fi
    
    if [[ -z "${N8N_MOCK_WORKFLOWS[$workflow_id]:-}" ]]; then
        echo "n8n update:workflow: Workflow with ID '$workflow_id' not found" >&2
        mock::n8n::save_state
        return 1
    fi
    
    if [[ -n "$active" ]]; then
        N8N_MOCK_WORKFLOW_ACTIVE["$workflow_id"]="$active"
        echo "Updated workflow $workflow_id active status to: $active"
    else
        echo "n8n update:workflow: No update parameters provided" >&2
        mock::n8n::save_state
        return 1
    fi
}

mock::n8n::cli_list_workflows() {
    # Check if we have any workflows after loading state
    mock::n8n::load_state
    
    if [[ ${#N8N_MOCK_WORKFLOWS[@]} -eq 0 ]]; then
        echo "No workflows found"
        return 0
    fi
    
    printf "%-15s %-30s %-10s\n" "ID" "Name" "Active"
    printf "%-15s %-30s %-10s\n" "---" "----" "------"
    
    for workflow_id in "${!N8N_MOCK_WORKFLOWS[@]}"; do
        local active_status="${N8N_MOCK_WORKFLOW_ACTIVE[$workflow_id]:-false}"
        printf "%-15s %-30s %-10s\n" "$workflow_id" "Mock Workflow" "$active_status"
    done
}

mock::n8n::cli_list_executions() {
    if [[ ${#N8N_MOCK_EXECUTIONS[@]} -eq 0 ]]; then
        echo "No executions found"
        return 0
    fi
    
    printf "%-20s %-15s %-10s %-20s\n" "Execution ID" "Workflow ID" "Status" "Started At"
    printf "%-20s %-15s %-10s %-20s\n" "------------" "-----------" "------" "----------"
    
    for execution_id in "${!N8N_MOCK_EXECUTIONS[@]}"; do
        printf "%-20s %-15s %-10s %-20s\n" "$execution_id" "wf_123" "success" "$(date -u +%Y-%m-%d:%H:%M:%S)"
    done
}

#######################################
# HTTP API endpoint setup and management
#######################################
mock::n8n::setup_api_endpoints() {
    local state="${1:-healthy}"
    local base_url="${N8N_MOCK_CONFIG[base_url]}"
    
    case "$state" in
        "healthy")
            mock::n8n::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::n8n::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::n8n::setup_installing_endpoints
            ;;
        "stopped")
            mock::n8n::setup_stopped_endpoints
            ;;
    esac
}

mock::n8n::setup_healthy_endpoints() {
    local base_url="${N8N_MOCK_CONFIG[base_url]}"
    
    # Health endpoint
    mock::http::set_endpoint_response "$base_url/healthz" \
        "{\"status\":\"ok\",\"version\":\"${N8N_MOCK_CONFIG[version]}\"}"
    
    # Workflows endpoints
    mock::http::set_endpoint_response "$base_url/api/v1/workflows" \
        '{
            "data": [
                {
                    "id": "1",
                    "name": "Welcome Workflow",
                    "active": true,
                    "createdAt": "2024-01-15T10:00:00.000Z",
                    "updatedAt": "2024-01-15T10:00:00.000Z",
                    "nodes": [
                        {
                            "id": "node1",
                            "name": "Start",
                            "type": "n8n-nodes-base.start",
                            "position": [100, 100]
                        }
                    ]
                }
            ],
            "nextCursor": null
        }'
    
    # Single workflow endpoint
    mock::http::set_endpoint_response "$base_url/api/v1/workflows/1" \
        '{
            "data": {
                "id": "1",
                "name": "Welcome Workflow",
                "active": true,
                "nodes": [],
                "connections": {}
            }
        }'
    
    # Executions endpoint
    mock::http::set_endpoint_response "$base_url/api/v1/executions" \
        '{
            "data": [
                {
                    "id": "100",
                    "finished": true,
                    "mode": "manual",
                    "startedAt": "2024-01-15T12:00:00.000Z",
                    "stoppedAt": "2024-01-15T12:00:05.000Z",
                    "workflowId": "1",
                    "status": "success"
                }
            ]
        }'
    
    # Execute workflow endpoint
    mock::http::set_endpoint_response "$base_url/api/v1/workflows/1/execute" \
        '{
            "data": {
                "executionId": "101",
                "status": "running"
            }
        }' \
        "POST"
    
    # Credentials endpoint
    mock::http::set_endpoint_response "$base_url/api/v1/credentials" \
        '{
            "data": [
                {
                    "id": "1",
                    "name": "Test Credentials",
                    "type": "httpHeaderAuth",
                    "createdAt": "2024-01-15T10:00:00.000Z"
                }
            ]
        }'
    
    # Webhook endpoints
    mock::http::set_endpoint_response "$base_url/webhook/test" \
        '{
            "status": "success",
            "message": "Webhook received",
            "timestamp": "'$(mock::n8n::current_iso_timestamp)'"
        }' \
        "POST"
}

mock::n8n::setup_unhealthy_endpoints() {
    local base_url="${N8N_MOCK_CONFIG[base_url]}"
    
    # Health endpoint returns error
    mock::http::set_endpoint_response "$base_url/healthz" \
        '{"status":"unhealthy","error":"Database connection failed"}' \
        "GET" \
        "503"
    
    # API endpoints return errors
    mock::http::set_endpoint_response "$base_url/api/v1/workflows" \
        '{"code":503,"message":"Service temporarily unavailable"}' \
        "GET" \
        "503"
}

mock::n8n::setup_installing_endpoints() {
    local base_url="${N8N_MOCK_CONFIG[base_url]}"
    
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$base_url/healthz" \
        '{"status":"installing","progress":80,"current_step":"Initializing database"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$base_url/api/v1/workflows" \
        '{"code":503,"message":"n8n is still initializing"}' \
        "GET" \
        "503"
}

mock::n8n::setup_stopped_endpoints() {
    local base_url="${N8N_MOCK_CONFIG[base_url]}"
    
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$base_url"
}

#######################################
# State management and configuration
#######################################
mock::n8n::reset() {
    mock::log_state "n8n" "Resetting n8n mock to clean state"
    
    # Reset all arrays
    N8N_MOCK_WORKFLOWS=()
    N8N_MOCK_EXECUTIONS=()
    N8N_MOCK_CREDENTIALS=()
    N8N_MOCK_WORKFLOW_ACTIVE=()
    N8N_MOCK_WEBHOOK_CALLS=()
    N8N_MOCK_CLI_HISTORY=()
    
    # Reset configuration to defaults
    N8N_MOCK_CONFIG[server_status]="running"
    N8N_MOCK_CONFIG[connected]="true"
    N8N_MOCK_CONFIG[error_mode]=""
    
    # Create some default test data
    mock::n8n::create_default_data
    
    # Clean state file
    rm -f "$N8N_MOCK_STATE_DIR/n8n-state.sh" 2>/dev/null || true
    
    mock::n8n::save_state
}

mock::n8n::create_default_data() {
    # Create a default workflow
    local default_workflow_id="wf_default_123"
    local default_workflow='{
        "id": "'$default_workflow_id'",
        "name": "Default Test Workflow",
        "active": false,
        "nodes": [
            {
                "id": "start",
                "name": "Start",
                "type": "n8n-nodes-base.start",
                "position": [100, 100]
            },
            {
                "id": "set",
                "name": "Set Data",
                "type": "n8n-nodes-base.set",
                "position": [300, 100]
            }
        ],
        "connections": {
            "Start": {
                "main": [[{"node": "Set Data", "type": "main", "index": 0}]]
            }
        }
    }'
    
    N8N_MOCK_WORKFLOWS["$default_workflow_id"]="$default_workflow"
    N8N_MOCK_WORKFLOW_ACTIVE["$default_workflow_id"]="false"
    
    # Create a default credential
    local default_cred_id="cred_default_456"
    local default_credential='{
        "id": "'$default_cred_id'",
        "name": "Default Test Credentials",
        "type": "httpHeaderAuth",
        "data": {"headerAuth": {"name": "Authorization", "value": "Bearer test123"}},
        "createdAt": "'$(mock::n8n::current_iso_timestamp)'"
    }'
    
    N8N_MOCK_CREDENTIALS["$default_cred_id"]="$default_credential"
}

mock::n8n::set_error_mode() {
    local error_mode="$1"
    N8N_MOCK_CONFIG[error_mode]="$error_mode"
    mock::n8n::save_state
    mock::log_state "n8n" "Set error mode to: $error_mode"
}

mock::n8n::clear_error_mode() {
    N8N_MOCK_CONFIG[error_mode]=""
    mock::n8n::save_state
    mock::log_state "n8n" "Cleared error mode"
}

mock::n8n::set_connection_status() {
    local status="$1"
    N8N_MOCK_CONFIG[connected]="$status"
    mock::n8n::save_state
    mock::log_state "n8n" "Set connection status to: $status"
}

mock::n8n::get_workflow_count() {
    echo "${#N8N_MOCK_WORKFLOWS[@]}"
}

mock::n8n::get_execution_count() {
    echo "${#N8N_MOCK_EXECUTIONS[@]}"
}

mock::n8n::get_cli_call_count() {
    echo "${#N8N_MOCK_CLI_HISTORY[@]}"
}

#######################################
# Test helper functions
#######################################
mock::n8n::create_test_workflow() {
    local workflow_name="${1:-Test Workflow}"
    local workflow_type="${2:-simple}"
    local active_status="${3:-false}"
    
    # Load current state to ensure we have latest data
    mock::n8n::load_state
    
    local workflow_id
    workflow_id=$(mock::n8n::generate_workflow_id)
    
    local workflow_data
    case "$workflow_type" in
        "webhook")
            workflow_data='{
                "id": "'$workflow_id'",
                "name": "'$workflow_name'",
                "active": '$active_status',
                "nodes": [
                    {"id": "webhook", "type": "n8n-nodes-base.webhook", "webhookId": "test_webhook"},
                    {"id": "respond", "type": "n8n-nodes-base.respondToWebhook"}
                ],
                "connections": {
                    "webhook": {"main": [[{"node": "respond", "type": "main", "index": 0}]]}
                }
            }'
            ;;
        *)
            workflow_data='{
                "id": "'$workflow_id'",
                "name": "'$workflow_name'",
                "active": '$active_status',
                "nodes": [
                    {"id": "start", "type": "n8n-nodes-base.start"},
                    {"id": "set", "type": "n8n-nodes-base.set"}
                ],
                "connections": {
                    "start": {"main": [[{"node": "set", "type": "main", "index": 0}]]}
                }
            }'
            ;;
    esac
    
    N8N_MOCK_WORKFLOWS["$workflow_id"]="$workflow_data"
    N8N_MOCK_WORKFLOW_ACTIVE["$workflow_id"]="$active_status"
    
    # Always save state after modification
    mock::n8n::save_state
    
    # Also write the ID to a temporary file for retrieval
    echo "$workflow_id" > "$N8N_MOCK_STATE_DIR/last_created_workflow_id"
    
    echo "$workflow_id"
}

mock::n8n::simulate_workflow_execution() {
    local workflow_id="$1"
    local execution_status="${2:-success}"
    local execution_time="${3:-2.5}"
    
    # Load current state
    mock::n8n::load_state
    
    if [[ -z "${N8N_MOCK_WORKFLOWS[$workflow_id]:-}" ]]; then
        echo "Workflow $workflow_id not found" >&2
        return 1
    fi
    
    local execution_id
    execution_id=$(mock::n8n::generate_execution_id)
    
    local execution_data="{
        \"id\": \"$execution_id\",
        \"workflowId\": \"$workflow_id\",
        \"status\": \"$execution_status\",
        \"mode\": \"manual\",
        \"startedAt\": \"$(mock::n8n::current_iso_timestamp)\",
        \"finishedAt\": \"$(mock::n8n::current_iso_timestamp)\",
        \"executionTime\": $execution_time,
        \"data\": {
            \"resultData\": {
                \"runData\": {
                    \"Set\": [{\"data\": {\"main\": [[{\"json\": {\"result\": \"$execution_status\"}}]]}}]
                }
            }
        }
    }"
    
    N8N_MOCK_EXECUTIONS["$execution_id"]="$execution_data"
    mock::n8n::save_state
    
    # Write execution ID to file for retrieval
    echo "$execution_id" > "$N8N_MOCK_STATE_DIR/last_created_execution_id"
    
    echo "$execution_id"
}

mock::n8n::trigger_webhook() {
    local webhook_id="$1"
    local payload="${2:-{\"test\": \"data\"}}"
    
    # Load current state
    mock::n8n::load_state
    
    # Increment webhook call count
    local current_count="${N8N_MOCK_WEBHOOK_CALLS[$webhook_id]:-0}"
    N8N_MOCK_WEBHOOK_CALLS["$webhook_id"]=$((current_count + 1))
    
    mock::n8n::save_state
    
    echo "{
        \"status\": \"success\",
        \"message\": \"Webhook triggered\",
        \"webhookId\": \"$webhook_id\",
        \"payload\": $payload,
        \"callCount\": $((current_count + 1))
    }"
}

#######################################
# Docker integration functions
#######################################
mock::n8n::setup_docker_state() {
    local container_state="${1:-running}"
    local container_name="${N8N_MOCK_CONFIG[container_name]}"
    
    # Set up Docker container state using the docker mock
    if command -v mock::docker::set_container_state >/dev/null 2>&1; then
        mock::docker::set_container_state "$container_name" "$container_state"
    fi
    
    # Update n8n configuration based on Docker state
    case "$container_state" in
        "running")
            N8N_MOCK_CONFIG[server_status]="running"
            N8N_MOCK_CONFIG[connected]="true"
            ;;
        "stopped"|"exited")
            N8N_MOCK_CONFIG[server_status]="stopped"
            N8N_MOCK_CONFIG[connected]="false"
            ;;
        "starting")
            N8N_MOCK_CONFIG[server_status]="starting"
            N8N_MOCK_CONFIG[connected]="false"
            ;;
    esac
    
    mock::n8n::save_state
}

#######################################
# Export functions for external use
#######################################
export -f n8n
export -f mock::n8n::reset
export -f mock::n8n::setup_api_endpoints
export -f mock::n8n::set_error_mode
export -f mock::n8n::clear_error_mode
export -f mock::n8n::set_connection_status
export -f mock::n8n::get_workflow_count
export -f mock::n8n::get_execution_count
export -f mock::n8n::get_cli_call_count
export -f mock::n8n::create_test_workflow
export -f mock::n8n::simulate_workflow_execution
export -f mock::n8n::trigger_webhook
export -f mock::n8n::setup_docker_state
export -f mock::n8n::save_state
export -f mock::n8n::load_state

# Initialize with default state
mock::n8n::reset

mock::log_state "n8n" "N8N comprehensive mock implementation loaded"