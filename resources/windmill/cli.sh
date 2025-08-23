#!/usr/bin/env bash
################################################################################
# Windmill Resource CLI
# 
# Ultra-thin CLI wrapper that delegates directly to library functions
#
# Usage:
#   resource-windmill <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    WINDMILL_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    WINDMILL_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
WINDMILL_CLI_DIR="$(cd "$(dirname "$WINDMILL_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source windmill configuration
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
windmill::export_config 2>/dev/null || true

# Source windmill libraries - these contain the actual functionality
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/lib/state.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/lib/common.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/lib/database.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/lib/docker.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/lib/status.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/lib/install.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/lib/apps.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/lib/api.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/lib/workers.sh" 2>/dev/null || true

# Initialize CLI framework
cli::init "windmill" "Windmill developer platform and workflow automation"

# Override help to provide Windmill-specific examples
cli::register_command "help" "Show this help message with examples" "windmill::show_help"

# Register core commands - direct library function calls
cli::register_command "install" "Install Windmill" "windmill::install" "modifies-system"
cli::register_command "uninstall" "Uninstall Windmill" "windmill::cli_uninstall" "modifies-system"
cli::register_command "start" "Start Windmill" "windmill::start" "modifies-system"
cli::register_command "stop" "Stop Windmill" "windmill::stop" "modifies-system"

# Register status and monitoring commands
cli::register_command "status" "Show service status" "windmill::status"
cli::register_command "validate" "Validate installation" "windmill::validate"
cli::register_command "logs" "Show Windmill logs" "windmill::cli_logs"

# Register workflow and app commands
cli::register_command "inject" "Inject workflow/app into Windmill" "windmill::cli_inject" "modifies-system"
cli::register_command "list-apps" "List available app examples" "apps::list"

# Register utility commands
cli::register_command "credentials" "Show Windmill credentials for integration" "windmill::cli_credentials"
cli::register_command "prepare-app" "Prepare app for deployment" "windmill::cli_prepare_app" "modifies-system"
cli::register_command "deploy-app" "Deploy app to workspace" "windmill::cli_deploy_app" "modifies-system"
cli::register_command "scale-workers" "Scale worker containers" "windmill::cli_scale_workers" "modifies-system"
cli::register_command "backup" "Backup Windmill data" "windmill::cli_backup" "modifies-system"
cli::register_command "restore" "Restore from backup" "windmill::cli_restore" "modifies-system"
cli::register_command "api-setup" "Show API setup instructions" "api::setup"

################################################################################
# CLI wrapper functions - minimal wrappers for commands that need argument handling
################################################################################

# Inject workflows or apps into windmill
windmill::cli_inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-windmill inject <file.json>"
        echo ""
        echo "Examples:"
        echo "  resource-windmill inject workflow.json"
        echo "  resource-windmill inject shared:initialization/automation/windmill/workflow.json"
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
    
    # Use existing injection function
    if command -v windmill::inject &>/dev/null; then
        windmill::inject "$file"
    else
        "${WINDMILL_CLI_DIR}/inject.sh" --inject "$(cat "$file")"
    fi
}

# Uninstall with force confirmation
windmill::cli_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Windmill and all its data. Use --force to confirm."
        return 1
    fi
    
    windmill::uninstall
}

# Show logs with line count
windmill::cli_logs() {
    local lines="${1:-50}"
    
    if command -v windmill::logs &>/dev/null; then
        windmill::logs "$lines"
    else
        docker logs --tail "$lines" windmill-app 2>/dev/null || log::error "Failed to show logs"
    fi
}

# Show credentials for Windmill integration
windmill::cli_credentials() {
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

# Prepare app for deployment
windmill::cli_prepare_app() {
    local app_path="${1:-}"
    
    if [[ -z "$app_path" ]]; then
        log::error "App path required"
        echo "Usage: resource-windmill prepare-app <app-directory>"
        return 1
    fi
    
    if command -v apps::prepare &>/dev/null; then
        apps::prepare "$app_path"
    else
        log::error "App preparation not available"
        return 1
    fi
}

# Deploy app to workspace
windmill::cli_deploy_app() {
    local app_name="${1:-}"
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: resource-windmill deploy-app <app-name>"
        return 1
    fi
    
    if command -v apps::deploy &>/dev/null; then
        apps::deploy "$app_name"
    else
        log::error "App deployment not available"
        return 1
    fi
}

# Scale worker containers
windmill::cli_scale_workers() {
    local count="${1:-1}"
    
    if command -v workers::scale &>/dev/null; then
        workers::scale "$count"
    else
        log::error "Worker scaling not available"
        return 1
    fi
}

# Create backup
windmill::cli_backup() {
    local backup_path="${1:-./windmill-backup-$(date +%Y%m%d-%H%M%S)}"
    
    if command -v windmill::backup &>/dev/null; then
        windmill::backup "$backup_path"
    else
        log::error "Backup functionality not available"
        return 1
    fi
}

# Restore from backup
windmill::cli_restore() {
    local backup_path="${1:-}"
    
    if [[ -z "$backup_path" ]]; then
        log::error "Backup path required"
        echo "Usage: resource-windmill restore <backup-path>"
        return 1
    fi
    
    if command -v windmill::restore &>/dev/null; then
        windmill::restore "$backup_path"
    else
        log::error "Restore functionality not available"
        return 1
    fi
}

# Show windmill status
windmill_status() {
    if command -v windmill::status &>/dev/null; then
        windmill::status "$@"
    else
        # Basic status
        log::header "Windmill Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "windmill"; then
            echo "Containers: ✅ Running"
            docker ps --filter "name=windmill" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | tail -n +2
        else
            echo "Containers: ❌ Not running"
        fi
    fi
}

# Start windmill
windmill_start() {
    if command -v windmill::start &>/dev/null; then
        windmill::start
    else
        log::error "windmill::start not available"
        return 1
    fi
}

# Stop windmill
windmill_stop() {
    if command -v windmill::stop &>/dev/null; then
        windmill::stop
    else
        log::error "windmill::stop not available"
        return 1
    fi
}

# Install windmill
windmill_install() {
    if command -v windmill::install &>/dev/null; then
        windmill::install
    else
        log::error "windmill::install not available"
        return 1
    fi
}

# Uninstall windmill
windmill_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Windmill and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v windmill::uninstall &>/dev/null; then
        windmill::uninstall
    else
        log::error "windmill::uninstall not available"
        return 1
    fi
}

# Get credentials for n8n integration
windmill_credentials() {
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "windmill"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "${WINDMILL_SERVER_CONTAINER:-windmill-server}")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Windmill HTTP API connection with basic auth
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "${WINDMILL_SERVER_PORT:-8000}" \
            --arg path "/api" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local auth_obj
        auth_obj=$(jq -n \
            --arg username "${WINDMILL_SUPERADMIN_EMAIL:-admin@example.com}" \
            --arg password "${WINDMILL_SUPERADMIN_PASSWORD:-password}" \
            '{
                username: $username,
                password: $password
            }')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "Windmill developer platform and workflow automation" \
            --arg web_ui "${WINDMILL_BASE_URL:-http://localhost:8000}" \
            --argjson worker_replicas "${WINDMILL_WORKER_REPLICAS:-2}" \
            --arg worker_group "${WINDMILL_WORKER_GROUP:-default}" \
            '{
                description: $description,
                web_ui_url: $web_ui,
                worker_replicas: $worker_replicas,
                worker_group: $worker_group
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "main" \
            "Windmill API" \
            "httpBasicAuth" \
            "$connection_obj" \
            "$auth_obj" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    local response
    response=$(credentials::build_response "windmill" "$status" "$connections_array")
    credentials::format_output "$response"
}

# List available apps
windmill_list_apps() {
    if command -v windmill::list_apps &>/dev/null; then
        windmill::list_apps
    else
        log::error "App listing not available"
        return 1
    fi
}

# Prepare app for deployment
windmill_prepare_app() {
    local app_name="${1:-}"
    local output_dir="${2:-.}"
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: resource-windmill prepare-app <app-name> [output-dir]"
        echo ""
        echo "Examples:"
        echo "  resource-windmill prepare-app admin-dashboard"
        echo "  resource-windmill prepare-app admin-dashboard ./apps/"
        return 1
    fi
    
    if command -v windmill::prepare_app &>/dev/null; then
        windmill::prepare_app "$app_name" "$output_dir"
    else
        log::error "App preparation not available"
        return 1
    fi
}

# Deploy app to workspace
windmill_deploy_app() {
    local app_name="${1:-}"
    local workspace="${2:-demo}"
    
    if [[ -z "$app_name" ]]; then
        log::error "App name required"
        echo "Usage: resource-windmill deploy-app <app-name> [workspace]"
        echo ""
        echo "Examples:"
        echo "  resource-windmill deploy-app admin-dashboard"
        echo "  resource-windmill deploy-app admin-dashboard production"
        return 1
    fi
    
    if command -v windmill::deploy_app &>/dev/null; then
        windmill::deploy_app "$app_name" "$workspace"
    else
        log::error "App deployment not available"
        return 1
    fi
}

# Scale workers
windmill_scale_workers() {
    local count="${1:-2}"
    
    if command -v windmill::scale_workers &>/dev/null; then
        windmill::scale_workers "$count"
    else
        log::error "Worker scaling not available"
        return 1
    fi
}

# Backup windmill data
windmill_backup() {
    local backup_path="${1:-./windmill-backup}"
    
    if command -v windmill::backup &>/dev/null; then
        windmill::backup "$backup_path"
    else
        log::error "Backup not available"
        return 1
    fi
}

# Restore windmill data
windmill_restore() {
    local backup_path="${1:-}"
    
    if [[ -z "$backup_path" ]]; then
        log::error "Backup path required"
        echo "Usage: resource-windmill restore <backup-path>"
        echo ""
        echo "Examples:"
        echo "  resource-windmill restore ./windmill-backup"
        echo "  resource-windmill restore /backups/windmill-2024-01-01.tar"
        return 1
    fi
    
    if command -v windmill::restore &>/dev/null; then
        windmill::restore "$backup_path"
    else
        log::error "Restore not available"
        return 1
    fi
}

# Show API setup instructions
windmill_api_setup() {
    if command -v windmill::show_api_setup_instructions &>/dev/null; then
        windmill::show_api_setup_instructions
    else
        log::error "API setup instructions not available"
        return 1
    fi
}

# Custom help function with Windmill-specific examples
windmill::show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add Windmill-specific examples
    echo ""
    echo "🌪️  Windmill Developer Platform Examples:"
    echo ""
    echo "Application Management:"
    echo "  resource-windmill list-apps                           # List app templates"
    echo "  resource-windmill prepare-app admin-dashboard ./apps # Prepare app deployment"
    echo "  resource-windmill deploy-app admin-dashboard prod     # Deploy to workspace"
    echo ""
    echo "Workflow Automation:"
    echo "  resource-windmill inject workflow.json               # Import workflow"
    echo "  resource-windmill inject shared:init/windmill/workflow.json  # Import shared"
    echo ""
    echo "Operations:"
    echo "  resource-windmill scale-workers 4                    # Scale to 4 workers"
    echo "  resource-windmill backup ./backup-2024-01-01         # Create backup"
    echo "  resource-windmill restore ./backup-2024-01-01        # Restore backup"
    echo "  resource-windmill api-setup                          # Show API setup"
    echo ""
    echo "Management:"
    echo "  resource-windmill status                             # Check all containers"
    echo "  resource-windmill credentials                        # Get API credentials"
    echo ""
    echo "Platform Features:"
    echo "  • TypeScript/Python/Go/Bash script execution"
    echo "  • Visual workflow builder with code integration"
    echo "  • Multi-tenant workspaces and RBAC"
    echo "  • Scalable worker architecture"
    echo ""
    echo "Default Port: 8000"
    echo "Web UI: http://localhost:8000"
    echo "Default Credentials: admin@example.com / password"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi