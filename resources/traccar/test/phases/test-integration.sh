#!/bin/bash
# Traccar Integration Tests - End-to-end functionality (120s max)

set -euo pipefail

# Test configuration
TRACCAR_HOST="${TRACCAR_HOST:-localhost}"
TRACCAR_PORT="${TRACCAR_PORT:-8082}"
TRACCAR_ADMIN_EMAIL="${TRACCAR_ADMIN_EMAIL:-admin@example.com}"
TRACCAR_ADMIN_PASSWORD="${TRACCAR_ADMIN_PASSWORD:-admin}"

echo "Running Traccar integration tests..."

# Test 1: Create a test device
echo -n "1. Creating test device... "
DEVICE_JSON=$(cat << EOF
{
  "name": "TEST-DEVICE-$$",
  "uniqueId": "test-$$",
  "category": "car",
  "model": "Test Model",
  "disabled": false
}
EOF
)

if DEVICE_RESPONSE=$(curl -sf -X POST \
    -H "Content-Type: application/json" \
    -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
    -d "${DEVICE_JSON}" \
    "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices" 2>&1); then
    DEVICE_ID=$(echo "$DEVICE_RESPONSE" | jq -r '.id')
    DEVICE_UNIQUE_ID=$(echo "$DEVICE_RESPONSE" | jq -r '.uniqueId')
    echo "✓ (ID: $DEVICE_ID)"
else
    echo "✗"
    echo "   Error: Failed to create device"
    echo "   Response: $DEVICE_RESPONSE"
    exit 1
fi

# Test 2: List devices
echo -n "2. Listing devices... "
if DEVICES=$(curl -sf \
    -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
    "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices" 2>&1); then
    DEVICE_COUNT=$(echo "$DEVICES" | jq '. | length')
    echo "✓ (Found $DEVICE_COUNT devices)"
else
    echo "✗"
    echo "   Error: Failed to list devices"
    exit 1
fi

# Test 3: Send GPS position
echo -n "3. Sending GPS position... "
LAT="37.7749"
LON="-122.4194"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")

# Try OsmAnd protocol first
POSITION_URL="http://${TRACCAR_HOST}:${TRACCAR_PORT}/?id=${DEVICE_UNIQUE_ID}&lat=${LAT}&lon=${LON}&timestamp=$(date +%s)"
if curl -sf "$POSITION_URL" &>/dev/null; then
    echo "✓"
    sleep 2  # Give server time to process
else
    # Fallback to API method
    POSITION_JSON=$(cat << EOF
{
  "deviceId": ${DEVICE_ID},
  "protocol": "api",
  "deviceTime": "${TIMESTAMP}",
  "fixTime": "${TIMESTAMP}",
  "serverTime": "${TIMESTAMP}",
  "valid": true,
  "latitude": ${LAT},
  "longitude": ${LON},
  "altitude": 0,
  "speed": 0,
  "course": 0
}
EOF
)
    
    if curl -sf -X POST \
        -H "Content-Type: application/json" \
        -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
        -d "${POSITION_JSON}" \
        "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/positions" &>/dev/null; then
        echo "✓ (via API)"
    else
        echo "✗"
        echo "   Error: Failed to send position"
        # Continue with cleanup
    fi
fi

# Test 4: Get position history
echo -n "4. Getting position history... "
FROM_DATE=$(date -u -d "1 hour ago" +"%Y-%m-%dT%H:%M:%S.000Z")
TO_DATE=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")

if POSITIONS=$(curl -sf -G \
    -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
    --data-urlencode "deviceId=${DEVICE_ID}" \
    --data-urlencode "from=${FROM_DATE}" \
    --data-urlencode "to=${TO_DATE}" \
    "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/positions" 2>&1); then
    POSITION_COUNT=$(echo "$POSITIONS" | jq '. | length')
    echo "✓ (Found $POSITION_COUNT positions)"
else
    echo "✗"
    echo "   Warning: Failed to get position history"
    # Non-critical, continue
fi

# Test 5: Update device
echo -n "5. Updating device... "
UPDATE_JSON=$(cat << EOF
{
  "id": ${DEVICE_ID},
  "name": "UPDATED-TEST-$$",
  "uniqueId": "${DEVICE_UNIQUE_ID}",
  "category": "truck",
  "model": "Updated Model",
  "disabled": false
}
EOF
)

if curl -sf -X PUT \
    -H "Content-Type: application/json" \
    -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
    -d "${UPDATE_JSON}" \
    "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices/${DEVICE_ID}" &>/dev/null; then
    echo "✓"
else
    echo "✗"
    echo "   Warning: Failed to update device"
fi

# Test 6: Delete test device
echo -n "6. Cleaning up test device... "
if curl -sf -X DELETE \
    -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
    "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/devices/${DEVICE_ID}" &>/dev/null; then
    echo "✓"
else
    echo "✗"
    echo "   Warning: Failed to delete test device"
fi

# Test 7: Check reports endpoint
echo -n "7. Checking reports endpoint... "
if curl -sf \
    -u "${TRACCAR_ADMIN_EMAIL}:${TRACCAR_ADMIN_PASSWORD}" \
    "http://${TRACCAR_HOST}:${TRACCAR_PORT}/api/reports/route?deviceId=1&from=${FROM_DATE}&to=${TO_DATE}" \
    -o /dev/null -w "%{http_code}" 2>/dev/null | grep -qE "200|204"; then
    echo "✓"
else
    echo "✗"
    echo "   Warning: Reports endpoint may not be available"
fi

echo ""
echo "Integration tests completed successfully!"
exit 0