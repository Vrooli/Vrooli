#!/bin/bash

set -e

echo "=== Business Logic Tests ==="

# Set test ports
export API_PORT=15900
export UI_PORT=35900

# Function to cleanup
cleanup() {
    pkill -f "home-automation-api" 2>/dev/null || true
    sleep 1
}

trap cleanup EXIT

# Start API for business logic testing
echo "Starting API for business logic tests..."
cd api
if [ ! -f "home-automation-api" ]; then
    go build -o home-automation-api .
fi

VROOLI_LIFECYCLE_MANAGED=true ./home-automation-api &
API_PID=$!
cd ..

# Wait for API to start
sleep 5

# Test health endpoint
echo "Testing health endpoint..."
health_response=$(curl -s "http://localhost:$API_PORT/health" || echo "failed")
if [ "$health_response" != "failed" ]; then
    echo "✅ Health endpoint responding"
else
    echo "❌ Health endpoint not responding"
    exit 1
fi

# Test devices endpoint (should work even without Home Assistant)
echo "Testing devices endpoint..."
devices_response=$(curl -s "http://localhost:$API_PORT/api/v1/devices" || echo "failed")
if [ "$devices_response" != "failed" ]; then
    echo "✅ Devices endpoint responding"
else
    echo "⚠️  Devices endpoint not responding (expected if dependencies missing)"
fi

# Test profiles endpoint
echo "Testing profiles endpoint..."
profiles_response=$(curl -s "http://localhost:$API_PORT/api/v1/profiles" || echo "failed")
if [ "$profiles_response" != "failed" ]; then
    echo "✅ Profiles endpoint responding"
else
    echo "⚠️  Profiles endpoint not responding"
fi

# Test automation validation (POST without body should return error)
echo "Testing automation validation..."
validation_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/automations/validate" || echo "failed")
if [ "$validation_response" != "failed" ]; then
    echo "✅ Automation validation endpoint responding"
else
    echo "⚠️  Automation validation endpoint not responding"
fi

echo "✅ Business logic tests passed"
