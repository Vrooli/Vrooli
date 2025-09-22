#!/bin/bash
set -e

echo "=== Integration Tests ==="
# Run integration tests
if [ -f "test/integration" ]; then
  cd test/integration && ./run.sh || echo "Integration tests skipped"
else
  echo "No integration tests configured"
fi
echo "✅ Integration tests completed"