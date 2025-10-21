#!/usr/bin/env bash
set -euo pipefail

# Test: Structural Validation
# Validates that SmartNotes has proper file structure and configuration

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "🏗️  Testing SmartNotes structure..."

# Track failures
FAILURES=0

check_file() {
    local file=$1
    local desc=$2
    if [ -f "${SCENARIO_DIR}/${file}" ]; then
        echo "  ✅ ${desc}"
    else
        echo "  ❌ ${desc} - Missing: ${file}"
        ((FAILURES++))
    fi
}

check_dir() {
    local dir=$1
    local desc=$2
    if [ -d "${SCENARIO_DIR}/${dir}" ]; then
        echo "  ✅ ${desc}"
    else
        echo "  ❌ ${desc} - Missing: ${dir}"
        ((FAILURES++))
    fi
}

# Check required directories
echo "📁 Checking directories..."
check_dir "api" "API directory"
check_dir "ui" "UI directory"
check_dir "cli" "CLI directory"
check_dir "test" "Test directory"
check_dir "initialization" "Initialization directory"

# Check required files
echo "📄 Checking required files..."
check_file "PRD.md" "Product Requirements Document"
check_file "README.md" "README documentation"
check_file "Makefile" "Makefile"
check_file ".vrooli/service.json" "Service configuration"

# Check API structure
echo "🔧 Checking API structure..."
check_file "api/main.go" "API main file"
check_file "api/go.mod" "Go module file"

# Check UI structure
echo "🎨 Checking UI structure..."
check_file "ui/index.html" "UI index file"
check_file "ui/server.js" "UI server"
check_file "ui/package.json" "UI package.json"

# Check CLI structure
echo "🖥️  Checking CLI structure..."
check_file "cli/notes" "CLI binary/script"
check_file "cli/install.sh" "CLI install script"

# Check initialization files
echo "⚙️  Checking initialization files..."
check_file "initialization/postgres/schema.sql" "PostgreSQL schema"
check_file "initialization/qdrant/collections.json" "Qdrant collections"

# Validate service.json
echo "🔍 Validating service.json..."
if command -v jq &> /dev/null; then
    if jq empty "${SCENARIO_DIR}/.vrooli/service.json" 2>/dev/null; then
        echo "  ✅ service.json is valid JSON"
    else
        echo "  ❌ service.json has invalid JSON"
        ((FAILURES++))
    fi
else
    echo "  ⚠️  jq not installed, skipping JSON validation"
fi

# Summary
echo ""
if [ ${FAILURES} -eq 0 ]; then
    echo "✅ Structure validation passed!"
    exit 0
else
    echo "❌ Structure validation failed with ${FAILURES} error(s)"
    exit 1
fi
