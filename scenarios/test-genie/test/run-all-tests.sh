#!/bin/bash

# Test Genie Master Test Runner
# Executes all test suites in proper sequence with comprehensive reporting

set -e

# Colors and formatting
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# Test configuration
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="/tmp/test-genie-logs-$$"
PARALLEL=${PARALLEL:-false}

# Create log directory
mkdir -p "$LOG_DIR"

echo -e "${BOLD}${CYAN}🧪 Test Genie Comprehensive Test Suite${NC}"
echo -e "${CYAN}==========================================${NC}"
echo -e "${BLUE}Test Directory: $TEST_DIR${NC}"
echo -e "${BLUE}Log Directory: $LOG_DIR${NC}"
echo -e "${BLUE}Parallel Mode: $PARALLEL${NC}"
echo ""

# Check if test-genie is running
API_PORT=$(vrooli scenario port test-genie API_PORT 2>/dev/null)
if [[ -z "$API_PORT" ]]; then
    echo -e "${RED}❌ test-genie scenario is not running${NC}"
    echo -e "${YELLOW}   Start it with: vrooli scenario run test-genie${NC}"
    echo -e "${YELLOW}   Or use: make run${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Test Genie detected running on port $API_PORT${NC}"
echo ""

# Test suite definitions
declare -A TEST_SUITES=(
    ["basic"]="test-basic-functionality.sh"
    ["database"]="test-database-integration.sh"
    ["ai"]="test-ai-integration.sh"
)

declare -A TEST_DESCRIPTIONS=(
    ["basic"]="Core API functionality and CLI integration"
    ["database"]="Database connectivity, persistence, and performance"
    ["ai"]="AI generation, Ollama integration, and fallback mechanisms"
)

# Track results
declare -A TEST_RESULTS=()
declare -A TEST_DURATIONS=()
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run individual test suite
run_test_suite() {
    local test_key=$1
    local test_script=${TEST_SUITES[$test_key]}
    local test_description=${TEST_DESCRIPTIONS[$test_key]}
    local log_file="$LOG_DIR/${test_key}-test.log"
    
    echo -e "${BOLD}${YELLOW}🔍 Running: $test_description${NC}"
    echo -e "${BLUE}   Script: $test_script${NC}"
    echo -e "${BLUE}   Log: $log_file${NC}"
    
    local start_time=$(date +%s)
    
    if bash "$TEST_DIR/$test_script" > "$log_file" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        TEST_RESULTS[$test_key]="PASSED"
        TEST_DURATIONS[$test_key]="${duration}s"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        
        echo -e "${GREEN}✅ PASSED${NC} (${duration}s)"
        echo -e "${GREEN}   All tests in $test_description completed successfully${NC}"
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        TEST_RESULTS[$test_key]="FAILED"
        TEST_DURATIONS[$test_key]="${duration}s"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        
        echo -e "${RED}❌ FAILED${NC} (${duration}s)"
        echo -e "${RED}   Check log for details: $log_file${NC}"
        echo -e "${YELLOW}   Last 10 lines of log:${NC}"
        tail -10 "$log_file" | sed 's/^/     /'
    fi
    
    echo ""
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

# Pre-test system check
echo -e "${BOLD}${CYAN}📊 Pre-Test System Check${NC}"
echo -e "${CYAN}========================${NC}"

# Check API health
if curl -s "http://localhost:$API_PORT/health" >/dev/null; then
    echo -e "${GREEN}✅ API health check passed${NC}"
else
    echo -e "${RED}❌ API health check failed${NC}"
    exit 1
fi

# Check required commands
for cmd in curl jq; do
    if command -v "$cmd" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ $cmd available${NC}"
    else
        echo -e "${RED}❌ $cmd not found - required for tests${NC}"
        exit 1
    fi
done

echo ""

# Run test suites
echo -e "${BOLD}${CYAN}🚀 Executing Test Suites${NC}"
echo -e "${CYAN}========================${NC}"

if [[ "$PARALLEL" == "true" ]]; then
    echo -e "${YELLOW}Running tests in parallel mode...${NC}"
    echo ""
    
    # Run all tests in parallel
    pids=()
    for test_key in "${!TEST_SUITES[@]}"; do
        run_test_suite "$test_key" &
        pids+=($!)
    done
    
    # Wait for all tests to complete
    wait "${pids[@]}"
else
    echo -e "${YELLOW}Running tests in sequential mode...${NC}"
    echo ""
    
    # Run tests sequentially with specific order
    for test_key in "basic" "database" "ai"; do
        run_test_suite "$test_key"
    done
fi

# Generate comprehensive report
echo -e "${BOLD}${CYAN}📊 Test Results Summary${NC}"
echo -e "${CYAN}======================${NC}"
echo ""

# Individual test results
for test_key in "basic" "database" "ai"; do
    local result=${TEST_RESULTS[$test_key]}
    local duration=${TEST_DURATIONS[$test_key]}
    local description=${TEST_DESCRIPTIONS[$test_key]}
    
    if [[ "$result" == "PASSED" ]]; then
        echo -e "${GREEN}✅ $test_key${NC} - $description (${duration})"
    else
        echo -e "${RED}❌ $test_key${NC} - $description (${duration})"
    fi
done

echo ""

# Overall statistics
echo -e "${BOLD}${CYAN}📈 Overall Statistics${NC}"
echo -e "${CYAN}====================${NC}"
echo -e "${BLUE}Total Test Suites: $TOTAL_TESTS${NC}"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"

if [[ $TOTAL_TESTS -gt 0 ]]; then
    local success_rate=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    echo -e "${BLUE}Success Rate: ${success_rate}%${NC}"
fi

echo ""

# Final status and recommendations
if [[ $FAILED_TESTS -eq 0 ]]; then
    echo -e "${BOLD}${GREEN}🎉 ALL TESTS PASSED! Test Genie is functioning perfectly.${NC}"
    echo -e "${GREEN}✅ Ready for production deployment${NC}"
    echo -e "${GREEN}✅ All core functionality verified${NC}"
    echo -e "${GREEN}✅ Database integration working${NC}"
    echo -e "${GREEN}✅ AI services operational${NC}"
else
    echo -e "${BOLD}${YELLOW}⚠️  SOME TESTS FAILED - Review required${NC}"
    echo ""
    echo -e "${YELLOW}📋 Next Steps:${NC}"
    echo -e "${YELLOW}1. Check individual test logs in: $LOG_DIR${NC}"
    echo -e "${YELLOW}2. Review failed test details above${NC}"
    echo -e "${YELLOW}3. Fix identified issues${NC}"
    echo -e "${YELLOW}4. Re-run tests with: $0${NC}"
fi

echo ""
echo -e "${BOLD}${CYAN}📁 Test Artifacts${NC}"
echo -e "${CYAN}=================${NC}"
echo -e "${BLUE}Log Directory: $LOG_DIR${NC}"
echo -e "${BLUE}Available Logs:${NC}"
ls -la "$LOG_DIR"/*.log 2>/dev/null | sed 's/^/  /' || echo "  No logs found"

echo ""
echo -e "${BOLD}${CYAN}🔧 Useful Commands${NC}"
echo -e "${CYAN}==================${NC}"
echo -e "${BLUE}View specific test log: ${NC}cat $LOG_DIR/<test-name>-test.log"
echo -e "${BLUE}Re-run single test: ${NC}bash $TEST_DIR/<test-script>.sh"
echo -e "${BLUE}Check API status: ${NC}curl http://localhost:$API_PORT/health"
echo -e "${BLUE}Test with parallel mode: ${NC}PARALLEL=true $0"

# Exit with appropriate code
if [[ $FAILED_TESTS -eq 0 ]]; then
    exit 0
else
    exit 1
fi