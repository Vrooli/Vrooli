#!/bin/bash
# Integration test phase for system-monitor scenario

set -euo pipefail

# Resolve APP_ROOT dynamically
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize test phase
testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running integration tests for system-monitor"

# Run Go integration tests with integration tag
if [ -d "api" ]; then
    cd api

    echo "→ Running Go integration tests..."
    go test -v -tags=integration ./... -timeout 120s || {
        echo "✗ Integration tests failed"
        exit 1
    }

    cd ..
fi

testing::phase::end_with_summary "Integration tests completed"
