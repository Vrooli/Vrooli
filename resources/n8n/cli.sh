#!/usr/bin/env bash
################################################################################
# n8n Resource CLI
# 
# Ultra-thin CLI wrapper that delegates directly to library functions
#
# Usage:
#   resource-n8n <command> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    N8N_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "$(dirname "$N8N_CLI_SCRIPT")/../.." && builtin pwd)"
fi
N8N_CLI_DIR="${APP_ROOT}/resources/n8n"

# Source standard variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source n8n configuration
# shellcheck disable=SC1091
source "${N8N_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
n8n::export_config 2>/dev/null || true

# Source n8n libraries - these contain the actual functionality
# shellcheck disable=SC1091
source "${N8N_CLI_DIR}/lib/core.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_CLI_DIR}/lib/docker.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_CLI_DIR}/lib/health.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_CLI_DIR}/lib/status.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_CLI_DIR}/lib/api.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${N8N_CLI_DIR}/lib/inject.sh" 2>/dev/null || true

# Initialize CLI framework
cli::init "n8n" "n8n workflow automation platform"

# Override help to provide n8n-specific examples
cli::register_command "help" "Show this help message with examples" "n8n::show_help"

# Register core commands - direct library function calls
cli::register_command "install" "Install n8n" "n8n::install" "modifies-system"
cli::register_command "uninstall" "Uninstall n8n" "n8n::cli_uninstall" "modifies-system"
cli::register_command "start" "Start n8n" "n8n::start" "modifies-system"
cli::register_command "stop" "Stop n8n" "n8n::stop" "modifies-system"
cli::register_command "restart" "Restart n8n" "n8n::restart" "modifies-system"

# Register status and monitoring commands
cli::register_command "status" "Show service status" "n8n::status"
cli::register_command "validate" "Validate installation" "n8n::check_health"
cli::register_command "logs" "Show n8n logs" "n8n::logs"

# Register workflow management commands
cli::register_command "inject" "Inject workflow/credentials into n8n" "n8n::cli_inject" "modifies-system"
cli::register_command "list-workflows" "List all workflows" "n8n::list_workflows"
cli::register_command "activate-workflow" "Activate a specific workflow" "n8n::cli_activate_workflow" "modifies-system"
cli::register_command "deactivate-workflow" "Deactivate a specific workflow" "n8n::cli_deactivate_workflow" "modifies-system"
cli::register_command "activate-all" "Activate all workflows" "n8n::activate_all_workflows" "modifies-system"
cli::register_command "activate-pattern" "Activate workflows matching pattern" "n8n::cli_activate_pattern" "modifies-system"
cli::register_command "delete-workflow" "Delete a specific workflow" "n8n::cli_delete_workflow" "modifies-system"
cli::register_command "delete-all-workflows" "Delete all workflows" "n8n::delete_all_workflows" "modifies-system"

# Register credential management commands
cli::register_command "auto-credentials" "Auto-discover and create credentials" "n8n::auto_manage_credentials" "modifies-system"
cli::register_command "list-credentials" "List existing n8n credentials" "n8n::validate_auto_credentials"
cli::register_command "list-discoverable" "List discoverable resources" "n8n::list_discoverable_resources"
cli::register_command "credential-registry" "Show credential registry info" "n8n::cli_credential_registry"
cli::register_command "credentials" "Show n8n credentials for integration" "n8n::cli_credentials"

################################################################################
# CLI wrapper functions - minimal wrappers for commands that need argument handling
################################################################################

# Inject workflow or credentials into n8n
n8n::cli_inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-n8n inject <file.json>"
        echo ""
        echo "Examples:"
        echo "  resource-n8n inject workflow.json"
        echo "  resource-n8n inject shared:initialization/automation/n8n/workflow.json"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${var_VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Use n8n injection framework
    if command -v n8n::inject &>/dev/null; then
        n8n::inject "$file"
    elif command -v n8n::main &>/dev/null; then
        n8n::main "$file"
    else
        log::error "n8n injection functions not available"
        return 1
    fi
}

# Uninstall with force confirmation
n8n::cli_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove n8n and all its data. Use --force to confirm."
        return 1
    fi
    
    n8n::uninstall
}

# Show credentials for n8n integration
n8n::cli_credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    credentials::parse_args "$@" || return $?
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "${N8N_CONTAINER_NAME:-n8n}")
    
    # Build connections array
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Get API key for n8n
        local api_key
        api_key=$(n8n::resolve_api_key 2>/dev/null || echo "")
        
        # n8n API connection
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "${N8N_PORT:-5678}" \
            --arg path "/api/v1" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local auth_obj="{}"
        if [[ -n "$api_key" ]]; then
            auth_obj=$(jq -n \
                --arg header_name "X-N8N-API-KEY" \
                --arg header_value "$api_key" \
                '{
                    header_name: $header_name,
                    header_value: $header_value
                }')
        fi
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "n8n workflow automation API" \
            --arg web_url "http://localhost:${N8N_PORT:-5678}" \
            --arg api_version "v1" \
            '{
                description: $description,
                web_interface: $web_url,
                api_version: $api_version
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "api" \
            "n8n API" \
            "httpHeaderAuth" \
            "$connection_obj" \
            "$auth_obj" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "n8n" "$status" "$connections_array")
    
    credentials::format_output "$response"
}

# Activate a single workflow
n8n::cli_activate_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo "Usage: resource-n8n activate-workflow <workflow_id>"
        echo ""
        echo "To find workflow IDs, run: resource-n8n list-workflows"
        return 1
    fi
    
    n8n::activate_workflow "$workflow_id"
}

# Deactivate a single workflow
n8n::cli_deactivate_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo "Usage: resource-n8n deactivate-workflow <workflow_id>"
        echo ""
        echo "To find workflow IDs, run: resource-n8n list-workflows"
        return 1
    fi
    
    n8n::deactivate_workflow "$workflow_id"
}

# Activate workflows by pattern
n8n::cli_activate_pattern() {
    local pattern="${1:-}"
    
    if [[ -z "$pattern" ]]; then
        log::error "Pattern is required"
        echo "Usage: resource-n8n activate-pattern <pattern>"
        echo ""
        echo "Examples:"
        echo "  resource-n8n activate-pattern \"embedding*\""
        echo "  resource-n8n activate-pattern \"*generator*\""
        echo "  resource-n8n activate-pattern \"reasoning-chain\""
        return 1
    fi
    
    n8n::activate_workflows_by_pattern "$pattern"
}

# Delete workflow
n8n::cli_delete_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo "Usage: resource-n8n delete-workflow <workflow_id>"
        echo ""
        echo "Examples:"
        echo "  resource-n8n delete-workflow wbu5QO9dVCD521pe"
        return 1
    fi
    
    n8n::delete_workflow "$workflow_id"
}

# Show credential registry statistics
n8n::cli_credential_registry() {
    local action="${1:-stats}"
    
    # Source the registry library
    # shellcheck disable=SC1091
    source "${N8N_CLI_DIR}/lib/credential-registry.sh"
    
    case "$action" in
        stats)
            credential_registry::stats
            ;;
        list)
            echo "📋 Registered Credentials:"
            local credentials
            credentials=$(credential_registry::list)
            if [[ -n "$credentials" ]]; then
                echo "$credentials" | jq -r '"  • \(.name) (\(.type)) - \(.resource) - ID: \(.id)"'
            else
                echo "  No credentials registered"
            fi
            ;;
        backup)
            credential_registry::backup
            ;;
        file)
            echo "Registry file: $(credential_registry::get_file_path)"
            ;;
        *)
            log::error "Unknown registry action: $action"
            echo "Available actions: stats, list, backup, file"
            return 1
            ;;
    esac
}

# Custom help function with examples
n8n::show_help() {
    cli::_handle_help
    
    echo ""
    echo "⚡ Examples:"
    echo ""
    echo "  # Workflow management"
    echo "  resource-n8n inject workflow.json"
    echo "  resource-n8n inject shared:initialization/n8n/workflow.json"
    echo "  resource-n8n list-workflows"
    echo "  resource-n8n activate-workflow 1irQ6tCYeRagjPh8"
    echo "  resource-n8n activate-pattern \"embedding*\""
    echo ""
    echo "  # Credential management"
    echo "  resource-n8n auto-credentials"
    echo "  resource-n8n list-credentials"
    echo "  resource-n8n credential-registry stats"
    echo ""
    echo "  # Management"
    echo "  resource-n8n status"
    echo "  resource-n8n logs"
    echo "  resource-n8n credentials"
    echo ""
    echo "  # Dangerous operations"
    echo "  resource-n8n delete-workflow wbu5QO9dVCD521pe"
    echo "  resource-n8n uninstall --force"
    echo ""
    echo "Default Port: ${N8N_PORT:-5678}"
    echo "Web UI: http://localhost:${N8N_PORT:-5678}"
    echo "Features: 400+ integrations, visual workflows, API automation"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi