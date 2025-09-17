#!/bin/bash

# API Manager - Main Test Runner
# Orchestrates all integration tests for the API Manager scenario

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="$SCRIPT_DIR/phases"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL="${API_MANAGER_URL:-http://localhost:${API_PORT}}"
VERBOSE="${VERBOSE:-false}"

# Print banner
print_banner() {
    echo -e "${CYAN}"
    echo "=================================================================="
    echo "           🔍 API Manager Integration Test Suite"
    echo "=================================================================="
    echo -e "${NC}"
    echo "API Base URL: $API_BASE_URL"
    echo "Test Directory: $PHASES_DIR"
    echo "Verbose Mode: $VERBOSE"
    echo
}

# Check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        echo -e "${RED}✗ jq is required but not installed${NC}"
        echo "  Install with: apt-get install jq (Ubuntu/Debian) or brew install jq (macOS)"
        exit 1
    fi
    echo -e "${GREEN}✓ jq is available${NC}"
    
    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        echo -e "${RED}✗ curl is required but not installed${NC}"
        echo "  Install with: apt-get install curl (Ubuntu/Debian)"
        exit 1
    fi
    echo -e "${GREEN}✓ curl is available${NC}"
    
    # Check if API is reachable
    echo -e "${YELLOW}Testing API connectivity...${NC}"
    if curl -s --connect-timeout 5 "$API_BASE_URL/api/v1/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ API Manager is reachable${NC}"
    else
        echo -e "${RED}✗ Cannot reach API Manager at $API_BASE_URL${NC}"
        echo "  Make sure the API Manager is running and accessible"
        echo "  You can start it with: vrooli scenario run api-manager"
        exit 1
    fi
    
    echo
}

# Run test phase
run_test_phase() {
    local test_file="$1"
    local test_name="$(basename "$test_file" .sh)"
    
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}Running: $test_name${NC}"
    echo -e "${BLUE}================================${NC}"
    
    # Make sure test file is executable
    chmod +x "$test_file"
    
    local start_time=$(date +%s)
    local exit_code=0
    
    if [[ "$VERBOSE" == "true" ]]; then
        # Run with full output
        "$test_file" || exit_code=$?
    else
        # Capture output and show summary
        local output_file="/tmp/test_${test_name}_output.log"
        "$test_file" > "$output_file" 2>&1 || exit_code=$?
        
        if [[ $exit_code -eq 0 ]]; then
            echo -e "${GREEN}✓ $test_name completed successfully${NC}"
            # Show summary lines
            grep -E "(✓|✗|Test Summary|Total tests|Passed|Failed)" "$output_file" | tail -10
        else
            echo -e "${RED}✗ $test_name failed${NC}"
            # Show error details
            tail -20 "$output_file"
        fi
        
        rm -f "$output_file"
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    echo -e "${BLUE}Duration: ${duration}s${NC}"
    echo
    
    return $exit_code
}

# Main test execution
main() {
    print_banner
    check_prerequisites
    
    echo -e "${YELLOW}Starting API Manager Integration Tests...${NC}"
    echo
    
    local total_phases=0
    local passed_phases=0
    local start_time=$(date +%s)
    
    # Test phases in order
    test_phases=(
        "$PHASES_DIR/test-health-monitoring.sh"
        "$PHASES_DIR/test-automated-fixes.sh"
        "$PHASES_DIR/test-performance-monitoring.sh"
    )
    
    # Run each test phase
    for test_phase in "${test_phases[@]}"; do
        if [[ -f "$test_phase" ]]; then
            total_phases=$((total_phases + 1))
            
            if run_test_phase "$test_phase"; then
                passed_phases=$((passed_phases + 1))
                echo -e "${GREEN}✅ Phase $(basename "$test_phase" .sh) PASSED${NC}"
            else
                echo -e "${RED}❌ Phase $(basename "$test_phase" .sh) FAILED${NC}"
            fi
            echo
        else
            echo -e "${YELLOW}⚠ Test phase not found: $test_phase${NC}"
        fi
    done
    
    # Final summary
    local end_time=$(date +%s)
    local total_duration=$((end_time - start_time))
    
    echo -e "${CYAN}=================================================================="
    echo "                    FINAL TEST SUMMARY"
    echo "=================================================================="
    echo -e "${NC}"
    echo "Total test phases: $total_phases"
    echo "Passed phases: $passed_phases"
    echo "Failed phases: $((total_phases - passed_phases))"
    echo "Total duration: ${total_duration}s"
    echo
    
    if [[ $passed_phases -eq $total_phases ]]; then
        echo -e "${GREEN}🎉 ALL INTEGRATION TESTS PASSED! 🎉${NC}"
        echo -e "${GREEN}API Manager is functioning correctly${NC}"
        exit 0
    else
        echo -e "${RED}❌ SOME TESTS FAILED${NC}"
        echo -e "${RED}Please review the test output and fix any issues${NC}"
        exit 1
    fi
}

# Handle command line options
case "${1:-}" in
    --help|-h)
        echo "API Manager Integration Test Suite"
        echo
        echo "Usage: $0 [options]"
        echo
        echo "Options:"
        echo "  --help, -h     Show this help message"
        echo "  --verbose, -v  Run tests in verbose mode"
        echo "  --check        Only check prerequisites"
        echo
        echo "Environment Variables:"
        echo "  API_MANAGER_URL  Base URL for API Manager (default: http://localhost:\$API_PORT)"
        echo "  API_PORT         API port number (default: 8100)"
        echo "  VERBOSE          Enable verbose output (default: false)"
        echo
        echo "Examples:"
        echo "  $0                           # Run all tests"
        echo "  $0 --verbose                 # Run all tests with verbose output"
        echo "  API_PORT=8200 $0            # Run tests against API on port 8200"
        exit 0
        ;;
    --verbose|-v)
        VERBOSE=true
        ;;
    --check)
        print_banner
        check_prerequisites
        echo -e "${GREEN}✓ All prerequisites satisfied${NC}"
        exit 0
        ;;
esac

# Run main function
main "$@"