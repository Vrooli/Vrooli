#!/bin/bash

# Health Check Test Phase
# Tests that the Email Triage API is healthy and accessible

set -euo pipefail

echo "🩺 Testing Email Triage health endpoints..."

# Always detect the scenario name from our location first
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Extract scenario name from the path
if [[ "$SCRIPT_DIR" =~ /scenarios/([^/]+)/ ]]; then
    SCENARIO_NAME="${BASH_REMATCH[1]}"
else
    # Fallback: try to determine from directory structure
    SCENARIO_NAME=$(basename "$(dirname "$(dirname "$SCRIPT_DIR")")" 2>/dev/null || echo "")
fi

# Always try to detect the port for THIS scenario from vrooli CLI
# This overrides any inherited API_PORT environment variable
if [ -n "$SCENARIO_NAME" ] && command -v vrooli >/dev/null 2>&1; then
    DETECTED_PORT=$(vrooli scenario status "$SCENARIO_NAME" --json 2>/dev/null | jq -r '.scenario_data.allocated_ports.API_PORT // empty' 2>/dev/null)
    if [ -n "$DETECTED_PORT" ]; then
        API_PORT="$DETECTED_PORT"
    fi
fi

# Fall back to environment variable if detection failed
if [ -z "${API_PORT:-}" ]; then
    # Fall back to service.json or default
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

# Test CLI health command (try PATH first, then direct path)
echo -n "  Testing CLI health command... "
CLI_CMD="email-triage"
if ! command -v email-triage >/dev/null 2>&1; then
    CLI_CMD="$HOME/.vrooli/bin/email-triage"
fi

if $CLI_CMD status >/dev/null 2>&1; then
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