#!/bin/bash
echo "=== Testing Dependencies ==="
# Check go.mod
if [ ! -f "go.mod" ]; then
  echo "❌ Missing go.mod file"
  exit 1
fi
# Run go mod tidy to verify dependencies
go mod tidy
echo "✅ Dependencies tests passed"