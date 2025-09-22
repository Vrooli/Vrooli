#!/bin/bash
set -e
echo "=== Test Dependencies ==="
# Check Go dependencies
if [ -f "../api/go.mod" ]; then
  cd ../api && go mod tidy && go mod verify
  echo "✅ Go dependencies OK"
else
  echo "❌ No go.mod found"
  exit 1
fi
# Check Makefile targets
if ! grep -q "run:" ../Makefile; then
  echo "❌ Missing run target in Makefile"
  exit 1
fi
echo "✅ Dependencies tests passed"