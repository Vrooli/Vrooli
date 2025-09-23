#!/bin/bash
set -e
echo "=== Running test-unit tests ==="
# Run unit tests
cd ../.. && if [ -d "api" ]; then cd api && go test ./... -short; fi
if [ -d "../../ui" ]; then cd ../../ui && npm test; fi
echo "✅ test-unit tests passed (placeholder)"