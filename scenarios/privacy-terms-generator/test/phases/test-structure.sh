#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running structure validation tests..."

# Check for required files
required_files=(
    "PRD.md"
    "README.md"
    "Makefile"
    "api/main.go"
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/main_test.go"
    "api/performance_test.go"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✓ Required file exists: $file"
    else
        echo "✗ Missing required file: $file"
        exit 1
    fi
done

# Check for required directories
required_dirs=(
    "api"
    "test"
    "test/phases"
)

for dir in "${required_dirs[@]}"; do
    if [ -d "$dir" ]; then
        echo "✓ Required directory exists: $dir"
    else
        echo "✗ Missing required directory: $dir"
        exit 1
    fi
done

# Validate Go module structure
if [ -f "api/go.mod" ]; then
    echo "✓ Go module file exists"
    cd api
    if go mod verify &> /dev/null; then
        echo "✓ Go module is valid"
    else
        echo "⚠ Go module verification failed (may need dependencies)"
    fi
    cd ..
fi

# Check for test coverage tools
echo "Checking test infrastructure..."
if [ -f "test/phases/test-unit.sh" ]; then
    echo "✓ Unit test phase exists"
fi
if [ -f "test/phases/test-integration.sh" ]; then
    echo "✓ Integration test phase exists"
fi
if [ -f "test/phases/test-performance.sh" ]; then
    echo "✓ Performance test phase exists"
fi

testing::phase::end_with_summary "Structure validation completed"
