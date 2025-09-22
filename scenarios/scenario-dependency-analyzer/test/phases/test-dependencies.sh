#!/bin/bash
set -euo pipefail

echo "🔗 Testing dependencies"

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [ -d "$BASE_DIR/api" ]; then
  cd "$BASE_DIR/api"
  go mod tidy
  go mod verify || exit 1
  cd "$BASE_DIR"
fi

echo "✅ Dependencies tests passed"