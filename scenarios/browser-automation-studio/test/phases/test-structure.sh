#!/bin/bash
set -e

echo "=== Test Structure ==="

# Verify required directories exist
dirs=(api ui cli initialization docs)
for dir in "${dirs[@]}"; do
  if [[ ! -d "$dir" ]]; then
    echo "❌ Missing directory: $dir"
    exit 1
  fi
done

# Verify key files
files=(api/main.go ui/package.json ui/vite.config.ts cli/install.sh Makefile .vrooli/service.json)
for file in "${files[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "❌ Missing file: $file"
    exit 1
  fi
done

echo "✅ Structure tests passed"