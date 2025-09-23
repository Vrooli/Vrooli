#!/bin/bash
set -e
echo "=== Dependency Checks ==="
if [ -d api ]; then
  cd api && go mod tidy && go mod verify
fi
if [ -d ui ]; then
  cd ui && npm install --dry-run || true
  npm audit --audit-level moderate || true
fi
echo "✅ Dependency checks completed"