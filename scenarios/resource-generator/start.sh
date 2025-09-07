#!/bin/bash
# Start Resource Generator services

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Load environment
if [ -f "$SCENARIO_DIR/.env" ]; then
    export $(grep -v '^#' "$SCENARIO_DIR/.env" | xargs)
fi

# Check required environment variables
if [ -z "$API_PORT" ]; then
    echo "Error: API_PORT environment variable is not set"
    echo "Usage: API_PORT=<port> UI_PORT=<port> $0"
    exit 1
fi

if [ -z "$UI_PORT" ]; then
    echo "Error: UI_PORT environment variable is not set"
    echo "Usage: API_PORT=<port> UI_PORT=<port> $0"
    exit 1
fi

API_HOST="${API_HOST:-localhost}"

# Start API server
echo "Starting API server on port ${API_PORT}..."
cd "$SCENARIO_DIR/api"
./resource-generator-api &
API_PID=$!
echo "API server PID: $API_PID"

# Start UI server if Node is available
if command -v node >/dev/null 2>&1 && [ -f "$SCENARIO_DIR/ui/server.js" ]; then
    echo "Starting UI server on port ${UI_PORT}..."
    cd "$SCENARIO_DIR/ui"
    API_HOST="$API_HOST" node server.js &
    UI_PID=$!
    echo "UI server PID: $UI_PID"
else
    echo "UI server not started (Node.js not available)"
fi

echo ""
echo "Resource Generator is running!"
echo "  API: http://${API_HOST}:${API_PORT}"
echo "  UI:  http://${API_HOST}:${UI_PORT}"
echo ""
echo "Press Ctrl+C to stop..."

# Wait for interrupt
trap "kill $API_PID $UI_PID 2>/dev/null; exit" INT
wait