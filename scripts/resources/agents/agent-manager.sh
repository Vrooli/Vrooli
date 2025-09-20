#!/usr/bin/env bash
################################################################################
# Generic Agent Manager
# 
# Unified agent management for all resources using plugin configuration
# Eliminates 12,000+ lines of duplicate code across 18 resources
################################################################################

# Source common utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/../../../" && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

# Source framework modules
source "${APP_ROOT}/scripts/resources/agents/core/registry.sh"
source "${APP_ROOT}/scripts/resources/agents/core/lifecycle.sh"
source "${APP_ROOT}/scripts/resources/agents/monitoring/logs.sh"
source "${APP_ROOT}/scripts/resources/agents/monitoring/health.sh"
source "${APP_ROOT}/scripts/resources/agents/monitoring/metrics.sh"
source "${APP_ROOT}/scripts/resources/agents/monitoring/dashboard.sh"
source "${APP_ROOT}/scripts/resources/agents/lib/wrapper.sh"

#######################################
# Load resource configuration
# Arguments:
#   $1 - Resource name
# Returns:
#   0 on success, 1 on error
# Sets global variables from config
#######################################
agent_manager::load_config() {
    local resource="$1"
    
    if [[ -z "$resource" ]]; then
        log::error "agent_manager::load_config: Resource name required"
        return 1
    fi
    
    local config_file="${APP_ROOT}/resources/${resource}/config/agents.conf"
    if [[ ! -f "$config_file" ]]; then
        log::error "Configuration file not found: $config_file"
        return 1
    fi
    
    # Source the configuration
    source "$config_file" || {
        log::error "Failed to load configuration: $config_file"
        return 1
    }
    
    # Validate required variables
    local required_vars=(
        "RESOURCE_NAME" 
        "REGISTRY_FILE" 
        "LOG_DIR" 
        "SEARCH_PATTERNS"
        "AGENT_ID_PREFIX"
    )
    
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var}" ]]; then
            log::error "Missing required configuration: $var in $config_file"
            return 1
        fi
    done
    
    # Set defaults for optional monitoring configuration
    HEALTH_CHECK_ENABLED="${HEALTH_CHECK_ENABLED:-false}"
    HEALTH_CHECK_INTERVAL="${HEALTH_CHECK_INTERVAL:-30}"
    METRICS_ENABLED="${METRICS_ENABLED:-true}"
    MONITOR_PIDFILE="${MONITOR_PIDFILE:-${APP_ROOT}/.vrooli/${resource}.monitor.pid}"
    
    return 0
}

#######################################
# Initialize agent registry
# Returns:
#   0 on success, 1 on error
#######################################
agent_manager::init_registry() {
    local registry_dir
    registry_dir=$(dirname "$REGISTRY_FILE")
    
    # Create .vrooli directory if needed
    if [[ ! -d "$registry_dir" ]]; then
        mkdir -p "$registry_dir" || return 1
    fi
    
    # Create empty registry if it doesn't exist
    # Use pretty-printed JSON format for consistency
    if [[ ! -f "$REGISTRY_FILE" ]]; then
        cat > "$REGISTRY_FILE" <<-'EOF'
{
  "agents": {}
}
EOF
        if [[ $? -ne 0 ]]; then
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Generate unique agent ID
# Returns:
#   Agent ID string
#######################################
agent_manager::generate_id() {
    echo "${AGENT_ID_PREFIX}-$(date +%s)-$$"
}

#######################################
# Register new agent
# Arguments:
#   $1 - Agent ID
#   $2 - Process ID 
#   $3 - Command string
# Returns:
#   0 on success, 1 on error
#######################################
agent_manager::register() {
    # Prune any dead agents before registering a new one
    agent_manager::cleanup >/dev/null || true

    # Register the agent
    agents::registry::register "$REGISTRY_FILE" "$RESOURCE_NAME" "$1" "$2" "$3" || return 1
    
    # Initialize metrics if enabled
    if [[ "$METRICS_ENABLED" == "true" ]]; then
        agents::metrics::init "$REGISTRY_FILE" "$1"
    fi
    
    # Start health monitor if enabled and not already running
    if [[ "$HEALTH_CHECK_ENABLED" == "true" ]]; then
        agents::health::start_monitor "$REGISTRY_FILE" "$RESOURCE_NAME" "$HEALTH_CHECK_INTERVAL" "$MONITOR_PIDFILE"
    fi
    
    return 0
}

#######################################
# Unregister agent
# Arguments:
#   $1 - Agent ID
# Returns:
#   0 on success, 1 on error
#######################################
agent_manager::unregister() {
    if [[ -z "$1" ]]; then
        log::error "agent_manager::unregister: Agent ID required"
        return 1
    fi
    
    [[ -f "$REGISTRY_FILE" ]] || return 0
    
    local lock_fd
    exec {lock_fd}>"${REGISTRY_FILE}.lock" || return 1
    if ! flock -w 5 "$lock_fd"; then
        exec {lock_fd}>&-
        log::error "Failed to lock $RESOURCE_NAME agent registry"
        return 1
    fi

    local temp_file
    local rc=0
    temp_file=$(agents::registry::create_temp_file "$REGISTRY_FILE") || rc=1

    if [[ $rc -eq 0 ]]; then
        if ! jq --arg id "$1" 'del(.agents[$id])' "$REGISTRY_FILE" > "$temp_file"; then
            log::error "Failed to update $RESOURCE_NAME agent registry"
            rc=1
        fi
    fi

    if [[ $rc -eq 0 ]]; then
        if ! mv "$temp_file" "$REGISTRY_FILE"; then
            log::error "Failed to save $RESOURCE_NAME agent registry"
            rc=1
        fi
    fi

    [[ -n "$temp_file" && -f "$temp_file" ]] && rm -f "$temp_file"

    flock -u "$lock_fd"
    exec {lock_fd}>&-

    if [[ $rc -eq 0 ]]; then
        log::debug "Unregistered $RESOURCE_NAME agent: $1"

        # Stop monitor when no agents remain
        if jq -e '.agents | length == 0' "$REGISTRY_FILE" >/dev/null 2>&1; then
            agents::health::stop_monitor "$MONITOR_PIDFILE" >/dev/null 2>&1 || true
        fi
    fi

    return $rc
}

#######################################
# Update agent heartbeat
# Arguments:
#   $1 - Agent ID
# Returns:
#   0 on success, 1 on error
#######################################
agent_manager::heartbeat() {
    if [[ -z "$1" ]]; then
        log::error "agent_manager::heartbeat: Agent ID required"
        return 1
    fi
    
    [[ -f "$REGISTRY_FILE" ]] || return 1
    
    local current_time
    current_time=$(date -Iseconds)
    
    local lock_fd
    exec {lock_fd}>"${REGISTRY_FILE}.lock" || return 1
    if ! flock -w 5 "$lock_fd"; then
        exec {lock_fd}>&-
        return 1
    fi

    local temp_file
    local rc=0
    temp_file=$(agents::registry::create_temp_file "$REGISTRY_FILE") || rc=1

    if [[ $rc -eq 0 ]]; then
        if ! jq --arg id "$1" \
                --arg time "$current_time" \
                '.agents[$id].last_seen = $time' "$REGISTRY_FILE" > "$temp_file"; then
            log::error "Failed to update $RESOURCE_NAME agent registry"
            rc=1
        fi
    fi

    if [[ $rc -eq 0 ]]; then
        if ! mv "$temp_file" "$REGISTRY_FILE"; then
            log::error "Failed to save $RESOURCE_NAME agent registry"
            rc=1
        fi
    fi

    [[ -n "$temp_file" && -f "$temp_file" ]] && rm -f "$temp_file"

    flock -u "$lock_fd"
    exec {lock_fd}>&-

    return $rc
}

#######################################
# Check if process is running
# Arguments:
#   $1 - Process ID
# Returns:
#   0 if running, 1 if not
#######################################
agent_manager::is_pid_running() {
    agents::lifecycle::is_pid_running "$1"
}

#######################################
# Clean up dead agents
# Returns:
#   Number of cleaned agents
#######################################
agent_manager::cleanup() {
    local cleaned=0

    [[ -f "$REGISTRY_FILE" ]] || return 0

    local lock_fd
    exec {lock_fd}>"${REGISTRY_FILE}.lock" || return 0
    if ! flock -w 5 "$lock_fd"; then
        exec {lock_fd}>&-
        return 0
    fi

    local temp_file
    temp_file=$(agents::registry::create_temp_file "$REGISTRY_FILE") || {
        flock -u "$lock_fd"
        exec {lock_fd}>&-
        return 0
    }

    local agent_ids
    agent_ids=$(jq -r '.agents | keys[]' "$REGISTRY_FILE" 2>/dev/null || echo "")

    if [[ -z "$agent_ids" ]]; then
        rm -f "$temp_file"
        flock -u "$lock_fd"
        exec {lock_fd}>&-
        return 0
    fi

    cp "$REGISTRY_FILE" "$temp_file" || {
        rm -f "$temp_file"
        flock -u "$lock_fd"
        exec {lock_fd}>&-
        return 0
    }

    while IFS= read -r agent_id; do
        [[ -n "$agent_id" ]] || continue

        local pid
        pid=$(jq -r --arg id "$agent_id" '.agents[$id].pid // empty' "$temp_file" 2>/dev/null)

        if [[ -n "$pid" ]] && ! agent_manager::is_pid_running "$pid"; then
            log::debug "Cleaning up dead $RESOURCE_NAME agent: $agent_id (PID: $pid)"
            if jq --arg id "$agent_id" 'del(.agents[$id])' "$temp_file" > "${temp_file}.new"; then
                mv "${temp_file}.new" "$temp_file"
                ((cleaned++))
            else
                rm -f "${temp_file}.new"
            fi
        fi
    done <<< "$agent_ids"

    if [[ $cleaned -gt 0 ]]; then
        mv "$temp_file" "$REGISTRY_FILE" || rm -f "$temp_file"
    else
        rm -f "$temp_file"
    fi

    flock -u "$lock_fd"
    exec {lock_fd}>&-

    if [[ $cleaned -gt 0 ]]; then
        if jq -e '.agents | length == 0' "$REGISTRY_FILE" >/dev/null 2>&1; then
            agents::health::stop_monitor "$MONITOR_PIDFILE" >/dev/null 2>&1 || true
        fi
    fi

    return $cleaned
}

#######################################
# List all agents
# Arguments:
#   $1 - Optional status filter
#   $2 - Output format (json|table)
# Returns:
#   0 on success
#######################################
agent_manager::list() {
    local status_filter="${1:-}"
    local output_format="${2:-table}"
    
    [[ -f "$REGISTRY_FILE" ]] || {
        if [[ "$output_format" == "json" ]]; then
            echo '{"agents": {}}'
        else
            echo "No agents registered"
        fi
        return 0
    }
    
    # Clean up dead agents first
    agent_manager::cleanup >/dev/null
    
    if [[ "$output_format" == "json" ]]; then
        if [[ -n "$status_filter" ]]; then
            jq --arg status "$status_filter" '{
                agents: .agents | to_entries | map(select(.value.status == $status)) | from_entries
            }' "$REGISTRY_FILE"
        else
            cat "$REGISTRY_FILE"
        fi
    else
        # Table format
        local agents_data
        if [[ -n "$status_filter" ]]; then
            agents_data=$(jq -r --arg status "$status_filter" \
                '.agents | to_entries | map(select(.value.status == $status)) | .[] | 
                [.value.id, .value.pid, .value.status, .value.start_time] | @tsv' \
                "$REGISTRY_FILE" 2>/dev/null || echo "")
        else
            agents_data=$(jq -r '.agents | to_entries | .[] | 
                [.value.id, .value.pid, .value.status, .value.start_time] | @tsv' \
                "$REGISTRY_FILE" 2>/dev/null || echo "")
        fi
        
        if [[ -n "$agents_data" ]]; then
            echo "AGENT_ID                            PID     STATUS   START_TIME"
            echo "────────────────────────────────────────────────────────────────────"
            echo "$agents_data" | column -t -s $'\t'
        else
            echo "No agents found"
            if [[ -n "$status_filter" ]]; then
                echo "(filtered by status: $status_filter)"
            fi
        fi
    fi
}

#######################################
# Stop agents
# Arguments:
#   $1 - Agent ID or "all"
#   $2 - Force flag (true/false)
# Returns:
#   0 on success, 1 on error
#######################################
agent_manager::stop() {
    local target="$1"
    local force="${2:-false}"
    local signal="TERM"
    
    if [[ -z "$target" ]]; then
        log::error "agent_manager::stop: Target required (agent-id or 'all')"
        return 1
    fi
    
    [[ -f "$REGISTRY_FILE" ]] || {
        log::warn "No agents registry found"
        return 0
    }
    
    # Use KILL signal if force is true
    if [[ "$force" == "true" ]]; then
        signal="KILL"
    fi
    
    if [[ "$target" == "all" ]]; then
        # Stop all agents
        local agent_ids stopped=0
        agent_ids=$(jq -r '.agents | keys[]' "$REGISTRY_FILE" 2>/dev/null || echo "")
        
        while IFS= read -r agent_id; do
            [[ -n "$agent_id" ]] || continue
            
            local pid
            pid=$(jq -r --arg id "$agent_id" '.agents[$id].pid // empty' "$REGISTRY_FILE" 2>/dev/null)
            
            if [[ -n "$pid" ]] && agent_manager::is_pid_running "$pid"; then
                log::info "Stopping agent: $agent_id (PID: $pid)"
                if kill -s "$signal" "$pid" 2>/dev/null; then
                    ((stopped++))
                    agent_manager::unregister "$agent_id"
                else
                    log::warn "Failed to stop agent: $agent_id"
                fi
            else
                # Remove dead agent from registry
                agent_manager::unregister "$agent_id"
            fi
        done <<< "$agent_ids"
        
        log::info "Stopped $stopped agents"
        return 0
    else
        # Stop specific agent
        local pid
        pid=$(jq -r --arg id "$target" '.agents[$id].pid // empty' "$REGISTRY_FILE" 2>/dev/null)
        
        if [[ -z "$pid" ]]; then
            log::error "Agent not found: $target"
            return 1
        fi
        
        if agent_manager::is_pid_running "$pid"; then
            log::info "Stopping agent: $target (PID: $pid)"
            if kill -s "$signal" "$pid" 2>/dev/null; then
                agent_manager::unregister "$target"
                log::info "Agent stopped successfully"
                return 0
            else
                log::error "Failed to stop agent: $target"
                return 1
            fi
        else
            log::warn "Agent process not running, cleaning up registry"
            agent_manager::unregister "$target"
            return 0
        fi
    fi
}

#######################################
# Get agent info
# Arguments:
#   $1 - Agent ID
#   $2 - Output format (json|table)
# Returns:
#   0 on success, 1 if not found
#######################################
agent_manager::info() {
    local agent_id="$1"
    local output_format="${2:-table}"
    
    if [[ -z "$agent_id" ]]; then
        log::error "agent_manager::info: Agent ID required"
        return 1
    fi
    
    [[ -f "$REGISTRY_FILE" ]] || {
        log::error "No agents registry found"
        return 1
    }
    
    local agent_data
    agent_data=$(jq --arg id "$agent_id" '.agents[$id] // empty' "$REGISTRY_FILE" 2>/dev/null)
    
    if [[ -z "$agent_data" || "$agent_data" == "null" ]]; then
        if [[ "$output_format" == "json" ]]; then
            echo '{"error": "Agent not found"}'
        else
            echo "Agent not found: $agent_id"
        fi
        return 1
    fi
    
    if [[ "$output_format" == "json" ]]; then
        echo "$agent_data"
    else
        echo "Agent Information:"
        echo "════════════════════"
        echo "$agent_data" | jq -r '
            "ID:           " + .id + "\n" +
            "PID:          " + (.pid | tostring) + "\n" + 
            "Status:       " + .status + "\n" +
            "Start Time:   " + .start_time + "\n" +
            "Last Seen:    " + .last_seen + "\n" +
            "Command:      " + .command
        '
    fi
    
    return 0
}

#######################################
# Get agent logs using framework
# Arguments:
#   $1 - Agent ID
#   $2 - Follow mode (true/false)
#   $3 - Number of lines to show
#   $4 - JSON output (true/false)
# Returns:
#   0 on success, 1 on error
#######################################
agent_manager::logs() {
    local agent_id="$1"
    local follow="$2"
    local lines="$3"
    local json_output="$4"
    
    # Delegate to the shared logs implementation with explicit parameters
    agents::logs::get "$REGISTRY_FILE" "$RESOURCE_NAME" "$SEARCH_PATTERNS" "$agent_id" "$follow" "$lines" "$json_output"
}

#######################################
# Main CLI command handler
# Arguments:
#   $@ - All command arguments
# Returns:
#   Exit code from subcommand
#######################################
main() {
    local resource=""
    local subcommand=""
    
    # Parse global options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --config=*)
                resource="${1#*=}"
                shift
                ;;
            --config)
                resource="$2"
                shift 2
                ;;
            *)
                subcommand="$1"
                shift
                break
                ;;
        esac
    done
    
    if [[ -z "$resource" ]]; then
        log::error "Usage: agent-manager.sh --config=<resource> <subcommand> [options]"
        log::error "Example: agent-manager.sh --config=ollama list"
        return 1
    fi
    
    # Load resource configuration
    if ! agent_manager::load_config "$resource"; then
        return 1
    fi
    
    # Default to list if no subcommand
    subcommand="${subcommand:-list}"
    
    case "$subcommand" in
        "list"|"ls")
            local status_filter=""
            local json_output="false"
            
            # Parse arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --json)
                        json_output="true"
                        shift
                        ;;
                    --status)
                        status_filter="$2"
                        shift 2
                        ;;
                    *)
                        log::error "Unknown option: $1"
                        return 1
                        ;;
                esac
            done
            
            if [[ "$json_output" == "true" ]]; then
                agent_manager::list "$status_filter" "json"
            else
                agent_manager::list "$status_filter" "table"
            fi
            ;;
            
        "stop")
            local target="$1"
            local force="false"
            shift || true
            
            if [[ -z "$target" ]]; then
                log::error "Usage: agents stop <agent-id|all> [--force]"
                return 1
            fi
            
            # Parse remaining arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --force)
                        force="true"
                        shift
                        ;;
                    *)
                        log::error "Unknown option: $1"
                        return 1
                        ;;
                esac
            done
            
            agent_manager::stop "$target" "$force"
            ;;
            
        "cleanup")
            local cleaned
            cleaned=$(agent_manager::cleanup)
            if [[ $cleaned -gt 0 ]]; then
                log::info "Cleaned up $cleaned dead agents"
            else
                log::info "No dead agents to clean up"
            fi
            ;;
            
        "info")
            local agent_id="$1"
            local json_output="false"
            shift || true
            
            if [[ -z "$agent_id" ]]; then
                log::error "Usage: agents info <agent-id> [--json]"
                return 1
            fi
            
            # Parse remaining arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --json)
                        json_output="true"
                        shift
                        ;;
                    *)
                        log::error "Unknown option: $1"
                        return 1
                        ;;
                esac
            done
            
            if [[ "$json_output" == "true" ]]; then
                agent_manager::info "$agent_id" "json"
            else
                agent_manager::info "$agent_id" "table"
            fi
            ;;
            
        "logs")
            local agent_id="$1"
            local follow="false"
            local lines="100"
            local json_output="false"
            shift || true
            
            if [[ -z "$agent_id" ]]; then
                log::error "Usage: agents logs <agent-id> [--follow|-f] [--lines|-n N] [--json]"
                return 1
            fi
            
            # Parse remaining arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    -f|--follow)
                        follow="true"
                        shift
                        ;;
                    -n|--lines)
                        if [[ -z "$2" || ! "$2" =~ ^[0-9]+$ ]]; then
                            log::error "Invalid lines value: $2"
                            return 1
                        fi
                        lines="$2"
                        shift 2
                        ;;
                    --json)
                        json_output="true"
                        shift
                        ;;
                    *)
                        log::error "Unknown option: $1"
                        return 1
                        ;;
                esac
            done
            
            agent_manager::logs "$agent_id" "$follow" "$lines" "$json_output"
            ;;
            
        "monitor")
            local refresh_interval="5"
            local output_format="terminal"
            local show_metrics="false"
            
            # Parse arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --interval)
                        if [[ -z "$2" || ! "$2" =~ ^[0-9]+$ ]]; then
                            log::error "Invalid interval value: $2"
                            return 1
                        fi
                        refresh_interval="$2"
                        shift 2
                        ;;
                    --json)
                        output_format="json"
                        shift
                        ;;
                    --metrics)
                        show_metrics="true"
                        shift
                        ;;
                    *)
                        log::error "Unknown option: $1"
                        return 1
                        ;;
                esac
            done
            
            # Display dashboard
            agents::dashboard::display "$REGISTRY_FILE" "$RESOURCE_NAME" "$refresh_interval" "$output_format"
            ;;
            
        "metrics")
            local agent_id="$1"
            local output_format="text"
            shift || true
            
            if [[ -z "$agent_id" ]]; then
                log::error "Usage: agents metrics <agent-id> [--json]"
                return 1
            fi
            
            # Parse arguments
            while [[ $# -gt 0 ]]; do
                case "$1" in
                    --json)
                        output_format="json"
                        shift
                        ;;
                    *)
                        log::error "Unknown option: $1"
                        return 1
                        ;;
                esac
            done
            
            # Show metrics for agent
            agents::metrics::get_summary "$REGISTRY_FILE" "$agent_id" "$output_format"
            ;;
            
        "help"|"-h"|"--help")
            echo "Usage: agent-manager.sh --config=<resource> <subcommand> [options]"
            echo ""
            echo "Subcommands:"
            echo "  list [--json] [--status STATUS]           List all agents (optionally filter by status)"
            echo "  stop <agent-id|all> [--force]             Stop specific agent or all agents"
            echo "  cleanup                                    Remove dead agents from registry"
            echo "  info <agent-id> [--json]                  Show detailed information about an agent"
            echo "  logs <agent-id> [-f] [-n N] [--json]      Show agent logs"
            echo "  monitor [--interval N] [--json]           Real-time monitoring dashboard"
            echo "  metrics <agent-id> [--json]               Show metrics for specific agent"
            echo "  help                                       Show this help message"
            echo ""
            echo "Logs options:"
            echo "  -f, --follow     Follow log output (like tail -f)"
            echo "  -n, --lines N    Show last N lines (default: 100)"
            echo "  --json           Output in JSON format"
            echo ""
            echo "Monitor options:"
            echo "  --interval N     Refresh interval in seconds (default: 5)"
            echo "  --json          Output as JSON instead of dashboard"
            echo ""
            echo "Status filters: running, stopped, crashed"
            ;;
            
        *)
            log::error "Unknown subcommand: $subcommand"
            log::error "Use 'agent-manager.sh --config=$resource help' for usage information"
            return 1
            ;;
    esac
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
