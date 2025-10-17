#!/bin/bash
set -euo pipefail

echo "=== Structure Checks ==="

SCENARIO_NAME="$(basename "$PWD")"
required_files=(
  "api/main.go"
  "ui/index.html"
  ".vrooli/service.json"
  "Makefile"
)

missing=0
for file in "${required_files[@]}"; do
  if [ ! -f "$file" ]; then
    echo "❌ Missing required file: $file"
    missing=1
  fi
done

if [ $missing -eq 0 ]; then
  echo "✅ Structure OK"
else
  exit 1
fi