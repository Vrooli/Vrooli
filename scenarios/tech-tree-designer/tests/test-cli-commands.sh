#!/bin/bash

# Tech Tree Designer CLI Command Tests
# Validates all strategic intelligence CLI commands

set -e

# Configuration
CLI_PATH="$(dirname "$0")/../cli/tech-tree-designer"
API_PORT=${API_PORT:-8080}

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TESTS_PASSED=0
TESTS_FAILED=0

log_test() {
    echo -e "${BLUE}🧪 Testing: $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
    ((TESTS_PASSED += 1))
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
    ((TESTS_FAILED += 1))
}

test_cli_command() {
    local command="$1"
    local expected_exit_code="$2"
    local expected_content="$3"
    local description="$4"
    local exit_code=0
    
    log_test "$description"
    
    # Run command and capture output and exit code
    output=$($command 2>&1) || exit_code=$?
    
    # Check exit code
    if [[ $exit_code -eq $expected_exit_code ]]; then
        # Check output content if specified
        if [[ -n "$expected_content" ]]; then
            if echo "$output" | grep -qi "$expected_content"; then
                log_success "$description - Exit code $exit_code, contains '$expected_content'"
                return 0
            else
                log_error "$description - Missing expected content '$expected_content'"
                echo "Output: $output"
                return 1
            fi
        else
            log_success "$description - Exit code $exit_code"
            return 0
        fi
    else
        log_error "$description - Expected exit code $expected_exit_code, got $exit_code"
        echo "Output: $output"
        return 1
    fi
}

test_json_output() {
    local command="$1"
    local description="$2"
    
    log_test "$description"
    
    if output=$($command 2>/dev/null); then
        if echo "$output" | jq . > /dev/null 2>&1; then
            log_success "$description - Valid JSON output"
            return 0
        else
            log_error "$description - Invalid JSON output"
            echo "Output: $output"
            return 1
        fi
    else
        log_error "$description - Command failed"
        return 1
    fi
}

echo -e "${BLUE}⚡ Tech Tree Designer CLI Tests${NC}"
echo -e "${BLUE}================================${NC}"

# Check if CLI exists and is executable
log_test "CLI executable exists"
if [[ -x "$CLI_PATH" ]]; then
    log_success "CLI found at $CLI_PATH"
else
    log_error "CLI not found or not executable at $CLI_PATH"
    exit 1
fi

echo -e "\n${YELLOW}🔧 Basic Commands${NC}"

# Test help command
test_cli_command "$CLI_PATH help" 0 "Strategic Intelligence" "Help command"

# Test version command  
test_cli_command "$CLI_PATH version" 0 "Tech Tree Designer CLI" "Version command"

# Test invalid command
test_cli_command "$CLI_PATH invalid-command" 1 "Unknown command" "Invalid command handling"

echo -e "\n${YELLOW}📊 Status Commands${NC}"

# Test basic status
test_cli_command "$CLI_PATH status" 0 "" "Status command (API online)"

# If API is running, test more commands
if curl -sf "http://localhost:$API_PORT/health" > /dev/null 2>&1; then
    echo -e "\n${GREEN}🚀 API is running - Testing full command set${NC}"
    
    # Test status with verbose
    test_cli_command "$CLI_PATH status --verbose" 0 "Sector Progress" "Verbose status command"
    
    # Test status with JSON output
    test_json_output "$CLI_PATH status --json" "Status JSON output"
    
    echo -e "\n${YELLOW}🧠 Analysis Commands${NC}"
    
    # Test analyze command
    test_cli_command "$CLI_PATH analyze --resources 5 --timeline 12" 0 "Strategic Intelligence" "Basic analysis command"
    
    # Test analyze with JSON output
    test_json_output "$CLI_PATH analyze --resources 5 --timeline 12 --json" "Analysis JSON output"
    
    # Test analyze with priority sectors
    test_cli_command "$CLI_PATH analyze --resources 8 --priority software,manufacturing" 0 "recommendations" "Analysis with priority sectors"
    
    echo -e "\n${YELLOW}📋 Information Commands${NC}"
    
    # Test sectors command
    test_cli_command "$CLI_PATH sectors" 0 "Technology Sectors" "Sectors listing"
    
    # Test sectors with JSON
    test_json_output "$CLI_PATH sectors --json" "Sectors JSON output"
    
    # Test milestones command
    test_cli_command "$CLI_PATH milestones" 0 "Strategic Milestones" "Milestones listing"
    
    # Test milestones with JSON
    test_json_output "$CLI_PATH milestones --json" "Milestones JSON output"
    
    # Test dependencies command
    test_cli_command "$CLI_PATH dependencies" 0 "Strategic Dependencies" "Dependencies listing"
    
    # Test dependencies with bottlenecks flag
    test_cli_command "$CLI_PATH dependencies --bottlenecks" 0 "Critical Dependencies" "Dependencies bottlenecks"
    
    echo -e "\n${YELLOW}🎯 Recommendation Commands${NC}"
    
    # Test recommend command
    test_cli_command "$CLI_PATH recommend" 0 "recommendations" "Basic recommendations"
    
    # Test recommend with priority
    test_cli_command "$CLI_PATH recommend --priority software,healthcare" 0 "recommendations" "Priority recommendations"
    
    # Test recommend with JSON
    test_json_output "$CLI_PATH recommend --json" "Recommendations JSON output"
    
    echo -e "\n${YELLOW}📈 Progress Commands${NC}"
    
    # Test progress listing
    test_cli_command "$CLI_PATH progress --list" 0 "Scenario Progress" "Progress listing"
    
    # Test progress with JSON
    test_json_output "$CLI_PATH progress --list --json" "Progress JSON output"
    
    # Test progress update (create test scenario)
    test_cli_command "$CLI_PATH progress --scenario cli-test-scenario --status in_progress" 0 "updated" "Progress update command"
    
    # Test updating same scenario to completed
    test_cli_command "$CLI_PATH progress --scenario cli-test-scenario --status completed" 0 "updated" "Progress completion update"
    
    echo -e "\n${YELLOW}🔍 Edge Cases${NC}"
    
    # Test command with missing parameters
    test_cli_command "$CLI_PATH analyze" 0 "" "Analysis without parameters (should use defaults)"
    
    # Test progress without required parameters
    test_cli_command "$CLI_PATH progress" 1 "Usage" "Progress without parameters (should show usage)"
    
    # Test invalid sector category
    test_cli_command "$CLI_PATH sectors --category nonexistent" 0 "" "Invalid sector category (should return empty)"
    
    echo -e "\n${YELLOW}⚡ Performance Tests${NC}"
    
    # Test command response times
    log_test "Command response time"
    start_time=$(date +%s%N)
    $CLI_PATH status > /dev/null 2>&1 || true
    end_time=$(date +%s%N)
    response_time=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds
    
    if [[ $response_time -lt 5000 ]]; then
        log_success "CLI response time: ${response_time}ms (under 5s threshold)"
    else
        log_error "CLI response time: ${response_time}ms (exceeds 5s threshold)"
    fi
    
else
    echo -e "\n${YELLOW}⚠️  API not running - Skipping API-dependent CLI tests${NC}"
    echo -e "${YELLOW}Start API with: vrooli scenario run tech-tree-designer${NC}"
fi

echo -e "\n${YELLOW}📋 Help System Tests${NC}"

# Test help for specific commands
test_cli_command "$CLI_PATH help" 0 "analyze" "Analyze command help"
test_cli_command "$CLI_PATH help" 0 "progress" "Progress command help"
test_cli_command "$CLI_PATH help" 0 "status" "Status command help"

echo -e "\n${YELLOW}🔧 Installation Tests${NC}"

# Test CLI install script exists
log_test "Install script exists"
if [[ -f "$(dirname "$CLI_PATH")/install.sh" ]]; then
    log_success "Install script found"
else
    log_error "Install script not found"
fi

# Test CLI permissions
log_test "CLI has correct permissions"
if [[ -x "$CLI_PATH" ]]; then
    log_success "CLI is executable"
else
    log_error "CLI lacks execute permissions"
fi

echo -e "\n${BLUE}📊 Test Results Summary${NC}"
echo -e "${BLUE}======================${NC}"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "\n${GREEN}🎉 All CLI tests passed! Strategic Intelligence Command Interface is operational.${NC}"
    exit 0
else
    echo -e "\n${RED}⚠️  Some CLI tests failed. Check the implementation.${NC}"
    exit 1
fi
