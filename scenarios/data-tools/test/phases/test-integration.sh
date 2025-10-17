#!/bin/bash
set -e
echo "=== Integration Tests ==="

# API_PORT is set by the lifecycle system when run via 'make test' or 'vrooli scenario test'
# For manual testing, use: API_PORT=19796 bash test/phases/test-integration.sh
API_PORT=${API_PORT:-19796}
# Read token from environment, fallback to default for development/testing
AUTH_TOKEN="Bearer ${DATA_TOOLS_API_TOKEN:-data-tools-secret-token}"

# Wait for API to be ready
echo "Waiting for API to be ready..."
max_retries=30
retry_count=0
while [ $retry_count -lt $max_retries ]; do
  if curl -sf -H "Authorization: $AUTH_TOKEN" http://localhost:${API_PORT}/health > /dev/null 2>&1; then
    echo "✅ API is ready"
    break
  fi
  retry_count=$((retry_count + 1))
  if [ $retry_count -ge $max_retries ]; then
    echo "❌ API failed to become ready"
    exit 1
  fi
  sleep 1
done

# Test PostgreSQL integration
echo "Testing PostgreSQL integration..."
result=$(curl -s -X POST \
  -H "Authorization: $AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"sql":"SELECT 1 as test"}' \
  http://localhost:${API_PORT}/api/v1/data/query)

if echo "$result" | grep -q '"success":true'; then
  echo "✅ PostgreSQL integration working"
else
  echo "❌ PostgreSQL integration failed"
  exit 1
fi

# Test streaming endpoint
echo "Testing streaming integration..."
result=$(curl -s -X POST \
  -H "Authorization: $AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"source_config":{"type":"webhook"},"processing_rules":[],"output_config":{"destination":"dataset"}}' \
  http://localhost:${API_PORT}/api/v1/data/stream/create)

if echo "$result" | grep -q '"success":true'; then
  echo "✅ Streaming integration working"
else
  echo "❌ Streaming integration failed"
  exit 1
fi

echo "✅ Integration tests completed"
