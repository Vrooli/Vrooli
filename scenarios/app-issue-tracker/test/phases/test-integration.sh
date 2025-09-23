#!/bin/bash
set -euo pipefail

echo "=== Running test-integration.sh ==="

# Integration tests

if [ -f scripts/test-claude-integration.sh ]; then
  bash scripts/test-claude-integration.sh
  echo "✓ Claude integration test passed"
else
  echo "⚠ No integration test script, skipping"
fi

# Test API if possible
if command -v curl >/dev/null 2>&1 && [ -f api/main.go ]; then
  # Assume API starts on port 8080 for test, but skip if not running
  echo "⚠ Integration API test skipped (start API first)"
else
  echo "⚠ No curl or API, skipping HTTP integration"
fi

echo "✅ test-integration.sh completed successfully"
