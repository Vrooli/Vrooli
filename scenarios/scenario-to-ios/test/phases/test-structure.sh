#!/bin/bash
set -e

echo "=== Structure Tests ==="

# Navigate to scenario root (we're running from test/phases)
cd "$(dirname "$0")/../.."

# Check required files
echo "Checking required files..."
if [ ! -f .vrooli/service.json ]; then
  echo "❌ Missing .vrooli/service.json"
  exit 1
fi
echo "✅ .vrooli/service.json found"

if [ ! -f api/main.go ]; then
  echo "❌ Missing api/main.go"
  exit 1
fi
echo "✅ api/main.go found"

if [ ! -f PRD.md ]; then
  echo "❌ Missing PRD.md"
  exit 1
fi
echo "✅ PRD.md found"

if [ ! -f README.md ]; then
  echo "❌ Missing README.md"
  exit 1
fi
echo "✅ README.md found"

echo "✅ All structure tests completed"