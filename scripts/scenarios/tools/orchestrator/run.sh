#!/usr/bin/env bash
#############################################################################
# Wrapper script to run SAFE Python App Orchestrator
# Uses fork-bomb-safe version with multiple protection layers
#############################################################################

set -euo pipefail

# Fork bomb prevention - check if orchestrator is already running
if [[ "${VROOLI_ORCHESTRATOR_RUNNING:-}" == "1" ]]; then
    echo "ERROR: Orchestrator is already running (recursive call blocked)" >&2
    echo "This safety mechanism prevents fork bombs" >&2
    exit 1
fi

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
ORCHESTRATOR_DIR="${APP_ROOT}/scripts/scenarios/tools/orchestrator"
VENV_DIR="${ORCHESTRATOR_DIR}/venv"

# Use the orchestrator with fork bomb prevention
PYTHON_SCRIPT="${ORCHESTRATOR_DIR}/app_orchestrator.py"

# Verify orchestrator exists
if [[ ! -f "$PYTHON_SCRIPT" ]]; then
    echo "ERROR: Orchestrator not found at $PYTHON_SCRIPT" >&2
    exit 1
fi
# Determine which Python environment to use
if [[ -d "$VENV_DIR" ]] && [[ -f "${VENV_DIR}/bin/activate" ]]; then
    # Virtual environment exists and is complete, use it
    source "${VENV_DIR}/bin/activate"
    echo "Using Python orchestrator (virtual environment)"
else
    # Use system packages (orchestrator handles missing deps gracefully)
    echo "Using Python orchestrator (system packages)"
fi

SCRIPT_TO_RUN="$PYTHON_SCRIPT"

# Run the chosen orchestrator with all arguments passed through
exec python3 "$SCRIPT_TO_RUN" "$@"