#!/bin/bash
set -e

echo "=== Structure Tests for Picker Wheel ==="

# Check required files
echo "Checking required files..."
REQUIRED_FILES=(
  ".vrooli/service.json"
  "api/main.go"
  "api/go.mod"
  "cli/picker-wheel"
  "ui/index.html"
  "ui/styles.css"
  "ui/script.js"
  "ui/server.js"
  "ui/package.json"
  "initialization/postgres/schema.sql"
  "PRD.md"
  "README.md"
  "Makefile"
)

for file in "${REQUIRED_FILES[@]}"; do
  if [ ! -f "$file" ]; then
    echo "❌ Missing required file: $file"
    exit 1
  fi
done
echo "✅ All required files present"

# Validate service.json
echo "Validating service.json..."
jq -e '.service.name == "picker-wheel"' .vrooli/service.json &> /dev/null || {
  echo "❌ Invalid service.json"
  exit 1
}
echo "✅ service.json is valid"

# Check lifecycle configuration
echo "Checking lifecycle configuration..."
jq -e '.lifecycle.version == "2.0.0"' .vrooli/service.json &> /dev/null || {
  echo "❌ Lifecycle version not 2.0.0"
  exit 1
}
echo "✅ Lifecycle v2.0 configured"

# Check Makefile targets
echo "Checking Makefile targets..."
REQUIRED_TARGETS=("run" "test" "stop" "status" "logs" "help")
for target in "${REQUIRED_TARGETS[@]}"; do
  grep -q "^${target}:" Makefile || {
    echo "❌ Missing Makefile target: $target"
    exit 1
  }
done
echo "✅ All required Makefile targets present"

# Validate SQL schema
echo "Validating PostgreSQL schema..."
grep -q "CREATE TABLE IF NOT EXISTS wheels" initialization/postgres/schema.sql || {
  echo "❌ Invalid PostgreSQL schema"
  exit 1
}
echo "✅ PostgreSQL schema valid"

echo "✅ All structure tests passed"