#!/usr/bin/env bash

# Scenario Improvement Loop Script
# Sends the scenario-improvement-loop.md prompt to resource-claude-code every 5 minutes

set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Try multiple possible locations for the prompt file
PROMPT_FILE=""
for candidate in \
    "${PROJECT_DIR}/scenario-improvement-loop.md" \
    "/tmp/scenario-improvement-loop.md"; do
    if [[ -f "$candidate" ]]; then
        PROMPT_FILE="$candidate"
        break
    fi
done
LOG_FILE="${PROJECT_DIR}/scenario-improvement-loop.log"
PID_FILE="${PROJECT_DIR}/scenario-improvement-loop.pid"

# Configuration
INTERVAL_SECONDS=180  # 3 minutes (allows overlap between Claude instances)
MAX_TURNS=20
TIMEOUT=3600  # 1 hour timeout per execution
MAX_CONCURRENT_CLAUDE=5  # Maximum concurrent Claude processes
RUNNING_PIDS_FILE="${PROJECT_DIR}/claude-pids.txt"

# Function to log with timestamp
log_with_timestamp() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# Function to clean up on exit
cleanup() {
    log_with_timestamp "Received termination signal, cleaning up scenario improvement loop..."
    
    # Kill any remaining Claude processes
    if [[ -f "$RUNNING_PIDS_FILE" ]]; then
        log_with_timestamp "Terminating Claude processes..."
        while IFS= read -r pid; do
            if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
                log_with_timestamp "Terminating Claude process $pid"
                kill -TERM "$pid" 2>/dev/null || true
                
                # Wait briefly for graceful shutdown
                local count=0
                while kill -0 "$pid" 2>/dev/null && [[ $count -lt 3 ]]; do
                    sleep 1
                    ((count++))
                done
                
                # Force kill if still running
                if kill -0 "$pid" 2>/dev/null; then
                    log_with_timestamp "Force killing Claude process $pid"
                    kill -9 "$pid" 2>/dev/null || true
                fi
            fi
        done < "$RUNNING_PIDS_FILE"
        rm -f "$RUNNING_PIDS_FILE"
    fi
    
    # Clean up all related files
    rm -f "$PID_FILE"
    log_with_timestamp "Cleanup complete, exiting"
    exit 0
}

# Set up signal handlers
trap cleanup SIGTERM SIGINT

# Function to clean up finished Claude processes
cleanup_finished_processes() {
    if [[ -f "$RUNNING_PIDS_FILE" ]]; then
        local temp_file=$(mktemp)
        while IFS= read -r pid; do
            if kill -0 "$pid" 2>/dev/null; then
                echo "$pid" >> "$temp_file"
            else
                log_with_timestamp "Claude process $pid finished, removing from tracking"
            fi
        done < "$RUNNING_PIDS_FILE"
        mv "$temp_file" "$RUNNING_PIDS_FILE"
    fi
}

# Function to check if we can start a new Claude process
can_start_new_claude() {
    cleanup_finished_processes
    local running_count=0
    if [[ -f "$RUNNING_PIDS_FILE" ]]; then
        running_count=$(wc -l < "$RUNNING_PIDS_FILE")
    fi
    
    if [[ $running_count -ge $MAX_CONCURRENT_CLAUDE ]]; then
        log_with_timestamp "Maximum concurrent Claude processes ($MAX_CONCURRENT_CLAUDE) reached, skipping iteration"
        return 1
    fi
    
    return 0
}

# Function to check if claude-code resource is available
check_claude_code() {
    # Check if vrooli command exists
    if ! command -v vrooli >/dev/null 2>&1; then
        log_with_timestamp "ERROR: vrooli command not found in PATH"
        return 1
    fi
    
    # Check if resource-claude-code command exists
    if ! command -v resource-claude-code >/dev/null 2>&1; then
        log_with_timestamp "ERROR: resource-claude-code command not found in PATH"
        return 1
    fi
    
    # Test resource-claude-code with a simple status check
    if ! resource-claude-code status >/dev/null 2>&1; then
        log_with_timestamp "WARNING: claude-code resource status check failed (may still work)"
        # Don't return 1 here as the status command might fail but run command might work
    fi
    
    return 0
}

# Function to validate configuration and files
validate_environment() {
    # Check if prompt file was found
    if [[ -z "$PROMPT_FILE" ]]; then
        log_with_timestamp "FATAL: Prompt file not found in any expected location:"
        log_with_timestamp "  - ${PROJECT_DIR}/scenario-improvement-loop.md"
        log_with_timestamp "  - /tmp/scenario-improvement-loop.md"
        return 1
    fi
    
    # Check if prompt file is readable
    if [[ ! -r "$PROMPT_FILE" ]]; then
        log_with_timestamp "FATAL: Prompt file exists but is not readable: $PROMPT_FILE"
        return 1
    fi
    
    # Check if prompt file has reasonable content
    local prompt_size
    prompt_size=$(wc -c < "$PROMPT_FILE" 2>/dev/null || echo 0)
    if [[ $prompt_size -lt 100 ]]; then
        log_with_timestamp "WARNING: Prompt file seems too small ($prompt_size bytes): $PROMPT_FILE"
    fi
    
    # Validate configuration values
    if [[ $INTERVAL_SECONDS -lt 60 ]]; then
        log_with_timestamp "WARNING: Interval is very short ($INTERVAL_SECONDS seconds)"
    fi
    
    if [[ $MAX_TURNS -lt 1 ]]; then
        log_with_timestamp "ERROR: MAX_TURNS must be at least 1, got: $MAX_TURNS"
        return 1
    fi
    
    if [[ $TIMEOUT -lt 60 ]]; then
        log_with_timestamp "ERROR: TIMEOUT must be at least 60 seconds, got: $TIMEOUT"
        return 1
    fi
    
    return 0
}

# Function to run claude-code with the scenario improvement prompt
run_improvement_iteration() {
    local iteration_count="$1"
    
    log_with_timestamp "Starting iteration #${iteration_count}"
    
    # Check if we can start a new Claude process
    if ! can_start_new_claude; then
        return 1
    fi
    
    # Validate prompt file (should have been checked in validate_environment)
    if [[ ! -f "$PROMPT_FILE" ]]; then
        log_with_timestamp "ERROR: Prompt file not found: $PROMPT_FILE"
        return 1
    fi
    
    if [[ ! -r "$PROMPT_FILE" ]]; then
        log_with_timestamp "ERROR: Prompt file not readable: $PROMPT_FILE"
        return 1
    fi
    
    # Read the prompt content
    local prompt_content
    prompt_content=$(cat "$PROMPT_FILE")
    
    # Prepend ultra think instruction to the prompt
    local full_prompt="Ultra think. ${prompt_content}"
    
    log_with_timestamp "Executing claude-code with max-turns=$MAX_TURNS, timeout=$TIMEOUT"
    
    # Run claude-code in background with logging
    local start_time=$(date +%s)
    
    # Create a temporary file for this iteration's output
    local temp_output="/tmp/scenario-improvement-${iteration_count}-$(date +%s).log"
    
    # Run the command in background and capture its PID
    (
        # Properly export environment variables for resource-claude-code
        export MAX_TURNS="$MAX_TURNS"
        export TIMEOUT="$TIMEOUT"
        export ALLOWED_TOOLS="Read,Write,Edit,Bash,LS,Glob,Grep"
        export SKIP_PERMISSIONS="yes"  # Enable for automation
        
        # Run resource-claude-code with proper error capture
        local claude_exit_code=0
        resource-claude-code run "$full_prompt" 2>&1 | tee "$temp_output"
        claude_exit_code=${PIPESTATUS[0]}
        
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        # Log results based on exit code
        if [[ $claude_exit_code -eq 0 ]]; then
            log_with_timestamp "Iteration #${iteration_count} completed successfully in ${duration}s"
        elif [[ $claude_exit_code -eq 124 ]]; then
            log_with_timestamp "Iteration #${iteration_count} timed out after ${duration}s (${TIMEOUT}s limit)"
        else
            log_with_timestamp "Iteration #${iteration_count} failed with exit code $claude_exit_code after ${duration}s"
        fi
        
        # Save exit code for parent process
        echo "$claude_exit_code" > "${temp_output}.exit"
        
        # Clean up temp output after a short delay
        sleep 5
        rm -f "$temp_output" "${temp_output}.exit"
        
        exit $claude_exit_code
    ) &
    
    local claude_pid=$!
    log_with_timestamp "Claude-code process started with PID: $claude_pid"
    
    # Add PID to tracking file
    echo "$claude_pid" >> "$RUNNING_PIDS_FILE"
    
    # Don't wait for completion - let it run in background
    return 0
}

# Function to check if another instance is running
check_existing_instance() {
    if [[ -f "$PID_FILE" ]]; then
        local existing_pid
        existing_pid=$(cat "$PID_FILE")
        if kill -0 "$existing_pid" 2>/dev/null; then
            log_with_timestamp "ERROR: Another instance is already running with PID: $existing_pid"
            log_with_timestamp "To stop it: kill $existing_pid"
            exit 1
        else
            log_with_timestamp "Removing stale PID file"
            rm -f "$PID_FILE"
        fi
    fi
}

# Main function
main() {
    log_with_timestamp "Starting scenario improvement loop"
    log_with_timestamp "Prompt file: $PROMPT_FILE"
    log_with_timestamp "Interval: ${INTERVAL_SECONDS}s (5 minutes)"
    log_with_timestamp "Max turns: $MAX_TURNS"
    log_with_timestamp "Timeout: ${TIMEOUT}s"
    
    # Check for existing instance
    check_existing_instance
    
    # Write PID file
    echo $$ > "$PID_FILE"
    
    # Initial validation
    log_with_timestamp "Validating environment..."
    if ! validate_environment; then
        log_with_timestamp "FATAL: Environment validation failed"
        cleanup
        exit 1
    fi
    
    if ! check_claude_code; then
        log_with_timestamp "FATAL: Claude-code resource check failed"
        cleanup
        exit 1
    fi
    
    log_with_timestamp "Environment validation successful"
    log_with_timestamp "Claude-code resource is available"
    log_with_timestamp "Using prompt file: $PROMPT_FILE"
    
    # Main loop
    local iteration=1
    while true; do
        log_with_timestamp "--- Iteration $iteration ---"
        
        # Run improvement iteration (non-blocking)
        if run_improvement_iteration "$iteration"; then
            log_with_timestamp "Iteration $iteration started successfully"
        else
            log_with_timestamp "Skipped iteration $iteration (max concurrent processes or error)"
        fi
        
        # Wait for next iteration with signal-responsive sleep
        log_with_timestamp "Waiting ${INTERVAL_SECONDS}s until next iteration..."
        local remaining_time=$INTERVAL_SECONDS
        while [[ $remaining_time -gt 0 ]]; do
            # Sleep in small chunks to be responsive to signals
            local sleep_chunk=$((remaining_time > 10 ? 10 : remaining_time))
            sleep $sleep_chunk
            remaining_time=$((remaining_time - sleep_chunk))
        done
        
        ((iteration++))
    done
}

# Show usage if --help is passed
if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
    cat << EOF
Scenario Improvement Loop Script

This script runs claude-code with the scenario-improvement-loop.md prompt
every 5 minutes in a continuous loop.

Usage: $0 [OPTIONS]

Options:
  --help, -h    Show this help message
  
Files:
  Prompt: $PROMPT_FILE
  Log:    $LOG_FILE
  PID:    $PID_FILE

To stop the loop:
  kill \$(cat $PID_FILE)
  
Configuration:
  Interval: ${INTERVAL_SECONDS}s (5 minutes)
  Max turns: $MAX_TURNS
  Timeout: ${TIMEOUT}s (1 hour)
EOF
    exit 0
fi

# Start the main loop
main "$@"