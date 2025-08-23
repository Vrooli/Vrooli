#!/usr/bin/env bash

set -euo pipefail

# Integration test for Workflow Scheduler scenario

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly API_URL="${SCHEDULER_API_URL:-http://localhost:8090}"

# Color codes
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m'

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "Testing ${test_name}... "
    
    if eval "${test_command}" &> /dev/null; then
        echo -e "${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
}

echo "======================================"
echo "Workflow Scheduler Integration Tests"
echo "======================================"
echo ""

# Test 1: API Health Check
run_test "API health endpoint" "curl -sf ${API_URL}/health"

# Test 2: Database connectivity
run_test "Database connection" "curl -sf ${API_URL}/api/system/db-status"

# Test 3: Redis connectivity
run_test "Redis connection" "curl -sf ${API_URL}/api/system/redis-status"

# Test 4: List schedules endpoint
run_test "List schedules endpoint" "curl -sf ${API_URL}/api/schedules"

# Test 5: Cron validation endpoint
run_test "Cron validation" "curl -sf '${API_URL}/api/cron/validate?expression=0%209%20*%20*%20*'"

# Test 6: Cron presets endpoint
run_test "Cron presets" "curl -sf ${API_URL}/api/cron/presets"

# Test 7: Dashboard stats endpoint
run_test "Dashboard statistics" "curl -sf ${API_URL}/api/dashboard/stats"

# Test 8: Create a test schedule
TEST_SCHEDULE_ID=""
if response=$(curl -sf -X POST ${API_URL}/api/schedules \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Test Schedule",
        "cron_expression": "0 * * * *",
        "target_type": "webhook",
        "target_url": "http://localhost:5678/webhook/test",
        "timezone": "UTC",
        "enabled": false
    }' 2>/dev/null); then
    TEST_SCHEDULE_ID=$(echo "${response}" | jq -r '.id' 2>/dev/null || echo "")
    if [[ -n "${TEST_SCHEDULE_ID}" ]]; then
        echo -e "Create test schedule... ${GREEN}✓${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "Create test schedule... ${RED}✗${NC}"
        ((TESTS_FAILED++))
    fi
else
    echo -e "Create test schedule... ${RED}✗${NC}"
    ((TESTS_FAILED++))
fi

# Test 9: Get schedule details
if [[ -n "${TEST_SCHEDULE_ID}" ]]; then
    run_test "Get schedule details" "curl -sf ${API_URL}/api/schedules/${TEST_SCHEDULE_ID}"
fi

# Test 10: Update schedule
if [[ -n "${TEST_SCHEDULE_ID}" ]]; then
    run_test "Update schedule" "curl -sf -X PUT ${API_URL}/api/schedules/${TEST_SCHEDULE_ID} -H 'Content-Type: application/json' -d '{\"description\":\"Updated description\"}'"
fi

# Test 11: Enable schedule
if [[ -n "${TEST_SCHEDULE_ID}" ]]; then
    run_test "Enable schedule" "curl -sf -X POST ${API_URL}/api/schedules/${TEST_SCHEDULE_ID}/enable"
fi

# Test 12: Disable schedule
if [[ -n "${TEST_SCHEDULE_ID}" ]]; then
    run_test "Disable schedule" "curl -sf -X POST ${API_URL}/api/schedules/${TEST_SCHEDULE_ID}/disable"
fi

# Test 13: Get schedule metrics
if [[ -n "${TEST_SCHEDULE_ID}" ]]; then
    run_test "Get schedule metrics" "curl -sf ${API_URL}/api/schedules/${TEST_SCHEDULE_ID}/metrics"
fi

# Test 14: Preview next runs
if [[ -n "${TEST_SCHEDULE_ID}" ]]; then
    run_test "Preview next runs" "curl -sf ${API_URL}/api/schedules/${TEST_SCHEDULE_ID}/next-runs?count=5"
fi

# Test 15: Get execution history
if [[ -n "${TEST_SCHEDULE_ID}" ]]; then
    run_test "Get execution history" "curl -sf ${API_URL}/api/schedules/${TEST_SCHEDULE_ID}/executions"
fi

# Test 16: Delete test schedule
if [[ -n "${TEST_SCHEDULE_ID}" ]]; then
    run_test "Delete schedule" "curl -sf -X DELETE ${API_URL}/api/schedules/${TEST_SCHEDULE_ID}"
fi

# Test 17: CLI availability
run_test "CLI installed" "command -v scheduler-cli"

# Test 18: n8n workflows endpoint
run_test "n8n engine reachable" "curl -sf http://localhost:5678/healthz || curl -sf http://localhost:5678"

# Test 19: Check if PostgreSQL tables exist
run_test "Database schema" "PGPASSWORD=postgres psql -h localhost -p 5432 -U postgres -d workflow_scheduler -c 'SELECT COUNT(*) FROM schedules;' 2>/dev/null"

echo ""
echo "======================================"
echo "Test Results"
echo "======================================"
echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
echo ""

if [[ ${TESTS_FAILED} -eq 0 ]]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed${NC}"
    exit 1
fi