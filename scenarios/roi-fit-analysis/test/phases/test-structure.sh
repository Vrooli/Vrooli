#!/bin/bash
# Structure test phase for roi-fit-analysis
# Validates project structure and configuration

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "📁 Running structure tests for roi-fit-analysis..."

# Test 1: Required files exist
echo "  ✓ Checking required files..."
required_files=(
    "service.json"
    ".vrooli/service.json"
    "api/main.go"
    "api/roi_engine.go"
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/main_test.go"
    "api/roi_engine_test.go"
    "test/phases/test-unit.sh"
    "test/phases/test-integration.sh"
    "test/phases/test-performance.sh"
)

for file in "${required_files[@]}"; do
    if [[ ! -f "$file" ]]; then
        testing::phase::fail "Required file missing: $file"
    fi
done

# Test 2: Service configuration is valid JSON
echo "  ✓ Validating service configuration..."
if ! jq empty .vrooli/service.json 2>/dev/null; then
    testing::phase::fail "Invalid JSON in .vrooli/service.json"
fi

# Test 3: Go code compiles
echo "  ✓ Checking Go code compilation..."
cd api
if ! go build -o /dev/null . 2>/dev/null; then
    testing::phase::fail "Go code does not compile"
fi

# Test 4: Test files are properly structured
echo "  ✓ Validating test file structure..."
test_files=$(find . -name "*_test.go" 2>/dev/null | wc -l)
if [[ $test_files -lt 3 ]]; then
    testing::phase::fail "Insufficient test files found (expected >= 3, got $test_files)"
fi

# Test 5: Required test functions exist
echo "  ✓ Checking for required test functions..."
if ! grep -q "func Test" main_test.go; then
    testing::phase::fail "No test functions found in main_test.go"
fi

if ! grep -q "func Test" roi_engine_test.go; then
    testing::phase::fail "No test functions found in roi_engine_test.go"
fi

# Test 6: Benchmark tests exist
echo "  ✓ Checking for benchmark tests..."
if ! grep -q "func Benchmark" main_test.go && ! grep -q "func Benchmark" roi_engine_test.go; then
    testing::phase::fail "No benchmark tests found"
fi

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::end_with_summary "Structure tests completed successfully"
