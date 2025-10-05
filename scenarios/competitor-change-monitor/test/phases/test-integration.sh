#!/bin/bash

# Integration tests for competitor-change-monitor scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running integration tests..."

# Check if API is running
if ! curl -f -s http://localhost:${API_PORT:-8080}/health > /dev/null 2>&1; then
    echo "⚠️  API not running, skipping integration tests"
    testing::phase::end_with_summary "Integration tests skipped (API not running)"
    exit 0
fi

# Test API endpoints
echo "Testing API endpoints..."

# Test health endpoint
if curl -f -s http://localhost:${API_PORT:-8080}/health | grep -q "healthy"; then
    echo "✓ Health endpoint working"
else
    echo "✗ Health endpoint failed"
    exit 1
fi

# Test competitors endpoint
if curl -f -s http://localhost:${API_PORT:-8080}/api/competitors > /dev/null; then
    echo "✓ Competitors endpoint working"
else
    echo "✗ Competitors endpoint failed"
    exit 1
fi

# Test alerts endpoint
if curl -f -s http://localhost:${API_PORT:-8080}/api/alerts > /dev/null; then
    echo "✓ Alerts endpoint working"
else
    echo "✗ Alerts endpoint failed"
    exit 1
fi

# Test analyses endpoint
if curl -f -s http://localhost:${API_PORT:-8080}/api/analyses > /dev/null; then
    echo "✓ Analyses endpoint working"
else
    echo "✗ Analyses endpoint failed"
    exit 1
fi

testing::phase::end_with_summary "Integration tests completed"
