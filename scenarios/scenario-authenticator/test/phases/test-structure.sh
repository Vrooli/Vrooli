#!/bin/bash
set -euo pipefail

echo "=== Running test-structure.sh ==="

# Structure checks for scenario-authenticator
required_dirs=(
  "api"
  "cli"
  "ui"
  "test"
  "initialization/postgres"
  "initialization/redis"
  "data/keys"
)

required_files=(
  ".vrooli/service.json"
  "PRD.md"
  "README.md"
  "api/main.go"
  "api/go.mod"
  "cli/scenario-authenticator"
  "initialization/postgres/schema.sql"
  "initialization/postgres/seed.sql"
  "ui/index.html"
  "ui/server.js"
  "ui/package.json"
)

echo "Checking required directories..."
for dir in "${required_dirs[@]}"; do
  if [ -d "$dir" ]; then
    echo "✓ Directory $dir exists"
  else
    echo "✗ Directory $dir missing"
    exit 1
  fi
done

echo ""
echo "Checking required files..."
for file in "${required_files[@]}"; do
  if [ -f "$file" ]; then
    echo "✓ File $file exists"
  else
    echo "✗ File $file missing"
    exit 1
  fi
done

echo ""
echo "Checking for legacy test files..."
if [ -f "scenario-test.yaml" ]; then
  echo "⚠ Legacy scenario-test.yaml found - consider migrating to phased tests"
else
  echo "✓ No legacy test files found"
fi

echo ""
echo "✅ test-structure.sh completed successfully"
