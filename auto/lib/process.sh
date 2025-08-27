#!/usr/bin/env bash

# Process Management Library - Process control and cleanup
# Part of the modular loop system

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
LIB_DIR="${APP_ROOT}/auto/lib"
# shellcheck disable=SC1091
source "$LIB_DIR/file-rotation.sh"
# shellcheck disable=SC1091
source "$LIB_DIR/network-utils.sh"

# -----------------------------------------------------------------------------
# Function: kill_tree
# Description: Kill a process and all its children/descendants
# Parameters:
#   $1 - Process ID to kill
#   $2 - Signal to send (default: TERM)
# Returns: 0 (always succeeds, errors are ignored)
# Side Effects: Terminates process tree
# Dependencies: pkill (optional for enhanced cleanup)
# -----------------------------------------------------------------------------
kill_tree() {
	local pid="$1" sig="${2:-TERM}"
	# Try to kill the entire process group first (more robust for pipelines)
	local pgid
	pgid=$(ps -o pgid= -p "$pid" 2>/dev/null | tr -d ' ' || true)
	if [[ -n "${pgid:-}" ]]; then
		kill -s "$sig" -- -"$pgid" 2>/dev/null || true
	fi
	# Also signal the specific PID
	kill -s "$sig" "$pid" 2>/dev/null || true
	# Kill children by parent relationship as a fallback
	if command -v pkill >/dev/null 2>&1; then
		pkill -"$sig" -P "$pid" 2>/dev/null || true
	fi
}

# -----------------------------------------------------------------------------
# Function: cleanup
# Description: Clean up all workers and temporary files on exit
# Parameters: 
#   $1 - Signal name (optional)
# Returns: Exits with 0
# Side Effects: Kills all tracked workers, removes PID files
# Usage: Registered as trap handler for SIGTERM and SIGINT
# -----------------------------------------------------------------------------
cleanup() {
	local signal="${1:-}"
	
	# Check if this is a user-initiated interrupt or a genuine termination request
	# If we're in the middle of a worker execution and the main loop PID matches,
	# this is likely a user interrupt, not a worker completion
	local is_user_interrupt=false
	
	# Check if the signal is coming from the terminal (user interrupt)
	# or if it's a genuine termination request
	if [[ "$signal" == "INT" ]]; then
		# SIGINT is typically from Ctrl+C - user interrupt
		is_user_interrupt=true
	elif [[ "$signal" == "TERM" ]]; then
		# Check if we have active workers - if not, this might be from a completed worker
		if [[ -f "$PIDS_FILE" ]]; then
			local active_workers=0
			while IFS= read -r pid; do
				if [[ -n "${pid:-}" ]] && kill -0 "$pid" 2>/dev/null; then
					((active_workers++))
				fi
			done < "$PIDS_FILE"
			
			# If we have active workers and get SIGTERM, it's likely a real termination
			if [[ $active_workers -gt 0 ]]; then
				is_user_interrupt=true
			fi
		fi
	fi
	
	# Only proceed with cleanup if this is a user interrupt or genuine termination
	if [[ "$is_user_interrupt" == "true" ]]; then
		log_with_timestamp "Cleanup requested (task=${LOOP_TASK}, signal=${signal:-unknown})"
		# Terminate tracked workers
		if [[ -f "$PIDS_FILE" ]]; then
			while IFS= read -r pid; do
				[[ -n "${pid:-}" ]] && kill_tree "$pid" TERM
			done < "$PIDS_FILE"
			rm -f "$PIDS_FILE"
		fi
		rm -f "$PID_FILE"
		exit 0
	else
		# This is likely from a worker completion - don't exit the main loop
		log_with_timestamp "Signal received (${signal:-unknown}) but continuing loop operation"
		return 0
	fi
}

# Modified trap handlers to pass signal name
trap 'cleanup INT' SIGINT
trap 'cleanup TERM' SIGTERM

# -----------------------------------------------------------------------------
# Function: cleanup_finished_workers
# Description: Remove finished workers from the PID tracking file
# Parameters: None
# Returns: 0 on success
# Side Effects: Updates PIDS_FILE to contain only running workers
# -----------------------------------------------------------------------------
cleanup_finished_workers() {
	if [[ -f "$PIDS_FILE" ]]; then
		local tmp; tmp=$(mktemp -p "$TMP_DIR")
		local success=true
		
		# Process PIDs with error handling
		while IFS= read -r pid; do
			if [[ -n "$pid" ]]; then
				if kill -0 "$pid" 2>/dev/null; then 
					if ! echo "$pid" >> "$tmp" 2>/dev/null; then
						success=false
						break
					fi
				else 
					log_with_timestamp "Worker $pid finished"
				fi
			fi
		done < "$PIDS_FILE"
		
		# Atomic replacement with validation
		if [[ "$success" == "true" ]]; then
			if ! mv "$tmp" "$PIDS_FILE" 2>/dev/null; then
				log_with_timestamp "ERROR: Failed to update workers PID file"
				rm -f "$tmp"
				return 1
			fi
		else
			log_with_timestamp "ERROR: Failed to process workers PID file"
			rm -f "$tmp"
			return 1
		fi
	fi
	return 0
}

# -----------------------------------------------------------------------------
# Function: current_tcp_connections
# Description: Count current TCP connections (wrapper for backward compatibility)
# Parameters: None (uses LOOP_TCP_FILTER environment variable)
# Returns: Number of matching TCP connections
# Side Effects: None
# Dependencies: network_utils module
# -----------------------------------------------------------------------------
current_tcp_connections() {
	network_utils::count_worker_processes "${LOOP_TCP_FILTER:-}"
}

# -----------------------------------------------------------------------------
# Function: can_start_new_worker
# Description: Check if a new worker can be started based on concurrency and network limits
# Parameters: None
# Returns: 0 if worker can start, 1 if limits exceeded
# Side Effects: Cleans up finished workers from tracking file
# Environment Variables:
#   - MAX_CONCURRENT_WORKERS: Maximum concurrent worker processes
#   - MAX_TCP_CONNECTIONS: Maximum TCP connections (network gating)
#   - SCENARIO: Skip network check if "plan"
# -----------------------------------------------------------------------------
can_start_new_worker() {
	cleanup_finished_workers
	
	# Check concurrent worker limit
	local running=0
	[[ -f "$PIDS_FILE" ]] && running=$(wc -l < "$PIDS_FILE")
	if [[ $running -ge $MAX_CONCURRENT_WORKERS ]]; then 
		log_with_timestamp "Max concurrent workers reached ($MAX_CONCURRENT_WORKERS)"
		return 1
	fi
	
	# Check TCP connection limit using network utils
	if ! network_utils::check_connection_limit "$MAX_TCP_CONNECTIONS" "${LOOP_TCP_FILTER:-}"; then
		return 1
	fi
	
	return 0
}

# -----------------------------------------------------------------------------
# Function: append_worker_pid
# Description: Atomically append a worker PID to the tracking file
# Parameters:
#   $1 - Process ID to track
# Returns: 0 on success
# Side Effects: Appends PID to PIDS_FILE
# Dependencies: flock (optional for atomic append)
# -----------------------------------------------------------------------------
append_worker_pid() {
	local pid="$1"
	if command -v flock >/dev/null 2>&1; then
		# Open file descriptor and use flock properly
		(
			exec 201>>"$PIDS_FILE"
			if flock -x 201; then
				printf '%s\n' "$pid" >&201
			else
				# Fallback to simple append if lock fails
				printf '%s\n' "$pid" >> "$PIDS_FILE"
			fi
		)
	else
		printf '%s\n' "$pid" >> "$PIDS_FILE"
	fi
}

# -----------------------------------------------------------------------------
# Function: check_and_rotate
# Description: Check and rotate log files if they exceed size limits
# Parameters: None
# Returns: 0 on success
# Side Effects: May rotate log files, clean temporary files
# Dependencies: file_rotation module
# Environment Variables:
#   - LOG_FILE: Main log file path
#   - EVENTS_JSONL: Events log file path
#   - TMP_DIR: Temporary directory path
#   - ITERATIONS_DIR: Iteration logs directory path
# -----------------------------------------------------------------------------
check_and_rotate() {
	# Use the comprehensive rotation check from file-rotation module
	file_rotation::check_and_rotate_all \
		"${LOG_FILE:-}" \
		"${EVENTS_JSONL:-}" \
		"${TMP_DIR:-}" \
		"${ITERATIONS_DIR:-}"
}