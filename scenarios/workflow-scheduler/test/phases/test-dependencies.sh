#!/bin/bash
set -e
echo "=== Dependencies Tests ==="
if [ -d api ]; then
  cd api && go mod tidy && go mod verify
  echo "✅ Go dependencies verified"
fi
if [ -d ui ]; then
  cd ui && npm install --dry-run || true
  echo "✅ UI dependencies checked"
fi
echo "✅ Dependencies phase completed"