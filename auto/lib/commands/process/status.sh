#!/usr/bin/env bash

# Status Command Module - Loop Status Reporting
# Provides detailed status information about the loop and its configuration

set -euo pipefail

#######################################
# Show loop status and configuration
# Arguments: None
# Returns: 0 on success
#######################################
cmd_execute() {
    echo "=== Loop Status ==="
    
    # Check main process status
    local status="STOPPED"
    local main_pid=""
    
    # First check PID file
    if [[ -f "$PID_FILE" ]]; then
        main_pid=$(commands::safe_read "$PID_FILE")
        if [[ -n "$main_pid" ]]; then
            if commands::pid_running "$main_pid"; then
                status="RUNNING (PID: $main_pid)"
            else
                status="STOPPED (stale PID file)"
            fi
        else
            status="STOPPED (empty PID file)"
        fi
    fi
    
    # If no PID file or stale, check for actual running processes using pgrep
    if [[ "$status" == "STOPPED"* ]] || [[ "$status" == "STOPPED" ]]; then
        # Use a simpler pgrep approach
        local running_pids
        if pgrep -f "task-manager" >/dev/null 2>&1; then
            # Check if any of the task-manager processes match our task
            running_pids=$(pgrep -f "task-manager" 2>/dev/null | while read -r pid; do
                if ps -p "$pid" -o cmd --no-headers 2>/dev/null | grep -q "$LOOP_TASK.*run-loop"; then
                    echo "$pid"
                    break
                fi
            done | head -1)
            
            if [[ -n "$running_pids" ]]; then
                status="RUNNING (PID: $running_pids, no PID file)"
                main_pid="$running_pids"
            fi
        fi
    fi
    
    echo "Status: $status"
    echo "Task: $LOOP_TASK"
    
    # Show worker status if running
    if [[ "$status" =~ ^RUNNING ]]; then
        local worker_count=0
        if [[ -f "$PIDS_FILE" ]]; then
            while IFS= read -r worker_pid; do
                if [[ -n "$worker_pid" ]] && commands::pid_running "$worker_pid"; then
                    ((worker_count++))
                fi
            done < "$PIDS_FILE"
        fi
        echo "Active workers: $worker_count"
    fi
    
    echo ""
    echo "=== Configuration ==="
    
    # Show prompt information
    local prompt_path
    prompt_path=$(select_prompt 2>/dev/null || echo "(not found)")
    echo "Prompt: $prompt_path"
    
    if [[ -f "$prompt_path" ]]; then
        local prompt_sha; prompt_sha=$(sha256sum "$prompt_path" 2>/dev/null | awk '{print $1}' || echo "unknown")
        local prompt_size; prompt_size=$(wc -c < "$prompt_path" 2>/dev/null || echo "unknown")
        echo "Prompt SHA: $prompt_sha"
        echo "Prompt size: $prompt_size bytes"
    fi
    
    echo ""
    echo "=== Runtime Configuration ==="
    echo "Interval: ${INTERVAL_SECONDS}s"
    echo "Max turns: $MAX_TURNS"
    echo "Timeout: ${TIMEOUT}s"
    echo "Max concurrent workers: $MAX_CONCURRENT_WORKERS"
    echo "Max TCP connections: $MAX_TCP_CONNECTIONS"
    
    echo ""
    echo "=== Files ==="
    echo "Log: $LOG_FILE"
    if [[ -f "$LOG_FILE" ]]; then
        local log_size; log_size=$(stat -c%s "$LOG_FILE" 2>/dev/null || echo "0")
        echo "Log size: $log_size bytes"
    fi
    
    echo "Events: $EVENTS_JSONL"
    if [[ -f "$EVENTS_JSONL" ]]; then
        local events_lines; events_lines=$(wc -l < "$EVENTS_JSONL" 2>/dev/null || echo "0")
        echo "Events count: $events_lines"
    fi
    
    echo "Data directory: $DATA_DIR"
    echo "PID file: $PID_FILE"
    echo "Lock file: $LOCK_FILE"
    
    # Show disk usage for data directory
    if [[ -d "$DATA_DIR" ]]; then
        echo ""
        echo "=== Data Directory Usage ==="
        local dir_size; dir_size=$(du -sh "$DATA_DIR" 2>/dev/null | cut -f1 || echo "unknown")
        echo "Total size: $dir_size"
        
        # Show iteration count
        if [[ -d "${ITERATIONS_DIR:-}" ]]; then
            local iter_count; iter_count=$(find "${ITERATIONS_DIR}" -name "iter-*.log" 2>/dev/null | wc -l || echo "0")
            echo "Iteration logs: $iter_count"
        fi
    fi
    
    return 0
}

#######################################
# Validate status command arguments
# Arguments: Command arguments
# Returns: 0 if valid, 1 if invalid
#######################################
cmd_validate() {
    # Status command takes no arguments
    if [[ $# -gt 0 ]]; then
        echo "ERROR: Status command does not accept arguments" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Show help for status command
#######################################
cmd_help() {
    cat << EOF
status - Show loop status and configuration

Usage: status

Description:
  Displays comprehensive status information about the auto loop including:
  
  - Process status (running/stopped with PID)
  - Active worker count
  - Prompt configuration and integrity
  - Runtime configuration values
  - File locations and sizes
  - Data directory usage
  - Iteration history

  The status includes both static configuration and dynamic runtime information.

Examples:
  task-manager.sh --task resource-improvement status
  manage-resource-loop.sh status

See also: start, stop, logs, health
EOF
}