#!/usr/bin/env bash
################################################################################
# Vrooli CLI - Orchestrator Management Commands
# 
# Unified interface for managing the Vrooli Orchestrator through the main
# vrooli CLI. This provides a consistent command experience while delegating
# to the orchestrator control script for actual functionality.
#
# Usage:
#   vrooli orchestrator <subcommand> [options]
#
################################################################################

set -euo pipefail

# Get CLI directory and orchestrator control script
CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ORCHESTRATOR_CTL="${CLI_DIR}/../../scripts/scenarios/tools/orchestrator-ctl.sh"

# Source utilities
# shellcheck disable=SC1091
source "${CLI_DIR}/../../scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Check if orchestrator control script exists
check_orchestrator_ctl() {
    if [[ ! -x "$ORCHESTRATOR_CTL" ]]; then
        log::error "Orchestrator control script not found or not executable: $ORCHESTRATOR_CTL"
        echo ""
        echo "Please run 'vrooli setup' to install the orchestrator"
        return 1
    fi
}

# Show help for orchestrator commands
show_orchestrator_help() {
    cat << EOF
${CYAN}🎼 Vrooli Orchestrator Commands${NC}

${YELLOW}USAGE:${NC}
    vrooli orchestrator <subcommand> [options]

${YELLOW}LIFECYCLE COMMANDS:${NC}
    start               Start the orchestrator daemon
    stop                Stop daemon, processes, and resources
                        (use --no-resources to skip resource shutdown)
    restart             Restart the orchestrator
    status              Show process status
    tree                Show process hierarchy tree
    
${YELLOW}MONITORING COMMANDS:${NC}
    list                List all processes with details
    logs <name>         Show logs for a process
    follow <name>       Follow logs in real-time
    monitor             Real-time dashboard
    health              Health check all processes
    
${YELLOW}MANAGEMENT COMMANDS:${NC}
    clean               Clean up orphaned processes
    backup              Backup process registry
    restore <file>      Restore process registry
    config              Show orchestrator configuration

${YELLOW}OPTIONS:${NC}
    --help, -h          Show this help message
    --verbose, -v       Verbose output
    --json              Output in JSON format (where applicable)
    --follow            Follow logs/status in real-time

${YELLOW}EXAMPLES:${NC}
    vrooli orchestrator start                    # Start the daemon
    vrooli orchestrator status --json            # Get status in JSON
    vrooli orchestrator logs research-assistant  # Show app logs
    vrooli orchestrator follow api               # Follow API logs
    vrooli orchestrator tree                     # Show process tree
    vrooli orchestrator monitor                  # Real-time dashboard
    vrooli orchestrator clean                    # Clean up orphaned processes

${YELLOW}INTEGRATION WITH VROOLI DEVELOP:${NC}
    When you run 'vrooli develop', it automatically:
    1. Starts the orchestrator daemon
    2. Discovers all enabled scenarios
    3. Starts their generated apps with proper port assignment
    4. Provides centralized monitoring and control

${YELLOW}ENVIRONMENT VARIABLES:${NC}
    VROOLI_ORCHESTRATOR_HOME    Orchestrator home directory
    VROOLI_MAX_APPS             Maximum total apps (default: 20)
    VROOLI_MAX_DEPTH            Maximum nesting depth (default: 5)
    VROOLI_MAX_PER_PARENT       Maximum apps per parent (default: 10)

For more information: https://docs.vrooli.com/orchestrator
EOF
}

# Delegate start command
orchestrator_start() {
    check_orchestrator_ctl || return 1
    
    log::info "Starting Vrooli Orchestrator..."
    exec "$ORCHESTRATOR_CTL" start "$@"
}

# Delegate stop command (enhanced to stop resources too)
orchestrator_stop() {
    check_orchestrator_ctl || return 1
    
    log::info "Stopping Vrooli Orchestrator..."
    
    # First stop the orchestrator and its managed processes
    "$ORCHESTRATOR_CTL" stop "$@"
    
    # Then stop all resources if --with-resources flag is provided
    local stop_resources=false
    for arg in "$@"; do
        if [[ "$arg" == "--with-resources" || "$arg" == "--all" ]]; then
            stop_resources=true
            break
        fi
    done
    
    # Default to stopping resources unless --no-resources is specified
    local skip_resources=false
    for arg in "$@"; do
        if [[ "$arg" == "--no-resources" ]]; then
            skip_resources=true
            break
        fi
    done
    
    # Stop resources by default (unless explicitly skipped)
    if [[ "$skip_resources" == "false" ]]; then
        log::info "Stopping all resources..."
        
        # Source the auto-install module if available
        if [[ -f "${var_LIB_DIR}/resources/auto-install.sh" ]]; then
            # shellcheck disable=SC1091
            source "${var_LIB_DIR}/resources/auto-install.sh"
            resource_auto::stop_all
        else
            # Fallback to using vrooli resource stop-all
            "${CLI_DIR}/../../cli/vrooli" resource stop-all
        fi
    fi
    
    log::success "✅ Orchestrator and resources stopped"
}

# Delegate restart command
orchestrator_restart() {
    check_orchestrator_ctl || return 1
    
    log::info "Restarting Vrooli Orchestrator..."
    exec "$ORCHESTRATOR_CTL" restart "$@"
}

# Delegate status command
orchestrator_status() {
    check_orchestrator_ctl || return 1
    
    exec "$ORCHESTRATOR_CTL" status "$@"
}

# Delegate tree command
orchestrator_tree() {
    check_orchestrator_ctl || return 1
    
    exec "$ORCHESTRATOR_CTL" tree "$@"
}

# Delegate list command
orchestrator_list() {
    check_orchestrator_ctl || return 1
    
    exec "$ORCHESTRATOR_CTL" list "$@"
}

# Delegate logs command
orchestrator_logs() {
    if [[ $# -eq 0 ]]; then
        log::error "Process name required"
        echo "Usage: vrooli orchestrator logs <process-name> [options]"
        echo ""
        echo "Examples:"
        echo "  vrooli orchestrator logs research-assistant    # Show recent logs"
        echo "  vrooli orchestrator logs api --follow          # Follow logs in real-time"
        return 1
    fi
    
    check_orchestrator_ctl || return 1
    
    exec "$ORCHESTRATOR_CTL" logs "$@"
}

# Delegate follow command (alias for logs --follow)
orchestrator_follow() {
    if [[ $# -eq 0 ]]; then
        log::error "Process name required"
        echo "Usage: vrooli orchestrator follow <process-name>"
        return 1
    fi
    
    check_orchestrator_ctl || return 1
    
    exec "$ORCHESTRATOR_CTL" follow "$@"
}

# Delegate monitor command
orchestrator_monitor() {
    check_orchestrator_ctl || return 1
    
    log::info "Starting real-time monitor (Ctrl+C to exit)..."
    exec "$ORCHESTRATOR_CTL" monitor "$@"
}

# Delegate health command
orchestrator_health() {
    check_orchestrator_ctl || return 1
    
    exec "$ORCHESTRATOR_CTL" health "$@"
}

# Delegate clean command
orchestrator_clean() {
    check_orchestrator_ctl || return 1
    
    log::info "Cleaning up orphaned processes..."
    exec "$ORCHESTRATOR_CTL" clean "$@"
}

# Delegate backup command
orchestrator_backup() {
    check_orchestrator_ctl || return 1
    
    log::info "Creating backup..."
    exec "$ORCHESTRATOR_CTL" backup "$@"
}

# Delegate restore command
orchestrator_restore() {
    if [[ $# -eq 0 ]]; then
        log::error "Backup file required"
        echo "Usage: vrooli orchestrator restore <backup-file>"
        return 1
    fi
    
    check_orchestrator_ctl || return 1
    
    log::info "Restoring from backup..."
    exec "$ORCHESTRATOR_CTL" restore "$@"
}

# Delegate config command
orchestrator_config() {
    check_orchestrator_ctl || return 1
    
    exec "$ORCHESTRATOR_CTL" config "$@"
}

# Show quick status for integration with other commands
orchestrator_quick_status() {
    check_orchestrator_ctl || return 1
    
    # Quick status without full formatting
    if "$ORCHESTRATOR_CTL" status >/dev/null 2>&1; then
        local registry_file="${VROOLI_ORCHESTRATOR_HOME:-$HOME/.vrooli/orchestrator}/processes.json"
        if [[ -f "$registry_file" ]]; then
            local total=$(jq '.processes | length' "$registry_file" 2>/dev/null || echo "0")
            local running=$(jq '[.processes[] | select(.state == "running")] | length' "$registry_file" 2>/dev/null || echo "0")
            echo -e "${GREEN}●${NC} Orchestrator running: $running/$total processes active"
        else
            echo -e "${GREEN}●${NC} Orchestrator running"
        fi
    else
        echo -e "${YELLOW}○${NC} Orchestrator not running"
    fi
}

# Main command handler
main() {
    # Handle no arguments
    if [[ $# -eq 0 ]]; then
        show_orchestrator_help
        return 0
    fi
    
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        start)
            orchestrator_start "$@"
            ;;
            
        stop)
            orchestrator_stop "$@"
            ;;
            
        restart)
            orchestrator_restart "$@"
            ;;
            
        status)
            orchestrator_status "$@"
            ;;
            
        tree)
            orchestrator_tree "$@"
            ;;
            
        list)
            orchestrator_list "$@"
            ;;
            
        logs)
            orchestrator_logs "$@"
            ;;
            
        follow)
            orchestrator_follow "$@"
            ;;
            
        monitor)
            orchestrator_monitor "$@"
            ;;
            
        health)
            orchestrator_health "$@"
            ;;
            
        clean)
            orchestrator_clean "$@"
            ;;
            
        backup)
            orchestrator_backup "$@"
            ;;
            
        restore)
            orchestrator_restore "$@"
            ;;
            
        config)
            orchestrator_config "$@"
            ;;
            
        # Hidden command for integration
        --quick-status)
            orchestrator_quick_status "$@"
            ;;
            
        --help|-h)
            show_orchestrator_help
            ;;
            
        *)
            log::error "Unknown orchestrator command: $subcommand"
            echo ""
            show_orchestrator_help
            return 1
            ;;
    esac
}

# Execute main function
main "$@"