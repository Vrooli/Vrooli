#!/usr/bin/env bash
################################################################################
# Vrooli Orchestrator Client Library
# 
# A client library that apps can source to interact with the Vrooli Orchestrator
# daemon. Provides simple functions for registering processes, starting/stopping
# them, and managing their lifecycle across any nesting level.
#
# Usage in any app script:
#   source scripts/scenarios/tools/orchestrator-client.sh
#   orchestrator::register "api" "./start-api.sh"
#   orchestrator::start "api"
#   orchestrator::stop "api"
#
# Features:
# - Automatic parent context detection
# - Hierarchical naming
# - Environment variable inheritance
# - Health checking
# - Auto-restart configuration
################################################################################

# Client configuration
ORCHESTRATOR_HOME="${VROOLI_ORCHESTRATOR_HOME:-$HOME/.vrooli/orchestrator}"
COMMAND_FIFO="$ORCHESTRATOR_HOME/commands.fifo"
CLIENT_TIMEOUT="${VROOLI_CLIENT_TIMEOUT:-10}"

# Auto-detect parent context (based on PWD and process hierarchy)
orchestrator::detect_parent() {
    local parent=""
    
    # Method 1: Check environment variable (set by parent orchestrator calls)
    if [[ -n "${VROOLI_ORCHESTRATOR_PARENT:-}" ]]; then
        parent="$VROOLI_ORCHESTRATOR_PARENT"
    
    # Method 2: Infer from current working directory
    elif [[ "$PWD" == *"/generated-apps/"* ]]; then
        # Extract app name from path like /home/user/generated-apps/my-app/...
        local app_name=$(echo "$PWD" | sed -E 's|.*/generated-apps/([^/]+).*|\1|')
        if [[ -n "$app_name" ]] && [[ "$app_name" != "$PWD" ]]; then
            parent="vrooli.$app_name"
        fi
    
    # Method 3: Check for .vrooli metadata (only for generated apps, not main project)
    elif [[ -f ".vrooli/service.json" ]] && [[ "$PWD" != */Vrooli ]] && [[ "$PWD" == */generated-apps/* ]]; then
        local service_name=$(jq -r '.service.name // ""' .vrooli/service.json 2>/dev/null)
        if [[ -n "$service_name" ]]; then
            parent="vrooli.$service_name"
        fi
    fi
    
    echo "$parent"
}

# Ensure orchestrator is running
orchestrator::ensure_daemon() {
    local orchestrator_script="${BASH_SOURCE[0]%/*}/vrooli-orchestrator.sh"
    local startup_lock="$ORCHESTRATOR_HOME/startup.lock"
    
    # Create orchestrator home directory
    mkdir -p "$ORCHESTRATOR_HOME"
    
    # Use file locking to prevent race condition during daemon startup
    local lockfile
    lockfile=$(mktemp "$startup_lock.XXXXXX") || {
        echo "❌ Failed to create startup lock" >&2
        return 1
    }
    
    # Acquire lock with timeout
    local lock_acquired=false
    local timeout=15  # 15 seconds timeout
    local start_time=$(date +%s)
    
    while [[ $(($(date +%s) - start_time)) -lt $timeout ]]; do
        if (set -C; echo $$ > "$startup_lock") 2>/dev/null; then
            lock_acquired=true
            break
        fi
        sleep 0.5
    done
    
    if [[ "$lock_acquired" != "true" ]]; then
        rm -f "$lockfile"
        echo "❌ Failed to acquire startup lock (timeout)" >&2
        return 1
    fi
    
    # Cleanup function for lock
    cleanup_startup_lock() {
        rm -f "$ORCHESTRATOR_HOME/startup.lock" 2>/dev/null || true
        [[ -n "${lockfile:-}" ]] && rm -f "$lockfile" 2>/dev/null || true
    }
    trap cleanup_startup_lock EXIT
    
    # Double-check if daemon is running (after acquiring lock)
    if [[ -f "$ORCHESTRATOR_HOME/orchestrator.pid" ]]; then
        local pid=$(cat "$ORCHESTRATOR_HOME/orchestrator.pid")
        if kill -0 "$pid" 2>/dev/null; then
            cleanup_startup_lock
            return 0  # Already running
        else
            # Clean up stale PID file
            rm -f "$ORCHESTRATOR_HOME/orchestrator.pid"
        fi
    fi
    
    # Start daemon (now protected by lock)
    echo "🚀 Starting Vrooli Orchestrator..."
    if [[ -x "$orchestrator_script" ]]; then
        # Start daemon in background and wait for it to be ready
        "$orchestrator_script" start >/dev/null 2>&1 &
        local daemon_start_pid=$!
        
        # Wait for daemon to be ready
        local count=0
        while [[ $count -lt 15 ]]; do
            if [[ -p "$COMMAND_FIFO" ]]; then
                echo "✅ Orchestrator ready"
                cleanup_startup_lock
                return 0
            fi
            
            # Check if daemon startup process is still running
            if ! kill -0 "$daemon_start_pid" 2>/dev/null; then
                echo "❌ Daemon startup process exited unexpectedly" >&2
                cleanup_startup_lock
                return 1
            fi
            
            sleep 1
            count=$((count + 1))
        done
        
        echo "❌ Failed to start orchestrator daemon (timeout waiting for FIFO)" >&2
        cleanup_startup_lock
        return 1
    else
        echo "❌ Orchestrator script not found: $orchestrator_script" >&2
        cleanup_startup_lock
        return 1
    fi
}

# Send command to orchestrator daemon
orchestrator::send_command() {
    local command="$1"
    shift
    local args=("$@")
    
    # Ensure daemon is running
    if ! orchestrator::ensure_daemon; then
        return 1
    fi
    
    # Prepare command JSON (compact format)
    local cmd_json
    if [[ ${#args[@]} -eq 0 ]]; then
        # Handle empty args array
        cmd_json=$(jq -n -c \
            --arg command "$command" \
            '{command: $command, args: []}')
    else
        # Handle non-empty args array
        cmd_json=$(jq -n -c \
            --arg command "$command" \
            --argjson args "$(printf '%s\n' "${args[@]}" | jq -R . | jq -s .)" \
            '{command: $command, args: $args}')
    fi
    
    # Send to daemon via FIFO (simplified without file locking)
    if [[ -p "$COMMAND_FIFO" ]]; then
        # Send command as a single atomic operation with explicit newline
        printf "%s\n" "$cmd_json" > "$COMMAND_FIFO"
        return $?
    else
        echo "❌ Cannot communicate with orchestrator (FIFO not found)" >&2
        return 1
    fi
}

# Register a process with the orchestrator
orchestrator::register() {
    local name="$1"
    local command="$2"
    local options=()
    shift 2
    
    # Parse optional arguments
    local working_dir="$(pwd)"
    local env_vars="{}"
    local metadata="{}"
    local auto_restart=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --working-dir)
                working_dir="$2"
                shift 2
                ;;
            --env)
                local key="$2"
                local value="$3"
                env_vars=$(echo "$env_vars" | jq -c --arg key "$key" --arg value "$value" '.[$key] = $value')
                shift 3
                ;;
            --env-file)
                if [[ -f "$2" ]]; then
                    while IFS='=' read -r key value; do
                        [[ -z "$key" || "$key" =~ ^# ]] && continue
                        env_vars=$(echo "$env_vars" | jq -c --arg key "$key" --arg value "$value" '.[$key] = $value')
                    done < "$2"
                fi
                shift 2
                ;;
            --metadata)
                local meta_key="$2"
                local meta_value="$3"
                metadata=$(echo "$metadata" | jq -c --arg key "$meta_key" --arg value "$meta_value" '.[$key] = $value')
                shift 3
                ;;
            --auto-restart)
                auto_restart=true
                metadata=$(echo "$metadata" | jq -c '.auto_restart = true')
                shift
                ;;
            --health-check)
                local health_cmd="$2"
                metadata=$(echo "$metadata" | jq -c --arg cmd "$health_cmd" '.health_check = $cmd')
                shift 2
                ;;
            --depends)
                local deps="$2"
                metadata=$(echo "$metadata" | jq -c --arg deps "$deps" '.dependencies = ($deps | split(","))')
                shift 2
                ;;
            *)
                echo "❌ Unknown option: $1" >&2
                return 1
                ;;
        esac
    done
    
    # Auto-detect parent and generate full process name
    local parent=$(orchestrator::detect_parent)
    local full_name
    if [[ -n "$parent" ]]; then
        full_name="$parent.$name"
    else
        full_name="vrooli.$name"
    fi
    
    echo "📝 Registering process: $full_name"
    
    # Send registration command
    orchestrator::send_command "register" "$full_name" "$command" "$working_dir" "$env_vars" "$metadata"
    local result=$?
    
    if [[ $result -eq 0 ]]; then
        echo "✅ Registered: $full_name"
        
        # Export namespace (without the final component) for child processes
        # This prevents infinite nesting like vrooli.app.develop.app.develop
        local namespace
        if [[ "$full_name" =~ ^(.+)\.[^.]+$ ]]; then
            namespace="${BASH_REMATCH[1]}"
        else
            namespace="$full_name"
        fi
        export VROOLI_ORCHESTRATOR_PARENT="$namespace"
    else
        echo "❌ Failed to register: $full_name" >&2
    fi
    
    return $result
}

# Start a process
orchestrator::start() {
    local name="$1"
    local wait="${2:-false}"
    
    # Auto-detect parent and generate full process name
    local parent=$(orchestrator::detect_parent)
    local full_name
    if [[ -n "$parent" ]]; then
        full_name="$parent.$name"
    else
        full_name="vrooli.$name"
    fi
    
    echo "🚀 Starting process: $full_name"
    
    # Send start command
    orchestrator::send_command "start" "$full_name"
    local result=$?
    
    if [[ $result -eq 0 ]]; then
        echo "✅ Started: $full_name"
        
        # Wait for process if requested
        if [[ "$wait" == "true" ]]; then
            orchestrator::wait_for_state "$name" "running"
        fi
    else
        echo "❌ Failed to start: $full_name" >&2
    fi
    
    return $result
}

# Stop a process
orchestrator::stop() {
    local name="$1"
    local cascade="${2:-true}"
    
    # Auto-detect parent and generate full process name
    local parent=$(orchestrator::detect_parent)
    local full_name
    if [[ -n "$parent" ]]; then
        full_name="$parent.$name"
    else
        full_name="vrooli.$name"
    fi
    
    echo "🛑 Stopping process: $full_name"
    
    # Send stop command
    orchestrator::send_command "stop" "$full_name" "$cascade"
    local result=$?
    
    if [[ $result -eq 0 ]]; then
        echo "✅ Stopped: $full_name"
    else
        echo "❌ Failed to stop: $full_name" >&2
    fi
    
    return $result
}

# Restart a process
orchestrator::restart() {
    local name="$1"
    
    echo "🔄 Restarting process: $name"
    
    # Stop then start
    if orchestrator::stop "$name" false; then
        sleep 2
        orchestrator::start "$name"
    else
        echo "❌ Failed to restart: $name" >&2
        return 1
    fi
}

# Get process status
orchestrator::status() {
    local name="${1:-}"
    
    if [[ -n "$name" ]]; then
        # Auto-detect parent and generate full process name
        local parent=$(orchestrator::detect_parent)
        local full_name
        if [[ -n "$parent" ]]; then
            full_name="$parent.$name"
        else
            full_name="vrooli.$name"
        fi
        
        orchestrator::send_command "status" "$full_name"
    else
        orchestrator::send_command "status"
    fi
}

# Wait for a process to reach a specific state
orchestrator::wait_for_state() {
    local name="$1"
    local target_state="$2"
    local timeout="${3:-30}"
    
    echo "⏳ Waiting for $name to reach state: $target_state"
    
    local count=0
    while [[ $count -lt $timeout ]]; do
        local status=$(orchestrator::status "$name" 2>/dev/null | jq -r '.state // ""')
        
        if [[ "$status" == "$target_state" ]]; then
            echo "✅ Process $name reached state: $target_state"
            return 0
        fi
        
        # Check for failure states
        if [[ "$status" == "failed" || "$status" == "crashed" ]]; then
            echo "❌ Process $name failed to reach $target_state (current: $status)" >&2
            return 1
        fi
        
        sleep 1
        count=$((count + 1))
    done
    
    echo "⏰ Timeout waiting for $name to reach state: $target_state" >&2
    return 1
}

# Show process tree
orchestrator::tree() {
    orchestrator::send_command "tree"
}

# Get process logs
orchestrator::logs() {
    local name="$1"
    local lines="${2:-50}"
    
    # Auto-detect parent and generate full process name
    local parent=$(orchestrator::detect_parent)
    local full_name
    if [[ -n "$parent" ]]; then
        full_name="$parent.$name"
    else
        full_name="vrooli.$name"
    fi
    
    local log_file="$ORCHESTRATOR_HOME/logs/$full_name.log"
    
    if [[ -f "$log_file" ]]; then
        echo "📜 Logs for $full_name (last $lines lines):"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        tail -n "$lines" "$log_file"
    else
        echo "❌ No logs found for: $full_name" >&2
        return 1
    fi
}

# Follow process logs
orchestrator::follow_logs() {
    local name="$1"
    
    # Auto-detect parent and generate full process name
    local parent=$(orchestrator::detect_parent)
    local full_name
    if [[ -n "$parent" ]]; then
        full_name="$parent.$name"
    else
        full_name="vrooli.$name"
    fi
    
    local log_file="$ORCHESTRATOR_HOME/logs/$full_name.log"
    
    if [[ -f "$log_file" ]]; then
        echo "📜 Following logs for $full_name (Ctrl+C to stop):"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        tail -f "$log_file"
    else
        echo "❌ No logs found for: $full_name" >&2
        return 1
    fi
}

# Helper: Register and start a process in one command
orchestrator::run() {
    local name="$1"
    local command="$2"
    shift 2
    
    echo "🎯 Registering and starting process: $name"
    
    if orchestrator::register "$name" "$command" "$@"; then
        orchestrator::start "$name"
    else
        echo "❌ Failed to run process: $name" >&2
        return 1
    fi
}

# Helper: Register multiple processes from a configuration
orchestrator::load_config() {
    local config_file="$1"
    
    if [[ ! -f "$config_file" ]]; then
        echo "❌ Configuration file not found: $config_file" >&2
        return 1
    fi
    
    echo "📋 Loading processes from: $config_file"
    
    # Parse and register each process
    jq -c '.processes[]' "$config_file" | while read -r process; do
        local name=$(echo "$process" | jq -r '.name')
        local command=$(echo "$process" | jq -r '.command')
        local working_dir=$(echo "$process" | jq -r '.working_dir // ""')
        local auto_start=$(echo "$process" | jq -r '.auto_start // false')
        
        # Build registration command
        local reg_args=("$name" "$command")
        
        if [[ -n "$working_dir" ]]; then
            reg_args+=("--working-dir" "$working_dir")
        fi
        
        # Add environment variables
        local env_vars=$(echo "$process" | jq -c '.env // {}')
        if [[ "$env_vars" != "{}" ]]; then
            echo "$env_vars" | jq -r 'to_entries | .[] | "--env \(.key) \(.value)"' | \
            while read -r env_arg; do
                reg_args+=($env_arg)
            done
        fi
        
        # Add metadata
        local metadata=$(echo "$process" | jq -c '.metadata // {}')
        if [[ "$metadata" != "{}" ]]; then
            echo "$metadata" | jq -r 'to_entries | .[] | "--metadata \(.key) \(.value)"' | \
            while read -r meta_arg; do
                reg_args+=($meta_arg)
            done
        fi
        
        # Register the process
        if orchestrator::register "${reg_args[@]}"; then
            if [[ "$auto_start" == "true" ]]; then
                orchestrator::start "$name"
            fi
        fi
    done
    
    echo "✅ Configuration loaded successfully"
}

# Helper: Stop all processes in current context
orchestrator::stop_all() {
    local parent=$(orchestrator::detect_parent)
    
    if [[ -n "$parent" ]]; then
        echo "🛑 Stopping all processes under: $parent"
        orchestrator::stop "${parent#vrooli.}" true
    else
        echo "🛑 Stopping all top-level processes"
        # Get all top-level processes and stop them
        local processes=$(orchestrator::send_command "status" | jq -r '.processes | keys[]' 2>/dev/null)
        while IFS= read -r process_name; do
            [[ -z "$process_name" ]] && continue
            if [[ "$process_name" =~ ^vrooli\.[^.]+$ ]]; then
                local short_name="${process_name#vrooli.}"
                orchestrator::stop "$short_name" true
            fi
        done <<< "$processes"
    fi
}

# Cleanup function for scripts
orchestrator::cleanup() {
    echo ""
    echo "🧹 Cleaning up processes..."
    orchestrator::stop_all
}

# Auto-register cleanup on EXIT (only for scripts, not when sourced interactively)
# trap orchestrator::cleanup EXIT

# Export functions for use in scripts
export -f orchestrator::register
export -f orchestrator::start
export -f orchestrator::stop
export -f orchestrator::restart
export -f orchestrator::status
export -f orchestrator::tree
export -f orchestrator::logs
export -f orchestrator::follow_logs
export -f orchestrator::run
export -f orchestrator::wait_for_state
export -f orchestrator::stop_all
export -f orchestrator::cleanup

# Indicate library is loaded
export VROOLI_ORCHESTRATOR_CLIENT_LOADED=true

echo "🎼 Vrooli Orchestrator Client loaded"
echo "   Parent context: $(orchestrator::detect_parent || echo 'none')"