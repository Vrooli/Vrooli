#!/usr/bin/env bash

# Scenario Improvement Loop Manager
# Helper script to start, stop, and monitor the scenario improvement loop

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOOP_SCRIPT="${SCRIPT_DIR}/scenario-improvement-loop.sh"
LOG_FILE="${SCRIPT_DIR}/scenario-improvement-loop.log"
PID_FILE="${SCRIPT_DIR}/scenario-improvement-loop.pid"

# Function to show status
show_status() {
    echo "Scenario Improvement Loop Status"
    echo "================================"
    
    if [[ -f "$PID_FILE" ]]; then
        local pid
        pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            echo "Status: RUNNING (PID: $pid)"
            echo "Started: $(ps -o lstart= -p "$pid" 2>/dev/null || echo "Unknown")"
        else
            echo "Status: STOPPED (stale PID file)"
            rm -f "$PID_FILE"
        fi
    else
        echo "Status: STOPPED"
    fi
    
    echo
    echo "Files:"
    echo "  Script: $LOOP_SCRIPT"
    echo "  Log:    $LOG_FILE"
    echo "  PID:    $PID_FILE"
    
    if [[ -f "$LOG_FILE" ]]; then
        local log_size
        log_size=$(wc -l < "$LOG_FILE" 2>/dev/null || echo 0)
        echo "  Log size: $log_size lines"
        
        # Warn if log is getting large
        if [[ $log_size -gt 10000 ]]; then
            echo "  WARNING: Log file is large ($log_size lines). Consider rotating."
        fi
        
        echo
        echo "Recent log entries:"
        tail -5 "$LOG_FILE" 2>/dev/null || echo "  (log file empty or unreadable)"
    fi
}

# Function to start the loop
start_loop() {
    echo "Starting scenario improvement loop..."
    
    # Check if loop script exists
    if [[ ! -f "$LOOP_SCRIPT" ]]; then
        echo "ERROR: Loop script not found: $LOOP_SCRIPT"
        echo "Check if the script exists and is executable"
        exit 1
    fi
    
    # Check if loop script is executable
    if [[ ! -x "$LOOP_SCRIPT" ]]; then
        echo "ERROR: Loop script is not executable: $LOOP_SCRIPT"
        echo "Run: chmod +x $LOOP_SCRIPT"
        exit 1
    fi
    
    if [[ -f "$PID_FILE" ]]; then
        local pid
        pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            echo "ERROR: Loop is already running with PID: $pid"
            exit 1
        else
            echo "Removing stale PID file"
            rm -f "$PID_FILE"
        fi
    fi
    
    # Start the loop in background
    nohup "$LOOP_SCRIPT" >/dev/null 2>&1 &
    local new_pid=$!
    
    # Wait a moment to see if it started successfully
    sleep 2
    if kill -0 "$new_pid" 2>/dev/null; then
        echo "Loop started successfully with PID: $new_pid"
        echo "To monitor: tail -f $LOG_FILE"
        echo "To stop: $0 stop"
    else
        echo "ERROR: Failed to start loop"
        exit 1
    fi
}

# Function to stop the loop
stop_loop() {
    echo "Stopping scenario improvement loop..."
    
    if [[ -f "$PID_FILE" ]]; then
        local pid
        pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            echo "Sending SIGTERM to PID: $pid"
            kill "$pid" 2>/dev/null || true
            
            # Wait for graceful shutdown
            local count=0
            echo -n "Waiting for graceful shutdown"
            while kill -0 "$pid" 2>/dev/null && [[ $count -lt 10 ]]; do
                echo -n "."
                sleep 1
                ((count++))
            done
            echo
            
            # Check if process is still running
            if kill -0 "$pid" 2>/dev/null; then
                echo "Process still running, sending SIGKILL"
                kill -9 "$pid" 2>/dev/null || true
                
                # Wait a bit more for SIGKILL to take effect
                sleep 2
                
                if kill -0 "$pid" 2>/dev/null; then
                    echo "WARNING: Process $pid still running after SIGKILL"
                fi
            fi
            
            # Always clean up PID file after attempting to kill process
            echo "Cleaning up PID file"
            rm -f "$PID_FILE"
            
            # Also clean up any related files
            rm -f "${SCRIPT_DIR}/claude-pids.txt"
            
            echo "Loop stopped"
        else
            echo "Process not running, removing stale PID file"
            rm -f "$PID_FILE"
            rm -f "${SCRIPT_DIR}/claude-pids.txt"
        fi
    else
        echo "Loop is not running (no PID file found)"
    fi
}

# Function to force stop the loop (emergency stop)
force_stop_loop() {
    echo "Force stopping scenario improvement loop..."
    
    # Kill any process matching the script name
    local killed_any=false
    
    # Find processes by script name
    local pids
    pids=$(pgrep -f "scenario-improvement-loop.sh" 2>/dev/null || true)
    
    if [[ -n "$pids" ]]; then
        echo "Found running processes: $pids"
        for pid in $pids; do
            if kill -0 "$pid" 2>/dev/null; then
                echo "Force killing process $pid"
                kill -9 "$pid" 2>/dev/null || true
                killed_any=true
            fi
        done
    fi
    
    # Also kill Claude processes from the tracking file
    local claude_pids_file="${SCRIPT_DIR}/claude-pids.txt"
    if [[ -f "$claude_pids_file" ]]; then
        echo "Cleaning up Claude processes..."
        while IFS= read -r pid; do
            if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                echo "Force killing Claude process $pid"
                kill -9 "$pid" 2>/dev/null || true
                killed_any=true
            fi
        done < "$claude_pids_file"
    fi
    
    # Clean up all files
    echo "Cleaning up all loop-related files..."
    rm -f "$PID_FILE"
    rm -f "${SCRIPT_DIR}/claude-pids.txt"
    
    if [[ "$killed_any" == "true" ]]; then
        echo "Force stop completed - processes were terminated"
    else
        echo "Force stop completed - no running processes found"
    fi
}

# Function to show logs
show_logs() {
    if [[ -f "$LOG_FILE" ]]; then
        if [[ "${1:-}" == "follow" ]] || [[ "${1:-}" == "-f" ]]; then
            echo "Following log file (Ctrl+C to stop):"
            tail -f "$LOG_FILE"
        else
            echo "Log file contents:"
            cat "$LOG_FILE"
        fi
    else
        echo "Log file not found: $LOG_FILE"
    fi
}

# Function to rotate logs
rotate_logs() {
    if [[ -f "$LOG_FILE" ]]; then
        local timestamp=$(date '+%Y%m%d_%H%M%S')
        local archived_log="${LOG_FILE}.${timestamp}"
        
        echo "Rotating log file..."
        mv "$LOG_FILE" "$archived_log"
        echo "Archived to: $archived_log"
        
        # Keep only last 5 archived logs
        local log_dir=$(dirname "$LOG_FILE")
        local log_name=$(basename "$LOG_FILE")
        find "$log_dir" -name "${log_name}.*" -type f | sort -r | tail -n +6 | xargs rm -f 2>/dev/null || true
        
        echo "Log rotation complete. New logs will start fresh."
    else
        echo "No log file to rotate: $LOG_FILE"
    fi
}

# Function to show help
show_help() {
    cat << EOF
Scenario Improvement Loop Manager

This script helps manage the scenario improvement loop that runs claude-code
with the scenario-improvement-loop.md prompt every 5 minutes.

Usage: $0 <command>

Commands:
  start       Start the improvement loop
  stop        Stop the improvement loop gracefully
  force-stop  Force stop the loop (emergency stop, kills all related processes)
  status      Show current status
  logs        Show log file contents
  logs -f     Follow log file (live updates)
  rotate      Rotate log files (archive current log and start fresh)
  restart     Stop and then start the loop
  help        Show this help message

Examples:
  $0 start         # Start the loop
  $0 status        # Check if running
  $0 logs -f       # Watch logs in real-time
  $0 stop          # Stop the loop gracefully
  $0 force-stop    # Emergency stop (if normal stop fails)

The loop will:
- Run every 5 minutes
- Use max-turns=20
- Include "Ultra think" in the prompt
- Run claude-code in background (non-blocking)
- Log all activity to: $LOG_FILE
EOF
}

# Main command handler
case "${1:-help}" in
    start)
        start_loop
        ;;
    stop)
        stop_loop
        ;;
    force-stop)
        force_stop_loop
        ;;
    status)
        show_status
        ;;
    logs)
        show_logs "${2:-}"
        ;;
    rotate)
        rotate_logs
        ;;
    restart)
        stop_loop
        sleep 2
        start_loop
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo "ERROR: Unknown command: ${1:-}"
        echo
        show_help
        exit 1
        ;;
esac