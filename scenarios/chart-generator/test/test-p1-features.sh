#!/usr/bin/env bash

set -euo pipefail

# Test script for P1 features: composite charts and data transformations

API_PORT="${API_PORT:-$(vrooli scenario port chart-generator | grep API_PORT | cut -d= -f2 || echo "20300")}"
API_URL="${CHART_API_URL:-http://localhost:${API_PORT}}"
TEST_OUTPUT="/tmp/chart-generator-p1-test-$$"
mkdir -p "$TEST_OUTPUT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "📊 Testing Chart Generator P1 Features..."
echo "API URL: $API_URL"
echo "Test output: $TEST_OUTPUT"
echo ""

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run a test
run_test() {
    local test_name="$1"
    local endpoint="$2"
    local method="$3"
    local data="$4"
    local expected_field="$5"
    
    echo -n "Testing $test_name... "
    
    local response
    if [ "$method" = "POST" ]; then
        response=$(curl -sf -X POST "$API_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data" 2>/dev/null || echo '{"success": false}')
    else
        response=$(curl -sf "$API_URL$endpoint" 2>/dev/null || echo '{"success": false}')
    fi
    
    if [ -n "$expected_field" ] && echo "$response" | jq -e "$expected_field" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ PASSED${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo "$response" > "$TEST_OUTPUT/${test_name// /_}_response.json"
    elif [ -z "$expected_field" ] && [ -n "$response" ]; then
        echo -e "${GREEN}✅ PASSED${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        echo "$response" > "$TEST_OUTPUT/${test_name// /_}_response.json"
    else
        echo -e "${RED}❌ FAILED${NC}"
        echo "Response: $response"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

echo "=== Testing Composite Charts (P1 Feature) ==="
echo ""

# Test 1: Composite chart with grid layout
run_test "Composite chart - grid layout" \
    "/api/v1/charts/composite" \
    "POST" \
    '{
        "chart_type": "composite",
        "config": {
            "composition": {
                "layout": "grid",
                "charts": [
                    {"chart_type": "bar", "data": [{"x": "A", "y": 10}, {"x": "B", "y": 20}]},
                    {"chart_type": "line", "data": [{"x": "Q1", "y": 30}, {"x": "Q2", "y": 40}]},
                    {"chart_type": "pie", "data": [{"name": "Part1", "value": 60}, {"name": "Part2", "value": 40}]}
                ]
            }
        }
    }' \
    '.success == true and .files.svg'

# Test 2: Composite chart with horizontal layout
run_test "Composite chart - horizontal layout" \
    "/api/v1/charts/composite" \
    "POST" \
    '{
        "chart_type": "composite",
        "config": {
            "composition": {
                "layout": "horizontal",
                "charts": [
                    {"chart_type": "scatter", "data": [{"x": 1, "y": 2}, {"x": 3, "y": 4}]},
                    {"chart_type": "area", "data": [{"x": "Jan", "y": 100}, {"x": "Feb", "y": 150}]}
                ]
            }
        }
    }' \
    '.success == true and .chart_id'

# Test 3: Composite chart with transformation
run_test "Composite chart - with data transformation" \
    "/api/v1/charts/composite" \
    "POST" \
    '{
        "chart_type": "composite",
        "config": {
            "composition": {
                "layout": "vertical",
                "charts": [
                    {
                        "chart_type": "bar",
                        "data": [{"x": "A", "y": 10}, {"x": "B", "y": 5}, {"x": "C", "y": 20}],
                        "transform": {"sort": {"field": "y", "direction": "desc"}}
                    }
                ]
            }
        }
    }' \
    '.success == true'

echo ""
echo "=== Testing Data Transformations (P1 Feature) ==="
echo ""

# Test 4: Sort transformation
run_test "Data transformation - sort ascending" \
    "/api/v1/data/transform" \
    "POST" \
    '{
        "data": [{"x": "C", "y": 30}, {"x": "A", "y": 10}, {"x": "B", "y": 20}],
        "transform": {"sort": {"field": "y", "direction": "asc"}}
    }' \
    '.success == true and .data[0].y == 10'

# Test 5: Sort transformation descending
run_test "Data transformation - sort descending" \
    "/api/v1/data/transform" \
    "POST" \
    '{
        "data": [{"x": "A", "y": 10}, {"x": "B", "y": 20}, {"x": "C", "y": 30}],
        "transform": {"sort": {"field": "y", "direction": "desc"}}
    }' \
    '.success == true and .data[0].y == 30'

# Test 6: Filter transformation
run_test "Data transformation - filter greater than" \
    "/api/v1/data/transform" \
    "POST" \
    '{
        "data": [{"x": "A", "y": 10}, {"x": "B", "y": 20}, {"x": "C", "y": 30}],
        "transform": {"filter": {"field": "y", "operator": "gt", "value": 15}}
    }' \
    '.success == true and .count == 2'

# Test 7: Aggregation
run_test "Data aggregation - sum" \
    "/api/v1/data/aggregate" \
    "POST" \
    '{
        "data": [{"category": "A", "value": 10}, {"category": "A", "value": 20}, {"category": "B", "value": 15}],
        "method": "sum",
        "field": "value",
        "group_by": "category"
    }' \
    '.success == true and .result'

# Test 8: Multiple transformations
run_test "Data transformation - filter and sort" \
    "/api/v1/data/transform" \
    "POST" \
    '{
        "data": [
            {"name": "Item1", "value": 5},
            {"name": "Item2", "value": 15},
            {"name": "Item3", "value": 25},
            {"name": "Item4", "value": 10}
        ],
        "transform": {
            "filter": {"field": "value", "operator": "gte", "value": 10},
            "sort": {"field": "value", "direction": "asc"}
        }
    }' \
    '.success == true and .data[0].value == 10'

echo ""
echo "=== Testing Style Builder (P1 Feature) ==="
echo ""

# Test 9: Style preview
run_test "Style builder - preview" \
    "/api/v1/styles/builder/preview" \
    "POST" \
    '{
        "chart_type": "bar",
        "style": {
            "name": "Custom Test Style",
            "color_palette": ["#FF5733", "#33FF57", "#3357FF"],
            "font_family": "Arial",
            "background_color": "#F0F0F0"
        }
    }' \
    '.success == true and .preview_url'

# Test 10: Get color palettes
run_test "Style builder - get palettes" \
    "/api/v1/styles/builder/palettes" \
    "GET" \
    "" \
    '.palettes'

echo ""
echo "=== Testing Advanced Chart Types (P1 Feature) ==="
echo ""

# Test 11: Gantt chart
run_test "Advanced chart - Gantt" \
    "/api/v1/charts/generate" \
    "POST" \
    '{
        "chart_type": "gantt",
        "data": [
            {"task": "Task 1", "start": "2024-01-01", "end": "2024-01-15"},
            {"task": "Task 2", "start": "2024-01-10", "end": "2024-01-25"}
        ],
        "export_formats": ["svg"]
    }' \
    '.success == true'

# Test 12: Heatmap
run_test "Advanced chart - Heatmap" \
    "/api/v1/charts/generate" \
    "POST" \
    '{
        "chart_type": "heatmap",
        "data": [
            {"x": "Mon", "y": "Morning", "value": 10},
            {"x": "Mon", "y": "Afternoon", "value": 20},
            {"x": "Tue", "y": "Morning", "value": 15}
        ],
        "export_formats": ["svg"]
    }' \
    '.success == true'

# Test 13: Treemap
run_test "Advanced chart - Treemap" \
    "/api/v1/charts/generate" \
    "POST" \
    '{
        "chart_type": "treemap",
        "data": [
            {"name": "Category A", "value": 100, "parent": "root"},
            {"name": "Sub A1", "value": 60, "parent": "Category A"},
            {"name": "Sub A2", "value": 40, "parent": "Category A"}
        ],
        "export_formats": ["svg"]
    }' \
    '.success == true'

# Test 14: Candlestick chart
run_test "Advanced chart - Candlestick" \
    "/api/v1/charts/generate" \
    "POST" \
    '{
        "chart_type": "candlestick",
        "data": [
            {"date": "2024-01-01", "open": 100, "high": 110, "low": 95, "close": 105},
            {"date": "2024-01-02", "open": 105, "high": 115, "low": 100, "close": 110}
        ],
        "export_formats": ["svg"]
    }' \
    '.success == true'

echo ""
echo "=== Testing Template Library (P1 Feature) ==="
echo ""

# Test 15: Get templates
run_test "Templates - list all" \
    "/api/v1/templates" \
    "GET" \
    "" \
    '.templates'

echo ""
echo "======================================="
echo "Test Results:"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
echo "======================================="

# Clean up test output directory if all tests passed
if [ "$TESTS_FAILED" -eq 0 ]; then
    rm -rf "$TEST_OUTPUT"
    echo -e "${GREEN}All P1 feature tests passed!${NC}"
    exit 0
else
    echo -e "${YELLOW}Test outputs saved in: $TEST_OUTPUT${NC}"
    exit 1
fi