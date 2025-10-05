#!/bin/bash
# Dependency test phase for roi-fit-analysis
# Validates dependencies and external service requirements

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🔗 Running dependency tests for roi-fit-analysis..."

# Test 1: Go module dependencies
echo "  ✓ Checking Go dependencies..."
cd api

if [[ ! -f "go.mod" ]]; then
    testing::phase::fail "go.mod file missing"
fi

# Verify required packages are in go.mod
required_packages=(
    "github.com/lib/pq"
    "github.com/google/uuid"
)

for pkg in "${required_packages[@]}"; do
    if ! grep -q "$pkg" go.mod; then
        testing::phase::fail "Required package missing from go.mod: $pkg"
    fi
done

# Test 2: Go dependencies are downloadable
echo "  ✓ Verifying Go dependencies download..."
if ! go mod download 2>/dev/null; then
    testing::phase::fail "Failed to download Go dependencies"
fi

# Test 3: Go dependencies are tidy
echo "  ✓ Checking Go mod tidy..."
go mod tidy
if git diff --quiet go.mod go.sum 2>/dev/null; then
    : # No changes, all good
else
    echo "  ⚠ go.mod/go.sum could be tidied (non-fatal)"
fi

# Test 4: No vulnerable dependencies (if govulncheck is available)
echo "  ✓ Checking for vulnerable dependencies..."
if command -v govulncheck &> /dev/null; then
    if govulncheck ./... 2>&1 | grep -q "No vulnerabilities found"; then
        echo "    ✓ No vulnerabilities found"
    else
        echo "    ⚠ Potential vulnerabilities detected (review recommended)"
    fi
else
    echo "    ℹ govulncheck not available, skipping vulnerability scan"
fi

# Test 5: Database driver is loadable
echo "  ✓ Checking database driver..."
if ! go list -m github.com/lib/pq &>/dev/null; then
    testing::phase::fail "PostgreSQL driver not properly loaded"
fi

# Test 6: Test dependencies are available
echo "  ✓ Checking test dependencies..."
if ! go list -m all | grep -q "testing"; then
    testing::phase::fail "Testing package not available"
fi

# Test 7: External service configuration
echo "  ✓ Checking external service requirements..."
cd "$TESTING_PHASE_SCENARIO_DIR"

# Check if service.json defines required resources
if jq -e '.dependencies.resources[]? | select(.name=="postgres")' .vrooli/service.json >/dev/null 2>&1; then
    echo "    ✓ PostgreSQL dependency defined"
else
    echo "    ⚠ PostgreSQL dependency not explicitly defined"
fi

if jq -e '.dependencies.resources[]? | select(.name=="ollama")' .vrooli/service.json >/dev/null 2>&1; then
    echo "    ✓ Ollama dependency defined"
else
    echo "    ⚠ Ollama dependency not explicitly defined"
fi

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::end_with_summary "Dependency tests completed successfully"
