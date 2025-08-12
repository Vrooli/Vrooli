#!/usr/bin/env bash
# Test runner script for Metareasoning API with coverage reporting

set -e

cd "$(dirname "$0")"

echo "=== Metareasoning API Test Suite ==="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Go is not installed${NC}"
    exit 1
fi

# Clean up previous coverage files
rm -f coverage.out coverage.html

# Run tests with coverage
echo "Running tests with coverage..."
echo "================================"

# Set test database URL if not already set
export TEST_DATABASE_URL="${TEST_DATABASE_URL:-postgres://postgres:postgres@localhost:5432/metareasoning_test?sslmode=disable}"

# Run tests with coverage for all packages
if go test -v -race -coverprofile=coverage.out -covermode=atomic ./... 2>&1; then
    echo -e "\n${GREEN}✓ All tests passed${NC}\n"
else
    echo -e "\n${RED}✗ Some tests failed${NC}\n"
    exit 1
fi

# Generate coverage report
echo "Generating coverage report..."
echo "=============================="

# Get coverage percentage
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo -e "Total Coverage: ${YELLOW}${COVERAGE}${NC}"

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
echo "HTML coverage report generated: coverage.html"

# Show uncovered lines for each file
echo
echo "Coverage by file:"
echo "================="
go tool cover -func=coverage.out | grep -v "total:"

# Check if we meet the target coverage (90%)
COVERAGE_NUM=$(echo $COVERAGE | sed 's/%//')
TARGET_COVERAGE=90

echo
if (( $(echo "$COVERAGE_NUM >= $TARGET_COVERAGE" | bc -l) )); then
    echo -e "${GREEN}✓ Coverage target met! ($COVERAGE >= ${TARGET_COVERAGE}%)${NC}"
else
    echo -e "${YELLOW}⚠ Coverage below target ($COVERAGE < ${TARGET_COVERAGE}%)${NC}"
    echo "Areas needing more tests:"
    go tool cover -func=coverage.out | grep -E "^.*\s+[0-8][0-9]\.[0-9]%$" | head -10
fi

# Run benchmarks (optional)
if [[ "$1" == "--bench" ]]; then
    echo
    echo "Running benchmarks..."
    echo "===================="
    go test -bench=. -benchmem ./...
fi

# Generate test report summary
echo
echo "Test Summary"
echo "============"
echo "Files tested: $(go list ./... | wc -l)"
echo "Test functions: $(grep -r "^func Test" *_test.go | wc -l)"
echo "Coverage: $COVERAGE"
echo "Report: coverage.html"

# Check for race conditions (in a separate run)
echo
echo "Checking for race conditions..."
if go test -race ./... > /dev/null 2>&1; then
    echo -e "${GREEN}✓ No race conditions detected${NC}"
else
    echo -e "${RED}✗ Race conditions detected${NC}"
fi

echo
echo "=== Test run complete ===
echo "To view the HTML coverage report, run: open coverage.html (or firefox coverage.html)"