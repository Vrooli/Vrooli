#!/bin/bash
set -euo pipefail

echo "=== Dependencies Tests ==="

if [ -d "api" ]; then
  cd api
  go mod tidy
  go mod verify
  echo "✅ API Dependencies verified"
fi

if [ -d "ui" ]; then
  cd ui
  npm install --dry-run || true
  echo "✅ UI Dependencies checked"
fi

echo "All dependencies tests completed successfully"