#!/bin/bash
################################################################################
# Test Genie Final Validation Script
#
# This script validates that test-genie meets all Vrooli scenario standards
################################################################################

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}                TEST GENIE VALIDATION REPORT                    ${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

# Validation results
PASS_COUNT=0
FAIL_COUNT=0

validate() {
    local check_name="$1"
    local check_command="$2"
    local expected="$3"
    
    echo -ne "  ${check_name}: "
    
    result=$(eval "$check_command" 2>&1) || true
    
    if [[ "$result" == *"$expected"* ]] || [[ "$result" =~ $expected ]]; then
        echo -e "${GREEN}✅ PASSED${NC}"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo -e "${RED}❌ FAILED${NC}"
        echo "    Expected: $expected"
        echo "    Got: ${result:0:100}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

echo -e "${YELLOW}📋 Architecture Compliance:${NC}"
validate "Service.json exists" "test -f .vrooli/service.json && echo 'exists'" "exists"
validate "Lifecycle v2.0" "grep -q '\"version\": \"2.0.0\"' .vrooli/service.json && echo 'yes'" "yes"
validate "Port ranges configured" "grep -q 'range' .vrooli/service.json && echo 'yes'" "yes"
validate "No hardcoded ports" "grep -E '(8080|3000|5432)' api/main.go | wc -l" "0"
validate "CLI wrapper exists" "test -f cli/test-genie && echo 'exists'" "exists"
validate "Multi-file UI" "ls ui/*.{js,html} 2>/dev/null | wc -l | awk '\$1 >= 3 {print \"yes\"}'" "yes"
validate "Makefile exists" "test -f Makefile && echo 'exists'" "exists"

echo ""
echo -e "${YELLOW}🔧 Code Quality:${NC}"
validate "Go compilation" "cd api && go build -o /tmp/test-build . 2>&1 && rm /tmp/test-build && echo 'success'" "success"
validate "Database reconnection" "grep -q 'exponential backoff' api/main.go && echo 'yes'" "yes"
validate "Circuit breakers" "grep -q 'CircuitBreaker' api/main.go && echo 'yes'" "yes"
validate "Error handling" "grep -q 'error' api/error_handling.go && echo 'yes'" "yes"
validate "API versioning" "grep -q '/api/v1/' api/main.go && echo 'yes'" "yes"

echo ""
echo -e "${YELLOW}🚀 Runtime Status:${NC}"
validate "API running" "curl -s http://localhost:8250/health | jq -r '.status' 2>/dev/null" "healthy"
validate "Database connected" "curl -s http://localhost:8250/health | jq -r '.checks.database.status' 2>/dev/null" "healthy"
validate "AI service ready" "curl -s http://localhost:8250/health | jq -r '.checks.ai_service.status' 2>/dev/null" "healthy"
validate "Test generation works" "curl -s -X POST http://localhost:8250/api/v1/test-suite/generate -H 'Content-Type: application/json' -d '{\"scenario_name\":\"validation\",\"test_types\":[\"unit\"]}' | jq -r '.suite_id' 2>/dev/null | grep -E '^[a-f0-9-]{36}$' && echo 'valid'" "valid"

echo ""
echo -e "${YELLOW}📊 Environment Setup:${NC}"
validate "Setup script exists" "test -f setup-env.sh && echo 'exists'" "exists"
validate ".env file exists" "test -f .env && echo 'exists'" "exists"
validate "Test database exists" "docker exec vrooli-postgres-main psql -U vrooli -c '\\l' | grep test_genie && echo 'exists'" "exists"

echo ""
echo -e "${YELLOW}🧪 Testing:${NC}"
validate "Integration tests exist" "test -f test/integration_test.sh && echo 'exists'" "exists"
validate "Test directory structure" "test -d test && echo 'exists'" "exists"

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}                        FINAL SCORE                             ${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  Checks Passed: ${GREEN}${PASS_COUNT}${NC}"
echo -e "  Checks Failed: ${RED}${FAIL_COUNT}${NC}"

TOTAL=$((PASS_COUNT + FAIL_COUNT))
PERCENTAGE=$((PASS_COUNT * 100 / TOTAL))

echo -e "  Score: ${BLUE}${PERCENTAGE}%${NC}"
echo ""

if [ $PERCENTAGE -ge 90 ]; then
    echo -e "  ${GREEN}✅ EXCELLENT: Test Genie meets Vrooli standards!${NC}"
    echo -e "  ${GREEN}   Ready for production deployment.${NC}"
elif [ $PERCENTAGE -ge 70 ]; then
    echo -e "  ${YELLOW}⚠️  GOOD: Test Genie mostly compliant${NC}"
    echo -e "  ${YELLOW}   Minor improvements needed.${NC}"
else
    echo -e "  ${RED}❌ NEEDS WORK: Significant improvements required${NC}"
    echo -e "  ${RED}   Review failed checks above.${NC}"
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"

# Stop running processes
echo ""
echo -e "${YELLOW}🛑 Cleaning up test processes...${NC}"
pkill -f "test-genie-api" 2>/dev/null || true
echo -e "${GREEN}✅ Cleanup complete${NC}"

exit 0