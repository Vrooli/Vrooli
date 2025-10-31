#!/usr/bin/env bash
set -euo pipefail

# Test: Dependency Validation
# Validates that all required resources and dependencies are available

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "📦 Testing Prompt Injection Arena dependencies..."

# Track failures
FAILURES=0

check_resource() {
    local resource=$1
    local desc=$2
    if vrooli resource status "${resource}" &> /dev/null; then
        echo "  ✅ ${desc}"
    else
        echo "  ❌ ${desc} - Resource not available: ${resource}"
        ((FAILURES++))
    fi
}

check_command() {
    local cmd=$1
    local desc=$2
    if command -v "${cmd}" &> /dev/null; then
        echo "  ✅ ${desc}"
    else
        echo "  ❌ ${desc} - Command not found: ${cmd}"
        ((FAILURES++))
    fi
}

# Check required resources
echo "🔌 Checking required resources..."
check_resource "postgres" "PostgreSQL database"
check_resource "ollama" "Ollama AI service"
check_resource "n8n" "n8n workflow engine"

# Check optional resources (warnings only)
echo "⚙️  Checking optional resources..."
if vrooli resource status "qdrant" &> /dev/null; then
    echo "  ✅ Qdrant vector database (optional)"
else
    echo "  ⚠️  Qdrant vector database not available (optional, similarity search disabled)"
fi

# Check system dependencies
echo "🛠️  Checking system dependencies..."
check_command "go" "Go compiler"
check_command "node" "Node.js"
check_command "npm" "npm package manager"
check_command "jq" "jq JSON processor"
check_command "curl" "curl HTTP client"

# Check Go dependencies
echo "🔧 Checking Go dependencies..."
if [ -f "${SCENARIO_DIR}/api/go.mod" ]; then
    cd "${SCENARIO_DIR}/api"
    if go mod verify &> /dev/null; then
        echo "  ✅ Go modules verified"
    else
        echo "  ⚠️  Go modules need downloading (run 'go mod download')"
    fi
else
    echo "  ❌ go.mod not found"
    ((FAILURES++))
fi

# Check Node dependencies
echo "📦 Checking Node dependencies..."
if [ -f "${SCENARIO_DIR}/ui/package.json" ]; then
    if [ -d "${SCENARIO_DIR}/ui/node_modules" ]; then
        echo "  ✅ Node modules installed"
    else
        echo "  ⚠️  Node modules need installation (run 'npm install')"
    fi
else
    echo "  ❌ package.json not found"
    ((FAILURES++))
fi

# Check environment variables
echo "🌍 Checking environment configuration..."
if [ -n "${POSTGRES_HOST:-}" ]; then
    echo "  ✅ POSTGRES_HOST configured"
else
    echo "  ⚠️  POSTGRES_HOST not set (will be provided by lifecycle)"
fi

if [ -n "${OLLAMA_URL:-}" ]; then
    echo "  ✅ OLLAMA_URL configured"
else
    echo "  ⚠️  OLLAMA_URL not set (will use default)"
fi

# Check CLI binary
echo "🔧 Checking CLI binary..."
if [ -f "${SCENARIO_DIR}/cli/prompt-injection-arena" ]; then
    if [ -x "${SCENARIO_DIR}/cli/prompt-injection-arena" ]; then
        echo "  ✅ CLI binary exists and is executable"
    else
        echo "  ⚠️  CLI binary not executable (run 'chmod +x cli/prompt-injection-arena')"
    fi
else
    echo "  ❌ CLI binary not found"
    ((FAILURES++))
fi

# Summary
echo ""
if [ ${FAILURES} -eq 0 ]; then
    echo "✅ Dependency validation passed!"
    exit 0
else
    echo "❌ Dependency validation failed with ${FAILURES} error(s)"
    exit 1
fi
