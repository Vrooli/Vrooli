#!/bin/bash
# Structure testing phase for react-component-library scenario

set -e

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🏗️  Testing React Component Library structure..."
echo "==============================================="

# Test required files
echo ""
echo "📄 Checking required files..."

required_files=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "Makefile"
    "api/main.go"
    "api/go.mod"
    "api/go.sum"
    "cli/react-component-library"
    "cli/install.sh"
    "initialization/storage/postgres/schema.sql"
    "initialization/automation/n8n/qdrant-search.json"
    "initialization/automation/n8n/component-tester.json"
    "test/phases/test-unit.sh"
    "test/phases/test-integration.sh"
    "test/phases/test-performance.sh"
)

missing_files=()
found_files=()

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        found_files+=("$file")
        echo "✅ $file"
    else
        missing_files+=("$file")
        echo "❌ Missing: $file"
    fi
done

# Test required directories
echo ""
echo "📁 Checking required directories..."

required_dirs=(
    "api"
    "api/handlers"
    "api/services"
    "api/models"
    "api/middleware"
    "cli"
    "ui"
    "ui/src"
    "initialization"
    "initialization/storage/postgres"
    "initialization/automation/n8n"
    "test"
    "test/phases"
    "docs"
)

missing_dirs=()
found_dirs=()

for dir in "${required_dirs[@]}"; do
    if [ -d "$dir" ]; then
        found_dirs+=("$dir")
        echo "✅ $dir/"
    else
        missing_dirs+=("$dir")
        echo "❌ Missing: $dir/"
    fi
done

# Test API structure
echo ""
echo "🔍 Validating API structure..."

api_files=(
    "api/main.go"
    "api/handlers/component.go"
    "api/handlers/health.go"
    "api/services/component.go"
    "api/services/search.go"
    "api/services/testing.go"
    "api/services/ai.go"
    "api/models/component.go"
    "api/middleware/rate_limit.go"
    "api/middleware/security.go"
)

for file in "${api_files[@]}"; do
    if [ -f "$file" ]; then
        # Check file is not empty
        if [ -s "$file" ]; then
            echo "✅ $file ($(wc -l < "$file") lines)"
        else
            echo "⚠️  $file is empty"
        fi
    else
        echo "❌ Missing: $file"
    fi
done

# Test test infrastructure
echo ""
echo "🧪 Validating test infrastructure..."

test_files=(
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/main_test.go"
    "api/performance_test.go"
)

for file in "${test_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file"
    else
        echo "⚠️  Missing: $file"
    fi
done

# Validate service.json structure
echo ""
echo "📋 Validating service.json..."

if [ -f ".vrooli/service.json" ]; then
    if command -v jq &> /dev/null; then
        # Check required fields
        required_fields=("service.name" "ports.api" "resources.postgres" "lifecycle.health")

        for field in "${required_fields[@]}"; do
            if jq -e ".$field" .vrooli/service.json > /dev/null 2>&1; then
                echo "✅ Field: $field"
            else
                echo "❌ Missing field: $field"
            fi
        done
    else
        echo "⚠️  jq not available, skipping detailed validation"
        echo "✅ service.json exists"
    fi
else
    echo "❌ service.json not found"
fi

# Test CLI structure
echo ""
echo "💻 Validating CLI structure..."

if [ -f "cli/react-component-library" ]; then
    if [ -x "cli/react-component-library" ]; then
        echo "✅ CLI binary is executable"
    else
        echo "⚠️  CLI binary is not executable"
    fi
else
    echo "⚠️  CLI binary not found (may need to run install.sh)"
fi

if [ -f "cli/install.sh" ]; then
    if [ -x "cli/install.sh" ]; then
        echo "✅ install.sh is executable"
    else
        echo "⚠️  install.sh is not executable"
    fi
fi

# Summary
echo ""
echo "✅ Structure tests completed"
echo ""
echo "Summary:"
echo "  - Required files: ${#found_files[@]}/${#required_files[@]} found"
echo "  - Required directories: ${#found_dirs[@]}/${#required_dirs[@]} found"

if [ ${#missing_files[@]} -gt 0 ] || [ ${#missing_dirs[@]} -gt 0 ]; then
    echo ""
    echo "⚠️  WARNING: Some required files/directories are missing"
    echo "   This may affect functionality"

    if [ ${#missing_files[@]} -gt 0 ]; then
        echo ""
        echo "Missing files:"
        for file in "${missing_files[@]}"; do
            echo "  - $file"
        done
    fi

    if [ ${#missing_dirs[@]} -gt 0 ]; then
        echo ""
        echo "Missing directories:"
        for dir in "${missing_dirs[@]}"; do
            echo "  - $dir"
        done
    fi
fi

testing::phase::end_with_summary "Structure tests completed"
