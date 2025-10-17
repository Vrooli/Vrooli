#!/bin/bash

set -e

echo "=== Structure Tests ==="

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
    "cli/home-automation"
    "test/run-tests.sh"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file exists"
    else
        echo "❌ $file missing"
        exit 1
    fi
done

# Check required directories
echo "Checking required directories..."

required_dirs=(
    "api"
    "cli"
    "ui"
    "test"
    "test/phases"
    "initialization/postgres"
)

for dir in "${required_dirs[@]}"; do
    if [ -d "$dir" ]; then
        echo "✅ $dir exists"
    else
        echo "❌ $dir missing"
        exit 1
    fi
done

# Check test phases
echo "Checking test phases..."

required_phases=(
    "test/phases/test-unit.sh"
    "test/phases/test-api.sh"
    "test/phases/test-integration.sh"
    "test/phases/test-business.sh"
    "test/phases/test-dependencies.sh"
    "test/phases/test-performance.sh"
    "test/phases/test-structure.sh"
)

for phase in "${required_phases[@]}"; do
    if [ -f "$phase" ]; then
        if [ -x "$phase" ]; then
            echo "✅ $phase exists and is executable"
        else
            echo "⚠️  $phase exists but not executable"
            chmod +x "$phase"
            echo "✅ Fixed permissions on $phase"
        fi
    else
        echo "❌ $phase missing"
        exit 1
    fi
done

# Check service.json structure
echo "Checking service.json structure..."

if command -v jq >/dev/null 2>&1; then
    if jq -e '.service.name' .vrooli/service.json >/dev/null 2>&1; then
        echo "✅ service.json is valid JSON"

        # Check required fields
        required_fields=(
            ".service.name"
            ".service.version"
            ".lifecycle.version"
            ".lifecycle.health"
            ".lifecycle.setup"
            ".lifecycle.develop"
            ".lifecycle.test"
            ".lifecycle.stop"
        )

        for field in "${required_fields[@]}"; do
            if jq -e "$field" .vrooli/service.json >/dev/null 2>&1; then
                echo "✅ service.json has $field"
            else
                echo "❌ service.json missing $field"
                exit 1
            fi
        done
    else
        echo "❌ service.json is not valid JSON"
        exit 1
    fi
else
    echo "⚠️  jq not installed, skipping detailed service.json validation"
fi

# Check Makefile structure
echo "Checking Makefile structure..."

required_targets=(
    "help"
    "start"
    "stop"
    "test"
    "logs"
    "status"
)

for target in "${required_targets[@]}"; do
    if grep -q "^$target:" Makefile; then
        echo "✅ Makefile has $target target"
    else
        echo "❌ Makefile missing $target target"
        exit 1
    fi
done

# Check CLI structure
echo "Checking CLI structure..."

if [ -x "cli/home-automation" ]; then
    # Check if CLI has help command
    if ./cli/home-automation help >/dev/null 2>&1 || ./cli/home-automation --help >/dev/null 2>&1; then
        echo "✅ CLI has help command"
    else
        echo "⚠️  CLI may not have help command"
    fi
else
    echo "❌ CLI not executable"
    exit 1
fi

echo "✅ Structure tests completed"
