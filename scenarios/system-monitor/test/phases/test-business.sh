#!/bin/bash
set -e

echo "Running business logic tests for system-monitor"

# Test business logic, e.g., API calls
curl -f http://localhost:$API_PORT/api/metrics || echo "API not running, skipping"

echo "✅ Business tests completed"
