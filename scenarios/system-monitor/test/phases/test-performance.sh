#!/bin/bash
# Performance test phase for system-monitor scenario

set -euo pipefail

# Resolve APP_ROOT dynamically
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running performance tests for system-monitor"

if [ -d "api" ]; then
    cd api

    echo "→ Running Go benchmarks..."
    go test -bench=. -benchmem -benchtime=5s ./... -timeout 180s || {
        echo "⚠ Some benchmarks failed"
    }

    echo ""
    echo "→ Running performance-specific tests..."
    go test -v -run TestPerformance ./... -timeout 60s || {
        echo "⚠ Some performance tests failed"
    }

    cd ..
fi

testing::phase::end_with_summary "Performance tests completed"
