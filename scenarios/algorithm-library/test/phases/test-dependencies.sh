#!/bin/bash
set -euo pipefail

echo "=== Dependency Checks ==="

if [ -d api ]; then
  cd api && go mod tidy && go mod verify || { echo "Dependency check failed ❌"; exit 1; }
  echo "✅ Go dependencies OK"
fi

if [ -d ui ] && [ -f ui/package.json ]; then
  cd ui && npm install --dry-run || { echo "UI dependency check failed ❌"; exit 1; }
  echo "✅ UI dependencies OK"
fi

echo "✅ All dependencies verified"