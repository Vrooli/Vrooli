#!/usr/bin/env bash
################################################################################
# Node-RED Resource CLI
# 
# Lightweight CLI interface for Node-RED using the CLI Command Framework
#
# Usage:
#   resource-node-red <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory
NODE_RED_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$NODE_RED_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$NODE_RED_CLI_DIR"
export NODE_RED_SCRIPT_DIR="$NODE_RED_CLI_DIR"  # For compatibility with existing libs

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

# Source node-red configuration
# shellcheck disable=SC1091
source "${NODE_RED_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
node_red::export_config 2>/dev/null || true

# Source node-red libraries
for lib in core docker health recovery; do
    lib_file="${NODE_RED_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "node-red" "Node-RED flow-based automation platform"

# Override help to provide Node-RED-specific examples
cli::register_command "help" "Show this help message with Node-RED examples" "resource_cli::show_help"

# Register additional Node-RED-specific commands
cli::register_command "inject" "Inject flows into Node-RED" "resource_cli::inject" "modifies-system"
cli::register_command "list-flows" "List all flows" "resource_cli::list_flows"
cli::register_command "export-flows" "Export flows to file" "resource_cli::export_flows" "modifies-system"
cli::register_command "import-flows" "Import flows from file" "resource_cli::import_flows" "modifies-system"
cli::register_command "enable-flow" "Enable specific flow" "resource_cli::enable_flow" "modifies-system"
cli::register_command "disable-flow" "Disable specific flow" "resource_cli::disable_flow" "modifies-system"
cli::register_command "create-backup" "Create backup" "resource_cli::create_backup" "modifies-system"
cli::register_command "list-backups" "List all backups" "resource_cli::list_backups"
cli::register_command "restore-backup" "Restore from backup" "resource_cli::restore_backup" "modifies-system"
cli::register_command "view-logs" "View container logs" "resource_cli::view_logs"
cli::register_command "credentials" "Show n8n credentials for Node-RED" "resource_cli::credentials"
cli::register_command "uninstall" "Uninstall Node-RED (requires --force)" "resource_cli::uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject flows into node-red
resource_cli::inject() {
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
        file="${VROOLI_ROOT}/${file#shared:}"
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

# Validate node-red configuration
resource_cli::validate() {
    if command -v node_red::health &>/dev/null; then
        node_red::health
    elif command -v node_red::is_responding &>/dev/null; then
        if node_red::is_responding; then
            log::success "Node-RED is responding"
        else
            log::error "Node-RED health check failed"
            return 1
        fi
    else
        # Basic validation
        log::header "Validating Node-RED"
        local container_name="${CONTAINER_NAME:-node-red}"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"; then
            log::success "Node-RED is running"
        else
            log::error "Node-RED container not running"
            return 1
        fi
    fi
}

# Show node-red status
resource_cli::status() {
    if command -v node_red::status &>/dev/null; then
        node_red::status
    else
        # Basic status
        log::header "Node-RED Status"
        local container_name="${CONTAINER_NAME:-node-red}"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=$container_name" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start node-red
resource_cli::start() {
    if command -v node_red::start &>/dev/null; then
        node_red::start
    else
        log::error "node_red::start not available"
        return 1
    fi
}

# Stop node-red
resource_cli::stop() {
    if command -v node_red::stop &>/dev/null; then
        node_red::stop
    else
        log::error "node_red::stop not available"
        return 1
    fi
}

# Install node-red
resource_cli::install() {
    if command -v node_red::install &>/dev/null; then
        node_red::install
    else
        log::error "node_red::install not available"
        return 1
    fi
}

# Uninstall node-red
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove node-red and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v node_red::uninstall &>/dev/null; then
        node_red::uninstall
    else
        log::error "node_red::uninstall not available"
        return 1
    fi
}

# Get credentials for n8n integration
resource_cli::credentials() {
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "node-red"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "${CONTAINER_NAME:-node-red}")
    
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
    
    local response
    response=$(credentials::build_response "node-red" "$status" "$connections_array")
    credentials::format_output "$response"
}

# List flows
resource_cli::list_flows() {
    if command -v node_red::list_flows &>/dev/null; then
        node_red::list_flows
    else
        log::error "Flow listing not available"
        return 1
    fi
}

# Export flows
resource_cli::export_flows() {
    local output_file="${1:-./flows_export.json}"
    
    if command -v node_red::export_flows &>/dev/null; then
        node_red::export_flows "$output_file"
    else
        log::error "Flow export not available"
        return 1
    fi
}

# Import flows
resource_cli::import_flows() {
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
        input_file="${VROOLI_ROOT}/${input_file#shared:}"
    fi
    
    if [[ ! -f "$input_file" ]]; then
        log::error "Input file not found: $input_file"
        return 1
    fi
    
    if command -v node_red::import_flows &>/dev/null; then
        node_red::import_flows "$input_file"
    else
        log::error "Flow import not available"
        return 1
    fi
}

# Enable flow
resource_cli::enable_flow() {
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
    
    if command -v node_red::enable_flow &>/dev/null; then
        node_red::enable_flow "$flow_id"
    else
        log::error "Flow enable not available"
        return 1
    fi
}

# Disable flow
resource_cli::disable_flow() {
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
    
    if command -v node_red::disable_flow &>/dev/null; then
        node_red::disable_flow "$flow_id"
    else
        log::error "Flow disable not available"
        return 1
    fi
}

# Create backup
resource_cli::create_backup() {
    local backup_name="${1:-backup-$(date +%Y%m%d-%H%M%S)}"
    
    if command -v node_red::create_backup &>/dev/null; then
        node_red::create_backup "$backup_name"
    else
        log::error "Backup creation not available"
        return 1
    fi
}

# List backups
resource_cli::list_backups() {
    if command -v node_red::list_backups &>/dev/null; then
        node_red::list_backups
    else
        log::error "Backup listing not available"
        return 1
    fi
}

# Restore backup
resource_cli::restore_backup() {
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
    
    if command -v node_red::restore_backup &>/dev/null; then
        node_red::restore_backup "$backup_name"
    else
        log::error "Backup restore not available"
        return 1
    fi
}

# View logs
resource_cli::view_logs() {
    local lines="${1:-50}"
    local follow="${2:-false}"
    local container_name="${CONTAINER_NAME:-node-red}"
    
    if command -v node_red::view_logs &>/dev/null; then
        node_red::view_logs "$lines"
    else
        if [[ "$follow" == "true" ]]; then
            docker logs -f --tail "$lines" "$container_name"
        else
            docker logs --tail "$lines" "$container_name"
        fi
    fi
}

# Custom help function with Node-RED-specific examples
resource_cli::show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add Node-RED-specific examples
    echo ""
    echo "🔴 Node-RED Flow Automation Examples:"
    echo ""
    echo "Flow Management:"
    echo "  resource-node-red list-flows                        # List all flows"
    echo "  resource-node-red export-flows ./my-flows.json     # Export flows to file"
    echo "  resource-node-red import-flows ./my-flows.json     # Import flows from file"
    echo "  resource-node-red inject shared:init/node-red/flows.json  # Import shared flows"
    echo ""
    echo "Flow Control:"
    echo "  resource-node-red enable-flow main-automation      # Enable specific flow"
    echo "  resource-node-red disable-flow test-flow           # Disable specific flow"
    echo ""
    echo "Backup & Recovery:"
    echo "  resource-node-red create-backup production-backup  # Create named backup"
    echo "  resource-node-red create-backup                    # Create timestamped backup"
    echo "  resource-node-red list-backups                     # List all backups"
    echo "  resource-node-red restore-backup production-backup # Restore backup"
    echo ""
    echo "Monitoring:"
    echo "  resource-node-red view-logs 100                    # View recent logs"
    echo "  resource-node-red status                           # Check service status"
    echo "  resource-node-red credentials                      # Get API details"
    echo ""
    echo "Automation Features:"
    echo "  • Visual flow-based programming"
    echo "  • 3000+ nodes for integrations"
    echo "  • MQTT, HTTP, WebSocket support"
    echo "  • Dashboard and UI components"
    echo ""
    echo "Default Port: 1880"
    echo "Web UI: http://localhost:1880"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi