#!/bin/bash
set -e
echo "=== Unit Tests ==="
if [ -d api ]; then
  cd api && go test -v ./... -short -cover
  echo "✅ Unit tests completed"
else
  echo "No API directory found, skipping unit tests"
fi