#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
SCENARIO_NAME="data-backup-manager"
TEST_ARTIFACT_DIR="test/artifacts"
TIMESTAMP=$(date +%s)

echo -e "${BLUE}🧪 Running Data Backup Manager Test Suite${NC}"
echo "==============================================="

# Create artifacts directory
mkdir -p "$TEST_ARTIFACT_DIR"

# Test phases to run
PHASES=(
    "structure"
    "dependencies"
    "integration"
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

    # Check if curl is installed
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}❌ curl is not installed${NC}"
        return 1
    fi

    # Check if scenario structure exists
    if [ ! -f ".vrooli/service.json" ]; then
        echo -e "${RED}❌ .vrooli/service.json not found${NC}"
        return 1
    fi

    # Check if scenario is running
    if ! vrooli scenario status "$SCENARIO_NAME" &> /dev/null; then
        echo -e "${YELLOW}⚠️  Scenario not running - some tests may fail${NC}"
    fi

    echo -e "${GREEN}✅ Prerequisites check passed${NC}"
    return 0
}

# Function to print test summary
print_summary() {
    echo ""
    echo "==============================================="
    echo -e "${BLUE}📊 Test Summary${NC}"
    echo "==============================================="
    echo "Total phases: $TOTAL_PHASES"
    echo -e "Passed: ${GREEN}${#PASSED_PHASES[@]}${NC}"
    echo -e "Failed: ${RED}${#FAILED_PHASES[@]}${NC}"

    if [ ${#PASSED_PHASES[@]} -gt 0 ]; then
        echo -e "\n${GREEN}✅ Passed phases:${NC}"
        for phase in "${PASSED_PHASES[@]}"; do
            echo "  - $phase"
        done
    fi

    if [ ${#FAILED_PHASES[@]} -gt 0 ]; then
        echo -e "\n${RED}❌ Failed phases:${NC}"
        for phase in "${FAILED_PHASES[@]}"; do
            echo "  - $phase"
        done
    fi

    echo ""
    echo "Artifacts saved in: $TEST_ARTIFACT_DIR"
    echo "Timestamp: $TIMESTAMP"
}

# Main execution
main() {
    # Check if we should run specific phases
    if [ $# -gt 0 ]; then
        PHASES=("$@")
        echo -e "${YELLOW}Running specific phases: ${PHASES[*]}${NC}"
    fi

    # Check prerequisites
    if ! check_prerequisites; then
        echo -e "${RED}❌ Prerequisites check failed${NC}"
        exit 1
    fi

    echo ""

    # Run each test phase
    for phase in "${PHASES[@]}"; do
        run_phase "$phase"
        echo ""
    done

    # Print summary
    print_summary

    # Exit with appropriate code
    if [ ${#FAILED_PHASES[@]} -eq 0 ]; then
        echo -e "${GREEN}🎉 All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}💥 Some tests failed${NC}"
        exit 1
    fi
}

# Help function
show_help() {
    echo "Usage: $0 [phases...]"
    echo ""
    echo "Run data-backup-manager test suite"
    echo ""
    echo "Available phases:"
    for phase in "${PHASES[@]}"; do
        echo "  - $phase"
    done
    echo ""
    echo "Examples:"
    echo "  $0                    # Run all phases"
    echo "  $0 structure integration   # Run specific phases"
    echo ""
}

# Parse arguments
case "${1:-}" in
    -h|--help|help)
        show_help
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac
