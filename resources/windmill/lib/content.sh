#!/usr/bin/env bash
# Windmill Content Management Functions
# Business functionality for managing apps, workflows, and scripts in Windmill

# Source required dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

#######################################
# List available content (apps, workflows, scripts)
#######################################
windmill::content::list() {
    log::info "Listing available Windmill content..."
    echo ""
    echo "Available App Examples:"
    
    local apps_dir="${WINDMILL_CLI_DIR}/examples/apps"
    if [[ -d "$apps_dir" ]]; then
        for app_file in "${apps_dir}"/*.json; do
            if [[ -f "$app_file" ]]; then
                local app_name=$(basename "$app_file" .json)
                echo "  - $app_name"
            fi
        done
    fi
    
    echo ""
    echo "Available Workflow Examples:"
    local flows_dir="${WINDMILL_CLI_DIR}/examples/flows"
    if [[ -d "$flows_dir" ]]; then
        for flow_file in "${flows_dir}"/*.json; do
            if [[ -f "$flow_file" ]]; then
                local flow_name=$(basename "$flow_file" .json)
                echo "  - $flow_name"
            fi
        done
    fi
    
    echo ""
    echo "To add content: resource-windmill content add <file.json>"
}

#######################################
# Add content to Windmill (apps, workflows, scripts)
# Arguments:
#   $1 - Content path or URL
#######################################
windmill::content::add() {
    local content_path="${1:-}"
    
    if [[ -z "$content_path" ]]; then
        log::error "Content path required"
        echo "Usage: resource-windmill content add <file.json>"
        echo ""
        echo "Examples:"
        echo "  resource-windmill content add workflow.json"
        echo "  resource-windmill content add app.json"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$content_path" == shared:* ]]; then
        content_path="${var_VROOLI_ROOT}/${content_path#shared:}"
    fi
    
    if [[ ! -f "$content_path" ]]; then
        log::error "File not found: $content_path"
        return 1
    fi
    
    log::info "Adding content to Windmill: $content_path"
    
    # Use injection functionality from legacy code
    if command -v windmill::inject &>/dev/null; then
        windmill::inject "$content_path"
    else
        log::error "Content injection not available"
        return 1
    fi
}

#######################################
# Get content from Windmill
# Arguments:
#   $1 - Content identifier
#######################################
windmill::content::get() {
    local content_id="${1:-}"
    
    if [[ -z "$content_id" ]]; then
        log::error "Content identifier required"
        echo "Usage: resource-windmill content get <content-id>"
        return 1
    fi
    
    log::info "Retrieving content: $content_id"
    
    # Use API functionality to get content
    if command -v windmill::get_job_result &>/dev/null; then
        windmill::get_job_result "$content_id"
    else
        log::error "Content retrieval not available"
        return 1
    fi
}

#######################################
# Remove content from Windmill
# Arguments:
#   $1 - Content identifier
#######################################
windmill::content::remove() {
    local content_id="${1:-}"
    
    if [[ -z "$content_id" ]]; then
        log::error "Content identifier required"
        echo "Usage: resource-windmill content remove <content-id>"
        return 1
    fi
    
    log::info "Removing content: $content_id"
    log::warning "Content removal requires manual action via Windmill UI"
    echo "To remove content:"
    echo "1. Open Windmill UI at http://localhost:8000"
    echo "2. Navigate to the appropriate workspace"
    echo "3. Delete the script/app/flow: $content_id"
}

#######################################
# Execute content in Windmill (run scripts/workflows)
# Arguments:
#   $1 - Script path or workflow ID
#   $@ - Additional parameters
#######################################
windmill::content::execute() {
    local content_ref="${1:-}"
    shift
    
    if [[ -z "$content_ref" ]]; then
        log::error "Content reference required"
        echo "Usage: resource-windmill content execute <script-path|workflow-id> [params...]"
        echo ""
        echo "Examples:"
        echo "  resource-windmill content execute u/admin/hello_world"
        echo "  resource-windmill content execute workflow-123 --param value"
        return 1
    fi
    
    log::info "Executing content: $content_ref"
    
    # Use API functionality to run content
    if command -v windmill::run_test_script &>/dev/null; then
        windmill::run_test_script "$content_ref" "$@"
    else
        log::error "Content execution not available"
        return 1
    fi
}

#######################################
# Wrapper function for app preparation
#######################################
windmill::prepare_app() {
    local app_path="${1:-}"
    
    if [[ -z "$app_path" ]]; then
        log::error "App path required"
        echo "Usage: resource-windmill content prepare <app-directory>"
        return 1
    fi
    
    if command -v windmill::prepare_app &>/dev/null; then
        windmill::prepare_app "$app_path"
    else
        log::error "App preparation not available"
        return 1
    fi
}

#######################################
# Wrapper function for app deployment
#######################################
windmill::deploy_app() {
    local app_name="${1:-}"
    local workspace="${2:-demo}"
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: resource-windmill content deploy <app-name> [workspace]"
        return 1
    fi
    
    if command -v windmill::deploy_app &>/dev/null; then
        windmill::deploy_app "$app_name" "$workspace"
    else
        log::error "App deployment not available"
        return 1
    fi
}

#######################################
# Wrapper for credentials function
#######################################
windmill::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    credentials::parse_args "$@" || return $?
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "windmill-app")
    
    # Build connections array
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Windmill API connection
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "${WINDMILL_PORT:-8000}" \
            --arg path "/api" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "Windmill developer platform and workflow automation" \
            --arg web_ui "http://localhost:${WINDMILL_PORT:-8000}" \
            --arg default_user "admin@example.com" \
            '{
                description: $description,
                web_ui_url: $web_ui,
                default_credentials: {
                    username: $default_user,
                    password: "password"
                }
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "api" \
            "Windmill API" \
            "httpHeaderAuth" \
            "$connection_obj" \
            "{}" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "windmill" "$status" "$connections_array")
    
    credentials::format_output "$response"
}