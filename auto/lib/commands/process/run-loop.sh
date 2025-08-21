#!/usr/bin/env bash

# Run-Loop Command Module - Main Loop Execution
# Executes the main loop in foreground with optional iteration limit

set -euo pipefail

#######################################
# Run the main loop in foreground
# Arguments: 
#   --max N - Maximum number of iterations (optional)
# Returns: Exit code from run_loop function
#######################################
cmd_execute() {
    local max_iterations=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --max)
                if [[ $# -lt 2 ]]; then
                    echo "ERROR: --max requires a number" >&2
                    return 1
                fi
                max_iterations="$2"
                shift 2
                ;;
            *)
                echo "ERROR: Unknown argument: $1" >&2
                return 1
                ;;
        esac
    done
    
    # Set max iterations if specified
    if [[ -n "$max_iterations" ]]; then
        if ! [[ "$max_iterations" =~ ^[0-9]+$ ]]; then
            echo "ERROR: --max must be a positive integer, got: $max_iterations" >&2
            return 1
        fi
        export RUN_MAX_ITERATIONS="$max_iterations"
        echo "Running loop with max iterations: $max_iterations"
    else
        echo "Running loop indefinitely (Ctrl-C to stop)"
    fi
    
    # Execute the main loop
    run_loop
}

#######################################
# Validate run-loop command arguments
# Arguments: Command arguments
# Returns: 0 if valid, 1 if invalid
#######################################
cmd_validate() {
    local expecting_number=false
    
    for arg in "$@"; do
        if [[ "$expecting_number" == "true" ]]; then
            if ! [[ "$arg" =~ ^[0-9]+$ ]] || [[ "$arg" -le 0 ]]; then
                echo "ERROR: --max requires a positive integer, got: $arg" >&2
                return 1
            fi
            expecting_number=false
        elif [[ "$arg" == "--max" ]]; then
            expecting_number=true
        else
            echo "ERROR: Invalid argument: $arg" >&2
            echo "Valid arguments: --max N" >&2
            return 1
        fi
    done
    
    if [[ "$expecting_number" == "true" ]]; then
        echo "ERROR: --max requires a number" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Show help for run-loop command
#######################################
cmd_help() {
    cat << EOF
run-loop - Execute the main loop in foreground

Usage: run-loop [--max N]

Arguments:
  --max N    Run for maximum N iterations then exit

Description:
  Executes the main auto loop in the foreground. This is the core
  command that performs the actual loop iterations.

  Without --max: Runs indefinitely until stopped with Ctrl-C
  With --max: Runs for the specified number of iterations then exits

  The loop will:
  1. Execute iterations at configured intervals
  2. Handle worker processes and timeouts
  3. Log all activity to the loop log
  4. Emit events to the events ledger
  5. Update summary files after each iteration

  This command blocks the terminal and should be used for:
  - Interactive development and testing
  - Foreground execution in containers
  - Limited iteration runs for testing

  For background operation, use the 'start' command instead.

Examples:
  task-manager.sh --task resource-improvement run-loop
  task-manager.sh --task resource-improvement run-loop --max 5
  manage-resource-loop.sh run-loop --max 10

See also: start, stop, once
EOF
}