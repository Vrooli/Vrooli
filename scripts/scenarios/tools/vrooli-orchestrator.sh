#!/usr/bin/env bash
################################################################################
# Vrooli Orchestrator - Singleton Process Manager for All Apps
# 
# A daemon-like service that manages all Vrooli processes across any nesting
# level, providing centralized control, resource limits, and monitoring.
#
# Architecture:
# - Singleton daemon with PID file locking
# - JSON-based process registry
# - Socket-based IPC for commands
# - Hierarchical process namespacing
# - Resource protection and limits
#
# Usage:
#   vrooli-orchestrator start     # Start the daemon
#   vrooli-orchestrator stop      # Stop daemon and all processes
#   vrooli-orchestrator status    # Show daemon and process status
#   vrooli-orchestrator restart   # Restart the daemon
################################################################################

set -euo pipefail

# Orchestrator configuration
ORCHESTRATOR_HOME="${VROOLI_ORCHESTRATOR_HOME:-$HOME/.vrooli/orchestrator}"
PID_FILE="$ORCHESTRATOR_HOME/orchestrator.pid"
SOCKET_FILE="$ORCHESTRATOR_HOME/orchestrator.sock"
REGISTRY_FILE="$ORCHESTRATOR_HOME/processes.json"
LOG_FILE="$ORCHESTRATOR_HOME/orchestrator.log"
COMMAND_FIFO="$ORCHESTRATOR_HOME/commands.fifo"

# Resource limits
MAX_TOTAL_APPS="${VROOLI_MAX_APPS:-100}"
MAX_NESTING_DEPTH="${VROOLI_MAX_DEPTH:-5}"
MAX_PER_PARENT="${VROOLI_MAX_PER_PARENT:-10}"

# Process states
STATE_REGISTERED="registered"
STATE_STARTING="starting"
STATE_RUNNING="running"
STATE_STOPPING="stopping"
STATE_STOPPED="stopped"
STATE_FAILED="failed"
STATE_CRASHED="crashed"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Initialize orchestrator home
init_orchestrator_home() {
    mkdir -p "$ORCHESTRATOR_HOME"
    mkdir -p "$ORCHESTRATOR_HOME/logs"
    mkdir -p "$ORCHESTRATOR_HOME/sockets"
    
    # Initialize empty registry if it doesn't exist
    if [[ ! -f "$REGISTRY_FILE" ]]; then
        echo '{"version": "1.0.0", "processes": {}, "metadata": {"started_at": null, "total_started": 0}}' > "$REGISTRY_FILE"
    fi
}

# Logging functions
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
    
    # Also output to console if not daemonized
    if [[ -t 1 ]]; then
        case "$level" in
            ERROR) echo -e "${RED}[ERROR]${NC} $message" >&2 ;;
            WARN)  echo -e "${YELLOW}[WARN]${NC} $message" >&2 ;;
            INFO)  echo -e "${GREEN}[INFO]${NC} $message" ;;
            DEBUG) [[ "${DEBUG:-false}" == "true" ]] && echo -e "${CYAN}[DEBUG]${NC} $message" ;;
        esac
    fi
}

# Check if orchestrator is running
is_running() {
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            return 0
        else
            # Stale PID file
            rm -f "$PID_FILE"
        fi
    fi
    return 1
}

# Acquire singleton lock
acquire_lock() {
    # Use atomic file operations to prevent race conditions
    local temp_pid_file="${PID_FILE}.tmp.$$"
    
    # First check if already running
    if is_running; then
        local pid=$(cat "$PID_FILE")
        log "INFO" "Orchestrator already running with PID $pid"
        return 1
    fi
    
    # Check for recent start attempts (prevent rapid restarts)
    local lock_timestamp_file="${PID_FILE}.timestamp"
    if [[ -f "$lock_timestamp_file" ]]; then
        local last_start=$(cat "$lock_timestamp_file" 2>/dev/null || echo 0)
        local current_time=$(date +%s)
        local time_diff=$((current_time - last_start))
        
        if [[ $time_diff -lt 5 ]]; then
            log "ERROR" "Orchestrator was started too recently (${time_diff}s ago). Wait 5 seconds between starts."
            return 1
        fi
    fi
    
    # Try to acquire lock atomically using flock for better reliability
    local lock_dir="$(dirname "$PID_FILE")"
    local lock_file="$lock_dir/orchestrator.lock"
    
    # Create lock file if it doesn't exist
    touch "$lock_file"
    
    # Try to acquire exclusive lock
    if flock -n 200; then
        # Write PID
        echo $$ > "$PID_FILE"
        echo $(date +%s) > "$lock_timestamp_file"
        log "INFO" "Acquired lock with PID $$"
        return 0
    else
        log "ERROR" "Failed to acquire lock - another process holds it"
        return 1
    fi 200>"$lock_file"
}

# Release singleton lock
release_lock() {
    if [[ -f "$PID_FILE" ]]; then
        local pid=$(cat "$PID_FILE")
        if [[ "$pid" == "$$" ]]; then
            rm -f "$PID_FILE"
            rm -f "${PID_FILE}.timestamp"
            # Release flock by closing file descriptor 200
            exec 200>&-
            log "INFO" "Released lock for PID $$"
        fi
    fi
}

# Load process registry
load_registry() {
    if [[ -f "$REGISTRY_FILE" ]]; then
        cat "$REGISTRY_FILE"
    else
        echo '{"version": "1.0.0", "processes": {}, "metadata": {"started_at": null, "total_started": 0}}'
    fi
}

# Save process registry
save_registry() {
    local registry="$1"
    local temp_file="$REGISTRY_FILE.tmp"
    echo "$registry" | jq '.' > "$temp_file"
    mv "$temp_file" "$REGISTRY_FILE"
}

# Generate hierarchical process name
generate_process_name() {
    local parent="${1:-}"
    local name="$2"
    
    if [[ -z "$parent" ]]; then
        echo "vrooli.$name"
    else
        echo "$parent.$name"
    fi
}

# Validate process name
validate_process_name() {
    local name="$1"
    
    # Check format (alphanumeric, dots, hyphens, underscores)
    if ! [[ "$name" =~ ^vrooli\.[a-zA-Z0-9._-]+$ ]]; then
        log "ERROR" "Invalid process name format: $name"
        return 1
    fi
    
    # Check nesting depth
    local depth=$(echo "$name" | tr -cd '.' | wc -c)
    if [[ $depth -gt $MAX_NESTING_DEPTH ]]; then
        log "ERROR" "Process nesting too deep: $depth > $MAX_NESTING_DEPTH"
        return 1
    fi
    
    return 0
}

# Check resource limits
check_resource_limits() {
    local registry="$1"
    local parent="${2:-}"
    
    # Count only active processes (not crashed/stopped/failed)
    local total_count=$(echo "$registry" | jq --arg running "$STATE_RUNNING" --arg starting "$STATE_STARTING" --arg registered "$STATE_REGISTERED" '
        .processes | to_entries | 
        map(select(.value.state == $running or .value.state == $starting or .value.state == $registered)) | 
        length
    ')
    if [[ $total_count -ge $MAX_TOTAL_APPS ]]; then
        log "ERROR" "Maximum active apps reached: $total_count >= $MAX_TOTAL_APPS"
        return 1
    fi
    
    # Count active processes under parent
    if [[ -n "$parent" ]]; then
        local parent_count=$(echo "$registry" | jq --arg parent "$parent" --arg running "$STATE_RUNNING" --arg starting "$STATE_STARTING" --arg registered "$STATE_REGISTERED" '
            .processes | to_entries | 
            map(select(.key | startswith($parent + ".")) | select(.value.state == $running or .value.state == $starting or .value.state == $registered)) | 
            length
        ')
        if [[ $parent_count -ge $MAX_PER_PARENT ]]; then
            log "ERROR" "Maximum active apps per parent reached: $parent_count >= $MAX_PER_PARENT"
            return 1
        fi
    fi
    
    return 0
}

# Register a process
register_process() {
    local name="$1"
    local command="$2"
    local working_dir="${3:-$(pwd)}"
    local env_vars="${4:-{\}}"
    local metadata="${5:-{\}}"
    
    log "INFO" "Registering process: $name"
    
    # Validate name
    if ! validate_process_name "$name"; then
        return 1
    fi
    
    # Load registry
    local registry=$(load_registry)
    
    # Extract parent from name
    local parent=""
    if [[ "$name" =~ ^(.*)\.[^.]+$ ]]; then
        parent="${BASH_REMATCH[1]}"
    fi
    
    # Check resource limits
    if ! check_resource_limits "$registry" "$parent"; then
        return 1
    fi
    
    # Create process entry
    local process_entry=$(jq -n \
        --arg name "$name" \
        --arg command "$command" \
        --arg working_dir "$working_dir" \
        --arg state "$STATE_REGISTERED" \
        --argjson env_vars "$env_vars" \
        --argjson metadata "$metadata" \
        '{
            name: $name,
            command: $command,
            working_dir: $working_dir,
            state: $state,
            pid: null,
            started_at: null,
            stopped_at: null,
            restart_count: 0,
            env_vars: $env_vars,
            metadata: $metadata,
            parent: (if ($name | index(".")) then ($name | split(".")[:-1] | join(".")) else null end),
            children: []
        }')
    
    # Add to registry
    registry=$(echo "$registry" | jq \
        --arg name "$name" \
        --argjson process "$process_entry" \
        '.processes[$name] = $process')
    
    # Update parent's children list if applicable
    if [[ -n "$parent" ]]; then
        registry=$(echo "$registry" | jq \
            --arg parent "$parent" \
            --arg child "$name" \
            '
            if .processes[$parent] then
                .processes[$parent].children += [$child] | 
                .processes[$parent].children |= unique
            else . end
            ')
    fi
    
    # Save registry
    save_registry "$registry"
    
    log "INFO" "Successfully registered: $name"
    return 0
}

# Start a process with resource limits
start_process() {
    local name="$1"
    local wait="${2:-false}"
    
    log "INFO" "Starting process: $name"
    
    # Check system CPU before starting new process
    local cpu_usage=$(top -bn1 | grep "Cpu(s)" | sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | awk '{print int(100 - $1)}')
    if [[ $cpu_usage -gt 90 ]]; then
        log "ERROR" "System CPU too high ($cpu_usage%), refusing to start new process"
        return 1
    fi
    
    # Load registry
    log "DEBUG" "Loading registry"
    local registry=$(load_registry)
    
    # Check if process exists
    log "DEBUG" "Checking if process exists: $name"
    local process=$(echo "$registry" | jq --arg name "$name" '.processes[$name]')
    if [[ "$process" == "null" ]]; then
        log "ERROR" "Process not found: $name"
        return 1
    fi
    
    # Check current state
    log "DEBUG" "Checking current process state"
    local state=$(echo "$process" | jq -r '.state')
    if [[ "$state" == "$STATE_RUNNING" ]]; then
        log "WARN" "Process already running: $name"
        return 0
    fi
    
    # Extract process details
    log "DEBUG" "Extracting process details"
    local command=$(echo "$process" | jq -r '.command')
    local working_dir=$(echo "$process" | jq -r '.working_dir')
    local env_vars=$(echo "$process" | jq -r '.env_vars')
    
    # Update state to starting
    registry=$(echo "$registry" | jq \
        --arg name "$name" \
        --arg state "$STATE_STARTING" \
        '.processes[$name].state = $state')
    save_registry "$registry"
    
    # Prepare environment
    local env_file="$ORCHESTRATOR_HOME/sockets/$name.env"
    echo "$env_vars" | jq -r 'to_entries | .[] | "\(.key)=\(.value)"' > "$env_file"
    
    # Start the process
    local log_file="$ORCHESTRATOR_HOME/logs/$name.log"
    
    # Change to working directory and set up environment  
    log "DEBUG" "Changing to working directory: $working_dir"
    local original_dir="$PWD"
    cd "$working_dir" || {
        log "ERROR" "Failed to change to working directory: $working_dir"
        return 1
    }
    
    # Source environment variables safely
    log "DEBUG" "Setting up environment variables"
    local old_allexport=""
    [[ $- =~ a ]] && old_allexport=true
    set -a
    
    # Use a temporary file to avoid process substitution deadlock
    local temp_env="/tmp/orchestrator_env_$$_$(date +%s).env"
    if [[ -n "$env_vars" ]] && [[ "$env_vars" != "{}" ]]; then
        echo "$env_vars" | jq -r 'to_entries | .[] | "\(.key)=\(.value)"' > "$temp_env" 2>/dev/null || true
        if [[ -f "$temp_env" ]] && [[ -s "$temp_env" ]]; then
            while IFS='=' read -r key value; do
                [[ -z "$key" || "$key" =~ ^# ]] && continue
                export "$key"="$value"
            done < "$temp_env"
        fi
        rm -f "$temp_env"
    fi
    
    [[ "$old_allexport" != "true" ]] && set +a
    
    # Validate command before starting
    if [[ -z "$command" ]] || [[ "$command" == "null" ]]; then
        log "ERROR" "Empty or invalid command for process: $name"
        cd "$original_dir"
        return 1
    fi
    
    # Start the process in background with proper error handling
    log "DEBUG" "Starting command: $command"
    
    # Use exec to replace bash process and prevent orphans
    (
        # Set up signal handlers to ensure clean exit
        trap 'exit 0' TERM INT
        trap 'exit 1' ERR
        
        # Execute command
        exec bash -c "$command"
    ) > "$log_file" 2>&1 &
    
    local pid=$!
    
    # Verify process actually started
    sleep 0.1
    if ! kill -0 "$pid" 2>/dev/null; then
        log "ERROR" "Process $name failed to start"
        cd "$original_dir"
        return 1
    fi
    
    log "DEBUG" "Started process with PID: $pid"
    
    # Restore working directory
    cd "$original_dir"
    
    # Update registry with running state
    registry=$(echo "$registry" | jq \
        --arg name "$name" \
        --arg state "$STATE_RUNNING" \
        --arg pid "$pid" \
        --arg started_at "$(date -Iseconds)" \
        '
        .processes[$name].state = $state |
        .processes[$name].pid = ($pid | tonumber) |
        .processes[$name].started_at = $started_at |
        .metadata.total_started += 1
        ')
    save_registry "$registry"
    
    log "INFO" "Started process $name with PID $pid"
    
    # Start children if any
    local children=$(echo "$process" | jq -r '.children[]?' 2>/dev/null)
    if [[ -n "$children" ]]; then
        while IFS= read -r child; do
            [[ -z "$child" ]] && continue
            log "INFO" "Starting child process: $child"
            start_process "$child" false
        done <<< "$children"
    fi
    
    return 0
}

# Stop a process
stop_process() {
    local name="$1"
    local cascade="${2:-true}"
    
    log "INFO" "Stopping process: $name"
    
    # Load registry
    local registry=$(load_registry)
    
    # Check if process exists
    local process=$(echo "$registry" | jq --arg name "$name" '.processes[$name]')
    if [[ "$process" == "null" ]]; then
        log "ERROR" "Process not found: $name"
        return 1
    fi
    
    # Stop children first if cascading
    if [[ "$cascade" == "true" ]]; then
        local children=$(echo "$process" | jq -r '.children[]?' 2>/dev/null)
        if [[ -n "$children" ]]; then
            while IFS= read -r child; do
                [[ -z "$child" ]] && continue
                log "INFO" "Stopping child process: $child"
                stop_process "$child" true
            done <<< "$children"
        fi
    fi
    
    # Get process PID
    local pid=$(echo "$process" | jq -r '.pid // ""')
    local state=$(echo "$process" | jq -r '.state')
    
    if [[ -n "$pid" ]] && [[ "$state" == "$STATE_RUNNING" ]]; then
        # Update state to stopping
        registry=$(echo "$registry" | jq \
            --arg name "$name" \
            --arg state "$STATE_STOPPING" \
            '.processes[$name].state = $state')
        save_registry "$registry"
        
        # Send SIGTERM
        if kill -TERM "$pid" 2>/dev/null; then
            log "INFO" "Sent SIGTERM to process $name (PID $pid)"
            
            # Wait for graceful shutdown (max 10 seconds)
            local count=0
            while [[ $count -lt 10 ]] && kill -0 "$pid" 2>/dev/null; do
                sleep 1
                count=$((count + 1))
            done
            
            # Force kill if still running
            if kill -0 "$pid" 2>/dev/null; then
                log "WARN" "Process $name didn't stop gracefully, sending SIGKILL"
                kill -KILL "$pid" 2>/dev/null || true
            fi
        fi
        
        # Clean up PID file
        rm -f "$ORCHESTRATOR_HOME/sockets/$name.pid"
    fi
    
    # Update registry with stopped state
    registry=$(echo "$registry" | jq \
        --arg name "$name" \
        --arg state "$STATE_STOPPED" \
        --arg stopped_at "$(date -Iseconds)" \
        '
        .processes[$name].state = $state |
        .processes[$name].pid = null |
        .processes[$name].stopped_at = $stopped_at
        ')
    save_registry "$registry"
    
    log "INFO" "Stopped process: $name"
    return 0
}

# Get process status
get_status() {
    local name="${1:-}"
    local format="${2:-text}"
    
    local registry=$(load_registry)
    
    if [[ -z "$name" ]]; then
        # Show all processes
        if [[ "$format" == "json" ]]; then
            echo "$registry" | jq '.processes'
        else
            echo -e "${CYAN}=== Vrooli Orchestrator Status ===${NC}"
            echo ""
            
            # Check daemon status
            if is_running; then
                local pid=$(cat "$PID_FILE")
                echo -e "${GREEN}● Orchestrator:${NC} Running (PID $pid)"
            else
                echo -e "${RED}● Orchestrator:${NC} Not running"
            fi
            
            echo ""
            echo "Processes:"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            printf "%-40s %-12s %-8s %s\n" "NAME" "STATE" "PID" "STARTED"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            
            # List all processes
            echo "$registry" | jq -r '.processes | to_entries | .[] | 
                [.value.name, .value.state, (.value.pid // "-"), (.value.started_at // "-")] | 
                @tsv' | while IFS=$'\t' read -r name state pid started; do
                
                # Color code by state
                case "$state" in
                    "$STATE_RUNNING") state_color="${GREEN}●${NC} $state" ;;
                    "$STATE_STOPPED") state_color="${YELLOW}○${NC} $state" ;;
                    "$STATE_FAILED"|"$STATE_CRASHED") state_color="${RED}✗${NC} $state" ;;
                    *) state_color="◌ $state" ;;
                esac
                
                # Format started time
                if [[ "$started" != "-" ]]; then
                    started=$(date -d "$started" '+%H:%M:%S' 2>/dev/null || echo "$started")
                fi
                
                printf "%-40s %-20s %-8s %s\n" "$name" "$state_color" "$pid" "$started"
            done
            
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            
            # Summary
            local total=$(echo "$registry" | jq '.processes | length')
            local running=$(echo "$registry" | jq --arg state "$STATE_RUNNING" '[.processes[] | select(.state == $state)] | length')
            local stopped=$(echo "$registry" | jq --arg state "$STATE_STOPPED" '[.processes[] | select(.state == $state)] | length')
            
            echo ""
            echo "Total: $total | Running: $running | Stopped: $stopped"
            echo "Limits: Max Total: $MAX_TOTAL_APPS | Max Depth: $MAX_NESTING_DEPTH | Max Per Parent: $MAX_PER_PARENT"
        fi
    else
        # Show specific process
        local process=$(echo "$registry" | jq --arg name "$name" '.processes[$name]')
        if [[ "$process" == "null" ]]; then
            log "ERROR" "Process not found: $name"
            return 1
        fi
        
        if [[ "$format" == "json" ]]; then
            echo "$process"
        else
            echo -e "${CYAN}Process: $name${NC}"
            echo "$process" | jq -r '
                "State: \(.state)\n" +
                "PID: \(.pid // "none")\n" +
                "Command: \(.command)\n" +
                "Working Dir: \(.working_dir)\n" +
                "Started: \(.started_at // "never")\n" +
                "Stopped: \(.stopped_at // "never")\n" +
                "Restart Count: \(.restart_count)\n" +
                "Parent: \(.parent // "none")\n" +
                "Children: \(.children | join(", "))"
            '
        fi
    fi
}

# Show process tree
show_tree() {
    local registry=$(load_registry)
    
    echo -e "${CYAN}=== Vrooli Process Tree ===${NC}"
    echo ""
    
    # Recursive tree printer
    print_tree_node() {
        local name="$1"
        local indent="$2"
        local is_last="$3"
        
        local process=$(echo "$registry" | jq --arg name "$name" '.processes[$name]')
        local state=$(echo "$process" | jq -r '.state')
        local pid=$(echo "$process" | jq -r '.pid // ""')
        
        # State indicator
        local state_icon
        case "$state" in
            "$STATE_RUNNING") state_icon="${GREEN}●${NC}" ;;
            "$STATE_STOPPED") state_icon="${YELLOW}○${NC}" ;;
            "$STATE_FAILED"|"$STATE_CRASHED") state_icon="${RED}✗${NC}" ;;
            *) state_icon="◌" ;;
        esac
        
        # Tree characters
        local prefix=""
        if [[ -n "$indent" ]]; then
            if [[ "$is_last" == "true" ]]; then
                prefix="└── "
            else
                prefix="├── "
            fi
        fi
        
        # Display node
        local display_name="${name#vrooli.}"
        if [[ -n "$pid" ]]; then
            echo -e "${indent}${prefix}${state_icon} ${display_name} (${pid})"
        else
            echo -e "${indent}${prefix}${state_icon} ${display_name}"
        fi
        
        # Process children
        local children=$(echo "$process" | jq -r '.children[]?' 2>/dev/null)
        if [[ -n "$children" ]]; then
            local child_count=$(echo "$children" | wc -l)
            local current_child=0
            
            while IFS= read -r child; do
                [[ -z "$child" ]] && continue
                current_child=$((current_child + 1))
                
                local new_indent="$indent"
                if [[ -n "$indent" ]]; then
                    if [[ "$is_last" == "true" ]]; then
                        new_indent="$indent    "
                    else
                        new_indent="$indent│   "
                    fi
                else
                    new_indent=""
                fi
                
                local child_is_last="false"
                [[ $current_child -eq $child_count ]] && child_is_last="true"
                
                print_tree_node "$child" "$new_indent" "$child_is_last"
            done <<< "$children"
        fi
    }
    
    # Find root processes (those without parents or top-level vrooli.*)
    local roots=$(echo "$registry" | jq -r '.processes | to_entries | .[] | 
        select(.value.parent == null or (.value.name | split(".") | length) == 2) | 
        .value.name' | sort)
    
    if [[ -z "$roots" ]]; then
        echo "No processes registered"
    else
        while IFS= read -r root; do
            print_tree_node "$root" "" "false"
        done <<< "$roots"
    fi
}

# Process monitor loop with safety checks
monitor_processes() {
    log "INFO" "Starting process monitor"
    
    # Add timeout protection
    local consecutive_errors=0
    local max_errors=5
    
    while true; do
        # Safety check - bail if too many consecutive errors
        if [[ $consecutive_errors -ge $max_errors ]]; then
            log "ERROR" "Monitor process had $max_errors consecutive errors, exiting"
            return 1
        fi
        
        # Use timeout to prevent hanging
        if ! timeout 30 bash -c '
            registry=$(cat "'"$REGISTRY_FILE"'" 2>/dev/null || echo "{}")
            echo "$registry"
        ' > /tmp/monitor_registry_$$.json; then
            log "ERROR" "Failed to load registry in monitor"
            consecutive_errors=$((consecutive_errors + 1))
            sleep 5
            continue
        fi
        
        local registry=$(cat /tmp/monitor_registry_$$.json)
        rm -f /tmp/monitor_registry_$$.json
        
        # Reset error counter on success
        consecutive_errors=0
        
        local updated=false
        
        # Check each running process with timeout
        local running_processes=$(echo "$registry" | jq -r --arg state "$STATE_RUNNING" '
            .processes | to_entries | .[] | 
            select(.value.state == $state) | 
            "\(.value.name):\(.value.pid)"
        ')
        
        while IFS=: read -r name pid; do
            [[ -z "$name" ]] && continue
            
            if [[ -n "$pid" ]] && ! kill -0 "$pid" 2>/dev/null; then
                log "WARN" "Process crashed: $name (PID $pid)"
                
                # Update state to crashed
                registry=$(echo "$registry" | jq \
                    --arg name "$name" \
                    --arg state "$STATE_CRASHED" \
                    --arg stopped_at "$(date -Iseconds)" \
                    '
                    .processes[$name].state = $state |
                    .processes[$name].pid = null |
                    .processes[$name].stopped_at = $stopped_at
                    ')
                updated=true
                
                # Check if auto-restart is enabled
                local auto_restart=$(echo "$registry" | jq -r --arg name "$name" '
                    .processes[$name].metadata.auto_restart // false')
                
                if [[ "$auto_restart" == "true" ]]; then
                    local restart_count=$(echo "$registry" | jq -r --arg name "$name" '
                        .processes[$name].restart_count // 0')
                    
                    if [[ $restart_count -lt 3 ]]; then
                        log "INFO" "Auto-restarting crashed process: $name (attempt $((restart_count + 1)))"
                        
                        # Update restart count
                        registry=$(echo "$registry" | jq \
                            --arg name "$name" \
                            --arg count "$((restart_count + 1))" \
                            '.processes[$name].restart_count = ($count | tonumber)')
                        
                        # Save and restart
                        save_registry "$registry"
                        start_process "$name" false
                        updated=false  # Already saved
                    else
                        log "ERROR" "Process $name exceeded restart limit (3)"
                    fi
                fi
            fi
        done <<< "$running_processes"
        
        if [[ "$updated" == "true" ]]; then
            save_registry "$registry"
        fi
        
        # Sleep before next check
        sleep 5
    done
}

# Command processor
process_commands() {
    log "INFO" "Starting command processor"
    
    # Create command FIFO if it doesn't exist
    [[ -p "$COMMAND_FIFO" ]] || mkfifo "$COMMAND_FIFO"
    
    while true; do
        if read -r line < "$COMMAND_FIFO"; then
            log "DEBUG" "Received command: $line"
            
            # Parse command
            local cmd=$(echo "$line" | jq -r '.command' 2>/dev/null || echo "")
            local args=$(echo "$line" | jq -c '.args' 2>/dev/null || echo "[]")
            
            case "$cmd" in
                register)
                    local name=$(echo "$args" | jq -r '.[0]')
                    local command=$(echo "$args" | jq -r '.[1]')
                    local working_dir=$(echo "$args" | jq -r '.[2] // ""')
                    local env_vars=$(echo "$args" | jq -c '.[3] // {}')
                    local metadata=$(echo "$args" | jq -c '.[4] // {}')
                    register_process "$name" "$command" "$working_dir" "$env_vars" "$metadata"
                    ;;
                    
                start)
                    local name=$(echo "$args" | jq -r '.[0]')
                    start_process "$name"
                    ;;
                    
                stop)
                    local name=$(echo "$args" | jq -r '.[0]')
                    local cascade=$(echo "$args" | jq -r '.[1] // true')
                    stop_process "$name" "$cascade"
                    ;;
                    
                status)
                    local name=$(echo "$args" | jq -r '.[0] // ""')
                    get_status "$name" "json"
                    ;;
                    
                tree)
                    show_tree
                    ;;
                    
                *)
                    log "ERROR" "Unknown command: $cmd"
                    ;;
            esac
        fi
    done
}

# Start the orchestrator daemon
start_daemon() {
    init_orchestrator_home
    
    if ! acquire_lock; then
        echo -e "${YELLOW}Orchestrator is already running${NC}"
        echo "Use 'vrooli-orchestrator status' to see process status"
        return 1
    fi
    
    log "INFO" "Starting Vrooli Orchestrator daemon"
    
    # Update metadata
    local registry=$(load_registry)
    registry=$(echo "$registry" | jq --arg started_at "$(date -Iseconds)" '
        .metadata.started_at = $started_at')
    
    # Auto-cleanup crashed/stopped/failed processes on startup
    local cleanup_count=$(echo "$registry" | jq --arg crashed "$STATE_CRASHED" --arg stopped "$STATE_STOPPED" --arg failed "$STATE_FAILED" '
        .processes | to_entries | 
        map(select(.value.state == $crashed or .value.state == $stopped or .value.state == $failed)) | 
        length
    ')
    
    if [[ $cleanup_count -gt 0 ]]; then
        log "INFO" "Auto-cleaning $cleanup_count crashed/stopped/failed processes"
        registry=$(echo "$registry" | jq --arg crashed "$STATE_CRASHED" --arg stopped "$STATE_STOPPED" --arg failed "$STATE_FAILED" '
            .processes |= with_entries(select(.value.state != $crashed and .value.state != $stopped and .value.state != $failed))
        ')
        log "INFO" "Cleaned up $cleanup_count old processes"
    fi
    
    save_registry "$registry"
    
    log "INFO" "Running orchestrator with PID $$"
    
    echo -e "${GREEN}✓ Vrooli Orchestrator started${NC}"
    echo "  PID: $$"
    echo "  Socket: $SOCKET_FILE"
    echo "  Registry: $REGISTRY_FILE"
    echo "  Logs: $LOG_FILE"
    
    # Set up signal handling with flag to prevent recursion
    local shutdown_requested=false
    
    # Signal handler that sets flag instead of directly calling stop_daemon
    handle_shutdown() {
        if [[ "$shutdown_requested" == "false" ]]; then
            shutdown_requested=true
            log "INFO" "Shutdown signal received"
            stop_daemon
            exit 0
        fi
    }
    
    # Trap signals for cleanup
    trap 'handle_shutdown' SIGTERM SIGINT
    
    # Start background workers with error handling
    monitor_processes &
    local monitor_pid=$!
    
    process_commands &
    local command_pid=$!
    
    log "INFO" "Orchestrator started successfully with workers: monitor=$monitor_pid, commands=$command_pid"
    
    # Monitor worker processes and restart if they die
    while true; do
        # Check if we should shut down
        if [[ ! -f "$PID_FILE" ]] || [[ "$(cat "$PID_FILE" 2>/dev/null)" != "$$" ]]; then
            log "INFO" "PID file removed or changed, shutting down"
            kill $monitor_pid $command_pid 2>/dev/null || true
            exit 0
        fi
        
        # Check monitor process
        if ! kill -0 $monitor_pid 2>/dev/null; then
            log "WARN" "Monitor process died, restarting"
            monitor_processes &
            monitor_pid=$!
        fi
        
        # Check command processor
        if ! kill -0 $command_pid 2>/dev/null; then
            log "WARN" "Command processor died, restarting"
            process_commands &
            command_pid=$!
        fi
        
        # Sleep before next check
        sleep 10
    done
}

# Stop the orchestrator daemon
stop_daemon() {
    log "INFO" "Stopping Vrooli Orchestrator daemon"
    
    if ! is_running; then
        echo -e "${YELLOW}Orchestrator is not running${NC}"
        return 1
    fi
    
    # First, kill all monitor and command processor background jobs
    log "INFO" "Killing background workers"
    pkill -f "monitor_processes" 2>/dev/null || true
    pkill -f "process_commands" 2>/dev/null || true
    
    # Stop all running processes
    local registry=$(load_registry)
    local running_processes=$(echo "$registry" | jq -r --arg state "$STATE_RUNNING" '
        .processes | to_entries | .[] | 
        select(.value.state == $state) | 
        .value.name')
    
    if [[ -n "$running_processes" ]]; then
        echo "Stopping all managed processes..."
        while IFS= read -r name; do
            [[ -z "$name" ]] && continue
            stop_process "$name" true
        done <<< "$running_processes"
    fi
    
    # Kill any orphaned bash processes that might be stuck
    local orchestrator_pids=$(pgrep -f "vrooli-orchestrator" | grep -v $$)
    if [[ -n "$orchestrator_pids" ]]; then
        log "INFO" "Killing orphaned orchestrator processes: $orchestrator_pids"
        echo "$orchestrator_pids" | xargs -r kill -9 2>/dev/null || true
    fi
    
    # Kill daemon process (avoid infinite recursion by removing trap first)
    local pid=$(cat "$PID_FILE")
    if [[ -n "$pid" ]] && [[ "$pid" != "$$" ]]; then
        # External stop - kill the daemon process
        kill -TERM "$pid" 2>/dev/null || true
        
        # Wait for process to stop
        local count=0
        while [[ $count -lt 10 ]] && kill -0 "$pid" 2>/dev/null; do
            sleep 1
            count=$((count + 1))
        done
        
        # Force kill if needed
        kill -0 "$pid" 2>/dev/null && kill -KILL "$pid" 2>/dev/null || true
    elif [[ "$pid" == "$$" ]]; then
        # Self-stop from signal handler - just clean up and exit
        trap - SIGTERM SIGINT  # Remove trap to prevent recursion
    fi
    
    # Clean up
    release_lock
    rm -f "$COMMAND_FIFO"
    
    log "INFO" "Orchestrator stopped"
    echo -e "${GREEN}✓ Vrooli Orchestrator stopped${NC}"
    
    # If this is a self-stop, exit the daemon
    if [[ "$pid" == "$$" ]]; then
        exit 0
    fi
}

# Main command handler
main() {
    local command="${1:-}"
    shift || true
    
    case "$command" in
        start)
            start_daemon "$@"
            ;;
            
        stop)
            stop_daemon "$@"
            ;;
            
        restart)
            stop_daemon
            sleep 2
            start_daemon "$@"
            ;;
            
        status)
            get_status "$@"
            ;;
            
        tree)
            show_tree
            ;;
            
        register|start-process|stop-process)
            # These commands are sent via the client library
            echo "This command should be invoked through the orchestrator client"
            echo "Source: scripts/scenarios/tools/orchestrator-client.sh"
            exit 1
            ;;
            
        *)
            echo "Vrooli Orchestrator - Singleton Process Manager"
            echo ""
            echo "Usage: $0 {start|stop|restart|status|tree}"
            echo ""
            echo "Commands:"
            echo "  start    - Start the orchestrator daemon"
            echo "  stop     - Stop the daemon and all processes"
            echo "  restart  - Restart the daemon"
            echo "  status   - Show process status"
            echo "  tree     - Show process hierarchy tree"
            echo ""
            echo "Environment Variables:"
            echo "  VROOLI_ORCHESTRATOR_HOME - Orchestrator home directory (default: ~/.vrooli/orchestrator)"
            echo "  VROOLI_MAX_APPS          - Maximum total apps (default: 20)"
            echo "  VROOLI_MAX_DEPTH         - Maximum nesting depth (default: 5)"
            echo "  VROOLI_MAX_PER_PARENT    - Maximum apps per parent (default: 10)"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"