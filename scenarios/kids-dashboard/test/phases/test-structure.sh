#!/bin/bash

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running structure validation tests for kids-dashboard..."

# Test 1: Verify required directory structure
testing::phase::log "Checking directory structure..."
REQUIRED_DIRS=("api" "ui" ".vrooli" "test/phases")
for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        testing::phase::log "✓ $dir exists"
    else
        testing::phase::log "❌ Missing required directory: $dir"
    fi
done

# Test 2: Verify required files
testing::phase::log "Checking required files..."
REQUIRED_FILES=(
    ".vrooli/service.json"
    "api/main.go"
    "api/go.mod"
    "test/phases/test-unit.sh"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        testing::phase::log "✓ $file exists"
    else
        testing::phase::log "❌ Missing required file: $file"
    fi
done

# Test 3: Verify Go code structure
testing::phase::log "Checking Go code structure..."
cd api

# Check for main package
if grep -q "^package main" main.go 2>/dev/null; then
    testing::phase::log "✓ Main package defined"
fi

# Check for required types/structs
if grep -q "type Scenario struct" main.go 2>/dev/null; then
    testing::phase::log "✓ Scenario struct defined"
fi

if grep -q "type ServiceConfig struct" main.go 2>/dev/null; then
    testing::phase::log "✓ ServiceConfig struct defined"
fi

# Check for test files
TEST_FILES=$(find . -name "*_test.go" | wc -l)
testing::phase::log "Found $TEST_FILES test files"

if [ $TEST_FILES -ge 3 ]; then
    testing::phase::log "✓ Adequate test coverage files"
else
    testing::phase::log "⚠️  Limited test files (found $TEST_FILES)"
fi

# Test 4: Check test helper patterns
testing::phase::log "Checking test infrastructure..."
if [ -f test_helpers.go ]; then
    testing::phase::log "✓ test_helpers.go exists"

    # Check for key helper functions
    if grep -q "setupTestLogger" test_helpers.go; then
        testing::phase::log "✓ setupTestLogger helper found"
    fi

    if grep -q "setupTestDirectory" test_helpers.go; then
        testing::phase::log "✓ setupTestDirectory helper found"
    fi

    if grep -q "makeHTTPRequest" test_helpers.go; then
        testing::phase::log "✓ makeHTTPRequest helper found"
    fi
fi

if [ -f test_patterns.go ]; then
    testing::phase::log "✓ test_patterns.go exists"
fi

# Test 5: Verify UI structure
cd "$TESTING_PHASE_SCENARIO_DIR/ui"

if [ -f package.json ]; then
    testing::phase::log "✓ package.json exists"

    # Check for required scripts
    if jq -e '.scripts.dev' package.json > /dev/null 2>&1; then
        testing::phase::log "✓ dev script defined"
    fi

    if jq -e '.scripts.build' package.json > /dev/null 2>&1; then
        testing::phase::log "✓ build script defined"
    fi
fi

# Check for React/Vue components
COMPONENT_COUNT=$(find src -name "*.jsx" -o -name "*.tsx" -o -name "*.vue" 2>/dev/null | wc -l)
testing::phase::log "Found $COMPONENT_COUNT UI components"

# Test 6: Verify service.json structure
cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Validating service.json structure..."
if [ -f .vrooli/service.json ]; then
    # Check required fields
    FIELDS=("service.name" "service.description" "category" "ports" "lifecycle")
    for field in "${FIELDS[@]}"; do
        if jq -e ".$field" .vrooli/service.json > /dev/null 2>&1; then
            testing::phase::log "✓ $field defined"
        else
            testing::phase::log "⚠️  $field missing"
        fi
    done

    # Check lifecycle steps
    if jq -e '.lifecycle.setup' .vrooli/service.json > /dev/null 2>&1; then
        testing::phase::log "✓ setup lifecycle defined"
    fi

    if jq -e '.lifecycle.develop' .vrooli/service.json > /dev/null 2>&1; then
        testing::phase::log "✓ develop lifecycle defined"
    fi

    if jq -e '.lifecycle.test' .vrooli/service.json > /dev/null 2>&1; then
        testing::phase::log "✓ test lifecycle defined"
    fi
fi

testing::phase::end_with_summary "Structure validation completed"
