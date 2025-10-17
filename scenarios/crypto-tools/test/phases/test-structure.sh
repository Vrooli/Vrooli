#!/bin/bash

set -e

echo "=== Structure Tests ==="

# Check required directories
echo "Checking directory structure..."

required_dirs=(
    ".vrooli"
    "api"
    "cli"
    "api/cmd/server"
    "ui"
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
    "PROBLEMS.md"
    "Makefile"
    "api/main.go"
    "api/go.mod"
    "api/cmd/server/main.go"
    "cli/install.sh"
    "test/run-tests.sh"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ File exists: $file"
    else
        echo "❌ Missing file: $file"
        exit 1
    fi
done

# Check service.json structure
echo "Validating service.json structure..."

if [ -f ".vrooli/service.json" ]; then
    # Check for required fields
    if jq -e '.service.name' .vrooli/service.json >/dev/null 2>&1; then
        echo "✅ service.name field exists"
    else
        echo "❌ Missing service.name field"
        exit 1
    fi

    if jq -e '.lifecycle' .vrooli/service.json >/dev/null 2>&1; then
        echo "✅ lifecycle field exists"
    else
        echo "❌ Missing lifecycle field"
        exit 1
    fi
else
    echo "❌ service.json not found"
    exit 1
fi

# Check executable permissions
echo "Checking executable permissions..."

executable_files=(
    "test/run-tests.sh"
    "cli/install.sh"
)

for file in "${executable_files[@]}"; do
    if [ -x "$file" ]; then
        echo "✅ Executable: $file"
    else
        echo "⚠️  Not executable: $file"
        chmod +x "$file" 2>/dev/null || echo "❌ Cannot make executable: $file"
    fi
done

echo "=== Structure Tests Complete ==="
