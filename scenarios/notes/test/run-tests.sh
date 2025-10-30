#!/bin/bash
# SmartNotes comprehensive test runner
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

echo -e "${BLUE}🧪 SmartNotes Test Suite${NC}"
echo "========================================"
echo ""

# Detect API port
API_PORT=$(lsof -i -P -n 2>/dev/null | grep "notes-api" | grep LISTEN | awk '{print $9}' | cut -d: -f2 | head -1)
if [ -z "$API_PORT" ]; then
    echo -e "${RED}❌ ERROR: notes-api is not running${NC}"
    exit 1
fi

# Detect UI port - find the node process listening on a port in the UI range (35000-39999)
# that belongs to the notes scenario (check cwd via pwdx)
UI_PORT=$(lsof -i -P -n 2>/dev/null | awk '/node.*LISTEN/ && $9 ~ /:3[5-9][0-9]{3}/ {print $2,$9}' | while read pid port; do
    cwd=$(pwdx "$pid" 2>/dev/null | cut -d: -f2 | tr -d ' ')
    if [[ "$cwd" == */scenarios/notes/ui ]]; then
        echo "$port" | cut -d: -f2
        break
    fi
done | head -1)

export API_PORT
export UI_PORT
export API_URL="http://localhost:${API_PORT}"
echo -e "${GREEN}✓ Detected API on port ${API_PORT}${NC}"
if [ -n "$UI_PORT" ]; then
    echo -e "${GREEN}✓ Detected UI on port ${UI_PORT}${NC}"
fi
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

# Run test phases (in order of increasing complexity)
run_phase "smoke"
run_phase "structure"
run_phase "dependencies"
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
