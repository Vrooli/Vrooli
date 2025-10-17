#!/bin/bash
#
# Structure Test Phase for knowledge-observatory
# Integrates with centralized Vrooli testing infrastructure
#

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "20s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Validating scenario structure..."

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

        # Check lifecycle version is 2.0.0
        lifecycle_version=$(jq -r '.lifecycle.version' .vrooli/service.json)
        if [ "$lifecycle_version" = "2.0.0" ]; then
            testing::phase::success "Using v2.0 lifecycle specification"
        else
            testing::phase::warn "Not using v2.0 lifecycle specification (found: $lifecycle_version)"
        fi

        # Check required lifecycle phases
        required_phases=("setup" "develop" "test" "stop")
        for phase in "${required_phases[@]}"; do
            if jq -e ".lifecycle.$phase" .vrooli/service.json >/dev/null 2>&1; then
                testing::phase::success "Has '$phase' lifecycle phase"
            else
                testing::phase::error "Missing required lifecycle phase: $phase"
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

# Check Go code structure
if [ -f "api/main.go" ]; then
    testing::phase::log "Checking Go code structure..."

    cd api

    # Check if code compiles
    if ! go build -o /tmp/knowledge-observatory-test . 2>/dev/null; then
        testing::phase::error "Go code does not compile"
        cd ..
        testing::phase::end_with_summary "Compilation failed" 1
    fi
    rm -f /tmp/knowledge-observatory-test

    # Check for required functions
    if ! grep -q "func main()" main.go; then
        testing::phase::error "main() function not found in main.go"
        cd ..
        testing::phase::end_with_summary "Missing main function" 1
    fi

    # Check for HTTP server setup
    if grep -q "http.ListenAndServe\|mux.NewRouter\|router :=" main.go; then
        testing::phase::success "HTTP server setup found"
    else
        testing::phase::error "HTTP server setup not found"
    fi

    # Check for required handlers
    required_handlers=("healthHandler" "searchHandler" "graphHandler")
    for handler in "${required_handlers[@]}"; do
        if grep -q "func.*$handler" main.go; then
            testing::phase::success "Found $handler"
        else
            testing::phase::warn "Missing $handler (may be named differently)"
        fi
    done

    # Lint Go code if available
    if command -v golangci-lint &>/dev/null; then
        testing::phase::log "Running golangci-lint..."
        if golangci-lint run --timeout=2m 2>&1 | head -20; then
            testing::phase::success "Linting passed"
        else
            testing::phase::warn "Linting issues found (non-fatal)"
        fi
    fi

    cd ..
    testing::phase::success "Go code structure valid"
else
    testing::phase::error "api/main.go not found"
    testing::phase::end_with_summary "Missing main.go" 1
fi

# Check CLI structure
if [ -d "cli" ]; then
    testing::phase::log "Checking CLI structure..."

    if [ -f "cli/knowledge-observatory" ] || [ -f "cli/install.sh" ]; then
        testing::phase::success "CLI files present"

        if [ -f "cli/install.sh" ]; then
            if grep -q "#!/bin/bash\|#!/usr/bin/env bash" cli/install.sh; then
                testing::phase::success "CLI install script has proper shebang"
            fi
        fi

        # Check for BATS test file
        if [ -f "cli/knowledge-observatory.bats" ]; then
            testing::phase::success "CLI BATS tests found"
        else
            testing::phase::warn "CLI BATS tests not found"
        fi
    else
        testing::phase::warn "CLI files missing"
    fi
else
    testing::phase::warn "cli/ directory not found"
fi

# Check UI structure
if [ -d "ui" ]; then
    testing::phase::log "Checking UI structure..."

    if [ -f "ui/server.js" ] || [ -f "ui/app.js" ] || [ -f "ui/index.html" ]; then
        testing::phase::success "UI files present"

        # Check for package.json
        if [ -f "ui/package.json" ]; then
            testing::phase::success "UI package.json found"
        else
            testing::phase::warn "UI package.json not found"
        fi

        # Check for node_modules (should exist after setup)
        if [ -d "ui/node_modules" ]; then
            testing::phase::success "UI dependencies installed"
        else
            testing::phase::warn "UI node_modules not found (dependencies may not be installed)"
        fi
    else
        testing::phase::warn "UI entry files missing"
    fi
else
    testing::phase::warn "ui/ directory not found"
fi

# Check initialization structure
if [ -d "initialization" ]; then
    testing::phase::log "Checking initialization structure..."

    # Check for postgres initialization
    if [ -d "initialization/postgres" ]; then
        if [ -f "initialization/postgres/schema.sql" ]; then
            testing::phase::success "PostgreSQL schema found"
        else
            testing::phase::warn "PostgreSQL schema.sql not found"
        fi
    fi

    # Check for n8n workflows
    if [ -d "initialization/n8n" ]; then
        workflow_count=$(find initialization/n8n -name "*.json" | wc -l)
        testing::phase::log "Found $workflow_count n8n workflow(s)"
        if [ "$workflow_count" -gt 0 ]; then
            testing::phase::success "n8n workflows present"
        fi
    fi
else
    testing::phase::warn "initialization/ directory not found"
fi

# Check test structure
testing::phase::log "Checking test structure..."

if [ -d "test/phases" ]; then
    test_count=$(find test/phases -name "test-*.sh" | wc -l)
    testing::phase::log "Found $test_count test phase scripts"

    if [ "$test_count" -eq 0 ]; then
        testing::phase::warn "No test phase scripts found"
    else
        testing::phase::success "Test phase scripts present"
    fi

    # Check each test script is executable
    for test_script in test/phases/test-*.sh; do
        if [ -f "$test_script" ]; then
            if [ -x "$test_script" ]; then
                testing::phase::success "$(basename "$test_script") is executable"
            else
                testing::phase::warn "$(basename "$test_script") is not executable"
            fi
        fi
    done
fi

if [ -d "api" ]; then
    go_test_count=$(find api -name "*_test.go" 2>/dev/null | wc -l)
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

    required_targets=("start" "run" "stop" "test" "logs" "status")
    for target in "${required_targets[@]}"; do
        if grep -q "^$target:" Makefile || grep -q "^\\.$target:" Makefile; then
            testing::phase::success "Makefile has '$target' target"
        else
            testing::phase::warn "Makefile missing '$target' target"
        fi
    done
fi

testing::phase::end_with_summary "Structure validation completed"
