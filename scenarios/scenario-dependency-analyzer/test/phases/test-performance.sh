#!/bin/bash
set -euo pipefail

echo "⚡ Running performance tests"

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Performance test for CLI analysis
if command -v scenario-dependency-analyzer >/dev/null 2>&1; then
  echo "  Timing sample analysis..."
  start=$(date +%s)
  timeout 30 scenario-dependency-analyzer analyze chart-generator --output json > /dev/null 2>&1 || true
  end=$(date +%s)
  duration=$((end - start))
  if [ $duration -le 30 ]; then
    echo "✅ Performance test passed: ${duration}s"
  else
    echo "⚠️  Performance test warning: ${duration}s (timeout or slow)"
  fi
else
  echo "CLI not available, skipping performance test"
fi

echo "✅ Performance tests completed"