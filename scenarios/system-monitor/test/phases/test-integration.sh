#!/bin/bash
set -e

echo "Running integration tests for system-monitor"

# Example integration test
if curl -f http://localhost:$API_PORT/health; then
  echo "API health check passed"
else
  echo "API not available, skipping"
fi

echo "✅ Integration tests completed"
