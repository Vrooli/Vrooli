#!/bin/bash
# Unit tests phase - <60 seconds
# Orchestrates all language-specific unit tests
set -euo pipefail

echo "=== Unit Tests Phase (Target: <60s) ==="
start_time=$(date +%s)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Get the test directory
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UNIT_DIR="$TEST_DIR/unit"

error_count=0
test_count=0
skipped_count=0

echo "🧪 Running unit tests from: $UNIT_DIR"

# Check if unit test directory exists
if [ ! -d "$UNIT_DIR" ]; then
    echo -e "${YELLOW}⚠️  Unit test directory not found: $UNIT_DIR${NC}"
    echo -e "${YELLOW}   Creating basic unit test structure...${NC}"
    mkdir -p "$UNIT_DIR"
fi

# Run Go unit tests
echo ""
echo "🔍 Running Go unit tests..."
if [ -d "api" ] && [ -f "api/main.go" ]; then
    if [ -x "$UNIT_DIR/go.sh" ]; then
        if "$UNIT_DIR/go.sh"; then
            echo -e "${GREEN}✅ Go unit tests passed${NC}"
            ((test_count++))
        else
            echo -e "${RED}❌ Go unit tests failed${NC}"
            ((error_count++))
        fi
    else
        # Run Go tests directly if no runner script
        echo "Running Go tests directly..."
        cd api
        if go test -v ./... -timeout 30s -cover; then
            echo -e "${GREEN}✅ Go unit tests passed${NC}"
            ((test_count++))
        else
            echo -e "${RED}❌ Go unit tests failed${NC}"
            ((error_count++))
        fi
        cd ..
    fi
else
    echo -e "${YELLOW}ℹ️  No Go code found, skipping Go tests${NC}"
    ((skipped_count++))
fi

# Run Node.js unit tests
echo ""
echo "🔍 Running Node.js unit tests..."
if [ -d "ui" ] && [ -f "ui/package.json" ]; then
    if [ -x "$UNIT_DIR/node.sh" ]; then
        if "$UNIT_DIR/node.sh"; then
            echo -e "${GREEN}✅ Node.js unit tests passed${NC}"
            ((test_count++))
        else
            echo -e "${RED}❌ Node.js unit tests failed${NC}"
            ((error_count++))
        fi
    else
        # Check if package.json has test script
        cd ui
        if jq -e '.scripts.test' package.json >/dev/null 2>&1; then
            echo "Running Node.js tests directly..."
            if npm test --passWithNoTests --silent; then
                echo -e "${GREEN}✅ Node.js unit tests passed${NC}"
                ((test_count++))
            else
                echo -e "${RED}❌ Node.js unit tests failed${NC}"
                ((error_count++))
            fi
        else
            echo -e "${YELLOW}ℹ️  No test script in package.json, skipping Node.js tests${NC}"
            ((skipped_count++))
        fi
        cd ..
    fi
else
    echo -e "${YELLOW}ℹ️  No Node.js code found, skipping Node.js tests${NC}"
    ((skipped_count++))
fi

# Run Python unit tests (if any Python components exist)
echo ""
echo "🔍 Checking for Python unit tests..."
if [ -f "requirements.txt" ] || [ -f "pyproject.toml" ]; then
    if [ -x "$UNIT_DIR/python.sh" ]; then
        if "$UNIT_DIR/python.sh"; then
            echo -e "${GREEN}✅ Python unit tests passed${NC}"
            ((test_count++))
        else
            echo -e "${RED}❌ Python unit tests failed${NC}"
            ((error_count++))
        fi
    else
        echo -e "${YELLOW}ℹ️  Python dependencies found but no test runner, skipping Python tests${NC}"
        ((skipped_count++))
    fi
else
    echo -e "${YELLOW}ℹ️  No Python code found, skipping Python tests${NC}"
    ((skipped_count++))
fi

# Check for any language-specific test files that we might have missed
echo ""
echo "🔍 Scanning for additional unit tests..."
additional_test_files=()

# Look for test files in api directory
if [ -d "api" ]; then
    while IFS= read -r -d '' file; do
        additional_test_files+=("$file")
    done < <(find api -name "*_test.go" -type f -print0 2>/dev/null)
fi

# Look for test files in ui directory
if [ -d "ui" ]; then
    while IFS= read -r -d '' file; do
        additional_test_files+=("$file")
    done < <(find ui -name "*.test.js" -o -name "*.spec.js" -o -name "__tests__/*" -type f -print0 2>/dev/null)
fi

if [ ${#additional_test_files[@]} -gt 0 ]; then
    echo -e "${BLUE}ℹ️  Found ${#additional_test_files[@]} additional test files:${NC}"
    for file in "${additional_test_files[@]}"; do
        echo "   - $file"
    done
    
    # Note: These are likely already run by the language-specific runners above
    echo -e "${GREEN}✅ Test files detected (likely executed by language runners)${NC}"
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
echo "   Total test suites: $total_tests"
echo "   Duration: ${duration}s"

if [ $error_count -eq 0 ]; then
    if [ $test_count -gt 0 ]; then
        echo -e "${GREEN}✅ All unit tests passed in ${duration}s${NC}"
    else
        echo -e "${YELLOW}⚠️  No unit tests were executed in ${duration}s${NC}"
        echo -e "${BLUE}💡 Consider adding unit tests for better coverage${NC}"
    fi
else
    echo -e "${RED}❌ Unit tests failed with $error_count failures in ${duration}s${NC}"
fi

if [ $duration -gt 60 ]; then
    echo -e "${YELLOW}⚠️  Unit tests phase exceeded 60s target${NC}"
fi

# Show recommendations for missing test infrastructure
if [ $test_count -eq 0 ] || [ $skipped_count -gt 1 ]; then
    echo ""
    echo -e "${BLUE}💡 Unit test infrastructure recommendations:${NC}"
    
    if [ ! -x "$UNIT_DIR/go.sh" ] && [ -d "api" ]; then
        echo "   • Create $UNIT_DIR/go.sh for Go unit test runner"
    fi
    
    if [ ! -x "$UNIT_DIR/node.sh" ] && [ -d "ui" ]; then
        echo "   • Create $UNIT_DIR/node.sh for Node.js unit test runner"
    fi
    
    if [ -d "ui" ] && [ -f "ui/package.json" ]; then
        if ! jq -e '.scripts.test' ui/package.json >/dev/null 2>&1; then
            echo "   • Add test script to ui/package.json"
        fi
    fi
    
    echo "   • See: docs/scenarios/PHASED_TESTING_ARCHITECTURE.md"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi