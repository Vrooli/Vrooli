#!/usr/bin/env bash

# Task module: resource-improvement
# Provides hooks for the generic loop core to continuously improve local resources

set -euo pipefail

LOOP_TASK="resource-improvement"
TASK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TASK_PROMPTS_DIR="${TASK_DIR}/prompts"

# Candidates in priority order
# 1) Explicit PROMPT_PATH
# 2) Task default prompt
# 3) /tmp override for ad-hoc testing
# shellcheck disable=SC2120
task_prompt_candidates() {
	cat << EOF
${PROMPT_PATH:-}
${TASK_PROMPTS_DIR}/resource-improvement-loop.md
/tmp/resource-improvement-loop.md
EOF
}

# Small helper context appended before the base prompt
# Keep this concise; the worker can call helper scripts for details
# Exposes mode and important paths. The worker must prefer CLI usage.
task_build_helper_context() {
	echo "Events ledger: ${EVENTS_JSONL}."
	echo "Cheatsheet: ${TASK_PROMPTS_DIR}/cheatsheet.md."
	echo "Mode: ${RESOURCE_IMPROVEMENT_MODE:-plan}. Allowed commands: 'vrooli resource <name> <cmd>', 'resource-<name>', docker (read-only status), jq, curl (read-only), grep, sed, awk, timeout. No direct file edits."
	echo "TCP gating: disabled in plan mode."
}

# Compose final prompt
# Args: $1 base_prompt, $2 summary_context, $3 helper_context
# Mirrors scenario-improvement composition for consistency
task_build_prompt() {
	local base="$1"; local summary="$2"; local helper="$3"
	if [[ -n "$summary" ]]; then
		printf '%s\n\n%s\n\n%s\n\n%s' "Context summary from previous runs:" "$summary" "$helper" "${ULTRA_THINK_PREFIX}${base}"
	else
		printf '%s\n\n%s' "$helper" "${ULTRA_THINK_PREFIX}${base}"
	fi
}

# Ensure worker commands exist
# Require vrooli and jq; resource CLIs are optional and may vary by target
task_check_worker_available() {
	command -v vrooli >/dev/null 2>&1 && command -v jq >/dev/null 2>&1
}

# Export task-specific env vars for the worker
# - Constrain concurrency by default (side-effectful work)
# - Expose config and convenience envs
# - Do not emit secrets
# shellcheck disable=SC2034
task_prepare_worker_env() {
	# Ensure conservative defaults unless explicitly overridden
	export MAX_CONCURRENT_WORKERS="${MAX_CONCURRENT_WORKERS:-1}"
	# In plan mode, disable TCP gating by clearing filter
	if [[ "${RESOURCE_IMPROVEMENT_MODE:-plan}" == "plan" ]]; then
		export LOOP_TCP_FILTER=""
	fi
	# Constrain allowed tools to align with guardrails
	export ALLOWED_TOOLS="${ALLOWED_TOOLS:-Bash,LS,Glob,Grep,Read}"
	export SKIP_PERMISSIONS="${SKIP_PERMISSIONS:-no}"
	# Provide references the worker may use
	export RESOURCE_EVENTS_JSONL="$EVENTS_JSONL"
	# Compute repo root for config references
	local repo_root
	repo_root="$(cd "${TASK_DIR}/../../.." && pwd)"
	export RESOURCES_CONFIG_PATH="${RESOURCES_CONFIG_PATH:-${repo_root}/.vrooli/service.json}"
	export RESOURCES_JSON_CMD="${RESOURCES_JSON_CMD:-vrooli resource list --format json}"
	# Baseline models for Ollama checks (comma-separated)
	export OLLAMA_BASELINE_MODELS="${OLLAMA_BASELINE_MODELS:-llama3.2:3b}"
} 