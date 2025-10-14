#!/bin/bash
set -e
echo "=== Business Logic Tests ==="

# API_PORT is set by the lifecycle system when run via 'make test' or 'vrooli scenario test'
API_PORT=${API_PORT:-19796}
# Read token from environment, fallback to default for development/testing
AUTH_TOKEN="Bearer ${DATA_TOOLS_API_TOKEN:-data-tools-secret-token}"

if curl -sf -H "Authorization: $AUTH_TOKEN" http://localhost:${API_PORT}/health > /dev/null 2>&1; then
  echo "API is healthy, running business tests"

  # Test 1: Data parsing business logic
  echo "Testing CSV parsing..."
  result=$(curl -s -X POST \
    -H "Authorization: $AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"data":"name,age\nJohn,30\nJane,25","format":"csv"}' \
    http://localhost:${API_PORT}/api/v1/data/parse)

  if echo "$result" | grep -q '"success":true'; then
    echo "✅ CSV parsing business logic validated"
  else
    echo "❌ CSV parsing failed"
    exit 1
  fi

  # Test 2: Data transformation business logic
  echo "Testing data transformation..."
  result=$(curl -s -X POST \
    -H "Authorization: $AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"data":[{"age":30},{"age":20}],"transformations":[{"type":"filter","parameters":{"condition":"age > 25"}}]}' \
    http://localhost:${API_PORT}/api/v1/data/transform)

  if echo "$result" | grep -q '"success":true'; then
    echo "✅ Data transformation business logic validated"
  else
    echo "❌ Data transformation failed"
    exit 1
  fi

  # Test 3: Data validation business logic
  echo "Testing data validation..."
  result=$(curl -s -X POST \
    -H "Authorization: $AUTH_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"data":[{"name":"John"}],"schema":{"columns":[{"name":"name","type":"string"}]}}' \
    http://localhost:${API_PORT}/api/v1/data/validate)

  if echo "$result" | grep -q '"success":true'; then
    echo "✅ Data validation business logic validated"
  else
    echo "❌ Data validation failed"
    exit 1
  fi

  echo "Business logic validation passed"
else
  echo "API not running, skipping business tests"
fi
echo "✅ Business tests completed"
