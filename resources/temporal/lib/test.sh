#!/usr/bin/env bash
# Temporal Resource - Test Library Functions

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core.sh"

# Run smoke tests (quick health check)
run_smoke_tests() {
    log_info "Running smoke tests..."
    
    local test_passed=true
    
    # Test 1: Check if Temporal is running
    echo -n "Checking if Temporal server is running... "
    if docker ps | grep -q "$CONTAINER_NAME"; then
        echo -e "${GREEN}PASS${NC}"
    else
        echo -e "${RED}FAIL${NC}"
        test_passed=false
    fi
    
    # Test 2: Check health endpoint
    echo -n "Checking health endpoint... "
    if timeout 5 curl -sf "http://localhost:${TEMPORAL_PORT}/health" >/dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
    else
        echo -e "${RED}FAIL${NC}"
        test_passed=false
    fi
    
    # Test 3: Check gRPC port
    echo -n "Checking gRPC port accessibility... "
    if nc -z localhost "$TEMPORAL_GRPC_PORT" 2>/dev/null; then
        echo -e "${GREEN}PASS${NC}"
    else
        echo -e "${RED}FAIL${NC}"
        test_passed=false
    fi
    
    # Test 4: Check database connection
    echo -n "Checking database connection... "
    if docker ps | grep -q "$DB_CONTAINER_NAME"; then
        echo -e "${GREEN}PASS${NC}"
    else
        echo -e "${RED}FAIL${NC}"
        test_passed=false
    fi
    
    if $test_passed; then
        log_info "All smoke tests passed"
        exit 0
    else
        log_error "Some smoke tests failed"
        exit 1
    fi
}

# Run integration tests
run_integration_tests() {
    log_info "Running integration tests..."
    
    local test_passed=true
    
    # Test 1: Check namespace operations
    echo -n "Testing namespace operations... "
    if docker exec "$CONTAINER_NAME" tctl namespace describe --namespace default >/dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
    else
        echo -e "${RED}FAIL${NC}"
        test_passed=false
    fi
    
    # Test 2: Test workflow listing
    echo -n "Testing workflow listing... "
    if docker exec "$CONTAINER_NAME" tctl workflow list >/dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
    else
        echo -e "${RED}FAIL${NC}"
        test_passed=false
    fi
    
    # Test 3: Test cluster info
    echo -n "Testing cluster information... "
    if docker exec "$CONTAINER_NAME" tctl cluster get-search-attributes >/dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
    else
        echo -e "${RED}FAIL${NC}"
        test_passed=false
    fi
    
    if $test_passed; then
        log_info "All integration tests passed"
        exit 0
    else
        log_error "Some integration tests failed"
        exit 1
    fi
}

# Run unit tests
run_unit_tests() {
    log_info "Running unit tests..."
    
    # Test configuration loading
    echo -n "Testing configuration loading... "
    if [[ -f "${SCRIPT_DIR}/../config/runtime.json" ]]; then
        echo -e "${GREEN}PASS${NC}"
    else
        echo -e "${RED}FAIL${NC}"
        exit 1
    fi
    
    # Test environment variables
    echo -n "Testing environment variables... "
    if [[ -n "$TEMPORAL_PORT" ]] && [[ -n "$TEMPORAL_GRPC_PORT" ]]; then
        echo -e "${GREEN}PASS${NC}"
    else
        echo -e "${RED}FAIL${NC}"
        exit 1
    fi
    
    log_info "All unit tests passed"
    exit 0
}

# Run all tests
run_all_tests() {
    log_info "Running all tests..."
    
    # Run tests in order
    run_unit_tests
    run_smoke_tests
    run_integration_tests
    
    log_info "All test suites completed successfully"
    exit 0
}