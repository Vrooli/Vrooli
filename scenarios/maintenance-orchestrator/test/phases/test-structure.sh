#!/bin/bash
set -euo pipefail

echo "=== Structure Tests ==="

# Check required files
required_files=(
  ".vrooli/service.json"
  "PRD.md"
  "README.md"
  "Makefile"
  "api/main.go"
  "cli/maintenance-orchestrator"
  "cli/install.sh"
  "ui/index.html"
  "ui/server.js"
)

failed=0
for file in "${required_files[@]}"; do
  if [ -f "$file" ]; then
    echo "✅ $file"
  else
    echo "❌ Missing: $file"
    failed=1
  fi
done

# Warn about legacy files
if [ -f "scenario-test.yaml" ]; then
  echo "⚠️  Legacy scenario-test.yaml found - consider migrating to phased testing"
fi

# Check API structure
if [ -d "api" ]; then
  echo "✅ API directory exists"
  go_files=$(find api -name "*.go" | wc -l)
  echo "   Found $go_files Go files"
fi

# Check CLI structure
if [ -d "cli" ]; then
  echo "✅ CLI directory exists"
fi

# Check UI structure
if [ -d "ui" ]; then
  echo "✅ UI directory exists"
  if [ -f "ui/package.json" ]; then
    echo "   Found package.json"
  fi
fi

exit $failed
