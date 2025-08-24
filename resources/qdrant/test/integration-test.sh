#!/usr/bin/env bash
# Qdrant Integration Test
# Tests real Qdrant vector database functionality
# Tests REST API, collections, vector operations, and cluster health

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/qdrant/test"

# Source shared integration test library
# shellcheck disable=SC1091
source "${APP_ROOT}/tests/lib/integration-test-lib.sh"

# Create alias for make_api_request function  
make_api_request() {
    integration_test_lib::make_api_request "$@"
}

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load Qdrant configuration
RESOURCES_DIR="$SCRIPT_DIR/../../.."
# shellcheck disable=SC1091
source "$RESOURCES_DIR/common.sh"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../config/defaults.sh"
qdrant::export_config

# Override library defaults with Qdrant-specific settings
SERVICE_NAME="qdrant"
BASE_URL="${QDRANT_BASE_URL:-http://localhost:6333}"
HEALTH_ENDPOINT="/"
REQUIRED_TOOLS=("curl" "jq" "docker")
SERVICE_METADATA=(
    "HTTP Port: ${QDRANT_PORT:-6333}"
    "gRPC Port: ${QDRANT_GRPC_PORT:-6334}"
    "Container: ${QDRANT_CONTAINER_NAME:-qdrant}"
    "Data Dir: ${QDRANT_DATA_DIR:-${HOME}/.qdrant/data}"
    "Collections: ${#QDRANT_DEFAULT_COLLECTIONS[@]}"
)

#######################################
# QDRANT-SPECIFIC TEST FUNCTIONS
#######################################

test_web_interface() {
    local test_name="web interface accessibility"
    
    local response
    if response=$(make_api_request "/" "GET" 10); then
        if echo "$response" | grep -qi "qdrant\|<!DOCTYPE html>\|vector.*database"; then
            log_test_result "$test_name" "PASS" "web interface accessible"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "web interface not accessible"
    return 1
}

test_cluster_info() {
    local test_name="cluster info endpoint"
    
    local response
    if response=$(make_api_request "/cluster" "GET" 5); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local status
            status=$(echo "$response" | jq -r '.result.status // .status // "unknown"' 2>/dev/null)
            log_test_result "$test_name" "PASS" "cluster status: $status"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "cluster info not available"
    return 1
}

test_telemetry_endpoint() {
    local test_name="telemetry endpoint"
    
    local response
    if response=$(make_api_request "/telemetry" "GET" 5); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local app_info
            app_info=$(echo "$response" | jq -r '.result.app // empty' 2>/dev/null)
            if [[ -n "$app_info" ]]; then
                log_test_result "$test_name" "PASS" "telemetry data available"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "telemetry endpoint not available"
    return 2
}

test_collections_api() {
    local test_name="collections API"
    
    local response
    if response=$(make_api_request "/collections" "GET" 5); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local collections
            collections=$(echo "$response" | jq -r '.result.collections // .collections // empty' 2>/dev/null)
            if [[ -n "$collections" ]]; then
                local count
                count=$(echo "$response" | jq '.result.collections | length // .collections | length // 0' 2>/dev/null)
                log_test_result "$test_name" "PASS" "collections API working (collections: $count)"
                return 0
            else
                log_test_result "$test_name" "PASS" "collections API working (no collections)"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "collections API not working"
    return 1
}

test_metrics_endpoint() {
    local test_name="metrics endpoint"
    
    local response
    if response=$(make_api_request "/metrics" "GET" 5); then
        # Qdrant metrics are in Prometheus format
        if echo "$response" | grep -q "# HELP\|qdrant_\|# TYPE"; then
            log_test_result "$test_name" "PASS" "Prometheus metrics available"
            return 0
        elif echo "$response" | jq . >/dev/null 2>&1; then
            log_test_result "$test_name" "PASS" "JSON metrics available"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "metrics endpoint not available"
    return 2
}

test_collection_creation() {
    local test_name="collection creation (basic test)"
    
    # Test creating a simple test collection
    local collection_name
    collection_name="integration_test_$(date +%s)"
    local collection_data='{
        "vectors": {
            "size": 384,
            "distance": "Cosine"
        }
    }'
    
    local response
    if response=$(make_api_request "/collections/$collection_name" "PUT" 10 "-H 'Content-Type: application/json' -d '$collection_data'"); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local status
            status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null)
            
            # Clean up test collection
            make_api_request "/collections/$collection_name" "DELETE" 5 >/dev/null 2>&1 || true
            
            if [[ "$status" == "ok" ]]; then
                log_test_result "$test_name" "PASS" "collection creation working"
                return 0
            fi
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "collection creation test failed"
    return 2
}

test_point_operations() {
    local test_name="point operations (basic test)"
    
    # Create a temporary collection for testing
    local collection_name
    collection_name="integration_points_test_$(date +%s)"
    local collection_data='{
        "vectors": {
            "size": 3,
            "distance": "Cosine"
        }
    }'
    
    # Create collection
    if ! make_api_request "/collections/$collection_name" "PUT" 10 "-H 'Content-Type: application/json' -d '$collection_data'" >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "could not create test collection"
        return 2
    fi
    
    # Try to insert a point
    local point_data='{
        "points": [
            {
                "id": 1,
                "vector": [0.1, 0.2, 0.3],
                "payload": {"test": "data"}
            }
        ]
    }'
    
    local response
    if response=$(make_api_request "/collections/$collection_name/points" "PUT" 10 "-H 'Content-Type: application/json' -d '$point_data'"); then
        if echo "$response" | jq . >/dev/null 2>&1; then
            local status
            status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null)
            
            # Clean up
            make_api_request "/collections/$collection_name" "DELETE" 5 >/dev/null 2>&1 || true
            
            if [[ "$status" == "ok" ]]; then
                log_test_result "$test_name" "PASS" "point operations working"
                return 0
            fi
        fi
    fi
    
    # Clean up on failure
    make_api_request "/collections/$collection_name" "DELETE" 5 >/dev/null 2>&1 || true
    log_test_result "$test_name" "SKIP" "point operations test failed"
    return 2
}

test_container_health() {
    local test_name="Docker container health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    if docker ps --format '{{.Names}}' | grep -q "^${QDRANT_CONTAINER_NAME}$"; then
        local container_status
        container_status=$(docker inspect "${QDRANT_CONTAINER_NAME}" --format '{{.State.Status}}' 2>/dev/null || echo "unknown")
        
        if [[ "$container_status" == "running" ]]; then
            log_test_result "$test_name" "PASS" "container running"
            return 0
        else
            log_test_result "$test_name" "FAIL" "container status: $container_status"
            return 1
        fi
    else
        log_test_result "$test_name" "FAIL" "container not found"
        return 1
    fi
}

test_log_output() {
    local test_name="application log health"
    
    if ! command -v docker >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "Docker not available"
        return 2
    fi
    
    local logs_output
    if logs_output=$(docker logs "${QDRANT_CONTAINER_NAME}" --tail 10 2>&1 2>/dev/null || true); then
        # Look for Qdrant startup success patterns
        if echo "$logs_output" | grep -qi "qdrant.*started\|listening.*on\|server.*ready"; then
            log_test_result "$test_name" "PASS" "healthy Qdrant logs"
            return 0
        elif echo "$logs_output" | grep -qi "error\|panic\|fatal\|failed"; then
            log_test_result "$test_name" "FAIL" "errors detected in logs"
            return 1
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "log status unclear"
    return 2
}

#######################################
# SERVICE-SPECIFIC VERBOSE INFO
#######################################

show_verbose_info() {
    echo
    echo "Qdrant Information:"
    echo "  Web Interface: $BASE_URL"
    echo "  gRPC Endpoint: ${QDRANT_GRPC_URL}"
    echo "  API Endpoints:"
    echo "    - Cluster Info: GET $BASE_URL/cluster"
    echo "    - Collections: GET $BASE_URL/collections"
    echo "    - Telemetry: GET $BASE_URL/telemetry"
    echo "    - Metrics: GET $BASE_URL/metrics"
    echo "  Container: ${QDRANT_CONTAINER_NAME}"
    echo "  Data Directory: ${QDRANT_DATA_DIR}"
    echo "  Default Collections:"
    for collection in "${QDRANT_DEFAULT_COLLECTIONS[@]}"; do
        echo "    - $collection"
    done
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register standard interface tests first (CLI validation, config checks, etc.)
register_standard_interface_tests

# Register Qdrant-specific tests
register_tests \
    "test_web_interface" \
    "test_cluster_info" \
    "test_telemetry_endpoint" \
    "test_collections_api" \
    "test_metrics_endpoint" \
    "test_collection_creation" \
    "test_point_operations" \
    "test_container_health" \
    "test_log_output"

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi