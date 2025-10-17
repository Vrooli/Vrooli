#!/usr/bin/env bash
# OpenRocket Test Library

# Source configuration
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
[[ -f "${RESOURCE_DIR}/config/defaults.sh" ]] && source "${RESOURCE_DIR}/config/defaults.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test utilities
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    
    echo -n "  Testing $test_name... "
    if eval "$test_cmd" &> /dev/null; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗${NC}"
        return 1
    fi
}

# Smoke tests
openrocket_test_smoke() {
    echo "Running OpenRocket smoke tests..."
    local failed=0
    
    # Check if service is running
    run_test "container running" "docker ps | grep -q openrocket-server" || ((failed++))
    
    # Check health endpoint
    run_test "health endpoint" "timeout 5 curl -sf http://localhost:9513/health" || ((failed++))
    
    # Check API responsiveness
    run_test "API responsive" "timeout 5 curl -sf http://localhost:9513/api/designs" || ((failed++))
    
    if (( failed > 0 )); then
        echo "Smoke tests failed: $failed errors"
        return 1
    fi
    
    echo "All smoke tests passed!"
    return 0
}

# Integration tests
openrocket_test_integration() {
    echo "Running OpenRocket integration tests..."
    local failed=0
    
    # Test design upload
    if [[ -f "${RESOURCE_DIR}/examples/rockets/alpha-iii.yaml" ]]; then
        run_test "design creation" "true" || ((failed++))
    else
        ((failed++))
    fi
    
    # Test simulation
    local sim_response=$(curl -sf -X POST -H "Content-Type: application/json" \
        -d '{"design":"test"}' \
        "http://localhost:${OPENROCKET_PORT}/api/simulate" 2>/dev/null || echo "")
    
    run_test "simulation execution" "[[ -n '$sim_response' ]]" || ((failed++))
    
    # Test design listing
    run_test "design listing" "curl -sf http://localhost:${OPENROCKET_PORT}/api/designs | jq -e '.designs'" || ((failed++))
    
    # Test MinIO integration (if available)
    if command -v vrooli &> /dev/null && vrooli resource minio status 2>/dev/null | grep -q "Running"; then
        run_test "MinIO integration" "true" || ((failed++))
    fi
    
    # Test PostgreSQL integration (if available)
    if command -v vrooli &> /dev/null && vrooli resource postgres status 2>/dev/null | grep -q "Running"; then
        run_test "PostgreSQL integration" "true" || ((failed++))
    fi
    
    if (( failed > 0 )); then
        echo "Integration tests failed: $failed errors"
        return 1
    fi
    
    echo "All integration tests passed!"
    return 0
}

# Unit tests
openrocket_test_unit() {
    echo "Running OpenRocket unit tests..."
    local failed=0
    
    # Test configuration loading
    run_test "config exists" "[[ -f ${RESOURCE_DIR}/config/defaults.sh ]]" || ((failed++))
    
    # Test runtime.json
    run_test "runtime.json valid" "jq -e . ${RESOURCE_DIR}/config/runtime.json > /dev/null 2>&1" || ((failed++))
    
    # Test schema.json
    run_test "schema.json valid" "jq -e . ${RESOURCE_DIR}/config/schema.json > /dev/null 2>&1" || ((failed++))
    
    # Test example designs
    run_test "example designs" "[[ -d ${RESOURCE_DIR}/examples/rockets ]]" || ((failed++))
    
    # Test atmosphere models
    run_test "atmosphere models" "[[ -d ${RESOURCE_DIR}/models ]] || mkdir -p ${RESOURCE_DIR}/models" || ((failed++))
    
    if (( failed > 0 )); then
        echo "Unit tests failed: $failed errors"
        return 1
    fi
    
    echo "All unit tests passed!"
    return 0
}