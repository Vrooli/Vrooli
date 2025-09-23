#!/bin/bash
set -e
echo "=== Integration Tests ==="
# Run integration tests if available
if [ -f integration/run-tests.sh ]; then
  ./integration/run-tests.sh
else
  echo "No integration tests found, skipping"
fi
echo "✅ Integration tests phase completed"