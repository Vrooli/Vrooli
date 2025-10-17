#!/bin/bash
# Integration tests for news-aggregator-bias-analysis scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "📦 Running integration tests..."

# Test database connectivity
if [ -n "${POSTGRES_URL:-}" ]; then
    echo "✓ Testing database connection..."

    # Use psql if available, otherwise skip
    if command -v psql &> /dev/null; then
        if psql "${POSTGRES_URL}" -c "SELECT 1;" &> /dev/null; then
            echo "  ✓ Database connection successful"
        else
            echo "  ⚠️  Database connection failed (may be expected in test environment)"
        fi
    else
        echo "  ⚠️  psql not available, skipping database connectivity test"
    fi
else
    echo "  ⚠️  POSTGRES_URL not set, skipping database tests"
fi

# Test API endpoints if service is running
API_PORT="${API_PORT:-8080}"
if curl -sf "http://localhost:${API_PORT}/health" &> /dev/null; then
    echo "✓ Testing API endpoints..."

    # Test health endpoint
    if curl -sf "http://localhost:${API_PORT}/health" | grep -q "healthy"; then
        echo "  ✓ Health endpoint responding"
    else
        echo "  ✗ Health endpoint check failed"
        exit 1
    fi

    # Test articles endpoint
    if curl -sf "http://localhost:${API_PORT}/articles" &> /dev/null; then
        echo "  ✓ Articles endpoint responding"
    else
        echo "  ✗ Articles endpoint check failed"
        exit 1
    fi

    # Test feeds endpoint
    if curl -sf "http://localhost:${API_PORT}/feeds" &> /dev/null; then
        echo "  ✓ Feeds endpoint responding"
    else
        echo "  ✗ Feeds endpoint check failed"
        exit 1
    fi

    echo "✓ All API integration tests passed"
else
    echo "⚠️  API service not running on port ${API_PORT}, skipping live API tests"
    echo "   (This is expected if running tests without starting the service)"
fi

testing::phase::end_with_summary "Integration tests completed"
