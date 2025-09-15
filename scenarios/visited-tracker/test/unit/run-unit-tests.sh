#!/bin/bash
# Visited Tracker unit test runner
# Uses centralized testing library
set -euo pipefail

# Get paths
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"
TESTING_LIB="$APP_ROOT/scripts/scenarios/testing/unit"

# Source the centralized unit test runner
source "$TESTING_LIB/run-all.sh"

# Run all unit tests for visited-tracker
# Using default directories: api/ for Go, ui/ for Node.js
echo "=== Running Visited Tracker Unit Tests ==="
echo "Using centralized testing library"
echo ""

# Change to scenario directory
cd "$SCENARIO_DIR"

# Run all tests with scenario-specific configuration
testing::unit::run_all_tests \
    --go-dir "api" \
    --node-dir "ui" \
    --skip-python  # visited-tracker doesn't have Python components

exit $?