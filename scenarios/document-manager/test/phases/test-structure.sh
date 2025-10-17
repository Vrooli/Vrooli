#!/bin/bash
# Phase 1: Structure validation - <15 seconds
set -euo pipefail

echo "=== Phase 1: Structure Validation ==="
start_time=$(date +%s)
error_count=0
test_count=0

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

# Check required files
required_files=(
  ".vrooli/service.json"
  "README.md"
  "PRD.md"
  "Makefile"
  "api/main.go"
  "cli/document-manager"
)

for file in "${required_files[@]}"; do
  ((test_count++))
  if [ -f "$file" ]; then
    echo "✅ Found required file: $file"
  else
    echo "❌ Missing required file: $file"
    ((error_count++))
  fi
done

# Check for deprecated files
((test_count++))
if [ -f "scenario-test.yaml" ]; then
  echo "⚠️  Found deprecated scenario-test.yaml - migrated to phased testing"
else
  echo "✅ No deprecated test configuration files found"
fi

# Validate service.json structure
((test_count++))
if jq -e '.lifecycle.version' .vrooli/service.json >/dev/null 2>&1; then
  lifecycle_version=$(jq -r '.lifecycle.version' .vrooli/service.json)
  echo "✅ Service configuration valid (lifecycle v$lifecycle_version)"
else
  echo "❌ Invalid or missing lifecycle version in service.json"
  ((error_count++))
fi

# Check directory structure
required_dirs=(
  "api"
  "cli"
  "test"
  "ui"
  "initialization"
)

for dir in "${required_dirs[@]}"; do
  ((test_count++))
  if [ -d "$dir" ]; then
    echo "✅ Found required directory: $dir"
  else
    echo "⚠️  Missing directory: $dir"
  fi
done

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))

if [ $error_count -eq 0 ]; then
  echo "✅ Structure validation completed successfully in ${duration}s"
  echo "   Tests run: $test_count, Errors: $error_count"
  exit 0
else
  echo "❌ Structure validation failed with $error_count errors in ${duration}s"
  exit 1
fi
