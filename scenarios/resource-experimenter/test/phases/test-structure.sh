#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "=== Structure Testing for resource-experimenter ==="

# Check required directories
echo "✓ Checking directory structure..."
REQUIRED_DIRS=("api" "ui" "cli" "initialization" "test" ".vrooli")
for dir in "${REQUIRED_DIRS[@]}"; do
    if [ ! -d "$dir" ]; then
        testing::phase::error "Required directory '$dir' not found"
    else
        echo "  ✓ Directory: $dir"
    fi
done

# Check service configuration
echo "✓ Checking service configuration..."
if [ ! -f ".vrooli/service.json" ]; then
    testing::phase::error "service.json not found"
fi

# Validate service.json structure
if command -v jq &> /dev/null; then
    if ! jq empty .vrooli/service.json 2>/dev/null; then
        testing::phase::error "service.json is not valid JSON"
    fi

    # Check required fields
    REQUIRED_FIELDS=("service.name" "service.version" "ports" "resources" "lifecycle")
    for field in "${REQUIRED_FIELDS[@]}"; do
        if ! jq -e ".$field" .vrooli/service.json &>/dev/null; then
            testing::phase::error "Required field '$field' missing in service.json"
        fi
    done
fi

# Check API structure
echo "✓ Checking API structure..."
if [ ! -f "api/main.go" ]; then
    testing::phase::error "api/main.go not found"
fi
if [ ! -f "api/go.mod" ]; then
    testing::phase::error "api/go.mod not found"
fi

# Check for test files
TEST_FILES=("api/main_test.go" "api/test_helpers.go" "api/test_patterns.go")
for test_file in "${TEST_FILES[@]}"; do
    if [ ! -f "$test_file" ]; then
        testing::phase::error "Required test file '$test_file' not found"
    else
        echo "  ✓ Test file: $test_file"
    fi
done

# Check UI structure
if [ -f "ui/package.json" ]; then
    echo "✓ Checking UI structure..."
    if [ ! -f "ui/server.js" ] && [ ! -f "ui/src/index.js" ]; then
        testing::phase::warn "UI entry point not found"
    fi
fi

# Check test phase structure
echo "✓ Checking test phase structure..."
if [ ! -d "test/phases" ]; then
    testing::phase::error "test/phases directory not found"
fi
if [ ! -f "test/phases/test-unit.sh" ]; then
    testing::phase::error "test-unit.sh not found"
fi

# Check Makefile
if [ -f "Makefile" ]; then
    echo "✓ Makefile exists"
    # Check for required targets
    REQUIRED_TARGETS=("start" "stop" "test")
    for target in "${REQUIRED_TARGETS[@]}"; do
        if ! grep -q "^$target:" Makefile; then
            testing::phase::warn "Makefile target '$target' not found"
        fi
    done
fi

# Check documentation
echo "✓ Checking documentation..."
if [ ! -f "README.md" ]; then
    testing::phase::warn "README.md not found"
fi
if [ ! -f "PRD.md" ]; then
    testing::phase::warn "PRD.md not found"
fi

# Check lifecycle compliance
echo "✓ Checking lifecycle compliance..."
if grep -q "VROOLI_LIFECYCLE_MANAGED" api/main.go; then
    echo "  ✓ API enforces lifecycle management"
else
    testing::phase::warn "API does not enforce lifecycle management"
fi

testing::phase::end_with_summary "Structure tests completed"
