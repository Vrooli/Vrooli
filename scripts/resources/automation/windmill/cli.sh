#!/usr/bin/env bash
################################################################################
# Windmill Resource CLI
# 
# Lightweight CLI interface for Windmill using the CLI Command Framework
#
# Usage:
#   resource-windmill <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (handle symlinks)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    WINDMILL_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    WINDMILL_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
WINDMILL_CLI_DIR="$(cd "$(dirname "$WINDMILL_CLI_SCRIPT")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$WINDMILL_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$WINDMILL_CLI_DIR"
export WINDMILL_SCRIPT_DIR="$WINDMILL_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-${VROOLI_ROOT}/scripts/resources/common.sh}" 2>/dev/null || true

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/resources/lib/cli-command-framework.sh"

# Source windmill configuration
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
windmill::export_config 2>/dev/null || true

# Source windmill libraries
for lib in common docker status install apps api workers; do
    lib_file="${WINDMILL_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "windmill" "Windmill developer platform and workflow automation"

# Override help to provide Windmill-specific examples
cli::register_command "help" "Show this help message with Windmill examples" "resource_cli::show_help"

# Register additional Windmill-specific commands
cli::register_command "inject" "Inject workflow/app into Windmill" "resource_cli::inject" "modifies-system"
cli::register_command "list-apps" "List available app examples" "resource_cli::list_apps"
cli::register_command "prepare-app" "Prepare app for deployment" "resource_cli::prepare_app" "modifies-system"
cli::register_command "deploy-app" "Deploy app to workspace" "resource_cli::deploy_app" "modifies-system"
cli::register_command "scale-workers" "Scale worker containers" "resource_cli::scale_workers" "modifies-system"
cli::register_command "backup" "Backup Windmill data" "resource_cli::backup" "modifies-system"
cli::register_command "restore" "Restore from backup" "resource_cli::restore" "modifies-system"
cli::register_command "api-setup" "Show API setup instructions" "resource_cli::api_setup"
cli::register_command "credentials" "Show n8n credentials for Windmill" "resource_cli::credentials"
cli::register_command "uninstall" "Uninstall Windmill (requires --force)" "resource_cli::uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject workflows or apps into windmill
resource_cli::inject() {
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
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Use existing injection function
    if command -v windmill::inject &>/dev/null; then
        windmill::inject
    else
        "${WINDMILL_CLI_DIR}/inject.sh" --inject "$(cat "$file")"
    fi
}

# Validate windmill configuration
resource_cli::validate() {
    if command -v windmill::validate &>/dev/null; then
        windmill::validate
    elif command -v windmill::status &>/dev/null; then
        windmill::status
    else
        # Basic validation
        log::header "Validating Windmill"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "windmill" || {
            log::error "Windmill containers not running"
            return 1
        }
        log::success "Windmill is running"
    fi
}

# Show windmill status
resource_cli::status() {
    if command -v windmill::status &>/dev/null; then
        windmill::status
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
resource_cli::start() {
    if command -v windmill::start &>/dev/null; then
        windmill::start
    else
        log::error "windmill::start not available"
        return 1
    fi
}

# Stop windmill
resource_cli::stop() {
    if command -v windmill::stop &>/dev/null; then
        windmill::stop
    else
        log::error "windmill::stop not available"
        return 1
    fi
}

# Install windmill
resource_cli::install() {
    if command -v windmill::install &>/dev/null; then
        windmill::install
    else
        log::error "windmill::install not available"
        return 1
    fi
}

# Uninstall windmill
resource_cli::uninstall() {
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
resource_cli::credentials() {
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
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
resource_cli::list_apps() {
    if command -v windmill::list_apps &>/dev/null; then
        windmill::list_apps
    else
        log::error "App listing not available"
        return 1
    fi
}

# Prepare app for deployment
resource_cli::prepare_app() {
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
resource_cli::deploy_app() {
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
resource_cli::scale_workers() {
    local count="${1:-2}"
    
    if command -v windmill::scale_workers &>/dev/null; then
        windmill::scale_workers "$count"
    else
        log::error "Worker scaling not available"
        return 1
    fi
}

# Backup windmill data
resource_cli::backup() {
    local backup_path="${1:-./windmill-backup}"
    
    if command -v windmill::backup &>/dev/null; then
        windmill::backup "$backup_path"
    else
        log::error "Backup not available"
        return 1
    fi
}

# Restore windmill data
resource_cli::restore() {
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
resource_cli::api_setup() {
    if command -v windmill::show_api_setup_instructions &>/dev/null; then
        windmill::show_api_setup_instructions
    else
        log::error "API setup instructions not available"
        return 1
    fi
}

# Custom help function with Windmill-specific examples
resource_cli::show_help() {
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