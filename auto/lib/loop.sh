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
		  ($fin | map(select(.exit_code!=0 and .exit_code!=124)) | length) as $fail |
		  ($fin | length) as $runs |
		  {
		    task: ("'""${LOOP_TASK}""'"),
		    generated_at: (now | todateiso8601),
		    totals: { runs: $runs, ok: $ok, timeout: $to, fail: $fail },
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
			if OUT=$(resource-ollama generate "$OLLAMA_SUMMARY_MODEL" "$(cat "$p")" 2>/dev/null); then
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

current_tcp_connections() {
	# Process-aware gating: count matching worker processes rather than raw TCP lines
	# Allow disable by empty filter
	if [[ -z "${LOOP_TCP_FILTER}" ]]; then echo 0; return 0; fi
	# Prefer pgrep for process counting
	if command -v pgrep >/dev/null 2>&1; then
		local cnt
		cnt=$(pgrep -fal "${LOOP_TCP_FILTER}" 2>/dev/null | wc -l | awk '{print $1}')
		echo "${cnt:-0}"
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
	can_start_new_worker || return 1
	local prompt_path; prompt_path=$(select_prompt) || { log_with_timestamp "ERROR: No prompt found"; return 1; }
	[[ -r "$prompt_path" ]] || { log_with_timestamp "ERROR: Prompt not readable: $prompt_path"; return 1; }
	local full_prompt; full_prompt=$(compose_prompt "$prompt_path")
	# Launch a wrapper subshell in background; run the pipeline in foreground within the subshell
	(
		set +e
		# Cleanup function for TCP connections (avoid pkill patterns; rely on wrapper lifecycle)
		trap - EXIT
		local start; start=$(date +%s)
		local tmp_out; tmp_out=$(mktemp -p "$TMP_DIR" "${LOOP_TASK}-${iter}-XXXX.log")
		# Run worker pipeline in foreground; capture the exit code of timeout (the first command)
		# Use more aggressive timeout with proper signal escalation
		timeout --signal=TERM --kill-after=30 "$TIMEOUT" resource-claude-code run "$full_prompt" 2>&1 | tee "$tmp_out"
		local exitc=${PIPESTATUS[0]}
		local end; end=$(date +%s); local dur=$((end-start))
		if [[ $exitc -eq 0 ]]; then
			log_with_timestamp "Iteration #$iter succeeded in ${dur}s"
		elif [[ $exitc -eq 124 ]]; then
			log_with_timestamp "Iteration #$iter timed out in ${dur}s"
		else
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
		exit $exitc
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
	set +e
	emit_start_event "$prompt_path" "$iter" "$$"
	# Use more aggressive timeout with proper signal escalation
	timeout --signal=TERM --kill-after=30 "$TIMEOUT" resource-claude-code run "$full_prompt" 2>&1 | tee "$tmp_out"
	local exitc=${PIPESTATUS[0]}
	set -e
	local end; end=$(date +%s); local dur=$((end-start))
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
	if ! check_worker_available; then log_with_timestamp "FATAL: worker not available"; exit 1; fi
	echo $$ > "$PID_FILE"
	local i=1
	local max_iter="${RUN_MAX_ITERATIONS:-}" # optional bound via env
	while true; do
		log_with_timestamp "--- Iteration $i ---"
		if run_iteration "$i"; then log_with_timestamp "Iteration $i dispatched"; else log_with_timestamp "Iteration $i skipped"; fi
		check_and_rotate
		log_with_timestamp "Waiting ${INTERVAL_SECONDS}s until next iteration..."
		local remain=$INTERVAL_SECONDS
		while [[ $remain -gt 0 ]]; do local chunk=$((remain>10?10:remain)); sleep $chunk; remain=$((remain-chunk)); done
		if [[ -n "$max_iter" ]] && [[ $i -ge $max_iter ]]; then log_with_timestamp "Reached max iterations ($max_iter); exiting."; break; fi
		((i++))
	done
}

# Redact helper for dry-run
redact() {
	sed -E 's/([A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,})/[redacted-email]/g; s/(api|token|secret|password|passwd|key)=[^ ]+/\1=[REDACTED]/ig'
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
			;;
		json)
			local sub="${1:-summary}"; shift || true
			if [[ ! -f "$EVENTS_JSONL" ]]; then echo "Events file not found: $EVENTS_JSONL"; exit 1; fi
			if ! command -v jq >/dev/null 2>&1; then echo "jq is required for json subcommands"; exit 1; fi
			case "$sub" in
				summary) tail -n 2000 "$EVENTS_JSONL" | jq -s 'def finishes: map(select(.type=="finish")); (. // []) as $e | ($e|finishes) as $fin | {runs: ($fin|length), ok: ($fin|map(select(.exit_code==0))|length), timeout: ($fin|map(select(.exit_code==124))|length), fail: ($fin|map(select(.exit_code!=0 and .exit_code!=124))|length)} | . + {success_rate: (if .runs==0 then 0 else (.ok/.runs) end)}' ;;
				recent) n="${1:-10}"; tail -n 2000 "$EVENTS_JSONL" | jq -c 'select(.type=="finish") | {ts,task,iteration,exit_code,duration_sec}' | tail -n "$n" ;;
				inflight) tail -n 2000 "$EVENTS_JSONL" | jq -s '[ group_by(.pid) | map({pid: .[0].pid, started:(map(select(.type=="start"))|length), finished:(map(select(.type=="finish"))|length)}) | map(select(.started > .finished)) ]' ;;
				durations) tail -n 2000 "$EVENTS_JSONL" | jq -s 'def finishes: map(select(.type=="finish")); (. // []) as $e | ($e|finishes|map(.duration_sec)|map(select(type=="number"))) as $d | {avg:(if ($d|length)==0 then null else ($d|add/($d|length)) end), p50:($d|sort|.[(length*0.5|floor)]), p90:($d|sort|.[(length*0.9|floor)]), p95:($d|sort|.[(length*0.95|floor)]), p99:($d|sort|.[(length*0.99|floor)]), min:($d|min), max:($d|max)}' ;;
				errors) n="${1:-5}"; tail -n 2000 "$EVENTS_JSONL" | jq -c 'select(.type=="finish" and .exit_code!=0) | {ts,task,iteration,exit_code,duration_sec}' | tail -n "$n" ;;
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
			for c in timeout vrooli resource-claude-code; do if ! command -v "$c" >/dev/null 2>&1; then echo "missing: $c"; ok=1; fi; done
			if [[ ! -w "$DATA_DIR" ]]; then echo "data dir not writable: $DATA_DIR"; ok=1; fi
			local prompt; prompt=$(select_prompt || true); if [[ -z "${prompt:-}" ]]; then echo "no prompt found"; ok=1; else echo "prompt ok: $prompt"; fi
			exit $ok
			;;
		once) check_worker_available || { echo "Worker unavailable"; exit 1; }; run_iteration_sync 1 ;;
		help|--help|-h) cat << EOF
Generic Loop Core (task=$LOOP_TASK)
Commands:
  run-loop [--max N] | start | stop | force-stop | status | logs [-f]
  rotate [--events [KEEP]] | json <name> | dry-run | health | once | help
EOF
			;;
		*) run_loop ;;
	esac
} 