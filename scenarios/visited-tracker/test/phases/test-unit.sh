#!/bin/bash
# Unit test phase - uses centralized testing library
set -euo pipefail

echo "=== Unit Tests Phase ==="

# Get paths
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"
TESTING_LIB="$APP_ROOT/scripts/lib/testing/unit"

# Check if centralized testing library exists
if [ ! -f "$TESTING_LIB/run-all.sh" ]; then
    echo "❌ Centralized testing library not found at $TESTING_LIB"
    echo "   Falling back to inline tests..."
    
    # Fallback to simple inline tests
    error_count=0
    test_count=0
    
    echo "Testing Go..."
    if [ -d "api" ] && [ -f "api/go.mod" ]; then
        cd api
        if go test ./... -timeout 5s >/dev/null 2>&1; then
            echo "Go tests passed"
            test_count=$((test_count + 1))
        else
            echo "Go tests failed"
            error_count=$((error_count + 1))
        fi
        cd ..
    fi
    
    echo "Testing Node.js..."
    if [ -d "ui" ] && [ -f "ui/package.json" ]; then
        cd ui
        if npm test --passWithNoTests --silent >/dev/null 2>&1; then
            echo "Node.js tests passed"
            test_count=$((test_count + 1))
        else
            echo "Node.js tests failed"  
            error_count=$((error_count + 1))
        fi
        cd ..
    fi
    
    echo "Summary: $test_count passed, $error_count failed"
    
    if [ $error_count -eq 0 ]; then
        echo "SUCCESS: All tests passed"
        exit 0
    else
        echo "ERROR: Some tests failed"
        exit 1
    fi
fi

# Use centralized testing library
echo "Using centralized testing library..."

# Source the testing library
source "$TESTING_LIB/run-all.sh"

# Change to scenario directory
cd "$SCENARIO_DIR"

# Run unit tests
if testing::unit::run_all_tests --go-dir "api" --node-dir "ui" --skip-python; then
    echo "SUCCESS: All unit tests passed"
    exit 0
else
    echo "ERROR: Some unit tests failed"
    exit 1
fi