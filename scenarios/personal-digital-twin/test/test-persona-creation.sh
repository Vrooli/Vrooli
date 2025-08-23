#!/bin/bash

# Test persona creation endpoint

set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://localhost:8200}"

# Test data
TEST_PERSONA_NAME="Test Persona $(date +%s)"
TEST_PERSONA_DESC="Test persona for integration testing"

echo "Testing persona creation..."

# Create persona
response=$(curl -sf -X POST "$API_BASE_URL/api/persona/create" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"$TEST_PERSONA_NAME\",
        \"description\": \"$TEST_PERSONA_DESC\",
        \"personality_traits\": {
            \"creativity\": 0.8,
            \"analytical\": 0.9
        }
    }")

# Check response
if echo "$response" | jq -e '.id' > /dev/null; then
    persona_id=$(echo "$response" | jq -r '.id')
    echo "✓ Persona created successfully with ID: $persona_id"
    
    # Verify persona can be retrieved
    persona_details=$(curl -sf "$API_BASE_URL/api/persona/$persona_id")
    if echo "$persona_details" | jq -e ".name == \"$TEST_PERSONA_NAME\"" > /dev/null; then
        echo "✓ Persona details retrieved successfully"
    else
        echo "✗ Failed to retrieve persona details"
        exit 1
    fi
    
    # Export persona ID for other tests
    echo "export TEST_PERSONA_ID='$persona_id'"
else
    echo "✗ Failed to create persona"
    echo "Response: $response"
    exit 1
fi

echo "Persona creation test passed!"