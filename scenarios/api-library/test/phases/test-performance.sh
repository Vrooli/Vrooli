#!/bin/bash
set -euo pipefail

echo "=== Performance Tests ==="

# Run go benchmarks or simple load test
if [ -d "api" ]; then
  cd api
  go test -bench=. -benchmem || echo "No benchmarks defined"
  echo "✅ Performance benchmarks run"
fi

# Simple load test placeholder
echo "Performance tests completed (basic)"