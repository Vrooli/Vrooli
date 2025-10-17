#!/bin/bash

# Validation script for AI Chatbot Manager refactoring
# This script verifies all refactoring requirements are met

set -e

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_DIR="${SCENARIO_DIR}/api"

echo "🔍 Validating AI Chatbot Manager Refactoring..."
echo "================================================"

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "Testing: $test_name... "
    
    if eval "$test_command" 2>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
}

echo ""
echo "1. Code Structure Validation"
echo "----------------------------"

run_test "main.go is modular (<100 lines)" \
    "[ $(wc -l < ${API_DIR}/main.go) -lt 100 ]"

run_test "config.go exists" \
    "[ -f ${API_DIR}/config.go ]"

run_test "models.go exists" \
    "[ -f ${API_DIR}/models.go ]"

run_test "database.go exists" \
    "[ -f ${API_DIR}/database.go ]"

run_test "handlers.go exists" \
    "[ -f ${API_DIR}/handlers.go ]"

run_test "websocket.go exists" \
    "[ -f ${API_DIR}/websocket.go ]"

run_test "middleware.go exists" \
    "[ -f ${API_DIR}/middleware.go ]"

run_test "server.go exists" \
    "[ -f ${API_DIR}/server.go ]"

run_test "widget.go exists" \
    "[ -f ${API_DIR}/widget.go ]"

echo ""
echo "2. No Hardcoded Values"
echo "----------------------"

run_test "No hardcoded port 8090" \
    "! grep -q '8090' ${API_DIR}/*.go"

run_test "No hardcoded localhost:11434" \
    "! grep -q 'localhost:11434' ${API_DIR}/*.go"

run_test "No defaultPort variable" \
    "! grep -q 'defaultPort' ${API_DIR}/*.go"

run_test "No defaultOllamaURL variable" \
    "! grep -q 'defaultOllamaURL' ${API_DIR}/*.go"

echo ""
echo "3. Lifecycle Compliance"
echo "-----------------------"

run_test "Checks VROOLI_LIFECYCLE_MANAGED" \
    "grep -q 'VROOLI_LIFECYCLE_MANAGED' ${API_DIR}/config.go"

run_test "Requires API_PORT with no default" \
    "grep -q 'API_PORT environment variable is required' ${API_DIR}/config.go"

run_test "Requires OLLAMA_URL with no default" \
    "grep -q 'OLLAMA_URL environment variable is required' ${API_DIR}/config.go"

echo ""
echo "4. Go Build Validation"
echo "----------------------"

cd "$API_DIR"
run_test "Go modules are valid" \
    "go mod tidy"

run_test "Code compiles successfully" \
    "go build -o /tmp/test-chatbot-build ."

# Clean up test build
rm -f /tmp/test-chatbot-build

echo ""
echo "5. Service Configuration"
echo "------------------------"

run_test "service.json exists" \
    "[ -f ${SCENARIO_DIR}/.vrooli/service.json ]"

run_test "service.json has lifecycle v2.0" \
    "grep -q '\"version\": \"2.0.0\"' ${SCENARIO_DIR}/.vrooli/service.json"

run_test "Port ranges configured correctly" \
    "grep -q '\"range\": \"15000-19999\"' ${SCENARIO_DIR}/.vrooli/service.json"

echo ""
echo "6. Database & WebSocket Features"
echo "--------------------------------"

run_test "Exponential backoff in database.go" \
    "grep -q 'exponential backoff' ${API_DIR}/database.go"

run_test "WebSocket connection manager exists" \
    "grep -q 'ConnectionManager' ${API_DIR}/websocket.go"

run_test "Rate limiting middleware exists" \
    "grep -q 'RateLimiter' ${API_DIR}/middleware.go"

echo ""
echo "7. Widget Configuration"
echo "-----------------------"

run_test "Widget has dynamic URL detection" \
    "grep -q 'window.CHATBOT_API_URL' ${API_DIR}/widget.go"

run_test "Widget supports meta tag config" \
    "grep -q 'chatbot-api-url' ${API_DIR}/widget.go"

echo ""
echo "8. SQL & Analytics"
echo "------------------"

run_test "Analytics query is fixed" \
    "! grep -q 'calculate_engagement_score(c.id)' ${API_DIR}/database.go"

run_test "Schema.sql has proper functions" \
    "grep -q 'calculate_engagement_score' ${SCENARIO_DIR}/initialization/storage/postgres/schema.sql"

echo ""
echo "================================================"
echo "                VALIDATION SUMMARY              "
echo "================================================"
echo -e "Tests Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests Failed: ${RED}${TESTS_FAILED}${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All validation tests passed!${NC}"
    echo "The AI Chatbot Manager has been successfully refactored."
    exit 0
else
    echo -e "${RED}❌ Some validation tests failed.${NC}"
    echo "Please review the failed tests and fix any issues."
    exit 1
fi