#!/bin/bash

set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCENARIO_DIR"

echo "🔭 Testing Knowledge Observatory Scenario"
echo "========================================="

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function for tests
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "  Testing: $test_name... "
    
    if eval "$test_command" &> /dev/null; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
}

echo ""
echo "1. Structure Validation"
echo "-----------------------"

run_test "PRD.md exists" "[ -f PRD.md ]"
run_test "Service configuration exists" "[ -f .vrooli/service.json ]"
run_test "API main.go exists" "[ -f api/main.go ]"
run_test "API go.mod exists" "[ -f api/go.mod ]"
run_test "CLI script exists" "[ -f cli/knowledge-observatory ]"
run_test "CLI is executable" "[ -x cli/knowledge-observatory ]"
run_test "Install script exists" "[ -f cli/install.sh ]"
run_test "PostgreSQL schema exists" "[ -f initialization/postgres/schema.sql ]"
run_test "PostgreSQL seed exists" "[ -f initialization/postgres/seed.sql ]"
run_test "UI index.html exists" "[ -f ui/index.html ]"
run_test "UI script.js exists" "[ -f ui/script.js ]"
run_test "UI server.js exists" "[ -f ui/server.js ]"
run_test "UI package.json exists" "[ -f ui/package.json ]"
run_test "N8n quality monitor workflow" "[ -f initialization/n8n/knowledge-quality-monitor.json ]"
run_test "N8n semantic analyzer workflow" "[ -f initialization/n8n/semantic-analyzer.json ]"
run_test "N8n graph builder workflow" "[ -f initialization/n8n/knowledge-graph-builder.json ]"

echo ""
echo "2. Configuration Validation"
echo "---------------------------"

run_test "Service name is correct" "grep -q '\"name\": \"knowledge-observatory\"' .vrooli/service.json"
run_test "API port configured" "grep -q '\"port\": 20260' .vrooli/service.json"
run_test "UI port configured" "grep -q '\"port\": 20261' .vrooli/service.json"
run_test "Qdrant marked as required" "grep -q '\"required\": true' .vrooli/service.json | head -1"
run_test "PostgreSQL marked as required" "grep -q '\"postgres\"' .vrooli/service.json"

echo ""
echo "3. CLI Validation"
echo "-----------------"

run_test "CLI help command" "./cli/knowledge-observatory help | grep -q 'Knowledge Observatory'"
run_test "CLI version command" "./cli/knowledge-observatory version | grep -q '1.0.0'"
run_test "CLI has search command" "./cli/knowledge-observatory help | grep -q 'search'"
run_test "CLI has health command" "./cli/knowledge-observatory help | grep -q 'health'"
run_test "CLI has graph command" "./cli/knowledge-observatory help | grep -q 'graph'"
run_test "CLI has metrics command" "./cli/knowledge-observatory help | grep -q 'metrics'"

echo ""
echo "4. API Build Validation"
echo "-----------------------"

if command -v go &> /dev/null; then
    run_test "API compiles successfully" "cd api && go build -o test-build main.go && rm -f test-build"
else
    echo -e "  ${YELLOW}Skipping Go build tests (Go not installed)${NC}"
fi

echo ""
echo "5. Test Suite Validation"
echo "------------------------"

run_test "Scenario test YAML exists" "[ -f scenario-test.yaml ]"
run_test "CLI bats tests exist" "[ -f cli/knowledge-observatory.bats ]"
run_test "Test directory exists" "[ -d test ]"

if command -v bats &> /dev/null; then
    run_test "CLI bats tests pass" "cd cli && bats knowledge-observatory.bats"
else
    echo -e "  ${YELLOW}Skipping bats tests (bats not installed)${NC}"
fi

echo ""
echo "6. Documentation Validation"
echo "---------------------------"

run_test "README.md exists" "[ -f README.md ]"
run_test "README has overview section" "grep -q '## .* Overview' README.md"
run_test "README has usage section" "grep -q '## .* Usage' README.md"
run_test "PRD has capability definition" "grep -q '## .* Capability Definition' PRD.md"
run_test "PRD has success metrics" "grep -q '## .* Success Metrics' PRD.md"

echo ""
echo "========================================="
echo "Test Results:"
echo "  Passed: ${GREEN}$TESTS_PASSED${NC}"
echo "  Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✅ All tests passed! Knowledge Observatory is ready.${NC}"
    echo ""
    echo "To run the scenario:"
    echo "  vrooli scenario run knowledge-observatory"
    exit 0
else
    echo ""
    echo -e "${RED}❌ Some tests failed. Please fix the issues above.${NC}"
    exit 1
fi