#!/bin/bash
set -e
echo "=== Integration Tests ==="
if [ -f test/integration-test.sh ]; then
  ./test/integration-test.sh
  echo "✅ Integration tests completed"
else
  echo "No integration tests found, skipping"
fi