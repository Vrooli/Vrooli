#!/bin/bash
# Structure Test Phase for idea-generator
# Validates file structure, naming conventions, and required files

set -euo pipefail

echo "=== Structure Phase (Target: <15s) ==="

cd "$(dirname "$0")/../.."
SCENARIO_NAME="idea-generator"

required_files=(
  "api/main.go"
  "api/idea-generator-api"
  "ui/server.js"
  "ui/index.html"
  ".vrooli/service.json"
  "Makefile"
  "PRD.md"
  "README.md"
  "PROBLEMS.md"
  "cli/idea-generator"
)

missing=0
for file in "${required_files[@]}"; do
  if [ ! -f "$file" ] && [ ! -L "$file" ]; then
    echo "❌ Missing required file: $file"
    missing=1
  fi
done

# Check required directories
required_dirs=(
  "api"
  "ui"
  "cli"
  "test/phases"
  "initialization/storage/postgres"
)

for dir in "${required_dirs[@]}"; do
  if [ ! -d "$dir" ]; then
    echo "❌ Missing required directory: $dir"
    missing=1
  fi
done

if [ $missing -eq 0 ]; then
  echo "✅ Structure validation completed"
else
  exit 1
fi
