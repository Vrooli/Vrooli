#!/bin/bash
# Test script for the new resource-improver endpoints

echo "Testing Resource Improver New Endpoints"
echo "======================================="

# Check if API port is provided
if [[ -z "$API_PORT" ]]; then
    echo "Error: API_PORT environment variable not set"
    echo "Usage: API_PORT=15001 $0"
    exit 1
fi

PORT=$API_PORT
BASE_URL="http://localhost:$PORT"
echo "Testing API at: $BASE_URL"

# Test 1: Submit improvement via /api/improvement/submit
echo -n "1. Testing improvement submission... "
RESPONSE=$(curl -s -X POST "$BASE_URL/api/improvement/submit" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test improvement from API",
    "description": "This is a test improvement submitted via endpoint",
    "type": "v2-compliance",
    "target": "postgres",
    "priority": "high"
  }' 2>/dev/null)

if echo "$RESPONSE" | grep -q "created"; then
    echo "✓ Success"
    ITEM_ID=$(echo "$RESPONSE" | grep -o '"item_id":"[^"]*"' | cut -d'"' -f4)
    echo "   Created item: $ITEM_ID"
else
    echo "✗ Failed"
    echo "   Response: $RESPONSE"
fi

# Test 2: Submit resource report via /api/reports
echo -n "2. Testing resource report submission... "
RESPONSE=$(curl -s -X POST "$BASE_URL/api/reports" \
  -H "Content-Type: application/json" \
  -d '{
    "resource_name": "redis",
    "issue_type": "health-check",
    "description": "Health check taking too long",
    "context": {"timeout": 30000, "expected": 5000}
  }' 2>/dev/null)

if echo "$RESPONSE" | grep -q "received"; then
    echo "✓ Success"
    echo "   $(echo "$RESPONSE" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)"
else
    echo "✗ Failed"
    echo "   Response: $RESPONSE"
fi

# Test 3: Analyze resource via path parameter endpoint
echo -n "3. Testing resource analysis (GET /api/resources/postgres/analyze)... "
RESPONSE=$(curl -s "$BASE_URL/api/resources/postgres/analyze" 2>/dev/null)
if echo "$RESPONSE" | grep -q "analysis" && echo "$RESPONSE" | grep -q "v2_compliance_score"; then
    echo "✓ Success"
    V2_SCORE=$(echo "$RESPONSE" | grep -o '"v2_compliance_score":[0-9.]*' | cut -d':' -f2)
    echo "   v2.0 compliance score: $V2_SCORE%"
else
    echo "✗ Failed"
    echo "   Response: $RESPONSE"
fi

# Test 4: Check resource status via path parameter endpoint
echo -n "4. Testing resource status (GET /api/resources/postgres/status)... "
RESPONSE=$(curl -s "$BASE_URL/api/resources/postgres/status" 2>/dev/null)
if echo "$RESPONSE" | grep -q "status" && echo "$RESPONSE" | grep -q "health"; then
    echo "✓ Success"
    STATUS=$(echo "$RESPONSE" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    HEALTH=$(echo "$RESPONSE" | grep -o '"health":"[^"]*"' | cut -d'"' -f4)
    echo "   Status: $STATUS, Health: $HEALTH"
else
    echo "✗ Failed"
    echo "   Response: $RESPONSE"
fi

# Test 5: Test resource metrics endpoint
echo -n "5. Testing resource metrics endpoint... "
RESPONSE=$(curl -s "$BASE_URL/api/resource/metrics?resource=qdrant" 2>/dev/null)
if echo "$RESPONSE" | grep -q "metrics"; then
    echo "✓ Success"
else
    echo "✗ Failed"
    echo "   Response: $RESPONSE"
fi

# Test 6: Test invalid resource name
echo -n "6. Testing invalid resource handling... "
RESPONSE=$(curl -s "$BASE_URL/api/resources/nonexistent/status" 2>/dev/null)
if echo "$RESPONSE" | grep -q "status"; then
    echo "✓ Success (handles missing resources gracefully)"
else
    echo "✗ Failed to handle missing resource"
fi

echo ""
echo "New endpoints testing complete!"