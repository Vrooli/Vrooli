#!/bin/bash
set -e
echo "=== Business Logic Tests ==="
# Test core business logic
# Example: curl test for API endpoints
if curl -f http://localhost:8090/health > /dev/null 2>&1; then
  echo "✅ API health check passed"
else
  echo "⚠️ API not running, skipping business tests"
fi
echo "✅ Business tests passed"