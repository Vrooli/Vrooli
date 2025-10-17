#!/bin/bash
# Dependencies test phase - validates all dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Validating dependencies"

# Check Node.js
if command -v node >/dev/null 2>&1; then
    node_version=$(node --version)
    log::success "✓ Node.js installed: $node_version"
else
    testing::phase::add_error "Node.js not found"
fi

# Check npm dependencies
if [ -f "package.json" ]; then
    log::info "Checking npm dependencies..."

    # Install dependencies if needed
    if [ ! -d "node_modules" ]; then
        log::info "Installing npm dependencies..."
        npm install --silent >/dev/null 2>&1
    fi

    # Check for security vulnerabilities
    if command -v npm >/dev/null 2>&1; then
        log::info "Running security audit..."
        npm audit --audit-level=high >/dev/null 2>&1 || log::warning "Some security vulnerabilities found (run 'npm audit' for details)"
    fi

    # Verify test dependencies
    if [ -d "node_modules/jest" ]; then
        log::success "✓ Jest installed"
    else
        testing::phase::add_error "Jest not installed"
    fi

    if [ -d "node_modules/supertest" ]; then
        log::success "✓ Supertest installed"
    else
        testing::phase::add_error "Supertest not installed"
    fi
fi

# Check PostgreSQL availability (optional for simple-test)
if command -v psql >/dev/null 2>&1; then
    log::success "✓ PostgreSQL client available"

    postgres_host="${POSTGRES_HOST:-localhost}"
    postgres_port="${POSTGRES_PORT:-5433}"

    if timeout 3 psql -h "$postgres_host" -p "$postgres_port" -U postgres -d postgres -c "SELECT 1;" >/dev/null 2>&1; then
        log::success "✓ PostgreSQL connection successful"
    else
        log::warning "PostgreSQL not available (optional for simple-test)"
    fi
else
    log::warning "PostgreSQL client not installed (optional)"
fi

# Check required system utilities
required_commands=("curl" "jq")
for cmd in "${required_commands[@]}"; do
    if command -v "$cmd" >/dev/null 2>&1; then
        log::success "✓ $cmd available"
    else
        testing::phase::add_error "$cmd not found"
    fi
done

# End with summary
testing::phase::end_with_summary "Dependencies validation completed"
