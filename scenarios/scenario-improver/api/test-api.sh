#!/bin/bash
# Test script for scenario-improver API

echo "Testing Scenario Improver API"
echo "=============================="

# Kill any existing instances
pkill -f scenario-improver-api 2>/dev/null
sleep 1

# Use provided API_PORT or find available port
if [[ -z "$API_PORT" ]]; then
    # Find an available port starting from 30150
    API_PORT=30150
    while netstat -tuln 2>/dev/null | grep -q ":$API_PORT "; do
        API_PORT=$((API_PORT + 1))
    done
    export API_PORT
fi

echo "Starting API on port $API_PORT..."
./scenario-improver-api &> /tmp/scenario-improver.log &
API_PID=$!

# Wait for API to start
sleep 3

# Test health endpoint
echo -n "Testing /health endpoint... "
if curl -s http://localhost:$API_PORT/health | grep -q "healthy"; then
    echo "✓ Success"
else
    echo "✗ Failed"
fi

# Test queue status endpoint
echo -n "Testing /api/queue/status endpoint... "
if curl -s http://localhost:$API_PORT/api/queue/status | grep -q "pending"; then
    echo "✓ Success"
else
    echo "✗ Failed"
fi

# Test scenarios list endpoint
echo -n "Testing /api/scenarios/list endpoint... "
if curl -s http://localhost:$API_PORT/api/scenarios/list | grep -q "scenarios"; then
    echo "✓ Success"
else
    echo "✗ Failed"
fi

# Kill the API process
kill $API_PID 2>/dev/null

echo ""
echo "Test complete! Check /tmp/scenario-improver.log for API logs."