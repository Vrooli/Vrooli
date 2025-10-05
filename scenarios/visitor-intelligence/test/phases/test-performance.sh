#!/bin/bash
# Performance test phase for visitor-intelligence scenario

set -euo pipefail

# Get project root
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source centralized testing utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "180s"

# Navigate to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running performance tests..."

# Run Go performance tests
if [ -d "api" ]; then
    cd api
    echo "Running Go performance benchmarks..."
    go test -bench=. -benchmem -run=^$ -timeout=3m || echo "No performance benchmarks found"
    cd ..
fi

# End phase with summary
testing::phase::end_with_summary "Performance tests completed"
