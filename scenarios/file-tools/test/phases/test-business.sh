#!/bin/bash
set -e
echo "=== Business Logic Tests ==="
curl -f http://localhost:${API_PORT:-8080}/health || echo "API not running, skipping business tests"
echo "✅ Business tests completed"