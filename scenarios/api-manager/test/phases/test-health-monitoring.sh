#!/bin/bash

# API Manager - Health Monitoring Integration Tests
# Tests the health monitoring endpoints and functionality

set -euo pipefail

# Test configuration
API_BASE_URL="${API_MANAGER_URL:-http://localhost:${API_PORT}}"
SCENARIO_NAME="test-scenario"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🧪 Running API Manager Health Monitoring Tests${NC}"
echo "API Base URL: $API_BASE_URL"
echo "Test Scenario: $SCENARIO_NAME"
echo

# Test functions
test_scenario_health() {
    echo -e "${YELLOW}Testing scenario health endpoint...${NC}"
    
    # Test with valid scenario
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/scenarios/$SCENARIO_NAME/health" -o /tmp/health_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "200" || "$http_code" == "404" ]]; then
        echo -e "${GREEN}✓ Scenario health endpoint responding${NC}"
        
        if [[ "$http_code" == "200" ]]; then
            # Check response structure
            if jq -e '.scenario and .status and .timestamp and .metrics' /tmp/health_response.json > /dev/null; then
                echo -e "${GREEN}✓ Health response structure valid${NC}"
            else
                echo -e "${RED}✗ Health response structure invalid${NC}"
                return 1
            fi
        fi
    else
        echo -e "${RED}✗ Scenario health endpoint failed (HTTP $http_code)${NC}"
        return 1
    fi
    
    rm -f /tmp/health_response.json
}

test_health_summary() {
    echo -e "${YELLOW}Testing health summary endpoint...${NC}"
    
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/health/summary" -o /tmp/summary_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "200" ]]; then
        echo -e "${GREEN}✓ Health summary endpoint responding${NC}"
        
        # Check response structure
        if jq -e '.status and .scenarios and .vulnerabilities and .system_health_score' /tmp/summary_response.json > /dev/null; then
            echo -e "${GREEN}✓ Summary response structure valid${NC}"
            
            # Check health score is numeric
            health_score=$(jq -r '.system_health_score' /tmp/summary_response.json)
            if [[ "$health_score" =~ ^[0-9]+\.?[0-9]*$ ]]; then
                echo -e "${GREEN}✓ Health score is numeric: $health_score${NC}"
            else
                echo -e "${RED}✗ Health score is not numeric: $health_score${NC}"
                return 1
            fi
        else
            echo -e "${RED}✗ Summary response structure invalid${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Health summary endpoint failed (HTTP $http_code)${NC}"
        return 1
    fi
    
    rm -f /tmp/summary_response.json
}

test_health_alerts() {
    echo -e "${YELLOW}Testing health alerts endpoint...${NC}"
    
    response=$(curl -s -w "%{http_code}" "$API_BASE_URL/api/v1/health/alerts" -o /tmp/alerts_response.json)
    http_code="${response: -3}"
    
    if [[ "$http_code" == "200" ]]; then
        echo -e "${GREEN}✓ Health alerts endpoint responding${NC}"
        
        # Check response structure
        if jq -e '.alerts and .count and .timestamp' /tmp/alerts_response.json > /dev/null; then
            echo -e "${GREEN}✓ Alerts response structure valid${NC}"
            
            # Check alert count matches array length
            alert_count=$(jq -r '.count' /tmp/alerts_response.json)
            array_length=$(jq -r '.alerts | length' /tmp/alerts_response.json)
            
            if [[ "$alert_count" == "$array_length" ]]; then
                echo -e "${GREEN}✓ Alert count matches array length: $alert_count${NC}"
            else
                echo -e "${RED}✗ Alert count mismatch: reported=$alert_count, actual=$array_length${NC}"
                return 1
            fi
        else
            echo -e "${RED}✗ Alerts response structure invalid${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Health alerts endpoint failed (HTTP $http_code)${NC}"
        return 1
    fi
    
    rm -f /tmp/alerts_response.json
}

# Main test execution
main() {
    echo -e "${YELLOW}Starting Health Monitoring Integration Tests...${NC}"
    echo
    
    local test_count=0
    local passed_count=0
    
    # Run tests
    tests=(
        "test_scenario_health"
        "test_health_summary" 
        "test_health_alerts"
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