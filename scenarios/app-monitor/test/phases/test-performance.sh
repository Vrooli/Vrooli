#!/bin/bash
set -e
echo "=== Performance Tests ==="
# Basic performance check, e.g. response time
if command -v hey >/dev/null; then
  hey -n 10 -c 1 http://localhost:${API_PORT:-8080}/health
else
  echo "hey not installed, skipping load test"
fi
echo "✅ Performance tests completed"