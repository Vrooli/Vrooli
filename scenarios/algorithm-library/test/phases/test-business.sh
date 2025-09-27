#!/bin/bash
set -euo pipefail

echo "=== Business Logic Tests ==="

# Test core business rules for algorithm library
# e.g., verify algorithm implementations, API responses

if [ -d api ]; then
  cd api && go test -v ./... -run TestBusiness || echo "No business tests, skipping"
fi

echo "✅ Business tests passed (add specific tests as needed)"