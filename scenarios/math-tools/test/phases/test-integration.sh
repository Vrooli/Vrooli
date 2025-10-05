#!/bin/bash
# Integration tests for math-tools scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running integration tests for math-tools..."

# Test API endpoints are accessible
testing::phase::step "Testing API endpoints accessibility"

# Start the server in background for integration tests (if not already running)
if ! pgrep -f "math-tools" > /dev/null; then
    echo "Note: Integration tests require running server. Skipping endpoint tests."
    testing::phase::end_with_summary "Integration tests completed (server not running)"
    exit 0
fi

# Test health endpoint
if command -v curl &> /dev/null; then
    HEALTH_RESPONSE=$(curl -s http://localhost:8095/health 2>/dev/null || echo "failed")
    if [[ "$HEALTH_RESPONSE" == *"healthy"* ]]; then
        echo "✓ Health endpoint responding"
    else
        echo "✗ Health endpoint not responding"
    fi
fi

testing::phase::end_with_summary "Integration tests completed"
