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
# Description: Close the lock file descriptor
# Parameters: None
# Returns: 0
# Side Effects: Closes file descriptor 9 if open
# -----------------------------------------------------------------------------
cleanup_lock_fd() {
	# Close FD 9 if it's open
	if [[ -e /proc/$$/fd/9 ]]; then
		exec 9>&-
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
	if [[ -f "$PID_FILE" ]]; then
		local stored_pid
		stored_pid=$(cat "$PID_FILE" 2>/dev/null || echo "")
		if [[ "$stored_pid" == "$$" ]]; then
			rm -f "$PID_FILE"
		fi
	fi
	return 0
}

run_loop() {
	# Lock to prevent duplicate instances
	if command -v flock >/dev/null 2>&1; then
		exec 9>"$LOCK_FILE"
		if ! flock -n 9; then echo "Another instance appears to be running (lock held)."; exit 1; fi
		# Register cleanup to close FD on exit
		if declare -F error_handler::register_cleanup >/dev/null 2>&1; then
			error_handler::register_cleanup cleanup_lock_fd
		fi
	else
		log_with_timestamp "WARNING: flock not available; running without lock"
	fi
	log_with_timestamp "Starting loop (task=$LOOP_TASK)"
	
	if ! check_worker_available; then log_with_timestamp "FATAL: worker not available"; exit 1; fi
	echo $$ > "$PID_FILE"
	# Register cleanup for PID file
	if declare -F error_handler::register_cleanup >/dev/null 2>&1; then
		error_handler::register_cleanup cleanup_pid_file
	fi
	local i=1
	local max_iter="${RUN_MAX_ITERATIONS:-}" # optional bound via env
	while true; do
		
		log_with_timestamp "--- Iteration $i ---"
		if run_iteration "$i"; then log_with_timestamp "Iteration $i dispatched"; else log_with_timestamp "Iteration $i skipped"; fi
		check_and_rotate
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
	done
}

# Load modular dispatcher if available
COMMANDS_DIR="${LIB_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}/commands"
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
	
	# Try modular dispatcher first for supported commands
	if [[ "$MODULAR_DISPATCHER_AVAILABLE" == "true" ]]; then
		# List of commands that have been modularized
		case "$cmd" in
			run-loop|start|stop|force-stop|status|logs|rotate)
				loop_dispatch_modular "$cmd" "$@"
				return $?
				;;
		esac
	fi
	
	# Fallback to original monolithic dispatcher
	case "$cmd" in
		run-loop)
			local maxflag="${1:-}"; local maxval=""; if [[ "$maxflag" == "--max" ]]; then maxval="${2:-}"; shift 2 || true; export RUN_MAX_ITERATIONS="$maxval"; fi
			run_loop
			;;
		start) 
			# Need to use the actual task-manager path, not $0 which could be wrong in sourced context
			local task_manager="${AUTO_DIR}/task-manager.sh"
			nohup "$task_manager" --task "$LOOP_TASK" run-loop >/dev/null 2>&1 & 
			local start_pid=$!
			echo "Started loop (task=$LOOP_TASK) PID: $start_pid"
			# Wait briefly for the actual loop to start and write its PID
			local wait_count=0
			while [[ ! -f "$PID_FILE" && $wait_count -lt $PID_FILE_WAIT_ITERATIONS ]]; do
				sleep 0.1
				((wait_count++))
			done
			if [[ -f "$PID_FILE" ]]; then
				local actual_pid
				actual_pid=$(cat "$PID_FILE")
				echo "Loop running with PID: $actual_pid"
			fi
			;;
		stop)
			if [[ -f "$PID_FILE" ]]; then
				local main_pid; main_pid=$(cat "$PID_FILE")
				if kill -0 "$main_pid" 2>/dev/null; then
					kill_tree "$main_pid" TERM
					local w=0; while kill -0 "$main_pid" 2>/dev/null && [[ $w -lt $PROCESS_TERMINATION_WAIT ]]; do sleep 1; ((w++)); done
					if kill -0 "$main_pid" 2>/dev/null; then kill_tree "$main_pid" KILL; fi
				fi
			fi
			# Kill any tracked workers
			if [[ -f "$PIDS_FILE" ]]; then while IFS= read -r wp; do kill_tree "$wp" TERM; done < "$PIDS_FILE"; rm -f "$PIDS_FILE"; fi
			rm -f "$PID_FILE" "$LOCK_FILE"; echo "Stopped (task=$LOOP_TASK)" ;;
		force-stop)
			if [[ -f "$PID_FILE" ]]; then main_pid=$(cat "$PID_FILE"); kill_tree "$main_pid" KILL; fi
			if [[ -f "$PIDS_FILE" ]]; then while IFS= read -r wp; do kill_tree "$wp" KILL; done < "$PIDS_FILE"; rm -f "$PIDS_FILE"; fi
			rm -f "$PID_FILE" "$LOCK_FILE"; echo "Force-stopped (task=$LOOP_TASK)" ;;
		status)
			local st="STOPPED"; local mp=""; if [[ -f "$PID_FILE" ]]; then mp=$(cat "$PID_FILE"); if kill -0 "$mp" 2>/dev/null; then st="RUNNING (PID: $mp)"; else st="STOPPED (stale PID file)"; fi; fi
			echo "$st"; echo "Task: $LOOP_TASK"; local pr; pr=$(select_prompt || echo "(not found)"); echo "Prompt: $pr"; if [[ -f "$pr" ]]; then echo "Prompt SHA: $(sha256sum "$pr" | awk '{print $1}')"; fi; echo "Log: $LOG_FILE"; echo "Events: $EVENTS_JSONL";
			;;
		logs) if [[ "${1:-}" == "-f" || "${1:-}" == "follow" ]]; then tail -f "$LOG_FILE"; else cat "$LOG_FILE"; fi ;;
		rotate)
			# log rotation
			if [[ -f "$LOG_FILE" ]]; then ts=$(date '+%Y%m%d_%H%M%S'); mv "$LOG_FILE" "${LOG_FILE}.${ts}"; echo "Rotated log to ${LOG_FILE}.${ts}"; else echo "No log to rotate"; fi
			# optional events rotation
			if [[ "${1:-}" == "--events" ]]; then
				local keep=${2:-$ROTATE_KEEP}
				if [[ -f "$EVENTS_JSONL" ]]; then
					ts=$(date '+%Y%m%d_%H%M%S'); mv -- "$EVENTS_JSONL" "${EVENTS_JSONL}.${ts}"
					# prune older events safely
					find "$(dirname "$EVENTS_JSONL")" -maxdepth 1 -type f -name "$(basename "$EVENTS_JSONL").*" -printf '%T@ %p\n' 2>/dev/null | sort -nr | awk -v k="$keep" 'NR>(k+1){print $2}' | xargs -r rm -f --
				else
					echo "No events file to rotate"
				fi
			fi
			# optional temp cleanup
			if [[ "${1:-}" == "--temp" ]]; then
				if [[ -d "$TMP_DIR" ]]; then
					local cleaned=0
					cleaned=$(find "$TMP_DIR" -name "tmp.*" -size 0 -delete -print 2>/dev/null | wc -l)
					echo "Cleaned up $cleaned empty temporary files"
				else
					echo "No temp directory to clean"
				fi
			fi
			;;
		sudo-init)
			if command -v sudo_override::init >/dev/null 2>&1; then
				local commands="${1:-}"
				if sudo_override::init "$commands"; then
					echo "✅ Sudo override initialized successfully"
				else
					echo "❌ Failed to initialize sudo override"
					exit 1
				fi
			else
				echo "❌ Sudo override not available"
				exit 1
			fi
			;;
		sudo-test)
			if command -v sudo_override::test >/dev/null 2>&1; then
				if sudo_override::test; then
					echo "✅ Sudo override test passed"
				else
					echo "❌ Sudo override test failed"
					exit 1
				fi
			else
				echo "❌ Sudo override not available"
				exit 1
			fi
			;;
		sudo-status)
			if command -v sudo_override::status >/dev/null 2>&1; then
				sudo_override::status
			else
				echo "❌ Sudo override not available"
				exit 1
			fi
			;;
		sudo-cleanup)
			if command -v sudo_override::cleanup >/dev/null 2>&1; then
				if sudo_override::cleanup; then
					echo "✅ Sudo override cleaned up successfully"
				else
					echo "❌ Failed to clean up sudo override"
					exit 1
				fi
			else
				echo "❌ Sudo override not available"
				exit 1
			fi
			;;
		json)
			process_json_command "$@"
			;;
		dry-run)
			local prompt; prompt=$(select_prompt || true)
			if [[ -z "${prompt:-}" ]]; then echo "No prompt found"; exit 1; fi
			local composed; composed=$(compose_prompt "$prompt")
			echo "Task: $LOOP_TASK"; echo "Prompt: $prompt"; echo "Prompt SHA: $(sha256sum "$prompt" | awk '{print $1}')"; echo "Prompt bytes: $(wc -c < "$prompt" 2>/dev/null || echo 0)";
			echo "Config: INTERVAL_SECONDS=$INTERVAL_SECONDS MAX_TURNS=$MAX_TURNS TIMEOUT=$TIMEOUT MAX_CONCURRENT_WORKERS=$MAX_CONCURRENT_WORKERS"
			echo "Composed prompt preview (first 20 lines, redacted):"; echo "$composed" | head -n 20 | redact
			;;
		health)
			local ok=0
			echo "=== Basic Binary Checks ==="
			for c in timeout vrooli resource-claude-code; do 
				if ! command -v "$c" >/dev/null 2>&1; then 
					echo "❌ missing: $c"; ok=1
				else
					echo "✅ found: $c"
				fi
			done
			
			echo "=== File System Checks ==="
			if [[ ! -w "$DATA_DIR" ]]; then 
				echo "❌ data dir not writable: $DATA_DIR"; ok=1
			else
				echo "✅ data dir writable: $DATA_DIR"
			fi
			
			# Check disk space (require at least 100MB free)
			local free_space; free_space=$(df "$DATA_DIR" | awk 'NR==2 {print $4}' 2>/dev/null || echo "0")
			if [[ "$free_space" -lt 102400 ]]; then  # 100MB in KB
				echo "❌ low disk space: ${free_space}KB free"; ok=1
			else
				echo "✅ disk space ok: ${free_space}KB free"
			fi
			
			echo "=== Prompt & Configuration ==="
			local prompt; prompt=$(select_prompt || true)
			if [[ -z "${prompt:-}" ]]; then 
				echo "❌ no prompt found"; ok=1
			else 
				echo "✅ prompt ok: $prompt"
				# Check prompt file size
				local prompt_size; prompt_size=$(wc -c < "$prompt" 2>/dev/null || echo "0")
				if [[ "$prompt_size" -lt 100 ]]; then
					echo "❌ prompt file too small: ${prompt_size} bytes"; ok=1
				else
					echo "✅ prompt size ok: ${prompt_size} bytes"
				fi
			fi
			
			echo "=== Resource Availability ==="
			# Check if jq is available (needed for JSON processing)
			if ! command -v jq >/dev/null 2>&1; then
				echo "❌ missing: jq (required for JSON processing)"; ok=1
			else
				echo "✅ found: jq"
			fi
			
			# Check Ollama availability if resource-ollama exists
			if command -v resource-ollama >/dev/null 2>&1; then
				if resource-ollama info >/dev/null 2>&1; then
					echo "✅ ollama available"
				else
					echo "⚠️  ollama installed but not responding"
				fi
			else
				echo "ℹ️  ollama not available (optional)"
			fi
			
			echo "=== Task-specific Checks ==="
			if declare -F task_check_worker_available >/dev/null 2>&1; then
				if task_check_worker_available; then
					echo "✅ task worker available"
				else
					echo "❌ task worker unavailable"; ok=1
				fi
			else
				echo "ℹ️  no task-specific worker checks defined"
			fi
			
			if [[ $ok -eq 0 ]]; then
				echo "=== Overall Status ==="
				echo "✅ All health checks passed"
				return 0
			else
				echo "=== Overall Status ==="
				echo "❌ Some health checks failed"
				return 1
			fi
			;;
		once)
			check_worker_available || exit 1
			run_iteration_sync 1; exitcode=$?; echo "Exit code: $exitcode"; exit $exitcode ;;
		skip-wait)
			touch "${DATA_DIR}/skip_wait.flag"
			echo "⏭️  Skip wait flag set - next iteration will skip wait interval"
			;;
		*) echo "ERROR: Unknown command: $cmd"; echo "Use 'health' to check configuration or see task-manager.sh help"; exit 1 ;;
	esac
}