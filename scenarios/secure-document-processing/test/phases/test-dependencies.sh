#!/bin/bash
set -euo pipefail

echo "🔗 Checking Secure Document Processing dependencies"

SCENARIO_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )/../.." &amp;&amp; pwd )"

# Go dependencies
if [ -f "$SCENARIO_DIR/api/go.mod" ]; then
    echo "📦 Verifying Go dependencies..."
    cd "$SCENARIO_DIR/api"
    go mod tidy
    go mod verify
    echo "✅ Go dependencies verified"
    cd "$SCENARIO_DIR"
fi

# Node.js dependencies
if [ -f "$SCENARIO_DIR/ui/package.json" ]; then
    echo "📦 Verifying Node.js dependencies..."
    cd "$SCENARIO_DIR/ui"
    npm install --dry-run --silent || {
        echo "❌ Node.js dependency issues detected"
        exit 1
    }
    echo "✅ Node.js dependencies verified"
    cd "$SCENARIO_DIR"
fi

# CLI dependencies (basic check)
if [ -f "$SCENARIO_DIR/cli/install.sh" ]; then
    echo "🔧 Verifying CLI installation script..."
    if grep -q "npm install" "$SCENARIO_DIR/cli/install.sh" 2>/dev/null || grep -q "go build" "$SCENARIO_DIR/cli/install.sh" 2>/dev/null; then
        echo "✅ CLI installation script looks correct"
    else
        echo "⚠️  CLI installation script may need dependencies check"
    fi
fi

# Resource dependencies (from service.json)
if [ -f "$SCENARIO_DIR/.vrooli/service.json" ]; then
    echo "📋 Checking required resources..."
    required_resources=$(jq -r '.resources | to_entries[] | select(.value.required == true) | .key' "$SCENARIO_DIR/.vrooli/service.json" 2>/dev/null || echo "")
    for resource in $required_resources; do
        echo "✅ Required resource declared: $resource"
    done
fi

echo "✅ All dependency checks passed"
