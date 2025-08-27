#!/usr/bin/env bash

# Once Command Module - Single Iteration Execution  
# Runs a single iteration synchronously for testing

set -euo pipefail

#######################################
# Execute once command
# Arguments: None
# Returns: Exit code from worker execution
#######################################
cmd_execute() {
    # Check if worker is available before proceeding
    if ! check_worker_available; then
        echo "ERROR: Worker not available" >&2
        return 1
    fi
    
    echo "=== Starting single iteration with real-time turn visibility ==="
    echo "Task: $LOOP_TASK"
    echo "Timeout: ${TIMEOUT}s"
    echo "Max turns: $MAX_TURNS"
    echo
    
    # For 'once' command, always use stream-json to show real-time turns
    # This makes Claude Code emit each turn as a separate JSON object in real-time
    export OUTPUT_FORMAT="stream-json"
    
    echo "🤖 Executing Claude Code with real-time turn streaming..."
    echo "Each turn will appear as JSON objects showing Claude's progress"
    echo
    
    # Run single iteration synchronously with stream-json output
    local exitcode
    run_iteration_sync 1
    exitcode=$?
    
    echo
    echo "=== Execution completed ==="
    echo "Exit code: $exitcode"
    
    # Show where the full log is saved
    local iter_log
    iter_log="${ITERATIONS_DIR}/iter-$(printf "%05d" 1).log"
    if [[ -f "$iter_log" ]]; then
        echo "Full iteration log: $iter_log"
    fi
    
    return $exitcode
}

#######################################
# Validate once command arguments
# Arguments: Command arguments  
# Returns: 0 if valid, 1 if invalid
#######################################
cmd_validate() {
    # Once command takes no arguments
    if [[ $# -gt 0 ]]; then
        echo "ERROR: Once command does not accept arguments" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Show help for once command
#######################################
cmd_help() {
    cat << EOF
once - Run single iteration synchronously with real-time turn visibility

Usage: once

Description:
  Runs a single iteration of the auto loop synchronously (blocking) with
  real-time streaming of Claude's conversation turns. This is useful for:
  - Testing worker execution
  - Debugging iteration issues  
  - Manual iteration execution
  - Validating prompt and configuration
  - Observing Claude's decision-making process
  
  The command will:
  1. Check worker availability
  2. Execute one iteration with stream-json output format
  3. Show each Claude turn as it happens (JSON objects)
  4. Wait for completion
  5. Report the exit code
  
  Unlike the normal loop, this runs in the foreground and shows
  Claude's actual conversation turns in real-time using stream-json
  format from Claude Code.

Real-time Output:
  Each conversation turn appears as a JSON object containing:
  - User messages and tool calls
  - Assistant responses and reasoning
  - System messages and progress updates
  - Tool execution results and outputs

Examples:
  task-manager.sh --task resource-improvement once
  manage-resource-loop.sh once
  
  # Check if iteration succeeds
  if task-manager.sh once; then
    echo "Iteration succeeded"
  else
    echo "Iteration failed with code: $?"
  fi

Exit Codes:
  Passes through the exit code from the worker execution:
  0   - Success
  1   - General error
  124 - Timeout
  142 - Quota exhausted
  150 - Configuration error  
  151 - Worker unavailable

Dependencies:
  - All worker dependencies must be available
  - Prompt must be accessible
  - Environment must be properly configured
  - Claude Code with stream-json support

See also: health, dry-run, status
EOF
}