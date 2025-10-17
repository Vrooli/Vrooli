#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "20s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Validating funnel-builder scenario structure..."

# Validate service.json structure
if [ -f ".vrooli/service.json" ]; then
    testing::phase::log "Validating service.json..."

    if command -v jq &>/dev/null; then
        # Check JSON is valid
        if ! jq empty .vrooli/service.json 2>/dev/null; then
            testing::phase::error "service.json is not valid JSON"
            testing::phase::end_with_summary "Invalid service.json" 1
        fi

        # Check required fields
        required_fields=("version" "service" "ports" "lifecycle")
        for field in "${required_fields[@]}"; do
            if ! jq -e ".$field" .vrooli/service.json >/dev/null 2>&1; then
                testing::phase::error "Missing required field in service.json: $field"
                testing::phase::end_with_summary "Invalid service.json structure" 1
            fi
        done

        testing::phase::success "service.json structure valid"
    else
        testing::phase::warn "jq not available, skipping service.json validation"
    fi
else
    testing::phase::error ".vrooli/service.json not found"
    testing::phase::end_with_summary "Missing service.json" 1
fi

# Check Go API structure
if [ -f "api/main.go" ]; then
    testing::phase::log "Checking Go API structure..."

    cd api

    # Check if code compiles
    if ! go build -o /tmp/funnel-builder-test . 2>/dev/null; then
        testing::phase::error "Go code does not compile"
        testing::phase::end_with_summary "Compilation failed" 1
    fi
    rm -f /tmp/funnel-builder-test

    # Check for required functions
    if ! grep -q "func main()" main.go; then
        testing::phase::error "main() function not found in main.go"
        testing::phase::end_with_summary "Missing main function" 1
    fi

    # Check for HTTP server setup (Gin framework)
    if ! grep -q "gin\\.New\\|gin\\.Default" main.go; then
        testing::phase::error "Gin server setup not found"
        testing::phase::end_with_summary "Missing HTTP server" 1
    fi

    # Lint Go code
    if command -v golangci-lint &>/dev/null; then
        testing::phase::log "Running golangci-lint..."
        if ! golangci-lint run --timeout=2m 2>&1 | head -20; then
            testing::phase::warn "Linting issues found (non-fatal)"
        fi
    fi

    cd ..
    testing::phase::success "Go API structure valid"
else
    testing::phase::error "api/main.go not found"
    testing::phase::end_with_summary "Missing main.go" 1
fi

# Check UI structure
if [ -d "ui" ]; then
    testing::phase::log "Checking UI structure..."

    if [ ! -f "ui/package.json" ]; then
        testing::phase::error "ui/package.json not found"
        testing::phase::end_with_summary "Missing package.json" 1
    fi

    # Check for React setup
    if [ -f "ui/src/main.tsx" ] || [ -f "ui/src/main.jsx" ]; then
        testing::phase::success "React entry point found"
    else
        testing::phase::warn "No React entry point found (main.tsx/jsx)"
    fi

    # Check for required UI components
    if [ -f "ui/src/App.tsx" ] || [ -f "ui/src/App.jsx" ]; then
        testing::phase::success "App component found"
    else
        testing::phase::warn "App.tsx/jsx not found"
    fi

    testing::phase::success "UI structure valid"
else
    testing::phase::warn "ui/ directory not found"
fi

# Check CLI structure
if [ -f "cli/funnel-builder" ]; then
    testing::phase::log "Checking CLI structure..."

    # Verify CLI is executable
    if [ ! -x "cli/funnel-builder" ]; then
        testing::phase::error "CLI binary is not executable"
        testing::phase::end_with_summary "CLI not executable" 1
    fi

    # Check for install script
    if [ -f "cli/install.sh" ]; then
        testing::phase::success "CLI install script found"
    else
        testing::phase::warn "cli/install.sh not found"
    fi

    testing::phase::success "CLI structure valid"
else
    testing::phase::error "cli/funnel-builder not found"
    testing::phase::end_with_summary "Missing CLI binary" 1
fi

# Check database initialization structure
if [ -d "initialization/storage/postgres" ]; then
    testing::phase::log "Checking database initialization..."

    if [ -f "initialization/storage/postgres/schema.sql" ]; then
        testing::phase::success "Database schema found"
    else
        testing::phase::warn "schema.sql not found"
    fi

    if [ -f "initialization/storage/postgres/seed.sql" ]; then
        testing::phase::success "Database seed data found"
    else
        testing::phase::warn "seed.sql not found"
    fi
else
    testing::phase::warn "Database initialization directory not found"
fi

# Check test structure
testing::phase::log "Checking test structure..."

if [ -d "test/phases" ]; then
    test_count=$(find test/phases -name "test-*.sh" | wc -l)
    testing::phase::log "Found $test_count test phase scripts"

    if [ "$test_count" -eq 0 ]; then
        testing::phase::warn "No test phase scripts found"
    fi
fi

if [ -d "api" ]; then
    go_test_count=$(find api -name "*_test.go" | wc -l)
    testing::phase::log "Found $go_test_count Go test files"

    if [ "$go_test_count" -eq 0 ]; then
        testing::phase::warn "No Go test files found in api/"
    fi
fi

# Check for documentation
testing::phase::log "Checking documentation..."
doc_files=("README.md" "PRD.md")

for doc in "${doc_files[@]}"; do
    if [ -f "$doc" ]; then
        testing::phase::success "Found $doc"
    else
        testing::phase::warn "Missing $doc"
    fi
done

# Validate Makefile targets
if [ -f "Makefile" ]; then
    testing::phase::log "Checking Makefile targets..."

    required_targets=("start" "stop" "test" "logs" "help")
    for target in "${required_targets[@]}"; do
        if grep -q "^$target:" Makefile || grep -q "^\\.$target:" Makefile; then
            testing::phase::success "Makefile has '$target' target"
        else
            testing::phase::warn "Makefile missing '$target' target"
        fi
    done
else
    testing::phase::warn "Makefile not found"
fi

# Check for legacy test format
if [ -f "scenario-test.yaml" ]; then
    testing::phase::warn "Legacy scenario-test.yaml found - consider migration to phased testing"
fi

testing::phase::end_with_summary "Structure validation completed"
