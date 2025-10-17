#!/bin/bash
set -e
echo "=== Test Structure ==="
# Check for required files and directories
required_files=("../api/main.go" "../Makefile" ".vrooli/service.json")
for file in "${required_files[@]}"; do
  if [ ! -f "$file" ]; then
    echo "❌ Missing required file: $file"
    exit 1
  fi
done
echo "✅ Structure tests passed"