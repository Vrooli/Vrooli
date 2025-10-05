#!/bin/bash

# Health Check Test Phase
# Tests that the Email Triage API is healthy and accessible

set -euo pipefail

echo "🩺 Testing Email Triage health endpoints..."

# Get API port from environment, service.json, or use default
if [ -z "$API_PORT" ]; then
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
    SERVICE_JSON="$(dirname "$(dirname "$SCRIPT_DIR")")/.vrooli/service.json"
    if [ -f "$SERVICE_JSON" ] && command -v jq >/dev/null 2>&1; then
        API_PORT=$(jq -r '.endpoints.api // "http://localhost:19528"' "$SERVICE_JSON" | sed 's/.*://')
    else
        API_PORT=19528
    fi
fi
API_URL="http://localhost:${API_PORT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# Test helper function
test_endpoint() {
    local endpoint="$1"
    local expected_status="${2:-200}"
    local description="$3"
    
    echo -n "  Testing $description... "
    
    response=$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}${endpoint}" 2>/dev/null || echo "000")
    
    if [[ "$response" == "$expected_status" ]]; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗ (got $response, expected $expected_status)${NC}"
        return 1
    fi
}

# Track failures
FAILURES=0

# Test main health endpoint
if ! test_endpoint "/health" 200 "Main health check"; then
    ((FAILURES++))
fi

# Test health details with jq validation
echo -n "  Validating health response... "
if health_response=$(curl -s "${API_URL}/health" 2>/dev/null); then
    if echo "$health_response" | jq -e '.status == "healthy"' >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗ (invalid response format or unhealthy status)${NC}"
        ((FAILURES++))
    fi
else
    echo -e "${RED}✗ (failed to fetch)${NC}"
    ((FAILURES++))
fi

# Test database health - check JSON response
echo -n "  Testing database health... "
if db_response=$(curl -s "${API_URL}/health/database" 2>/dev/null); then
    if echo "$db_response" | jq -e '.connected == true' >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗ (database not connected)${NC}"
        ((FAILURES++))
    fi
else
    echo -e "${RED}✗ (endpoint failed)${NC}"
    ((FAILURES++))
fi

# Test Qdrant health - check JSON response
echo -n "  Testing Qdrant health... "
if qdrant_response=$(curl -s "${API_URL}/health/qdrant" 2>/dev/null); then
    if echo "$qdrant_response" | jq -e '.connected == true' >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗ (Qdrant not connected)${NC}"
        ((FAILURES++))
    fi
else
    echo -e "${RED}✗ (endpoint failed)${NC}"
    ((FAILURES++))
fi

# Test CLI health command
echo -n "  Testing CLI health command... "
if email-triage status >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    ((FAILURES++))
fi

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo -e "\n${GREEN}✅ All health checks passed${NC}"
    exit 0
else
    echo -e "\n${RED}❌ $FAILURES health check(s) failed${NC}"
    exit 1
fi