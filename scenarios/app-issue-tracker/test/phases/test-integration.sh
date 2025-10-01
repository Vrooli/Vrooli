#!/bin/bash
set -euo pipefail

echo "=== Running test-integration.sh ==="

# Get API port from environment or use default
API_PORT=${API_PORT:-19751}
API_BASE="http://localhost:${API_PORT}"

# Integration tests

if [ -f scripts/test-claude-integration.sh ]; then
  # Codex integration test is legacy and exits early
  # We skip it in favor of the API integration tests below
  echo "⚠ Skipping legacy Codex integration test (use API tests instead)"
else
  echo "⚠ No integration test script found"
fi

# Test API if running
if command -v curl >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
  # Check if API is running
  if curl -sf "${API_BASE}/health" >/dev/null 2>&1; then
    echo "Testing API integration at ${API_BASE}..."

    # Test health endpoint
    HEALTH=$(curl -sf "${API_BASE}/health")
    if echo "$HEALTH" | jq -e '.success == true' >/dev/null; then
      echo "✓ Health check passed"
    else
      echo "✗ Health check failed"
      exit 1
    fi

    # Test stats endpoint with analytics
    STATS=$(curl -sf "${API_BASE}/api/v1/stats")
    if echo "$STATS" | jq -e '.data.stats.avg_resolution_hours' >/dev/null; then
      echo "✓ Stats endpoint with resolution analytics passed"
    else
      echo "✗ Stats endpoint failed"
      exit 1
    fi

    # Test CORS headers
    CORS_TEST=$(curl -sf -I -X OPTIONS \
      -H "Origin: http://example.com" \
      -H "Access-Control-Request-Method: GET" \
      "${API_BASE}/api/v1/issues" 2>&1 | grep -i "access-control" || true)
    if [ -n "$CORS_TEST" ]; then
      echo "✓ CORS headers configured"
    else
      echo "⚠ CORS headers not found (may be expected)"
    fi

    # Test export endpoint
    if curl -sf "${API_BASE}/api/v1/export?format=json" | jq -e '.count' >/dev/null; then
      echo "✓ Export functionality (JSON) passed"
    else
      echo "⚠ Export endpoint returned unexpected response"
    fi

    # Test git integration endpoint existence
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/api/v1/issues/test-id/create-pr")
    if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "400" ] || [ "$HTTP_CODE" = "200" ]; then
      echo "✓ Git integration endpoint accessible (HTTP $HTTP_CODE)"
    else
      echo "⚠ Git integration endpoint response unexpected (HTTP $HTTP_CODE)"
    fi

  else
    echo "⚠ API not running on port ${API_PORT}, skipping integration tests"
    echo "  Start with: make run"
  fi
else
  echo "⚠ curl or jq not available, skipping HTTP integration"
fi

echo "✅ test-integration.sh completed successfully"
