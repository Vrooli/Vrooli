#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 15-second target
testing::phase::init --target-time "15s"

echo "🔍 Validating recommendation-engine structure..."

# Check required files
echo ""
echo "Checking required files..."
testing::phase::check_files \
    ".vrooli/service.json" \
    "README.md" \
    "PRD.md"

# Check required directories
echo ""
echo "Checking required directories..."
testing::phase::check_directories \
    "api" \
    "test"

# Check optional directories
if [ -d "ui" ]; then
    log::success "✅ UI directory present"
else
    log::info "ℹ️  UI directory not present (optional)"
fi

if [ -d "cli" ]; then
    log::success "✅ CLI directory present"
else
    log::info "ℹ️  CLI directory not present (optional)"
fi

# Validate service.json schema
echo ""
echo "🔍 Validating service.json..."
if command -v jq >/dev/null 2>&1; then
    if ! jq empty < .vrooli/service.json >/dev/null 2>&1; then
        testing::phase::add_error "❌ Invalid JSON in service.json"
    else
        log::success "✅ service.json is valid JSON"
        testing::phase::add_test passed

        # Check required fields
        required_fields=("service.name" "service.version" "ports" "lifecycle")
        for field in "${required_fields[@]}"; do
            if ! jq -e ".$field" < .vrooli/service.json >/dev/null 2>&1; then
                testing::phase::add_error "❌ Missing required field in service.json: $field"
            else
                testing::phase::add_test passed
            fi
        done

        # Validate service name
        service_name=$(jq -r '.service.name' .vrooli/service.json)
        if [ "$service_name" = "recommendation-engine" ]; then
            log::success "✅ service.json contains correct service name"
            testing::phase::add_test passed
        else
            testing::phase::add_error "❌ Incorrect service name in service.json: $service_name"
        fi

        # Check for resources configuration
        if jq -e '.resources' < .vrooli/service.json >/dev/null 2>&1; then
            resources_count=$(jq '.resources | length' .vrooli/service.json)
            log::success "✅ Resources configured: $resources_count resource(s)"
            testing::phase::add_test passed

            # Validate postgres is configured
            if jq -e '.resources.postgres' < .vrooli/service.json >/dev/null 2>&1; then
                log::success "✅ PostgreSQL resource configured"
                testing::phase::add_test passed
            else
                testing::phase::add_warning "⚠️  PostgreSQL resource not configured (may be required)"
                testing::phase::add_test skipped
            fi

            # Check if qdrant is configured
            if jq -e '.resources.qdrant' < .vrooli/service.json >/dev/null 2>&1; then
                log::success "✅ Qdrant resource configured"
                testing::phase::add_test passed
            else
                log::info "ℹ️  Qdrant resource not configured (optional - enables similar items)"
            fi
        else
            testing::phase::add_warning "⚠️  No resources configured in service.json"
            testing::phase::add_test skipped
        fi
    fi
else
    testing::phase::add_warning "⚠️  jq not available, skipping JSON validation"
    testing::phase::add_test skipped
fi

# Check Go module structure
echo ""
echo "🔍 Validating Go API module..."
if [ -f "api/go.mod" ]; then
    if grep -q "module " api/go.mod; then
        module_name=$(grep "^module " api/go.mod | awk '{print $2}')
        log::success "✅ Go module properly defined: $module_name"
        testing::phase::add_test passed
    else
        testing::phase::add_error "❌ Invalid go.mod structure"
    fi

    # Check for main.go
    if [ -f "api/main.go" ]; then
        log::success "✅ API main.go present"
        testing::phase::add_test passed
    else
        testing::phase::add_error "❌ API main.go missing"
    fi

    # Check for test files
    test_files=$(find api -name "*_test.go" 2>/dev/null | wc -l)
    if [ "$test_files" -gt 0 ]; then
        log::success "✅ API test files present: $test_files test file(s)"
        testing::phase::add_test passed
    else
        testing::phase::add_warning "⚠️  No API test files found"
        testing::phase::add_test skipped
    fi
else
    testing::phase::add_error "❌ api/go.mod missing"
fi

# Check for UI if present
if [ -d "ui" ]; then
    echo ""
    echo "🔍 Validating UI structure..."
    if [ -f "ui/package.json" ]; then
        if command -v jq >/dev/null 2>&1; then
            if jq -e '.name' ui/package.json >/dev/null 2>&1; then
                package_name=$(jq -r '.name' ui/package.json)
                log::success "✅ Node.js package properly defined: $package_name"
                testing::phase::add_test passed
            else
                testing::phase::add_error "❌ Invalid package.json structure"
            fi
        fi
    else
        testing::phase::add_warning "⚠️  ui/package.json missing"
        testing::phase::add_test skipped
    fi
fi

# Check test infrastructure
echo ""
echo "🔍 Validating test infrastructure..."
test_structure_valid=true

# Check for test phases directory
if [ -d "test/phases" ]; then
    log::success "✅ Test phases directory exists"
    testing::phase::add_test passed
else
    testing::phase::add_error "❌ Test phases directory missing"
    test_structure_valid=false
fi

# Check for required test phases
required_phases=("test-unit.sh" "test-integration.sh" "test-business.sh" "test-performance.sh" "test-dependencies.sh" "test-structure.sh")
missing_phases=0
for phase in "${required_phases[@]}"; do
    if [ ! -f "test/phases/$phase" ]; then
        testing::phase::add_warning "⚠️  Missing test phase: test/phases/$phase"
        ((missing_phases++))
    else
        if [ -x "test/phases/$phase" ]; then
            testing::phase::add_test passed
        else
            testing::phase::add_warning "⚠️  Test phase not executable: test/phases/$phase"
            testing::phase::add_test skipped
        fi
    fi
done

if [ $missing_phases -eq 0 ]; then
    log::success "✅ All required test phases present"
else
    log::warning "⚠️  $missing_phases test phase(s) missing"
fi

# Check for test helpers
if [ -f "api/test_helpers.go" ]; then
    log::success "✅ Test helpers present"
    testing::phase::add_test passed
else
    testing::phase::add_warning "⚠️  api/test_helpers.go missing"
    testing::phase::add_test skipped
fi

if [ -f "api/test_patterns.go" ]; then
    log::success "✅ Test patterns present"
    testing::phase::add_test passed
else
    testing::phase::add_warning "⚠️  api/test_patterns.go missing"
    testing::phase::add_test skipped
fi

# Check initialization directory
echo ""
echo "🔍 Validating initialization scripts..."
if [ -d "initialization" ]; then
    log::success "✅ Initialization directory exists"
    testing::phase::add_test passed

    # Check for database schema
    if [ -f "initialization/postgres/schema.sql" ] || [ -f "initialization/postgres/init.sql" ]; then
        log::success "✅ PostgreSQL schema present"
        testing::phase::add_test passed
    else
        testing::phase::add_warning "⚠️  PostgreSQL schema not found in initialization/"
        testing::phase::add_test skipped
    fi

    # Check for qdrant setup if applicable
    if [ -f "initialization/qdrant/setup.sh" ] || [ -f "initialization/qdrant/collections.sh" ]; then
        log::success "✅ Qdrant initialization present"
        testing::phase::add_test passed
    else
        log::info "ℹ️  Qdrant initialization not found (optional)"
    fi
else
    testing::phase::add_warning "⚠️  initialization/ directory missing (database setup may be manual)"
    testing::phase::add_test skipped
fi

# Check documentation
echo ""
echo "🔍 Validating documentation..."
docs_score=0
if [ -f "README.md" ]; then
    readme_lines=$(wc -l < README.md)
    if [ "$readme_lines" -gt 20 ]; then
        log::success "✅ README.md present and substantive ($readme_lines lines)"
        testing::phase::add_test passed
        ((docs_score++))
    else
        testing::phase::add_warning "⚠️  README.md is very short ($readme_lines lines)"
        testing::phase::add_test skipped
    fi
fi

if [ -f "PRD.md" ]; then
    log::success "✅ PRD.md present"
    testing::phase::add_test passed
    ((docs_score++))
fi

if [ -f "PROBLEMS.md" ]; then
    log::success "✅ PROBLEMS.md present"
    testing::phase::add_test passed
    ((docs_score++))
fi

if [ $docs_score -ge 2 ]; then
    log::success "✅ Documentation coverage good ($docs_score/3 key files)"
else
    log::warning "⚠️  Limited documentation coverage ($docs_score/3 key files)"
fi

# Check API structure
echo ""
echo "🔍 Validating API code structure..."
api_handlers=0
api_models=0

if grep -q "func.*Handler" api/main.go 2>/dev/null; then
    api_handlers=$(grep -c "func.*Handler" api/main.go)
    log::success "✅ API handlers defined: $api_handlers handler(s)"
    testing::phase::add_test passed
fi

if grep -q "type.*struct" api/main.go 2>/dev/null; then
    api_models=$(grep -c "type.*struct" api/main.go)
    log::success "✅ Data models defined: $api_models model(s)"
    testing::phase::add_test passed
fi

echo ""
echo "📊 Structure Validation Summary:"
echo "═══════════════════════════════════"
echo "Tests run: $TESTING_PHASE_TEST_COUNT"
echo "Errors: $TESTING_PHASE_ERROR_COUNT"
echo "Skipped: $TESTING_PHASE_SKIPPED_COUNT"

if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
    log::success "✅ SUCCESS: All structural requirements met"
else
    log::error "❌ ERROR: $TESTING_PHASE_ERROR_COUNT structural issue(s) found"
fi

# End with summary
testing::phase::end_with_summary "Structure validation completed"
