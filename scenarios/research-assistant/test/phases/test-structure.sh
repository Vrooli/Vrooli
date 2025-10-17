#!/bin/bash
set -euo pipefail

echo "=== Testing Structure ==="

# Check required directories and files
required_dirs=("api" "ui" "cli" "initialization")
for dir in "${required_dirs[@]}"; do
  if [ ! -d "$dir" ]; then
    echo "❌ Missing required directory: $dir"
    exit 1
  fi
done

required_files=("api/main.go" "ui/server.js" "cli/research-assistant")
for file in "${required_files[@]}"; do
  if [ ! -f "$file" ]; then
    echo "❌ Missing required file: $file"
    exit 1
  fi
done

# Check Go structure
if [ -d "api" ]; then
  cd api
  if ! go mod tidy -v > /dev/null 2>&1; then
    echo "⚠️ Go modules need tidying"
  fi
  cd ..
fi

# Check UI package.json if exists
if [ -f "ui/package.json" ]; then
  cd ui
  npm install --dry-run > /dev/null 2>&1 || echo "⚠️ UI dependencies may need update"
  cd ..
fi

echo "✅ Structure tests passed"