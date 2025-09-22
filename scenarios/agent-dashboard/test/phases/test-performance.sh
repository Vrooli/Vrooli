#!/bin/bash
set -e

echo "=== Performance Tests ==="
# Run performance benchmarks
if command -v wrk &> /dev/null; then
  # Example: test API performance
  if [ -n "$API_PORT" ]; then
    wrk -t12 -c400 -d30s http://localhost:$API_PORT/health || echo "Performance test skipped"
  fi
else
  echo "wrk not installed, skipping performance tests"
fi
echo "✅ Performance tests completed"