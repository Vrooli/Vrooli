#!/usr/bin/env bash

# Start Command Module - Background Process Management
# Starts the auto loop in the background using nohup

set -euo pipefail

#######################################
# Start the loop in background
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
cmd_execute() {
    # Check if PID file exists first - most common case
    if [[ -f "$PID_FILE" ]]; then
        local existing_pid; existing_pid=$(cat "$PID_FILE")
        if commands::pid_running "$existing_pid"; then
            echo "ERROR: Loop already running with PID: $existing_pid (from PID file)"
            echo "Use 'stop' or 'force-stop' to terminate existing processes first"
            return 1
        else
            echo "Removing stale PID file"
            commands::safe_remove "$PID_FILE"
        fi
    fi
    
    # Simplified process check - avoid complex grep patterns that can hang
    local running_pids=""
    if pgrep -f "task-manager.*$LOOP_TASK.*run-loop" >/dev/null 2>&1; then
        running_pids=$(pgrep -f "task-manager.*$LOOP_TASK.*run-loop" | head -1)
    fi
    
    if [[ -n "$running_pids" ]]; then
        echo "ERROR: Loop already running for task '$LOOP_TASK' with PID(s): $running_pids"
        echo "Use 'stop' or 'force-stop' to terminate existing processes first"
        return 1
    fi
    
    # Start the loop in background
    local task_manager="${AUTO_DIR}/task-manager.sh"
    if [[ ! -x "$task_manager" ]]; then
        echo "ERROR: Task manager not found or not executable: $task_manager" >&2
        return 1
    fi
    
    # Create a log file for nohup output to capture startup errors
    local nohup_log="${DATA_DIR}/nohup.out"
    echo "Starting loop (task=$LOOP_TASK) in background..."
    nohup "$task_manager" --task "$LOOP_TASK" run-loop >"$nohup_log" 2>&1 & 
    local start_pid=$!
    echo "Started loop launcher PID: $start_pid"
    
    # Wait for the actual loop to start and write its PID
    local wait_count=0
    while [[ ! -f "$PID_FILE" && $wait_count -lt $PID_FILE_WAIT_ITERATIONS ]]; do
        sleep 0.1
        ((wait_count++))
    done
    
    if [[ -f "$PID_FILE" ]]; then
        local actual_pid; actual_pid=$(cat "$PID_FILE")
        echo "Loop running with PID: $actual_pid"
        return 0
    else
        echo "WARNING: Loop process failed to create PID file within $(echo "scale=1; $PID_FILE_WAIT_ITERATIONS / 10" | bc)s"
        echo "Check ${nohup_log} for startup errors"
        # Check if the launcher process is still running
        if ! kill -0 "$start_pid" 2>/dev/null; then
            echo "Launcher process $start_pid already died - loop failed to start"
            # Show last few lines of nohup log if it exists
            if [[ -f "$nohup_log" ]] && [[ -s "$nohup_log" ]]; then
                echo ""
                echo "Last lines from nohup.out:"
                tail -5 "$nohup_log"
            fi
            return 1
        fi
        echo "Launcher process is still running, loop may take time to initialize"
        return 0
    fi
}

#######################################
# Validate start command arguments
# Arguments: Command arguments
# Returns: 0 if valid, 1 if invalid
#######################################
cmd_validate() {
    # Start command takes no arguments, but may receive shell redirect descriptors
    # Filter out numeric arguments that are likely shell redirections
    local filtered_args=()
    for arg in "$@"; do
        if [[ ! "$arg" =~ ^[0-9]+$ ]]; then
            filtered_args+=("$arg")
        fi
    done
    
    if [[ ${#filtered_args[@]} -gt 0 ]]; then
        echo "ERROR: Start command does not accept arguments: ${filtered_args[*]}" >&2
        return 1
    fi
    
    # Check if task manager exists
    local task_manager="${AUTO_DIR}/task-manager.sh"
    if [[ ! -f "$task_manager" ]]; then
        echo "ERROR: Task manager not found: $task_manager" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Show help for start command
#######################################
cmd_help() {
    cat << EOF
start - Start the auto loop in background

Usage: start

Description:
  Starts the auto loop process in the background using nohup.
  The loop will continue running until explicitly stopped.

  The command will:
  1. Check if a loop is already running
  2. Start the loop in background
  3. Wait for PID file creation
  4. Report the running PID

Examples:
  task-manager.sh --task resource-improvement start
  manage-resource-loop.sh start

See also: stop, force-stop, status
EOF
}