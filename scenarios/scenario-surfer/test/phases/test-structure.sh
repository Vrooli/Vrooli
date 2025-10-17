#!/bin/bash
set -euo pipefail
echo "=== Test Structure ==="
required_files=(
  "api/main.go"
  "ui/index.html"
  "ui/server.js"
  ".vrooli/service.json"
  "Makefile"
)
for file in "${required_files[@]}"; do
  if [ ! -f "$file" ]; then
    echo "❌ Missing required file: $file"
    exit 1
  fi
done
required_dirs=( "api" "ui" "cli" )
for dir in "${required_dirs[@]}"; do
  if [ ! -d "$dir" ]; then
    echo "❌ Missing required directory: $dir"
    exit 1
  fi
done
echo "✅ Structure OK"
