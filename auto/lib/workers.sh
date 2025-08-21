#!/usr/bin/env bash

# Workers Library - Worker management and execution
# Part of the modular loop system
#
# This module provides:
# - Worker process execution with timeout management
# - Iteration lifecycle management
# - Output capture and redaction
# - Error classification and reporting
# - Both async and sync execution modes

set -euo pipefail

# -----------------------------------------------------------------------------
# Function: run_iteration
# Description: Execute a single iteration asynchronously with full lifecycle management
# Parameters:
#   $1 - Iteration number
# Returns: 0 on successful dispatch, 1 on failure to start
# Side Effects:
#   - Creates background worker process
#   - Writes to iteration log files
#   - Emits start/finish events
#   - Updates summary files
#   - Appends worker PID to tracking file
# -----------------------------------------------------------------------------
run_iteration() {
	local iter="$1"; log_with_timestamp "Starting iteration #$iter (task=$LOOP_TASK)"
	
	# Prepare env before gating so per-task overrides (e.g., MAX_CONCURRENT_WORKERS) take effect
	if ! prepare_worker_env; then
		log_with_timestamp "ERROR: Failed to prepare worker environment for iteration #$iter"
		return 1
	fi
	
	# Use error context for better debugging
	if declare -F error_handler::set_context >/dev/null 2>&1; then
		error_handler::set_context "iteration #$iter setup"
	fi
	
	can_start_new_worker || { log_with_timestamp "ERROR: Cannot start new worker (gating or concurrency limit)"; return 1; }
	local prompt_path; prompt_path=$(select_prompt) || { log_with_timestamp "ERROR: No prompt found"; return 1; }
	[[ -r "$prompt_path" ]] || { log_with_timestamp "ERROR: Prompt not readable: $prompt_path"; return 1; }
	
	if declare -F error_handler::clear_context >/dev/null 2>&1; then
		error_handler::clear_context
	fi
	
	# Compose prompt with comprehensive error handling
	local full_prompt
	if ! full_prompt=$(compose_prompt "$prompt_path"); then
		log_with_timestamp "ERROR: Failed to compose prompt for iteration #$iter"
		return 1
	fi
	
	if [[ -z "$full_prompt" ]]; then
		log_with_timestamp "ERROR: Composed prompt is empty for iteration #$iter"
		return 1
	fi
	# Launch a wrapper subshell in background; run the pipeline in foreground within the subshell
	(
		set +e
		local start; start=$(date +%s)
		local tmp_out; tmp_out=$(mktemp -p "$TMP_DIR" "${LOOP_TASK}-${iter}-XXXX.log")
		local exitc=1  # Default to failure
		local dur=0
		
		# Ensure finish event is emitted even on early exit
		trap 'emit_finish_event "$iter" "$$" "$exitc" "$dur"; update_summary_files || true; rm -f "$tmp_out"; exit $exitc' EXIT
		
		# Run worker pipeline in foreground; capture the exit code of timeout (the first command)
		# Use more aggressive timeout with proper signal escalation
		if ! timeout --signal=TERM --kill-after="$WORKER_KILL_AFTER_SECONDS" "$TIMEOUT" resource-claude-code run "$full_prompt" 2>&1 | tee "$tmp_out"; then
			# Safely extract timeout command exit code
			if [[ ${#PIPESTATUS[@]} -gt 0 ]]; then
				exitc=${PIPESTATUS[0]}
			else
				exitc=1  # Default to general error if PIPESTATUS unavailable
			fi
			log_with_timestamp "ERROR: Worker execution failed with exit code $exitc"
		else
			# Safely extract timeout command exit code
			if [[ ${#PIPESTATUS[@]} -gt 0 ]]; then
				exitc=${PIPESTATUS[0]}
			else
				exitc=0  # Default to success if PIPESTATUS unavailable
			fi
		fi
		
		local end; end=$(date +%s); dur=$((end-start))
		
		# Enhanced error classification using shared function
		if [[ $exitc -eq 0 ]]; then
			log_with_timestamp "Iteration #$iter succeeded in ${dur}s"
		elif [[ $exitc -eq 124 ]]; then
			log_with_timestamp "Iteration #$iter timed out in ${dur}s"
		else
			# Use shared error classification function from error-codes module
			exitc=$(error_codes::classify_worker_exit "$exitc" "$tmp_out" "iteration #$iter")
			local error_desc; error_desc=$(error_codes::describe "$exitc")
			log_with_timestamp "Iteration #$iter failed ($error_desc, exit=$exitc) in ${dur}s"
		fi
		
		# Persist worker output (redacted) and append tail to main log
		if [[ -f "$tmp_out" ]]; then
			local iter_log
			iter_log="${ITERATIONS_DIR}/iter-$(printf "%05d" "$iter").log"
			head -n "$ITERATION_LOG_MAX_LINES" "$tmp_out" | redact > "$iter_log" 2>/dev/null || true
			{
				echo "===== Iteration #$iter worker output (last $ITERATION_LOG_TAIL_LINES lines, redacted) ====="
				tail -n "$ITERATION_LOG_TAIL_LINES" "$tmp_out" | redact
				echo "===== End iteration #$iter worker output ====="
			} >> "$LOG_FILE" 2>/dev/null || true
		fi
	) &
	local wrapper_pid=$!
	# Emit event first to avoid race condition where PID exists without event
	emit_start_event "$prompt_path" "$iter" "$wrapper_pid"
	append_worker_pid "$wrapper_pid"
	return 0
}

# -----------------------------------------------------------------------------
# Function: run_iteration_sync
# Description: Execute a single iteration synchronously (blocking)
# Parameters:
#   $1 - Iteration number
# Returns: Exit code from worker execution
# Side Effects:
#   - Blocks until worker completes
#   - Writes to iteration log files
#   - Emits start/finish events
#   - Updates summary files
#   - Does NOT create background process
# Usage: Used for "once" command where immediate feedback is needed
# -----------------------------------------------------------------------------
run_iteration_sync() {
	local iter="$1"; log_with_timestamp "Starting iteration (sync) #$iter (task=$LOOP_TASK)"
	# Prepare env before gating so per-task overrides (e.g., MAX_CONCURRENT_WORKERS) take effect
	prepare_worker_env
	can_start_new_worker || return 1
	local prompt_path; prompt_path=$(select_prompt) || { log_with_timestamp "ERROR: No prompt found"; return 1; }
	[[ -r "$prompt_path" ]] || { log_with_timestamp "ERROR: Prompt not readable: $prompt_path"; return 1; }
	local full_prompt; full_prompt=$(compose_prompt "$prompt_path")
	local start; start=$(date +%s)
	local tmp_out; tmp_out=$(mktemp -p "$TMP_DIR" "${LOOP_TASK}-${iter}-XXXX.log")
	local exitc=1  # Default to failure
	local dur=0
	
	set +e
	emit_start_event "$prompt_path" "$iter" "$$"
	# Use more aggressive timeout with proper signal escalation
	if ! timeout --signal=TERM --kill-after="$WORKER_KILL_AFTER_SECONDS" "$TIMEOUT" resource-claude-code run "$full_prompt" 2>&1 | tee "$tmp_out"; then
		exitc=${PIPESTATUS[0]}
		log_with_timestamp "ERROR: Worker execution failed with exit code $exitc"
	else
		exitc=${PIPESTATUS[0]}
	fi
	set -e
	
	local end; end=$(date +%s); dur=$((end-start))
	
	# Enhanced error classification using shared function
	if [[ $exitc -eq 0 ]]; then
		log_with_timestamp "Iteration #$iter succeeded in ${dur}s"
	elif [[ $exitc -eq 124 ]]; then
		log_with_timestamp "Iteration #$iter timed out in ${dur}s"
	else
		# Use shared error classification function from error-codes module
		exitc=$(error_codes::classify_worker_exit "$exitc" "$tmp_out" "iteration #$iter")
		local error_desc; error_desc=$(error_codes::describe "$exitc")
		log_with_timestamp "Iteration #$iter failed ($error_desc, exit=$exitc) in ${dur}s"
	fi
	
	# Persist worker output (redacted) and append tail to main log
	if [[ -f "$tmp_out" ]]; then
		local iter_log
		iter_log="${ITERATIONS_DIR}/iter-$(printf "%05d" "$iter").log"
		head -n "$ITERATION_LOG_MAX_LINES" "$tmp_out" | redact > "$iter_log" 2>/dev/null || true
		{
			echo "===== Iteration #$iter worker output (last $ITERATION_LOG_TAIL_LINES lines, redacted) ====="
			tail -n "$ITERATION_LOG_TAIL_LINES" "$tmp_out" | redact
			echo "===== End iteration #$iter worker output ====="
		} >> "$LOG_FILE" 2>/dev/null || true
	fi
	emit_finish_event "$iter" "$$" "$exitc" "$dur"
	update_summary_files || true
	rm -f "$tmp_out"
	return "$exitc"
}