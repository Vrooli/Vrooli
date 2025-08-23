#!/usr/bin/env bash
# Test the analysis endpoint with mock data
set -euo pipefail

# Configuration
API_BASE="${SERVICE_PORT:-8090}"
API_URL="http://localhost:$API_BASE"

# Test data
TEST_TRANSCRIPTION_ID="00000000-0000-0000-0000-000000000001"
TEST_ANALYSIS_TYPE="summary"

echo "Testing analysis endpoint at $API_URL"

# Test the analysis endpoint
response=$(curl -s -X POST "$API_URL/api/transcriptions/$TEST_TRANSCRIPTION_ID/analyze" \
    -H "Content-Type: application/json" \
    -d "{\"analysis_type\": \"$TEST_ANALYSIS_TYPE\"}" || echo "ERROR")

if [[ "$response" == "ERROR" ]]; then
    echo "❌ Analysis endpoint test failed"
    exit 1
fi

# Check if response contains expected fields
if echo "$response" | grep -q "analysis_type"; then
    echo "✅ Analysis endpoint responded correctly"
else
    echo "❌ Analysis endpoint response format incorrect"
    echo "Response: $response"
    exit 1
fi