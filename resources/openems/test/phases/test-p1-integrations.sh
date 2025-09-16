#!/usr/bin/env bash

# Test P1 Integration Features for OpenEMS
# Validates n8n, Superset, Ditto, and Forecast integrations

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -e "${YELLOW}Testing:${NC} ${test_name}"
    
    if eval "${test_command}" &>/dev/null; then
        echo -e "${GREEN}✅ PASSED${NC}: ${test_name}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}❌ FAILED${NC}: ${test_name}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Header
echo "========================================="
echo "    OpenEMS P1 Integration Tests        "
echo "========================================="
echo ""

# Test n8n Integration
echo -e "\n${YELLOW}=== n8n Workflow Integration ===${NC}"
run_test "n8n workflow creation" \
    "${RESOURCE_DIR}/cli.sh n8n create-workflows"
run_test "n8n workflow templates exist" \
    "test -f /tmp/openems-workflow.json && test -f /tmp/solar-optimization-workflow.json"

# Test Superset Integration  
echo -e "\n${YELLOW}=== Apache Superset Integration ===${NC}"
run_test "Superset dashboard creation" \
    "${RESOURCE_DIR}/cli.sh superset create-dashboards"
run_test "Superset templates exist" \
    "test -f /tmp/energy-overview-dashboard.json && test -f /tmp/solar-analytics-dashboard.json"
run_test "QuestDB views created" \
    "test -f /tmp/questdb-openems-views.sql"

# Test Ditto Integration
echo -e "\n${YELLOW}=== Eclipse Ditto Integration ===${NC}"
run_test "Ditto twin creation" \
    "${RESOURCE_DIR}/cli.sh ditto create-twins"
run_test "Ditto twin templates exist" \
    "test -f /tmp/solar-panel-twin.json && test -f /tmp/battery-storage-twin.json"
run_test "Co-simulation bridge creation" \
    "${RESOURCE_DIR}/cli.sh ditto create-cosim"
run_test "SimPy bridge exists" \
    "test -f /tmp/simpy-cosim-bridge.py"

# Test Forecast Models
echo -e "\n${YELLOW}=== Energy Forecast Models ===${NC}"
run_test "Forecast model creation" \
    "${RESOURCE_DIR}/cli.sh forecast create-models"
run_test "Solar forecast model" \
    "python3 /tmp/solar_forecast.py 10 hourly"
run_test "Battery forecast model" \
    "python3 /tmp/battery_forecast.py schedule"
run_test "Consumption forecast model" \
    "python3 /tmp/consumption_forecast.py residential hourly"
run_test "Integrated forecast" \
    "${RESOURCE_DIR}/cli.sh forecast integrated"

# Test CLI Commands
echo -e "\n${YELLOW}=== CLI Command Integration ===${NC}"
run_test "n8n command in help" \
    "${RESOURCE_DIR}/cli.sh help | grep -q 'n8n'"
run_test "superset command in help" \
    "${RESOURCE_DIR}/cli.sh help | grep -q 'superset'"
run_test "ditto command in help" \
    "${RESOURCE_DIR}/cli.sh help | grep -q 'ditto'"
run_test "forecast command in help" \
    "${RESOURCE_DIR}/cli.sh help | grep -q 'forecast'"

# Summary
echo ""
echo "========================================="
echo "           TEST SUMMARY                 "
echo "========================================="
echo -e "Tests Run:    ${TESTS_RUN}"
echo -e "Tests Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Tests Failed: ${RED}${TESTS_FAILED}${NC}"

if [[ ${TESTS_FAILED} -eq 0 ]]; then
    echo -e "\n${GREEN}✅ All P1 integration tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed. Please review.${NC}"
    exit 1
fi