#!/bin/bash
set -e

echo "🧪 Running unit tests for bedtime-story-generator..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
API_DIR="${SCENARIO_DIR}/api"

# Test Go API compilation
echo "📦 Testing Go API compilation..."
if (cd "${API_DIR}" && go build -o /tmp/test-build main.go); then
    echo "✅ API builds successfully"
    rm -f /tmp/test-build
else
    echo "❌ API build failed"
    exit 1
fi

# Run Go tests if they exist
echo "🔍 Running Go unit tests..."
if (cd "${API_DIR}" && go test -v ./... 2>&1 | grep -q "no test files"); then
    echo "⚠️  No Go test files found (will add in future)"
else
    (cd "${API_DIR}" && go test -v ./...)
fi

# Test CLI functionality
echo "🖥️  Testing CLI commands..."
CLI_BIN="${SCENARIO_DIR}/cli/bedtime-story"

if [ ! -x "${CLI_BIN}" ]; then
    echo "❌ CLI binary not found or not executable: ${CLI_BIN}"
    exit 1
fi

# Test help command
if "${CLI_BIN}" help | grep -q "Bedtime Story Generator CLI"; then
    echo "✅ CLI help command works"
else
    echo "❌ CLI help command failed"
    exit 1
fi

# Test status command with API running
if "${CLI_BIN}" status | grep -q "API Server"; then
    echo "✅ CLI status command works"
else
    echo "❌ CLI status command failed"
    exit 1
fi

# Test themes command
if "${CLI_BIN}" themes | grep -q "Adventure"; then
    echo "✅ CLI themes command works"
else
    echo "❌ CLI themes command failed"
    exit 1
fi

# Test UI build configuration
echo "🌐 Testing UI configuration..."
if [ -f "${SCENARIO_DIR}/ui/package.json" ]; then
    echo "✅ UI package.json exists"
    
    # Check for required dependencies
    if grep -q "react" "${SCENARIO_DIR}/ui/package.json"; then
        echo "✅ React dependency found"
    else
        echo "❌ React dependency missing"
        exit 1
    fi
    
    if grep -q "vite" "${SCENARIO_DIR}/ui/package.json"; then
        echo "✅ Vite dependency found"
    else
        echo "❌ Vite dependency missing"
        exit 1
    fi
else
    echo "❌ UI package.json not found"
    exit 1
fi

echo "✅ All unit tests passed!"