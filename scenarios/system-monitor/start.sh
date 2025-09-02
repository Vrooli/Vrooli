#!/bin/bash
# System Monitor Quick Start Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Load environment variables
if [ -f .env ]; then
    set -a
    source .env
    set +a
    echo "✅ Loaded environment configuration"
fi

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}🚀 Starting System Monitor...${NC}"

# Function to check if port is in use
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 0  # Port is in use
    else
        return 1  # Port is free
    fi
}

# Kill existing processes if needed
echo "Cleaning up existing processes..."
pkill -f "system-monitor-api" 2>/dev/null || true
if check_port 8080; then
    echo -e "${YELLOW}⚠️  Port 8080 is in use, attempting to free it...${NC}"
    fuser -k 8080/tcp 2>/dev/null || true
    sleep 1
fi

# Build API if needed
echo "Building API..."
cd api
if [ ! -f "system-monitor-api" ] || [ "main.go" -nt "system-monitor-api" ]; then
    go build -o system-monitor-api main.go
    echo -e "${GREEN}✅ API built successfully${NC}"
else
    echo "✅ API binary is up to date"
fi

# Start API
echo "Starting API on port ${API_PORT}..."
API_PORT=${API_PORT} PORT=${PORT} ./system-monitor-api &
API_PID=$!
echo "API started with PID: $API_PID"
cd ..

# Wait for API to be ready
echo "Waiting for API to be ready..."
for i in {1..10}; do
    if curl -s "http://localhost:${API_PORT}/health" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ API is ready${NC}"
        break
    fi
    sleep 1
done

# Start UI if not already running
if ! check_port ${UI_PORT}; then
    echo "Starting UI on port ${UI_PORT}..."
    cd ui
    if [ ! -d "node_modules" ]; then
        echo "Installing UI dependencies..."
        npm install express >/dev/null 2>&1
    fi
    PORT=${UI_PORT} API_PORT=${API_PORT} node server.js &
    UI_PID=$!
    echo "UI started with PID: $UI_PID"
    cd ..
else
    echo -e "${YELLOW}⚠️  UI already running on port ${UI_PORT}${NC}"
fi

# Create PID file for cleanup
echo "$API_PID" > .api.pid
[ ! -z "$UI_PID" ] && echo "$UI_PID" > .ui.pid

echo ""
echo -e "${GREEN}✨ System Monitor is running!${NC}"
echo ""
echo "📊 Dashboard: http://localhost:${UI_PORT}"
echo "🔧 API:       http://localhost:${API_PORT}/health"
echo ""
echo "CLI Commands:"
echo "  ./cli/system-monitor health       - Check system health"
echo "  ./cli/system-monitor metrics      - View current metrics"
echo "  ./cli/system-monitor investigate  - Trigger AI investigation"
echo "  ./cli/system-monitor watch        - Live monitoring mode"
echo ""
echo "To stop: ./stop.sh or press Ctrl+C"
echo ""

# Keep script running
trap 'echo "Stopping services..."; [ -f .api.pid ] && kill $(cat .api.pid) 2>/dev/null; [ -f .ui.pid ] && kill $(cat .ui.pid) 2>/dev/null; rm -f .*.pid; exit' INT TERM
wait