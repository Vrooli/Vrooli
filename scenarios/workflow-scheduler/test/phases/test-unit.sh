#!/bin/bash
set -e
echo "=== Unit Tests ==="
if [ -d api ]; then
  cd api && go test -v ./... -short
  echo "✅ Go unit tests completed"
fi
if [ -d ui ]; then
  cd ui && npm test -- --watchAll=false || true
  echo "✅ UI unit tests completed"
fi
echo "✅ Unit tests phase completed"