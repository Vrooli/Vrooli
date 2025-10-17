#!/usr/bin/env bash
# Integration tests for Fall Foliage Explorer
# Tests API endpoints and data flow

set -e

# Use registered ports for fall-foliage-explorer
FOLIAGE_API_PORT=17175
API_BASE="http://localhost:${FOLIAGE_API_PORT}"
FAIL_COUNT=0

echo "🔗 Running integration tests..."

# Test 1: Get all regions
echo "  [1/7] Testing GET /api/regions..."
REGIONS=$(curl -sf "$API_BASE/api/regions" | jq -r '.status')
if [[ "$REGIONS" != "success" ]]; then
    echo "    ❌ Failed to get regions"
    ((FAIL_COUNT++))
else
    echo "    ✅ Regions retrieved successfully"
fi

# Test 2: Get foliage data for a region
echo "  [2/7] Testing GET /api/foliage?region_id=1..."
FOLIAGE=$(curl -sf "$API_BASE/api/foliage?region_id=1" | jq -r '.status')
if [[ "$FOLIAGE" != "success" ]]; then
    echo "    ❌ Failed to get foliage data"
    ((FAIL_COUNT++))
else
    echo "    ✅ Foliage data retrieved successfully"
fi

# Test 3: Get weather data
echo "  [3/7] Testing GET /api/weather?region_id=1..."
WEATHER=$(curl -sf "$API_BASE/api/weather?region_id=1" | jq -r '.status')
if [[ "$WEATHER" != "success" ]]; then
    echo "    ❌ Failed to get weather data"
    ((FAIL_COUNT++))
else
    echo "    ✅ Weather data retrieved successfully"
fi

# Test 4: Submit user report
echo "  [4/7] Testing POST /api/reports..."
REPORT_DATA='{"region_id":1,"foliage_status":"peak","description":"Beautiful colors today!"}'
REPORT=$(curl -sf -X POST "$API_BASE/api/reports" \
    -H "Content-Type: application/json" \
    -d "$REPORT_DATA" | jq -r '.status')
if [[ "$REPORT" != "success" ]]; then
    echo "    ❌ Failed to submit user report"
    ((FAIL_COUNT++))
else
    echo "    ✅ User report submitted successfully"
fi

# Test 5: Get user reports
echo "  [5/7] Testing GET /api/reports?region_id=1..."
REPORTS=$(curl -sf "$API_BASE/api/reports?region_id=1" | jq -r '.status')
if [[ "$REPORTS" != "success" ]]; then
    echo "    ❌ Failed to get user reports"
    ((FAIL_COUNT++))
else
    echo "    ✅ User reports retrieved successfully"
fi

# Test 6: Trigger prediction
echo "  [6/7] Testing POST /api/predict..."
PREDICT_DATA='{"region_id":1}'
PREDICT=$(curl -sf -X POST "$API_BASE/api/predict" \
    -H "Content-Type: application/json" \
    -d "$PREDICT_DATA" | jq -r '.status')
if [[ "$PREDICT" != "success" ]]; then
    echo "    ❌ Failed to trigger prediction"
    ((FAIL_COUNT++))
else
    echo "    ✅ Prediction triggered successfully"
fi

# Test 7: Verify foliage data contains actual observations
echo "  [7/7] Testing foliage data quality..."
INTENSITY=$(curl -sf "$API_BASE/api/foliage?region_id=2" | jq -r '.data.color_intensity')
if [[ "$INTENSITY" =~ ^[0-9]+$ ]] && [[ "$INTENSITY" -ge 0 ]] && [[ "$INTENSITY" -le 10 ]]; then
    echo "    ✅ Foliage data quality check passed (intensity: $INTENSITY)"
else
    echo "    ❌ Invalid foliage data quality"
    ((FAIL_COUNT++))
fi

if [[ $FAIL_COUNT -eq 0 ]]; then
    echo "✅ All integration tests passed!"
    exit 0
else
    echo "❌ $FAIL_COUNT integration test(s) failed"
    exit 1
fi
