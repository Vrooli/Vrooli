#!/bin/bash
set -e
echo "=== Structure Tests ==="
# Check required files and directories
required_files=( "api/main.go" "ui/package.json" ".vrooli/service.json" "Makefile" "README.md" "PRD.md" "cli/workflow-scheduler" )
for file in "${required_files[@]}"; do
  if [ ! -f "$file" ]; then
    echo "❌ Missing required file: $file"
    exit 1
  fi
done
echo "✅ Structure tests passed"