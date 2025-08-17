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
INTERVAL_SECONDS=300  # 5 minutes (allows overlap between Claude instances)
MAX_TURNS=25
TIMEOUT=1800  # 30 minutes timeout per execution
MAX_CONCURRENT_CLAUDE=5  # Maximum concurrent Claude processes
RUNNING_PIDS_FILE="${PROJECT_DIR}/claude-pids.txt"
MAX_TCP_CONNECTIONS=15  # Maximum allowed TCP connections before pausing new iterations
# JSONL events file for structured machine-readable logs
EVENTS_JSONL="${PROJECT_DIR}/scenario-improvement-events.ndjson"
# Summary artifacts (derived from EVENTS_JSONL)
SCENARIO_SUMMARY_JSON="${PROJECT_DIR}/scenario-summary.json"
SCENARIO_SUMMARY_TXT="${PROJECT_DIR}/scenario-summary.txt"
# Model for Ollama summary generation (can be overridden via env)
OLLAMA_SUMMARY_MODEL="${OLLAMA_SUMMARY_MODEL:-llama3.2:3b}"

# Function to log with timestamp
log_with_timestamp() {
	echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# Append a single JSON line to EVENTS_JSONL atomically
log_json_event() {
	# $1: JSON string (one line)
	# Ensure file exists (ignore errors)
	touch "$EVENTS_JSONL" 2>/dev/null || true
	# Use flock on fd 200 for atomic append
	{
		flock -x 200
		printf '%s\n' "$1" >&200
	} 200>>"$EVENTS_JSONL"
}

# Build/update scenario summary files (JSON and text)
update_summary_files() {
	# Requires jq for JSON aggregation; if missing, write a minimal JSON and skip text summary
	if [[ ! -f "$EVENTS_JSONL" ]]; then
		return 0
	fi
	
	local tmp_input
	tmp_input=$(mktemp)
	# Limit to last 2000 lines to keep processing fast
	tail -n 2000 "$EVENTS_JSONL" > "$tmp_input" 2>/dev/null || cp "$EVENTS_JSONL" "$tmp_input" 2>/dev/null || true
	
	if command -v jq >/dev/null 2>&1; then
		local tmp_summary
		tmp_summary=$(mktemp)
		jq -s '
		  def perc(p): (length as $n | if $n==0 then null else (sort | .[((($n-1)*p/100)|floor)]) end);
		  def finishes: map(select(.type=="finish"));
		  def ok: select(.type=="finish" and .exit_code==0);
		  def timeout: select(.type=="finish" and .exit_code==124);
		  def fail: select(.type=="finish" and (.exit_code!=0 and .exit_code!=124));
		  def inflight_count:
		    (group_by(.pid)
		     | map({pid:.[0].pid, s:(map(select(.type=="start"))|length), f:(map(select(.type=="finish"))|length)})
		     | map(select(.s > .f)) | length);
		  
		  (. // []) as $events |
		  ($events | finishes) as $fin |
		  ($fin | map(.duration_sec) | map(select(type=="number"))) as $durs |
		  ($fin | map(select(.exit_code==0)) | length) as $ok |
		  ($fin | map(select(.exit_code==124)) | length) as $to |
		  ($fin | map(select(.exit_code!=0 and .exit_code!=124)) | length) as $fail |
		  ($fin | length) as $runs |
		  {
		    generated_at: (now | todateiso8601),
		    totals: { runs: $runs, ok: $ok, timeout: $to, fail: $fail },
		    success_rate: (if $runs==0 then 0 else ($ok / $runs) end),
		    duration_sec: {
		      avg: (if ($durs|length)==0 then null else ($durs|add/($durs|length)) end),
		      p50: ($durs|perc(50)),
		      p90: ($durs|perc(90)),
		      p95: ($durs|perc(95)),
		      p99: ($durs|perc(99)),
		      min: (if ($durs|length)==0 then null else ($durs|min) end),
		      max: (if ($durs|length)==0 then null else ($durs|max) end)
		    },
		    inflight_count: ($events | inflight_count),
		    recent: ($fin | sort_by(.ts) | reverse | .[:10]
		             | map({ts, iter:(.iteration), status:(if .exit_code==0 then "ok" else (if .exit_code==124 then "timeout" else "fail" end) end), exit:(.exit_code), dur:(.duration_sec)})),
		    last_error: ($fin | map(select(.exit_code!=0)) | sort_by(.ts) | last | if .==null then null else {ts, iter:(.iteration), exit:(.exit_code)} end)
		  }' "$tmp_input" > "$tmp_summary" 2>/dev/null || true
		
		if [[ -s "$tmp_summary" ]]; then
			mv "$tmp_summary" "$SCENARIO_SUMMARY_JSON"
		else
			# Fallback minimal summary
			printf '{"generated_at":"%s","totals":{"runs":0,"ok":0,"timeout":0,"fail":0},"success_rate":0}' "$(date -Is)" > "$SCENARIO_SUMMARY_JSON"
		fi
	else
		# jq not available; write minimal stub
		printf '{"generated_at":"%s","totals":{"runs":null,"ok":null,"timeout":null,"fail":null},"note":"jq not available"}' "$(date -Is)" > "$SCENARIO_SUMMARY_JSON"
	fi
	
	# Try to generate a natural-language summary via Ollama (optional)
	if command -v resource-ollama >/dev/null 2>&1 && [[ -s "$SCENARIO_SUMMARY_JSON" ]]; then
		local prompt_file
		prompt_file=$(mktemp)
		{
			printf "Summarize the following JSON metrics into 5-10 lines."
			printf " Focus on totals (ok/timeout/fail), success rate (percent), avg and p50/p90 durations, last error (when/exit), and inflight count."
			printf " Be concise, neutral, and factual.\nJSON:\n"
			cat "$SCENARIO_SUMMARY_JSON"
		} > "$prompt_file"
		local prompt_text
		prompt_text=$(cat "$prompt_file")
		resource-ollama generate "$OLLAMA_SUMMARY_MODEL" "$prompt_text" > "$SCENARIO_SUMMARY_TXT" 2>/dev/null || true
		rm -f "$prompt_file"
		# If generation failed or empty, create a simple textual fallback from JSON via jq
		if [[ ! -s "$SCENARIO_SUMMARY_TXT" ]] && command -v jq >/dev/null 2>&1; then
			jq -r '
			  def pct(x): (x*100|round/100);
			  . as $s |
			  "Runs: \(.totals.runs // 0), OK: \(.totals.ok // 0), Timeout: \(.totals.timeout // 0), Fail: \(.totals.fail // 0)\n"
			  + "Success rate: \(pct(.success_rate // 0)*100)%\n"
			  + ("Avg: \(.duration_sec.avg // "n/a")s; p50: \(.duration_sec.p50 // "n/a")s; p90: \(.duration_sec.p90 // "n/a")s\n")
			  + ("Inflight: \(.inflight_count // 0)\n")
			  + (if (.last_error!=null) then ("Last error: \(.last_error.ts) iter=\(.last_error.iter) exit=\(.last_error.exit)\n") else "" end)
			' "$SCENARIO_SUMMARY_JSON" > "$SCENARIO_SUMMARY_TXT" 2>/dev/null || true
		fi
	fi
	
	rm -f "$tmp_input" 2>/dev/null || true
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
	
	# Also cleanup any orphaned TCP connections
	cleanup_orphaned_tcp_connections
}

# Function to cleanup orphaned TCP connections
cleanup_orphaned_tcp_connections() {
	# Check for Claude API connections that might be stuck
	local orphaned_connections
	orphaned_connections=$(ss -tn state established 2>/dev/null | grep -E "(claude|anthropic|api\.anthropic)" | wc -l)
	
	if [[ $orphaned_connections -gt 10 ]]; then
		log_with_timestamp "WARNING: Found $orphaned_connections potential orphaned Claude connections"
		log_with_timestamp "Consider restarting the loop if connections continue to accumulate"
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
	
	# Check TCP connections to prevent accumulation
	local current_connections
	current_connections=$(ss -tn state established 2>/dev/null | grep -E "(claude|anthropic|443)" | wc -l)
	
	if [[ $current_connections -gt $MAX_TCP_CONNECTIONS ]]; then
		log_with_timestamp "Too many TCP connections ($current_connections > $MAX_TCP_CONNECTIONS), skipping iteration"
		log_with_timestamp "Waiting for connections to clear..."
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
	
	# Optional context: include latest summary and helper references if available
	local summary_context=""
	if [[ -f "$SCENARIO_SUMMARY_TXT" ]]; then
		summary_context=$(cat "$SCENARIO_SUMMARY_TXT")
	fi
	local helper_context="Events ledger: $EVENTS_JSONL. Cheatsheet: ${PROJECT_DIR}/scenario-improvement-cheatsheet.md. SCENARIO_EVENTS_JSONL exported. Use commands only if deeper research is required."
	
	# Prepend summary and ultra-think instruction to the prompt
	local full_prompt=""
	if [[ -n "$summary_context" ]]; then
		full_prompt="Context summary from previous runs:\n${summary_context}\n\n${helper_context}\n\nUltra think. ${prompt_content}"
	else
		full_prompt="${helper_context}\n\nUltra think. ${prompt_content}"
	fi
	
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
		# Expose events path to the agent
		export SCENARIO_EVENTS_JSONL="$EVENTS_JSONL"
		
		# Function to cleanup TCP connections on exit
		cleanup_tcp_connections() {
			# Check for any remaining Claude TCP connections and close them
			local claude_connections
			claude_connections=$(ss -tnp 2>/dev/null | grep -i claude | awk '{print $5}' | cut -d: -f1 | sort -u)
			if [[ -n "$claude_connections" ]]; then
				log_with_timestamp "Closing orphaned TCP connections for iteration #${iteration_count}"
				# Force close connections by killing any remaining claude processes
				pkill -f "claude.*${iteration_count}" 2>/dev/null || true
			fi
		}
		
		# Set trap to cleanup on any exit
		trap cleanup_tcp_connections EXIT
		
		# Run resource-claude-code with proper error capture and timeout with graceful shutdown
		local claude_exit_code=0
		# Use timeout with --kill-after to ensure process cleanup
		timeout --kill-after=30 "$TIMEOUT" resource-claude-code run "$full_prompt" 2>&1 | tee "$temp_output"
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
		
		# Emit finish event to JSONL
		local ts_finish
		ts_finish=$(date -Is)
		local finish_json
		finish_json=$(printf '{"type":"finish","ts":"%s","iteration":%d,"pid":%d,"exit_code":%d,"duration_sec":%d}' \
			"$ts_finish" "$iteration_count" "$$" "$claude_exit_code" "$duration")
		log_json_event "$finish_json"
		
		# Update summary artifacts (JSON and text) for next iteration
		update_summary_files || true
		
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
	
	# Emit start event to JSONL
	local ts_start
	ts_start=$(date -Is)
	local prompt_sha
	prompt_sha=$(sha256sum "$PROMPT_FILE" 2>/dev/null | awk '{print $1}')
	local start_json
	start_json=$(printf '{"type":"start","ts":"%s","iteration":%d,"pid":%d,"prompt_sha":"%s","timeout":%d,"max_turns":%d}' \
		"$ts_start" "$iteration_count" "$claude_pid" "${prompt_sha:-}" "$TIMEOUT" "$MAX_TURNS")
	log_json_event "$start_json"
	
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