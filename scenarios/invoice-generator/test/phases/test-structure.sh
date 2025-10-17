#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 15-second target
testing::phase::init --target-time "15s"

# Check required files
testing::phase::check_files \
    ".vrooli/service.json" \
    "README.md" \
    "PRD.md"

# Check required directories
testing::phase::check_directories \
    "api" \
    "cli" \
    "data" \
    "test"

# Validate service.json schema
echo "🔍 Validating service.json..."
if command -v jq >/dev/null 2>&1; then
    if ! jq empty < .vrooli/service.json >/dev/null 2>&1; then
        testing::phase::add_error "❌ Invalid JSON in service.json"
    else
        log::success "✅ service.json is valid JSON"

        # Check required fields
        required_fields=("service.name" "service.version" "ports" "lifecycle")
        for field in "${required_fields[@]}"; do
            if ! jq -e ".$field" < .vrooli/service.json >/dev/null 2>&1; then
                testing::phase::add_error "❌ Missing required field in service.json: $field"
            fi
        done

        if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
            service_name=$(jq -r '.service.name' .vrooli/service.json)
            if [ "$service_name" = "invoice-generator" ]; then
                log::success "✅ service.json contains correct service name"
            else
                testing::phase::add_error "❌ Incorrect service name in service.json: $service_name"
            fi
        fi
    fi
else
    testing::phase::add_warning "⚠️  jq not available, skipping JSON validation"
fi

# Check Go module structure
echo "🔍 Validating Go module..."
if [ -f "api/go.mod" ]; then
    if grep -q "module " api/go.mod; then
        log::success "✅ Go module properly defined"
    else
        testing::phase::add_error "❌ Invalid go.mod structure"
    fi
else
    testing::phase::add_error "❌ go.mod missing"
fi

# Check modern test structure
echo "🔍 Validating test infrastructure..."
test_structure_valid=true

required_phases=("test-dependencies.sh" "test-structure.sh" "test-unit.sh")
for phase in "${required_phases[@]}"; do
    if [ ! -f "test/phases/$phase" ]; then
        testing::phase::add_error "❌ Missing test phase: test/phases/$phase"
        test_structure_valid=false
    fi
done

if [ "$test_structure_valid" = "true" ]; then
    log::success "✅ Test infrastructure present"
fi

# Check Go test files
echo "🔍 Validating test files..."
if [ -f "api/test_helpers.go" ]; then
    log::success "✅ Test helpers found"
else
    testing::phase::add_error "❌ api/test_helpers.go missing"
fi

if [ -f "api/test_patterns.go" ]; then
    log::success "✅ Test patterns found"
else
    testing::phase::add_error "❌ api/test_patterns.go missing"
fi

# Count test files
test_file_count=$(find api -name "*_test.go" 2>/dev/null | wc -l)
if [ "$test_file_count" -gt 0 ]; then
    log::success "✅ Found $test_file_count Go test files"
else
    testing::phase::add_error "❌ No Go test files found"
fi

# Check data directory structure
echo "🔍 Checking data directory..."
if [ -d "data" ]; then
    log::success "✅ Data directory exists"
else
    testing::phase::add_warning "⚠️  data directory missing (will be created on setup)"
fi

# Check database initialization
echo "🔍 Checking database initialization..."
if [ -d "initialization/postgres" ]; then
    if [ -f "initialization/postgres/schema.sql" ]; then
        log::success "✅ Database schema defined"
    else
        testing::phase::add_warning "⚠️  Database schema file missing"
    fi
else
    testing::phase::add_warning "⚠️  Database initialization directory missing"
fi

# End with summary
testing::phase::end_with_summary "Structure validation completed"
