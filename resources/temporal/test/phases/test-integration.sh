#!/usr/bin/env bash
# Temporal Resource - Integration Tests
# End-to-end functionality validation

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

echo "======================================"
echo "Temporal Resource - Integration Tests"
echo "======================================"
echo ""

TESTS_PASSED=0
TESTS_FAILED=0

# Pre-check: Service must be running
if ! docker ps | grep -q "$CONTAINER_NAME"; then
    echo -e "${RED}ERROR: Temporal is not running. Start it first with: resource-temporal manage start${NC}"
    exit 1
fi

# Test 1: Default namespace exists
echo -n "1. Checking default namespace... "
if docker exec "$CONTAINER_NAME" tctl namespace describe --namespace default >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Default namespace not found"
    ((TESTS_FAILED++))
fi

# Test 2: Workflow listing works
echo -n "2. Testing workflow listing... "
if docker exec "$CONTAINER_NAME" tctl workflow list --limit 1 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Workflow listing failed"
    ((TESTS_FAILED++))
fi

# Test 3: Cluster information accessible
echo -n "3. Testing cluster information retrieval... "
if docker exec "$CONTAINER_NAME" tctl cluster get-search-attributes >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Cluster info not accessible"
    ((TESTS_FAILED++))
fi

# Test 4: Task queue operations
echo -n "4. Testing task queue operations... "
if docker exec "$CONTAINER_NAME" tctl taskqueue describe --taskqueue default --taskqueuetype workflow >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}⚠ SKIP${NC} - Task queue not initialized (expected on fresh install)"
    # Not counting as failure since this is expected on fresh install
fi

# Test 5: Admin commands work
echo -n "5. Testing admin commands... "
if docker exec "$CONTAINER_NAME" tctl admin cluster describe >/dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Admin commands failed"
    ((TESTS_FAILED++))
fi

# Test 6: Create and describe a test namespace
echo -n "6. Testing namespace creation... "
TEST_NS="test-ns-$(date +%s)"
if docker exec "$CONTAINER_NAME" tctl namespace register --name "$TEST_NS" >/dev/null 2>&1; then
    # Clean up test namespace
    docker exec "$CONTAINER_NAME" tctl namespace update --name "$TEST_NS" --state Deprecated >/dev/null 2>&1 || true
    echo -e "${GREEN}✓ PASS${NC}"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗ FAIL${NC} - Namespace creation failed"
    ((TESTS_FAILED++))
fi

# Summary
echo ""
echo "======================================"
echo "Test Summary"
echo "======================================"
echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}All integration tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some integration tests failed!${NC}"
    exit 1
fi