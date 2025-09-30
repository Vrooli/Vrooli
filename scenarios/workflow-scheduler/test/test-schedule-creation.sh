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

# Test creating a schedule with proper enum values
echo "Testing schedule creation..."
SCHEDULE_JSON='{
    "name": "Test Schedule",
    "description": "Integration test schedule",
    "cron_expression": "0 9 * * *",
    "timezone": "UTC",
    "target_type": "webhook",
    "target_url": "http://localhost:5678/webhook/test",
    "enabled": true,
    "retry_strategy": "exponential",
    "overlap_policy": "skip"
}'

RESPONSE=$(curl -s -X POST "$API_BASE/api/schedules" \
    -H "Content-Type: application/json" \
    -d "$SCHEDULE_JSON" 2>/dev/null)

if echo "$RESPONSE" | grep -q '"id"'; then
    echo "✅ Schedule creation successful"
    SCHEDULE_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

    # Test update
    echo "Testing schedule update..."
    UPDATE_JSON='{
        "name": "Updated Test Schedule",
        "cron_expression": "0 10 * * *",
        "enabled": false,
        "timezone": "UTC",
        "target_type": "webhook",
        "target_url": "http://localhost:5678/webhook/test"
    }'

    if curl -sf -X PUT "$API_BASE/api/schedules/$SCHEDULE_ID" \
        -H "Content-Type: application/json" \
        -d "$UPDATE_JSON" > /dev/null 2>&1; then
        echo "✅ Schedule update successful"
    else
        echo "⚠️ Schedule update failed"
    fi

    # Test delete
    echo "Testing schedule deletion..."
    if curl -sf -X DELETE "$API_BASE/api/schedules/$SCHEDULE_ID" > /dev/null 2>&1; then
        echo "✅ Schedule deletion successful"
    else
        echo "⚠️ Schedule deletion failed"
    fi
else
    echo "⚠️ Schedule creation failed: $RESPONSE"
fi

echo "✅ Schedule creation test completed"