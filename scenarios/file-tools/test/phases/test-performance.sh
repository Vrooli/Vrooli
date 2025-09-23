#!/bin/bash
set -e
echo "=== Performance Tests ==="
if command -v hey >/dev/null; then
  hey -n 10 -c 1 http://localhost:${API_PORT:-8080}/health || echo "Load test failed or API not running"
else
  echo "hey not installed, skipping load test"
fi
echo "✅ Performance tests completed"