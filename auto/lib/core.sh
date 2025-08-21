#!/usr/bin/env bash

# Core Loop Library - Basic configuration, logging, and utilities
# Part of the modular loop system

set -euo pipefail

# Get the library directory using consistent pattern
LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source constants first
# shellcheck disable=SC1091
source "$LIB_DIR/constants.sh"

# Source error codes module
# shellcheck disable=SC1091
source "$LIB_DIR/error-codes.sh"

# Validate required vars
if [[ -z "${LOOP_TASK:-}" ]]; then
	echo "FATAL: LOOP_TASK is not set for loop core" >&2
	exit 1
fi

# Directories - use standardized path resolution
AUTO_DIR="${AUTO_DIR:-$(cd "$LIB_DIR/.." && pwd)}"
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
MAX_CONCURRENT_WORKERS=${MAX_CONCURRENT_WORKERS:-$DEFAULT_MAX_CONCURRENT_WORKERS}
MAX_TCP_CONNECTIONS=${MAX_TCP_CONNECTIONS:-$DEFAULT_MAX_TCP_CONNECTIONS}
LOOP_TCP_FILTER="${LOOP_TCP_FILTER:-claude|anthropic|resource-claude-code}"  # Narrow default filter; exclude generic 443 to avoid false positives
OLLAMA_SUMMARY_MODEL="${OLLAMA_SUMMARY_MODEL:-llama3.2:3b}"
ULTRA_THINK_PREFIX="${ULTRA_THINK_PREFIX:-Ultra think. }"
ALLOWED_TOOLS="${ALLOWED_TOOLS:-Read,Write,Edit,Bash,LS,Glob,Grep}"
SKIP_PERMISSIONS="${SKIP_PERMISSIONS:-yes}"
PROMPT_PATH="${PROMPT_PATH:-${SCENARIO_PROMPT_PATH:-}}"
ROTATE_KEEP=${ROTATE_KEEP:-$DEFAULT_ROTATE_KEEP}
# Use constants for file size and line limits (from constants.sh)
# Note: These are already defined as readonly in constants.sh, so we don't need to redefine them

# Error classification exit codes (use constants)
# These are used across multiple files in the loop system
# shellcheck disable=SC2034
CONFIGURATION_ERROR=$EXIT_CONFIGURATION_ERROR
# shellcheck disable=SC2034
WORKER_UNAVAILABLE=$EXIT_WORKER_UNAVAILABLE
# shellcheck disable=SC2034
TIMEOUT_ERROR=124  # Standard timeout command exit code

# -----------------------------------------------------------------------------
# Function: redact
# Description: Sanitize sensitive information from log output
# Parameters: None (reads from stdin)
# Returns: Sanitized text to stdout
# Side Effects: None
# Usage: echo "password=secret123" | redact
# -----------------------------------------------------------------------------
redact() {
	sed -E 's/([A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,})/[redacted-email]/g; s/(api|token|secret|password|passwd|key)=[^ ]+/\1=[REDACTED]/ig'
}

# -----------------------------------------------------------------------------
# Function: log_with_timestamp
# Description: Log message with timestamp to both stdout and log file
# Parameters:
#   $* - Message to log
# Returns: 0 on success
# Side Effects: Appends to LOG_FILE
# Usage: log_with_timestamp "Starting iteration 5"
# -----------------------------------------------------------------------------
log_with_timestamp() {
	echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# -----------------------------------------------------------------------------
# Function: select_prompt
# Description: Select the appropriate prompt file based on priority
# Parameters: None
# Returns: 0 on success, 1 if no prompt found
# Output: Path to selected prompt file
# Side Effects: None
# Priority: 1) PROMPT_PATH env var, 2) task_prompt_candidates function
# -----------------------------------------------------------------------------
select_prompt() {
	local p="$PROMPT_PATH"
	if [[ -n "$p" ]]; then
		if [[ -f "$p" && -r "$p" ]]; then 
			echo "$p"
			return 0
		else
			if declare -F log_with_timestamp >/dev/null 2>&1; then
				log_with_timestamp "WARNING: PROMPT_PATH set but file not readable: $p"
			fi
		fi
	fi
	if declare -F task_prompt_candidates >/dev/null 2>&1; then
		while IFS= read -r cand; do 
			if [[ -n "$cand" && -f "$cand" && -r "$cand" ]]; then 
				echo "$cand"
				return 0
			fi
		done < <(task_prompt_candidates 2>/dev/null || true)
	fi
	return 1
}

# -----------------------------------------------------------------------------
# Function: check_worker_available
# Description: Verify all required worker dependencies are available
# Parameters: None
# Returns: 0 if all dependencies available, 1 otherwise
# Side Effects: Logs warnings for missing tools
# Dependencies: timeout, resource-claude-code, task-specific requirements
# -----------------------------------------------------------------------------
check_worker_available() {
	local missing=()
	command -v timeout >/dev/null 2>&1 || missing+=(timeout)
	command -v resource-claude-code >/dev/null 2>&1 || missing+=(resource-claude-code)
	
	# Validate prompt can be selected before worker starts
	local prompt_path
	if ! prompt_path=$(select_prompt 2>/dev/null); then
		missing+=(prompt)
	elif [[ -z "$prompt_path" ]]; then
		missing+=(prompt)
	elif [[ ! -f "$prompt_path" ]]; then
		missing+=(prompt-file)
		log_with_timestamp "ERROR: Selected prompt file does not exist: $prompt_path"
	elif [[ ! -r "$prompt_path" ]]; then
		missing+=(prompt-readable)
		log_with_timestamp "ERROR: Selected prompt file is not readable: $prompt_path"
	fi
	
	if declare -F task_check_worker_available >/dev/null 2>&1; then
		if ! task_check_worker_available; then missing+=(worker); fi
	fi
	if ((${#missing[@]})); then
		log_with_timestamp "WARNING: Missing tools: ${missing[*]}"
		return 1
	fi
	return 0
}

# -----------------------------------------------------------------------------
# Function: prepare_worker_env
# Description: Set up environment variables for worker execution
# Parameters: None
# Returns: 0 on success
# Side Effects: Exports environment variables for worker process
# Exports: MAX_TURNS, TIMEOUT, ALLOWED_TOOLS, SKIP_PERMISSIONS, EVENTS_JSONL
# -----------------------------------------------------------------------------
prepare_worker_env() {
	# Validate core files exist before proceeding
	if [[ ! -f "$EVENTS_JSONL" ]]; then
		# Create events file if it doesn't exist
		touch "$EVENTS_JSONL" 2>/dev/null || {
			log_with_timestamp "ERROR: Cannot create events file: $EVENTS_JSONL"
			return 1
		}
	fi
	
	if [[ ! -w "$(dirname "$EVENTS_JSONL")" ]]; then
		log_with_timestamp "ERROR: Events directory not writable: $(dirname "$EVENTS_JSONL")"
		return 1
	fi
	
	# Validate data directory structure
	if [[ ! -d "$DATA_DIR" ]]; then
		log_with_timestamp "ERROR: Data directory does not exist: $DATA_DIR"
		return 1
	fi
	
	if [[ ! -w "$DATA_DIR" ]]; then
		log_with_timestamp "ERROR: Data directory not writable: $DATA_DIR"
		return 1
	fi
	
	# Export core environment variables
	export MAX_TURNS TIMEOUT ALLOWED_TOOLS SKIP_PERMISSIONS EVENTS_JSONL # expose to tasks
	
	# Call task-specific environment preparation with error handling
	if declare -F task_prepare_worker_env >/dev/null 2>&1; then
		if ! task_prepare_worker_env; then
			if declare -F log_with_timestamp >/dev/null 2>&1; then
				log_with_timestamp "ERROR: Task-specific worker environment preparation failed"
			fi
			return 1
		fi
	fi
	return 0
}

# -----------------------------------------------------------------------------
# Function: compose_prompt
# Description: Compose final prompt from base prompt, summary, and helper context
# Parameters:
#   $1 - Path to base prompt file
# Returns: 0 on success
# Output: Composed prompt string
# Side Effects: Reads from SUMMARY_TXT if exists
# Dependencies: task_build_helper_context, task_build_prompt (optional)
# -----------------------------------------------------------------------------
compose_prompt() {
	local prompt_file="$1"
	
	# Validate prompt file exists and is readable
	if [[ ! -f "$prompt_file" ]] || [[ ! -r "$prompt_file" ]]; then
		if declare -F log_with_timestamp >/dev/null 2>&1; then
			log_with_timestamp "ERROR: Prompt file not readable: $prompt_file"
		fi
		return 1
	fi
	
	# Read base prompt with error handling
	local base
	if ! base=$(cat "$prompt_file" 2>/dev/null); then
		if declare -F log_with_timestamp >/dev/null 2>&1; then
			log_with_timestamp "ERROR: Failed to read prompt file: $prompt_file"
		fi
		return 1
	fi
	
	# Validate base prompt is not empty
	if [[ -z "$base" ]]; then
		if declare -F log_with_timestamp >/dev/null 2>&1; then
			log_with_timestamp "ERROR: Prompt file is empty: $prompt_file"
		fi
		return 1
	fi
	
	# Read summary with error handling
	local summary=""
	if [[ -f "$SUMMARY_TXT" ]]; then
		summary=$(cat "$SUMMARY_TXT" 2>/dev/null || true)
	fi
	
	# Get helper context with error handling
	local helper=""
	if declare -F task_build_helper_context >/dev/null 2>&1; then
		if ! helper=$(task_build_helper_context 2>/dev/null); then
			if declare -F log_with_timestamp >/dev/null 2>&1; then
				log_with_timestamp "WARNING: Failed to get helper context, continuing without it"
			fi
			helper=""
		fi
	fi
	
	# Compose final prompt
	local final=""
	if declare -F task_build_prompt >/dev/null 2>&1; then
		if ! final=$(task_build_prompt "$base" "$summary" "$helper" 2>/dev/null); then
			if declare -F log_with_timestamp >/dev/null 2>&1; then
				log_with_timestamp "ERROR: Failed to build task-specific prompt"
			fi
			return 1
		fi
	else
		if [[ -n "$summary" ]]; then 
			final="Context summary from previous runs:\n${summary}\n\n${helper}\n\n${ULTRA_THINK_PREFIX}${base}"
		else 
			final="${helper}\n\n${ULTRA_THINK_PREFIX}${base}"
		fi
	fi
	
	# Validate final prompt is not empty
	if [[ -z "$final" ]]; then
		if declare -F log_with_timestamp >/dev/null 2>&1; then
			log_with_timestamp "ERROR: Final prompt composition resulted in empty prompt"
		fi
		return 1
	fi
	
	echo "$final"
}

