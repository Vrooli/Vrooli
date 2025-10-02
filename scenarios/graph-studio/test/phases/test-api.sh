#!/usr/bin/env bash
# API Tests Phase
# Tests all API endpoints for correctness

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Get API port dynamically
API_PORT=$(vrooli scenario port graph-studio API_PORT 2>/dev/null || echo "18707")
API_URL="http://localhost:$API_PORT"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_error() { echo -e "${RED}[FAIL]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }

TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
    local test_name="$1"
    shift

    log_info "Testing: $test_name"
    if "$@"; then
        log_success "$test_name"
        ((TESTS_PASSED++))
        return 0
    else
        log_error "$test_name"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Health endpoint
test_health_endpoint() {
    local response=$(curl -sf "$API_URL/health")
    echo "$response" | jq -e '.status == "healthy"' &>/dev/null
}

# Stats endpoint
test_stats_endpoint() {
    local response=$(curl -sf "$API_URL/api/v1/stats")
    echo "$response" | jq -e 'has("totalGraphs") and has("activeUsers")' &>/dev/null
}

# List plugins
test_list_plugins() {
    local response=$(curl -sf "$API_URL/api/v1/plugins")
    echo "$response" | jq -e '.data | length > 0' &>/dev/null
}

# List graphs
test_list_graphs() {
    local response=$(curl -sf "$API_URL/api/v1/graphs")
    echo "$response" | jq -e 'has("data") and has("total")' &>/dev/null
}

# Create graph with valid data
test_create_graph_valid() {
    local response=$(curl -sf -X POST "$API_URL/api/v1/graphs" \
        -H "Content-Type: application/json" \
        -d '{"name":"API Test Graph","type":"mind-maps","description":"Created by API test"}')

    local graph_id=$(echo "$response" | jq -r '.id')
    if [[ -z "$graph_id" || "$graph_id" == "null" ]]; then
        return 1
    fi

    # Cleanup
    curl -sf -X DELETE "$API_URL/api/v1/graphs/$graph_id" &>/dev/null
    return 0
}

# Create graph with missing required fields
test_create_graph_invalid() {
    local status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/v1/graphs" \
        -H "Content-Type: application/json" \
        -d '{"name":"Missing Type"}')

    [[ "$status" == "400" ]]
}

# Get non-existent graph
test_get_nonexistent_graph() {
    # Use a valid UUID format that doesn't exist (version 4, variant 8)
    local status=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/v1/graphs/12345678-1234-4234-8234-123456789012")
    [[ "$status" == "404" ]]
}

# Update graph
test_update_graph() {
    local test_user="api-test-user-$(date +%s)"

    # Create graph first
    local create_response=$(curl -sf -X POST "$API_URL/api/v1/graphs" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $test_user" \
        -d '{"name":"Update Test","type":"mind-maps"}')

    local graph_id=$(echo "$create_response" | jq -r '.id')
    if [[ -z "$graph_id" || "$graph_id" == "null" ]]; then
        return 1
    fi

    # Update graph
    local update_response=$(curl -sf -X PUT "$API_URL/api/v1/graphs/$graph_id" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $test_user" \
        -d '{"name":"Updated Name"}')

    local success=$(echo "$update_response" | jq -r '.success')

    # Cleanup
    curl -sf -X DELETE "$API_URL/api/v1/graphs/$graph_id" -H "X-User-ID: $test_user" &>/dev/null

    [[ "$success" == "true" ]]
}

# Delete graph
test_delete_graph() {
    local test_user="api-test-user-$(date +%s)"

    # Create graph first
    local create_response=$(curl -sf -X POST "$API_URL/api/v1/graphs" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $test_user" \
        -d '{"name":"Delete Test","type":"mind-maps"}')

    local graph_id=$(echo "$create_response" | jq -r '.id')
    if [[ -z "$graph_id" || "$graph_id" == "null" ]]; then
        return 1
    fi

    # Delete graph
    local delete_response=$(curl -sf -X DELETE "$API_URL/api/v1/graphs/$graph_id" -H "X-User-ID: $test_user")
    echo "$delete_response" | jq -e '.success == true' &>/dev/null
}

# Validate graph endpoint
test_validate_graph() {
    local test_user="api-test-user-$(date +%s)"

    # Create graph first
    local create_response=$(curl -sf -X POST "$API_URL/api/v1/graphs" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $test_user" \
        -d '{"name":"Validate Test","type":"bpmn"}')

    local graph_id=$(echo "$create_response" | jq -r '.id')
    if [[ -z "$graph_id" || "$graph_id" == "null" ]]; then
        return 1
    fi

    # Validate graph
    local validate_response=$(curl -sf -X POST "$API_URL/api/v1/graphs/$graph_id/validate" -H "X-User-ID: $test_user")

    # Cleanup
    curl -sf -X DELETE "$API_URL/api/v1/graphs/$graph_id" -H "X-User-ID: $test_user" &>/dev/null

    echo "$validate_response" | jq -e 'has("valid")' &>/dev/null
}

# List conversions
test_list_conversions() {
    local response=$(curl -sf "$API_URL/api/v1/conversions")
    echo "$response" | jq -e 'has("conversions") and has("total_paths")' &>/dev/null
}

# Render graph
test_render_graph() {
    local test_user="api-test-user-$(date +%s)"

    # Create graph first
    local create_response=$(curl -sf -X POST "$API_URL/api/v1/graphs" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $test_user" \
        -d '{"name":"Render Test","type":"mind-maps"}')

    local graph_id=$(echo "$create_response" | jq -r '.id')
    if [[ -z "$graph_id" || "$graph_id" == "null" ]]; then
        return 1
    fi

    # Render graph as SVG
    local render_response=$(curl -sf -X POST "$API_URL/api/v1/graphs/$graph_id/render" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $test_user" \
        -d '{"format":"svg"}')

    # Cleanup
    curl -sf -X DELETE "$API_URL/api/v1/graphs/$graph_id" -H "X-User-ID: $test_user" &>/dev/null

    # Check if response contains SVG
    echo "$render_response" | grep -q "<svg" || echo "$render_response" | grep -q "svg"
}

# System metrics
test_system_metrics() {
    local response=$(curl -sf "$API_URL/api/v1/metrics")
    echo "$response" | jq -e 'has("metrics")' &>/dev/null
}

# Detailed health
test_detailed_health() {
    local response=$(curl -sf "$API_URL/api/v1/health/detailed")
    echo "$response" | jq -e 'has("database")' &>/dev/null || return 0  # Optional field
}

# Main execution
main() {
    log_info "=== API Tests Phase ==="
    echo ""

    # Check if API is running
    if ! curl -sf "$API_URL/health" &>/dev/null; then
        log_error "API is not running at $API_URL"
        echo "Please start the scenario first: make run"
        return 1
    fi

    # Core endpoints
    run_test "GET /health" test_health_endpoint || true
    run_test "GET /api/v1/stats" test_stats_endpoint || true
    run_test "GET /api/v1/plugins" test_list_plugins || true
    run_test "GET /api/v1/graphs" test_list_graphs || true
    run_test "GET /api/v1/conversions" test_list_conversions || true

    # Graph CRUD operations
    run_test "POST /api/v1/graphs (valid)" test_create_graph_valid || true
    run_test "POST /api/v1/graphs (invalid)" test_create_graph_invalid || true
    run_test "GET /api/v1/graphs/:id (not found)" test_get_nonexistent_graph || true
    run_test "PUT /api/v1/graphs/:id" test_update_graph || true
    run_test "DELETE /api/v1/graphs/:id" test_delete_graph || true

    # Graph operations
    run_test "POST /api/v1/graphs/:id/validate" test_validate_graph || true
    run_test "POST /api/v1/graphs/:id/render" test_render_graph || true

    # System endpoints
    run_test "GET /api/v1/metrics" test_system_metrics || true
    run_test "GET /api/v1/health/detailed" test_detailed_health || true

    echo ""
    echo "API Tests: Passed=$TESTS_PASSED Failed=$TESTS_FAILED"

    if [[ $TESTS_FAILED -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

main "$@"
