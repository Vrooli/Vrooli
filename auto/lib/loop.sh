#!/usr/bin/env bash

# Generic Loop Core Library - Modular Version
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

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
AUTO_LIB_DIR="${APP_ROOT}/auto/lib"

# Source all module files in dependency order
# Note: core.sh now sources constants.sh automatically
# shellcheck disable=SC1090
source "$AUTO_LIB_DIR/core.sh"
# shellcheck disable=SC1090
source "${AUTO_LIB_DIR}/error-handler.sh"
# shellcheck disable=SC1090
source "${AUTO_LIB_DIR}/events.sh"
# shellcheck disable=SC1090
source "${AUTO_LIB_DIR}/process.sh"
# shellcheck disable=SC1090
source "${AUTO_LIB_DIR}/workers.sh"
# shellcheck disable=SC1090
source "${AUTO_LIB_DIR}/dispatch.sh"

# Error handler is now auto-initialized when sourced from error-handler.sh

# Export the main dispatch function for external use
# This maintains backward compatibility with existing callers
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
	# Being sourced - export functions for external use
	export -f loop_dispatch
fi