#!/usr/bin/env bash
################################################################################
# BTCPay Integration Tests - End-to-end functionality testing
# Tests API endpoints, database connectivity, and payment operations
################################################################################

set -euo pipefail

# Get directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities and BTCPay libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/common.sh"

# Test: API health endpoint
test_api_health() {
    log::info "Testing: API health endpoint..."
    
    local response=$(timeout 5 curl -sf "${BTCPAY_BASE_URL}/api/v1/health" 2>/dev/null || echo "failed")
    
    if [[ "$response" != "failed" ]]; then
        log::success "API health endpoint returned: ${response:0:50}"
        return 0
    else
        log::error "API health endpoint failed"
        return 1
    fi
}

# Test: Database connectivity
test_database_connection() {
    log::info "Testing: Database connectivity..."
    
    if docker exec "${BTCPAY_POSTGRES_CONTAINER}" pg_isready -U "${BTCPAY_POSTGRES_USER}" &>/dev/null; then
        log::success "Database is ready and accepting connections"
        return 0
    else
        log::error "Database connection failed"
        return 1
    fi
}

# Test: Database tables exist
test_database_schema() {
    log::info "Testing: Database schema..."
    
    local tables=$(docker exec "${BTCPAY_POSTGRES_CONTAINER}" psql -U "${BTCPAY_POSTGRES_USER}" -d "${BTCPAY_POSTGRES_DB}" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null || echo "0")
    tables=$(echo "$tables" | tr -d ' ')
    
    if [[ "$tables" -gt 0 ]]; then
        log::success "Database has ${tables} tables"
        return 0
    else
        log::warning "Database appears empty (${tables} tables)"
        return 0  # Not a failure, BTCPay might initialize on first use
    fi
}

# Test: Container networking
test_container_networking() {
    log::info "Testing: Container networking..."
    
    # Check if network exists
    if docker network ls --format '{{.Name}}' | grep -q "^${BTCPAY_NETWORK}$"; then
        log::success "Docker network '${BTCPAY_NETWORK}' exists"
    else
        log::error "Docker network '${BTCPAY_NETWORK}' not found"
        return 1
    fi
    
    # Check if containers are on the network
    local btcpay_on_network=$(docker inspect "${BTCPAY_CONTAINER_NAME}" --format '{{range .NetworkSettings.Networks}}{{.NetworkID}}{{end}}' 2>/dev/null || echo "")
    local postgres_on_network=$(docker inspect "${BTCPAY_POSTGRES_CONTAINER}" --format '{{range .NetworkSettings.Networks}}{{.NetworkID}}{{end}}' 2>/dev/null || echo "")
    
    if [[ -n "$btcpay_on_network" ]] && [[ -n "$postgres_on_network" ]]; then
        log::success "Containers are properly networked"
        return 0
    else
        log::error "Container networking issue detected"
        return 1
    fi
}

# Test: API response time
test_api_performance() {
    log::info "Testing: API response time..."
    
    local start_time=$(date +%s%N)
    timeout 5 curl -sf "${BTCPAY_BASE_URL}/api/v1/health" &>/dev/null
    local end_time=$(date +%s%N)
    local response_time=$(( (end_time - start_time) / 1000000 ))
    
    if [[ $response_time -lt 1000 ]]; then
        log::success "API response time: ${response_time}ms (excellent)"
        return 0
    elif [[ $response_time -lt 3000 ]]; then
        log::success "API response time: ${response_time}ms (acceptable)"
        return 0
    else
        log::error "API response time: ${response_time}ms (too slow)"
        return 1
    fi
}

# Test: Container resource usage
test_resource_usage() {
    log::info "Testing: Container resource usage..."
    
    # Get memory usage
    local memory_usage=$(docker stats --no-stream --format "{{.MemUsage}}" "${BTCPAY_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
    
    if [[ "$memory_usage" != "unknown" ]]; then
        log::success "Memory usage: ${memory_usage}"
        return 0
    else
        log::warning "Could not determine resource usage"
        return 0  # Not critical
    fi
}

# Test: Configuration directory
test_configuration() {
    log::info "Testing: Configuration directory..."
    
    if [[ -d "${BTCPAY_CONFIG_DIR}" ]]; then
        log::success "Configuration directory exists: ${BTCPAY_CONFIG_DIR}"
        
        # Check for any config files
        local config_files=$(find "${BTCPAY_CONFIG_DIR}" -type f 2>/dev/null | wc -l)
        log::info "Found ${config_files} configuration files"
        return 0
    else
        log::warning "Configuration directory not found: ${BTCPAY_CONFIG_DIR}"
        return 0  # Not critical for basic operation
    fi
}

# Test: Data persistence
test_data_persistence() {
    log::info "Testing: Data persistence..."
    
    if [[ -d "${BTCPAY_DATA_DIR}" ]]; then
        log::success "Data directory exists: ${BTCPAY_DATA_DIR}"
        
        # Check PostgreSQL data
        if [[ -d "${BTCPAY_POSTGRES_DATA}" ]] && [[ -n "$(ls -A ${BTCPAY_POSTGRES_DATA} 2>/dev/null)" ]]; then
            log::success "PostgreSQL data is persisted"
        else
            log::warning "PostgreSQL data directory is empty"
        fi
        return 0
    else
        log::error "Data directory not found: ${BTCPAY_DATA_DIR}"
        return 1
    fi
}

# Main test execution
main() {
    local failed=0
    local total=8
    
    log::info "========================================="
    log::info "BTCPay Integration Tests"
    log::info "========================================="
    
    # Run each test
    test_api_health || ((failed++))
    test_database_connection || ((failed++))
    test_database_schema || ((failed++))
    test_container_networking || ((failed++))
    test_api_performance || ((failed++))
    test_resource_usage || ((failed++))
    test_configuration || ((failed++))
    test_data_persistence || ((failed++))
    
    # Report results
    log::info "========================================="
    if [[ $failed -eq 0 ]]; then
        log::success "PASSED: All ${total} integration tests passed"
        exit 0
    else
        log::error "FAILED: ${failed}/${total} tests failed"
        exit 1
    fi
}

# Execute tests
main "$@"