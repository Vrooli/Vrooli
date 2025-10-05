#!/bin/bash
# CLI test runner - executes BATS tests for prompt-manager CLI
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🧪 Running CLI tests for prompt-manager..."

# Check if bats is installed
if ! command -v bats &> /dev/null; then
    echo "❌ BATS is not installed. Please install it first:"
    echo "   Debian/Ubuntu: apt-get install bats"
    echo "   macOS: brew install bats-core"
    echo "   Manual: git clone https://github.com/bats-core/bats-core.git && cd bats-core && sudo ./install.sh /usr/local"
    exit 1
fi

# Check if CLI is installed
if ! command -v prompt-manager &> /dev/null; then
    echo "⚠️  prompt-manager CLI not found in PATH"
    echo "   Attempting to install from $SCENARIO_DIR/cli/prompt-manager"

    if [ -f "$SCENARIO_DIR/cli/prompt-manager" ]; then
        export PATH="$SCENARIO_DIR/cli:$PATH"
        echo "✅ Added $SCENARIO_DIR/cli to PATH"
    else
        echo "❌ CLI not found. Please run: cd $SCENARIO_DIR && make setup"
        exit 1
    fi
fi

# Run BATS tests
echo "📝 Running BATS tests..."
if bats "$SCRIPT_DIR/prompt-manager.bats"; then
    echo "✅ All CLI tests passed!"
    exit 0
else
    echo "❌ Some CLI tests failed"
    exit 1
fi
