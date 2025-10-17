#!/bin/bash
set -e
echo "=== Dependency Checks ==="
if [ -f api/go.mod ]; then cd api && go mod tidy && go mod verify; fi
if [ -f ui/package.json ]; then cd ui && npm install --dry-run; fi
echo "✅ Dependencies verified"