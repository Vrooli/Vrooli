#!/usr/bin/env bash

# Process Management Library - Process control and cleanup
# Part of the modular loop system

set -euo pipefail

# Source required modules
LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
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
		pkill -$sig -P "$pid" 2>/dev/null || true
	fi
}

# -----------------------------------------------------------------------------
# Function: cleanup
# Description: Clean up all workers and temporary files on exit
# Parameters: None
# Returns: Exits with 0
# Side Effects: Kills all tracked workers, removes PID files
# Usage: Registered as trap handler for SIGTERM and SIGINT
# -----------------------------------------------------------------------------
cleanup() {
	log_with_timestamp "Cleanup requested (task=${LOOP_TASK})"
	# Terminate tracked workers
	if [[ -f "$PIDS_FILE" ]]; then
		while IFS= read -r pid; do
			[[ -n "${pid:-}" ]] && kill_tree "$pid" TERM
		done < "$PIDS_FILE"
		rm -f "$PIDS_FILE"
	fi
	rm -f "$PID_FILE"
	exit 0
}
trap cleanup SIGTERM SIGINT

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
		while IFS= read -r pid; do
			if kill -0 "$pid" 2>/dev/null; then echo "$pid" >> "$tmp"; else log_with_timestamp "Worker $pid finished"; fi
		done < "$PIDS_FILE"
		mv "$tmp" "$PIDS_FILE"
	fi
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
#   - RESOURCE_IMPROVEMENT_MODE: Skip network check if "plan"
#   - SCENARIO_IMPROVEMENT_MODE: Skip network check if "plan"
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
	
	# Skip network gating for plan-only modes
	if [[ "${RESOURCE_IMPROVEMENT_MODE:-}" == "plan" || "${SCENARIO_IMPROVEMENT_MODE:-}" == "plan" ]]; then
		return 0
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
		{
			flock -x $FD_PIDS_LOCK
			printf '%s\n' "$pid" >&$FD_PIDS_LOCK
		} $FD_PIDS_LOCK>>"$PIDS_FILE"
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