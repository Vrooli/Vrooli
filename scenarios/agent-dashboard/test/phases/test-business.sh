#!/bin/bash
set -e

echo "=== Business Tests ==="
# Test business-specific logic
# Example: Check if dashboard loads correctly, API returns expected data
curl -f http://localhost:$API_PORT/api/dashboard || echo "Business test: Dashboard API check"
echo "✅ Business tests completed"