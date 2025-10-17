#!/bin/bash

# Email Triage Test Suite
# Main test runner that executes phased testing

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test configuration
export TEST_TIMEOUT="${TEST_TIMEOUT:-300}"
export VERBOSE="${VERBOSE:-false}"
export DEV_MODE="${DEV_MODE:-true}"  # Pass DEV_MODE to test phases

# Get ports from scenario status (always read from status to ensure correct ports)
if command -v vrooli >/dev/null 2>&1; then
    SCENARIO_NAME=$(basename "$SCENARIO_DIR")
    PORTS_JSON=$(vrooli scenario status "$SCENARIO_NAME" --json 2>/dev/null || echo '{}')

    # Extract and export API_PORT
    API_PORT_FROM_STATUS=$(echo "$PORTS_JSON" | jq -r '.scenario_data.allocated_ports.API_PORT // empty' 2>/dev/null || echo "")
    if [ -n "$API_PORT_FROM_STATUS" ]; then
        export API_PORT="$API_PORT_FROM_STATUS"
        [ "${VERBOSE:-false}" = "true" ] && echo "ℹ️  Using API_PORT=$API_PORT from scenario status"
    fi

    # Extract and export UI_PORT
    UI_PORT_FROM_STATUS=$(echo "$PORTS_JSON" | jq -r '.scenario_data.allocated_ports.UI_PORT // empty' 2>/dev/null || echo "")
    if [ -n "$UI_PORT_FROM_STATUS" ]; then
        export UI_PORT="$UI_PORT_FROM_STATUS"
        [ "${VERBOSE:-false}" = "true" ] && echo "ℹ️  Using UI_PORT=$UI_PORT from scenario status"
    fi
fi

# Test phases
PHASES=(
    "test-structure"
    "test-health"
    "test-unit"
    "test-api"
    "test-integration"
    "test-ui"
)

# Phase results tracking
declare -A PHASE_RESULTS
FAILED_PHASES=()

echo -e "${BLUE}📧 Email Triage Test Suite${NC}"
echo "================================"

# Function to run a test phase
run_phase() {
    local phase="$1"
    local script="$SCRIPT_DIR/phases/${phase}.sh"
    
    echo -e "\n${BLUE}Running phase: ${phase}${NC}"
    
    if [[ ! -f "$script" ]]; then
        echo -e "${YELLOW}⚠️  Phase script not found: ${phase}${NC}"
        return 0
    fi
    
    if timeout "$TEST_TIMEOUT" bash "$script"; then
        echo -e "${GREEN}✅ Phase passed: ${phase}${NC}"
        PHASE_RESULTS[$phase]="PASSED"
        return 0
    else
        echo -e "${RED}❌ Phase failed: ${phase}${NC}"
        PHASE_RESULTS[$phase]="FAILED"
        FAILED_PHASES+=("$phase")
        return 1
    fi
}

# Main test execution
TOTAL_PHASES=${#PHASES[@]}
PASSED_PHASES=0

for phase in "${PHASES[@]}"; do
    if run_phase "$phase"; then
        PASSED_PHASES=$((PASSED_PHASES + 1))
    fi
done

# Print summary
echo -e "\n${BLUE}=== Test Summary ===${NC}"
echo "Total phases: $TOTAL_PHASES"
echo "Passed: $PASSED_PHASES"
echo "Failed: ${#FAILED_PHASES[@]}"

if [[ ${#FAILED_PHASES[@]} -gt 0 ]]; then
    echo -e "\n${RED}Failed phases:${NC}"
    for phase in "${FAILED_PHASES[@]}"; do
        echo "  - $phase"
    done
    exit 1
else
    echo -e "\n${GREEN}All test phases passed!${NC}"
    exit 0
fi