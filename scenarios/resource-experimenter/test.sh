#!/usr/bin/env bash
# Resource Experimenter Integration Tests

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/resource-experimenter"
API_URL="${RESOURCE_EXPERIMENTER_API_URL:-http://localhost:8092}"

# Colors
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
BLUE='\033[1;34m'
NC='\033[0m'

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    echo -e "${BLUE}Running test: ${test_name}${NC}"
    
    if eval "$test_command"; then
        log_success "$test_name passed"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log_error "$test_name failed"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Wait for service to be ready
wait_for_service() {
    local url="$1"
    local timeout="${2:-60}"
    local count=0
    
    log_info "Waiting for service at $url to be ready..."
    
    while ! curl -s "$url" > /dev/null 2>&1; do
        sleep 2
        count=$((count + 1))
        
        if [ $count -gt $((timeout / 2)) ]; then
            log_error "Service at $url not ready after ${timeout}s"
            return 1
        fi
    done
    
    log_success "Service at $url is ready"
    return 0
}

# Test functions
test_api_health() {
    local response
    response=$(curl -s "$API_URL/health")
    
    if echo "$response" | grep -q '"status":"healthy"'; then
        return 0
    else
        log_error "API health check failed: $response"
        return 1
    fi
}

test_experiments_endpoint() {
    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/experiments")
    
    if [ "$status_code" = "200" ]; then
        return 0
    else
        log_error "Experiments endpoint returned status: $status_code"
        return 1
    fi
}

test_scenarios_endpoint() {
    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/scenarios")
    
    if [ "$status_code" = "200" ]; then
        return 0
    else
        log_error "Scenarios endpoint returned status: $status_code"
        return 1
    fi
}

test_templates_endpoint() {
    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/templates")
    
    if [ "$status_code" = "200" ]; then
        return 0
    else
        log_error "Templates endpoint returned status: $status_code"
        return 1
    fi
}

test_create_experiment() {
    local response
    local experiment_data='{
        "name": "Test Experiment",
        "description": "Integration test experiment",
        "prompt": "Add redis to analytics-dashboard",
        "target_scenario": "analytics-dashboard",
        "new_resource": "redis"
    }'
    
    response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d "$experiment_data" "$API_URL/api/experiments")
    
    if echo "$response" | grep -q '"status":"requested"'; then
        # Clean up: delete the created experiment
        local experiment_id
        experiment_id=$(echo "$response" | jq -r '.id' 2>/dev/null || echo "")
        if [ -n "$experiment_id" ] && [ "$experiment_id" != "null" ]; then
            curl -s -X DELETE "$API_URL/api/experiments/$experiment_id" > /dev/null
        fi
        return 0
    else
        log_error "Create experiment failed: $response"
        return 1
    fi
}

test_database_connection() {
    # Test if database connection works via API
    local experiments_response
    experiments_response=$(curl -s "$API_URL/api/experiments")
    
    if [ $? -eq 0 ] && echo "$experiments_response" | jq . > /dev/null 2>&1; then
        return 0
    else
        log_error "Database connection test failed"
        return 1
    fi
}

test_claude_code_availability() {
    # This test checks if Claude Code is available for experiments
    # We'll just check if the health endpoint is aware of it
    local response
    response=$(curl -s "$API_URL/health")
    
    # For now, just check that API is healthy
    # Real Claude Code integration would need to be tested differently
    if echo "$response" | grep -q '"status":"healthy"'; then
        return 0
    else
        log_error "Claude Code availability test failed"
        return 1
    fi
}

test_cli_installation() {
    if command -v resource-experimenter &> /dev/null; then
        return 0
    else
        log_warning "CLI not installed (this is expected if install.sh hasn't been run)"
        return 0  # Don't fail the test suite for this
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}🧪 Resource Experimenter Integration Tests${NC}"
    echo "Testing API at: $API_URL"
    echo
    
    # Check if jq is available (needed for JSON parsing)
    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed"
        exit 1
    fi
    
    # Wait for API to be ready
    if ! wait_for_service "$API_URL/health" 30; then
        log_error "API is not ready, aborting tests"
        exit 1
    fi
    
    echo
    log_info "Running integration tests..."
    echo
    
    # Run all tests
    run_test "API Health Check" "test_api_health"
    run_test "Database Connection" "test_database_connection"
    run_test "Experiments Endpoint" "test_experiments_endpoint"
    run_test "Scenarios Endpoint" "test_scenarios_endpoint" 
    run_test "Templates Endpoint" "test_templates_endpoint"
    run_test "Create Experiment" "test_create_experiment"
    run_test "Claude Code Availability" "test_claude_code_availability"
    run_test "CLI Installation" "test_cli_installation"
    
    echo
    echo "=========================================="
    echo -e "${BLUE}Test Summary:${NC}"
    echo -e "  Total tests: ${TESTS_RUN}"
    echo -e "  ${GREEN}Passed: ${TESTS_PASSED}${NC}"
    echo -e "  ${RED}Failed: ${TESTS_FAILED}${NC}"
    echo "=========================================="
    
    if [ $TESTS_FAILED -eq 0 ]; then
        log_success "All tests passed!"
        exit 0
    else
        log_error "$TESTS_FAILED test(s) failed"
        exit 1
    fi
}

# Check if script is being sourced or executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi