#!/usr/bin/env bash

# Wrapper: resource-improvement-loop
# Delegates to task-manager with resource-improvement task

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if (($#)); then
	exec "${SCRIPT_DIR}/task-manager.sh" --task resource-improvement "$@"
else
	exec "${SCRIPT_DIR}/task-manager.sh" --task resource-improvement run-loop
fi 