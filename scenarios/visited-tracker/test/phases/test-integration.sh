#!/bin/bash
set -euo pipefail

echo "=== Integration Tests Phase ==="
echo "Comprehensive integration testing: API, CLI (BATS), and Database"

# Setup paths and utilities
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

# Test counters
error_count=0
test_count=0
skipped_count=0

# API Port (should match service configuration)
API_PORT="${API_PORT:-17695}"

echo ""
echo "🌐 Testing API Integration..."
if timeout 10 curl -sf --max-time 5 "http://localhost:${API_PORT}/health" >/dev/null 2>&1; then
    echo "✅ API health check passed"
    test_count=$((test_count + 1))
    
    # Test key API endpoints
    echo "🔍 Testing API endpoints..."
    
    # Test campaigns endpoint
    if timeout 5 curl -sf "http://localhost:${API_PORT}/api/v1/campaigns" >/dev/null 2>&1; then
        echo "  ✅ Campaigns endpoint accessible"
    else
        echo "  ⚠️  Campaigns endpoint failed"
        error_count=$((error_count + 1))
    fi
    
    # Test health endpoint content
    health_response=$(timeout 5 curl -sf "http://localhost:${API_PORT}/health" 2>/dev/null || echo "")
    if [[ "$health_response" =~ "status" ]] || [[ "$health_response" =~ "ok" ]] || [[ "$health_response" =~ "healthy" ]]; then
        echo "  ✅ Health endpoint returns valid response"
    else
        echo "  ⚠️  Health endpoint response unclear: $health_response"
        error_count=$((error_count + 1))
    fi
    
else
    echo "❌ API integration tests failed - service not responding"
    echo "   Expected API at: http://localhost:${API_PORT}"
    echo "   💡 Tip: Start with 'vrooli scenario run visited-tracker'"
    error_count=$((error_count + 1))
fi

echo ""
echo "🖥️  Testing CLI Integration with Comprehensive BATS Suite..."
CLI_TEST_SCRIPT="$SCENARIO_DIR/test/cli/run-cli-tests.sh"

if [ -f "$CLI_TEST_SCRIPT" ] && [ -x "$CLI_TEST_SCRIPT" ]; then
    echo "🔧 Running comprehensive CLI BATS tests..."
    
    # Set environment for CLI tests
    export API_PORT="$API_PORT"
    
    if "$CLI_TEST_SCRIPT"; then
        echo "✅ CLI BATS integration tests passed"
        test_count=$((test_count + 1))
    else
        echo "❌ CLI BATS integration tests failed"
        error_count=$((error_count + 1))
    fi
else
    echo "⚠️  CLI test runner not found at $CLI_TEST_SCRIPT"
    echo "   Falling back to basic CLI test..."
    
    # Fallback to basic CLI test
    CLI_BINARY="$SCENARIO_DIR/cli/visited-tracker"
    if [ -f "$CLI_BINARY" ] && [ -x "$CLI_BINARY" ]; then
        if "$CLI_BINARY" version >/dev/null 2>&1; then
            echo "✅ Basic CLI test passed"
            test_count=$((test_count + 1))
        else
            echo "❌ Basic CLI test failed"
            error_count=$((error_count + 1))
        fi
    else
        echo "⚠️  CLI not available at $CLI_BINARY - skipping"
        skipped_count=$((skipped_count + 1))
    fi
fi

echo ""
echo "🗄️  Testing Database Integration..."
if command -v resource-postgres >/dev/null 2>&1; then
    echo "🔍 Running PostgreSQL resource smoke tests..."
    if timeout 30 resource-postgres test smoke >/dev/null 2>&1; then
        echo "✅ Database integration tests passed"
        test_count=$((test_count + 1))
    else
        echo "❌ Database integration tests failed"
        error_count=$((error_count + 1))
    fi
else
    echo "⚠️  PostgreSQL resource not available - skipping database tests"
    echo "   💡 Tip: Install with 'vrooli resource start postgres'"
    skipped_count=$((skipped_count + 1))
fi

echo ""
echo "🔄 Testing End-to-End Workflow (if API is running)..."
if timeout 5 curl -sf "http://localhost:${API_PORT}/health" >/dev/null 2>&1; then
    echo "🧪 Running end-to-end workflow test..."
    
    # Create test campaign via API
    campaign_response=$(timeout 10 curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d '{"name":"integration-test-campaign","description":"Test campaign for integration tests"}' \
        "http://localhost:${API_PORT}/api/v1/campaigns" 2>/dev/null || echo "")
    
    if [[ "$campaign_response" =~ "id" ]] || [[ "$campaign_response" =~ "success" ]]; then
        echo "✅ End-to-end workflow test passed"
        test_count=$((test_count + 1))
    else
        echo "⚠️  End-to-end workflow test had issues: $campaign_response"
        error_count=$((error_count + 1))
    fi
else
    echo "⚠️  Skipping end-to-end workflow test - API not available"
    skipped_count=$((skipped_count + 1))
fi

echo ""
echo "📊 Integration Test Summary:"
echo "════════════════════════════"
echo "   Tests passed: $test_count"
echo "   Tests failed: $error_count"
echo "   Tests skipped: $skipped_count"
echo "   Total coverage: $((test_count + error_count + skipped_count)) integration scenarios"

if [ $error_count -eq 0 ]; then
    echo ""
    echo "✅ SUCCESS: All integration tests passed!"
    if [ $skipped_count -gt 0 ]; then
        echo "   ℹ️  Note: $skipped_count tests were skipped due to missing dependencies"
    fi
    exit 0
else
    echo ""
    echo "❌ ERROR: $error_count integration tests failed"
    echo ""
    echo "🔧 Troubleshooting:"
    echo "   • Ensure service is running: vrooli scenario run visited-tracker"
    echo "   • Check service logs: vrooli scenario logs visited-tracker"
    echo "   • Verify PostgreSQL: vrooli resource start postgres"
    echo "   • Install CLI: cd cli && ./install.sh"
    exit 1
fi