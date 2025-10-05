#!/bin/bash
# Dependency test phase for travel-map-filler
# Validates dependencies and external integrations

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🔗 Testing dependencies and integrations..."

EXIT_CODE=0

# Test Go dependencies
if [ -d "api" ] && [ -f "api/go.mod" ]; then
    echo "Verifying Go dependencies..."
    cd api

    # Check for missing dependencies
    go mod verify
    if [ $? -eq 0 ]; then
        echo "✅ Go dependencies verified"
    else
        echo "❌ Go dependency verification failed"
        EXIT_CODE=1
    fi

    # Check for outdated dependencies
    echo ""
    echo "Checking for available updates..."
    go list -u -m all 2>/dev/null || echo "Note: Some updates may be available"

    # Test compilation
    echo ""
    echo "Testing compilation..."
    go build -o /tmp/travel-map-test . 2>&1
    if [ $? -eq 0 ]; then
        echo "✅ Code compiles successfully"
        rm -f /tmp/travel-map-test
    else
        echo "❌ Compilation failed"
        EXIT_CODE=1
    fi

    cd ..
fi

# Test database connectivity
echo ""
echo "Testing database connectivity..."
if [ -n "$POSTGRES_HOST" ] && [ -n "$POSTGRES_PORT" ]; then
    if command -v pg_isready &> /dev/null; then
        pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "${POSTGRES_USER:-postgres}"
        if [ $? -eq 0 ]; then
            echo "✅ Database connection successful"
        else
            echo "⚠️  Database not ready (may be starting up)"
        fi
    else
        echo "ℹ️  pg_isready not available, skipping database check"
    fi
else
    echo "⚠️  Database configuration not found in environment"
fi

# Test optional resource connectivity
echo ""
echo "Testing optional resource connectivity..."

# Test N8N connectivity (optional)
if [ -n "$N8N_URL" ]; then
    echo "Testing N8N connectivity..."
    if command -v curl &> /dev/null; then
        curl -sf --connect-timeout 5 "$N8N_URL" > /dev/null 2>&1
        if [ $? -eq 0 ]; then
            echo "✅ N8N connection successful"
        else
            echo "ℹ️  N8N not available (optional resource)"
        fi
    fi
fi

# Test Qdrant connectivity (optional)
if [ -n "$QDRANT_URL" ]; then
    echo "Testing Qdrant connectivity..."
    if command -v curl &> /dev/null; then
        curl -sf --connect-timeout 5 "$QDRANT_URL/collections" > /dev/null 2>&1
        if [ $? -eq 0 ]; then
            echo "✅ Qdrant connection successful"
        else
            echo "ℹ️  Qdrant not available (optional resource)"
        fi
    fi
fi

testing::phase::end_with_summary "Dependency tests completed"

exit $EXIT_CODE
