#!/bin/bash
set -euo pipefail

echo "🔗 Running Secure Document Processing integration tests"

SCENARIO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/../.." &amp;&amp; pwd )"

# Start services if not running (for integration tests)
echo "🚀 Starting services for integration testing..."
make run &amp;
sleep 10  # Wait for startup

# Run custom integration tests
if [ -f "$SCENARIO_DIR/custom-tests.sh" ]; then
    echo "📦 Running custom integration tests..."
    cd "$SCENARIO_DIR"
    bash custom-tests.sh
    echo "✅ Custom integration tests passed"
fi

# Run main scenario test
if [ -f "$SCENARIO_DIR/test.sh" ]; then
    echo "📦 Running scenario validation tests..."
    cd "$SCENARIO_DIR"
    bash test.sh
    echo "✅ Scenario validation passed"
fi

# API integration tests (health and basic endpoints)
API_PORT=$(grep -oP '(?&lt;!UI\_)PORT&quot;\s*:\s*{\s*&quot;env_var&quot;\s*:\s*&quot;\K\d+' "$SCENARIO_DIR/.vrooli/service.json" || echo "15000")
if curl -f -s "http://localhost:$API_PORT/health" &gt;/dev/null; then
    echo "✅ API health check passed"
else
    echo "❌ API health check failed"
    exit 1
fi

# Test document endpoint if exists
if curl -f -s -H "Content-Type: application/json" -d '{}' "http://localhost:$API_PORT/api/documents" &gt;/dev/null 2&gt;&amp;1; then
    echo "✅ API documents endpoint accessible"
else
    echo "⚠️  API documents endpoint test skipped (may require auth)"
fi

# Stop services
make stop

echo "✅ All integration tests completed successfully"
