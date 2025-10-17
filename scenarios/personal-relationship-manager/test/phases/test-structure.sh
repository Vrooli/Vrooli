#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "30s"

echo "🔍 Validating scenario structure..."

# Check required directories
REQUIRED_DIRS=(
    "api"
    "test"
    "test/phases"
)

for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d "$TESTING_PHASE_SCENARIO_DIR/$dir" ]; then
        log::success "✅ Required directory exists: $dir"
    else
        testing::phase::add_error "❌ Missing required directory: $dir"
    fi
done

# Check required files
REQUIRED_FILES=(
    ".vrooli/service.json"
    "api/main.go"
    "api/relationship_processor.go"
    "PRD.md"
)

for file in "${REQUIRED_FILES[@]}"; do
    if [ -f "$TESTING_PHASE_SCENARIO_DIR/$file" ]; then
        log::success "✅ Required file exists: $file"
    else
        testing::phase::add_error "❌ Missing required file: $file"
    fi
done

# Validate service.json structure
if [ -f "$TESTING_PHASE_SCENARIO_DIR/.vrooli/service.json" ]; then
    echo "🔍 Validating service.json structure..."

    if command -v jq >/dev/null 2>&1; then
        # Check required fields
        if jq -e '.name' "$TESTING_PHASE_SCENARIO_DIR/.vrooli/service.json" >/dev/null 2>&1; then
            log::success "✅ service.json has 'name' field"
        else
            testing::phase::add_error "❌ service.json missing 'name' field"
        fi

        if jq -e '.description' "$TESTING_PHASE_SCENARIO_DIR/.vrooli/service.json" >/dev/null 2>&1; then
            log::success "✅ service.json has 'description' field"
        else
            testing::phase::add_warning "⚠️  service.json missing 'description' field"
        fi

        if jq -e '.lifecycle' "$TESTING_PHASE_SCENARIO_DIR/.vrooli/service.json" >/dev/null 2>&1; then
            log::success "✅ service.json has 'lifecycle' configuration"
        else
            testing::phase::add_warning "⚠️  service.json missing 'lifecycle' configuration"
        fi
    else
        testing::phase::add_warning "⚠️  jq not available, skipping service.json validation"
    fi
fi

# Check Go module setup
if [ -f "$TESTING_PHASE_SCENARIO_DIR/api/go.mod" ]; then
    log::success "✅ Go module file exists"

    # Verify module name
    if grep -q "module personal-relationship-manager" "$TESTING_PHASE_SCENARIO_DIR/api/go.mod"; then
        log::success "✅ Go module name is correct"
    else
        testing::phase::add_warning "⚠️  Go module name may be incorrect"
    fi
else
    testing::phase::add_error "❌ Missing go.mod file"
fi

# Check test infrastructure
echo "🔍 Validating test infrastructure..."

TEST_FILES=(
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/main_test.go"
)

for file in "${TEST_FILES[@]}"; do
    if [ -f "$TESTING_PHASE_SCENARIO_DIR/$file" ]; then
        log::success "✅ Test file exists: $file"
    else
        testing::phase::add_warning "⚠️  Missing test file: $file"
    fi
done

# Validate test phase scripts
TEST_PHASE_SCRIPTS=(
    "test/phases/test-unit.sh"
    "test/phases/test-dependencies.sh"
    "test/phases/test-structure.sh"
)

for script in "${TEST_PHASE_SCRIPTS[@]}"; do
    if [ -f "$TESTING_PHASE_SCENARIO_DIR/$script" ]; then
        if [ -x "$TESTING_PHASE_SCENARIO_DIR/$script" ]; then
            log::success "✅ Test phase script exists and is executable: $script"
        else
            testing::phase::add_warning "⚠️  Test phase script not executable: $script"
        fi
    else
        testing::phase::add_warning "⚠️  Missing test phase script: $script"
    fi
done

# Check initialization directory
if [ -d "$TESTING_PHASE_SCENARIO_DIR/initialization/postgres" ]; then
    log::success "✅ PostgreSQL initialization directory exists"

    if [ -f "$TESTING_PHASE_SCENARIO_DIR/initialization/postgres/schema.sql" ]; then
        log::success "✅ PostgreSQL schema file exists"
    else
        testing::phase::add_warning "⚠️  Missing PostgreSQL schema file"
    fi
else
    testing::phase::add_warning "⚠️  Missing PostgreSQL initialization directory"
fi

# Validate Go source structure
echo "🔍 Validating Go source code structure..."

if command -v go >/dev/null 2>&1; then
    cd "$TESTING_PHASE_SCENARIO_DIR/api" || exit 1

    # Check for build tags in test files
    if grep -q "// +build testing" ./*_test.go 2>/dev/null; then
        log::success "✅ Test files use build tags"
    else
        testing::phase::add_warning "⚠️  Test files may be missing build tags"
    fi

    # Verify main package
    if grep -q "^package main" main.go 2>/dev/null; then
        log::success "✅ Main package declaration found"
    else
        testing::phase::add_error "❌ Missing or incorrect package declaration in main.go"
    fi
fi

# End with summary
testing::phase::end_with_summary "Structure validation completed"
