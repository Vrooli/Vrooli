#!/bin/bash

# Integration Test Phase
# Tests Email Triage integration with resources

set -euo pipefail

echo "🔗 Testing Email Triage integrations..."

# Configuration
API_PORT="${API_PORT:-19528}"
API_URL="http://localhost:${API_PORT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

FAILURES=0

# Test database connectivity
echo "Testing PostgreSQL integration:"
echo -n "  Checking database connection... "
if curl -s "${API_URL}/health/database" | jq -e '.status == "connected"' >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    ((FAILURES++))
fi

# Test Qdrant connectivity
echo -e "\nTesting Qdrant integration:"
echo -n "  Checking Qdrant connection... "
if curl -s "${API_URL}/health/qdrant" | jq -e '.status == "connected"' >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    ((FAILURES++))
fi

# Test scenario-authenticator integration
echo -e "\nTesting authentication service:"
echo -n "  Checking auth service availability... "
AUTH_URL="http://localhost:8080/health"
if curl -s -o /dev/null -w "%{http_code}" "$AUTH_URL" 2>/dev/null | grep -q "200\|404"; then
    echo -e "${GREEN}✓ Auth service reachable${NC}"
else
    echo -e "${YELLOW}⚠ Auth service not running (expected for isolated tests)${NC}"
fi

# Test Ollama integration (optional)
echo -e "\nTesting Ollama integration (optional):"
echo -n "  Checking Ollama availability... "
OLLAMA_URL="http://localhost:11434/api/tags"
if curl -s -o /dev/null -w "%{http_code}" "$OLLAMA_URL" 2>/dev/null | grep -q "200"; then
    echo -e "${GREEN}✓ Ollama available${NC}"
else
    echo -e "${YELLOW}⚠ Ollama not available (optional resource)${NC}"
fi

# Test Redis integration (optional)
echo -e "\nTesting Redis integration (optional):"
echo -n "  Checking Redis availability... "
if command -v redis-cli >/dev/null 2>&1 && redis-cli ping 2>/dev/null | grep -q "PONG"; then
    echo -e "${GREEN}✓ Redis available${NC}"
else
    echo -e "${YELLOW}⚠ Redis not available (optional resource)${NC}"
fi

# Test mail-in-a-box integration
echo -e "\nTesting mail server integration:"
echo -n "  Checking mail server (mail-in-a-box)... "
# This would need actual mail-in-a-box instance
echo -e "${YELLOW}⚠ Mail server test skipped (requires mail-in-a-box setup)${NC}"

# Test CLI integration
echo -e "\nTesting CLI integration:"
commands=(
    "email-triage status"
    "email-triage --version"
    "email-triage help"
)

for cmd in "${commands[@]}"; do
    echo -n "  Testing: $cmd... "
    if $cmd >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
        ((FAILURES++))
    fi
done

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo -e "\n${GREEN}✅ Integration tests passed${NC}"
    exit 0
else
    echo -e "\n${RED}❌ $FAILURES integration test(s) failed${NC}"
    exit 1
fi