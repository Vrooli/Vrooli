#!/bin/bash

# Code Smell Detector Startup Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Starting Code Smell Detector..."

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "Installing Node.js dependencies..."
    npm install
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Please install Go 1.21+ to run the API server."
    echo "Visit: https://golang.org/dl/"
    exit 1
fi

# Build Go API if needed
if [ ! -f "api/code-smell-api" ]; then
    echo "Building API server..."
    cd api
    go mod download
    go build -o code-smell-api main.go
    cd ..
fi

# Start API server
echo "Starting API server on port 8090..."
cd api
./code-smell-api &
API_PID=$!
cd ..

# Start UI server
echo "Starting UI on port 3090..."
cd ui
python3 -m http.server 3090 &
UI_PID=$!
cd ..

echo ""
echo "✓ Code Smell Detector is running!"
echo ""
echo "  API: http://localhost:8090/api/v1"
echo "  UI:  http://localhost:3090"
echo "  CLI: ./cli/code-smell --help"
echo ""
echo "Press Ctrl+C to stop..."

# Cleanup function
cleanup() {
    echo "Stopping services..."
    kill $API_PID 2>/dev/null || true
    kill $UI_PID 2>/dev/null || true
    exit 0
}

# Set up signal handler
trap cleanup SIGINT SIGTERM

# Wait for processes
wait