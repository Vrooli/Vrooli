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
    # Check if already running
    if [[ -f "$PID_FILE" ]]; then
        local existing_pid; existing_pid=$(cat "$PID_FILE")
        if commands::pid_running "$existing_pid"; then
            echo "Loop already running with PID: $existing_pid"
            return 1
        else
            echo "Removing stale PID file"
            commands::safe_remove "$PID_FILE"
        fi
    fi
    
    # Start the loop in background
    local task_manager="${AUTO_DIR}/task-manager.sh"
    if [[ ! -x "$task_manager" ]]; then
        echo "ERROR: Task manager not found or not executable: $task_manager" >&2
        return 1
    fi
    
    nohup "$task_manager" --task "$LOOP_TASK" run-loop >/dev/null 2>&1 & 
    local start_pid=$!
    echo "Started loop (task=$LOOP_TASK) PID: $start_pid"
    
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
        echo "WARNING: Loop started but PID file not created within expected time"
        echo "Background process PID: $start_pid"
        return 0
    fi
}

#######################################
# Validate start command arguments
# Arguments: Command arguments
# Returns: 0 if valid, 1 if invalid
#######################################
cmd_validate() {
    # Start command takes no arguments
    if [[ $# -gt 0 ]]; then
        echo "ERROR: Start command does not accept arguments" >&2
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