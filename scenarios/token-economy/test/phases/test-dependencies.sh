#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running dependency tests for token-economy..."

# Check Go module dependencies
if [ -f "api/go.mod" ]; then
    echo "Checking Go module dependencies..."
    cd api

    # Verify go.mod is well-formed
    if go mod verify; then
        echo "✓ Go modules verified successfully"
    else
        echo "✗ Go module verification failed"
        testing::phase::fail "Go module verification failed"
    fi

    # Check for vulnerabilities
    if command -v govulncheck &> /dev/null; then
        echo "Checking for known vulnerabilities..."
        if govulncheck ./...; then
            echo "✓ No known vulnerabilities found"
        else
            echo "⚠ Vulnerabilities detected"
        fi
    fi

    # List direct dependencies
    echo "Direct dependencies:"
    go list -m all | grep -v "^github.com/yourorg/token-economy" | head -20

    cd ..
fi

# Check Node.js dependencies if UI exists
if [ -f "ui/package.json" ]; then
    echo "Checking Node.js dependencies..."
    cd ui

    if [ -f "package-lock.json" ]; then
        echo "✓ package-lock.json exists"
    else
        echo "⚠ package-lock.json not found"
    fi

    # Check for security vulnerabilities (if npm is available)
    if command -v npm &> /dev/null; then
        echo "Running npm audit..."
        npm audit --audit-level=high || echo "⚠ Security vulnerabilities found"
    fi

    cd ..
fi

# Check database schema dependencies
if [ -f "initialization/storage/postgres/schema.sql" ]; then
    echo "Checking database schema..."

    # Verify schema file syntax
    if grep -q "CREATE TABLE" initialization/storage/postgres/schema.sql; then
        echo "✓ Database schema file contains table definitions"
    else
        echo "✗ Database schema file appears invalid"
        testing::phase::fail "Invalid database schema"
    fi

    # Check for required tables
    required_tables=("tokens" "wallets" "balances" "transactions")
    for table in "${required_tables[@]}"; do
        if grep -q "CREATE TABLE.*${table}" initialization/storage/postgres/schema.sql; then
            echo "✓ Table '${table}' defined"
        else
            echo "✗ Required table '${table}' not found"
            testing::phase::fail "Missing required table: ${table}"
        fi
    done
fi

# Check Redis dependency (optional)
echo "Checking Redis dependency..."
if command -v redis-cli &> /dev/null; then
    if redis-cli ping &> /dev/null; then
        echo "✓ Redis is available and responding"
    else
        echo "⚠ Redis not responding (optional dependency)"
    fi
else
    echo "⚠ Redis CLI not installed (optional dependency)"
fi

# Check PostgreSQL dependency
echo "Checking PostgreSQL dependency..."
if command -v psql &> /dev/null; then
    echo "✓ PostgreSQL client is installed"
else
    echo "⚠ PostgreSQL client not installed"
fi

# Check for circular dependencies in Go code
if [ -f "api/go.mod" ]; then
    echo "Checking for circular dependencies..."
    cd api
    if go list -json ./... 2>&1 | grep -q "import cycle"; then
        echo "✗ Circular dependencies detected"
        testing::phase::fail "Circular dependencies found"
    else
        echo "✓ No circular dependencies found"
    fi
    cd ..
fi

testing::phase::end_with_summary "Dependency tests completed"
