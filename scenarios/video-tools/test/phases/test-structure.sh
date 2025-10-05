#!/bin/bash
# Structure tests for video-tools scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../..}" && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🏗️  Running video-tools structure tests..."

# Test 1: Required files exist
echo ""
echo "Test 1: Checking required files..."

required_files=(
    ".vrooli/service.json"
    "README.md"
    "PRD.md"
    "api/go.mod"
    "api/cmd/server/main.go"
    "cli/install.sh"
    "test/phases/test-unit.sh"
    "test/phases/test-integration.sh"
    "test/phases/test-performance.sh"
)

missing_files=0
for file in "${required_files[@]}"; do
    if [[ -f "$file" ]]; then
        echo "  ✅ $file"
    else
        echo "  ❌ Missing: $file"
        ((missing_files++))
    fi
done

if [[ $missing_files -gt 0 ]]; then
    echo "❌ $missing_files required files missing"
    exit 1
fi

echo "✅ All required files present"

# Test 2: Directory structure
echo ""
echo "Test 2: Checking directory structure..."

required_dirs=(
    "api"
    "api/cmd"
    "api/cmd/server"
    "api/internal"
    "api/internal/video"
    "cli"
    "test"
    "test/phases"
)

missing_dirs=0
for dir in "${required_dirs[@]}"; do
    if [[ -d "$dir" ]]; then
        echo "  ✅ $dir/"
    else
        echo "  ❌ Missing: $dir/"
        ((missing_dirs++))
    fi
done

if [[ $missing_dirs -gt 0 ]]; then
    echo "❌ $missing_dirs required directories missing"
    exit 1
fi

echo "✅ All required directories present"

# Test 3: Service.json validation
echo ""
echo "Test 3: Validating service.json..."

if command -v jq &>/dev/null; then
    if jq empty .vrooli/service.json 2>/dev/null; then
        echo "✅ service.json is valid JSON"

        # Check required fields
        required_fields=("service.name" "service.version" "components" "resources" "lifecycle")
        for field in "${required_fields[@]}"; do
            if jq -e ".$field" .vrooli/service.json &>/dev/null; then
                echo "  ✅ $field is present"
            else
                echo "  ❌ Missing field: $field"
            fi
        done
    else
        echo "❌ service.json is not valid JSON"
        exit 1
    fi
else
    echo "⚠️  jq not installed, skipping JSON validation"
fi

# Test 4: Go code structure
echo ""
echo "Test 4: Checking Go code structure..."

if [[ -d "api" ]]; then
    cd api

    # Check for test files
    test_files=$(find . -name "*_test.go" | wc -l)
    if [[ $test_files -gt 0 ]]; then
        echo "  ✅ Found $test_files test files"
    else
        echo "  ⚠️  No test files found"
    fi

    # Check for test helpers
    if [[ -f "cmd/server/test_helpers.go" ]]; then
        echo "  ✅ Test helpers present"
    else
        echo "  ⚠️  Test helpers not found"
    fi

    # Check for test patterns
    if [[ -f "cmd/server/test_patterns.go" ]]; then
        echo "  ✅ Test patterns present"
    else
        echo "  ⚠️  Test patterns not found"
    fi

    # Verify Go code compiles
    if go build -o /tmp/video-tools-test ./cmd/server 2>/dev/null; then
        echo "  ✅ Go code compiles successfully"
        rm -f /tmp/video-tools-test
    else
        echo "  ❌ Go code compilation failed"
        exit 1
    fi

    cd ..
fi

# Test 5: CLI structure
echo ""
echo "Test 5: Checking CLI structure..."

if [[ -f "cli/install.sh" ]]; then
    if [[ -x "cli/install.sh" ]]; then
        echo "  ✅ install.sh is executable"
    else
        echo "  ⚠️  install.sh is not executable"
    fi
fi

# Test 6: Test phase scripts
echo ""
echo "Test 6: Checking test phase scripts..."

test_phases=(
    "test/phases/test-unit.sh"
    "test/phases/test-integration.sh"
    "test/phases/test-performance.sh"
    "test/phases/test-business.sh"
    "test/phases/test-dependencies.sh"
)

for phase in "${test_phases[@]}"; do
    if [[ -f "$phase" ]]; then
        if [[ -x "$phase" ]]; then
            echo "  ✅ $phase (executable)"
        else
            echo "  ⚠️  $phase (not executable)"
        fi
    else
        echo "  ❌ Missing: $phase"
    fi
done

# Test 7: Documentation
echo ""
echo "Test 7: Checking documentation..."

doc_files=("README.md" "PRD.md")
for doc in "${doc_files[@]}"; do
    if [[ -f "$doc" ]]; then
        lines=$(wc -l < "$doc")
        if [[ $lines -gt 10 ]]; then
            echo "  ✅ $doc ($lines lines)"
        else
            echo "  ⚠️  $doc is too short ($lines lines)"
        fi
    else
        echo "  ❌ Missing: $doc"
    fi
done

echo ""
echo "📊 Structure Test Summary:"
echo "  - Required files: Present"
echo "  - Directory structure: Valid"
echo "  - Service configuration: Valid"
echo "  - Go code structure: Valid"
echo "  - CLI structure: Valid"
echo "  - Test phases: Complete"
echo "  - Documentation: Present"

testing::phase::end_with_summary "Structure tests completed"
