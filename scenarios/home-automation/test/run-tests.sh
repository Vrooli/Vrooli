#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
SCENARIO_NAME="home-automation"
TEST_ARTIFACT_DIR="test/artifacts"
TIMESTAMP=$(date +%s)

echo -e "${BLUE}🧪 Running Home Automation Test Suite${NC}"
echo "==============================================="

# Create artifacts directory
mkdir -p "$TEST_ARTIFACT_DIR"

# Test phases to run
PHASES=(
    "structure"
    "dependencies"
    "unit"
    "integration"
    "api"
    "business"
    "performance"
)

# Track test results
PASSED_PHASES=()
FAILED_PHASES=()
TOTAL_PHASES=${#PHASES[@]}

# Function to run a test phase
run_phase() {
    local phase=$1
    local script="test/phases/test-${phase}.sh"
    local log_file="${TEST_ARTIFACT_DIR}/${phase}-${TIMESTAMP}.log"

    echo -e "${YELLOW}Running ${phase} tests...${NC}"

    if [ -f "$script" ]; then
        if bash "$script" > "$log_file" 2>&1; then
            echo -e "${GREEN}✅ ${phase} tests passed${NC}"
            PASSED_PHASES+=("$phase")
            return 0
        else
            echo -e "${RED}❌ ${phase} tests failed${NC}"
            echo -e "${RED}   Log: $log_file${NC}"
            FAILED_PHASES+=("$phase")
            # Show last 20 lines of log for quick debugging
            echo -e "${YELLOW}   Last 20 lines:${NC}"
            tail -20 "$log_file" | sed 's/^/   /'
            return 1
        fi
    else
        echo -e "${YELLOW}⚠️  ${phase} test script not found: $script${NC}"
        return 0
    fi
}

# Function to check prerequisites
check_prerequisites() {
    echo -e "${BLUE}Checking prerequisites...${NC}"

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        echo -e "${RED}❌ Go is not installed${NC}"
        return 1
    fi
    echo -e "${GREEN}✅ Go is installed${NC}"

    # Check if curl is installed
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}❌ curl is not installed${NC}"
        return 1
    fi
    echo -e "${GREEN}✅ curl is installed${NC}"

    # Check if scenario structure exists
    if [ ! -f ".vrooli/service.json" ]; then
        echo -e "${RED}❌ .vrooli/service.json not found${NC}"
        return 1
    fi
    echo -e "${GREEN}✅ service.json exists${NC}"

    # Check if API can be built
    if [ -d "api" ] && [ -f "api/go.mod" ]; then
        echo -e "${GREEN}✅ API structure exists${NC}"
    else
        echo -e "${RED}❌ API structure missing${NC}"
        return 1
    fi

    return 0
}

# Main test execution
main() {
    # Check prerequisites
    if ! check_prerequisites; then
        echo -e "${RED}❌ Prerequisites check failed${NC}"
        exit 1
    fi

    echo ""
    echo -e "${BLUE}Running test phases...${NC}"
    echo ""

    # Run each test phase
    for phase in "${PHASES[@]}"; do
        run_phase "$phase" || true  # Continue even if a phase fails
        echo ""
    done

    # Print summary
    echo -e "${BLUE}Test Summary${NC}"
    echo "============================================="
    echo -e "Total phases: ${TOTAL_PHASES}"
    echo -e "${GREEN}Passed: ${#PASSED_PHASES[@]}${NC}"
    echo -e "${RED}Failed: ${#FAILED_PHASES[@]}${NC}"

    if [ ${#PASSED_PHASES[@]} -gt 0 ]; then
        echo -e "${GREEN}Passed phases: ${PASSED_PHASES[*]}${NC}"
    fi

    if [ ${#FAILED_PHASES[@]} -gt 0 ]; then
        echo -e "${RED}Failed phases: ${FAILED_PHASES[*]}${NC}"
        echo ""
        echo -e "${YELLOW}To view detailed logs:${NC}"
        for phase in "${FAILED_PHASES[@]}"; do
            echo -e "  cat ${TEST_ARTIFACT_DIR}/${phase}-${TIMESTAMP}.log"
        done
        exit 1
    fi

    echo ""
    echo -e "${GREEN}✅ All tests passed!${NC}"
}

# Run main function
main "$@"
