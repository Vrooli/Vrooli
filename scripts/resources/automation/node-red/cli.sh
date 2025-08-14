#!/usr/bin/env bash
################################################################################
# Node-RED Resource CLI
# 
# Lightweight CLI interface for Node-RED that delegates to existing lib functions.
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
source "${var_RESOURCES_COMMON_FILE}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

# Initialize with resource name first (before sourcing config)
resource_cli::init "node-red"

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

################################################################################
# Delegate to existing node-red functions
################################################################################

# Inject flows into node-red
resource_cli::inject() {
    local file="${1:-}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$file" ]]; then
        log::error "File path required for injection"
        echo "Usage: resource-node-red inject <file.json>"
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
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "node-red"; then
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
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "node-red"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=node-red" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start node-red
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start node-red"
        return 0
    fi
    
    if command -v node_red::start &>/dev/null; then
        node_red::start
    else
        log::error "node_red::start not available"
        return 1
    fi
}

# Stop node-red
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop node-red"
        return 0
    fi
    
    if command -v node_red::stop &>/dev/null; then
        node_red::stop
    else
        log::error "node_red::stop not available"
        return 1
    fi
}

# Install node-red
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install node-red"
        return 0
    fi
    
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
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove node-red and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall node-red"
        return 0
    fi
    
    if command -v node_red::uninstall &>/dev/null; then
        node_red::uninstall
    else
        log::error "node_red::uninstall not available"
        return 1
    fi
}

################################################################################
# Node-RED-specific commands (if functions exist)
################################################################################

# List flows
node_red_list_flows() {
    if command -v node_red::list_flows &>/dev/null; then
        node_red::list_flows
    else
        log::error "Flow listing not available"
        return 1
    fi
}

# Export flows
node_red_export_flows() {
    local output_file="${1:-./flows_export.json}"
    
    if command -v node_red::export_flows &>/dev/null; then
        node_red::export_flows "$output_file"
    else
        log::error "Flow export not available"
        return 1
    fi
}

# Import flows
node_red_import_flows() {
    local input_file="${1:-}"
    if [[ -z "$input_file" ]]; then
        log::error "Input file required"
        echo "Usage: resource-node-red import-flows <file.json>"
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
node_red_enable_flow() {
    local flow_id="${1:-}"
    if [[ -z "$flow_id" ]]; then
        log::error "Flow ID required"
        echo "Usage: resource-node-red enable-flow <flow-id>"
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
node_red_disable_flow() {
    local flow_id="${1:-}"
    if [[ -z "$flow_id" ]]; then
        log::error "Flow ID required"
        echo "Usage: resource-node-red disable-flow <flow-id>"
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
node_red_create_backup() {
    local backup_name="${1:-backup-$(date +%Y%m%d-%H%M%S)}"
    
    if command -v node_red::create_backup &>/dev/null; then
        node_red::create_backup "$backup_name"
    else
        log::error "Backup creation not available"
        return 1
    fi
}

# List backups
node_red_list_backups() {
    if command -v node_red::list_backups &>/dev/null; then
        node_red::list_backups
    else
        log::error "Backup listing not available"
        return 1
    fi
}

# Restore backup
node_red_restore_backup() {
    local backup_name="${1:-}"
    if [[ -z "$backup_name" ]]; then
        log::error "Backup name required"
        echo "Usage: resource-node-red restore-backup <backup-name>"
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
node_red_view_logs() {
    local lines="${1:-50}"
    
    if command -v node_red::view_logs &>/dev/null; then
        node_red::view_logs "$lines"
    else
        docker logs "node-red" --tail "$lines" -f
    fi
}

# Show help
resource_cli::show_help() {
    cat << EOF
🔴 Node-RED Resource CLI

USAGE:
    resource-node-red <command> [options]

CORE COMMANDS:
    inject <file>       Inject flows into node-red
    validate            Validate node-red configuration
    status              Show node-red status
    start               Start node-red container
    stop                Stop node-red container
    install             Install node-red
    uninstall           Uninstall node-red (requires --force)
    
NODE-RED COMMANDS:
    list-flows          List all flows
    export-flows [file] Export flows to file (default: ./flows_export.json)
    import-flows <file> Import flows from file
    enable-flow <id>    Enable specific flow
    disable-flow <id>   Disable specific flow
    create-backup [name] Create backup (default: auto-named)
    list-backups        List all backups
    restore-backup <name> Restore from backup
    view-logs [lines]   View container logs (default: 50)

OPTIONS:
    --verbose, -v       Show detailed output
    --dry-run           Preview actions without executing
    --force             Force operation (skip confirmations)

EXAMPLES:
    resource-node-red status
    resource-node-red list-flows
    resource-node-red export-flows ./my-flows.json
    resource-node-red import-flows ./my-flows.json
    resource-node-red inject shared:initialization/automation/node-red/flows.json
    resource-node-red create-backup production-backup
    resource-node-red view-logs 100

For more information: https://docs.vrooli.com/resources/node-red
EOF
}

# Main command router
resource_cli::main() {
    # Parse common options first
    local remaining_args
    remaining_args=$(resource_cli::parse_options "$@")
    set -- $remaining_args
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Standard resource commands
        inject|validate|status|start|stop|install|uninstall)
            resource_cli::$command "$@"
            ;;
            
        # Node-RED-specific commands
        list-flows)
            node_red_list_flows "$@"
            ;;
        export-flows)
            node_red_export_flows "$@"
            ;;
        import-flows)
            node_red_import_flows "$@"
            ;;
        enable-flow)
            node_red_enable_flow "$@"
            ;;
        disable-flow)
            node_red_disable_flow "$@"
            ;;
        create-backup)
            node_red_create_backup "$@"
            ;;
        list-backups)
            node_red_list_backups "$@"
            ;;
        restore-backup)
            node_red_restore_backup "$@"
            ;;
        view-logs)
            node_red_view_logs "$@"
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