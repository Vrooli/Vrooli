#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Checking Go dependencies..."

# Check if go.mod exists
if [ ! -f "api/go.mod" ]; then
    echo "❌ go.mod not found in api directory"
    exit 1
fi

# Verify Go dependencies
cd api
if ! go mod verify; then
    echo "❌ Go dependency verification failed"
    exit 1
fi

echo "✓ Go dependencies verified"

# Check for outdated dependencies
echo "Checking for outdated dependencies..."
go list -u -m all | grep '\[' || echo "✓ All dependencies are up to date"

# Check for vulnerable dependencies
if command -v govulncheck &> /dev/null; then
    echo "Running vulnerability check..."
    govulncheck ./... || echo "⚠️ Vulnerability check found issues (non-fatal)"
fi

# Verify critical dependencies are present
echo "Checking critical dependencies..."
REQUIRED_DEPS=(
    "github.com/redis/go-redis/v9"
)

for dep in "${REQUIRED_DEPS[@]}"; do
    if ! go list -m "$dep" &> /dev/null; then
        echo "❌ Required dependency missing: $dep"
        exit 1
    else
        echo "✓ Found: $dep"
    fi
done

# Check CLI dependencies
echo "Checking CLI dependencies..."
cd "$TESTING_PHASE_SCENARIO_DIR/cli"

CLI_COMMANDS=(
    "curl"
    "jq"
)

for cmd in "${CLI_COMMANDS[@]}"; do
    if ! command -v "$cmd" &> /dev/null; then
        echo "❌ Required CLI command missing: $cmd"
        exit 1
    else
        echo "✓ Found: $cmd"
    fi
done

testing::phase::end_with_summary "Dependency tests completed"
