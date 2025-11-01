#!/bin/bash
# Video Tools comprehensive test runner
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[1;34m'
NC='\033[0m'

# Track results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

echo -e "${BLUE}🧪 Video Tools Test Suite${NC}"
echo "========================================"
echo ""

# Detect API port - check scenario-specific env var first to avoid conflicts with other scenarios
VIDEO_TOOLS_API_PORT="${VIDEO_TOOLS_API_PORT:-}"
if [ -z "$VIDEO_TOOLS_API_PORT" ]; then
    # Try to detect from running process by checking port 18125 specifically
    # This avoids false matches with other processes
    if lsof -i :18125 -P -n 2>/dev/null | grep -q LISTEN; then
        VIDEO_TOOLS_API_PORT="18125"
    fi
fi
if [ -z "$VIDEO_TOOLS_API_PORT" ]; then
    echo -e "${YELLOW}⚠️  WARNING: video-tools-api is not running - using default port${NC}"
    VIDEO_TOOLS_API_PORT="18125"
fi

# Set API_PORT for backward compatibility with test scripts
API_PORT="$VIDEO_TOOLS_API_PORT"

export API_PORT
export API_URL="http://localhost:${API_PORT}"
echo -e "${GREEN}✓ Using API on port ${API_PORT}${NC}"
echo ""

# Run each test phase
run_phase() {
    local phase=$1
    local script="test/phases/test-${phase}.sh"

    if [ ! -f "$script" ]; then
        echo -e "${YELLOW}⚠️  Skipping ${phase} (script not found)${NC}"
        return 0
    fi

    echo -e "${BLUE}━━━ Phase: ${phase} ━━━${NC}"

    if bash "$script"; then
        echo -e "${GREEN}✓ ${phase} tests passed${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ ${phase} tests failed${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo ""
}

# Run test phases
run_phase "structure"
run_phase "dependencies"
run_phase "unit"
run_phase "integration"
run_phase "business"
run_phase "performance"

# Summary
echo "========================================"
echo -e "${BLUE}📊 Test Summary${NC}"
echo "  Total phases: $TOTAL_TESTS"
echo -e "  ${GREEN}Passed: $PASSED_TESTS${NC}"
if [ $FAILED_TESTS -gt 0 ]; then
    echo -e "  ${RED}Failed: $FAILED_TESTS${NC}"
    exit 1
else
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
fi
