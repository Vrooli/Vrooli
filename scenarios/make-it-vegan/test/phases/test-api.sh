#!/bin/bash
# API tests for Make It Vegan

set -e

echo "Running API tests for Make It Vegan..."

# Get the API port
API_PORT="${API_PORT:-$(vrooli scenario info make-it-vegan --json 2>/dev/null | jq -r '.api.port' || echo '8200')}"
API_URL="http://localhost:${API_PORT}"

# Wait for API to be ready
echo "Waiting for API at ${API_URL}..."
for i in {1..30}; do
    if curl -sf "${API_URL}/health" > /dev/null 2>&1; then
        echo "✓ API is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ API failed to start"
        exit 1
    fi
    sleep 1
done

# Test health endpoint
echo "✓ Testing health endpoint..."
curl -sf "${API_URL}/health" || { echo "❌ Health check failed"; exit 1; }

# Test ingredient checking
echo "✓ Testing ingredient check endpoint..."
RESULT=$(curl -sf -X POST "${API_URL}/api/check" \
    -H "Content-Type: application/json" \
    -d '{"ingredients":"milk, eggs"}')
echo "$RESULT" | jq -e '.isVegan == false' > /dev/null || { echo "❌ Ingredient check failed"; exit 1; }

# Test vegan ingredients
echo "✓ Testing vegan ingredient check..."
RESULT=$(curl -sf -X POST "${API_URL}/api/check" \
    -H "Content-Type: application/json" \
    -d '{"ingredients":"flour, sugar, salt"}')
echo "$RESULT" | jq -e '.isVegan == true' > /dev/null || { echo "❌ Vegan check failed"; exit 1; }

# Test substitute endpoint
echo "✓ Testing substitute endpoint..."
RESULT=$(curl -sf -X POST "${API_URL}/api/substitute" \
    -H "Content-Type: application/json" \
    -d '{"ingredient":"milk","context":"baking"}')
echo "$RESULT" | jq -e '.alternatives | length > 0' > /dev/null || { echo "❌ Substitute check failed"; exit 1; }

# Test veganize endpoint
echo "✓ Testing veganize endpoint..."
RESULT=$(curl -sf -X POST "${API_URL}/api/veganize" \
    -H "Content-Type: application/json" \
    -d '{"recipe":"scrambled eggs with milk"}')
echo "$RESULT" | jq -e '.veganRecipe != null' > /dev/null || { echo "❌ Veganize check failed"; exit 1; }

# Test products endpoint
echo "✓ Testing products endpoint..."
curl -sf "${API_URL}/api/products" | jq -e '.dairy != null' > /dev/null || { echo "❌ Products check failed"; exit 1; }

# Test nutrition endpoint
echo "✓ Testing nutrition endpoint..."
curl -sf "${API_URL}/api/nutrition" | jq -e '.nutritionalInfo.protein != null' > /dev/null || { echo "❌ Nutrition check failed"; exit 1; }

echo "✅ API tests completed"
