#!/bin/bash
# Unit test phase - uses centralized testing library with fallback support
set -euo pipefail

echo "=== Unit Tests Phase ==="

# Get paths
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"
TESTING_LIB="$APP_ROOT/scripts/scenarios/testing/unit"

# Check if centralized testing library exists
if [ ! -f "$TESTING_LIB/run-all.sh" ]; then
    echo "⚠️  Centralized testing library not found at $TESTING_LIB"
    echo "   Using fallback test runner..."
    
    # Use the fallback library instead of inline code
    source "$TESTING_LIB/fallback.sh"
    
    # Change to scenario directory
    cd "$SCENARIO_DIR"
    
    # Run fallback tests with same parameters as the main library would use
    if testing::unit::fallback::run_all \
        --go-dir "api" \
        --node-dir "ui" \
        --skip-python \
        --verbose false; then
        exit 0
    else
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
if testing::unit::run_all_tests --go-dir "api" --node-dir "ui" --skip-python --coverage-warn 78; then
    echo "SUCCESS: All unit tests passed"
    exit 0
else
    echo "ERROR: Some unit tests failed"
    exit 1
fi
