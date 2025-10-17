#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Checking scenario structure..."

# Check required directories
REQUIRED_DIRS=(
    "api"
    "cli"
    "ui"
    "test"
    "test/phases"
)

for dir in "${REQUIRED_DIRS[@]}"; do
    if [ ! -d "$dir" ]; then
        echo "❌ Required directory missing: $dir"
        exit 1
    else
        echo "✓ Found directory: $dir"
    fi
done

# Check required files
REQUIRED_FILES=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "Makefile"
    "api/main.go"
    "cli/palette-gen"
    "test/phases/test-unit.sh"
    "test/phases/test-integration.sh"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ Required file missing: $file"
        exit 1
    else
        echo "✓ Found file: $file"
    fi
done

# Check test files
echo "Checking test files..."
TEST_FILES=(
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/main_test.go"
    "api/performance_test.go"
    "api/systematic_test.go"
)

for file in "${TEST_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo "⚠️  Test file missing: $file"
    else
        echo "✓ Found test file: $file"
    fi
done

# Verify service.json structure
echo "Validating service.json..."
if ! jq -e '.service.name == "palette-gen"' .vrooli/service.json &> /dev/null; then
    echo "❌ service.json validation failed"
    exit 1
fi
echo "✓ service.json is valid"

# Check Go package structure
echo "Checking Go package structure..."
cd api
if ! go list ./... &> /dev/null; then
    echo "❌ Go package structure is invalid"
    exit 1
fi
echo "✓ Go package structure is valid"

# Check for proper main package
if ! grep -q "^package main" main.go; then
    echo "❌ main.go should be in package main"
    exit 1
fi
echo "✓ main.go has correct package declaration"

# Check CLI is executable
echo "Checking CLI executable..."
if [ ! -x "../cli/palette-gen" ]; then
    echo "⚠️  CLI is not executable, fixing..."
    chmod +x ../cli/palette-gen
fi
echo "✓ CLI is executable"

testing::phase::end_with_summary "Structure tests completed"
