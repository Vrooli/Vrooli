#!/bin/bash
# Structure validation phase - validates required files, configuration, and directory structure
source "$(dirname "${BASH_SOURCE[0]}")/../../../../scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 15-second target
testing::phase::init --target-time "15s"

# Check required files for data-structurer
testing::phase::check_files \
    ".vrooli/service.json" \
    "README.md" \
    "PRD.md" \
    "PROBLEMS.md" \
    "Makefile"

# Check required directories
testing::phase::check_directories \
    "api" \
    "cli" \
    "initialization" \
    "test" \
    "tests"

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

        # Check service name matches
        expected_name="data-structurer"
        if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
            service_name=$(jq -r '.service.name' .vrooli/service.json)
            if [ "$service_name" = "$expected_name" ]; then
                log::success "✅ service.json contains correct service name"
            else
                testing::phase::add_error "❌ Incorrect service name in service.json: $service_name (expected: $expected_name)"
            fi
        fi
    fi
else
    testing::phase::add_warning "⚠️  jq not available, skipping JSON validation"
fi

# Validate Go module
if [ -f "api/go.mod" ]; then
    if grep -q "module " api/go.mod; then
        log::success "✅ Go module properly defined"
    else
        testing::phase::add_error "❌ Invalid go.mod structure"
    fi
fi

# Check for CLI binary
if [ -f "cli/data-structurer" ]; then
    log::success "✅ CLI binary exists"
else
    testing::phase::add_warning "⚠️  CLI binary not found (may need setup)"
fi

# Check for initialization files
if [ -d "initialization/storage/postgres" ]; then
    log::success "✅ PostgreSQL initialization directory exists"
    if [ -f "initialization/storage/postgres/schema.sql" ]; then
        log::success "✅ Database schema file exists"
    else
        testing::phase::add_error "❌ Missing database schema file"
    fi
fi

# End with summary
testing::phase::end_with_summary "Structure validation completed"
