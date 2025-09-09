#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_PORT="${IMAGE_TOOLS_API_PORT:-8080}"
UI_PORT="${IMAGE_TOOLS_UI_PORT:-3000}"

echo "Starting Image Tools - Digital Darkroom..."

# Install CLI
echo "Installing CLI..."
"$SCRIPT_DIR/cli/install.sh"

# Start API server
echo "Starting API server on port $API_PORT..."
cd "$SCRIPT_DIR/api"
PORT=$API_PORT go run main.go &
API_PID=$!

# Start UI server
echo "Starting UI server on port $UI_PORT..."
cd "$SCRIPT_DIR/ui"
python3 -m http.server $UI_PORT &> /dev/null &
UI_PID=$!

echo ""
echo "✓ Image Tools is running!"
echo "  API: http://localhost:$API_PORT"
echo "  UI:  http://localhost:$UI_PORT"
echo "  CLI: image-tools help"
echo ""
echo "Press Ctrl+C to stop..."

# Handle shutdown
trap "echo 'Shutting down...'; kill $API_PID $UI_PID 2>/dev/null; exit" INT TERM

# Wait for processes
wait