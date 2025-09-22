#!/bin/bash

set -e

echo "=== Structure Tests ==="

# Test required directory structure
echo "Testing directory structure..."

# Check for required directories
required_dirs=(
    ".vrooli"
    "api"
    "ui"
    "cli"
    "docs"
    "initialization"
    "test"
    "test/phases"
)

for dir in "${required_dirs[@]}"; do
    if [ ! -d "$dir" ]; then
        echo "❌ Missing required directory: $dir"
        exit 1
    fi
done

# Check for required files
required_files=(
    ".vrooli/service.json"
    "api/main.go"
    "ui/package.json"
    "cli/install.sh"
    "Makefile"
    "README.md"
    "test/run-tests.sh"
)

for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ Missing required file: $file"
        exit 1
    fi
done

# Check for executable scripts
executable_scripts=(
    "test/run-tests.sh"
    "cli/install.sh"
)

for script in "${executable_scripts[@]}"; do
    if [ ! -x "$script" ]; then
        echo "❌ Script not executable: $script"
        exit 1
    fi
done

# Check service.json structure
echo "Testing service.json structure..."
if [ -f ".vrooli/service.json" ]; then
    # Check for required top-level keys
    required_keys=("lifecycle" "setup" "health")
    for key in "${required_keys[@]}"; do
        if ! jq -e ".$key" .vrooli/service.json > /dev/null 2>&1; then
            echo "❌ Missing required key in service.json: $key"
            exit 1
        fi
    done
else
    echo "❌ service.json not found"
    exit 1
fi

# Check Makefile targets
echo "Testing Makefile targets..."
if [ -f "Makefile" ]; then
    # Check for required targets
    required_targets=("start" "stop" "test" "logs" "clean" "help")
    for target in "${required_targets[@]}"; do
        if ! grep -q "^$target:" Makefile; then
            echo "❌ Missing required Makefile target: $target"
            exit 1
        fi
    done
else
    echo "❌ Makefile not found"
    exit 1
fi

echo "✅ Structure tests passed"

echo "=== Structure Tests Complete ==="