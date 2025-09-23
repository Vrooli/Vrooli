#!/bin/bash
set -e

echo "=== Running API Library tests ==="

if [ -d api ]; then
  cd api && go test ./... -v || { echo "API tests failed"; exit 1; }
fi

if [ -d ui ]; then
  cd ui && npm test || { echo "UI tests failed"; exit 1; }
fi

echo "✅ All tests passed"
