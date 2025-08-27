#!/usr/bin/env bash

# Events Library - Event handling and JSON processing
# Part of the modular loop system

set -euo pipefail

# -----------------------------------------------------------------------------
# Function: emit_event_jsonl
# Description: Atomically append JSON event to the events log file
# Parameters:
#   $1 - JSON string to append
# Returns: 0 on success
# Side Effects: Appends to EVENTS_JSONL file
# Dependencies: flock (optional, falls back to simple append)
# -----------------------------------------------------------------------------
emit_event_jsonl() {
	if command -v flock >/dev/null 2>&1; then
		# Open file descriptor and use flock properly
		(
			exec 200>>"$EVENTS_JSONL"
			if flock -x 200; then
				printf '%s\n' "$1" >&200
			else
				# Fallback to simple append if lock fails
				printf '%s\n' "$1" >> "$EVENTS_JSONL"
			fi
		)
	else
		printf '%s\n' "$1" >> "$EVENTS_JSONL"
	fi
}

# -----------------------------------------------------------------------------
# Function: emit_start_event
# Description: Emit an event marking the start of an iteration
# Parameters:
#   $1 - Path to prompt file (for SHA calculation)
#   $2 - Iteration number
#   $3 - Process ID of worker
# Returns: 0 on success
# Side Effects: Appends start event to EVENTS_JSONL
# -----------------------------------------------------------------------------
emit_start_event() {
	local ts; ts=$(date -Is)
	local sha; sha=$(sha256sum "$1" 2>/dev/null | awk '{print $1}')
	emit_event_jsonl "$(printf '{"type":"start","task":"%s","ts":"%s","iteration":%d,"pid":%d,"prompt_sha":"%s","timeout":%d,"max_turns":%d}' "$LOOP_TASK" "$ts" "$2" "$3" "${sha:-}" "$TIMEOUT" "$MAX_TURNS")"
}

# -----------------------------------------------------------------------------
# Function: emit_finish_event
# Description: Emit an event marking the completion of an iteration
# Parameters:
#   $1 - Iteration number
#   $2 - Process ID of worker
#   $3 - Exit code of worker
#   $4 - Duration in seconds
# Returns: 0 on success
# Side Effects: Appends finish event to EVENTS_JSONL
# -----------------------------------------------------------------------------
emit_finish_event() {
	local ts; ts=$(date -Is)
	emit_event_jsonl "$(printf '{"type":"finish","task":"%s","ts":"%s","iteration":%d,"pid":%d,"exit_code":%d,"duration_sec":%d}' "$LOOP_TASK" "$ts" "$1" "$2" "$3" "$4")"
}

# -----------------------------------------------------------------------------
# Function: update_summary_files
# Description: Generate summary statistics from event log and optionally create NL summary
# Parameters: None
# Returns: 0 on success
# Side Effects: Updates SUMMARY_JSON and optionally SUMMARY_TXT files
# Dependencies: jq (required for JSON processing), resource-ollama (optional for NL summary)
# -----------------------------------------------------------------------------
update_summary_files() {
	[[ -f "$EVENTS_JSONL" ]] || return 0
	local tmp_input; tmp_input=$(mktemp -p "$TMP_DIR")
	tail -n "$EVENTS_TAIL_SIZE" "$EVENTS_JSONL" > "$tmp_input" 2>/dev/null || cp "$EVENTS_JSONL" "$tmp_input" 2>/dev/null || true
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
		  ($fin | map(select(.exit_code==150)) | length) as $config |
		  ($fin | map(select(.exit_code==151)) | length) as $worker |
		  ($fin | map(select(.exit_code!=0 and .exit_code!=124 and .exit_code!=150 and .exit_code!=151)) | length) as $other |
		  ($fin | length) as $runs |
		  {
		    task: ("'"${LOOP_TASK}"'"),
		    generated_at: (now | todateiso8601),
		    totals: { runs: $runs, ok: $ok, timeout: $to, config_error: $config, worker_error: $worker, other_failures: $other },
		    success_rate: (if $runs==0 then 0 else ($ok / $runs) end),
		    duration_sec: {
		      avg: (if ($durs|length)==0 then null else ($durs|add/($durs|length)) end),
		      p50: ($durs|perc(50)), p90: ($durs|perc(90)), p95: ($durs|perc(95)), p99: ($durs|perc(99)),
		      min: (if ($durs|length)==0 then null else ($durs|min) end),
		      max: (if ($durs|length)==0 then null else ($durs|max) end)
		    },
		    inflight_count: ($events | inflight_count),
		    recent: ($fin | sort_by(.ts) | reverse | .[:'"$DEFAULT_RECENT_EVENTS"']
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
		if resource-ollama test smoke >/dev/null 2>&1 || resource-ollama content list >/dev/null 2>&1; then
			local p; p=$(mktemp -p "$TMP_DIR")
			{
				printf "Summarize this task metrics in 5-10 lines. Be concise.\nJSON:\n"; cat "$SUMMARY_JSON"
			} > "$p"
			if OUT=$(resource-ollama content generate "$OLLAMA_SUMMARY_MODEL" "$(cat "$p")" 2>/dev/null); then
				printf '%s\n' "$OUT" > "$SUMMARY_TXT"
			fi
			rm -f "$p"
		fi
	fi
	rm -f "$tmp_input" 2>/dev/null || true
}

# -----------------------------------------------------------------------------
# Function: cleanup_orphaned_events
# Description: Remove orphaned start events that lack matching finish events
# Parameters: None
# Returns: 0 on success, 1 on failure
# Side Effects: Modifies EVENTS_JSONL file atomically with backup
# Dependencies: jq (optional, falls back to grep)
# Usage: Called before shutdown to clean up incomplete iterations
# -----------------------------------------------------------------------------
cleanup_orphaned_events() {
	if [[ -f "$EVENTS_JSONL" ]]; then
		local backup_file; backup_file="${EVENTS_JSONL}.backup.$(date +%s)"
		local tmp_file; tmp_file=$(mktemp -p "$TMP_DIR")
		local success=false
		
		# Create backup first
		if ! cp "$EVENTS_JSONL" "$backup_file" 2>/dev/null; then
			if declare -F log_with_timestamp >/dev/null 2>&1; then
				log_with_timestamp "ERROR: Failed to create backup for events cleanup"
			fi
			rm -f "$tmp_file"
			return 1
		fi
		
		# Process events with error handling
		if command -v jq >/dev/null 2>&1; then
			# Keep only non-start events (finish events and other events)
			if jq -c 'map(select(.type != "start"))' "$EVENTS_JSONL" > "$tmp_file" 2>/dev/null; then
				success=true
			fi
		else
			# Fallback: simple cleanup without jq - just remove start events
			if grep -v '"type":"start"' "$EVENTS_JSONL" > "$tmp_file" 2>/dev/null; then
				success=true
			fi
		fi
		
		# Atomic replacement with validation
		if [[ "$success" == "true" ]] && [[ -s "$tmp_file" ]]; then
			if mv "$tmp_file" "$EVENTS_JSONL" 2>/dev/null; then
				rm -f "$backup_file"
				if declare -F log_with_timestamp >/dev/null 2>&1; then
					log_with_timestamp "Cleaned up orphaned events"
				fi
				return 0
			else
				# Restore from backup if move failed
				mv "$backup_file" "$EVENTS_JSONL" 2>/dev/null || true
				if declare -F log_with_timestamp >/dev/null 2>&1; then
					log_with_timestamp "ERROR: Failed to replace events file, restored from backup"
				fi
			fi
		else
			# Restore from backup if processing failed
			mv "$backup_file" "$EVENTS_JSONL" 2>/dev/null || true
			if declare -F log_with_timestamp >/dev/null 2>&1; then
				log_with_timestamp "ERROR: Events cleanup failed, restored from backup"
			fi
		fi
		
		rm -f "$tmp_file" "$backup_file"
		return 1
	fi
	return 0
}

# -----------------------------------------------------------------------------
# Function: process_json_command
# Description: Process various JSON analysis commands on the events log
# Parameters:
#   $1 - Subcommand (summary|recent|inflight|durations|errors|error-breakdown|hourly)
#   $2 - Optional parameter (e.g., count for recent/errors)
# Returns: 0 on success, 1 on error
# Output: JSON analysis results to stdout
# Side Effects: None (read-only)
# Dependencies: jq (required)
# -----------------------------------------------------------------------------
process_json_command() {
	local sub="${1:-summary}"; shift || true
	if [[ ! -f "$EVENTS_JSONL" ]]; then echo "Events file not found: $EVENTS_JSONL"; exit 1; fi
	if ! command -v jq >/dev/null 2>&1; then echo "jq is required for json subcommands"; exit 1; fi
	case "$sub" in
		summary) tail -n "$EVENTS_TAIL_SIZE" "$EVENTS_JSONL" | jq -s 'def finishes: map(select(.type=="finish")); (. // []) as $e | ($e|finishes) as $fin | {runs: ($fin|length), ok: ($fin|map(select(.exit_code==0))|length), timeout: ($fin|map(select(.exit_code==124))|length), quota_exhausted: ($fin|map(select(.exit_code==142))|length), config_error: ($fin|map(select(.exit_code==150))|length), worker_error: ($fin|map(select(.exit_code==151))|length), other_failures: ($fin|map(select(.exit_code!=0 and .exit_code!=124 and .exit_code!=142 and .exit_code!=150 and .exit_code!=151))|length)} | . + {success_rate: (if .runs==0 then 0 else (.ok/.runs) end)}' ;;
		recent) n="${1:-$DEFAULT_RECENT_EVENTS}"; tail -n "$EVENTS_TAIL_SIZE" "$EVENTS_JSONL" | jq -c 'select(.type=="finish") | {ts,task,iteration,exit_code,duration_sec}' | tail -n "$n" ;;
		inflight) tail -n "$EVENTS_TAIL_SIZE" "$EVENTS_JSONL" | jq -s '[ group_by(.pid) | map({pid: .[0].pid, started:(map(select(.type=="start"))|length), finished:(map(select(.type=="finish"))|length)}) | map(select(.started > .finished)) ]' ;;
		durations) tail -n "$EVENTS_TAIL_SIZE" "$EVENTS_JSONL" | jq -s 'def finishes: map(select(.type=="finish")); (. // []) as $e | ($e|finishes|map(.duration_sec)|map(select(type=="number"))) as $d | {avg:(if ($d|length)==0 then null else ($d|add/($d|length)) end), p50:($d|sort|.[(length*0.5|floor)]), p90:($d|sort|.[(length*0.9|floor)]), p95:($d|sort|.[(length*0.95|floor)]), p99:($d|sort|.[(length*0.99|floor)]), min:($d|min), max:($d|max)}' ;;
		errors) n="${1:-5}"; tail -n "$EVENTS_TAIL_SIZE" "$EVENTS_JSONL" | jq -c 'select(.type=="finish" and .exit_code!=0) | {ts,task,iteration,exit_code,duration_sec}' | tail -n "$n" ;;
		error-breakdown) tail -n "$EVENTS_TAIL_SIZE" "$EVENTS_JSONL" | jq -s 'def finishes: map(select(.type=="finish")); (. // []) as $e | ($e|finishes) as $fin | {quota_exhausted: ($fin|map(select(.exit_code==142))|length), config_error: ($fin|map(select(.exit_code==150))|length), worker_error: ($fin|map(select(.exit_code==151))|length), timeout: ($fin|map(select(.exit_code==124))|length), other_failures: ($fin|map(select(.exit_code!=0 and .exit_code!=124 and .exit_code!=142 and .exit_code!=150 and .exit_code!=151))|length)}' ;;
		hourly) tail -n "$EVENTS_TAIL_SIZE" "$EVENTS_JSONL" | jq -s '[ .[] | select(.type=="finish") | {h: (.ts|split(":"))[0], exit_code, duration_sec} ] | group_by(.h) | map({hour: .[0].h, count: length, ok: (map(select(.exit_code==0))|length), timeout: (map(select(.exit_code==124))|length), avg_duration: (map(.duration_sec)|add/length)})' ;;
		*) echo "Unknown json subcommand: $sub"; exit 1 ;;
	esac
}