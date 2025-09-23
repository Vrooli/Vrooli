#!/bin/bash
set -e
echo "=== Structure Tests ==="
required_files=( "api/main.go" ".vrooli/service.json" "Makefile" "cli/file-tools" )
for file in "${required_files[@]}"; do
  if [ ! -f "$file" ]; then
    echo "❌ Missing required file: $file"
    exit 1
  fi
done
required_dirs=( "api" "cli" "initialization" )
for dir in "${required_dirs[@]}"; do
  if [ ! -d "$dir" ]; then
    echo "❌ Missing required directory: $dir"
    exit 1
  fi
done
echo "✅ Structure tests passed"