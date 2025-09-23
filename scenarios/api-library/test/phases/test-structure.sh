#!/bin/bash
set -euo pipefail

echo "=== Structure Tests ==="

# Check required files exist
required_files=("api/main.go" "ui/package.json" "Makefile" ".vrooli/service.json")

for file in "${required_files[@]}"; do
  if [ ! -f "$file" ]; then
    echo "❌ Missing required file: $file"
    exit 1
  fi
done

echo "✅ Structure tests passed"