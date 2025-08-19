#!/usr/bin/env bash

# Generic Loop Core Library
# Provides reusable loop orchestration for Claude-code style tasks.
# Expected to be sourced by a task script that sets at minimum:
# - LOOP_TASK: short task name (e.g., "scenario-improvement")
# Optional task hooks (functions):
# - task_prompt_candidates (echo candidates separated by newlines)
# - task_build_helper_context (echo helper context string)
# - task_build_prompt "base_prompt" "summary" "helper" (echo final prompt)
# - task_check_worker_available (return 0/1)
# - task_prepare_worker_env (export env vars)
# - task_run_worker "full_prompt" "iteration" (execute worker)

set -euo pipefail

# Validate required vars
if [[ -z "${LOOP_TASK:-}" ]]; then
	echo "FATAL: LOOP_TASK is not set for loop core" >&2
	exit 1
fi

# Directories
AUTO_DIR="${AUTO_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)}"
DATA_DIR="${DATA_DIR:-${AUTO_DIR}/data/${LOOP_TASK}}"
mkdir -p "$DATA_DIR" 2>/dev/null || true
TMP_DIR="${DATA_DIR}/tmp"
mkdir -p "$TMP_DIR" 2>/dev/null || true
# Per-iteration logs directory (for persisted worker outputs)
ITERATIONS_DIR="${ITERATIONS_DIR:-${DATA_DIR}/iterations}"
mkdir -p "$ITERATIONS_DIR" 2>/dev/null || true

# Files
LOG_FILE="${LOG_FILE:-${DATA_DIR}/loop.log}"
PID_FILE="${PID_FILE:-${DATA_DIR}/loop.pid}"
PIDS_FILE="${PIDS_FILE:-${DATA_DIR}/workers.pids}"
EVENTS_JSONL="${EVENTS_JSONL:-${DATA_DIR}/events.ndjson}"
SUMMARY_JSON="${SUMMARY_JSON:-${DATA_DIR}/summary.json}"
SUMMARY_TXT="${SUMMARY_TXT:-${DATA_DIR}/summary.txt}"
LOCK_FILE="${LOCK_FILE:-${DATA_DIR}/loop.lock}"

# Config (env-overridable)
INTERVAL_SECONDS=${INTERVAL_SECONDS:-300}
MAX_TURNS=${MAX_TURNS:-25}
TIMEOUT=${TIMEOUT:-1800}
MAX_CONCURRENT_WORKERS=${MAX_CONCURRENT_WORKERS:-3}  # Reduced from 5 to prevent TCP accumulation
MAX_TCP_CONNECTIONS=${MAX_TCP_CONNECTIONS:-15}
LOOP_TCP_FILTER="${LOOP_TCP_FILTER:-claude|anthropic|resource-claude-code}"  # Narrow default filter; exclude generic 443 to avoid false positives
OLLAMA_SUMMARY_MODEL="${OLLAMA_SUMMARY_MODEL:-llama3.2:3b}"
ULTRA_THINK_PREFIX="${ULTRA_THINK_PREFIX:-Ultra think. }"
ALLOWED_TOOLS="${ALLOWED_TOOLS:-Read,Write,Edit,Bash,LS,Glob,Grep}"
SKIP_PERMISSIONS="${SKIP_PERMISSIONS:-yes}"
PROMPT_PATH="${PROMPT_PATH:-${SCENARIO_PROMPT_PATH:-}}"
ROTATE_KEEP=${ROTATE_KEEP:-5}
LOG_MAX_BYTES=${LOG_MAX_BYTES:-52428800} # 50MB
EVENTS_MAX_LINES=${EVENTS_MAX_LINES:-200000}
# Iteration log limits
ITERATION_LOG_MAX_LINES=${ITERATION_LOG_MAX_LINES:-10000}
ITERATION_LOG_TAIL_LINES=${ITERATION_LOG_TAIL_LINES:-200}

# Error classification exit codes
API_QUOTA_EXHAUSTED=142
CONFIGURATION_ERROR=143
WORKER_UNAVAILABLE=144
TIMEOUT_ERROR=124  # Already used by timeout command

# Redact helper for log sanitization
redact() {
	sed -E 's/([A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,})/[redacted-email]/g; s/(api|token|secret|password|passwd|key)=[^ ]+/\1=[REDACTED]/ig'
}

log_with_timestamp() {
	echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# Atomic append (fallback if flock unavailable)
emit_event_jsonl() {
	if command -v flock >/dev/null 2>&1; then
		{
			flock -x 200
			printf '%s\n' "$1" >&200
		} 200>>"$EVENTS_JSONL"
	else
		printf '%s\n' "$1" >> "$EVENTS_JSONL"
	fi
}

# Summaries from events
update_summary_files() {
	[[ -f "$EVENTS_JSONL" ]] || return 0
	local tmp_input; tmp_input=$(mktemp -p "$TMP_DIR")
	tail -n 2000 "$EVENTS_JSONL" > "$tmp_input" 2>/dev/null || cp "$EVENTS_JSONL" "$tmp_input" 2>/dev/null || true
	if command -v jq >/dev/null 2>&1; then
		local tmp_summary; tmp_summary=$(mktemp -p "$TMP_DIR")
		jq -s '
		  def perc(p): (length as $n | if $n==0 then null else (sort | .[((($n-1)*p/100)|floor)]) end);
		  def finishes: map(select(.type=="finish"));
		  def inflight_count:
		    (group_by(.pid) | map({pid:.[0].pid, s:(map(select(.type=="start"))|length), f:(map(select(.type=="finish"))|length)}) | map(select(.s > .f)) | length);
		  (. // []) as $events |
		  ($events | finishes) as $fin |
		  ($fin | map(.duration_sec) | map(select(type=="number"))) as $durs |
		  ($fin | map(select(.exit_code==0)) | length) as $ok |
		  ($fin | map(select(.exit_code==124)) | length) as $to |
		  ($fin | map(select(.exit_code==142)) | length) as $quota |
		  ($fin | map(select(.exit_code==143)) | length) as $config |
		  ($fin | map(select(.exit_code==144)) | length) as $worker |
		  ($fin | map(select(.exit_code!=0 and .exit_code!=124 and .exit_code!=142 and .exit_code!=143 and .exit_code!=144)) | length) as $other |
		  ($fin | length) as $runs |
		  {
		    task: ("'""${LOOP_TASK}""'"),
		    generated_at: (now | todateiso8601),
		    totals: { runs: $runs, ok: $ok, timeout: $to, quota_exhausted: $quota, config_error: $config, worker_error: $worker, other_failures: $other },
		    success_rate: (if $runs==0 then 0 else ($ok / $runs) end),
		    duration_sec: {
		      avg: (if ($durs|length)==0 then null else ($durs|add/($durs|length)) end),
		      p50: ($durs|perc(50)), p90: ($durs|perc(90)), p95: ($durs|perc(95)), p99: ($durs|perc(99)),
		      min: (if ($durs|length)==0 then null else ($durs|min) end),
		      max: (if ($durs|length)==0 then null else ($durs|max) end)
		    },
		    inflight_count: ($events | inflight_count),
		    recent: ($fin | sort_by(.ts) | reverse | .[:10]
		             | map({ts, iter:(.iteration), status:(if .exit_code==0 then "ok" else (if .exit_code==124 then "timeout" else "fail" end) end), exit:(.exit_code), dur:(.duration_sec)})),
		    last_error: ($fin | map(select(.exit_code!=0)) | sort_by(.ts) | last | if .==null then null else {ts, iter:(.iteration), exit:(.exit_code)} end)
		  }' "$tmp_input" > "$tmp_summary" 2>/dev/null || true
		[[ -s "$tmp_summary" ]] && mv "$tmp_summary" "$SUMMARY_JSON" || printf '{"task":"%s","totals":{"runs":0}}' "$LOOP_TASK" > "$SUMMARY_JSON"
	else
		printf '{"task":"%s","note":"jq not available"}' "$LOOP_TASK" > "$SUMMARY_JSON"
	fi
	# Optional NL summary via Ollama
	if command -v resource-ollama >/dev/null 2>&1 && [[ -s "$SUMMARY_JSON" ]]; then
		# Quick health/model check; skip if not healthy
		if resource-ollama info >/dev/null 2>&1 || resource-ollama list-models >/dev/null 2>&1; then
			local p; p=$(mktemp -p "$TMP_DIR")
			{
				printf "Summarize this task metrics in 5-10 lines. Be concise.\nJSON:\n"; cat "$SUMMARY_JSON"
			} > "$p"
			if OUT=$(resource-ollama generate --from-file "$p" "$OLLAMA_SUMMARY_MODEL" 2>/dev/null); then
				printf '%s\n' "$OUT" > "$SUMMARY_TXT"
			fi
			rm -f "$p"
		fi
	fi
	rm -f "$tmp_input" 2>/dev/null || true
}

# Kill process tree helper
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

cleanup_finished_workers() {
	if [[ -f "$PIDS_FILE" ]]; then
		local tmp; tmp=$(mktemp -p "$TMP_DIR")
		while IFS= read -r pid; do
			if kill -0 "$pid" 2>/dev/null; then echo "$pid" >> "$tmp"; else log_with_timestamp "Worker $pid finished"; fi
		done < "$PIDS_FILE"
		mv "$tmp" "$PIDS_FILE"
	fi
}

# Clean up orphaned events (start events without matching finish events)
cleanup_orphaned_events() {
	if [[ -f "$EVENTS_JSONL" ]]; then
		local tmp_file; tmp_file=$(mktemp -p "$TMP_DIR")
		# Simple approach: remove all start events since they're all orphaned
		if command -v jq >/dev/null 2>&1; then
			# Keep only non-start events (finish events and other events)
			jq -c 'map(select(.type != "start"))' "$EVENTS_JSONL" > "$tmp_file" 2>/dev/null || cp "$EVENTS_JSONL" "$tmp_file"
		else
			# Fallback: simple cleanup without jq - just remove start events
			grep -v '"type":"start"' "$EVENTS_JSONL" > "$tmp_file" 2>/dev/null || cp "$EVENTS_JSONL" "$tmp_file"
		fi
		mv "$tmp_file" "$EVENTS_JSONL"
		log_with_timestamp "Cleaned up orphaned events"
	fi
}

current_tcp_connections() {
	# Process-aware gating: count matching worker processes rather than raw TCP lines
	# Allow disable by empty filter
	if [[ -z "${LOOP_TCP_FILTER}" ]]; then echo 0; return 0; fi
	# Prefer pgrep for process counting with more precise filtering
	if command -v pgrep >/dev/null 2>&1; then
		local cnt
		# Count only actual claude processes, not bash processes containing "claude"
		cnt=$(pgrep -f "^claude" 2>/dev/null | wc -l | awk '{print $1}')
		# Also count resource-claude-code processes specifically
		local resource_cnt
		resource_cnt=$(pgrep -f "resource-claude-code" 2>/dev/null | wc -l | awk '{print $1}')
		echo $((cnt + resource_cnt))
		return 0
	fi
	# Fallback: ss with process details if available
	if command -v ss >/dev/null 2>&1; then
		ss -ptn 2>/dev/null | grep -E "(${LOOP_TCP_FILTER})" | wc -l
	elif command -v netstat >/dev/null 2>&1; then
		netstat -tanp 2>/dev/null | grep -E "(${LOOP_TCP_FILTER})" | wc -l
	else
		echo 0
	fi
}

can_start_new_worker() {
	cleanup_finished_workers
	local running=0; [[ -f "$PIDS_FILE" ]] && running=$(wc -l < "$PIDS_FILE")
	if [[ $running -ge $MAX_CONCURRENT_WORKERS ]]; then log_with_timestamp "Max concurrent workers reached ($MAX_CONCURRENT_WORKERS)"; return 1; fi
	# Disable gating for explicit plan-only modes
	if [[ "${RESOURCE_IMPROVEMENT_MODE:-}" == "plan" || "${SCENARIO_IMPROVEMENT_MODE:-}" == "plan" ]]; then
		return 0
	fi
	local c; c=$(current_tcp_connections || echo 0)
	if [[ $c -gt $MAX_TCP_CONNECTIONS ]]; then log_with_timestamp "Too many worker processes matched filter ($c > $MAX_TCP_CONNECTIONS)"; return 1; fi
	return 0
}

select_prompt() {
	local p="$PROMPT_PATH"
	if [[ -n "$p" && -f "$p" ]]; then echo "$p"; return 0; fi
	if declare -F task_prompt_candidates >/dev/null 2>&1; then
		while IFS= read -r cand; do [[ -f "$cand" ]] && echo "$cand" && return 0; done < <(task_prompt_candidates)
	fi
	return 1
}

check_worker_available() {
	local missing=()
	command -v timeout >/dev/null 2>&1 || missing+=(timeout)
	command -v resource-claude-code >/dev/null 2>&1 || missing+=(resource-claude-code)
	if declare -F task_check_worker_available >/dev/null 2>&1; then
		if ! task_check_worker_available; then missing+=(worker); fi
	fi
	if ((${#missing[@]})); then
		log_with_timestamp "WARNING: Missing tools: ${missing[*]}"
		return 1
	fi
	return 0
}

prepare_worker_env() {
	export MAX_TURNS TIMEOUT ALLOWED_TOOLS SKIP_PERMISSIONS EVENTS_JSONL # expose to tasks
	if declare -F task_prepare_worker_env >/dev/null 2>&1; then task_prepare_worker_env; fi
}

compose_prompt() {
	local base; base=$(cat "$1")
	local summary=""; [[ -f "$SUMMARY_TXT" ]] && summary=$(cat "$SUMMARY_TXT")
	local helper=""; if declare -F task_build_helper_context >/dev/null 2>&1; then helper=$(task_build_helper_context); fi
	local final=""
	if declare -F task_build_prompt >/dev/null 2>&1; then
		final=$(task_build_prompt "$base" "$summary" "$helper")
	else
		if [[ -n "$summary" ]]; then final="Context summary from previous runs:\n${summary}\n\n${helper}\n\n${ULTRA_THINK_PREFIX}${base}"; else final="${helper}\n\n${ULTRA_THINK_PREFIX}${base}"; fi
	fi
	echo "$final"
}

emit_start_event() {
	local ts; ts=$(date -Is)
	local sha; sha=$(sha256sum "$1" 2>/dev/null | awk '{print $1}')
	emit_event_jsonl "$(printf '{"type":"start","task":"%s","ts":"%s","iteration":%d,"pid":%d,"prompt_sha":"%s","timeout":%d,"max_turns":%d}' "$LOOP_TASK" "$ts" "$2" "$3" "${sha:-}" "$TIMEOUT" "$MAX_TURNS")"
}

emit_finish_event() {
	local ts; ts=$(date -Is)
	emit_event_jsonl "$(printf '{"type":"finish","task":"%s","ts":"%s","iteration":%d,"pid":%d,"exit_code":%d,"duration_sec":%d}' "$LOOP_TASK" "$ts" "$1" "$2" "$3" "$4")"
}

append_worker_pid() {
	local pid="$1"
	if command -v flock >/dev/null 2>&1; then
		{
			flock -x 201
			printf '%s\n' "$pid" >&201
		} 201>>"$PIDS_FILE"
	else
		printf '%s\n' "$pid" >> "$PIDS_FILE"
	fi
}

run_iteration() {
	local iter="$1"; log_with_timestamp "Starting iteration #$iter (task=$LOOP_TASK)"
	# Prepare env before gating so per-task overrides (e.g., MAX_CONCURRENT_WORKERS) take effect
	prepare_worker_env
	can_start_new_worker || { log_with_timestamp "ERROR: Cannot start new worker (gating or concurrency limit)"; return 1; }
	local prompt_path; prompt_path=$(select_prompt) || { log_with_timestamp "ERROR: No prompt found"; return 1; }
	[[ -r "$prompt_path" ]] || { log_with_timestamp "ERROR: Prompt not readable: $prompt_path"; return 1; }
	local full_prompt; full_prompt=$(compose_prompt "$prompt_path")
	if [[ -z "$full_prompt" ]]; then
		log_with_timestamp "ERROR: Failed to compose prompt"
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
		if ! timeout --signal=TERM --kill-after=30 "$TIMEOUT" resource-claude-code run "$full_prompt" 2>&1 | tee "$tmp_out"; then
			exitc=${PIPESTATUS[0]}
			log_with_timestamp "ERROR: Worker execution failed with exit code $exitc"
		else
			exitc=${PIPESTATUS[0]}
		fi
		
		local end; end=$(date +%s); dur=$((end-start))
		
		# Enhanced error classification
		if [[ $exitc -eq 0 ]]; then
			log_with_timestamp "Iteration #$iter succeeded in ${dur}s"
		elif [[ $exitc -eq 124 ]]; then
			log_with_timestamp "Iteration #$iter timed out in ${dur}s"
		else
			# Check for specific error patterns in output
			if [[ -f "$tmp_out" ]]; then
				# TEMPORARILY DISABLED: API quota exhaustion detection
				# if grep -q "Claude AI usage limit reached" "$tmp_out" 2>/dev/null; then
				# 	exitc=$API_QUOTA_EXHAUSTED
				# 	log_with_timestamp "WARNING: Claude API quota exhausted, marking as quota error"
				# 	# Set flag for quota exhaustion
				# 	echo "$(date +%s)" > "${DATA_DIR}/quota_exhausted.flag" 2>/dev/null || true
				# elif grep -q "command not found\|No such file or directory" "$tmp_out" 2>/dev/null; then
				if grep -q "command not found\|No such file or directory" "$tmp_out" 2>/dev/null; then
					exitc=$WORKER_UNAVAILABLE
					log_with_timestamp "WARNING: Worker unavailable, marking as worker error"
				elif grep -q "permission denied\|access denied" "$tmp_out" 2>/dev/null; then
					exitc=$CONFIGURATION_ERROR
					log_with_timestamp "WARNING: Configuration error detected"
				fi
			fi
			log_with_timestamp "Iteration #$iter failed (exit=$exitc) in ${dur}s"
		fi
		
		# Persist worker output (redacted) and append tail to main log
		if [[ -f "$tmp_out" ]]; then
			local iter_log="${ITERATIONS_DIR}/iter-${iter}.log"
			head -n "$ITERATION_LOG_MAX_LINES" "$tmp_out" | redact > "$iter_log" 2>/dev/null || true
			{
				echo "===== Iteration #$iter worker output (last ${ITERATION_LOG_TAIL_LINES} lines, redacted) ====="
				tail -n "$ITERATION_LOG_TAIL_LINES" "$tmp_out" | redact
				echo "===== End iteration #$iter worker output ====="
			} >> "$LOG_FILE" 2>/dev/null || true
		fi
	) &
	local wrapper_pid=$!
	append_worker_pid "$wrapper_pid"
	emit_start_event "$prompt_path" "$iter" "$wrapper_pid"
	return 0
}

# Synchronous iteration for once
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
	if ! timeout --signal=TERM --kill-after=30 "$TIMEOUT" resource-claude-code run "$full_prompt" 2>&1 | tee "$tmp_out"; then
		exitc=${PIPESTATUS[0]}
		log_with_timestamp "ERROR: Worker execution failed with exit code $exitc"
	else
		exitc=${PIPESTATUS[0]}
	fi
	set -e
	
	local end; end=$(date +%s); dur=$((end-start))
	
	# Enhanced error classification (same as async version)
	if [[ $exitc -eq 0 ]]; then
		log_with_timestamp "Iteration #$iter succeeded in ${dur}s"
	elif [[ $exitc -eq 124 ]]; then
		log_with_timestamp "Iteration #$iter timed out in ${dur}s"
	else
		# Check for specific error patterns in output
		if [[ -f "$tmp_out" ]]; then
			# TEMPORARILY DISABLED: API quota exhaustion detection
			# if grep -q "Claude AI usage limit reached" "$tmp_out" 2>/dev/null; then
			# 	exitc=$API_QUOTA_EXHAUSTED
			# 	log_with_timestamp "WARNING: Claude API quota exhausted, marking as quota error"
			# 	# Set flag for quota exhaustion
			# 	echo "$(date +%s)" > "${DATA_DIR}/quota_exhausted.flag" 2>/dev/null || true
			# elif grep -q "command not found\|No such file or directory" "$tmp_out" 2>/dev/null; then
			if grep -q "command not found\|No such file or directory" "$tmp_out" 2>/dev/null; then
				exitc=$WORKER_UNAVAILABLE
				log_with_timestamp "WARNING: Worker unavailable, marking as worker error"
			elif grep -q "permission denied\|access denied" "$tmp_out" 2>/dev/null; then
				exitc=$CONFIGURATION_ERROR
				log_with_timestamp "WARNING: Configuration error detected"
			fi
		fi
		log_with_timestamp "Iteration #$iter failed (exit=$exitc) in ${dur}s"
	fi
	
	# Persist worker output (redacted) and append tail to main log
	if [[ -f "$tmp_out" ]]; then
		local iter_log="${ITERATIONS_DIR}/iter-${iter}.log"
		head -n "$ITERATION_LOG_MAX_LINES" "$tmp_out" | redact > "$iter_log" 2>/dev/null || true
		{
			echo "===== Iteration #$iter worker output (last ${ITERATION_LOG_TAIL_LINES} lines, redacted) ====="
			tail -n "$ITERATION_LOG_TAIL_LINES" "$tmp_out" | redact
			echo "===== End iteration #$iter worker output ====="
		} >> "$LOG_FILE" 2>/dev/null || true
	fi
	emit_finish_event "$iter" "$$" "$exitc" "$dur"
	update_summary_files || true
	rm -f "$tmp_out"
	return "$exitc"
}

check_and_rotate() {
	# Auto-rotate large log
	if [[ -f "$LOG_FILE" ]]; then
		local sz; sz=$(stat -c%s "$LOG_FILE" 2>/dev/null || echo 0)
		if [[ "$sz" -gt "$LOG_MAX_BYTES" ]]; then
			local ts; ts=$(date '+%Y%m%d_%H%M%S')
			mv -- "$LOG_FILE" "${LOG_FILE}.${ts}"
			echo "[auto-rotate] log -> ${LOG_FILE}.${ts}" >>"$LOG_FILE" 2>/dev/null || true
			# Prune older rotated logs safely
			find "$(dirname "$LOG_FILE")" -maxdepth 1 -type f -name "$(basename "$LOG_FILE").*" -printf '%T@ %p\n' 2>/dev/null | sort -nr | awk 'NR>ENVIRON["ROTATE_KEEP"]{print $2}' ROTATE_KEEP=$((ROTATE_KEEP)) | xargs -r rm -f --
		fi
	fi
	# Auto-rotate large events by line count
	if [[ -f "$EVENTS_JSONL" ]]; then
		local lines; lines=$(wc -l < "$EVENTS_JSONL" 2>/dev/null || echo 0)
		if [[ "$lines" -gt "$EVENTS_MAX_LINES" ]]; then
			local ts; ts=$(date '+%Y%m%d_%H%M%S')
			mv -- "$EVENTS_JSONL" "${EVENTS_JSONL}.${ts}"
			echo "[auto-rotate] events -> ${EVENTS_JSONL}.${ts}" >>"$LOG_FILE" 2>/dev/null || true
			# Prune older rotated events safely
			find "$(dirname "$EVENTS_JSONL")" -maxdepth 1 -type f -name "$(basename "$EVENTS_JSONL").*" -printf '%T@ %p\n' 2>/dev/null | sort -nr | awk 'NR>ENVIRON["ROTATE_KEEP"]{print $2}' ROTATE_KEEP=$((ROTATE_KEEP)) | xargs -r rm -f --
		fi
	fi
	# Clean up temporary files (empty files and old logs)
	if [[ -d "$TMP_DIR" ]]; then
		# Remove empty temporary files older than 1 hour
		find "$TMP_DIR" -name "tmp.*" -size 0 -mmin +60 -delete 2>/dev/null || true
		# Remove temporary log files older than 24 hours
		find "$TMP_DIR" -name "*.log" -mtime +1 -delete 2>/dev/null || true
		# Remove any temporary files older than 7 days
		find "$TMP_DIR" -type f -mtime +7 -delete 2>/dev/null || true
	fi
}

run_loop() {
	# Lock to prevent duplicate instances
	if command -v flock >/dev/null 2>&1; then
		exec 9>"$LOCK_FILE"
		if ! flock -n 9; then echo "Another instance appears to be running (lock held)."; exit 1; fi
	else
		log_with_timestamp "WARNING: flock not available; running without lock"
	fi
	log_with_timestamp "Starting loop (task=$LOOP_TASK)"
	
	# TEMPORARILY DISABLED: Initialize sudo override if available
	# if command -v sudo_override::init >/dev/null 2>&1; then
	# 	log_with_timestamp "🔧 Initializing sudo override for loop operations"
	# 	if sudo_override::init; then
	# 		log_with_timestamp "✅ Sudo override initialized successfully"
	# 		# Load configuration for this session
	# 		if sudo_override::load_config; then
	# 			log_with_timestamp "✅ Sudo override configuration loaded"
	# 		else
	# 			log_with_timestamp "WARNING: Failed to load sudo override configuration"
	# 		fi
	# 	else
	# 		log_with_timestamp "WARNING: Failed to initialize sudo override - continuing without sudo access"
	# 	fi
	# else
	# 	log_with_timestamp "INFO: Sudo override not available - continuing without sudo access"
	# fi
	
	if ! check_worker_available; then log_with_timestamp "FATAL: worker not available"; exit 1; fi
	echo $$ > "$PID_FILE"
	local i=1
	local max_iter="${RUN_MAX_ITERATIONS:-}" # optional bound via env
	while true; do
		# TEMPORARILY DISABLED: API quota exhaustion check
		# # Check for API quota exhaustion before starting new iteration
		# if [[ -f "${DATA_DIR}/quota_exhausted.flag" ]]; then
		# 	local quota_reset_time
		# 	quota_reset_time=$(cat "${DATA_DIR}/quota_exhausted.flag" 2>/dev/null || echo "")
		# 	if [[ -n "$quota_reset_time" ]]; then
		# 		local current_time=$(date +%s)
		# 		local time_until_reset=$((quota_reset_time + 3600 - current_time))  # Assume 1 hour reset
		# 		if [[ $time_until_reset -gt 0 ]]; then
		# 			log_with_timestamp "API quota exhausted, waiting ${time_until_reset}s until estimated reset"
		# 			sleep $time_until_reset
		# 		fi
		# 		rm -f "${DATA_DIR}/quota_exhausted.flag"
		# 		log_with_timestamp "Resuming loop after quota reset"
		# 	fi
		# fi
		
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
			local chunk=$((remain>10?10:remain)); 
			sleep $chunk; 
			remain=$((remain-chunk)); 
		done
		if [[ -n "$max_iter" ]] && [[ $i -ge $max_iter ]]; then log_with_timestamp "Reached max iterations ($max_iter); exiting."; break; fi
		((i++))
	done
}

# Common subcommands
loop_dispatch() {
	local cmd="${1:-run-loop}"; shift || true
	case "$cmd" in
		run-loop)
			local maxflag="${1:-}"; local maxval=""; if [[ "$maxflag" == "--max" ]]; then maxval="${2:-}"; shift 2 || true; export RUN_MAX_ITERATIONS="$maxval"; fi
			run_loop
			;;
		start) nohup "$0" --task "$LOOP_TASK" run-loop >/dev/null 2>&1 & echo "Started loop (task=$LOOP_TASK) PID: $!" ;;
		stop)
			if [[ -f "$PID_FILE" ]]; then
				local main_pid; main_pid=$(cat "$PID_FILE")
				if kill -0 "$main_pid" 2>/dev/null; then
					kill_tree "$main_pid" TERM
					local w=0; while kill -0 "$main_pid" 2>/dev/null && [[ $w -lt 10 ]]; do sleep 1; ((w++)); done
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
			local sub="${1:-summary}"; shift || true
			if [[ ! -f "$EVENTS_JSONL" ]]; then echo "Events file not found: $EVENTS_JSONL"; exit 1; fi
			if ! command -v jq >/dev/null 2>&1; then echo "jq is required for json subcommands"; exit 1; fi
			case "$sub" in
				summary) tail -n 2000 "$EVENTS_JSONL" | jq -s 'def finishes: map(select(.type=="finish")); (. // []) as $e | ($e|finishes) as $fin | {runs: ($fin|length), ok: ($fin|map(select(.exit_code==0))|length), timeout: ($fin|map(select(.exit_code==124))|length), quota_exhausted: ($fin|map(select(.exit_code==142))|length), config_error: ($fin|map(select(.exit_code==143))|length), worker_error: ($fin|map(select(.exit_code==144))|length), other_failures: ($fin|map(select(.exit_code!=0 and .exit_code!=124 and .exit_code!=142 and .exit_code!=143 and .exit_code!=144))|length)} | . + {success_rate: (if .runs==0 then 0 else (.ok/.runs) end)}' ;;
				recent) n="${1:-10}"; tail -n 2000 "$EVENTS_JSONL" | jq -c 'select(.type=="finish") | {ts,task,iteration,exit_code,duration_sec}' | tail -n "$n" ;;
				inflight) tail -n 2000 "$EVENTS_JSONL" | jq -s '[ group_by(.pid) | map({pid: .[0].pid, started:(map(select(.type=="start"))|length), finished:(map(select(.type=="finish"))|length)}) | map(select(.started > .finished)) ]' ;;
				durations) tail -n 2000 "$EVENTS_JSONL" | jq -s 'def finishes: map(select(.type=="finish")); (. // []) as $e | ($e|finishes|map(.duration_sec)|map(select(type=="number"))) as $d | {avg:(if ($d|length)==0 then null else ($d|add/($d|length)) end), p50:($d|sort|.[(length*0.5|floor)]), p90:($d|sort|.[(length*0.9|floor)]), p95:($d|sort|.[(length*0.95|floor)]), p99:($d|sort|.[(length*0.99|floor)]), min:($d|min), max:($d|max)}' ;;
				errors) n="${1:-5}"; tail -n 2000 "$EVENTS_JSONL" | jq -c 'select(.type=="finish" and .exit_code!=0) | {ts,task,iteration,exit_code,duration_sec}' | tail -n "$n" ;;
				error-breakdown) tail -n 2000 "$EVENTS_JSONL" | jq -s 'def finishes: map(select(.type=="finish")); (. // []) as $e | ($e|finishes) as $fin | {quota_exhausted: ($fin|map(select(.exit_code==142))|length), config_error: ($fin|map(select(.exit_code==143))|length), worker_error: ($fin|map(select(.exit_code==144))|length), timeout: ($fin|map(select(.exit_code==124))|length), other_failures: ($fin|map(select(.exit_code!=0 and .exit_code!=124 and .exit_code!=142 and .exit_code!=143 and .exit_code!=144))|length)}' ;;
				hourly) tail -n 2000 "$EVENTS_JSONL" | jq -s '[ .[] | select(.type=="finish") | {h: (.ts|split(":"))[0], exit_code, duration_sec} ] | group_by(.h) | map({hour: .[0].h, count: length, ok: (map(select(.exit_code==0))|length), timeout: (map(select(.exit_code==124))|length), avg_duration: (map(.duration_sec)|add/length)})' ;;
				*) echo "Unknown json subcommand: $sub"; exit 1 ;;
			esac
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
			
			echo "=== Network Connectivity ==="
			# Basic network connectivity check
			if ping -c 1 -W 3 8.8.8.8 >/dev/null 2>&1; then
				echo "✅ network connectivity ok"
			else
				echo "⚠️  network connectivity issues (may affect external APIs)"
			fi
			
			echo "=== Process & Lock Files ==="
			# Check for stale PID files
			if [[ -f "$PID_FILE" ]]; then
				local pid; pid=$(cat "$PID_FILE" 2>/dev/null || echo "")
				if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
					echo "⚠️  loop appears to be running (PID: $pid)"
				else
					echo "❌ stale PID file found"; ok=1
				fi
			else
				echo "✅ no PID file (loop not running)"
			fi
			
			# Check for lock file
			if [[ -f "$LOCK_FILE" ]]; then
				echo "⚠️  lock file exists (may indicate previous crash)"
			else
				echo "✅ no lock file"
			fi
			
			echo "=== Summary ==="
			if [[ $ok -eq 0 ]]; then
				echo "✅ All health checks passed"
			else
				echo "❌ Health checks failed - see issues above"
			fi
			exit $ok
			;;
		once) check_worker_available || { echo "Worker unavailable"; exit 1; }; run_iteration_sync 1 ;;
		help|--help|-h) cat << EOF
Generic Loop Core (task=$LOOP_TASK)
Commands:
  run-loop [--max N] | start | stop | force-stop | status | logs [-f]
  rotate [--events [KEEP] | --temp] | json <name> | dry-run | health | once | skip-wait | cleanup | help

JSON subcommands:
  summary | recent [N] | inflight | durations | errors [N] | error-breakdown | hourly
EOF
			;;
		skip-wait)
			if [[ -f "$PID_FILE" ]]; then
				local pid; pid=$(cat "$PID_FILE" 2>/dev/null || echo "")
				if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
					touch "${DATA_DIR}/skip_wait.flag"
					echo "⏭️  Skip wait flag set - next iteration will start immediately"
				else
					echo "❌ Loop is not running (PID: $pid)"
					exit 1
				fi
			else
				echo "❌ Loop is not running (no PID file)"
				exit 1
			fi
			;;
		cleanup)
			log_with_timestamp "Manual cleanup requested"
			# Clean up orphaned events first
			cleanup_orphaned_events
			# Clean up temporary files
			if [[ -d "$TMP_DIR" ]]; then
				local cleaned=0
				cleaned=$(find "$TMP_DIR" -name "tmp.*" -size 0 -delete -print 2>/dev/null | wc -l)
				log_with_timestamp "Cleaned up $cleaned empty temporary files"
				# Also clean old log files
				local old_logs=0
				old_logs=$(find "$TMP_DIR" -name "*.log" -mtime +1 -delete -print 2>/dev/null | wc -l)
				log_with_timestamp "Cleaned up $old_logs old log files"
			fi
			# Clean up any orphaned PID files
			if [[ -f "$PID_FILE" ]]; then
				local pid; pid=$(cat "$PID_FILE" 2>/dev/null || echo "")
				if [[ -n "$pid" ]] && ! kill -0 "$pid" 2>/dev/null; then
					rm -f "$PID_FILE"
					log_with_timestamp "Removed stale PID file"
				fi
			fi
			if [[ -f "$PIDS_FILE" ]]; then
				local valid_pids=0
				while IFS= read -r pid; do
					if kill -0 "$pid" 2>/dev/null; then
						echo "$pid" >> "$TMP_DIR/valid_pids.tmp"
						((valid_pids++))
					fi
				done < "$PIDS_FILE"
				if [[ $valid_pids -eq 0 ]]; then
					rm -f "$PIDS_FILE"
					log_with_timestamp "Removed empty workers PID file"
				else
					mv "$TMP_DIR/valid_pids.tmp" "$PIDS_FILE"
					log_with_timestamp "Updated workers PID file with $valid_pids valid PIDs"
				fi
			fi
			# TEMPORARILY DISABLED: Enhanced quota exhaustion flag cleanup logic
			# # Enhanced quota exhaustion flag cleanup logic
			# if [[ -f "${DATA_DIR}/quota_exhausted.flag" ]]; then
			# 	local quota_time; quota_time=$(cat "${DATA_DIR}/quota_exhausted.flag" 2>/dev/null || echo "")
			# 	if [[ -n "$quota_time" ]]; then
			# 		local current_time=$(date +%s)
			# 		local time_since_quota=$((current_time - quota_time))
			# 		local should_remove=false
			# 		
			# 		# Remove if older than 2 hours (stale)
			# 		if [[ $time_since_quota -gt 7200 ]]; then
			# 			should_remove=true
			# 			log_with_timestamp "Removing stale quota exhaustion flag (${time_since_quota}s old)"
			# 		# Remove if we have recent successful runs (system is healthy)
			# 		elif [[ -f "$EVENTS_JSONL" ]]; then
			# 			local recent_successes=0
			# 			if command -v jq >/dev/null 2>&1; then
			# 			recent_successes=$(tail -n 100 "$EVENTS_JSONL" | jq -s 'map(select(.type=="finish" and .exit_code==0)) | length' 2>/dev/null || echo "0")
			# 			fi
			# 			if [[ "$recent_successes" -gt 0 ]]; then
			# 				should_remove=true
			# 				log_with_timestamp "Removing quota flag due to recent successful runs ($recent_successes)"
			# 			fi
			# 		fi
			# 		
			# 		if [[ "$should_remove" == "true" ]]; then
			# 			rm -f "${DATA_DIR}/quota_exhausted.flag"
			# 			log_with_timestamp "Removed quota exhaustion flag"
			# 		else
			# 			log_with_timestamp "Keeping quota flag (${time_since_quota}s old, no recent successes)"
			# 		fi
			# 	fi
			# fi
			echo "Cleanup completed"
			;;
		*) run_loop ;;
	esac
} 