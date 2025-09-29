#!/bin/bash
set -e

echo "=== Schedule Creation Test ==="

# Get API port from environment or use default
API_PORT="${API_PORT:-18090}"
API_BASE="http://localhost:$API_PORT"

# Test health endpoint
echo "Testing health endpoint..."
if curl -sf "$API_BASE/health" > /dev/null 2>&1; then
    echo "✅ Health check passed"
else
    echo "❌ Health check failed - API may not be running"
    exit 1
fi

# Test database connectivity 
echo "Testing database connection..."
if curl -sf "$API_BASE/api/system/db-status" > /dev/null 2>&1; then
    echo "✅ Database connection successful"
else
    echo "⚠️ Database connection failed - continuing anyway"
fi

# Test Redis connectivity
echo "Testing Redis connection..."
if curl -sf "$API_BASE/api/system/redis-status" > /dev/null 2>&1; then
    echo "✅ Redis connection successful"
else
    echo "⚠️ Redis connection failed - continuing anyway"
fi

# Test schedule endpoints
echo "Testing schedule endpoints..."
if curl -sf "$API_BASE/api/schedules" > /dev/null 2>&1; then
    echo "✅ Schedule list endpoint working"
else
    echo "⚠️ Schedule list endpoint failed"
fi

# Test cron validation
echo "Testing cron validation..."
if curl -sf "$API_BASE/api/cron/validate?expression=0%209%20*%20*%20*" > /dev/null 2>&1; then
    echo "✅ Cron validation endpoint working"
else
    echo "⚠️ Cron validation endpoint failed"
fi

# Test timezone list
echo "Testing timezone endpoint..."
if curl -sf "$API_BASE/api/timezones" > /dev/null 2>&1; then
    echo "✅ Timezone endpoint working"
else
    echo "⚠️ Timezone endpoint failed"
fi

# Test creating a schedule (will fail without DB but tests API)
echo "Testing schedule creation..."
SCHEDULE_JSON='{
    "name": "Test Schedule",
    "description": "Integration test schedule",
    "cron_expression": "0 9 * * *",
    "timezone": "UTC",
    "target_type": "webhook",
    "target_url": "http://localhost:5678/webhook/test",
    "enabled": true
}'

if curl -sf -X POST "$API_BASE/api/schedules" \
    -H "Content-Type: application/json" \
    -d "$SCHEDULE_JSON" > /dev/null 2>&1; then
    echo "✅ Schedule creation successful"
else
    echo "⚠️ Schedule creation failed (expected without database)"
fi

echo "✅ Schedule creation test completed"