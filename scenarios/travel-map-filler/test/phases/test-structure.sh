#!/bin/bash
# Structure test phase for travel-map-filler
# Validates project structure and configuration

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🏗️  Validating project structure..."

EXIT_CODE=0

# Check required files
echo "Checking required files..."
REQUIRED_FILES=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "api/main.go"
    "api/test_helpers.go"
    "api/test_patterns.go"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file"
    else
        echo "❌ Missing: $file"
        EXIT_CODE=1
    fi
done

# Check test files
echo ""
echo "Checking test files..."
TEST_FILES=(
    "api/main_test.go"
    "api/comprehensive_test.go"
    "api/integration_test.go"
    "api/performance_test.go"
    "test/phases/test-unit.sh"
    "test/phases/test-integration.sh"
    "test/phases/test-performance.sh"
)

for file in "${TEST_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file"
    else
        echo "⚠️  Missing test file: $file"
    fi
done

# Validate service.json
echo ""
echo "Validating service.json..."
if [ -f ".vrooli/service.json" ]; then
    if command -v jq &> /dev/null; then
        jq empty .vrooli/service.json 2>&1
        if [ $? -eq 0 ]; then
            echo "✅ service.json is valid JSON"

            # Check required fields
            SERVICE_NAME=$(jq -r '.service.name' .vrooli/service.json)
            if [ "$SERVICE_NAME" = "travel-map-filler" ]; then
                echo "✅ Service name is correct"
            else
                echo "❌ Service name mismatch: $SERVICE_NAME"
                EXIT_CODE=1
            fi
        else
            echo "❌ service.json is invalid JSON"
            EXIT_CODE=1
        fi
    else
        echo "ℹ️  jq not available, skipping JSON validation"
    fi
fi

# Check Go project structure
echo ""
echo "Checking Go project structure..."
if [ -f "api/go.mod" ]; then
    echo "✅ go.mod exists"

    MODULE_NAME=$(grep -E "^module " api/go.mod | awk '{print $2}')
    echo "   Module: $MODULE_NAME"

    # Count Go files
    GO_FILES=$(find api -name "*.go" -not -name "*_test.go" | wc -l)
    TEST_FILES=$(find api -name "*_test.go" | wc -l)
    echo "   Source files: $GO_FILES"
    echo "   Test files: $TEST_FILES"

    if [ $TEST_FILES -eq 0 ]; then
        echo "⚠️  No test files found"
    fi
else
    echo "❌ go.mod not found in api directory"
    EXIT_CODE=1
fi

# Check initialization scripts
echo ""
echo "Checking initialization scripts..."
INIT_DIRS=(
    "initialization/postgres"
    "initialization/qdrant"
    "initialization/n8n"
)

for dir in "${INIT_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        FILE_COUNT=$(find "$dir" -type f | wc -l)
        echo "✅ $dir ($FILE_COUNT files)"
    else
        echo "ℹ️  Optional init directory not found: $dir"
    fi
done

# Validate test helper structure
echo ""
echo "Validating test helper structure..."
if [ -f "api/test_helpers.go" ]; then
    # Check for required helper functions
    HELPER_FUNCTIONS=(
        "setupTestLogger"
        "setupTestDB"
        "makeHTTPRequest"
        "assertJSONResponse"
        "assertErrorResponse"
    )

    for func in "${HELPER_FUNCTIONS[@]}"; do
        if grep -q "func $func" api/test_helpers.go; then
            echo "✅ Helper function: $func"
        else
            echo "⚠️  Missing helper function: $func"
        fi
    done
fi

# Check documentation
echo ""
echo "Checking documentation..."
if [ -f "PRD.md" ]; then
    LINES=$(wc -l < PRD.md)
    echo "✅ PRD.md ($LINES lines)"
else
    echo "❌ PRD.md missing"
    EXIT_CODE=1
fi

if [ -f "README.md" ]; then
    LINES=$(wc -l < README.md)
    echo "✅ README.md ($LINES lines)"
else
    echo "⚠️  README.md missing"
fi

testing::phase::end_with_summary "Structure validation completed"

exit $EXIT_CODE
