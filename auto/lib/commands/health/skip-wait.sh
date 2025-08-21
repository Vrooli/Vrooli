#!/usr/bin/env bash

# Skip-Wait Command Module - Iteration Wait Control
# Sets flag to skip current iteration wait interval

set -euo pipefail

#######################################
# Execute skip-wait command
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
cmd_execute() {
    # Create the skip wait flag file
    local flag_file="${DATA_DIR}/skip_wait.flag"
    
    if touch "$flag_file" 2>/dev/null; then
        echo "⏭️  Skip wait flag set - next iteration will skip wait interval"
        return 0
    else
        echo "ERROR: Failed to create skip wait flag: $flag_file" >&2
        return 1
    fi
}

#######################################
# Validate skip-wait command arguments
# Arguments: Command arguments
# Returns: 0 if valid, 1 if invalid  
#######################################
cmd_validate() {
    # Skip-wait command takes no arguments
    if [[ $# -gt 0 ]]; then
        echo "ERROR: Skip-wait command does not accept arguments" >&2
        return 1
    fi
    
    # Check if data directory is writable
    if [[ ! -w "$DATA_DIR" ]]; then
        echo "ERROR: Data directory not writable: $DATA_DIR" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Show help for skip-wait command
#######################################
cmd_help() {
    cat << EOF
skip-wait - Skip current iteration wait interval

Usage: skip-wait

Description:
  Creates a flag file that causes the running loop to skip its
  current wait interval and immediately proceed to the next iteration.
  
  The command creates: \$DATA_DIR/skip_wait.flag
  
  This is useful for:
  - Accelerating testing during development
  - Triggering immediate iteration after configuration changes
  - Debugging iteration scheduling issues
  
  The flag is automatically removed by the loop after being processed,
  so it only affects the next iteration.

Examples:
  task-manager.sh --task resource-improvement skip-wait
  manage-resource-loop.sh skip-wait
  
  # Trigger immediate iteration
  task-manager.sh skip-wait
  
  # Chain with other commands
  task-manager.sh skip-wait && echo "Flag set, next iteration will run immediately"

Behavior:
  - Only affects the currently running loop
  - Flag is consumed (deleted) after processing
  - Has no effect if no loop is currently running
  - Safe to call multiple times

Dependencies:
  - Data directory must be writable
  - No other dependencies

See also: status, logs, once
EOF
}