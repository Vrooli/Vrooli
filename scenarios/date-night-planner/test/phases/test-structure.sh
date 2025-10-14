#!/bin/bash
set -euo pipefail

echo "=== Structure Checks ==="

SCENARIO_NAME="$(basename "$PWD")"
required_files=(
  "api/main.go"
  "api/date-night-planner-api"
  "ui/server.js"
  "ui/index.html"
  "ui/package.json"
  ".vrooli/service.json"
  "Makefile"
  "PRD.md"
  "README.md"
  "cli/date-night-planner"
  "initialization/storage/postgres/schema.sql"
)

required_dirs=(
  "api"
  "ui"
  "cli"
  "test/phases"
  "initialization/storage"
  "initialization/automation"
)

missing=0
for file in "${required_files[@]}"; do
  if [ ! -f "$file" ]; then
    echo "❌ Missing required file: $file"
    missing=1
  fi
done

for dir in "${required_dirs[@]}"; do
  if [ ! -d "$dir" ]; then
    echo "❌ Missing required directory: $dir"
    missing=1
  fi
done

if [ $missing -eq 0 ]; then
  echo "✅ Structure OK - all required files and directories present"
else
  exit 1
fi
