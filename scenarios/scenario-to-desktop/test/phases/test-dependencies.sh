#!/bin/bash
# Dependency tests for scenario-to-desktop

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../..}" && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🔍 Checking dependencies..."

# Check Go dependencies
if [[ -f "api/go.mod" ]]; then
    echo "📦 Checking Go module dependencies..."
    cd api

    # Verify go.mod is valid
    go mod verify
    if [[ $? -ne 0 ]]; then
        testing::phase::end_with_error "go.mod verification failed"
    fi

    # Check for outdated dependencies
    go list -u -m all | grep '\['  && echo "⚠️  Some dependencies have updates available" || echo "✅ All dependencies up to date"

    # Tidy go.mod
    go mod tidy
    if git diff --exit-code go.mod go.sum; then
        echo "✅ go.mod and go.sum are tidy"
    else
        echo "⚠️  go.mod or go.sum need tidying"
    fi

    cd ..
fi

# Check for required system dependencies
echo "🔧 Checking system dependencies..."
MISSING_DEPS=()

# Check for Node.js (optional but recommended)
if ! command -v node &> /dev/null; then
    echo "⚠️  Node.js not found (optional for Electron builds)"
else
    echo "✅ Node.js $(node --version) found"
fi

# Check for npm (optional but recommended)
if ! command -v npm &> /dev/null; then
    echo "⚠️  npm not found (optional for Electron builds)"
else
    echo "✅ npm $(npm --version) found"
fi

# Check for Go
if ! command -v go &> /dev/null; then
    MISSING_DEPS+=("go")
    echo "❌ Go not found (required)"
else
    echo "✅ Go $(go version | awk '{print $3}') found"
fi

if [[ ${#MISSING_DEPS[@]} -gt 0 ]]; then
    echo "❌ Missing required dependencies: ${MISSING_DEPS[*]}"
    testing::phase::end_with_error "Required dependencies not installed"
fi

testing::phase::end_with_summary "Dependency checks completed"
