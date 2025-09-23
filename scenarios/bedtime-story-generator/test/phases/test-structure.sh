#!/bin/bash
set -e

echo "=== Phase 1: Structure Tests ==="

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

required_dirs=("api" "ui" "cli" "initialization" "test")
required_files=("api/go.mod" "ui/package.json" "cli/install.sh" ".gitignore" "Makefile" ".vrooli/service.json")

for dir in "${required_dirs[@]}"; do
  if [ ! -d "$SCENARIO_DIR/$dir" ]; then
    echo "❌ Missing required directory: $dir"
    exit 1
  fi
done

for file in "${required_files[@]}"; do
  if [ ! -f "$SCENARIO_DIR/$file" ]; then
    echo "❌ Missing required file: $file"
    exit 1
  fi
done

echo "✅ Structure tests passed"
