#!/usr/bin/env bash
# [REQ:CLI-TICK-001] [REQ:CLI-TICK-002] [REQ:CLI-STATUS-001] [REQ:CLI-STATUS-002]
# [REQ:UI-HEALTH-001] [REQ:UI-HEALTH-002] [REQ:PERSIST-QUERY-001] [REQ:PERSIST-QUERY-002]
# API Integration tests for vrooli-autoheal endpoints
set -euo pipefail

SCENARIO_NAME="vrooli-autoheal"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Output directory for test artifacts
ARTIFACT_DIR="${PROJECT_DIR}/test/artifacts/integration"
mkdir -p "$ARTIFACT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# Test result helpers
pass() {
    TESTS_RUN=$((TESTS_RUN + 1))
    TESTS_PASSED=$((TESTS_PASSED + 1))
    echo -e "${GREEN}✓${NC} $1"
}

fail() {
    TESTS_RUN=$((TESTS_RUN + 1))
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo -e "${RED}✗${NC} $1"
    if [[ -n "${2:-}" ]]; then
        echo "  Details: $2"
    fi
}

# Get API port dynamically
get_api_port() {
    local port
    port=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || echo "")
    if [[ -z "$port" ]]; then
        # Fallback to checking if scenario is running and getting port from status
        port=$(vrooli scenario status "$SCENARIO_NAME" 2>/dev/null | grep "API_PORT" | awk '{print $2}' || echo "")
    fi
    echo "$port"
}

# Wait for API to be ready
wait_for_api() {
    local port="$1"
    local max_attempts=30
    local attempt=0

    while [[ $attempt -lt $max_attempts ]]; do
        if curl -s "http://localhost:${port}/health" &>/dev/null; then
            return 0
        fi
        sleep 1
        ((attempt++))
    done
    return 1
}

# Test: Health endpoint returns valid response
# [REQ:CLI-STATUS-001]
test_health_endpoint() {
    local port="$1"
    local response
    local status

    response=$(curl -s "http://localhost:${port}/health")

    # Check status field exists and is "healthy"
    status=$(echo "$response" | jq -r '.status // empty')
    if [[ "$status" == "healthy" ]]; then
        pass "Health endpoint returns healthy status"
    else
        fail "Health endpoint status check" "Expected 'healthy', got '$status'"
    fi

    # Check service name
    local service
    service=$(echo "$response" | jq -r '.service // empty')
    if [[ -n "$service" ]]; then
        pass "Health endpoint includes service name"
    else
        fail "Health endpoint service name missing"
    fi

    # Check version
    local version
    version=$(echo "$response" | jq -r '.version // empty')
    if [[ -n "$version" ]]; then
        pass "Health endpoint includes version"
    else
        fail "Health endpoint version missing"
    fi

    # Check readiness
    local readiness
    readiness=$(echo "$response" | jq -r '.readiness // empty')
    if [[ "$readiness" == "true" ]]; then
        pass "Health endpoint reports readiness"
    else
        fail "Health endpoint readiness check" "Expected 'true', got '$readiness'"
    fi

    # Check dependencies
    local db_status
    db_status=$(echo "$response" | jq -r '.dependencies.database // empty')
    if [[ "$db_status" == "connected" ]]; then
        pass "Health endpoint shows database connected"
    else
        fail "Health endpoint database status" "Expected 'connected', got '$db_status'"
    fi

    # Save response artifact
    echo "$response" > "$ARTIFACT_DIR/health-response.json"
}

# Test: Platform endpoint returns valid capabilities
# [REQ:PLAT-DETECT-001] [REQ:PLAT-DETECT-002]
test_platform_endpoint() {
    local port="$1"
    local response

    response=$(curl -s "http://localhost:${port}/api/v1/platform")

    # Check platform field
    local platform
    platform=$(echo "$response" | jq -r '.platform // empty')
    case "$platform" in
        linux|windows|macos|other)
            pass "Platform endpoint returns valid platform type: $platform"
            ;;
        *)
            fail "Platform endpoint platform type" "Invalid value: $platform"
            ;;
    esac

    # Check capability fields exist
    local caps=("supportsRdp" "supportsSystemd" "supportsLaunchd" "hasDocker" "isWsl" "supportsCloudflared" "isHeadlessServer")
    for cap in "${caps[@]}"; do
        local value
        value=$(echo "$response" | jq -r "if has(\"$cap\") then \"\(.$cap)\" else \"missing\" end")
        if [[ "$value" == "true" || "$value" == "false" ]]; then
            pass "Platform endpoint includes capability: $cap"
        else
            fail "Platform endpoint capability $cap" "Expected boolean, got '$value'"
        fi
    done

    # Save response artifact
    echo "$response" > "$ARTIFACT_DIR/platform-response.json"
}

# Test: Status endpoint returns health summary
# [REQ:CLI-STATUS-001] [REQ:CLI-STATUS-002] [REQ:PERSIST-QUERY-002]
test_status_endpoint() {
    local port="$1"
    local response

    response=$(curl -s "http://localhost:${port}/api/v1/status")

    # Check status field
    local status
    status=$(echo "$response" | jq -r '.status // empty')
    case "$status" in
        ok|warning|critical)
            pass "Status endpoint returns valid status: $status"
            ;;
        *)
            fail "Status endpoint status field" "Invalid value: $status"
            ;;
    esac

    # Check summary fields
    local total ok warning critical
    total=$(echo "$response" | jq -r '.summary.total // -1')
    ok=$(echo "$response" | jq -r '.summary.ok // -1')
    warning=$(echo "$response" | jq -r '.summary.warning // -1')
    critical=$(echo "$response" | jq -r '.summary.critical // -1')

    if [[ "$total" -ge 0 ]]; then
        pass "Status endpoint includes total count: $total"
    else
        fail "Status endpoint total count" "Missing or invalid"
    fi

    if [[ "$ok" -ge 0 ]]; then
        pass "Status endpoint includes ok count: $ok"
    else
        fail "Status endpoint ok count" "Missing or invalid"
    fi

    # Check counts add up (if we have checks)
    if [[ "$total" -gt 0 ]]; then
        local sum=$((ok + warning + critical))
        if [[ "$sum" -eq "$total" ]]; then
            pass "Status endpoint counts are consistent ($ok + $warning + $critical = $total)"
        else
            fail "Status endpoint count consistency" "$ok + $warning + $critical != $total"
        fi
    fi

    # Check platform info is included
    local platform
    platform=$(echo "$response" | jq -r '.platform.platform // empty')
    if [[ -n "$platform" ]]; then
        pass "Status endpoint includes platform info"
    else
        fail "Status endpoint platform info missing"
    fi

    # Check timestamp
    local timestamp
    timestamp=$(echo "$response" | jq -r '.timestamp // empty')
    if [[ -n "$timestamp" ]]; then
        pass "Status endpoint includes timestamp"
    else
        fail "Status endpoint timestamp missing"
    fi

    # Save response artifact
    echo "$response" > "$ARTIFACT_DIR/status-response.json"
}

# Test: Tick endpoint executes health checks
# [REQ:CLI-TICK-001] [REQ:CLI-TICK-002]
test_tick_endpoint() {
    local port="$1"
    local response

    response=$(curl -s -X POST "http://localhost:${port}/api/v1/tick?force=true")

    # Check success field
    local success
    success=$(echo "$response" | jq -r '.success // false')
    if [[ "$success" == "true" ]]; then
        pass "Tick endpoint returns success"
    else
        fail "Tick endpoint success field" "Expected 'true', got '$success'"
    fi

    # Check results array exists
    local results_count
    results_count=$(echo "$response" | jq -r '.results | length // 0')
    if [[ "$results_count" -gt 0 ]]; then
        pass "Tick endpoint returns $results_count check results"
    else
        fail "Tick endpoint results" "Expected non-empty results array"
    fi

    # Validate first result structure
    local first_check_id first_status first_message
    first_check_id=$(echo "$response" | jq -r '.results[0].checkId // empty')
    first_status=$(echo "$response" | jq -r '.results[0].status // empty')
    first_message=$(echo "$response" | jq -r '.results[0].message // empty')

    if [[ -n "$first_check_id" ]]; then
        pass "Tick result includes checkId: $first_check_id"
    else
        fail "Tick result checkId missing"
    fi

    case "$first_status" in
        ok|warning|critical)
            pass "Tick result has valid status: $first_status"
            ;;
        *)
            fail "Tick result status" "Invalid value: $first_status"
            ;;
    esac

    if [[ -n "$first_message" ]]; then
        pass "Tick result includes message"
    else
        fail "Tick result message missing"
    fi

    # Check summary is included
    local summary_total
    summary_total=$(echo "$response" | jq -r '.summary.total // -1')
    if [[ "$summary_total" -ge 0 ]]; then
        pass "Tick endpoint includes summary"
    else
        fail "Tick endpoint summary missing"
    fi

    # Save response artifact
    echo "$response" > "$ARTIFACT_DIR/tick-response.json"
}

# Test: Checks endpoint lists registered checks
# [REQ:HEALTH-REGISTRY-001]
test_checks_endpoint() {
    local port="$1"
    local response

    response=$(curl -s "http://localhost:${port}/api/v1/checks")

    # Check it's an array
    local is_array
    is_array=$(echo "$response" | jq -r 'if type == "array" then "yes" else "no" end')
    if [[ "$is_array" == "yes" ]]; then
        pass "Checks endpoint returns array"
    else
        fail "Checks endpoint format" "Expected array, got $(echo "$response" | jq -r 'type')"
        return
    fi

    # Check we have some checks registered
    local count
    count=$(echo "$response" | jq -r 'length')
    if [[ "$count" -gt 0 ]]; then
        pass "Checks endpoint returns $count registered checks"
    else
        fail "Checks endpoint count" "Expected at least 1 check"
    fi

    # Validate first check structure
    local first_id first_desc first_interval
    first_id=$(echo "$response" | jq -r '.[0].id // empty')
    first_desc=$(echo "$response" | jq -r '.[0].description // empty')
    first_interval=$(echo "$response" | jq -r '.[0].intervalSeconds // -1')

    if [[ -n "$first_id" ]]; then
        pass "Check info includes id: $first_id"
    else
        fail "Check info id missing"
    fi

    if [[ -n "$first_desc" ]]; then
        pass "Check info includes description"
    else
        fail "Check info description missing"
    fi

    if [[ "$first_interval" -gt 0 ]]; then
        pass "Check info includes intervalSeconds: ${first_interval}s"
    else
        fail "Check info intervalSeconds" "Expected positive number"
    fi

    # Save response artifact
    echo "$response" > "$ARTIFACT_DIR/checks-response.json"
}

# Test: Check result endpoint returns specific check
# [REQ:HEALTH-REGISTRY-004]
test_check_result_endpoint() {
    local port="$1"

    # First run a tick to ensure we have results
    curl -s -X POST "http://localhost:${port}/api/v1/tick?force=true" > /dev/null

    # Get the first check ID from the checks list
    local check_id
    check_id=$(curl -s "http://localhost:${port}/api/v1/checks" | jq -r '.[0].id // empty')

    if [[ -z "$check_id" ]]; then
        fail "Check result endpoint" "No checks available"
        return
    fi

    local response
    response=$(curl -s "http://localhost:${port}/api/v1/checks/${check_id}")

    # Check status code by checking if we got valid JSON
    local status
    status=$(echo "$response" | jq -r '.status // empty')

    if [[ -n "$status" ]]; then
        pass "Check result endpoint returns result for $check_id"
    else
        fail "Check result endpoint" "Failed to get result for $check_id"
        return
    fi

    # Validate result structure
    local result_check_id result_message
    result_check_id=$(echo "$response" | jq -r '.checkId // empty')
    result_message=$(echo "$response" | jq -r '.message // empty')

    if [[ "$result_check_id" == "$check_id" ]]; then
        pass "Check result has correct checkId"
    else
        fail "Check result checkId mismatch" "Expected $check_id, got $result_check_id"
    fi

    if [[ -n "$result_message" ]]; then
        pass "Check result includes message: $result_message"
    else
        fail "Check result message missing"
    fi

    # Test non-existent check returns 404
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${port}/api/v1/checks/non-existent-check")

    if [[ "$http_code" == "404" ]]; then
        pass "Check result endpoint returns 404 for non-existent check"
    else
        fail "Check result endpoint non-existent check" "Expected 404, got $http_code"
    fi

    # Save response artifact
    echo "$response" > "$ARTIFACT_DIR/check-result-response.json"
}

# Test: UI health check
# [REQ:UI-HEALTH-001]
test_ui_health() {
    local ui_port
    ui_port=$(vrooli scenario port "$SCENARIO_NAME" UI_PORT 2>/dev/null || echo "")

    if [[ -z "$ui_port" ]]; then
        log_warn "UI port not available, skipping UI tests"
        return
    fi

    local http_code

    http_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${ui_port}/health")

    if [[ "$http_code" == "200" ]]; then
        pass "UI health endpoint responds with 200"
    else
        fail "UI health endpoint" "Expected 200, got $http_code"
    fi

    # Check UI serves content
    http_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${ui_port}/")

    if [[ "$http_code" == "200" ]]; then
        pass "UI serves content on root path"
    else
        fail "UI root path" "Expected 200, got $http_code"
    fi
}

# Main test runner
main() {
    log_info "Starting API integration tests for $SCENARIO_NAME"

    # Get API port
    local api_port
    api_port=$(get_api_port)

    if [[ -z "$api_port" ]]; then
        log_error "Could not determine API port. Is the scenario running?"
        log_info "Start it with: vrooli scenario start $SCENARIO_NAME"
        exit 1
    fi

    log_info "Using API port: $api_port"

    # Wait for API to be ready
    log_info "Waiting for API to be ready..."
    if ! wait_for_api "$api_port"; then
        log_error "API did not become ready in time"
        exit 1
    fi
    log_info "API is ready"

    echo ""
    echo "=== Health Endpoint Tests ==="
    test_health_endpoint "$api_port"

    echo ""
    echo "=== Platform Endpoint Tests ==="
    test_platform_endpoint "$api_port"

    echo ""
    echo "=== Status Endpoint Tests ==="
    test_status_endpoint "$api_port"

    echo ""
    echo "=== Tick Endpoint Tests ==="
    test_tick_endpoint "$api_port"

    echo ""
    echo "=== Checks Endpoint Tests ==="
    test_checks_endpoint "$api_port"

    echo ""
    echo "=== Check Result Endpoint Tests ==="
    test_check_result_endpoint "$api_port"

    echo ""
    echo "=== UI Tests ==="
    test_ui_health

    # Summary
    echo ""
    echo "========================================"
    echo "API Integration Test Summary"
    echo "========================================"
    echo "Tests run:    $TESTS_RUN"
    echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
    echo "Artifacts:    $ARTIFACT_DIR/"
    echo "========================================"

    # Exit with appropriate code
    if [[ "$TESTS_FAILED" -gt 0 ]]; then
        exit 1
    fi
    exit 0
}

main "$@"
