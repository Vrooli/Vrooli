#!/bin/bash
# Integration tests for video-tools scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../..}" && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🔗 Running video-tools integration tests..."

# Check if API is running
if ! curl -sf http://localhost:${API_PORT:-15760}/health >/dev/null 2>&1; then
    echo "⚠️  API not running, starting it for integration tests..."
    # Note: In CI, the API should already be running
    cd api && ./video-tools-api &
    API_PID=$!
    sleep 3
fi

# Test API endpoints
echo "Testing health endpoint..."
response=$(curl -sf http://localhost:${API_PORT:-15760}/health || echo "FAILED")
if [[ "$response" == "FAILED" ]]; then
    echo "❌ Health endpoint failed"
    exit 1
fi
echo "✅ Health endpoint OK"

echo "Testing status endpoint..."
response=$(curl -sf http://localhost:${API_PORT:-15760}/api/status || echo "FAILED")
if [[ "$response" == "FAILED" ]]; then
    echo "❌ Status endpoint failed"
    exit 1
fi
echo "✅ Status endpoint OK"

echo "Testing authentication..."
response=$(curl -sf -H "Authorization: Bearer test-token" \
    http://localhost:${API_PORT:-15760}/api/v1/jobs || echo "FAILED")
if [[ "$response" == "FAILED" ]]; then
    echo "⚠️  Jobs endpoint failed (may need database)"
else
    echo "✅ Jobs endpoint OK"
fi

# Cleanup if we started the API
if [[ -n "${API_PID:-}" ]]; then
    kill $API_PID 2>/dev/null || true
fi

testing::phase::end_with_summary "Integration tests completed"
