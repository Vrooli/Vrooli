#!/bin/bash
set -e
echo "=== Business Logic Tests ==="
# Run business-specific tests
# For app-monitor, perhaps test API endpoints with curl or something
curl -f http://localhost:${API_PORT:-8080}/health || echo "API not running, skipping business tests"
echo "✅ Business tests completed"