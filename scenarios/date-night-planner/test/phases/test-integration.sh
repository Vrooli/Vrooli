#!/bin/bash
set -e
echo "=== Integration Tests ==="
# Add integration tests, e.g., API calls
curl -f http://localhost:8080/api/v1/health || { echo "API health check failed"; exit 1; }
echo "Integration tests passed"
exit 0