#!/bin/bash

# Test Genie Functionality Test Suite
# Tests test generation, vault creation, and coverage analysis

set -e

# Get the API port dynamically
API_PORT=$(vrooli scenario port test-genie API_PORT 2>/dev/null)
if [[ -z "$API_PORT" ]]; then
    echo "❌ test-genie scenario is not running"
    echo "   Start it with: vrooli scenario run test-genie"
    exit 1
fi

API_URL="http://localhost:${API_PORT}"
TEST_DIR="/tmp/test-genie-test-$$"
CLI_PATH="test-genie"  # Should be installed globally

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}🧪 Testing Test Genie Functionality${NC}"

# Cleanup function
cleanup() {
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

# Setup test environment
echo "Setting up test scenario environment..."
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR/src" "$TEST_DIR/api" "$TEST_DIR/ui" "$TEST_DIR/cli"

# Create test scenario structure
echo "console.log('test scenario main');" > "$TEST_DIR/src/main.js"
echo "export function calculate() { return 42; }" > "$TEST_DIR/src/calculator.js"
echo "export class UserManager { getUser(id) { return {id}; } }" > "$TEST_DIR/src/user.js"
echo "package main" > "$TEST_DIR/api/main.go"
echo "func Add(a, b int) int { return a + b }" >> "$TEST_DIR/api/main.go"
echo "<html><body>Test App</body></html>" > "$TEST_DIR/ui/index.html"
echo "#!/bin/bash" > "$TEST_DIR/cli/app"
echo 'echo "Test CLI"' >> "$TEST_DIR/cli/app"

echo -e "${BLUE}📊 Test Data Setup Complete${NC}"

# Test 1: API Health Check
echo -e "\n${YELLOW}🔍 Test 1: API Health Check${NC}"
response=$(curl -s -w "%{http_code}" "$API_URL/health")
http_code="${response: -3}"
if [[ "$http_code" == "200" ]]; then
    echo -e "${GREEN}✅ API health check passed${NC}"
else
    echo -e "${RED}❌ API health check failed (HTTP $http_code)${NC}"
    exit 1
fi

# Test 2: Test Suite Generation
echo -e "\n${YELLOW}🤖 Test 2: AI Test Generation${NC}"
suite_response=$(curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
    -H "Content-Type: application/json" \
    -d "{
        \"scenario_name\": \"test-scenario\",
        \"test_types\": [\"unit\", \"integration\"],
        \"coverage_target\": 80,
        \"options\": {
            \"include_performance_tests\": true,
            \"include_security_tests\": false,
            \"execution_timeout\": 300
        }
    }")

if echo "$suite_response" | jq -e '.suite_id' >/dev/null 2>&1; then
    SUITE_ID=$(echo "$suite_response" | jq -r '.suite_id')
    echo -e "${GREEN}✅ Test suite generated successfully${NC}"
    echo -e "   Suite ID: $SUITE_ID"
    echo -e "   Generated tests: $(echo "$suite_response" | jq -r '.generated_tests')"
else
    echo -e "${RED}❌ Test suite generation failed${NC}"
    echo "   Response: $suite_response"
    exit 1
fi

# Test 3: Get Test Suite Details
echo -e "\n${YELLOW}📋 Test 3: Retrieve Test Suite${NC}"
suite_details=$(curl -s "$API_URL/api/v1/test-suite/$SUITE_ID")
if echo "$suite_details" | jq -e '.id' >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Test suite retrieval successful${NC}"
    echo -e "   Scenario: $(echo "$suite_details" | jq -r '.scenario_name')"
    echo -e "   Test cases: $(echo "$suite_details" | jq -r '.test_cases | length')"
else
    echo -e "${RED}❌ Test suite retrieval failed${NC}"
    echo "   Response: $suite_details"
    exit 1
fi

# Test 4: Coverage Analysis
echo -e "\n${YELLOW}📊 Test 4: Coverage Analysis${NC}"
coverage_response=$(curl -s -X POST "$API_URL/api/v1/test-analysis/coverage" \
    -H "Content-Type: application/json" \
    -d "{
        \"scenario_name\": \"test-scenario\",
        \"source_code_paths\": [\"$TEST_DIR/src\", \"$TEST_DIR/api\"],
        \"existing_test_paths\": [],
        \"analysis_depth\": \"comprehensive\"
    }")

if echo "$coverage_response" | jq -e '.overall_coverage' >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Coverage analysis successful${NC}"
    echo -e "   Overall coverage: $(echo "$coverage_response" | jq -r '.overall_coverage')%"
    echo -e "   Priority areas: $(echo "$coverage_response" | jq -r '.priority_areas | length') identified"
else
    echo -e "${RED}❌ Coverage analysis failed${NC}"
    echo "   Response: $coverage_response"
    exit 1
fi

# Test 5: Test Vault Creation
echo -e "\n${YELLOW}🏗️  Test 5: Test Vault Creation${NC}"
vault_response=$(curl -s -X POST "$API_URL/api/v1/test-vault/create" \
    -H "Content-Type: application/json" \
    -d "{
        \"scenario_name\": \"test-scenario\",
        \"vault_name\": \"test-vault-$$\",
        \"phases\": [\"setup\", \"develop\", \"test\", \"cleanup\"],
        \"phase_configurations\": {
            \"setup\": {
                \"timeout\": 300,
                \"environment\": \"testing\"
            },
            \"test\": {
                \"timeout\": 600,
                \"parallel\": true
            }
        },
        \"success_criteria\": {
            \"min_test_pass_rate\": 0.8,
            \"max_execution_time\": 1800
        }
    }")

if echo "$vault_response" | jq -e '.vault_id' >/dev/null 2>&1; then
    VAULT_ID=$(echo "$vault_response" | jq -r '.vault_id')
    echo -e "${GREEN}✅ Test vault created successfully${NC}"
    echo -e "   Vault ID: $VAULT_ID"
    echo -e "   Phases: $(echo "$vault_response" | jq -r '.phases | length')"
else
    echo -e "${RED}❌ Test vault creation failed${NC}"
    echo "   Response: $vault_response"
    exit 1
fi

# Test 6: System Status Check
echo -e "\n${YELLOW}⚙️  Test 6: System Status Check${NC}"
status_response=$(curl -s "$API_URL/api/v1/system/status")
if echo "$status_response" | jq -e '.status' >/dev/null 2>&1; then
    echo -e "${GREEN}✅ System status check successful${NC}"
    echo -e "   Status: $(echo "$status_response" | jq -r '.status')"
    echo -e "   Services: $(echo "$status_response" | jq -r '.services | keys | length') monitored"
else
    echo -e "${RED}❌ System status check failed${NC}"
    echo "   Response: $status_response"
    exit 1
fi

# Test 7: CLI Integration Test (if CLI is available)
echo -e "\n${YELLOW}💻 Test 7: CLI Integration${NC}"
if command -v "$CLI_PATH" >/dev/null 2>&1; then
    # Test CLI help
    if "$CLI_PATH" --help >/dev/null 2>&1; then
        echo -e "${GREEN}✅ CLI help command works${NC}"
    else
        echo -e "${YELLOW}⚠️  CLI help command failed${NC}"
    fi
    
    # Test CLI generate command (dry run)
    if "$CLI_PATH" generate test-scenario --types unit --dry-run >/dev/null 2>&1; then
        echo -e "${GREEN}✅ CLI generate command works${NC}"
    else
        echo -e "${YELLOW}⚠️  CLI generate command failed or not implemented${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  CLI not installed or not in PATH${NC}"
    echo "   Install with: cd cli && ./install.sh"
fi

# Test 8: Performance Test (Basic Load)
echo -e "\n${YELLOW}⚡ Test 8: Basic Load Test${NC}"
start_time=$(date +%s)
for i in {1..5}; do
    curl -s "$API_URL/health" >/dev/null &
done
wait
end_time=$(date +%s)
duration=$((end_time - start_time))
if [[ $duration -lt 10 ]]; then
    echo -e "${GREEN}✅ Basic load test passed (${duration}s)${NC}"
else
    echo -e "${YELLOW}⚠️  Basic load test slow (${duration}s)${NC}"
fi

# Final Summary
echo -e "\n${CYAN}📊 Test Results Summary${NC}"
echo -e "${GREEN}✅ Core API functionality verified${NC}"
echo -e "${GREEN}✅ Test generation working${NC}"
echo -e "${GREEN}✅ Coverage analysis functional${NC}"
echo -e "${GREEN}✅ Test vault system operational${NC}"
echo -e "${GREEN}✅ System monitoring active${NC}"

echo -e "\n${BLUE}🎉 Test Genie functionality tests completed successfully!${NC}"
echo -e "${BLUE}   Generated Suite ID: $SUITE_ID${NC}"
echo -e "${BLUE}   Created Vault ID: $VAULT_ID${NC}"
echo -e "${BLUE}   Test environment: $TEST_DIR${NC}"