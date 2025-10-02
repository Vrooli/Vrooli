#!/bin/bash

set -e

echo "=== Structure Tests ==="

# Check required directories
echo "Checking directory structure..."

required_dirs=(
    ".vrooli"
    "api"
    "cli"
    "initialization"
    "initialization/postgres"
    "test"
    "test/phases"
)

for dir in "${required_dirs[@]}"; do
    if [ -d "$dir" ]; then
        echo "✅ Directory exists: $dir"
    else
        echo "❌ Missing directory: $dir"
        exit 1
    fi
done

# Check required files
echo "Checking required files..."

required_files=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "Makefile"
    "api/main.go"
    "api/go.mod"
    "cli/install.sh"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ File exists: $file"
    else
        echo "❌ Missing file: $file"
        exit 1
    fi
done

# Check scenario-test.yaml exists (legacy compatibility)
if [ -f "scenario-test.yaml" ]; then
    echo "✅ Legacy scenario-test.yaml exists"
else
    echo "❌ Missing scenario-test.yaml (legacy support)"
    exit 1
fi

echo "✅ All structure tests passed"
