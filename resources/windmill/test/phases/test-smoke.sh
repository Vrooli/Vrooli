#!/usr/bin/env bash
################################################################################
# Windmill Smoke Tests - Quick Health Validation
# 
# Must complete in <30 seconds per v2.0 contract
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WINDMILL_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${WINDMILL_DIR}/../.." && pwd)"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
source "${WINDMILL_DIR}/config/defaults.sh"

# Test configuration
readonly TEST_NAME="Windmill Smoke Tests"
readonly MAX_RETRIES=3
readonly RETRY_DELAY=2

# Color codes
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m'

################################################################################
# Test Functions
################################################################################

test_service_installed() {
    log::info "Testing: Windmill installation status..."
    
    if [[ -f "${WINDMILL_COMPOSE_FILE}" ]]; then
        echo -e "${GREEN}✓${NC} Docker Compose file exists"
        return 0
    else
        echo -e "${RED}✗${NC} Docker Compose file not found"
        return 1
    fi
}

test_service_running() {
    log::info "Testing: Windmill service status..."
    
    # Check if any Windmill containers are running
    if docker ps --format "table {{.Names}}" | grep -q "${WINDMILL_PROJECT_NAME}"; then
        echo -e "${GREEN}✓${NC} Windmill containers are running"
        return 0
    else
        # Try to start the service if not running
        echo -e "${YELLOW}⚠${NC} Windmill not running, attempting to start..."
        if "${WINDMILL_DIR}/cli.sh" manage start --wait --timeout 30 &>/dev/null; then
            echo -e "${GREEN}✓${NC} Windmill started successfully"
            return 0
        else
            echo -e "${RED}✗${NC} Failed to start Windmill"
            return 1
        fi
    fi
}

test_health_endpoint() {
    log::info "Testing: Health endpoint responsiveness..."
    
    local retries=0
    while [[ $retries -lt $MAX_RETRIES ]]; do
        if timeout 5 curl -sf "${WINDMILL_BASE_URL}/api/version" &>/dev/null; then
            echo -e "${GREEN}✓${NC} Health endpoint responding"
            return 0
        fi
        
        retries=$((retries + 1))
        if [[ $retries -lt $MAX_RETRIES ]]; then
            echo -e "${YELLOW}⚠${NC} Health check attempt $retries failed, retrying..."
            sleep $RETRY_DELAY
        fi
    done
    
    echo -e "${RED}✗${NC} Health endpoint not responding after $MAX_RETRIES attempts"
    return 1
}

test_api_accessibility() {
    log::info "Testing: API accessibility..."
    
    # Test if API returns version information
    local response
    response=$(timeout 5 curl -sf "${WINDMILL_BASE_URL}/api/version" 2>/dev/null || echo "")
    
    # Windmill CE returns plain text version like "CE v1.528.0"
    if echo "$response" | grep -qE "(CE|EE) v[0-9]+\.[0-9]+\.[0-9]+"; then
        echo -e "${GREEN}✓${NC} API is accessible and returning version: $response"
        return 0
    else
        echo -e "${RED}✗${NC} API not returning expected version format (got: $response)"
        return 1
    fi
}

test_database_connectivity() {
    log::info "Testing: Database connectivity..."
    
    # Check if database container is running (if using internal DB)
    if [[ "${WINDMILL_DB_EXTERNAL}" == "no" ]]; then
        if docker ps --format "table {{.Names}}" | grep -q "${WINDMILL_DB_CONTAINER_NAME}"; then
            echo -e "${GREEN}✓${NC} Database container is running"
            return 0
        else
            echo -e "${RED}✗${NC} Database container not running"
            return 1
        fi
    else
        echo -e "${YELLOW}⚠${NC} External database configured, skipping container check"
        return 0
    fi
}

test_worker_status() {
    log::info "Testing: Worker container status..."
    
    local worker_count
    worker_count=$(docker ps --format "table {{.Names}}" | grep -c "${WINDMILL_WORKER_CONTAINER}" || true)
    
    if [[ $worker_count -gt 0 ]]; then
        echo -e "${GREEN}✓${NC} Found $worker_count worker container(s)"
        return 0
    else
        echo -e "${YELLOW}⚠${NC} No worker containers found (may be starting)"
        return 0  # Don't fail smoke test for workers
    fi
}

test_port_availability() {
    log::info "Testing: Port availability..."
    
    if netstat -tuln 2>/dev/null | grep -q ":${WINDMILL_SERVER_PORT}"; then
        echo -e "${GREEN}✓${NC} Port ${WINDMILL_SERVER_PORT} is in use (expected)"
        return 0
    else
        echo -e "${YELLOW}⚠${NC} Port ${WINDMILL_SERVER_PORT} not bound (service may be starting)"
        return 0  # Don't fail if port not yet bound
    fi
}

################################################################################
# Main Test Execution
################################################################################

main() {
    echo "================================"
    echo "$TEST_NAME"
    echo "================================"
    echo ""
    
    local failed_tests=()
    local exit_code=0
    
    # Run each test
    local tests=(
        "test_service_installed"
        "test_service_running"
        "test_health_endpoint"
        "test_api_accessibility"
        "test_database_connectivity"
        "test_worker_status"
        "test_port_availability"
    )
    
    for test in "${tests[@]}"; do
        if $test; then
            :  # Test passed
        else
            failed_tests+=("$test")
            exit_code=1
        fi
        echo ""
    done
    
    # Summary
    echo "================================"
    if [[ $exit_code -eq 0 ]]; then
        echo -e "${GREEN}✓ All smoke tests passed!${NC}"
    else
        echo -e "${RED}✗ Some tests failed:${NC}"
        for test in "${failed_tests[@]}"; do
            echo -e "  ${RED}- ${test}${NC}"
        done
    fi
    echo "================================"
    
    return $exit_code
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi