#!/usr/bin/env bash
################################################################################
# n8n Resource CLI
# 
# Lightweight CLI interface for n8n that delegates to existing lib functions.
#
# Usage:
#   resource-n8n <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    N8N_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    N8N_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
N8N_CLI_DIR="$(cd "$(dirname "$N8N_CLI_SCRIPT")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$N8N_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$N8N_CLI_DIR"
export N8N_SCRIPT_DIR="$N8N_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-${VROOLI_ROOT}/scripts/resources/common.sh}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

# Source n8n configuration
# shellcheck disable=SC1091
source "${N8N_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
n8n::export_config 2>/dev/null || true

# Source n8n libraries
for lib in core docker health status api inject; do
    lib_file="${N8N_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize with resource name
resource_cli::init "n8n"

################################################################################
# Delegate to existing n8n functions
################################################################################

# Inject workflow or credentials into n8n
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-n8n inject <file.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would inject: $file"
        return 0
    fi
    
    # Use existing injection function
    if command -v n8n::inject_file &>/dev/null; then
        n8n::inject_file "$file"
    elif command -v n8n::inject_workflow &>/dev/null; then
        n8n::inject_workflow "$file"
    else
        log::error "n8n injection functions not available"
        return 1
    fi
}

# Validate n8n configuration
resource_cli::validate() {
    if command -v n8n::validate &>/dev/null; then
        n8n::validate
    elif command -v n8n::check_health &>/dev/null; then
        n8n::check_health
    else
        # Basic validation
        log::header "Validating n8n"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "n8n" || {
            log::error "n8n container not running"
            return 1
        }
        log::success "n8n is running"
    fi
}

# Show n8n status
resource_cli::status() {
    if command -v n8n::status &>/dev/null; then
        n8n::status
    else
        # Basic status
        log::header "n8n Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "n8n"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=n8n" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start n8n
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start n8n"
        return 0
    fi
    
    if command -v n8n::start &>/dev/null; then
        n8n::start
    elif command -v n8n::docker::start &>/dev/null; then
        n8n::docker::start
    else
        docker start n8n || log::error "Failed to start n8n"
    fi
}

# Stop n8n
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop n8n"
        return 0
    fi
    
    if command -v n8n::stop &>/dev/null; then
        n8n::stop
    elif command -v n8n::docker::stop &>/dev/null; then
        n8n::docker::stop
    else
        docker stop n8n || log::error "Failed to stop n8n"
    fi
}

# Install n8n
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install n8n"
        return 0
    fi
    
    if command -v n8n::install &>/dev/null; then
        n8n::install
    else
        log::error "n8n::install not available"
        return 1
    fi
}

# Uninstall n8n
resource_cli::uninstall() {
    # Debug: Show what FORCE is set to
    # echo "DEBUG: FORCE='$FORCE'" >&2
    
    if [[ "${FORCE:-false}" != "true" ]]; then
        echo "⚠️  This will remove n8n and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall n8n"
        return 0
    fi
    
    if command -v n8n::uninstall &>/dev/null; then
        n8n::uninstall
    else
        docker stop n8n 2>/dev/null || true
        docker rm n8n 2>/dev/null || true
        log::success "n8n uninstalled"
    fi
}

################################################################################
# n8n-specific commands (if functions exist)
################################################################################

# List workflows
n8n_list_workflows() {
    if command -v n8n::list_workflows &>/dev/null; then
        n8n::list_workflows
    elif command -v n8n::get_workflows &>/dev/null; then
        n8n::get_workflows
    else
        log::error "Workflow listing not available"
        return 1
    fi
}

# Export workflows
n8n_export_workflows() {
    local output_dir="${1:-.}"
    
    if command -v n8n::api::export_workflows &>/dev/null; then
        n8n::api::export_workflows "$output_dir"
    elif command -v n8n::export_all &>/dev/null; then
        n8n::export_all "$output_dir"
    else
        log::error "Workflow export not available"
        return 1
    fi
}

# Activate a single workflow
n8n_activate_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo "Usage: resource-n8n activate-workflow <workflow_id>"
        echo ""
        echo "To find workflow IDs, run: resource-n8n list-workflows"
        return 1
    fi
    
    if command -v n8n::activate_workflow &>/dev/null; then
        n8n::activate_workflow "$workflow_id"
    else
        log::error "Workflow activation not available"
        return 1
    fi
}

# Deactivate a single workflow
n8n_deactivate_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo "Usage: resource-n8n deactivate-workflow <workflow_id>"
        echo ""
        echo "To find workflow IDs, run: resource-n8n list-workflows"
        return 1
    fi
    
    if command -v n8n::deactivate_workflow &>/dev/null; then
        n8n::deactivate_workflow "$workflow_id"
    else
        log::error "Workflow deactivation not available"
        return 1
    fi
}

# Activate all workflows
n8n_activate_all() {
    if command -v n8n::activate_all_workflows &>/dev/null; then
        n8n::activate_all_workflows
    else
        log::error "Bulk workflow activation not available"
        return 1
    fi
}

# Activate workflows by pattern
n8n_activate_pattern() {
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
    
    if command -v n8n::activate_workflows_by_pattern &>/dev/null; then
        n8n::activate_workflows_by_pattern "$pattern"
    else
        log::error "Pattern-based workflow activation not available"
        return 1
    fi
}

# Delete workflow
n8n_delete_workflow() {
    local workflow_id="${1:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID is required"
        echo "Usage: resource-n8n delete-workflow <workflow_id>"
        echo ""
        echo "Examples:"
        echo "  resource-n8n delete-workflow wbu5QO9dVCD521pe"
        return 1
    fi
    
    if command -v n8n::delete_workflow &>/dev/null; then
        n8n::delete_workflow "$workflow_id"
    else
        log::error "Workflow deletion not available"
        return 1
    fi
}

# Delete all workflows
n8n_delete_all_workflows() {
    if command -v n8n::delete_all_workflows &>/dev/null; then
        n8n::delete_all_workflows
    else
        log::error "Bulk workflow deletion not available"
        return 1
    fi
}

################################################################################
# Credential Management Commands
################################################################################

# Auto-discover and create credentials
n8n_auto_credentials() {
    if command -v n8n::auto_manage_credentials &>/dev/null; then
        n8n::auto_manage_credentials
    else
        log::error "Auto-credentials not available"
        return 1
    fi
}

# List current credentials
n8n_list_credentials() {
    if command -v n8n::validate_auto_credentials &>/dev/null; then
        n8n::validate_auto_credentials
    else
        log::error "Credential listing not available"
        return 1
    fi
}

# List discoverable resources (debug)
n8n_list_discoverable() {
    if command -v n8n::list_discoverable_resources &>/dev/null; then
        n8n::list_discoverable_resources
    else
        log::error "Resource discovery listing not available"
        return 1
    fi
}

# Show credential registry statistics
n8n_credential_registry() {
    local action="${1:-stats}"
    
    # Source the registry library
    # shellcheck disable=SC1091
    source "${N8N_SCRIPT_DIR}/lib/credential-registry.sh"
    
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

# Get credentials for n8n integration
resource_cli::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    # Parse arguments
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "n8n"; return 0; }
        return 1
    fi
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "${N8N_CONTAINER_NAME}")
    
    # Build connections array - n8n provides API access
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Get API key for n8n
        local api_key
        api_key=$(n8n::resolve_api_key 2>/dev/null || echo "")
        
        # n8n API connection
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "${N8N_PORT}" \
            --arg path "/api/v1" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local auth_obj
        if [[ -n "$api_key" ]]; then
            auth_obj=$(jq -n \
                --arg header_name "X-N8N-API-KEY" \
                --arg header_value "$api_key" \
                '{
                    header_name: $header_name,
                    header_value: $header_value
                }')
        else
            auth_obj='{"header_name": "X-N8N-API-KEY", "header_value": ""}'
        fi
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "n8n workflow automation API" \
            --arg web_url "http://localhost:${N8N_PORT}" \
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
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🚀 n8n Resource CLI

USAGE:
    resource-n8n <command> [options]

CORE COMMANDS:
    inject <file>       Inject workflow/credentials into n8n
    validate            Validate n8n configuration
    status              Show n8n status
    start               Start n8n container
    stop                Stop n8n container
    install             Install n8n
    uninstall           Uninstall n8n (requires --force)
    
N8N COMMANDS:
    list-workflows              List all workflows
    export-workflows            Export workflows to directory
    activate-workflow <id>      Activate a specific workflow by ID
    deactivate-workflow <id>    Deactivate a specific workflow by ID
    activate-all                Activate all workflows
    activate-pattern <pattern>  Activate workflows matching pattern
    delete-workflow <id>        Delete a specific workflow by ID
    delete-all-workflows        Delete all workflows (with confirmation)
    
CREDENTIAL COMMANDS:
    auto-credentials            Auto-discover and create resource credentials
    list-credentials            List existing n8n credentials  
    list-discoverable           List discoverable resources (debug)
    credential-registry [action] Show credential registry stats (actions: stats, list, backup, file)
    credentials                 Get connection credentials for n8n integration

OPTIONS:
    --verbose, -v       Show detailed output
    --dry-run           Preview actions without executing
    --force             Force operation (skip confirmations)

EXAMPLES:
    resource-n8n status
    resource-n8n inject shared:initialization/automation/n8n/workflow.json
    resource-n8n list-workflows
    resource-n8n activate-workflow 1irQ6tCYeRagjPh8
    resource-n8n activate-pattern "embedding*"
    resource-n8n activate-all
    resource-n8n export-workflows ./backups/
    resource-n8n delete-workflow wbu5QO9dVCD521pe
    resource-n8n delete-all-workflows
    resource-n8n auto-credentials
    resource-n8n list-credentials
    resource-n8n credentials --format pretty

PATTERN EXAMPLES:
    resource-n8n activate-pattern "embedding*"     # Workflows starting with "embedding"
    resource-n8n activate-pattern "*generator*"    # Workflows containing "generator"
    resource-n8n activate-pattern "reasoning-chain" # Exact substring match

For more information: https://docs.vrooli.com/resources/n8n
EOF
}

# Main command router
resource_cli::main() {
    # Initialize flags
    export VERBOSE=false
    export DRY_RUN=false
    export FORCE=false
    
    # Get command first (it's the first non-flag argument)
    local command=""
    local args=()
    
    # Collect all arguments, separating command from flags
    for arg in "$@"; do
        if [[ "$arg" == --* ]] || [[ "$arg" == -* ]]; then
            # It's a flag
            case "$arg" in
                --verbose|-v)
                    VERBOSE=true
                    ;;
                --dry-run)
                    DRY_RUN=true
                    ;;
                --force)
                    FORCE=true
                    ;;
                --help|-h)
                    resource_cli::show_help
                    exit 0
                    ;;
                *)
                    # Unknown flag, pass it through
                    args+=("$arg")
                    ;;
            esac
        else
            # First non-flag is the command
            if [[ -z "$command" ]]; then
                command="$arg"
            else
                # Additional non-flag arguments
                args+=("$arg")
            fi
        fi
    done
    
    # Default to help if no command
    command="${command:-help}"
    
    case "$command" in
        # Standard resource commands
        inject|validate|status|start|stop|install|uninstall|credentials)
            resource_cli::$command "${args[@]}"
            ;;
            
        # n8n-specific commands
        list-workflows)
            n8n_list_workflows "${args[@]}"
            ;;
        export-workflows)
            n8n_export_workflows "${args[@]}"
            ;;
        activate-workflow)
            n8n_activate_workflow "${args[@]}"
            ;;
        deactivate-workflow)
            n8n_deactivate_workflow "${args[@]}"
            ;;
        activate-all)
            n8n_activate_all "${args[@]}"
            ;;
        activate-pattern)
            n8n_activate_pattern "${args[@]}"
            ;;
        delete-workflow)
            n8n_delete_workflow "${args[@]}"
            ;;
        delete-all-workflows)
            n8n_delete_all_workflows "${args[@]}"
            ;;
            
        # Credential management commands
        auto-credentials)
            n8n_auto_credentials "${args[@]}"
            ;;
        list-credentials)
            n8n_list_credentials "${args[@]}"
            ;;
        list-discoverable)
            n8n_list_discoverable "${args[@]}"
            ;;
        credential-registry)
            n8n_credential_registry "${args[@]}"
            ;;
            
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resource_cli::main "$@"
fi