#!/bin/bash
set -euo pipefail

echo "=== Running test-structure.sh ==="

# Structure checks

required_dirs=("issues/open" "issues/templates" "api" "ui" "cli" "test")
required_files=("api/main.go" "ui/index.html" "cli/app-issue-tracker.sh" "issues/README.md")

for dir in "${required_dirs[@]}"; do
  if [ -d "$dir" ]; then
    echo "✓ Directory $dir exists"
  else
    echo "✗ Directory $dir missing"
    exit 1
  fi
done

for file in "${required_files[@]}"; do
  if [ -f "$file" ]; then
    echo "✓ File $file exists"
  else
    echo "✗ File $file missing"
    exit 1
  fi
done

echo "✅ test-structure.sh completed successfully"
