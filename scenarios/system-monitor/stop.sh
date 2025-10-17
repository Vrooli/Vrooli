#!/bin/bash
# System Monitor Stop Script

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${RED}🛑 Stopping System Monitor...${NC}"

# Stop API
if [ -f .api.pid ]; then
    PID=$(cat .api.pid)
    if kill -0 $PID 2>/dev/null; then
        kill $PID
        echo "✅ API stopped (PID: $PID)"
    fi
    rm -f .api.pid
else
    # Try to find and kill by name
    pkill -f "system-monitor-api" 2>/dev/null && echo "✅ API stopped"
fi

# Stop UI
if [ -f .ui.pid ]; then
    PID=$(cat .ui.pid)
    if kill -0 $PID 2>/dev/null; then
        kill $PID
        echo "✅ UI stopped (PID: $PID)"
    fi
    rm -f .ui.pid
else
    # Try to find process on port 3003
    fuser -k 3003/tcp 2>/dev/null && echo "✅ UI stopped"
fi

# Clean up any remaining processes
pkill -f "system-monitor" 2>/dev/null || true

echo -e "${GREEN}✨ System Monitor stopped${NC}"