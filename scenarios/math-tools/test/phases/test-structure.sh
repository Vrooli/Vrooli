#!/bin/bash
# Structure tests for math-tools scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running structure tests for math-tools..."

# Check required files exist
testing::phase::step "Checking scenario structure"

REQUIRED_FILES=(
    "api/cmd/server/main.go"
    "api/cmd/server/main_test.go"
    "api/cmd/server/test_helpers.go"
    "api/cmd/server/test_patterns.go"
    ".vrooli/service.json"
    "test/phases/test-unit.sh"
)

MISSING_FILES=()
for file in "${REQUIRED_FILES[@]}"; do
    if [[ ! -f "$file" ]]; then
        MISSING_FILES+=("$file")
        echo "✗ Missing required file: $file"
    else
        echo "✓ Found: $file"
    fi
done

if [[ ${#MISSING_FILES[@]} -gt 0 ]]; then
    echo "Error: Missing ${#MISSING_FILES[@]} required files"
    testing::phase::end_with_summary "Structure tests failed"
    exit 1
fi

# Check Go code compiles
testing::phase::step "Checking Go code compiles"
if command -v go &> /dev/null; then
    cd api/cmd/server
    if go build -o /dev/null . 2>&1; then
        echo "✓ Go code compiles successfully"
    else
        echo "✗ Go code compilation failed"
        testing::phase::end_with_summary "Structure tests failed (compilation error)"
        exit 1
    fi
    cd ../../..
fi

testing::phase::end_with_summary "Structure tests completed successfully"
