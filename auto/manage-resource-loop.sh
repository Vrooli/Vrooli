#!/usr/bin/env bash

# Resource Improvement Loop Manager (shim)
# Delegates to the task manager with the resource-improvement task

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TASK_MANAGER="${SCRIPT_DIR}/task-manager.sh"

if [[ ! -x "$TASK_MANAGER" ]]; then
	echo "ERROR: Task manager not found or not executable: $TASK_MANAGER" >&2
	exit 1
fi

case "${1:-help}" in
	start) "$TASK_MANAGER" --task resource-improvement start ;;
	stop) "$TASK_MANAGER" --task resource-improvement stop ;;
	force-stop) "$TASK_MANAGER" --task resource-improvement force-stop ;;
	status) "$TASK_MANAGER" --task resource-improvement status ;;
	logs) shift || true; "$TASK_MANAGER" --task resource-improvement logs "${1:-}" ;;
	rotate) "$TASK_MANAGER" --task resource-improvement rotate ;;
	restart) "$TASK_MANAGER" --task resource-improvement stop || true; sleep 2; "$TASK_MANAGER" --task resource-improvement start ;;
	json) shift || true; "$TASK_MANAGER" --task resource-improvement json "${1:-summary}" "${2:-}" ;;
	help|--help|-h)
		cat << EOF
Resource Improvement Loop Manager (shim)

Commands:
  start       Start the improvement loop
  stop        Stop the improvement loop gracefully
  force-stop  Force stop the loop (emergency stop)
  status      Show current status
  logs [-f]   Show or follow logs
  rotate      Rotate log file
  restart     Stop and then start the loop
  json <cmd>  JSON summaries: summary | recent [N] | inflight | durations | errors [N] | hourly
  help        Show this help message

This manager delegates to: $TASK_MANAGER --task resource-improvement
EOF
		;;
	*) echo "ERROR: Unknown command: ${1:-}" >&2; "$TASK_MANAGER" --task resource-improvement help; exit 1 ;;
 esac 