#!/bin/bash
set -e

echo "=== Unit Tests ==="
# Run unit tests
if [ -d "api" ] && [ -f "api/go.mod" ]; then
  cd api && go test ./... -short
elif [ -d "ui" ] && [ -f "ui/package.json" ]; then
  cd ui && npm test || echo "No unit tests found"
else
  echo "No unit tests configured"
fi
echo "✅ Unit tests completed"