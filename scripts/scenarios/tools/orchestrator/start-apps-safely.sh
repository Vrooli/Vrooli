#!/usr/bin/env bash
set -euo pipefail

#############################################################################
# Safe App Starter - Primary entry point with comprehensive safety checks
# 
# This script performs system health checks before starting the orchestrator
# to prevent fork bombs and system overload.
#############################################################################

echo "========================================="
echo "Vrooli Safe App Starter"
echo "========================================="
echo

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/scenarios/tools/orchestrator"
PREFLIGHT_CHECK="${SCRIPT_DIR}/preflight-check.sh"

if [[ -f "$PREFLIGHT_CHECK" ]]; then
    echo "Running pre-flight safety checks..."
    if ! bash "$PREFLIGHT_CHECK"; then
        echo -e "${RED}Pre-flight checks failed. Aborting for safety.${NC}"
        exit 1
    fi
    echo
fi

# Safety check 1: Check if orchestrator is already running
if [[ -f /tmp/vrooli-orchestrator.pid ]]; then
    PID=$(cat /tmp/vrooli-orchestrator.pid 2>/dev/null || echo "unknown")
    if kill -0 "$PID" 2>/dev/null; then
        echo -e "${RED}ERROR: Orchestrator is already running (PID: $PID)${NC}"
        echo "To stop it: kill $PID"
        exit 1
    else
        echo -e "${YELLOW}Cleaning up stale PID file${NC}"
        rm -f /tmp/vrooli-orchestrator.pid /tmp/vrooli-orchestrator.lock
    fi
fi

# Safety check 2: Check system process count
PROCESS_COUNT=$(ps aux | wc -l)
echo "Current system processes: $PROCESS_COUNT"

if [[ $PROCESS_COUNT -gt 500 ]]; then
    echo -e "${YELLOW}WARNING: High process count detected ($PROCESS_COUNT processes)${NC}"
    
    if [[ $PROCESS_COUNT -gt 1000 ]]; then
        echo -e "${RED}CRITICAL: Extremely high process count!${NC}"
        echo "This may indicate a fork bomb or system issue."
        echo
        echo "Top processes by count:"
        ps aux | awk '{print $11}' | sort | uniq -c | sort -rn | head -10
        echo
        read -p "Override safety and continue anyway? (type 'yes' to confirm): " response
        if [[ "$response" != "yes" ]]; then
            echo "Aborted for safety."
            exit 1
        fi
    fi
fi

# Safety check 3: Check available memory
if command -v free >/dev/null 2>&1; then
    AVAILABLE_MEM=$(free -m | awk 'NR==2{print $7}')
    echo "Available memory: ${AVAILABLE_MEM}MB"
    
    if [[ $AVAILABLE_MEM -lt 500 ]]; then
        echo -e "${YELLOW}WARNING: Low memory available (${AVAILABLE_MEM}MB)${NC}"
        echo "Apps may fail to start or system may become unresponsive."
        read -p "Continue anyway? (y/n): " response
        if [[ "$response" != "y" ]]; then
            echo "Aborted."
            exit 1
        fi
    fi
fi

# Safety check 4: Check for recursive call
if [[ "${VROOLI_ORCHESTRATOR_RUNNING:-}" == "1" ]]; then
    echo -e "${RED}ERROR: Recursive orchestrator call detected!${NC}"
    echo "This is a fork bomb prevention mechanism."
    exit 1
fi

# Safety check 5: Clean up old PID files
echo "Cleaning up old PID files..."
if [[ -d /tmp/vrooli-apps ]]; then
    for pid_file in /tmp/vrooli-apps/*.pid; do
        if [[ -f "$pid_file" ]]; then
            PID=$(cat "$pid_file" 2>/dev/null || echo "0")
            if ! kill -0 "$PID" 2>/dev/null; then
                echo "  Removing stale PID file: $(basename "$pid_file")"
                rm -f "$pid_file"
            fi
        fi
    done
fi

# Safety check 6: Verify orchestrator exists
ORCHESTRATOR_DIR="$SCRIPT_DIR"
ORCHESTRATOR="${ORCHESTRATOR_DIR}/enhanced_orchestrator.py"

if [[ ! -f "$ORCHESTRATOR" ]]; then
    echo -e "${RED}ERROR: Orchestrator not found at $ORCHESTRATOR${NC}"
    exit 1
fi

# All checks passed - start the orchestrator
echo -e "${GREEN}All safety checks passed!${NC}"
echo "Starting safe orchestrator..."
echo

# Export environment variable to prevent recursive calls
export VROOLI_ORCHESTRATOR_RUNNING=0  # Set to 0 here, orchestrator will check internally

# Determine if we should use virtual environment
VENV_DIR="${ORCHESTRATOR_DIR}/venv"
if [[ -d "$VENV_DIR" ]] && [[ -f "${VENV_DIR}/bin/activate" ]]; then
    echo "Using Python virtual environment"
    source "${VENV_DIR}/bin/activate"
else
    echo "Using system Python"
fi

# Check for psutil
if ! python3 -c "import psutil" 2>/dev/null; then
    echo -e "${YELLOW}Installing required dependency: psutil${NC}"
    pip3 install psutil || {
        echo -e "${RED}Failed to install psutil. Please install manually: pip3 install psutil${NC}"
        exit 1
    }
fi

# Set orchestrator port from project service.json or fallback
export ORCHESTRATOR_PORT="${ORCHESTRATOR_PORT:-9500}"

# Pass all arguments to the orchestrator
# Check if we should run in background (when called from vrooli develop)
if [[ "${VROOLI_DEVELOP_MODE:-}" == "1" ]] || [[ "$*" == *"--background"* ]]; then
    echo "Starting orchestrator in background mode on port $ORCHESTRATOR_PORT..."
    ORCHESTRATOR_PORT=$ORCHESTRATOR_PORT python3 "$ORCHESTRATOR" "$@" > /tmp/vrooli-orchestrator.log 2>&1 &
    ORCHESTRATOR_PID=$!
    echo "$ORCHESTRATOR_PID" > /tmp/vrooli-orchestrator.pid
    echo -e "${GREEN}✓ Orchestrator started in background (PID: $ORCHESTRATOR_PID)${NC}"
    echo "Logs available at: /tmp/vrooli-orchestrator.log"
else
    # Run in foreground (for debugging or direct calls)
    echo "Starting orchestrator in foreground mode on port $ORCHESTRATOR_PORT..."
    exec env ORCHESTRATOR_PORT=$ORCHESTRATOR_PORT python3 "$ORCHESTRATOR" "$@"
fi