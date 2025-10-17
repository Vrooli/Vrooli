#!/bin/bash
# Structure tests for scenario-to-desktop

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../..}" && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "15s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🏗️  Checking project structure..."

MISSING_FILES=()
REQUIRED_FILES=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "api/main.go"
    "api/go.mod"
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/main_test.go"
    "api/performance_test.go"
    "cli/scenario-to-desktop"
    "cli/install.sh"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [[ ! -f "$file" ]]; then
        MISSING_FILES+=("$file")
        echo "❌ Missing required file: $file"
    else
        echo "✅ Found: $file"
    fi
done

# Check for required directories
REQUIRED_DIRS=(
    "api"
    "cli"
    "templates"
    "test/phases"
)

for dir in "${REQUIRED_DIRS[@]}"; do
    if [[ ! -d "$dir" ]]; then
        MISSING_FILES+=("$dir/")
        echo "❌ Missing required directory: $dir"
    else
        echo "✅ Found directory: $dir"
    fi
done

# Verify service.json structure
if [[ -f ".vrooli/service.json" ]]; then
    echo "🔍 Validating service.json..."

    # Check for required fields using jq
    if command -v jq &> /dev/null; then
        REQUIRED_JSON_FIELDS=(
            ".service.name"
            ".version"
            ".description"
            ".type"
            ".ports.api"
            ".apis.endpoints"
        )

        for field in "${REQUIRED_JSON_FIELDS[@]}"; do
            if ! jq -e "$field" .vrooli/service.json &> /dev/null; then
                echo "⚠️  Missing or invalid field in service.json: $field"
            fi
        done

        echo "✅ service.json structure validated"
    else
        echo "⚠️  jq not found, skipping service.json validation"
    fi
fi

# Check test coverage
echo "📊 Checking test coverage..."
if [[ -f "api/main.go" ]]; then
    cd api

    # Count test functions
    TEST_COUNT=$(grep -c "^func Test" *_test.go 2>/dev/null || echo "0")
    echo "✅ Found $TEST_COUNT test functions"

    if [[ $TEST_COUNT -lt 10 ]]; then
        echo "⚠️  Low test count (recommended: 10+)"
    fi

    cd ..
fi

if [[ ${#MISSING_FILES[@]} -gt 0 ]]; then
    echo ""
    echo "❌ Structure validation failed. Missing ${#MISSING_FILES[@]} required files/directories:"
    printf '   - %s\n' "${MISSING_FILES[@]}"
    testing::phase::end_with_error "Project structure incomplete"
fi

testing::phase::end_with_summary "Structure validation completed"
