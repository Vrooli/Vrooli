#!/usr/bin/env bash
set -euo pipefail

# Test: Code Structure Verification
# Validates that all required files and directories exist

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "📂 Testing Prompt Injection Arena structure..."

# Track failures
FAILURES=0

check_file() {
    local file=$1
    local desc=$2
    if [ -f "${SCENARIO_DIR}/${file}" ]; then
        echo "  ✅ ${desc}"
    else
        echo "  ❌ ${desc} - File not found: ${file}"
        ((FAILURES++))
    fi
}

check_dir() {
    local dir=$1
    local desc=$2
    if [ -d "${SCENARIO_DIR}/${dir}" ]; then
        echo "  ✅ ${desc}"
    else
        echo "  ❌ ${desc} - Directory not found: ${dir}"
        ((FAILURES++))
    fi
}

# Check required directories
echo "📁 Checking required directories..."
check_dir "api" "API directory"
check_dir "ui" "UI directory"
check_dir "cli" "CLI directory"
check_dir "test" "Test directory"
check_dir ".vrooli" "Vrooli configuration directory"
check_dir "initialization" "Initialization directory"
check_dir "initialization/postgres" "PostgreSQL initialization"
check_dir "initialization/n8n" "N8N workflows directory"

# Check required API files
echo "🔧 Checking API files..."
check_file "api/main.go" "Main API entrypoint"
check_file "api/go.mod" "Go module definition"
check_file "api/ollama.go" "Ollama integration"
check_file "api/tournament.go" "Tournament system"
check_file "api/export.go" "Research export functionality"
check_file "api/vector_search.go" "Vector similarity search"
check_file "api/logger.go" "Structured logging module"
check_file "api/config.go" "Environment configuration module"

# Check required UI files
echo "🎨 Checking UI files..."
check_file "ui/index.html" "UI HTML entrypoint"
check_file "ui/package.json" "UI package definition"
check_file "ui/server.js" "UI development server"
check_file "ui/app.js" "UI application code"
check_file "ui/styles.css" "UI styles"

# Check required CLI files
echo "⌨️  Checking CLI files..."
check_file "cli/prompt-injection-arena" "CLI binary"
check_file "cli/install.sh" "CLI install script"
check_file "cli/prompt-injection-arena.bats" "CLI tests"

# Check initialization files
echo "🚀 Checking initialization files..."
check_file "initialization/postgres/schema.sql" "Database schema"
check_file "initialization/postgres/seed.sql" "Database seed data"
check_file "initialization/n8n/security-sandbox.json" "Security sandbox workflow"
check_file "initialization/n8n/injection-tester.json" "Injection tester workflow"

# Check configuration files
echo "⚙️  Checking configuration files..."
check_file ".vrooli/service.json" "Vrooli service configuration"
check_file "PRD.md" "Product Requirements Document"
check_file "README.md" "README documentation"
check_file "PROBLEMS.md" "Known issues documentation"
check_file "Makefile" "Makefile for lifecycle management"

# Check documentation files
echo "📚 Checking documentation..."
check_file "docs/api.md" "API documentation"
check_file "docs/cli.md" "CLI documentation"
check_file "docs/security.md" "Security guidelines"

# Check test structure
echo "🧪 Checking test structure..."
check_dir "test/phases" "Phased test directory"
check_file "test/run-tests.sh" "Test runner"
check_file "test/test-agent-security.sh" "Agent security tests"
check_file "test/test-security-sandbox.sh" "Security sandbox tests"

# Summary
echo ""
if [ ${FAILURES} -eq 0 ]; then
    echo "✅ Structure validation passed!"
    exit 0
else
    echo "❌ Structure validation failed with ${FAILURES} error(s)"
    exit 1
fi
