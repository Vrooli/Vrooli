#!/bin/bash
echo "=== Testing Structure ==="
# Check required directories
required_dirs=("api" "test")
for dir in "${required_dirs[@]}"; do
  if [ ! -d "$dir" ]; then
    echo "❌ Missing required directory: $dir"
    exit 1
  fi
done
# Check for key files
required_files=("go.mod" "README.md")
for file in "${required_files[@]}"; do
  if [ ! -f "$file" ]; then
    echo "❌ Missing required file: $file"
    exit 1
  fi
done
echo "✅ Structure tests passed"