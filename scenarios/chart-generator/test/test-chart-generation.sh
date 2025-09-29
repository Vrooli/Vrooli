#!/usr/bin/env bash

set -euo pipefail

# Test script for chart generation functionality

API_PORT="${API_PORT:-$(vrooli scenario port chart-generator | grep API_PORT | cut -d= -f2 || echo "20300")}"
API_URL="${CHART_API_URL:-http://localhost:${API_PORT}}"
TEST_OUTPUT="/tmp/chart-generator-test-$$"
mkdir -p "$TEST_OUTPUT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "📊 Testing Chart Generator API..."
echo "API URL: $API_URL"
echo "Test output: $TEST_OUTPUT"
echo ""

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Function to test chart generation
test_chart_generation() {
    local chart_type="$1"
    local data="$2"
    local formats="$3"
    local test_name="$4"
    
    echo -n "Testing $test_name... "
    
    local response
    response=$(curl -sf -X POST "$API_URL/api/v1/charts/generate" \
        -H "Content-Type: application/json" \
        -d "{
            \"chart_type\": \"$chart_type\",
            \"title\": \"Test $test_name\",
            \"data\": $data,
            \"export_formats\": $formats
        }" 2>/dev/null || echo '{"success": false, "error": {"message": "Request failed"}}')
    
    if echo "$response" | jq -e '.success == true' >/dev/null 2>&1; then
        echo -e "${GREEN}✅ PASSED${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        
        # Save response for debugging
        echo "$response" > "$TEST_OUTPUT/${chart_type}_response.json"
        return 0
    else
        echo -e "${RED}❌ FAILED${NC}"
        echo "  Error: $(echo "$response" | jq -r '.error.message // "Unknown error"')"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test health check
echo "1. Health Check"
if curl -sf "$API_URL/health" | jq -e '.status == "healthy"' >/dev/null 2>&1; then
    echo -e "  ${GREEN}✅ API is healthy${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "  ${RED}❌ API health check failed${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
echo ""

# Test P0 Requirements - Core chart types
echo "2. P0 Requirements - Core Chart Types"

# Bar chart
test_chart_generation "bar" \
    '[{"x": "Q1", "y": 100}, {"x": "Q2", "y": 150}, {"x": "Q3", "y": 200}]' \
    '["png", "svg"]' \
    "Bar Chart"

# Line chart  
test_chart_generation "line" \
    '[{"x": "Jan", "y": 10}, {"x": "Feb", "y": 20}, {"x": "Mar", "y": 15}]' \
    '["png", "svg"]' \
    "Line Chart"

# Pie chart
test_chart_generation "pie" \
    '[{"name": "Sales", "value": 45}, {"name": "Marketing", "value": 30}, {"name": "R&D", "value": 25}]' \
    '["png", "svg"]' \
    "Pie Chart"

# Scatter chart
test_chart_generation "scatter" \
    '[{"x": 1.5, "y": 2.5}, {"x": 3.2, "y": 5.1}, {"x": 4.8, "y": 3.9}]' \
    '["png", "svg"]' \
    "Scatter Chart"

# Area chart
test_chart_generation "area" \
    '[{"x": "2021", "y": 100}, {"x": "2022", "y": 150}, {"x": "2023", "y": 200}]' \
    '["png", "svg"]' \
    "Area Chart"

echo ""

# Test P1 Requirements - Advanced chart types
echo "3. P1 Requirements - Advanced Chart Types"

# Gantt chart
test_chart_generation "gantt" \
    '[{"task": "Design", "start": 0, "duration": 30}, {"task": "Development", "start": 20, "duration": 50}]' \
    '["png", "svg"]' \
    "Gantt Chart"

# Heatmap
test_chart_generation "heatmap" \
    '[{"x": "Mon", "y": "Morning", "value": 10}, {"x": "Mon", "y": "Evening", "value": 25}]' \
    '["png", "svg"]' \
    "Heatmap"

# Treemap
test_chart_generation "treemap" \
    '[{"name": "Root", "value": 100}, {"name": "Child1", "value": 60}, {"name": "Child2", "value": 40}]' \
    '["png", "svg"]' \
    "Treemap"

# Candlestick chart
test_chart_generation "candlestick" \
    '[{"date": "2024-01-01", "open": 100, "high": 110, "low": 95, "close": 105}, {"date": "2024-01-02", "open": 105, "high": 115, "low": 100, "close": 112}]' \
    '["png", "svg"]' \
    "Candlestick Chart"

echo ""

# Test PDF Export (P1 Requirement)
echo "4. P1 Feature - PDF Export"
test_chart_generation "bar" \
    '[{"x": "Product A", "y": 500}, {"x": "Product B", "y": 750}]' \
    '["pdf"]' \
    "PDF Export"

echo ""

# Test multiple export formats
echo "5. Multiple Export Formats"
test_chart_generation "line" \
    '[{"x": "Day 1", "y": 10}, {"x": "Day 2", "y": 20}]' \
    '["png", "svg", "pdf"]' \
    "Multi-format Export"

echo ""

# Test edge cases
echo "6. Edge Cases"

# Empty data
echo -n "Testing empty data handling... "
response=$(curl -sf -X POST "$API_URL/api/v1/charts/generate" \
    -H "Content-Type: application/json" \
    -d '{
        "chart_type": "bar",
        "data": [],
        "export_formats": ["png"]
    }' 2>/dev/null || echo '{"success": false}')

if echo "$response" | jq -e '.success == false' >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Properly handles empty data${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}❌ Failed to handle empty data${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Invalid chart type
echo -n "Testing invalid chart type... "
response=$(curl -sf -X POST "$API_URL/api/v1/charts/generate" \
    -H "Content-Type: application/json" \
    -d '{
        "chart_type": "invalid_type",
        "data": [{"x": 1, "y": 2}],
        "export_formats": ["png"]
    }' 2>/dev/null || echo '{"success": false}')

if echo "$response" | jq -e '.success == false' >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Properly rejects invalid chart type${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}❌ Failed to reject invalid chart type${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo ""

# Performance test
echo "7. Performance Test"
echo -n "Testing large dataset (1000 points)... "
large_data='['
for i in {1..1000}; do
    if [ $i -ne 1 ]; then large_data+=','; fi
    large_data+="{\"x\": \"Item$i\", \"y\": $((RANDOM % 1000))}"
done
large_data+=']'

start_time=$(date +%s%N)
response=$(curl -sf -X POST "$API_URL/api/v1/charts/generate" \
    -H "Content-Type: application/json" \
    -d "{
        \"chart_type\": \"line\",
        \"data\": $large_data,
        \"export_formats\": [\"svg\"]
    }" 2>/dev/null || echo '{"success": false}')
end_time=$(date +%s%N)

duration=$((($end_time - $start_time) / 1000000))

if echo "$response" | jq -e '.success == true' >/dev/null 2>&1 && [ $duration -lt 2000 ]; then
    echo -e "${GREEN}✅ Generated in ${duration}ms (<2000ms target)${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}❌ Performance issue: ${duration}ms (>2000ms target)${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo ""
echo "═══════════════════════════════════════"
echo "Test Results Summary"
echo "═══════════════════════════════════════"
echo -e "${GREEN}Passed:${NC} $TESTS_PASSED"
echo -e "${RED}Failed:${NC} $TESTS_FAILED"
echo ""

# Cleanup
rm -rf "$TEST_OUTPUT"

# Exit with appropriate code
if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi