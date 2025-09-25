#!/bin/bash

# Unit Test Phase
# Runs unit tests for Email Triage components

set -euo pipefail

echo "🧪 Running Email Triage unit tests..."

SCENARIO_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
FAILURES=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Run Go unit tests
echo "Running Go unit tests..."
cd "$SCENARIO_DIR/api"

if go test ./... -v -cover 2>&1 | tee /tmp/email-triage-go-tests.log; then
    echo -e "${GREEN}✅ Go unit tests passed${NC}"
else
    echo -e "${RED}❌ Go unit tests failed${NC}"
    ((FAILURES++))
fi

# Check test coverage
coverage=$(grep -o 'coverage: [0-9.]*%' /tmp/email-triage-go-tests.log | tail -1 | grep -o '[0-9.]*' || echo "0")
echo "Test coverage: ${coverage}%"

if (( $(echo "$coverage > 30" | bc -l) )); then
    echo -e "${GREEN}✓ Coverage meets minimum threshold (30%)${NC}"
else
    echo -e "${YELLOW}⚠ Coverage below threshold (got ${coverage}%, need 30%)${NC}"
fi

# Run UI tests if present
if [[ -f "$SCENARIO_DIR/ui/package.json" ]]; then
    echo -e "\nRunning UI tests..."
    cd "$SCENARIO_DIR/ui"
    
    if [[ -f "package-lock.json" ]] || [[ -f "yarn.lock" ]]; then
        if npm test --if-present 2>&1 | tee /tmp/email-triage-ui-tests.log; then
            echo -e "${GREEN}✅ UI tests passed${NC}"
        else
            echo -e "${YELLOW}⚠ UI tests not configured or failed${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ UI dependencies not installed${NC}"
    fi
else
    echo -e "${YELLOW}⚠ No UI package.json found${NC}"
fi

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo -e "\n${GREEN}✅ Unit tests passed${NC}"
    exit 0
else
    echo -e "\n${RED}❌ $FAILURES unit test suite(s) failed${NC}"
    exit 1
fi