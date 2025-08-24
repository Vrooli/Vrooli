#!/usr/bin/env bash

# Scenario Improvement Loop Manager (shim)
# Delegates to the task manager with the scenario-improvement task

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)}"
AUTO_DIR="${APP_ROOT}/auto"
TASK_MANAGER="${AUTO_DIR}/task-manager.sh"

if [[ ! -x "$TASK_MANAGER" ]]; then
	echo "ERROR: Task manager not found or not executable: $TASK_MANAGER" >&2
	exit 1
fi

case "${1:-help}" in
	help|--help|-h)
		cat << EOF
Scenario Improvement Loop Manager (shim)

This manager delegates all commands to: $TASK_MANAGER --task scenario-improvement

Common Commands:
  start       Start the improvement loop
  stop        Stop the improvement loop gracefully  
  force-stop  Force stop the loop (emergency stop)
  status      Show current status
  logs [-f]   Show or follow logs
  rotate      Rotate log file
  json <cmd>  JSON summaries: summary | recent [N] | inflight | durations | errors [N] | hourly
  health      Run comprehensive health checks
  dry-run     Show configuration preview
  once        Run single iteration synchronously
  skip-wait   Skip current iteration wait (for testing)
  help        Show this help message

For complete command list and help: $0 help-full
EOF
		;;
	help-full)
		"$TASK_MANAGER" --task scenario-improvement help
		;;
	restart) 
		"$TASK_MANAGER" --task scenario-improvement stop || true
		sleep 2
		"$TASK_MANAGER" --task scenario-improvement start
		;;
	*) 
		# Delegate all other commands to task manager
		"$TASK_MANAGER" --task scenario-improvement "$@"
		;;
esac