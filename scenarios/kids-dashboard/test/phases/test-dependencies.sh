#!/bin/bash

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running dependency tests for kids-dashboard..."

# Test 1: Check Go dependencies
testing::phase::log "Checking Go module dependencies..."
cd api

if [ -f go.mod ]; then
    go mod verify 2>&1
    if [ $? -eq 0 ]; then
        testing::phase::log "✓ Go dependencies verified"
    else
        testing::phase::log "❌ Go dependency verification failed"
        exit 1
    fi

    # Check for outdated dependencies
    go list -u -m all 2>&1 | grep -E "\[" | head -5
    testing::phase::log "Checked for outdated dependencies"
else
    testing::phase::log "⚠️  No go.mod found"
fi

# Test 2: Check for required Go version
testing::phase::log "Checking Go version compatibility..."
if grep -q "^go " go.mod 2>/dev/null; then
    REQUIRED_VERSION=$(grep "^go " go.mod | awk '{print $2}')
    CURRENT_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    testing::phase::log "Required Go version: $REQUIRED_VERSION"
    testing::phase::log "Current Go version: $CURRENT_VERSION"
fi

# Test 3: Check Node.js dependencies (if UI exists)
cd "$TESTING_PHASE_SCENARIO_DIR"

if [ -d ui ]; then
    testing::phase::log "Checking Node.js dependencies..."
    cd ui

    if [ -f package.json ]; then
        # Check if package-lock.json exists
        if [ -f package-lock.json ]; then
            testing::phase::log "✓ package-lock.json found"
        else
            testing::phase::log "⚠️  No package-lock.json (consider adding for reproducibility)"
        fi

        # Check for security vulnerabilities
        if command -v npm &> /dev/null; then
            testing::phase::log "Checking for npm vulnerabilities..."
            npm audit --audit-level=moderate 2>&1 | head -10 || true
        fi
    fi
fi

# Test 4: Check for required system dependencies
cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Checking required system tools..."
REQUIRED_TOOLS=("jq" "curl")
for tool in "${REQUIRED_TOOLS[@]}"; do
    if command -v "$tool" &> /dev/null; then
        testing::phase::log "✓ $tool available"
    else
        testing::phase::log "⚠️  $tool not found (optional)"
    fi
done

# Test 5: Check service.json dependencies
testing::phase::log "Checking service configuration..."
if [ -f .vrooli/service.json ]; then
    # Verify JSON is valid
    if jq empty .vrooli/service.json 2>/dev/null; then
        testing::phase::log "✓ service.json is valid JSON"

        # Check for required fields
        NAME=$(jq -r '.service.name // empty' .vrooli/service.json)
        if [ -n "$NAME" ]; then
            testing::phase::log "✓ Service name: $NAME"
        else
            testing::phase::log "❌ Service name missing"
        fi
    else
        testing::phase::log "❌ service.json is invalid JSON"
        exit 1
    fi
else
    testing::phase::log "⚠️  No service.json found"
fi

testing::phase::end_with_summary "Dependency checks completed"
