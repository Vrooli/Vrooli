#!/usr/bin/env bash

# Test data export functionality (JSON, CSV, YAML formats)
set -euo pipefail

# Color codes for output
BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}[TEST INFO]${NC} Starting Data Structurer export functionality tests..."

# Get API port dynamically
API_PORT=$(vrooli scenario port data-structurer API_PORT 2>/dev/null || echo "15769")
API_URL="http://localhost:${API_PORT}"

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to test endpoint
test_export() {
    local test_name="$1"
    local format="$2"
    local validation="$3"

    echo -e "${BLUE}[TEST INFO]${NC} Testing ${test_name}"

    local response
    if ! response=$(curl -sf "${API_URL}/api/v1/data/${SCHEMA_ID}?format=${format}&limit=2" 2>&1); then
        echo -e "${RED}[TEST FAIL]${NC} ${test_name} - Request failed"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi

    # Validate response content
    if echo "$response" | eval "$validation" &> /dev/null; then
        echo -e "${GREEN}[TEST PASS]${NC} ${test_name}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}[TEST FAIL]${NC} ${test_name} - Validation failed"
        echo "Response: $response"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test 1: Check API health
echo -e "${BLUE}[TEST INFO]${NC} Test 1: Checking API health"
if curl -sf "${API_URL}/health" > /dev/null; then
    echo -e "${GREEN}[TEST PASS]${NC} API is healthy"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}[TEST FAIL]${NC} API is not responding"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    exit 1
fi

# Test 2: Create test schema with data for export testing
echo -e "${BLUE}[TEST INFO]${NC} Test 2: Creating test schema for export"
SCHEMA_RESPONSE=$(curl -sf -X POST "${API_URL}/api/v1/schemas" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "export_test_'$(date +%s)'",
        "description": "Test schema for export validation",
        "schema_definition": {
            "name": "string",
            "email": "string",
            "age": "integer"
        }
    }')

SCHEMA_ID=$(echo "$SCHEMA_RESPONSE" | jq -r '.id')
if [[ -n "$SCHEMA_ID" && "$SCHEMA_ID" != "null" ]]; then
    echo -e "${GREEN}[TEST PASS]${NC} Schema created with ID: $SCHEMA_ID"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}[TEST FAIL]${NC} Failed to create schema"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    exit 1
fi

# Test 3: Add test data to schema
echo -e "${BLUE}[TEST INFO]${NC} Test 3: Adding test data for export"
for i in {1..3}; do
    curl -sf -X POST "${API_URL}/api/v1/process" \
        -H "Content-Type: application/json" \
        -d "{
            \"schema_id\": \"${SCHEMA_ID}\",
            \"input_type\": \"text\",
            \"input_data\": \"Person ${i}: Name: Test User ${i}, Email: test${i}@example.com, Age: $((20 + i))\"
        }" > /dev/null
done

# Wait for processing
sleep 2

DATA_COUNT=$(curl -sf "${API_URL}/api/v1/data/${SCHEMA_ID}" | jq '.data | length')
if [[ "$DATA_COUNT" -ge 1 ]]; then
    echo -e "${GREEN}[TEST PASS]${NC} Test data created (${DATA_COUNT} items)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}[TEST FAIL]${NC} Failed to create test data"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 4: Export as JSON (default format)
test_export "JSON export (default)" "" "jq -e '.data | length > 0'"

# Test 5: Export as CSV
test_export "CSV export format" "csv" "grep -q 'id,source_file_name,confidence_score'"

# Test 6: Export as YAML
test_export "YAML export format" "yaml" "grep -q 'confidence_score:'"

# Test 7: Validate CSV has correct confidence score format (not memory address)
echo -e "${BLUE}[TEST INFO]${NC} Test 7: Validate CSV confidence score format"
CSV_OUTPUT=$(curl -sf "${API_URL}/api/v1/data/${SCHEMA_ID}?format=csv&limit=1")
if echo "$CSV_OUTPUT" | grep -E '[0-9]+\.[0-9]{2}' | grep -v '0x' > /dev/null; then
    echo -e "${GREEN}[TEST PASS]${NC} CSV confidence scores are properly formatted"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}[TEST FAIL]${NC} CSV confidence scores may contain memory addresses"
    echo "CSV Output: $CSV_OUTPUT"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 8: Validate YAML structure
echo -e "${BLUE}[TEST INFO]${NC} Test 8: Validate YAML structure"
YAML_OUTPUT=$(curl -sf "${API_URL}/api/v1/data/${SCHEMA_ID}?format=yaml&limit=1")
if echo "$YAML_OUTPUT" | grep -q 'structured_data:' && echo "$YAML_OUTPUT" | grep -q 'confidence_score:'; then
    echo -e "${GREEN}[TEST PASS]${NC} YAML output has correct structure"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}[TEST FAIL]${NC} YAML structure validation failed"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 9: Test limit parameter across formats
echo -e "${BLUE}[TEST INFO]${NC} Test 9: Test limit parameter"
JSON_LIMIT=$(curl -sf "${API_URL}/api/v1/data/${SCHEMA_ID}?limit=2" | jq '.data | length')
if [[ "$JSON_LIMIT" -le 2 ]]; then
    echo -e "${GREEN}[TEST PASS]${NC} Limit parameter works (returned ${JSON_LIMIT} items)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}[TEST FAIL]${NC} Limit parameter not working correctly"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 10: Test pagination with offset
echo -e "${BLUE}[TEST INFO]${NC} Test 10: Test pagination with offset"
OFFSET_RESPONSE=$(curl -sf "${API_URL}/api/v1/data/${SCHEMA_ID}?limit=1&offset=1")
if echo "$OFFSET_RESPONSE" | jq -e '.pagination.offset == 1' > /dev/null; then
    echo -e "${GREEN}[TEST PASS]${NC} Pagination offset works correctly"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}[TEST FAIL]${NC} Pagination offset failed"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 11: Verify exports contain actual data
echo -e "${BLUE}[TEST INFO]${NC} Test 11: Verify exports contain actual data"
CSV_DATA=$(curl -sf "${API_URL}/api/v1/data/${SCHEMA_ID}?format=csv&limit=1" | wc -l)
YAML_DATA=$(curl -sf "${API_URL}/api/v1/data/${SCHEMA_ID}?format=yaml&limit=1" | wc -l)

if [[ "$CSV_DATA" -ge 2 ]] && [[ "$YAML_DATA" -ge 5 ]]; then
    echo -e "${GREEN}[TEST PASS]${NC} Exports contain actual data (CSV: ${CSV_DATA} lines, YAML: ${YAML_DATA} lines)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}[TEST FAIL]${NC} Exports may be empty"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Cleanup: Delete test schema
echo -e "${BLUE}[TEST INFO]${NC} Cleaning up test schema..."
curl -sf -X DELETE "${API_URL}/api/v1/schemas/${SCHEMA_ID}" > /dev/null

# Summary
echo ""
echo -e "${BLUE}[TEST INFO]${NC} Export Functionality Test Summary:"
echo "Total Tests: $((TESTS_PASSED + TESTS_FAILED))"
echo "Passed: $TESTS_PASSED"
echo "Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}[TEST PASS]${NC} All export tests passed!"
    exit 0
else
    echo -e "${RED}[TEST FAIL]${NC} Some export tests failed"
    exit 1
fi
