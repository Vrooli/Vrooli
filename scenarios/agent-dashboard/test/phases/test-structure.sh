#!/bin/bash
set -e

echo "=== Structure Tests ==="
# Check required files and directories exist
required_files=("Makefile" "README.md" ".vrooli/service.json")
for file in "${required_files[@]}"; do
  if [ ! -f "$file" ]; then
    echo "❌ Missing required file: $file"
    exit 1
  fi
done
echo "✅ Structure tests passed"