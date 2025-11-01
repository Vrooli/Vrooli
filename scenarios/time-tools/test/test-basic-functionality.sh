#!/bin/bash
################################################################################
# Time Tools Basic Functionality Tests
# Tests core API endpoints and CLI commands
################################################################################

set -e

# Configuration
SCENARIO_NAME="time-tools"
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$TEST_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

################################################################################
# Helper Functions
################################################################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    ((TESTS_TOTAL++))
    
    log_info "Running test: $test_name"
    
    if eval "$test_command"; then
        log_success "$test_name"
        return 0
    else
        log_error "$test_name"
        return 1
    fi
}

get_api_url() {
    local api_port

    # Prefer modern API_PORT but fall back to legacy TIME_TOOLS_PORT
    api_port="${API_PORT:-${TIME_TOOLS_PORT:-}}"

    # If not set, try to get from vrooli CLI using both port keys
    if [[ -z "$api_port" ]]; then
        api_port=$(vrooli scenario port "${SCENARIO_NAME}" API_PORT 2>/dev/null || true)
    fi
    if [[ -z "$api_port" ]]; then
        api_port=$(vrooli scenario port "${SCENARIO_NAME}" TIME_TOOLS_PORT 2>/dev/null || true)
    fi

    # Fall back to default port
    if [[ -z "$api_port" ]]; then
        api_port=18765
    fi

    # Verify the API is actually responding
    if ! curl -sf "http://localhost:${api_port}/api/v1/health" >/dev/null 2>&1; then
        log_error "Time Tools API is not responding at port ${api_port}. Ensure scenario is running: vrooli scenario run ${SCENARIO_NAME}"
        return 1
    fi

    echo "http://localhost:${api_port}"
}

################################################################################
# API Tests
################################################################################

test_health_endpoint() {
    local api_url
    api_url=$(get_api_url) || return 1
    curl -sf "${api_url}/api/v1/health" >/dev/null
}

test_timezone_conversion() {
    local api_url
    api_url=$(get_api_url) || return 1
    local response
    response=$(curl -sf -X POST "${api_url}/api/v1/time/convert" \
        -H "Content-Type: application/json" \
        -d '{
            "time": "2024-01-15T10:00:00Z",
            "from_timezone": "UTC",
            "to_timezone": "America/New_York"
        }')
    
    # Check if response contains expected fields
    echo "$response" | jq -e '.converted_time' >/dev/null && \
    echo "$response" | jq -e '.offset_minutes' >/dev/null
}

test_duration_calculation() {
    local api_url
    api_url=$(get_api_url) || return 1
    local response
    response=$(curl -sf -X POST "${api_url}/api/v1/time/duration" \
        -H "Content-Type: application/json" \
        -d '{
            "start_time": "2024-01-15T09:00:00Z",
            "end_time": "2024-01-15T17:00:00Z",
            "exclude_weekends": false
        }')
    
    # Check if response contains expected fields
    echo "$response" | jq -e '.total_hours' >/dev/null && \
    echo "$response" | jq -e '.total_days' >/dev/null
}

test_schedule_optimization() {
    local api_url
    api_url=$(get_api_url) || return 1
    local response
    response=$(curl -sf -X POST "${api_url}/api/v1/schedule/optimal" \
        -H "Content-Type: application/json" \
        -d '{
            "participants": ["alice", "bob"],
            "duration_minutes": 60,
            "timezone": "UTC",
            "earliest_date": "2024-01-15",
            "latest_date": "2024-01-22"
        }')
    
    # Check if response contains expected fields
    echo "$response" | jq -e '.optimal_slots' >/dev/null
}

test_conflict_detection() {
    local api_url
    api_url=$(get_api_url) || return 1
    local response
    response=$(curl -sf -X POST "${api_url}/api/v1/schedule/conflicts" \
        -H "Content-Type: application/json" \
        -d '{
            "organizer_id": "test_user",
            "start_time": "2024-01-15T14:00:00Z",
            "end_time": "2024-01-15T15:00:00Z"
        }')
    
    # Check if response contains expected fields
    echo "$response" | jq -e '.has_conflicts' >/dev/null && \
    echo "$response" | jq -e '.conflict_count' >/dev/null
}

test_time_formatting() {
    local api_url
    api_url=$(get_api_url) || return 1
    local response
    response=$(curl -sf -X POST "${api_url}/api/v1/time/format" \
        -H "Content-Type: application/json" \
        -d '{
            "time": "2024-01-15T10:00:00Z",
            "format": "human",
            "timezone": "America/New_York"
        }')
    
    # Check if response contains expected fields
    echo "$response" | jq -e '.formatted' >/dev/null
}

################################################################################
# CLI Tests
################################################################################

test_cli_help() {
    time-tools help >/dev/null 2>&1
}

test_cli_version() {
    time-tools version >/dev/null 2>&1
}

test_cli_status() {
    time-tools status >/dev/null 2>&1
}

test_cli_convert() {
    time-tools convert '2024-01-15T10:00:00Z' UTC 'America/New_York' >/dev/null 2>&1
}

test_cli_duration() {
    time-tools duration '2024-01-15T09:00:00Z' '2024-01-15T17:00:00Z' >/dev/null 2>&1
}

test_cli_now() {
    time-tools now UTC >/dev/null 2>&1
}

test_cli_format() {
    time-tools format '2024-01-15T10:00:00Z' human >/dev/null 2>&1
}

################################################################################
# Build Tests
################################################################################

test_go_build() {
    cd "$ROOT_DIR/api"
    go build -o test-build >/dev/null 2>&1
    local result=$?
    rm -f test-build
    return $result
}

test_go_mod_tidy() {
    cd "$ROOT_DIR/api"
    go mod tidy >/dev/null 2>&1
}

################################################################################
# Main Test Execution
################################################################################

main() {
    log_info "Starting Time Tools Basic Functionality Tests"
    log_info "Root directory: $ROOT_DIR"
    
    # Build tests
    log_info "Running build tests..."
    run_test "Go build check" "test_go_build"
    run_test "Go mod tidy" "test_go_mod_tidy"
    
    # API tests
    log_info "Running API tests..."
    run_test "Health endpoint" "test_health_endpoint"
    run_test "Timezone conversion" "test_timezone_conversion"
    run_test "Duration calculation" "test_duration_calculation"
    run_test "Schedule optimization" "test_schedule_optimization"
    run_test "Conflict detection" "test_conflict_detection"
    run_test "Time formatting" "test_time_formatting"
    
    # CLI tests
    log_info "Running CLI tests..."
    run_test "CLI help command" "test_cli_help"
    run_test "CLI version command" "test_cli_version"
    run_test "CLI status command" "test_cli_status"
    run_test "CLI convert command" "test_cli_convert"
    run_test "CLI duration command" "test_cli_duration"
    run_test "CLI now command" "test_cli_now"
    run_test "CLI format command" "test_cli_format"
    
    # Summary
    echo
    log_info "Test Summary"
    echo "  Total tests: $TESTS_TOTAL"
    echo "  Passed: $TESTS_PASSED"
    echo "  Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log_success "All tests passed!"
        exit 0
    else
        log_error "$TESTS_FAILED tests failed"
        exit 1
    fi
}

# Check if jq is available (required for JSON parsing)
if ! command -v jq >/dev/null 2>&1; then
    log_error "jq is required for JSON parsing but not installed"
    exit 1
fi

# Check if curl is available
if ! command -v curl >/dev/null 2>&1; then
    log_error "curl is required for API testing but not installed"
    exit 1
fi

# Run main function
main "$@"
