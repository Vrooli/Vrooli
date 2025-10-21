#!/usr/bin/env bash
# Test Phase 1: Structure Validation
# Validates that all required files and directories exist according to v2.0 contract

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "================================================"
echo "🔍 Phase 1: Structure Validation"
echo "================================================"

TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
check_file() {
    local file="$1"
    local description="$2"

    if [ -f "$SCENARIO_DIR/$file" ]; then
        echo "  ✅ $description: $file"
        ((TESTS_PASSED++))
    else
        echo "  ❌ $description: $file (missing)"
        ((TESTS_FAILED++))
    fi
}

check_dir() {
    local dir="$1"
    local description="$2"

    if [ -d "$SCENARIO_DIR/$dir" ]; then
        echo "  ✅ $description: $dir/"
        ((TESTS_PASSED++))
    else
        echo "  ❌ $description: $dir/ (missing)"
        ((TESTS_FAILED++))
    fi
}

# Required Files
echo ""
echo "📄 Required Files:"
check_file ".vrooli/service.json" "Service configuration"
check_file "PRD.md" "Product requirements"
check_file "README.md" "Documentation"
check_file "Makefile" "Lifecycle commands"

# API Component
echo ""
echo "⚙️ API Component:"
check_file "api/go.mod" "Go module definition"
check_file "api/cmd/server/main.go" "API server entry point"
check_file "api/network-tools-api" "Compiled API binary"

# CLI Component
echo ""
echo "💻 CLI Component:"
check_file "cli/network-tools" "CLI binary/script"
check_file "cli/install.sh" "CLI installer"

# UI Component
echo ""
echo "🎨 UI Component:"
check_dir "ui" "UI directory"
check_file "ui/server.js" "UI server"
check_file "ui/public/index.html" "UI entry point"

# Database Initialization
echo ""
echo "🗄️ Database Initialization:"
check_dir "initialization/storage/postgres" "PostgreSQL initialization"
check_file "initialization/storage/postgres/schema.sql" "Database schema"
check_file "initialization/storage/postgres/seed.sql" "Seed data"

# Test Infrastructure
echo ""
echo "🧪 Test Infrastructure:"
check_dir "test/phases" "Phased test directory"
check_file "test/phases/test-api.sh" "API tests"
check_file "test/phases/test-integration.sh" "Integration tests"
check_file "test/phases/test-unit.sh" "Unit tests"

# Summary
echo ""
echo "================================================"
echo "📊 Structure Validation Summary"
echo "================================================"
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [ $TESTS_FAILED -eq 0 ]; then
    echo ""
    echo "✅ All structure validation tests passed!"
    exit 0
else
    echo ""
    echo "❌ Some structure validation tests failed"
    exit 1
fi
