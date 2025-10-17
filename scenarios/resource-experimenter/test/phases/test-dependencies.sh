#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "=== Dependency Testing for resource-experimenter ==="

# Check Go dependencies
if [ -f "api/go.mod" ]; then
    echo "✓ Checking Go dependencies..."
    cd api
    go mod verify || testing::phase::error "Go dependencies verification failed"
    go mod tidy -diff || testing::phase::warn "Go dependencies need tidying"
    cd ..
else
    testing::phase::error "api/go.mod not found"
fi

# Check Node.js dependencies if UI exists
if [ -f "ui/package.json" ]; then
    echo "✓ Checking Node.js dependencies..."
    cd ui
    if [ ! -d "node_modules" ]; then
        testing::phase::warn "node_modules not found - run npm install"
    fi
    cd ..
fi

# Check resource dependencies
echo "✓ Checking resource dependencies..."
REQUIRED_RESOURCES=("postgres" "claude-code")
for resource in "${REQUIRED_RESOURCES[@]}"; do
    if ! grep -q "\"$resource\"" .vrooli/service.json; then
        testing::phase::error "Required resource '$resource' not found in service.json"
    else
        echo "  ✓ Resource dependency: $resource"
    fi
done

# Check database schema files
if [ -d "initialization/postgres" ]; then
    echo "✓ Checking database schema files..."
    if [ ! -f "initialization/postgres/schema.sql" ]; then
        testing::phase::error "Database schema file not found"
    fi
    if [ ! -f "initialization/postgres/seed.sql" ]; then
        testing::phase::warn "Database seed file not found"
    fi
else
    testing::phase::error "initialization/postgres directory not found"
fi

# Check CLI installation script
if [ -d "cli" ]; then
    if [ -f "cli/install.sh" ]; then
        echo "✓ CLI installation script exists"
    else
        testing::phase::warn "CLI installation script not found"
    fi
fi

testing::phase::end_with_summary "Dependency tests completed"
