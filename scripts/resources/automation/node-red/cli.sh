#!/usr/bin/env bash
################################################################################
# Node-RED Resource CLI
# 
# Ultra-thin CLI wrapper that delegates directly to library functions
#
# Usage:
#   resource-node-red <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    NODE_RED_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    NODE_RED_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
NODE_RED_CLI_DIR="$(cd "$(dirname "$NODE_RED_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${NODE_RED_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source node-red configuration
# shellcheck disable=SC1091
source "${NODE_RED_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
node_red::export_config 2>/dev/null || true

# Source node-red libraries - these contain the actual functionality
# shellcheck disable=SC1091
source "${NODE_RED_CLI_DIR}/lib/core.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${NODE_RED_CLI_DIR}/lib/docker.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${NODE_RED_CLI_DIR}/lib/health.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${NODE_RED_CLI_DIR}/lib/recovery.sh" 2>/dev/null || true

# Initialize CLI framework
cli::init "node-red" "Node-RED flow-based automation platform"

# Override help to provide Node-RED-specific examples
cli::register_command "help" "Show this help message with examples" "node_red::show_help"

# Register core commands - direct library function calls
cli::register_command "install" "Install Node-RED" "node_red::install" "modifies-system"
cli::register_command "uninstall" "Uninstall Node-RED" "node_red::cli_uninstall" "modifies-system"
cli::register_command "start" "Start Node-RED" "node_red::start" "modifies-system"
cli::register_command "stop" "Stop Node-RED" "node_red::stop" "modifies-system"

# Register status and monitoring commands
cli::register_command "status" "Show service status" "node_red::status"
cli::register_command "validate" "Validate installation" "node_red::health"
cli::register_command "logs" "Show container logs" "node_red::cli_logs"

# Register flow management commands
cli::register_command "inject" "Inject flows into Node-RED" "node_red::cli_inject" "modifies-system"
cli::register_command "list-flows" "List all flows" "node_red::list_flows"
cli::register_command "export-flows" "Export flows to file" "node_red::cli_export_flows" "modifies-system"
cli::register_command "import-flows" "Import flows from file" "node_red::cli_import_flows" "modifies-system"
cli::register_command "enable-flow" "Enable specific flow" "node_red::cli_enable_flow" "modifies-system"
cli::register_command "disable-flow" "Disable specific flow" "node_red::cli_disable_flow" "modifies-system"

# Register backup commands
cli::register_command "create-backup" "Create backup" "node_red::cli_create_backup" "modifies-system"
cli::register_command "list-backups" "List all backups" "node_red::list_backups"
cli::register_command "restore-backup" "Restore from backup" "node_red::cli_restore_backup" "modifies-system"

# Register utility commands
cli::register_command "credentials" "Show Node-RED credentials for integration" "node_red::cli_credentials"

################################################################################
# CLI wrapper functions - minimal wrappers for commands that need argument handling
################################################################################

# Inject flows into node-red
node_red::cli_inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-node-red inject <file.json>"
        echo ""
        echo "Examples:"
        echo "  resource-node-red inject flows.json"
        echo "  resource-node-red inject shared:initialization/automation/node-red/flows.json"
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
    if command -v node_red::inject &>/dev/null; then
        node_red::inject "$file"
    elif command -v node_red::import_flows &>/dev/null; then
        node_red::import_flows "$file"
    else
        log::error "node-red injection functions not available"
        return 1
    fi
}

# Uninstall with force confirmation
node_red::cli_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Node-RED and all its data. Use --force to confirm."
        return 1
    fi
    
    node_red::uninstall
}

# Show logs with line count
node_red::cli_logs() {
    local lines="${1:-50}"
    local follow="${2:-false}"
    local container_name="${NODE_RED_CONTAINER_NAME:-node-red}"
    
    # Use shared utility with follow support
    docker_resource::show_logs_with_follow "$container_name" "$lines" "$follow"
}

# Show credentials for Node-RED integration
node_red::cli_credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    credentials::parse_args "$@" || return $?
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "${CONTAINER_NAME:-node-red}")
    
    # Build connections array
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Build connection JSON for Node-RED API
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "${NODE_RED_PORT:-1880}" \
            --arg path "/" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "Node-RED flow-based automation API" \
            --arg web_ui "${NODE_RED_BASE_URL:-http://localhost:1880}" \
            '{
                description: $description,
                web_ui_url: $web_ui,
                capabilities: ["flows", "automation", "webhook", "mqtt", "http"],
                version: "latest"
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "api" \
            "Node-RED API" \
            "httpHeaderAuth" \
            "$connection_obj" \
            "{}" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "node-red" "$status" "$connections_array")
    
    credentials::format_output "$response"
}

# Export flows to file
node_red::cli_export_flows() {
    local output_file="${1:-./flows_export.json}"
    
    node_red::export_flows "$output_file"
}

# Import flows from file
node_red::cli_import_flows() {
    local input_file="${1:-}"
    
    if [[ -z "$input_file" ]]; then
        log::error "Input file required"
        echo "Usage: resource-node-red import-flows <file.json>"
        echo ""
        echo "Examples:"
        echo "  resource-node-red import-flows my-flows.json"
        echo "  resource-node-red import-flows shared:examples/node-red/flows.json"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$input_file" == shared:* ]]; then
        input_file="${var_VROOLI_ROOT}/${input_file#shared:}"
    fi
    
    if [[ ! -f "$input_file" ]]; then
        log::error "Input file not found: $input_file"
        return 1
    fi
    
    node_red::import_flows "$input_file"
}

# Enable specific flow
node_red::cli_enable_flow() {
    local flow_id="${1:-}"
    
    if [[ -z "$flow_id" ]]; then
        log::error "Flow ID required"
        echo "Usage: resource-node-red enable-flow <flow-id>"
        echo ""
        echo "Examples:"
        echo "  resource-node-red enable-flow flow-123"
        echo "  resource-node-red enable-flow main-automation"
        return 1
    fi
    
    node_red::enable_flow "$flow_id"
}

# Disable specific flow
node_red::cli_disable_flow() {
    local flow_id="${1:-}"
    
    if [[ -z "$flow_id" ]]; then
        log::error "Flow ID required"
        echo "Usage: resource-node-red disable-flow <flow-id>"
        echo ""
        echo "Examples:"
        echo "  resource-node-red disable-flow flow-123"
        echo "  resource-node-red disable-flow main-automation"
        return 1
    fi
    
    node_red::disable_flow "$flow_id"
}

# Create backup with optional name
node_red::cli_create_backup() {
    local backup_name="${1:-backup-$(date +%Y%m%d-%H%M%S)}"
    
    node_red::create_backup "$backup_name"
}

# Restore backup by name
node_red::cli_restore_backup() {
    local backup_name="${1:-}"
    
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name required"
        echo "Usage: resource-node-red restore-backup <backup-name>"
        echo ""
        echo "Examples:"
        echo "  resource-node-red restore-backup backup-20240101-120000"
        echo "  resource-node-red restore-backup production-backup"
        return 1
    fi
    
    node_red::restore_backup "$backup_name"
}

# Custom help function with examples
node_red::show_help() {
    cli::_handle_help
    
    echo ""
    echo "⚡ Examples:"
    echo ""
    echo "  # Flow management"
    echo "  resource-node-red inject flows.json"
    echo "  resource-node-red inject shared:initialization/node-red/flows.json"
    echo "  resource-node-red list-flows"
    echo "  resource-node-red export-flows ./backup-flows.json"
    echo "  resource-node-red enable-flow main-automation"
    echo ""
    echo "  # Backup and recovery"
    echo "  resource-node-red create-backup production-backup"
    echo "  resource-node-red list-backups"
    echo "  resource-node-red restore-backup production-backup"
    echo ""
    echo "  # Management"
    echo "  resource-node-red status"
    echo "  resource-node-red logs 100"
    echo "  resource-node-red credentials"
    echo ""
    echo "  # Dangerous operations"
    echo "  resource-node-red uninstall --force"
    echo ""
    echo "Default Port: ${NODE_RED_PORT:-1880}"
    echo "Web UI: http://localhost:${NODE_RED_PORT:-1880}"
    echo "Features: Flow-based programming, 3000+ nodes, dashboards"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi