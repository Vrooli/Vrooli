#!/bin/bash
# Unit test phase - <60 seconds
# Runs language-specific unit tests with coverage reporting
set -euo pipefail

echo "=== Unit Tests Phase ==="
start_time=$(date +%s)

# Get paths
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

# Change to scenario directory
cd "$SCENARIO_DIR"

# Source the testing orchestration module
source "$APP_ROOT/scripts/scenarios/testing/shell/orchestration.sh"

# Run unit tests with coverage thresholds
# Auto-detects scenario and languages
if testing::orchestration::run_unit_tests "" 80 70; then
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    echo "✅ Unit tests completed successfully in ${duration}s"
    
    if [ $duration -gt 60 ]; then
        echo "⚠️  Unit phase exceeded 60s target"
    fi
    
    exit 0
else
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    echo "❌ Unit tests failed in ${duration}s"
    exit 1
fi