#!/bin/bash
set -e

echo "=== Unit Tests for Picker Wheel ==="

# Test Go API
if [ -d api ]; then
  echo "Testing Go API..."
  cd api
  go test -v ./... -short -cover || { echo "❌ Go unit tests failed"; exit 1; }
  cd ..
  echo "✅ Go unit tests completed"
else
  echo "⚠️ No API directory found, skipping Go unit tests"
fi

# Test CLI commands
echo "Testing CLI commands..."
if [ -x cli/picker-wheel ]; then
  cli/picker-wheel help &> /dev/null || { echo "❌ CLI help command failed"; exit 1; }
  echo "✅ CLI tests completed"
else
  echo "⚠️ CLI not executable, skipping CLI tests"
fi

echo "✅ All unit tests passed"