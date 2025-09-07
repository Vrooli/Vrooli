#!/bin/bash
# Test script for resource-improver API

echo "Testing Resource Improver API"
echo "=============================="

# Kill any existing instances
pkill -f resource-improver-api 2>/dev/null
sleep 1

# Use provided API_PORT or find available port
if [[ -z "$API_PORT" ]]; then
    # Find an available port starting from 15001
    API_PORT=15001
    while netstat -tuln 2>/dev/null | grep -q ":$API_PORT "; do
        API_PORT=$((API_PORT + 1))
    done
    export API_PORT
fi

echo "Starting API on port $API_PORT..."
./resource-improver-api &> /tmp/resource-improver.log &
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

# Test resources list endpoint
echo -n "Testing /api/resources/list endpoint... "
if curl -s http://localhost:$API_PORT/api/resources/list | grep -q "resources"; then
    echo "✓ Success"
else
    echo "✗ Failed"
fi

# Test new resource analyze endpoint (with path parameter)
echo -n "Testing /api/resources/postgres/analyze endpoint... "
if curl -s http://localhost:$API_PORT/api/resources/postgres/analyze | grep -q "analysis"; then
    echo "✓ Success"
else
    echo "✗ Failed"
fi

# Test new resource status endpoint
echo -n "Testing /api/resources/postgres/status endpoint... "
if curl -s http://localhost:$API_PORT/api/resources/postgres/status | grep -q "status"; then
    echo "✓ Success"
else
    echo "✗ Failed"
fi

# Test reports submission endpoint
echo -n "Testing /api/reports endpoint... "
RESPONSE=$(curl -s -X POST http://localhost:$API_PORT/api/reports \
  -H "Content-Type: application/json" \
  -d '{
    "resource_name": "test-resource", 
    "issue_type": "performance",
    "description": "Test issue report",
    "context": {"severity": "low"}
  }' 2>/dev/null)
if echo "$RESPONSE" | grep -q "received"; then
    echo "✓ Success"
else
    echo "✗ Failed"
fi

# Test improvement submission endpoint
echo -n "Testing /api/improvement/submit endpoint... "
RESPONSE=$(curl -s -X POST http://localhost:$API_PORT/api/improvement/submit \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test API improvement",
    "target": "postgres", 
    "type": "health-check",
    "priority": "medium"
  }' 2>/dev/null)
if echo "$RESPONSE" | grep -q "created"; then
    echo "✓ Success"
else
    echo "✗ Failed"
fi

# Kill the API process
kill $API_PID 2>/dev/null

echo ""
echo "Test complete! Check /tmp/resource-improver.log for API logs."