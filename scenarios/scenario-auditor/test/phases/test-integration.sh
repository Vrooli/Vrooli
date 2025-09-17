#!/bin/bash

set -e

echo "=== Integration Tests ==="

# Set test ports to avoid conflicts
export API_PORT=15999
export UI_PORT=35999

# Function to cleanup on exit
cleanup() {
    echo "Cleaning up test processes..."
    pkill -f "scenario-auditor-api" 2>/dev/null || true
    pkill -f "vite.*scenario-auditor" 2>/dev/null || true
    sleep 2
}

trap cleanup EXIT

# Test API startup and health check
echo "Testing API startup and health check..."

# Build API if needed
cd api
if [ ! -f "scenario-auditor-api" ]; then
    echo "Building API..."
    go build -o scenario-auditor-api .
fi

# Start API in background
echo "Starting API on port $API_PORT..."
VROOLI_LIFECYCLE_MANAGED=true ./scenario-auditor-api &
API_PID=$!
cd ..

# Wait for API to start
echo "Waiting for API to start..."
sleep 5

# Test health endpoint
echo "Testing health endpoint..."
if curl -s "http://localhost:$API_PORT/api/v1/health" | grep -q "healthy"; then
    echo "✅ API health check passed"
else
    echo "❌ API health check failed"
    exit 1
fi

# Test rules endpoint
echo "Testing rules endpoint..."
if curl -s "http://localhost:$API_PORT/api/v1/rules" | grep -q "rules"; then
    echo "✅ Rules endpoint accessible"
else
    echo "❌ Rules endpoint failed"
    exit 1
fi

# Test dashboard endpoint
echo "Testing dashboard endpoint..."
if curl -s "http://localhost:$API_PORT/api/v1/dashboard" | grep -q "overview"; then
    echo "✅ Dashboard endpoint accessible"
else
    echo "❌ Dashboard endpoint failed"
    exit 1
fi

# Test CLI integration
echo "Testing CLI integration..."
cd cli
if [ ! -f "scenario-auditor" ]; then
    echo "Building CLI..."
    go build -o scenario-auditor .
fi

# Test CLI health command
if ./scenario-auditor health; then
    echo "✅ CLI health command passed"
else
    echo "❌ CLI health command failed"
    exit 1
fi

# Test CLI rules command
if ./scenario-auditor rules | grep -q "rules"; then
    echo "✅ CLI rules command passed"
else
    echo "❌ CLI rules command failed"
    exit 1
fi
cd ..

# Test basic scan functionality
echo "Testing basic scan functionality..."
cd cli
if ./scenario-auditor scan current; then
    echo "✅ Basic scan functionality passed"
else
    echo "❌ Basic scan functionality failed"
    exit 1
fi
cd ..

# Test UI build (if Node.js available)
echo "Testing UI build..."
cd ui
if [ -f "package.json" ] && command -v node >/dev/null 2>&1; then
    if [ -d "node_modules" ]; then
        echo "Building UI..."
        npm run build
        echo "✅ UI build passed"
    else
        echo "⚠️  UI dependencies not installed, skipping UI build test"
    fi
else
    echo "⚠️  Node.js or package.json not found, skipping UI build test"
fi
cd ..

echo "=== Integration Tests Complete ==="