#!/bin/bash
set -euo pipefail
echo "=== Test Dependencies ==="
# Check binaries
if [ ! -f "api/scenario-surfer-api" ]; then
  echo "❌ API binary missing"
  exit 1
fi
# Check CLI
if [ ! -f "cli/scenario-surfer" ]; then
  echo "❌ CLI binary missing"
  exit 1
fi
# Check UI deps
if [ ! -d "ui/node_modules" ]; then
  echo "⚠️ UI dependencies not installed, running npm install"
  cd ui &amp;&amp; npm install
  cd ..
fi
echo "✅ Dependencies OK"
