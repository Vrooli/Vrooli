#!/bin/bash

# Smart Shopping Assistant Test Suite
# Version: 1.0.0

set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "Testing: $test_name... "
    
    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
}

echo "🧪 Running Smart Shopping Assistant Tests"
echo "========================================="

# Structure tests
echo -e "\n📁 Structure Tests:"
run_test "PRD exists" "test -f ../PRD.md"
run_test "README exists" "test -f ../README.md"
run_test "Service config exists" "test -f ../.vrooli/service.json"
run_test "API main exists" "test -f ../api/main.go"
run_test "CLI script exists" "test -f ../cli/smart-shopping-assistant"
run_test "UI package.json exists" "test -f ../ui/package.json"

# API tests (if running)
echo -e "\n🔌 API Tests:"
if curl -sf http://localhost:${API_PORT:-3300}/health > /dev/null 2>&1; then
    run_test "API health check" "curl -sf http://localhost:${API_PORT:-3300}/health"
    run_test "Research endpoint" "curl -sf -X POST http://localhost:${API_PORT:-3300}/api/v1/shopping/research -H 'Content-Type: application/json' -d '{\"query\":\"test\"}'"
else
    echo -e "${YELLOW}API not running - skipping API tests${NC}"
fi

# CLI tests
echo -e "\n💻 CLI Tests:"
run_test "CLI is executable" "test -x ../cli/smart-shopping-assistant"
run_test "CLI help works" "../cli/smart-shopping-assistant help | grep -q 'Smart Shopping Assistant'"
run_test "CLI version works" "../cli/smart-shopping-assistant version | grep -q 'v1.0.0'"

# Database tests (if PostgreSQL is running)
echo -e "\n🗄️ Database Tests:"
if psql -h localhost -p ${RESOURCE_PORTS_postgres:-5432} -U postgres -c "SELECT 1" > /dev/null 2>&1; then
    run_test "Schema file exists" "test -f ../initialization/storage/postgres/schema.sql"
    run_test "Seed file exists" "test -f ../initialization/storage/postgres/seed.sql"
else
    echo -e "${YELLOW}PostgreSQL not running - skipping database tests${NC}"
fi

# Summary
echo -e "\n========================================="
echo "Test Results:"
echo -e "  Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "  Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}✗ Some tests failed${NC}"
    exit 1
fi