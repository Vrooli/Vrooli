#!/bin/bash
set -euo pipefail

echo "🧪 Running Secure Document Processing unit tests"

SCENARIO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/../.." &amp;&amp; pwd )"

# Go unit tests
if [ -d "$SCENARIO_DIR/api" ] &amp;&amp; [ -f "$SCENARIO_DIR/api/go.mod" ]; then
    echo "📦 Running Go unit tests..."
    cd "$SCENARIO_DIR/api"
    go test ./... -short -v -coverprofile=coverage.out
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    echo "✅ Go unit tests passed (coverage: $coverage)"
    rm -f coverage.out
    cd "$SCENARIO_DIR"
else
    echo "⚠️  No Go API found, skipping Go unit tests"
fi

# Node.js unit tests (if Jest or similar configured)
if [ -f "$SCENARIO_DIR/ui/package.json" ]; then
    cd "$SCENARIO_DIR/ui"
    if jq -e '.scripts | has("test")' package.json >/dev/null 2>&amp;1; then
        echo "📦 Running Node.js unit tests..."
        npm test -- --watchAll=false
        echo "✅ Node.js unit tests passed"
    else
        echo "⚠️  No test script in package.json, skipping Node.js unit tests"
        # Basic syntax check
        node -c server.js 2>/dev/null || echo "⚠️  UI server.js has syntax errors"
        node -c script.js 2>/dev/null || echo "⚠️  UI script.js has syntax errors"
    fi
    cd "$SCENARIO_DIR"
fi

# CLI unit tests (bats if available)
if [ -f "$SCENARIO_DIR/cli/secure-document-processing.bats" ]; then
    echo "🔧 Running CLI unit tests..."
    cd "$SCENARIO_DIR/cli"
    if command -v bats >/dev/null 2>&amp;1; then
        bats secure-document-processing.bats
        echo "✅ CLI unit tests passed"
    else
        echo "⚠️  bats not installed, skipping CLI tests"
    fi
    cd "$SCENARIO_DIR"
fi

echo "✅ All unit tests completed successfully"
