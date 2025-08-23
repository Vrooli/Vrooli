#!/usr/bin/env bash
#############################################################################
# Wrapper script to run Python App Orchestrator
# Intelligently chooses between full and minimal versions
#############################################################################

set -euo pipefail

# Get script directory
ORCHESTRATOR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="${ORCHESTRATOR_DIR}/venv"
PYTHON_SCRIPT="${ORCHESTRATOR_DIR}/app_orchestrator.py"
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