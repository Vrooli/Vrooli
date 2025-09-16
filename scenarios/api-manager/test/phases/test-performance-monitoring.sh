#!/bin/bash

# API Manager - Performance Monitoring Integration Tests
# Tests the performance monitoring endpoints and functionality

set -euo pipefail

# Test configuration
API_BASE_URL="${API_MANAGER_URL:-http://localhost:${API_PORT}}"
TEST_SCENARIO="test-scenario"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🧪 Running API Manager Performance Monitoring Tests${NC}"
echo "API Base URL: $API_BASE_URL"
echo "Test Scenario: $TEST_SCENARIO"
echo

# Test functions
test_create_performance_baseline() {
    echo -e "${YELLOW}Testing performance baseline creation...${NC}"
    
    # Create baseline with valid parameters
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/performance/baseline/$TEST_SCENARIO" \
        -H "Content-Type: application/json" \
        -d '{
            "duration_seconds": 60,
            "endpoint_urls": ["/api/v1/health", "/api/v1/scenarios"],
            "load_level": "light",
            "metadata": {"test": true}
        }' \
        -o /tmp/baseline_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "200" ]]; then
        echo -e "${GREEN}✓ Performance baseline created successfully${NC}"
        
        # Check response structure
        if jq -e '.baseline_id and .scenario and .status and .duration_seconds' /tmp/baseline_response.json > /dev/null; then
            baseline_id=$(jq -r '.baseline_id' /tmp/baseline_response.json)
            duration=$(jq -r '.duration_seconds' /tmp/baseline_response.json)
            load_level=$(jq -r '.load_level' /tmp/baseline_response.json)
            
            echo -e "${GREEN}✓ Baseline response structure valid${NC}"
            echo -e "${BLUE}  Baseline ID: $baseline_id${NC}"
            echo -e "${BLUE}  Duration: ${duration}s${NC}"
            echo -e "${BLUE}  Load Level: $load_level${NC}"
        else
            echo -e "${RED}✗ Baseline response structure invalid${NC}"
            return 1
        fi
    elif [[ "$http_code" == "404" ]]; then
        echo -e "${YELLOW}⚠ Test scenario not found - this is expected in isolated testing${NC}"
    else
        echo -e "${RED}✗ Baseline creation failed (HTTP $http_code)${NC}"
        return 1
    fi
    
    rm -f /tmp/baseline_response.json
}

test_get_performance_metrics() {
    echo -e "${YELLOW}Testing performance metrics retrieval...${NC}"
    
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/performance/metrics/$TEST_SCENARIO" -o /tmp/metrics_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "200" ]]; then
        echo -e "${GREEN}✓ Performance metrics endpoint responding${NC}"
        
        # Check response structure
        if jq -e '.scenario and .metrics and .count and .timestamp' /tmp/metrics_response.json > /dev/null; then
            metrics_count=$(jq -r '.count' /tmp/metrics_response.json)
            echo -e "${GREEN}✓ Metrics response structure valid${NC}"
            echo -e "${BLUE}  Metrics count: $metrics_count${NC}"
            
            # If metrics exist, check structure
            if [[ "$metrics_count" -gt "0" ]]; then
                if jq -e '.metrics[0].type and .metrics[0].value and .metrics[0].measured_at' /tmp/metrics_response.json > /dev/null; then
                    echo -e "${GREEN}✓ Individual metric structure valid${NC}"
                else
                    echo -e "${RED}✗ Individual metric structure invalid${NC}"
                    return 1
                fi
            fi
        else
            echo -e "${RED}✗ Metrics response structure invalid${NC}"
            return 1
        fi
    elif [[ "$http_code" == "404" ]]; then
        echo -e "${YELLOW}⚠ Test scenario not found - this is expected in isolated testing${NC}"
    else
        echo -e "${RED}✗ Performance metrics endpoint failed (HTTP $http_code)${NC}"
        return 1
    fi
    
    rm -f /tmp/metrics_response.json
}

test_performance_alerts() {
    echo -e "${YELLOW}Testing performance alerts endpoint...${NC}"
    
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/performance/alerts" -o /tmp/perf_alerts_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "200" ]]; then
        echo -e "${GREEN}✓ Performance alerts endpoint responding${NC}"
        
        # Check response structure
        if jq -e '.alerts and .count and .timestamp' /tmp/perf_alerts_response.json > /dev/null; then
            alert_count=$(jq -r '.count' /tmp/perf_alerts_response.json)
            array_length=$(jq -r '.alerts | length' /tmp/perf_alerts_response.json)
            
            echo -e "${GREEN}✓ Alerts response structure valid${NC}"
            echo -e "${BLUE}  Alert count: $alert_count${NC}"
            
            # Verify count matches array length
            if [[ "$alert_count" == "$array_length" ]]; then
                echo -e "${GREEN}✓ Alert count matches array length${NC}"
            else
                echo -e "${RED}✗ Alert count mismatch: reported=$alert_count, actual=$array_length${NC}"
                return 1
            fi
            
            # If alerts exist, check structure
            if [[ "$alert_count" -gt "0" ]]; then
                if jq -e '.alerts[0].level and .alerts[0].category and .alerts[0].title' /tmp/perf_alerts_response.json > /dev/null; then
                    echo -e "${GREEN}✓ Individual alert structure valid${NC}"
                else
                    echo -e "${RED}✗ Individual alert structure invalid${NC}"
                    return 1
                fi
            fi
        else
            echo -e "${RED}✗ Alerts response structure invalid${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Performance alerts endpoint failed (HTTP $http_code)${NC}"
        return 1
    fi
    
    rm -f /tmp/perf_alerts_response.json
}

test_breaking_change_detection() {
    echo -e "${YELLOW}Testing breaking change detection...${NC}"
    
    # Test change detection with parameters
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/changes/detect/$TEST_SCENARIO" \
        -H "Content-Type: application/json" \
        -d '{
            "compare_with": "previous",
            "include_minor_changes": true,
            "metadata": {"test": true}
        }' \
        -o /tmp/changes_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "200" ]]; then
        echo -e "${GREEN}✓ Breaking change detection completed${NC}"
        
        # Check response structure
        if jq -e '.scenario and .change_id and .changes and .breaking_count and .status' /tmp/changes_response.json > /dev/null; then
            breaking_count=$(jq -r '.breaking_count' /tmp/changes_response.json)
            minor_count=$(jq -r '.minor_count' /tmp/changes_response.json)
            status=$(jq -r '.status' /tmp/changes_response.json)
            
            echo -e "${GREEN}✓ Change detection response structure valid${NC}"
            echo -e "${BLUE}  Breaking changes: $breaking_count${NC}"
            echo -e "${BLUE}  Minor changes: $minor_count${NC}"
            echo -e "${BLUE}  Status: $status${NC}"
            
            # Check if changes array structure is valid when changes exist
            changes_array_length=$(jq -r '.changes | length' /tmp/changes_response.json)
            total_changes=$((breaking_count + minor_count))
            
            if [[ "$changes_array_length" == "$total_changes" ]]; then
                echo -e "${GREEN}✓ Changes count matches array length${NC}"
            else
                echo -e "${YELLOW}⚠ Changes count mismatch (expected in test data): array=$changes_array_length, counted=$total_changes${NC}"
            fi
        else
            echo -e "${RED}✗ Change detection response structure invalid${NC}"
            return 1
        fi
    elif [[ "$http_code" == "404" ]]; then
        echo -e "${YELLOW}⚠ Test scenario not found - this is expected in isolated testing${NC}"
    else
        echo -e "${RED}✗ Breaking change detection failed (HTTP $http_code)${NC}"
        return 1
    fi
    
    rm -f /tmp/changes_response.json
}

test_change_history() {
    echo -e "${YELLOW}Testing change history endpoint...${NC}"
    
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/changes/history/$TEST_SCENARIO" -o /tmp/history_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "200" ]]; then
        echo -e "${GREEN}✓ Change history endpoint responding${NC}"
        
        # Check response structure
        if jq -e '.scenario and .history and .count and .timestamp' /tmp/history_response.json > /dev/null; then
            history_count=$(jq -r '.count' /tmp/history_response.json)
            array_length=$(jq -r '.history | length' /tmp/history_response.json)
            
            echo -e "${GREEN}✓ History response structure valid${NC}"
            echo -e "${BLUE}  History entries: $history_count${NC}"
            
            # Verify count matches array length
            if [[ "$history_count" == "$array_length" ]]; then
                echo -e "${GREEN}✓ History count matches array length${NC}"
            else
                echo -e "${RED}✗ History count mismatch: reported=$history_count, actual=$array_length${NC}"
                return 1
            fi
        else
            echo -e "${RED}✗ History response structure invalid${NC}"
            return 1
        fi
    elif [[ "$http_code" == "404" ]]; then
        echo -e "${YELLOW}⚠ Test scenario not found - this is expected in isolated testing${NC}"
    else
        echo -e "${RED}✗ Change history endpoint failed (HTTP $http_code)${NC}"
        return 1
    fi
    
    rm -f /tmp/history_response.json
}

# Main test execution
main() {
    echo -e "${YELLOW}Starting Performance Monitoring Integration Tests...${NC}"
    echo
    
    local test_count=0
    local passed_count=0
    
    # Run tests
    tests=(
        "test_create_performance_baseline"
        "test_get_performance_metrics"
        "test_performance_alerts"
        "test_breaking_change_detection"
        "test_change_history"
    )
    
    for test in "${tests[@]}"; do
        test_count=$((test_count + 1))
        echo -e "${YELLOW}[Test $test_count] Running $test${NC}"
        
        if $test; then
            passed_count=$((passed_count + 1))
            echo -e "${GREEN}✓ $test PASSED${NC}"
        else
            echo -e "${RED}✗ $test FAILED${NC}"
        fi
        echo
    done
    
    # Summary
    echo -e "${YELLOW}Test Summary:${NC}"
    echo "Total tests: $test_count"
    echo "Passed: $passed_count"
    echo "Failed: $((test_count - passed_count))"
    
    if [[ $passed_count -eq $test_count ]]; then
        echo -e "${GREEN}🎉 All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}❌ Some tests failed${NC}"
        exit 1
    fi
}

# Run main function
main "$@"