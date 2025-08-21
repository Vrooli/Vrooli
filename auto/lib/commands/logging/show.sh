#!/usr/bin/env bash

# Logs Show Command Module - Log File Viewing
# Displays or follows the loop log file

set -euo pipefail

#######################################
# Show or follow log file
# Arguments: 
#   -f, --follow, follow - Follow log file (tail -f)
# Returns: 0 on success, 1 on failure
#######################################
cmd_execute() {
    local follow_mode=false
    
    # Parse arguments
    for arg in "$@"; do
        case "$arg" in
            -f|--follow|follow)
                follow_mode=true
                ;;
            *)
                echo "ERROR: Unknown argument: $arg" >&2
                return 1
                ;;
        esac
    done
    
    # Check if log file exists
    if [[ ! -f "$LOG_FILE" ]]; then
        echo "Log file not found: $LOG_FILE"
        echo "Loop may not have been started yet"
        return 1
    fi
    
    # Show or follow log
    if [[ "$follow_mode" == "true" ]]; then
        echo "Following log file: $LOG_FILE (Ctrl-C to stop)"
        echo "--- Log output ---"
        tail -f "$LOG_FILE"
    else
        echo "Showing log file: $LOG_FILE"
        echo "--- Log output ---"
        cat "$LOG_FILE"
    fi
    
    return 0
}

#######################################
# Validate logs command arguments
# Arguments: Command arguments
# Returns: 0 if valid, 1 if invalid
#######################################
cmd_validate() {
    # Check for valid arguments
    for arg in "$@"; do
        case "$arg" in
            -f|--follow|follow)
                # Valid arguments
                ;;
            *)
                echo "ERROR: Invalid argument: $arg" >&2
                echo "Valid arguments: -f, --follow, follow" >&2
                return 1
                ;;
        esac
    done
    
    return 0
}

#######################################
# Show help for logs command
#######################################
cmd_help() {
    cat << EOF
logs - Display or follow the loop log file

Usage: logs [-f|--follow|follow]

Arguments:
  -f, --follow, follow    Follow log file in real-time (like tail -f)

Description:
  Displays the contents of the loop log file. Without arguments,
  shows the entire log file and exits. With follow mode, continuously
  displays new log entries as they are written.

  The log file contains:
  - Loop start/stop events
  - Iteration progress
  - Worker execution results
  - Error messages and warnings
  - System status changes

Examples:
  task-manager.sh --task resource-improvement logs
  task-manager.sh --task resource-improvement logs -f
  task-manager.sh --task resource-improvement logs --follow
  manage-resource-loop.sh logs follow

See also: status, rotate
EOF
}