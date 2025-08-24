#!/usr/bin/env bash
# Qdrant Health Check Module
# Uses health-framework.sh for tiered health checking

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
QDRANT_HEALTH_DIR="${APP_ROOT}/resources/qdrant/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/health-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"

# Source configuration and core functions
# shellcheck disable=SC1091
source "${QDRANT_HEALTH_DIR}/../config/defaults.sh"
# shellcheck disable=SC1091
source "${QDRANT_HEALTH_DIR}/api-client.sh"
# shellcheck disable=SC1091
source "${QDRANT_HEALTH_DIR}/core.sh"

# Export configuration
qdrant::export_config

#######################################
# Get health configuration for framework
# Returns:
#   JSON configuration for health checks
#######################################
qdrant::get_health_config() {
    cat <<EOF
{
    "service_name": "qdrant",
    "checks": {
        "basic": {
            "container": "${QDRANT_CONTAINER_NAME}",
            "port": ${QDRANT_PORT},
            "endpoint": "${QDRANT_BASE_URL}/"
        },
        "advanced": {
            "api_endpoint": "${QDRANT_BASE_URL}/cluster",
            "expected_response": "status",
            "collections_endpoint": "${QDRANT_BASE_URL}/collections"
        },
        "performance": {
            "response_time_ms": 1000,
            "memory_limit_mb": 2048
        }
    }
}
EOF
}

#######################################
# Perform basic health check
# Returns:
#   0 if healthy, 1 if not
#######################################
qdrant::health::basic() {
    # Check if container is running
    if ! docker::is_running "${QDRANT_CONTAINER_NAME}"; then
        log::error "Container ${QDRANT_CONTAINER_NAME} is not running"
        return 1
    fi
    
    # Check if API responds using centralized health check
    if qdrant::client::health_check; then
        log::success "Basic health check passed"
        return 0
    else
        log::error "Qdrant API not responding"
        return 1
    fi
}

#######################################
# Perform advanced health check
# Returns:
#   0 if healthy, 1 if not
#######################################
qdrant::health::advanced() {
    # Test cluster endpoint
    local cluster_response
    cluster_response=$(qdrant::client::get "/cluster" "cluster endpoint" 2>/dev/null || true)
    local cluster_code=$?
    
    if [[ $cluster_code -ne 0 ]]; then
        log::error "Cluster endpoint not responding"
        return 1
    fi
    
    # Check cluster status
    local status
    status=$(echo "$cluster_response" | jq -r '.result.status // .status // "unknown"' 2>/dev/null)
    
    if [[ "$status" == "unknown" ]]; then
        log::error "Unable to determine cluster status"
        return 1
    fi
    
    # Test collections endpoint
    local collections_response
    collections_response=$(qdrant::client::get_collections 2>/dev/null || true)
    local collections_code=$?
    
    if [[ $collections_code -ne 0 ]]; then
        log::error "Collections endpoint not responding"
        return 1
    fi
    
    # Check if response is valid JSON with expected structure
    if ! echo "$collections_response" | jq -e '.result.collections' >/dev/null 2>&1; then
        if ! echo "$collections_response" | jq -e '.collections' >/dev/null 2>&1; then
            log::error "Collections response has invalid structure"
            return 1
        fi
    fi
    
    log::success "Advanced health check passed"
    return 0
}

#######################################
# Perform comprehensive health check
# Returns:
#   0 if healthy, 1 if not
#######################################
qdrant::health() {
    # First check Docker daemon
    if ! docker::check_daemon; then
        log::error "Docker daemon is not running"
        return 1
    fi
    
    # Use framework for tiered health checking
    local config
    config=$(qdrant::get_health_config)
    
    # Try basic health first
    if qdrant::health::basic; then
        # Try advanced health
        if qdrant::health::advanced; then
            return 0
        else
            # Basic works but advanced fails - partially healthy
            log::warn "Qdrant is running but some features may be unavailable"
            return 0  # Still consider it healthy if basic checks pass
        fi
    fi
    
    return 1
}

#######################################
# Test Qdrant functionality
# Returns:
#   0 if all tests pass, 1 if any fail
#######################################
qdrant::test() {
    log::info "Testing Qdrant functionality..."
    
    local test_passed=true
    
    # Test API connectivity
    if qdrant::client::health_check; then
        log::success "API connectivity test passed"
    else
        log::error "API connectivity test failed"
        test_passed=false
    fi
    
    # Test collection operations
    local test_collection="test_health_check"
    
    # Try to create a test collection
    if qdrant::client::create_collection "$test_collection" 128 "Cosine" >/dev/null 2>&1; then
        log::success "Collection creation test passed"
        
        # Try to get collection info
        if qdrant::client::get_collection_info "$test_collection" >/dev/null 2>&1; then
            log::success "Collection info test passed"
        else
            log::error "Collection info test failed"
            test_passed=false
        fi
        
        # Clean up test collection
        if qdrant::client::delete_collection "$test_collection" >/dev/null 2>&1; then
            log::success "Collection deletion test passed"
        else
            log::error "Collection deletion test failed"
            test_passed=false
        fi
    else
        log::error "Collection creation test failed"
        test_passed=false
    fi
    
    if [[ "$test_passed" == "true" ]]; then
        log::success "All functionality tests passed"
        return 0
    else
        log::error "Some functionality tests failed"
        return 1
    fi
}

#######################################
# Tiered health check for status framework
# Returns: Health tier via stdout (HEALTHY|DEGRADED|UNHEALTHY)
#######################################
qdrant::tiered_health_check() {
    # Check if container is running first
    if ! docker::is_running "${QDRANT_CONTAINER_NAME}"; then
        echo "UNHEALTHY"
        return 0
    fi
    
    # Check basic API connectivity
    if ! qdrant::client::health_check 2>/dev/null; then
        echo "UNHEALTHY"
        return 0
    fi
    
    # Check collections endpoint for advanced health
    local collections_response
    collections_response=$(qdrant::client::get_collections 2>/dev/null || true)
    
    if [[ $? -ne 0 ]] || [[ -z "$collections_response" ]]; then
        # API works but collections endpoint doesn't - degraded
        echo "DEGRADED"
        return 0
    fi
    
    # Check if collections response is valid JSON
    if echo "$collections_response" | jq . >/dev/null 2>&1; then
        # Everything works - healthy
        echo "HEALTHY"
        return 0
    else
        # Collections endpoint responds but with invalid data - degraded
        echo "DEGRADED"
        return 0
    fi
}

#######################################
# Check if Qdrant is healthy (missing function)
# Returns:
#   0 if healthy, 1 if not healthy
#######################################
qdrant::health::is_healthy() {
    local health_status
    health_status=$(qdrant::tiered_health_check 2>/dev/null || echo "FAILED")
    
    if [[ "$health_status" == "HEALTHY" || "$health_status" == "DEGRADED" ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Export health functions
#######################################
export -f qdrant::get_health_config
export -f qdrant::health::basic
export -f qdrant::health::advanced
export -f qdrant::health
export -f qdrant::test
export -f qdrant::tiered_health_check
export -f qdrant::health::is_healthy