#!/bin/bash
set -euo pipefail

echo "=== Integration Tests ==="

if [ -f tests/test-judge0-integration.sh ]; then
  bash tests/test-judge0-integration.sh || { echo "Integration tests failed ❌"; exit 1; }
  echo "✅ Integration tests completed"
else
  echo "No integration tests found, skipping"
fi

# Additional integration tests can be added here