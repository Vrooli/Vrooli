#!/bin/bash
set -euo pipefail

echo "🧪 Running unit tests"

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [ -d "$BASE_DIR/api" ]; then
  cd "$BASE_DIR/api"
  if ! go test ./... -short -v; then
    echo "❌ Unit tests failed"
    exit 1
  fi
  cd "$BASE_DIR"
fi

# CLI unit tests if any
if [ -d "$BASE_DIR/cli" ]; then
  cd "$BASE_DIR/cli"
  if [ -f "go.mod" ]; then
    go test ./... -short -v || exit 1
  fi
  cd "$BASE_DIR"
fi

echo "✅ Unit tests passed"