#!/bin/bash
# Test runner for api-library scenario

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Starting api-library test suite${NC}"

# Get API port
API_PORT=$(vrooli scenario port api-library api 2>/dev/null || echo "18904")
export API_PORT

# Phase 1: Unit Tests
echo -e "\n${YELLOW}Phase 1: Unit Tests${NC}"
if [ -f test/phases/test-unit.sh ]; then
    bash test/phases/test-unit.sh
else
    echo "No unit tests found"
fi

# Phase 2: Integration Tests
echo -e "\n${YELLOW}Phase 2: Integration Tests${NC}"
if [ -f test/phases/test-integration.sh ]; then
    bash test/phases/test-integration.sh
else
    echo "No integration tests found"
fi

# Phase 3: Smoke Tests
echo -e "\n${YELLOW}Phase 3: Smoke Tests${NC}"
if [ -f test/phases/test-smoke.sh ]; then
    bash test/phases/test-smoke.sh
else
    echo "No smoke tests found"
fi

echo -e "\n${GREEN}All tests completed successfully!${NC}"
