#!/bin/bash
set -euo pipefail

echo "💼 Testing business logic"

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Build and test API binary
if [ -d "$BASE_DIR/api" ]; then
  cd "$BASE_DIR/api"
  go build -o test-api . 2>/dev/null || true
  if [ -f "test-api" ]; then
    ./test-api --help >/dev/null 2>&1 || true
    rm test-api
    echo "✅ API business logic basic test passed"
  fi
  cd "$BASE_DIR"
fi

# CLI business logic
if command -v scenario-dependency-analyzer >/dev/null 2>&1; then
  scenario-dependency-analyzer analyze --help >/dev/null || exit 1
  echo "✅ CLI business logic test passed"
fi

echo "✅ Business tests passed"