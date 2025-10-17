#!/bin/bash
set -euo pipefail

# Recipe Book Dependencies Test Runner

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Starting Recipe Book dependencies tests"

# Test 1: PostgreSQL dependency
testing::phase::log "Testing PostgreSQL dependency..."

if command -v psql &> /dev/null; then
    testing::phase::log "✓ PostgreSQL client available"

    # Try to connect to database
    if PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h "${POSTGRES_HOST:-localhost}" \
        -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-recipe_book}" \
        -c "SELECT 1;" > /dev/null 2>&1; then
        testing::phase::log "✓ PostgreSQL connection successful"

        # Verify tables exist
        TABLES=$(PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql -h "${POSTGRES_HOST:-localhost}" \
            -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-recipe_book}" \
            -t -c "SELECT tablename FROM pg_tables WHERE schemaname='public';" 2>/dev/null || echo "")

        if echo "$TABLES" | grep -q "recipes"; then
            testing::phase::log "✓ recipes table exists"
        else
            testing::phase::log "⚠ recipes table not found"
        fi

        if echo "$TABLES" | grep -q "recipe_ratings"; then
            testing::phase::log "✓ recipe_ratings table exists"
        else
            testing::phase::log "⚠ recipe_ratings table not found"
        fi
    else
        testing::phase::log "⚠ PostgreSQL connection failed (may not be running)"
    fi
else
    testing::phase::log "⚠ PostgreSQL client not available"
fi

# Test 2: Go dependencies
testing::phase::log "Testing Go dependencies..."

cd api

if [ -f "go.mod" ]; then
    # Check dependency versions
    GO_VERSION=$(go version 2>/dev/null | awk '{print $3}' || echo "unknown")
    testing::phase::log "✓ Go version: $GO_VERSION"

    # Verify dependencies are downloadable
    if go mod download &> /dev/null; then
        testing::phase::log "✓ Go dependencies downloadable"
    else
        testing::phase::log "⚠ Go dependency download issues"
    fi

    # Check for outdated dependencies
    if command -v go &> /dev/null; then
        if go list -u -m all 2>/dev/null | grep -q '\['; then
            testing::phase::log "⚠ Some dependencies have updates available"
        else
            testing::phase::log "✓ Dependencies are up to date"
        fi
    fi

    # Verify key dependencies
    DEPS=(
        "github.com/gorilla/mux"
        "github.com/lib/pq"
        "github.com/google/uuid"
    )

    for dep in "${DEPS[@]}"; do
        if go list -m "$dep" &> /dev/null; then
            VERSION=$(go list -m "$dep" 2>/dev/null | awk '{print $2}')
            testing::phase::log "✓ $dep ($VERSION)"
        else
            testing::phase::log "✗ $dep not found"
        fi
    done
else
    testing::phase::log "✗ go.mod not found"
fi

cd ..

# Test 3: Qdrant dependency (optional)
testing::phase::log "Testing Qdrant dependency..."

if curl -sf "http://${QDRANT_HOST:-localhost}:${QDRANT_PORT:-6333}/collections" > /dev/null 2>&1; then
    testing::phase::log "✓ Qdrant connection successful"
else
    testing::phase::log "⚠ Qdrant not available (semantic search may not work)"
fi

# Test 4: N8n workflow dependency (optional)
testing::phase::log "Testing N8n dependency..."

if curl -sf "http://${N8N_HOST:-localhost}:${N8N_PORT:-5678}/healthz" > /dev/null 2>&1; then
    testing::phase::log "✓ N8n connection successful"
else
    testing::phase::log "⚠ N8n not available (AI features may not work)"
fi

# Test 5: Redis dependency (optional)
testing::phase::log "Testing Redis dependency..."

if command -v redis-cli &> /dev/null; then
    if redis-cli -h "${REDIS_HOST:-localhost}" -p "${REDIS_PORT:-6379}" ping > /dev/null 2>&1; then
        testing::phase::log "✓ Redis connection successful"
    else
        testing::phase::log "⚠ Redis not available (caching may not work)"
    fi
else
    testing::phase::log "⚠ Redis client not available"
fi

# Test 6: Environment variables
testing::phase::log "Testing environment variables..."

ENV_VARS=(
    "API_PORT"
    "POSTGRES_HOST"
    "POSTGRES_PORT"
    "POSTGRES_USER"
    "POSTGRES_DB"
)

for var in "${ENV_VARS[@]}"; do
    if [ -n "${!var:-}" ]; then
        testing::phase::log "✓ $var=${!var}"
    else
        testing::phase::log "⚠ $var not set (using defaults)"
    fi
done

# Test 7: Port availability
testing::phase::log "Testing port availability..."

API_PORT="${API_PORT:-3250}"

if command -v nc &> /dev/null; then
    if nc -z localhost "$API_PORT" 2>/dev/null; then
        testing::phase::log "✓ API port $API_PORT is available"
    else
        testing::phase::log "⚠ API port $API_PORT not responding (API may not be running)"
    fi
else
    testing::phase::log "⚠ nc not available, cannot check port"
fi

# Test 8: Go build dependencies
testing::phase::log "Testing Go build dependencies..."

cd api

# Check for build tags
if grep -r "+build" *.go > /dev/null 2>&1; then
    testing::phase::log "✓ Build tags found (testing isolation)"
fi

# Verify test build
if go test -tags testing -c -o /tmp/recipe-book-test.test . 2>&1 | grep -v "warning"; then
    testing::phase::log "✓ Test binary builds successfully"
    rm -f /tmp/recipe-book-test.test
else
    testing::phase::log "✗ Test binary build failed"
fi

cd ..

# Test 9: Test framework dependencies
testing::phase::log "Testing test framework dependencies..."

# Check for testing tools
if command -v go &> /dev/null; then
    if go version | grep -q "go1"; then
        testing::phase::log "✓ Go testing framework available"
    fi
fi

# Verify test helpers
if [ -f "api/test_helpers.go" ]; then
    HELPER_FUNCS=(
        "setupTestLogger"
        "setupTestEnvironment"
        "setupTestDB"
        "makeHTTPRequest"
        "assertJSONResponse"
        "assertErrorResponse"
    )

    for func in "${HELPER_FUNCS[@]}"; do
        if grep -q "func $func" api/test_helpers.go; then
            testing::phase::log "✓ Helper function: $func"
        else
            testing::phase::log "⚠ Missing helper: $func"
        fi
    done
fi

# Test 10: External service dependencies (service.json)
testing::phase::log "Testing service.json dependencies..."

if [ -f ".vrooli/service.json" ] && command -v jq &> /dev/null; then
    DEPS=$(jq -r '.dependencies[]? // empty' < .vrooli/service.json 2>/dev/null || echo "")

    if [ -n "$DEPS" ]; then
        testing::phase::log "Declared dependencies:"
        echo "$DEPS" | while read -r dep; do
            testing::phase::log "  - $dep"
        done
    else
        testing::phase::log "⚠ No dependencies declared in service.json"
    fi
fi

testing::phase::end_with_summary "Recipe Book dependencies tests completed"
