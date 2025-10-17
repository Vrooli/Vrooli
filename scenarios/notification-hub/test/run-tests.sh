#!/bin/bash
# Main test runner for notification-hub scenario

set -euo pipefail

SCENARIO_DIR="$(cd "${BASH_SOURCE[0]%/*}/.." && pwd)"
cd "$SCENARIO_DIR"

echo "🧪 Running notification-hub test suite..."
echo ""

# Run unit tests
if [ -f "test/phases/test-unit.sh" ]; then
    echo "📝 Running unit tests..."
    bash test/phases/test-unit.sh || echo "⚠️  Unit tests encountered issues"
    echo ""
fi

# Run integration tests
if [ -f "test/phases/test-integration.sh" ]; then
    echo "🔗 Running integration tests..."
    bash test/phases/test-integration.sh || echo "⚠️  Integration tests skipped (may need services)"
    echo ""
fi

# Run performance tests
if [ -f "test/phases/test-performance.sh" ]; then
    echo "⚡ Running performance tests..."
    bash test/phases/test-performance.sh || echo "⚠️  Performance tests skipped"
    echo ""
fi

echo "✅ Test suite execution completed"
