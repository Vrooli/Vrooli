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
	if [[ -n "$p" && -f "$p" ]]; then echo "$p"; return 0; fi
	if declare -F task_prompt_candidates >/dev/null 2>&1; then
		while IFS= read -r cand; do [[ -f "$cand" ]] && echo "$cand" && return 0; done < <(task_prompt_candidates)
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
	export MAX_TURNS TIMEOUT ALLOWED_TOOLS SKIP_PERMISSIONS EVENTS_JSONL # expose to tasks
	if declare -F task_prepare_worker_env >/dev/null 2>&1; then task_prepare_worker_env; fi
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

