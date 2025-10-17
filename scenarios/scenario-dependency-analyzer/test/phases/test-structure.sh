#!/bin/bash
set -euo pipefail

echo "🏗️  Testing project structure"

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

required_dirs=( "api" "cli" "ui" ".vrooli" "test" )
for dir in "${required_dirs[@]}"; do
  if [ ! -d "$BASE_DIR/$dir" ]; then
    echo "❌ Missing directory: $dir"
    exit 1
  fi
done

required_files=( "api/main.go" "cli/install.sh" ".vrooli/service.json" )
for file in "${required_files[@]}"; do
  if [ ! -f "$BASE_DIR/$file" ]; then
    echo "❌ Missing file: $file"
    exit 1
  fi
done

# Optional lint
if command -v golangci-lint >/dev/null 2>&1; then
  cd "$BASE_DIR/api" && golangci-lint run --fast || exit 1
  cd "$BASE_DIR"
fi

echo "✅ Structure tests passed"