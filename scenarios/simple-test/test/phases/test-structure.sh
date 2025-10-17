#!/bin/bash
# Structure test phase - validates scenario structure and dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Validating scenario structure"

# Check required files
required_files=(
    "server.js"
    "package.json"
    ".vrooli/service.json"
    "initialization/storage/postgres/schema.sql"
    "initialization/storage/postgres/seed.sql"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        log::success "✓ Required file exists: $file"
    else
        testing::phase::add_error "Missing required file: $file"
    fi
done

# Validate package.json
log::info "Validating package.json..."
if jq empty < package.json >/dev/null 2>&1; then
    log::success "✓ package.json is valid JSON"

    # Check for required scripts
    if jq -e '.scripts.test' package.json >/dev/null; then
        log::success "✓ Test script defined"
    else
        testing::phase::add_error "Missing test script in package.json"
    fi
else
    testing::phase::add_error "Invalid JSON in package.json"
fi

# Validate service.json
log::info "Validating service.json..."
if jq empty < .vrooli/service.json >/dev/null 2>&1; then
    log::success "✓ service.json is valid JSON"

    # Check required fields
    if jq -e '.service.name' .vrooli/service.json >/dev/null; then
        log::success "✓ Service name defined"
    else
        testing::phase::add_error "Missing service name in service.json"
    fi
else
    testing::phase::add_error "Invalid JSON in service.json"
fi

# Validate SQL files
log::info "Validating SQL files..."
for sql_file in initialization/storage/postgres/*.sql; do
    if [ -f "$sql_file" ]; then
        if [ -s "$sql_file" ] && grep -q -E "(CREATE|INSERT|SELECT|DROP|ALTER)" "$sql_file"; then
            log::success "✓ SQL file valid: $(basename $sql_file)"
        else
            testing::phase::add_error "Invalid SQL file: $sql_file"
        fi
    fi
done

# Check test infrastructure
log::info "Validating test infrastructure..."
if [ -d "__tests__" ]; then
    test_count=$(find __tests__ -name "*.test.js" | wc -l)
    log::success "✓ Found $test_count test files"
else
    testing::phase::add_error "Missing __tests__ directory"
fi

if [ -d "test/phases" ]; then
    phase_count=$(find test/phases -name "test-*.sh" | wc -l)
    log::success "✓ Found $phase_count test phases"
else
    testing::phase::add_error "Missing test/phases directory"
fi

# End with summary
testing::phase::end_with_summary "Structure validation completed"
