#!/bin/bash
# Test script for the new improvement submission endpoint

echo "Testing Scenario Improver Submission Endpoint"
echo "============================================="

# Check if API port is provided
if [[ -z "$API_PORT" ]]; then
    echo "Error: API_PORT environment variable not set"
    echo "Usage: API_PORT=30150 $0"
    exit 1
fi

PORT=$API_PORT
BASE_URL="http://localhost:$PORT"
echo "Testing API at: $BASE_URL"

# Test 1: Submit a basic improvement
echo -n "1. Testing improvement submission... "
RESPONSE=$(curl -s -X POST "$BASE_URL/api/improvement/submit" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test improvement from API",
    "description": "This is a test improvement submitted via the new endpoint",
    "type": "optimization",
    "target": "scenario-improver",
    "priority": "medium"
  }' 2>/dev/null)

if echo "$RESPONSE" | grep -q "created"; then
    echo "✓ Success"
    ITEM_ID=$(echo "$RESPONSE" | grep -o '"item_id":"[^"]*"' | cut -d'"' -f4)
    echo "   Created item: $ITEM_ID"
else
    echo "✗ Failed"
    echo "   Response: $RESPONSE"
fi

# Test 2: Submit with minimal fields
echo -n "2. Testing minimal submission... "
RESPONSE=$(curl -s -X POST "$BASE_URL/api/improvement/submit" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Minimal test improvement",
    "target": "resource-improver"
  }' 2>/dev/null)

if echo "$RESPONSE" | grep -q "created"; then
    echo "✓ Success (defaults applied)"
else
    echo "✗ Failed"
    echo "   Response: $RESPONSE"
fi

# Test 3: Test validation (missing required fields)
echo -n "3. Testing validation... "
RESPONSE=$(curl -s -X POST "$BASE_URL/api/improvement/submit" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Missing title and target"
  }' 2>/dev/null)

if echo "$RESPONSE" | grep -q "required"; then
    echo "✓ Validation working"
else
    echo "✗ Validation failed"
    echo "   Response: $RESPONSE"
fi

# Test 4: Check queue status
echo -n "4. Checking queue status... "
RESPONSE=$(curl -s "$BASE_URL/api/queue/status" 2>/dev/null)
if echo "$RESPONSE" | grep -q "pending"; then
    echo "✓ Queue status available"
    echo "   $RESPONSE"
else
    echo "✗ Queue status failed"
fi

echo ""
echo "Test complete! Check ../queue/pending/ for submitted items."