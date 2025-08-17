#!/usr/bin/env bash

# Wrapper: scenario-improvement-loop
# Delegates to task-manager with scenario-improvement task

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if (($#)); then
	exec "${SCRIPT_DIR}/task-manager.sh" --task scenario-improvement "$@"
else
	exec "${SCRIPT_DIR}/task-manager.sh" --task scenario-improvement run-loop
fi