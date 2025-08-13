#!/usr/bin/env bash
################################################################################
# Vrooli Orchestrator Control Script
# 
# User-friendly interface for managing the Vrooli Orchestrator and all its
# managed processes. This script provides commands for lifecycle management,
# monitoring, and troubleshooting.
#
# Usage:
#   orchestrator-ctl <command> [options]
#
# Commands:
#   start       - Start the orchestrator daemon
#   stop        - Stop daemon and all processes
#   restart     - Restart the orchestrator
#   status      - Show process status
#   tree        - Show process hierarchy
#   list        - List all processes
#   logs        - Show process logs
#   monitor     - Real-time process monitoring
#   clean       - Clean up orphaned processes
#   backup      - Backup process registry
#   restore     - Restore process registry
################################################################################

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ORCHESTRATOR_SCRIPT="$SCRIPT_DIR/vrooli-orchestrator.sh"
CLIENT_SCRIPT="$SCRIPT_DIR/orchestrator-client.sh"
ORCHESTRATOR_HOME="${VROOLI_ORCHESTRATOR_HOME:-$HOME/.vrooli/orchestrator}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# Check dependencies
check_dependencies() {
    local missing=()
    
    # Check for required commands
    for cmd in jq curl; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing+=("$cmd")
        fi
    done
    
    # Check for orchestrator script
    if [[ ! -f "$ORCHESTRATOR_SCRIPT" ]]; then
        missing+=("vrooli-orchestrator.sh")
    fi
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        echo -e "${RED}❌ Missing dependencies:${NC}" >&2
        printf '  %s\n' "${missing[@]}" >&2
        echo ""
        echo "Please install the missing dependencies and try again." >&2
        return 1
    fi
}

# Show help
show_help() {
    cat << EOF
${CYAN}🎼 Vrooli Orchestrator Control${NC}

${YELLOW}USAGE:${NC}
    orchestrator-ctl <command> [options]

${YELLOW}LIFECYCLE COMMANDS:${NC}
    start               Start the orchestrator daemon
    stop                Stop daemon and all processes  
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
    limits              Show and modify resource limits
    
${YELLOW}DEVELOPMENT COMMANDS:${NC}
    debug               Enable debug mode and show detailed logs
    validate            Validate process registry integrity
    reset               Reset orchestrator to clean state (DANGER!)

${YELLOW}OPTIONS:${NC}
    --help, -h          Show this help message
    --verbose, -v       Verbose output
    --json             Output in JSON format (where applicable)
    --follow           Follow logs/status in real-time

${YELLOW}EXAMPLES:${NC}
    orchestrator-ctl start                    # Start the daemon
    orchestrator-ctl status --json            # Get status in JSON
    orchestrator-ctl logs api --follow        # Follow API logs
    orchestrator-ctl tree                     # Show process tree
    orchestrator-ctl monitor                  # Real-time dashboard
    orchestrator-ctl clean                    # Clean up orphaned processes

${YELLOW}ENVIRONMENT VARIABLES:${NC}
    VROOLI_ORCHESTRATOR_HOME    Orchestrator home directory
    VROOLI_MAX_APPS             Maximum total apps (default: 20)
    VROOLI_MAX_DEPTH            Maximum nesting depth (default: 5)
    VROOLI_MAX_PER_PARENT       Maximum apps per parent (default: 10)

EOF
}

# Start orchestrator
start_orchestrator() {
    echo -e "${CYAN}🚀 Starting Vrooli Orchestrator...${NC}"
    
    # Start orchestrator in background
    "$ORCHESTRATOR_SCRIPT" start >/dev/null 2>&1 &
    local daemon_pid=$!
    
    # Wait a moment for daemon to start
    sleep 3
    
    # Check if daemon started successfully
    if [[ -f "$ORCHESTRATOR_HOME/orchestrator.pid" ]]; then
        echo -e "${GREEN}✅ Orchestrator started successfully${NC}"
        
        # Show initial status
        echo ""
        show_status
    else
        echo -e "${RED}❌ Failed to start orchestrator${NC}" >&2
        return 1
    fi
}

# Stop orchestrator
stop_orchestrator() {
    echo -e "${CYAN}🛑 Stopping Vrooli Orchestrator...${NC}"
    
    if "$ORCHESTRATOR_SCRIPT" stop; then
        echo -e "${GREEN}✅ Orchestrator stopped successfully${NC}"
    else
        echo -e "${RED}❌ Failed to stop orchestrator${NC}" >&2
        return 1
    fi
}

# Restart orchestrator
restart_orchestrator() {
    echo -e "${CYAN}🔄 Restarting Vrooli Orchestrator...${NC}"
    
    if "$ORCHESTRATOR_SCRIPT" restart; then
        echo -e "${GREEN}✅ Orchestrator restarted successfully${NC}"
        
        # Show status after restart
        sleep 2
        echo ""
        show_status
    else
        echo -e "${RED}❌ Failed to restart orchestrator${NC}" >&2
        return 1
    fi
}

# Show status
show_status() {
    local format="${1:-text}"
    
    if [[ "$format" == "json" ]]; then
        "$ORCHESTRATOR_SCRIPT" status --json
    else
        "$ORCHESTRATOR_SCRIPT" status
    fi
}

# Show process tree
show_tree() {
    "$ORCHESTRATOR_SCRIPT" tree
}

# List processes with details
list_processes() {
    local format="${1:-text}"
    
    echo -e "${CYAN}📋 Process List${NC}"
    echo ""
    
    if [[ "$format" == "json" ]]; then
        "$ORCHESTRATOR_SCRIPT" status --json | jq '.processes'
    else
        # Enhanced listing with additional details
        local registry_file="$ORCHESTRATOR_HOME/processes.json"
        
        if [[ ! -f "$registry_file" ]]; then
            echo "No processes registered"
            return 0
        fi
        
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        printf "%-40s %-12s %-8s %-10s %-8s %-12s %-25s %s\n" \
            "NAME" "STATE" "PID" "RESTARTS" "UPTIME" "MEMORY" "URL" "COMMAND"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        jq -r '.processes | to_entries | .[] | 
            [.value.name, .value.state, (.value.pid // "-"), 
             .value.restart_count, .value.started_at, .value.command,
             (.value.env_vars // "{}" | fromjson.PORT // ""),
             (.value.metadata // "{}" | fromjson.port // "")] | 
            @tsv' "$registry_file" | while IFS=$'\t' read -r name state pid restarts started command env_port meta_port; do
            
            # Get port from either env vars or metadata
            local port=""
            [[ -n "$env_port" ]] && port="$env_port"
            [[ -z "$port" && -n "$meta_port" ]] && port="$meta_port"
            
            # Color code by state
            case "$state" in
                "running") state_display="${GREEN}●${NC} $state" ;;
                "stopped") state_display="${YELLOW}○${NC} $state" ;;
                "failed"|"crashed") state_display="${RED}✗${NC} $state" ;;
                *) state_display="◌ $state" ;;
            esac
            
            # Calculate uptime
            local uptime="-"
            if [[ "$started" != "null" && "$started" != "-" ]]; then
                if command -v python3 >/dev/null 2>&1; then
                    uptime=$(python3 -c "
import datetime
import sys
try:
    start = datetime.datetime.fromisoformat('$started'.replace('Z', '+00:00'))
    now = datetime.datetime.now(datetime.timezone.utc)
    delta = now - start
    days = delta.days
    hours, remainder = divmod(delta.seconds, 3600)
    minutes, _ = divmod(remainder, 60)
    if days > 0:
        print(f'{days}d{hours}h')
    elif hours > 0:
        print(f'{hours}h{minutes}m')
    else:
        print(f'{minutes}m')
except:
    print('-')
")
                fi
            fi
            
            # Get memory usage if PID available
            local memory="-"
            if [[ "$pid" != "-" ]] && kill -0 "$pid" 2>/dev/null; then
                if command -v ps >/dev/null 2>&1; then
                    memory=$(ps -p "$pid" -o rss= 2>/dev/null | awk '{print int($1/1024)"M"}' || echo "-")
                fi
            fi
            
            # Show URL if port is available and app is running
            local url_display="-"
            if [[ -n "$port" && "$state" == "running" ]]; then
                url_display="http://localhost:$port"
            fi
            
            # Truncate command
            local short_command=$(echo "$command" | cut -c1-20)
            [[ ${#command} -gt 20 ]] && short_command="${short_command}..."
            
            # Use echo -e to properly handle color codes
            if [[ "$url_display" != "-" ]]; then
                printf "%-40s " "$name"
                echo -en "$state_display"
                printf " %-8s %-10s %-8s %-12s " "$pid" "$restarts" "$uptime" "$memory"
                echo -en "${BLUE}"
                printf "%-25s" "$url_display"
                echo -en "${NC}"
                printf " %s\n" "$short_command"
            else
                printf "%-40s " "$name"
                echo -en "$state_display"
                printf " %-8s %-10s %-8s %-12s %-25s %s\n" "$pid" "$restarts" "$uptime" "$memory" "$url_display" "$short_command"
            fi
        done
        
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    fi
}

# Show logs for a process
show_logs() {
    local process_name="$1"
    local lines="${2:-50}"
    local follow="${3:-false}"
    
    # Source client library to use log functions
    source "$CLIENT_SCRIPT"
    
    if [[ "$follow" == "true" ]]; then
        orchestrator::follow_logs "$process_name"
    else
        orchestrator::logs "$process_name" "$lines"
    fi
}

# Real-time monitoring dashboard
monitor_dashboard() {
    echo -e "${CYAN}🖥️  Starting real-time monitor (Ctrl+C to exit)${NC}"
    echo ""
    
    # Clear screen and hide cursor
    clear
    tput civis
    
    # Restore cursor on exit
    trap 'tput cnorm; exit' INT TERM
    
    while true; do
        # Move cursor to top
        tput cup 0 0
        
        # Show header
        echo -e "${CYAN}=== Vrooli Orchestrator Monitor ===${NC} $(date '+%Y-%m-%d %H:%M:%S')"
        echo ""
        
        # Show status
        show_status text
        
        # Show resource usage
        echo ""
        echo -e "${CYAN}=== System Resources ===${NC}"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        # Memory usage
        if command -v free >/dev/null 2>&1; then
            local mem_info=$(free -h | grep '^Mem:')
            local mem_used=$(echo "$mem_info" | awk '{print $3}')
            local mem_total=$(echo "$mem_info" | awk '{print $2}')
            echo "Memory: $mem_used / $mem_total"
        fi
        
        # Load average
        if [[ -f /proc/loadavg ]]; then
            local load=$(cat /proc/loadavg | cut -d' ' -f1-3)
            echo "Load Average: $load"
        fi
        
        # Disk usage for orchestrator home
        if command -v du >/dev/null 2>&1; then
            local disk_usage=$(du -sh "$ORCHESTRATOR_HOME" 2>/dev/null | cut -f1)
            echo "Orchestrator Storage: $disk_usage"
        fi
        
        echo ""
        echo -e "${YELLOW}Press Ctrl+C to exit${NC}"
        
        # Wait 5 seconds
        sleep 5
    done
}

# Health check all processes
health_check() {
    echo -e "${CYAN}🏥 Health Check${NC}"
    echo ""
    
    local registry_file="$ORCHESTRATOR_HOME/processes.json"
    
    if [[ ! -f "$registry_file" ]]; then
        echo "No processes to check"
        return 0
    fi
    
    local healthy=0
    local unhealthy=0
    local unknown=0
    
    jq -r '.processes | to_entries | .[] | 
        [.value.name, .value.state, (.value.pid // ""), 
         (.value.metadata.health_check // "")] | @tsv' "$registry_file" | \
    while IFS=$'\t' read -r name state pid health_cmd; do
        
        printf "%-40s " "$name"
        
        if [[ "$state" != "running" ]]; then
            echo -e "${YELLOW}⚠️  Not running${NC}"
            continue
        fi
        
        if [[ -z "$health_cmd" ]]; then
            echo -e "${BLUE}ℹ️  No health check${NC}"
            continue
        fi
        
        # Execute health check
        if eval "$health_cmd" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ Healthy${NC}"
            healthy=$((healthy + 1))
        else
            echo -e "${RED}❌ Unhealthy${NC}"
            unhealthy=$((unhealthy + 1))
        fi
    done
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Health Summary: ${GREEN}${healthy} healthy${NC} | ${RED}${unhealthy} unhealthy${NC} | ${BLUE}${unknown} unknown${NC}"
}

# Clean up orphaned processes
cleanup_orphans() {
    echo -e "${CYAN}🧹 Cleaning up orphaned processes${NC}"
    echo ""
    
    local registry_file="$ORCHESTRATOR_HOME/processes.json"
    
    if [[ ! -f "$registry_file" ]]; then
        echo "No process registry found"
        return 0
    fi
    
    local cleaned=0
    local temp_registry=$(mktemp)
    
    # Copy original registry
    cp "$registry_file" "$temp_registry"
    
    # Check each process
    jq -r '.processes | to_entries | .[] | 
        [.key, .value.state, (.value.pid // "")] | @tsv' "$registry_file" | \
    while IFS=$'\t' read -r name state pid; do
        
        if [[ "$state" == "running" ]] && [[ -n "$pid" ]]; then
            if ! kill -0 "$pid" 2>/dev/null; then
                echo "🗑️  Cleaning up orphaned process: $name (PID $pid)"
                
                # Update registry to mark as crashed
                jq --arg name "$name" \
                   --arg stopped_at "$(date -Iseconds)" \
                   '.processes[$name].state = "crashed" | 
                    .processes[$name].pid = null |
                    .processes[$name].stopped_at = $stopped_at' \
                   "$temp_registry" > "${temp_registry}.tmp"
                
                mv "${temp_registry}.tmp" "$temp_registry"
                cleaned=$((cleaned + 1))
            fi
        fi
    done
    
    # Save cleaned registry
    if [[ $cleaned -gt 0 ]]; then
        cp "$temp_registry" "$registry_file"
        echo -e "${GREEN}✅ Cleaned up $cleaned orphaned processes${NC}"
    else
        echo "✨ No orphaned processes found"
    fi
    
    rm -f "$temp_registry" "${temp_registry}.tmp"
}

# Backup process registry
backup_registry() {
    local backup_dir="$ORCHESTRATOR_HOME/backups"
    local backup_name="registry-$(date '+%Y%m%d-%H%M%S').json"
    local backup_path="$backup_dir/$backup_name"
    
    echo -e "${CYAN}💾 Creating backup${NC}"
    
    mkdir -p "$backup_dir"
    
    if [[ -f "$ORCHESTRATOR_HOME/processes.json" ]]; then
        cp "$ORCHESTRATOR_HOME/processes.json" "$backup_path"
        echo -e "${GREEN}✅ Backup created: $backup_path${NC}"
        
        # Keep only last 10 backups
        local backup_count=$(ls -1 "$backup_dir"/registry-*.json 2>/dev/null | wc -l)
        if [[ $backup_count -gt 10 ]]; then
            local to_remove=$((backup_count - 10))
            ls -1t "$backup_dir"/registry-*.json | tail -n "$to_remove" | xargs rm -f
            echo "🗑️  Removed $to_remove old backups"
        fi
    else
        echo -e "${YELLOW}⚠️  No registry file to backup${NC}"
    fi
}

# Restore process registry
restore_registry() {
    local backup_file="$1"
    
    if [[ ! -f "$backup_file" ]]; then
        echo -e "${RED}❌ Backup file not found: $backup_file${NC}" >&2
        return 1
    fi
    
    echo -e "${CYAN}📁 Restoring from backup${NC}"
    echo "Source: $backup_file"
    echo "Target: $ORCHESTRATOR_HOME/processes.json"
    
    # Validate backup file
    if ! jq '.' "$backup_file" >/dev/null 2>&1; then
        echo -e "${RED}❌ Invalid backup file (not valid JSON)${NC}" >&2
        return 1
    fi
    
    # Create backup of current registry
    if [[ -f "$ORCHESTRATOR_HOME/processes.json" ]]; then
        backup_registry
    fi
    
    # Restore
    cp "$backup_file" "$ORCHESTRATOR_HOME/processes.json"
    echo -e "${GREEN}✅ Registry restored successfully${NC}"
}

# Show configuration
show_config() {
    echo -e "${CYAN}⚙️  Orchestrator Configuration${NC}"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Home Directory: $ORCHESTRATOR_HOME"
    echo "Max Total Apps: ${VROOLI_MAX_APPS:-20}"
    echo "Max Nesting Depth: ${VROOLI_MAX_DEPTH:-5}" 
    echo "Max Per Parent: ${VROOLI_MAX_PER_PARENT:-10}"
    echo "Client Timeout: ${VROOLI_CLIENT_TIMEOUT:-10}s"
    echo ""
    
    # File locations
    echo "Files:"
    echo "  Registry: $ORCHESTRATOR_HOME/processes.json"
    echo "  PID File: $ORCHESTRATOR_HOME/orchestrator.pid"
    echo "  Command FIFO: $ORCHESTRATOR_HOME/commands.fifo"
    echo "  Logs: $ORCHESTRATOR_HOME/orchestrator.log"
    echo ""
    
    # Status
    echo "Status:"
    if [[ -f "$ORCHESTRATOR_HOME/orchestrator.pid" ]]; then
        local pid=$(cat "$ORCHESTRATOR_HOME/orchestrator.pid")
        if kill -0 "$pid" 2>/dev/null; then
            echo -e "  Daemon: ${GREEN}Running (PID $pid)${NC}"
        else
            echo -e "  Daemon: ${RED}Not running (stale PID file)${NC}"
        fi
    else
        echo -e "  Daemon: ${YELLOW}Not running${NC}"
    fi
    
    # Registry info
    if [[ -f "$ORCHESTRATOR_HOME/processes.json" ]]; then
        local total=$(jq '.processes | length' "$ORCHESTRATOR_HOME/processes.json")
        local running=$(jq '[.processes[] | select(.state == "running")] | length' "$ORCHESTRATOR_HOME/processes.json")
        echo "  Processes: $total registered, $running running"
    else
        echo "  Processes: No registry file"
    fi
}

# Main command handler
main() {
    # Check dependencies first
    if ! check_dependencies; then
        exit 1
    fi
    
    local command="${1:-}"
    local verbose=false
    local json_output=false
    local follow_mode=false
    
    # Parse global options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose|-v)
                verbose=true
                shift
                ;;
            --json)
                json_output=true
                shift
                ;;
            --follow)
                follow_mode=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                break
                ;;
        esac
    done
    
    command="${1:-}"
    shift || true
    
    case "$command" in
        start)
            start_orchestrator "$@"
            ;;
            
        stop)
            stop_orchestrator "$@"
            ;;
            
        restart)
            restart_orchestrator "$@"
            ;;
            
        status)
            local format="text"
            [[ "$json_output" == "true" ]] && format="json"
            show_status "$format" "$@"
            ;;
            
        tree)
            show_tree "$@"
            ;;
            
        list)
            local format="text"
            [[ "$json_output" == "true" ]] && format="json"
            list_processes "$format" "$@"
            ;;
            
        logs)
            local process_name="${1:-}"
            local lines="${2:-50}"
            
            if [[ -z "$process_name" ]]; then
                echo "Usage: orchestrator-ctl logs <process-name> [lines]" >&2
                exit 1
            fi
            
            show_logs "$process_name" "$lines" "$follow_mode"
            ;;
            
        follow)
            local process_name="${1:-}"
            
            if [[ -z "$process_name" ]]; then
                echo "Usage: orchestrator-ctl follow <process-name>" >&2
                exit 1
            fi
            
            show_logs "$process_name" 50 true
            ;;
            
        monitor)
            monitor_dashboard "$@"
            ;;
            
        health)
            health_check "$@"
            ;;
            
        clean)
            cleanup_orphans "$@"
            ;;
            
        backup)
            backup_registry "$@"
            ;;
            
        restore)
            local backup_file="${1:-}"
            
            if [[ -z "$backup_file" ]]; then
                echo "Usage: orchestrator-ctl restore <backup-file>" >&2
                exit 1
            fi
            
            restore_registry "$backup_file"
            ;;
            
        config)
            show_config "$@"
            ;;
            
        *)
            if [[ -z "$command" ]]; then
                show_help
            else
                echo -e "${RED}❌ Unknown command: $command${NC}" >&2
                echo ""
                echo "Run 'orchestrator-ctl --help' for usage information"
                exit 1
            fi
            ;;
    esac
}

# Execute main function
main "$@"