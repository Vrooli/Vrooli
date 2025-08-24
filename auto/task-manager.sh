#!/usr/bin/env bash

# Task Manager: routes to task-specific module and generic loop core

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)}"
AUTO_DIR="${APP_ROOT}/auto"
TASKS_DIR="$AUTO_DIR/tasks"
LIB_DIR="$AUTO_DIR/lib"

TASK_NAME="scenario-improvement"
CMD="run-loop"
PROMPT_PATH="${PROMPT_PATH:-}"

# Parse minimal flags: --task, --prompt, subcommand, passthrough
if [[ "${1:-}" == "--task" ]]; then
	TASK_NAME="${2:-scenario-improvement}"; shift 2 || true
fi
if [[ "${1:-}" == "--prompt" ]]; then
	PROMPT_PATH="${2:-}"; export PROMPT_PATH; shift 2 || true
fi
CMD="${1:-run-loop}"; shift || true

TASK_FILE="${TASKS_DIR}/${TASK_NAME}/task.sh"
if [[ ! -f "$TASK_FILE" ]]; then
	echo "FATAL: Task module not found: $TASK_FILE" >&2
	exit 1
fi

# Source task and core
# shellcheck disable=SC1090
source "$TASK_FILE"
# shellcheck disable=SC1090
source "${LIB_DIR}/loop.sh"

# Dispatch via loop core
loop_dispatch "$CMD" "$@" 