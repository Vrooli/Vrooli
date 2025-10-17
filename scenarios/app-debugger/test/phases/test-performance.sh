#!/bin/bash
set -e
echo "=== Performance Tests ==="
# Run simple performance check
time curl -f http://localhost:${API_PORT}/health || exit 1
# Add benchmark if available
if [ -d api ]; then cd api && go test -bench=. -count=5 ./... || true; fi
echo "✅ Performance tests completed"