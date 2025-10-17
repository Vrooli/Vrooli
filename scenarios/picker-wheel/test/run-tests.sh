#!/bin/bash
set -e

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCENARIO_DIR"

echo "🧪 Running Picker Wheel Test Suite"
echo "=================================="

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test phases
PHASES=(
  "structure"
  "unit"
  "integration"
  "performance"
)

# Track results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Run each phase
for phase in "${PHASES[@]}"; do
  TEST_FILE="test/phases/test-${phase}.sh"
  TOTAL_TESTS=$((TOTAL_TESTS + 1))
  
  if [ ! -f "$TEST_FILE" ]; then
    echo -e "${YELLOW}⚠️  Skipping $phase tests (file not found)${NC}"
    SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
    continue
  fi
  
  echo ""
  echo "🔍 Running $phase tests..."
  echo "------------------------"
  
  if bash "$TEST_FILE"; then
    echo -e "${GREEN}✅ $phase tests passed${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
  else
    echo -e "${RED}❌ $phase tests failed${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    
    # Continue running other phases even if one fails
    if [ "$CONTINUE_ON_FAILURE" != "true" ]; then
      echo "Stopping test execution due to failure"
      break
    fi
  fi
done

# Summary
echo ""
echo "=================================="
echo "📊 Test Summary"
echo "=================================="
echo "Total:   $TOTAL_TESTS"
echo -e "Passed:  ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed:  ${RED}$FAILED_TESTS${NC}"
echo -e "Skipped: ${YELLOW}$SKIPPED_TESTS${NC}"

# Exit with failure if any tests failed
if [ $FAILED_TESTS -gt 0 ]; then
  exit 1
fi

echo ""
echo -e "${GREEN}🎉 All tests passed!${NC}"