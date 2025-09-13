#!/bin/bash
# Simplified Unit Tests Phase - <60 seconds  
# Runs unit tests for Go and Node.js components
set -uo pipefail

echo "=== Unit Tests Phase (Target: <60s) ==="
start_time=$(date +%s)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

error_count=0
test_count=0
skipped_count=0

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UNIT_DIR="$TEST_DIR/unit"

echo "🧪 Running unit tests from: $TEST_DIR/unit"

# Run Go unit tests
echo ""
echo "🔍 Running Go unit tests..."
if [ -d "api" ] && [ -f "api/go.mod" ]; then
    if [ -x "$UNIT_DIR/go.sh" ]; then
        if "$UNIT_DIR/go.sh" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ Go unit tests passed${NC}"
            ((test_count++))
        else
            echo -e "${RED}❌ Go unit tests failed${NC}"
            ((error_count++))
        fi
    else
        echo -e "${YELLOW}ℹ️  Go test runner not found, skipping Go tests${NC}"
        ((skipped_count++))
    fi
else
    echo -e "${YELLOW}ℹ️  No Go code found, skipping Go tests${NC}"
    ((skipped_count++))
fi

# Run Node.js unit tests (simplified)
echo ""
echo "🔍 Running Node.js unit tests..."
if [ -d "ui" ] && [ -f "ui/package.json" ]; then
    cd ui
    # Check if test script exists and jest is installed
    if jq -e '.scripts.test' package.json >/dev/null 2>&1 && [ -d "node_modules/.bin" ] && [ -f "node_modules/.bin/jest" ]; then
        echo "📦 Running Node.js tests..."
        set +e  # Temporarily disable exit on error
        npm test --passWithNoTests --silent >/dev/null 2>&1
        npm_exit_code=$?
        set -e  # Re-enable exit on error
        
        if [ $npm_exit_code -eq 0 ]; then
            echo -e "${GREEN}✅ Node.js unit tests passed${NC}"
            ((test_count++))
        else
            echo -e "${RED}❌ Node.js unit tests failed${NC}"
            ((error_count++))
        fi
    else
        echo -e "${YELLOW}ℹ️  No test script or Jest not installed, skipping Node.js tests${NC}"
        ((skipped_count++))
    fi
    cd ..
else
    echo -e "${YELLOW}ℹ️  No Node.js code found, skipping Node.js tests${NC}"
    ((skipped_count++))
fi

# Performance and summary
end_time=$(date +%s)
duration=$((end_time - start_time))
total_tests=$((test_count + error_count))

echo ""
echo "📊 Unit Test Summary:"
echo "   Tests run: $test_count"
echo "   Tests failed: $error_count" 
echo "   Tests skipped: $skipped_count"
echo "   Duration: ${duration}s"

if [ $error_count -eq 0 ]; then
    if [ $test_count -gt 0 ]; then
        echo -e "${GREEN}✅ All unit tests passed in ${duration}s${NC}"
    else
        echo -e "${YELLOW}⚠️  No unit tests were executed in ${duration}s${NC}"
    fi
else
    echo -e "${RED}❌ Unit tests failed with $error_count failures in ${duration}s${NC}"
fi

if [ $duration -gt 60 ]; then
    echo -e "${YELLOW}⚠️  Unit tests phase exceeded 60s target${NC}"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi