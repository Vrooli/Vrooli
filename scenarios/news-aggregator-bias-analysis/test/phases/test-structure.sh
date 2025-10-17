#!/bin/bash
# Structure tests for news-aggregator-bias-analysis scenario

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
    "Makefile"

# Check required directories
testing::phase::check_directories \
    "api" \
    "cli" \
    "ui" \
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
            if [ "$service_name" = "news-aggregator-bias-analysis" ]; then
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

# Check Node.js package.json structure
echo "🔍 Validating Node.js package..."
if [ -f "ui/package.json" ]; then
    if command -v jq >/dev/null 2>&1; then
        if jq -e '.name' ui/package.json >/dev/null 2>&1; then
            package_name=$(jq -r '.name' ui/package.json)
            log::success "✅ Node.js package properly defined: $package_name"
        else
            testing::phase::add_error "❌ Invalid package.json structure"
        fi
    fi
else
    testing::phase::add_error "❌ ui/package.json missing"
fi

# Check CLI structure
echo "🔍 Validating CLI tooling..."
if [ -f "cli/install.sh" ]; then
    log::success "✅ CLI install script present"
else
    testing::phase::add_warning "⚠️  cli/install.sh missing"
fi

# Check modern test structure
echo "🔍 Validating test infrastructure..."
test_structure_valid=true

required_phases=("test-structure.sh" "test-dependencies.sh" "test-unit.sh" "test-integration.sh" "test-performance.sh")
for phase in "${required_phases[@]}"; do
    if [ ! -f "test/phases/$phase" ]; then
        testing::phase::add_error "❌ Missing test phase: test/phases/$phase"
        test_structure_valid=false
    fi
done

if [ "$test_structure_valid" = "true" ]; then
    log::success "✅ Test infrastructure complete"
fi

# Check API test files
echo "🔍 Checking API test files..."
api_test_files=("test_helpers.go" "test_patterns.go" "main_test.go" "processor_test.go" "performance_test.go")
for test_file in "${api_test_files[@]}"; do
    if [ -f "api/$test_file" ]; then
        log::success "✅ API test file present: $test_file"
    else
        testing::phase::add_warning "⚠️  API test file missing: $test_file"
    fi
done

# Check initialization directory structure
echo "🔍 Checking initialization directory..."
if [ -d "initialization" ]; then
    if [ -d "initialization/postgres" ]; then
        if [ -f "initialization/postgres/schema.sql" ]; then
            log::success "✅ Database initialization structure correct"
        else
            testing::phase::add_warning "⚠️  schema.sql missing in initialization/postgres"
        fi
    else
        testing::phase::add_warning "⚠️  initialization/postgres directory missing"
    fi
else
    testing::phase::add_warning "⚠️  initialization directory missing"
fi

# Check binary names in lifecycle
echo "🔍 Validating binary names..."
if command -v jq >/dev/null 2>&1 && [ -f ".vrooli/service.json" ]; then
    # Check that binary paths exist in lifecycle
    if jq -e '.lifecycle.develop.steps[] | select(.run | contains("news-aggregator-bias-analysis-api"))' .vrooli/service.json >/dev/null 2>&1; then
        log::success "✅ API binary referenced in lifecycle"
    else
        testing::phase::add_warning "⚠️  API binary not found in lifecycle configuration"
    fi
fi

# End with summary
testing::phase::end_with_summary "Structure validation completed"
