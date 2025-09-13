#!/bin/bash
# Go unit test runner
# Runs all Go unit tests in the scenario
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🐹 Running Go unit tests..."

# Check if Go is available
if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}❌ Go is not installed${NC}"
    exit 1
fi

# Check if we have Go code
if [ ! -d "api/" ]; then
    echo -e "${YELLOW}ℹ️  No api/ directory found, skipping Go tests${NC}"
    exit 0
fi

if [ ! -f "api/go.mod" ]; then
    echo -e "${YELLOW}ℹ️  No go.mod found in api/, skipping Go tests${NC}"
    exit 0
fi

# Change to API directory
cd api

# Check if there are any test files
test_files=$(find . -name "*_test.go" -type f | wc -l)
if [ "$test_files" -eq 0 ]; then
    echo -e "${YELLOW}ℹ️  No Go test files (*_test.go) found, creating basic test structure${NC}"
    
    # Create a basic test file if main.go exists
    if [ -f "main.go" ]; then
        cat > main_test.go << 'EOF'
package main

import (
    "testing"
)

func TestBasicFunctionality(t *testing.T) {
    t.Log("Basic Go test infrastructure working")
    // Add actual tests here
}

func TestAPIVersion(t *testing.T) {
    expected := "3.0.0"
    if apiVersion != expected {
        t.Errorf("Expected API version %s, got %s", expected, apiVersion)
    }
}

func TestServiceName(t *testing.T) {
    expected := "visited-tracker"
    if serviceName != expected {
        t.Errorf("Expected service name %s, got %s", expected, serviceName)
    }
}
EOF
        echo -e "${GREEN}✅ Created basic test file: main_test.go${NC}"
    fi
fi

echo "📦 Downloading Go module dependencies..."
if ! go mod download; then
    echo -e "${RED}❌ Failed to download Go dependencies${NC}"
    exit 1
fi

echo "🧪 Running Go tests with coverage..."

# Run tests with verbose output, timeout, and coverage
if go test -v ./... -timeout 30s -cover -coverprofile=coverage.out; then
    echo -e "${GREEN}✅ Go unit tests completed successfully${NC}"
    
    # Display coverage summary if coverage file exists
    if [ -f "coverage.out" ]; then
        echo ""
        echo "📊 Go Test Coverage Summary:"
        go tool cover -func=coverage.out | tail -1
        
        # Generate HTML coverage report for manual inspection
        go tool cover -html=coverage.out -o coverage.html 2>/dev/null && \
            echo -e "${BLUE}ℹ️  HTML coverage report generated: api/coverage.html${NC}"
    fi
    
    exit 0
else
    echo -e "${RED}❌ Go unit tests failed${NC}"
    exit 1
fi