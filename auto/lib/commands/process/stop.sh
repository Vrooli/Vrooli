#!/usr/bin/env bash

# Stop Command Module - Process Termination
# Handles both graceful stop and force-stop operations

set -euo pipefail

#######################################
# Stop the loop process
# Arguments: 
#   --force - Force immediate termination (optional)
# Returns: 0 on success
#######################################
cmd_execute() {
    local force_stop=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force_stop=true
                shift
                ;;
            *)
                echo "ERROR: Unknown argument: $1" >&2
                return 1
                ;;
        esac
    done
    
    local stopped=false
    
    # Stop main loop process
    if [[ -f "$PID_FILE" ]]; then
        local main_pid; main_pid=$(commands::safe_read "$PID_FILE")
        if [[ -n "$main_pid" ]]; then
            if commands::pid_running "$main_pid"; then
                if [[ "$force_stop" == "true" ]]; then
                    echo "Force-stopping main process PID: $main_pid"
                    kill_tree "$main_pid" KILL
                else
                    echo "Stopping main process PID: $main_pid"
                    kill_tree "$main_pid" TERM
                    
                    # Wait for graceful termination
                    local wait_count=0
                    while commands::pid_running "$main_pid" && [[ $wait_count -lt $PROCESS_TERMINATION_WAIT ]]; do
                        sleep 1
                        ((wait_count++))
                    done
                    
                    # Force kill if still running
                    if commands::pid_running "$main_pid"; then
                        echo "Process not responding, force killing PID: $main_pid"
                        kill_tree "$main_pid" KILL
                    fi
                fi
                stopped=true
            else
                echo "Main process not running (stale PID file)"
            fi
        fi
    else
        echo "No PID file found - loop not running"
    fi
    
    # Stop any tracked worker processes
    if [[ -f "$PIDS_FILE" ]]; then
        local worker_count=0
        while IFS= read -r worker_pid; do
            if [[ -n "$worker_pid" ]] && commands::pid_running "$worker_pid"; then
                if [[ "$force_stop" == "true" ]]; then
                    kill_tree "$worker_pid" KILL
                else
                    kill_tree "$worker_pid" TERM
                fi
                ((worker_count++))
            fi
        done < "$PIDS_FILE"
        
        if [[ $worker_count -gt 0 ]]; then
            echo "Stopped $worker_count worker process(es)"
        fi
        
        commands::safe_remove "$PIDS_FILE"
    fi
    
    # Clean up files
    commands::safe_remove "$PID_FILE"
    commands::safe_remove "$LOCK_FILE"
    
    # Report status
    if [[ "$force_stop" == "true" ]]; then
        echo "Force-stopped (task=$LOOP_TASK)"
    else
        echo "Stopped (task=$LOOP_TASK)"
    fi
    
    return 0
}

#######################################
# Validate stop command arguments
# Arguments: Command arguments
# Returns: 0 if valid, 1 if invalid
#######################################
cmd_validate() {
    # Check for valid arguments
    for arg in "$@"; do
        case "$arg" in
            --force)
                # Valid argument
                ;;
            *)
                echo "ERROR: Invalid argument: $arg" >&2
                echo "Valid arguments: --force" >&2
                return 1
                ;;
        esac
    done
    
    return 0
}

#######################################
# Show help for stop command
#######################################
cmd_help() {
    cat << EOF
stop - Stop the auto loop process

Usage: stop [--force]

Arguments:
  --force    Force immediate termination (SIGKILL)

Description:
  Stops the auto loop process gracefully or forcefully.

  Normal stop:
  1. Sends SIGTERM to main process
  2. Waits for graceful shutdown
  3. Force kills if process doesn't respond
  4. Stops all tracked worker processes
  5. Cleans up PID and lock files

  Force stop:
  1. Immediately sends SIGKILL to all processes
  2. Cleans up PID and lock files

Examples:
  task-manager.sh --task resource-improvement stop
  task-manager.sh --task resource-improvement stop --force
  manage-resource-loop.sh stop
  manage-resource-loop.sh force-stop

See also: start, status
EOF
}