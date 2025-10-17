#!/usr/bin/env bash
# Integration Tests Phase
# Tests end-to-end workflows and component interactions

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
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_error() { echo -e "${RED}[FAIL]${NC} $1"; }

TESTS_PASSED=0
TESTS_FAILED=0

run_test() {
    local test_name="$1"
    shift

    log_info "Running: $test_name"
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

# Test: Full graph lifecycle (create -> get -> update -> delete)
test_graph_lifecycle() {
    local test_user="test-user-$(date +%s)"

    # Create graph
    local response=$(curl -sf -X POST "$API_URL/api/v1/graphs" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $test_user" \
        -d '{"name":"Integration Test Graph","type":"mind-maps","description":"Test graph"}' 2>&1)

    local graph_id=$(echo "$response" | jq -r '.id' 2>/dev/null)
    if [[ -z "$graph_id" || "$graph_id" == "null" ]]; then
        echo "Failed to create graph: $response"
        return 1
    fi

    # Get graph
    if ! curl -sf "$API_URL/api/v1/graphs/$graph_id" -H "X-User-ID: $test_user" | jq -e '.id' &>/dev/null; then
        echo "Failed to retrieve graph"
        return 1
    fi

    # Update graph
    if ! curl -sf -X PUT "$API_URL/api/v1/graphs/$graph_id" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $test_user" \
        -d '{"name":"Updated Test Graph"}' | jq -e '.success' &>/dev/null; then
        echo "Failed to update graph"
        return 1
    fi

    # Delete graph
    if ! curl -sf -X DELETE "$API_URL/api/v1/graphs/$graph_id" -H "X-User-ID: $test_user" | jq -e '.success' &>/dev/null; then
        echo "Failed to delete graph"
        return 1
    fi

    return 0
}

# Test: Graph validation
test_graph_validation() {
    local test_user="test-user-$(date +%s)"

    # Create a test graph with valid BPMN structure
    local response=$(curl -sf -X POST "$API_URL/api/v1/graphs" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $test_user" \
        -d '{"name":"Validation Test","type":"bpmn","data":{"nodes":[{"id":"start","type":"start"}]}}' 2>&1)

    local graph_id=$(echo "$response" | jq -r '.id' 2>/dev/null)
    if [[ -z "$graph_id" || "$graph_id" == "null" ]]; then
        return 1
    fi

    # Validate graph
    local validation=$(curl -sf -X POST "$API_URL/api/v1/graphs/$graph_id/validate" -H "X-User-ID: $test_user")
    if ! echo "$validation" | jq -e 'has("valid")' &>/dev/null; then
        curl -sf -X DELETE "$API_URL/api/v1/graphs/$graph_id" -H "X-User-ID: $test_user" &>/dev/null
        return 1
    fi

    # Cleanup
    curl -sf -X DELETE "$API_URL/api/v1/graphs/$graph_id" -H "X-User-ID: $test_user" &>/dev/null
    return 0
}

# Test: Graph conversion
test_graph_conversion() {
    local test_user="test-user-$(date +%s)"

    # Create source graph with valid mind-map structure (requires root node)
    local response=$(curl -sf -X POST "$API_URL/api/v1/graphs" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $test_user" \
        -d '{"name":"Conversion Source","type":"mind-maps","data":{"root":{"id":"root","text":"Root"}}}' 2>&1)

    local source_id=$(echo "$response" | jq -r '.id' 2>/dev/null)
    if [[ -z "$source_id" || "$source_id" == "null" ]]; then
        return 1
    fi

    # Convert to mermaid
    local conversion=$(curl -sf -X POST "$API_URL/api/v1/graphs/$source_id/convert" \
        -H "Content-Type: application/json" \
        -H "X-User-ID: $test_user" \
        -d '{"target_format":"mermaid"}' 2>&1)

    local target_id=$(echo "$conversion" | jq -r '.converted_graph_id' 2>/dev/null)

    # Cleanup
    curl -sf -X DELETE "$API_URL/api/v1/graphs/$source_id" -H "X-User-ID: $test_user" &>/dev/null
    if [[ -n "$target_id" && "$target_id" != "null" ]]; then
        curl -sf -X DELETE "$API_URL/api/v1/graphs/$target_id" -H "X-User-ID: $test_user" &>/dev/null
    fi

    if [[ -z "$target_id" || "$target_id" == "null" ]]; then
        return 1
    fi

    return 0
}

# Test: Plugin listing
test_plugin_listing() {
    local plugins=$(curl -sf "$API_URL/api/v1/plugins")

    if ! echo "$plugins" | jq -e '.data | length > 0' &>/dev/null; then
        echo "No plugins found"
        return 1
    fi

    if ! echo "$plugins" | jq -e '.data[] | select(.id == "mind-maps")' &>/dev/null; then
        echo "mind-maps plugin not found"
        return 1
    fi

    return 0
}

# Test: Conversion matrix
test_conversion_matrix() {
    local conversions=$(curl -sf "$API_URL/api/v1/conversions")

    if ! echo "$conversions" | jq -e 'has("conversions")' &>/dev/null; then
        echo "Conversion matrix not found"
        return 1
    fi

    if ! echo "$conversions" | jq -e '.total_paths > 0' &>/dev/null; then
        echo "No conversion paths available"
        return 1
    fi

    return 0
}

# Main execution
main() {
    log_info "=== Integration Tests Phase ==="
    echo ""

    # Check if API is running
    if ! curl -sf "$API_URL/health" &>/dev/null; then
        log_error "API is not running at $API_URL"
        echo "Please start the scenario first: make run"
        return 1
    fi

    run_test "Plugin Listing" test_plugin_listing || true
    run_test "Conversion Matrix" test_conversion_matrix || true
    run_test "Graph Lifecycle (CRUD)" test_graph_lifecycle || true
    run_test "Graph Validation" test_graph_validation || true
    run_test "Graph Conversion" test_graph_conversion || true

    echo ""
    echo "Integration Tests: Passed=$TESTS_PASSED Failed=$TESTS_FAILED"

    if [[ $TESTS_FAILED -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

main "$@"
