#!/usr/bin/env bash

# Dispatch Library - Command dispatching and main loop
# Part of the modular loop system
#
# This module provides:
# - Main loop execution logic
# - Command dispatch routing
# - Process lifecycle management
# - Lock file management
# - Integration with modular command handlers

set -euo pipefail

# -----------------------------------------------------------------------------
# Function: run_loop
# Description: Main loop execution with iteration scheduling and management
# Parameters: None (uses environment variables)
# Returns: 0 on clean exit
# Side Effects:
#   - Creates lock file to prevent duplicates
#   - Writes PID file
#   - Runs iterations at INTERVAL_SECONDS intervals
#   - Rotates logs when needed
#   - Responds to skip_wait.flag for immediate continuation
# Environment Variables:
#   - INTERVAL_SECONDS: Delay between iterations (default: 300)
#   - RUN_MAX_ITERATIONS: Optional max iteration limit
#   - LOOP_TASK: Task identifier
# -----------------------------------------------------------------------------
# -----------------------------------------------------------------------------
# Function: cleanup_lock_fd
# Description: Close the lock file descriptor and remove lock file if owned by current process
# Parameters: None
# Returns: 0
# Side Effects: Closes file descriptor 9 if open, removes lock file if we own it
# -----------------------------------------------------------------------------
cleanup_lock_fd() {
	# Remove lock file if it contains our PID
	if [[ -f "$LOCK_FILE" ]]; then
		local lock_pid
		lock_pid=$(cat "$LOCK_FILE" 2>/dev/null || echo "")
		if [[ "$lock_pid" == "$$" ]]; then
			rm -f "$LOCK_FILE" 2>/dev/null || true
		fi
	fi
	
	# Close FD 9 if it's open
	if [[ -e /proc/$$/fd/9 ]]; then
		exec 9>&- 2>/dev/null || true
	fi
	return 0
}

# -----------------------------------------------------------------------------
# Function: cleanup_pid_file
# Description: Remove PID file on exit
# Parameters: None
# Returns: 0
# Side Effects: Removes PID_FILE if it contains current process ID
# -----------------------------------------------------------------------------
cleanup_pid_file() {
	# Only cleanup if we're the main loop process (not a subshell)
	# Check that we're in the main shell by comparing BASHPID with stored PID
	if [[ -f "$PID_FILE" ]]; then
		local stored_pid
		stored_pid=$(cat "$PID_FILE" 2>/dev/null || echo "")
		# Use BASHPID instead of $$ to detect subshells
		# $$ is inherited by subshells, but BASHPID is not
		if [[ "$stored_pid" == "$BASHPID" ]]; then
			log_with_timestamp "Removing PID file (main loop exiting)"
			rm -f "$PID_FILE"
		fi
	fi
	return 0
}

run_loop() {
	log_with_timestamp "DEBUG: run_loop starting, PID=$$, BASHPID=$BASHPID"
	# Lock to prevent duplicate instances with PID validation
	if command -v flock >/dev/null 2>&1; then
		# Check for stale lock before attempting to acquire
		if [[ -f "$LOCK_FILE" ]]; then
			local lock_pid
			lock_pid=$(cat "$LOCK_FILE" 2>/dev/null || echo "")
			if [[ -n "$lock_pid" ]] && [[ "$lock_pid" =~ ^[0-9]+$ ]]; then
				if ! kill -0 "$lock_pid" 2>/dev/null; then
					log_with_timestamp "Removing stale lock file (PID $lock_pid no longer running)"
					rm -f "$LOCK_FILE"
				fi
			fi
		fi
		
		# Attempt to acquire lock
		exec 9>"$LOCK_FILE"
		if ! flock -n 9; then 
			# Check if the process holding the lock is still running
			local current_lock_pid
			current_lock_pid=$(cat "$LOCK_FILE" 2>/dev/null || echo "")
			if [[ -n "$current_lock_pid" ]] && [[ "$current_lock_pid" =~ ^[0-9]+$ ]]; then
				if kill -0 "$current_lock_pid" 2>/dev/null; then
					echo "Another instance is running (PID $current_lock_pid, lock held)."
				else
					echo "Stale lock detected (PID $current_lock_pid not running). Retry should succeed."
				fi
			else
				echo "Another instance appears to be running (lock held)."
			fi
			exit 1
		fi
		
		# Write our PID to the lock file
		echo $$ >&9
		
		# Register cleanup to close FD and remove lock on exit
		if declare -F error_handler::register_cleanup >/dev/null 2>&1; then
			error_handler::register_cleanup cleanup_lock_fd
		fi
	else
		log_with_timestamp "WARNING: flock not available; running without lock"
	fi
	log_with_timestamp "Starting loop (task=$LOOP_TASK)"
	
	if ! check_worker_available; then log_with_timestamp "FATAL: worker not available"; exit 1; fi
	log_with_timestamp "Writing PID $BASHPID to $PID_FILE"
	# Ensure the directory exists
	local pid_dir; pid_dir=${PID_FILE%/*}
	if [[ ! -d "$pid_dir" ]]; then
		log_with_timestamp "ERROR: PID directory does not exist: $pid_dir"
		mkdir -p "$pid_dir" || log_with_timestamp "ERROR: Failed to create PID directory"
	fi
	# Write the PID file with explicit error checking
	# Use BASHPID to get the actual PID of this shell (not inherited from parent)
	if echo "$BASHPID" > "$PID_FILE"; then
		log_with_timestamp "Successfully wrote PID file"
		if [[ -f "$PID_FILE" ]]; then
			log_with_timestamp "PID file exists with contents: $(cat "$PID_FILE")"
		else
			log_with_timestamp "ERROR: PID file was written but doesn't exist!"
		fi
	else
		log_with_timestamp "ERROR: Failed to write PID file to $PID_FILE (exit code: $?)"
	fi
	# Register cleanup for PID file
	if declare -F error_handler::register_cleanup >/dev/null 2>&1; then
		error_handler::register_cleanup cleanup_pid_file
	fi
	local i=1
	local max_iter="${RUN_MAX_ITERATIONS:-}" # optional bound via env
	while true; do
		
		log_with_timestamp "--- Iteration $i ---"
		local iter_status=0
		if run_iteration "$i"; then 
			log_with_timestamp "Iteration $i dispatched"
		else 
			iter_status=$?
			log_with_timestamp "Iteration $i skipped (status=$iter_status)"
		fi
		
		# Check and rotate logs (ignore non-zero return since it just means no rotation needed)
		check_and_rotate || true
		
		log_with_timestamp "Waiting ${INTERVAL_SECONDS}s until next iteration..."
		local remain=$INTERVAL_SECONDS
		while [[ $remain -gt 0 ]]; do 
			# Check for skip-wait flag
			if [[ -f "${DATA_DIR}/skip_wait.flag" ]]; then
				log_with_timestamp "⏭️  Skip wait flag detected - continuing immediately"
				rm -f "${DATA_DIR}/skip_wait.flag"
				break
			fi
			local chunk=$((remain>SLEEP_CHUNK_SIZE?SLEEP_CHUNK_SIZE:remain)); 
			sleep $chunk; 
			remain=$((remain-chunk)); 
		done
		if [[ -n "$max_iter" ]] && [[ $i -ge $max_iter ]]; then log_with_timestamp "Reached max iterations ($max_iter); exiting."; break; fi
		((i++))
		log_with_timestamp "DEBUG: End of iteration $((i-1)), continuing to iteration $i"
	done
	log_with_timestamp "DEBUG: Exited while loop normally"
}

# Load modular dispatcher if available
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
COMMANDS_DIR="${APP_ROOT}/auto/lib/commands"
if [[ -f "$COMMANDS_DIR/dispatcher.sh" ]]; then
	# shellcheck disable=SC1090
	source "$COMMANDS_DIR/dispatcher.sh"
	MODULAR_DISPATCHER_AVAILABLE=true
else
	MODULAR_DISPATCHER_AVAILABLE=false
fi

# -----------------------------------------------------------------------------
# Function: loop_dispatch
# Description: Main command dispatcher for loop operations
# Parameters:
#   $1 - Command to execute (run-loop, start, stop, status, etc.)
#   $@ - Additional command arguments
# Returns: Command-specific exit code
# Side Effects: Varies by command
# Commands:
#   - run-loop: Run loop in foreground
#   - start: Start loop in background
#   - stop: Gracefully stop loop
#   - force-stop: Force kill loop
#   - status: Show loop status
#   - logs: Show/follow logs
#   - rotate: Rotate log files
#   - once: Run single iteration
#   - json: Generate JSON reports
#   - health: Check system health
#   - dry-run: Test configuration
# -----------------------------------------------------------------------------
loop_dispatch() {
	local cmd="${1:-run-loop}"; shift || true
	
	# Use modular dispatcher for all commands
	if [[ "$MODULAR_DISPATCHER_AVAILABLE" == "true" ]]; then
		loop_dispatch_modular "$cmd" "$@"
		return $?
	else
		echo "ERROR: Modular dispatcher not available" >&2
		echo "Commands directory not found: $COMMANDS_DIR" >&2
		return 1
	fi
	
}