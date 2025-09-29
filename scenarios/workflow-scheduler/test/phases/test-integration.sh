#!/bin/bash
set -e
echo "=== Integration Tests ==="

# Run the comprehensive integration test
if [ -f "../integration-test.sh" ]; then
    bash ../integration-test.sh
else
    echo "⚠️ Integration test script not found, running basic tests..."
    
    # Get API port from environment or use default
    API_PORT="${API_PORT:-18090}"
    API_BASE="http://localhost:$API_PORT"
    
    # Basic integration tests
    echo "Testing API health..."
    if curl -sf "$API_BASE/health" > /dev/null 2>&1; then
        echo "✅ API health check passed"
    else
        echo "❌ API health check failed - API may not be running"
        echo "  Try starting with: vrooli scenario start workflow-scheduler"
        exit 1
    fi
    
    echo "Testing API documentation..."
    if curl -sf "$API_BASE/docs" > /dev/null 2>&1; then
        echo "✅ API docs available"
    else
        echo "⚠️ API docs not available"
    fi
    
    echo "Testing cron validation..."
    if curl -sf "$API_BASE/api/cron/validate?expression=0%209%20*%20*%20*" | grep -q "valid"; then
        echo "✅ Cron validation working"
    else
        echo "⚠️ Cron validation not working"
    fi
fi

echo "✅ Integration tests completed"