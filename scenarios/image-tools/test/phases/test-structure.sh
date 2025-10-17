#!/bin/bash
# Test: Structure validation
# Ensures all required files and directories exist

set -e

echo "  ✓ Checking required files..."

required_files=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "api/main.go"
    "api/go.mod"
    "api/storage.go"
    "cli/image-tools"
    "cli/install.sh"
    "ui/index.html"
    "ui/app.js"
    "ui/styles.css"
    "ui/server.js"
    "Makefile"
    "test.sh"
)

for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "  ❌ Missing required file: $file"
        exit 1
    fi
done

echo "  ✓ Checking required directories..."

required_dirs=(
    "api"
    "cli"
    "ui"
    "test"
    "test/phases"
    "api/plugins"
    "api/plugins/jpeg"
    "api/plugins/png"
    "api/plugins/webp"
    "api/plugins/svg"
)

for dir in "${required_dirs[@]}"; do
    if [ ! -d "$dir" ]; then
        echo "  ❌ Missing required directory: $dir"
        exit 1
    fi
done

echo "  ✓ Structure validation complete"