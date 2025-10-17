#!/usr/bin/env bash
# Temporal Resource - Smoke Tests
# Quick validation that service is running and responsive

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source libraries
source "${RESOURCE_DIR}/lib/core.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "================================"
echo "Temporal Resource - Smoke Tests"
echo "================================"
echo ""

TESTS_PASSED=0
TESTS_FAILED=0

# Test 1: Service is running
echo -n "1. Checking if Temporal server is running... "
if docker ps 2>/dev/null | grep -q "$CONTAINER_NAME"; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Container not running"
    ((TESTS_FAILED++))
fi

# Test 2: Health endpoint responds
echo -n "2. Checking health endpoint (http://localhost:${TEMPORAL_PORT}/health)... "
if timeout 5 curl -sf "http://localhost:${TEMPORAL_PORT}/health" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Health check failed"
    ((TESTS_FAILED++))
fi

# Test 3: gRPC port is accessible
echo -n "3. Checking gRPC port ${TEMPORAL_GRPC_PORT} accessibility... "
if nc -z localhost "$TEMPORAL_GRPC_PORT" 2>/dev/null; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Port not accessible"
    ((TESTS_FAILED++))
fi

# Test 4: Database is running
echo -n "4. Checking PostgreSQL database... "
if docker ps 2>/dev/null | grep -q "$DB_CONTAINER_NAME"; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Database not running"
    ((TESTS_FAILED++))
fi

# Test 5: CLI is accessible
echo -n "5. Checking Temporal CLI (tctl) accessibility... "
if docker exec "$CONTAINER_NAME" tctl --version >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - CLI not accessible"
    ((TESTS_FAILED++))
fi

# Summary
echo ""
echo "================================"
echo "Test Summary"
echo "================================"
echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}All smoke tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some smoke tests failed!${NC}"
    exit 1
fi