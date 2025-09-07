#!/usr/bin/env bash
################################################################################
# Clean Orchestrator Launcher
# Starts the enhanced orchestrator with proper checks and logging
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/scenarios/tools/orchestrator"
ORCHESTRATOR="${SCRIPT_DIR}/enhanced_orchestrator.py"
ORCHESTRATOR_PORT="${ORCHESTRATOR_PORT:-9500}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Add orchestrator directory to Python path for imports
export PYTHONPATH="${SCRIPT_DIR}:${PYTHONPATH:-}"

# Check if orchestrator is already running
if python3 -c "from pid_manager import OrchestrationLockManager; import sys; m = OrchestrationLockManager(); sys.exit(0 if m._find_running_orchestrator() is None else 1)" 2>/dev/null; then
    echo -e "${GREEN}✓ No existing orchestrator detected${NC}"
else
    EXISTING_PID=$(python3 -c "from pid_manager import OrchestrationLockManager; m = OrchestrationLockManager(); r = m._find_running_orchestrator(); print(r.get('pid', 'unknown') if r else '')" 2>/dev/null || echo "unknown")
    
    if [[ "$*" == *"--force"* ]]; then
        echo -e "${YELLOW}Force flag detected, stopping existing orchestrator (PID: $EXISTING_PID)${NC}"
        kill "$EXISTING_PID" 2>/dev/null || true
        sleep 2
    else
        echo -e "${YELLOW}Orchestrator already running (PID: $EXISTING_PID) on port $ORCHESTRATOR_PORT${NC}"
        echo "Use --force to restart it"
        exit 0
    fi
fi

# Ensure dependencies are installed
if ! python3 -c "import psutil" 2>/dev/null; then
    echo -e "${YELLOW}Installing required dependency: psutil${NC}"
    pip3 install psutil || {
        echo -e "${RED}Failed to install psutil. Please install manually: pip3 install psutil${NC}"
        exit 1
    }
fi

if ! python3 -c "import fastapi, uvicorn" 2>/dev/null; then
    echo -e "${YELLOW}Installing required dependencies: fastapi uvicorn${NC}"
    pip3 install fastapi uvicorn || {
        echo -e "${RED}Failed to install dependencies. Please install manually: pip3 install fastapi uvicorn${NC}"
        exit 1
    }
fi

# Start the orchestrator
echo -e "${GREEN}Starting orchestrator on port $ORCHESTRATOR_PORT...${NC}"

if [[ "$*" == *"--background"* ]]; then
    # Background mode
    python3 "$ORCHESTRATOR" --port "$ORCHESTRATOR_PORT" "$@" > /tmp/vrooli-orchestrator.log 2>&1 &
    ORCHESTRATOR_PID=$!
    
    # Wait briefly to check if it started successfully
    sleep 3
    if kill -0 "$ORCHESTRATOR_PID" 2>/dev/null; then
        echo -e "${GREEN}✓ Orchestrator started in background (PID: $ORCHESTRATOR_PID)${NC}"
        echo "Logs available at: /tmp/vrooli-orchestrator.log"
        
        # If --start-all flag is present, trigger scenario startup
        if [[ "$*" == *"--start-all"* ]]; then
            echo "Triggering scenario startup..."
            sleep 2
            if curl -sf -X POST "http://localhost:$ORCHESTRATOR_PORT/scenarios/start-all" >/dev/null 2>&1; then
                echo -e "${GREEN}✓ Scenario startup triggered${NC}"
            else
                echo -e "${YELLOW}⚠️  Failed to trigger scenario startup${NC}"
            fi
        fi
    else
        echo -e "${RED}✗ Orchestrator failed to start${NC}"
        echo "Check logs at: /tmp/vrooli-orchestrator.log"
        tail -n 20 /tmp/vrooli-orchestrator.log 2>/dev/null || true
        exit 1
    fi
else
    # Foreground mode (for debugging)
    exec python3 "$ORCHESTRATOR" --port "$ORCHESTRATOR_PORT" "$@"
fi