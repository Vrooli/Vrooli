#!/usr/bin/env bash
# Retro Game Launcher Test Script

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/retro-game-launcher"

echo "🎮 Testing Retro Game Launcher..."

# Test Go API compilation
if [[ -f "$SCRIPT_DIR/api/main.go" ]]; then
    echo "✅ Testing Go API compilation..."
    cd "$SCRIPT_DIR/api" && go build -o test-build main.go && rm test-build
    echo "✅ Go API compiles successfully"
else
    echo "❌ Go API main.go not found"
    exit 1
fi

# Test CLI script
if [[ -f "$SCRIPT_DIR/cli/retro-game-launcher" ]]; then
    echo "✅ Testing CLI script..."
    chmod +x "$SCRIPT_DIR/cli/retro-game-launcher"
    "$SCRIPT_DIR/cli/retro-game-launcher" version
    echo "✅ CLI script works"
else
    echo "❌ CLI script not found"
    exit 1
fi

# Test React app structure
if [[ -f "$SCRIPT_DIR/ui/package.json" ]] && [[ -f "$SCRIPT_DIR/ui/src/App.js" ]]; then
    echo "✅ React UI structure looks good"
else
    echo "❌ React UI files missing"
    exit 1
fi

# Test BATS if available
if command -v bats &> /dev/null && [[ -f "$SCRIPT_DIR/cli/retro-game-launcher.bats" ]]; then
    echo "✅ Running CLI tests with BATS..."
    cd "$SCRIPT_DIR/cli" && bats retro-game-launcher.bats
    echo "✅ CLI tests passed"
else
    echo "⚠️  BATS not available, skipping CLI tests"
fi

echo "🚀 All tests passed! Retro Game Launcher is ready."
echo ""
echo "Next steps:"
echo "  1. Run: ./scripts/manage.sh setup"
echo "  2. Run: ./scripts/manage.sh develop"
echo "  3. Open: http://localhost:3000"