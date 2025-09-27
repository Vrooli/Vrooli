#!/bin/bash
set -euo pipefail

echo "=== Running test-business.sh ==="

# Business logic tests would go here
# For now, verify basic business flow

if [ -f data/issues/templates/bug-template.yaml ]; then
  echo "✓ Business templates exist"
else
  echo "✗ Business templates missing"
  exit 1
fi

echo "✅ test-business.sh completed successfully"
