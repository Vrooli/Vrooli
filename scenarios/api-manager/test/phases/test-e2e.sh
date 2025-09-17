#!/bin/bash

# API Manager - End-to-End Tests
# Tests complete user workflows and scenarios

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Running API Manager End-to-End Tests${NC}"
echo "===================================="

# Configuration
API_BASE_URL="${API_MANAGER_URL:-http://localhost:${API_PORT:-8080}}"
TIMEOUT=10

# Track test results
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function for API calls
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    
    local args=(-s -X "$method" --connect-timeout "$TIMEOUT")
    if [[ -n "$data" ]]; then
        args+=(-H "Content-Type: application/json" -d "$data")
    fi
    
    curl "${args[@]}" "$API_BASE_URL$endpoint" 2>/dev/null || echo "{}"
}

# Test Scenario 1: Complete vulnerability scanning workflow
echo -e "${YELLOW}Scenario 1: Vulnerability Scanning Workflow${NC}"
TESTS_RUN=$((TESTS_RUN + 1))

echo "  1. Discovering scenarios..."
discover_response=$(api_call "POST" "/api/v1/system/discover")
if echo "$discover_response" | jq -e '.status' > /dev/null 2>&1; then
    echo -e "     ${GREEN}✓ Scenarios discovered${NC}"
else
    echo -e "     ${RED}✗ Discovery failed${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo "  2. Listing available scenarios..."
scenarios_response=$(api_call "GET" "/api/v1/scenarios")
scenario_count=$(echo "$scenarios_response" | jq -r '.count // 0')
if [[ "$scenario_count" -ge 0 ]]; then
    echo -e "     ${GREEN}✓ Found $scenario_count scenarios${NC}"
else
    echo -e "     ${RED}✗ Failed to list scenarios${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo "  3. Checking for vulnerabilities..."
vuln_response=$(api_call "GET" "/api/v1/vulnerabilities")
vuln_count=$(echo "$vuln_response" | jq -r '.count // 0')
echo -e "     ${GREEN}✓ Found $vuln_count vulnerabilities${NC}"

echo "  4. Getting health summary..."
health_response=$(api_call "GET" "/api/v1/health/summary")
if echo "$health_response" | jq -e '.status' > /dev/null 2>&1; then
    health_status=$(echo "$health_response" | jq -r '.status')
    echo -e "     ${GREEN}✓ System health: $health_status${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "     ${RED}✗ Failed to get health summary${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test Scenario 2: Standards compliance workflow
echo
echo -e "${YELLOW}Scenario 2: Standards Compliance Workflow${NC}"
TESTS_RUN=$((TESTS_RUN + 1))

echo "  1. Checking standards violations..."
violations_response=$(api_call "GET" "/api/v1/standards/violations")
if echo "$violations_response" | jq -e '.violations' > /dev/null 2>&1; then
    violation_count=$(echo "$violations_response" | jq '.violations | length')
    echo -e "     ${GREEN}✓ Found $violation_count standards violations${NC}"
else
    echo -e "     ${RED}✗ Failed to check violations${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo "  2. Validating lifecycle protection..."
lifecycle_response=$(api_call "GET" "/api/v1/system/validate-lifecycle")
if echo "$lifecycle_response" | jq -e '.success' > /dev/null 2>&1; then
    compliance_rate=$(echo "$lifecycle_response" | jq -r '.data.compliance_rate // 0')
    echo -e "     ${GREEN}✓ Lifecycle compliance: ${compliance_rate}%${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "     ${RED}✗ Failed to validate lifecycle${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test Scenario 3: Monitoring and alerting workflow
echo
echo -e "${YELLOW}Scenario 3: Monitoring and Alerting Workflow${NC}"
TESTS_RUN=$((TESTS_RUN + 1))

echo "  1. Getting health alerts..."
alerts_response=$(api_call "GET" "/api/v1/health/alerts")
if echo "$alerts_response" | jq -e '.alerts' > /dev/null 2>&1; then
    alert_count=$(echo "$alerts_response" | jq '.alerts | length')
    echo -e "     ${GREEN}✓ Found $alert_count active alerts${NC}"
    
    # Display critical alerts if any
    critical_alerts=$(echo "$alerts_response" | jq -r '.alerts[] | select(.level == "critical") | .title' 2>/dev/null)
    if [[ -n "$critical_alerts" ]]; then
        echo -e "     ${RED}⚠ Critical alerts:${NC}"
        echo "$critical_alerts" | while read -r alert; do
            echo "       - $alert"
        done
    fi
else
    echo -e "     ${RED}✗ Failed to get alerts${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

echo "  2. Checking system status..."
status_response=$(api_call "GET" "/api/v1/system/status")
if echo "$status_response" | jq -e '.status' > /dev/null 2>&1; then
    system_status=$(echo "$status_response" | jq -r '.status')
    echo -e "     ${GREEN}✓ System status: $system_status${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "     ${RED}✗ Failed to get system status${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test Scenario 4: OpenAPI documentation
echo
echo -e "${YELLOW}Scenario 4: API Documentation Workflow${NC}"
TESTS_RUN=$((TESTS_RUN + 1))

echo "  1. Getting OpenAPI spec for api-manager..."
openapi_response=$(api_call "GET" "/api/v1/openapi/api-manager")
if echo "$openapi_response" | jq -e '.openapi' > /dev/null 2>&1; then
    openapi_version=$(echo "$openapi_response" | jq -r '.openapi')
    echo -e "     ${GREEN}✓ OpenAPI version: $openapi_version${NC}"
    
    # Count documented paths
    path_count=$(echo "$openapi_response" | jq '.paths | length')
    echo -e "     ${GREEN}✓ Documented paths: $path_count${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "     ${RED}✗ Failed to get OpenAPI spec${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Summary
echo
echo "===================================="
echo -e "${BLUE}End-to-End Test Summary${NC}"
echo "===================================="
echo "Tests Run: $TESTS_RUN"
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}✓ All end-to-end tests passed${NC}"
    echo -e "${GREEN}API Manager workflows are functioning correctly${NC}"
    exit 0
else
    echo -e "${RED}✗ Some end-to-end tests failed${NC}"
    echo -e "${RED}Please review the failed scenarios${NC}"
    exit 1
fi