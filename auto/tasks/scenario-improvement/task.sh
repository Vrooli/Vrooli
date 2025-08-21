#!/usr/bin/env bash

# Task module: scenario-improvement
# Provides hooks for the generic loop core

set -euo pipefail

LOOP_TASK="scenario-improvement"
# Use standardized path handling pattern
TASK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TASK_PROMPTS_DIR="$TASK_DIR/prompts"

# Candidates in priority order
task_prompt_candidates() {
	cat << EOF
${PROMPT_PATH:-}
${TASK_PROMPTS_DIR}/scenario-improvement-loop.md
/tmp/scenario-improvement-loop.md
EOF
}

# Small helper context appended before the base prompt
task_build_helper_context() {
	echo "Events ledger: ${EVENTS_JSONL}. Cheatsheet: ${TASK_PROMPTS_DIR}/cheatsheet.md. SCENARIO_EVENTS_JSONL exported. Use commands only if deeper research is required."
}

# Compose final prompt
# Args: $1 base_prompt, $2 summary_context, $3 helper_context
task_build_prompt() {
	local base="$1"; local summary="$2"; local helper="$3"
	if [[ -n "$summary" ]]; then
		printf '%s\n\n%s\n\n%s\n\n%s' "Context summary from previous runs:" "$summary" "$helper" "${ULTRA_THINK_PREFIX}${base}"
	else
		printf '%s\n\n%s' "$helper" "${ULTRA_THINK_PREFIX}${base}"
	fi
}

# Ensure worker commands exist
task_check_worker_available() {
	command -v vrooli >/dev/null 2>&1 && command -v resource-claude-code >/dev/null 2>&1
}

# Export task-specific env vars for the worker
task_prepare_worker_env() {
	# Validate cheatsheet file exists
	local cheatsheet_path="${TASK_PROMPTS_DIR}/cheatsheet.md"
	if [[ ! -f "$cheatsheet_path" ]]; then
		if declare -F log_with_timestamp >/dev/null 2>&1; then
			log_with_timestamp "WARNING: Cheatsheet file not found: $cheatsheet_path"
		fi
	elif [[ ! -r "$cheatsheet_path" ]]; then
		if declare -F log_with_timestamp >/dev/null 2>&1; then
			log_with_timestamp "WARNING: Cheatsheet file not readable: $cheatsheet_path"
		fi
	fi
	
	export SCENARIO_EVENTS_JSONL="$EVENTS_JSONL"
} 