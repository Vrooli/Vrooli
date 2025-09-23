#!/bin/bash
set -e
echo "=== Integration Tests ==="
# Run integration tests against running services
if command -v curl >/dev/null; then
  curl -f http://localhost:${API_PORT}/health || exit 1
  # Add more integration checks
fi
echo "✅ Integration tests completed"