#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 30-second target
testing::phase::init --target-time "30s"

SERVICE_JSON="$TESTING_PHASE_SCENARIO_DIR/.vrooli/service.json"

check_resource_cli() {
    local resource_name="$1"
    local required_flag="$2"
    local cli_name="resource-${resource_name}"

    if ! command -v "$cli_name" >/dev/null 2>&1; then
        if [ "$required_flag" = "true" ]; then
            testing::phase::add_error "❌ Required resource CLI missing: $cli_name"
        else
            testing::phase::add_warning "⚠️  Optional resource CLI missing: $cli_name"
        fi
        return
    fi

    if "$cli_name" test smoke >/dev/null 2>&1; then
        log::success "✅ ${resource_name^} resource smoke test passed"
        testing::phase::add_test passed
        return
    fi

    if "$cli_name" status >/dev/null 2>&1; then
        log::success "✅ ${resource_name^} resource status OK"
        testing::phase::add_test passed
        return
    fi

    if [ "$required_flag" = "true" ]; then
        testing::phase::add_error "❌ Required resource '$resource_name' is unavailable"
    else
        testing::phase::add_warning "⚠️  Optional resource '$resource_name' could not be verified"
        testing::phase::add_test skipped
    fi
}

echo "🔍 Inspecting declared resources from service.json..."
if [ -f "$SERVICE_JSON" ] && command -v jq >/dev/null 2>&1; then
    mapfile -t RESOURCE_ROWS < <(jq -r '.resources // {} | to_entries[] | "\(.key)|\(.value.required // false)|\(.value.enabled // false)"' "$SERVICE_JSON")

    if [ ${#RESOURCE_ROWS[@]} -eq 0 ]; then
        log::info "ℹ️  No resources declared in service.json"
    else
        for row in "${RESOURCE_ROWS[@]}"; do
            IFS='|' read -r resource_name resource_required resource_enabled <<< "$row"
            # If enabled, treat as required for this test
            if [ "$resource_enabled" = "true" ]; then
                resource_required="true"
            fi
            check_resource_cli "$resource_name" "$resource_required"
        done
    fi
else
    testing::phase::add_warning "⚠️  Unable to parse resources from service.json (missing file or jq)"
    if [ ! -f "$SERVICE_JSON" ]; then
        log::warning "   service.json not found at: $SERVICE_JSON"
    fi
fi

echo ""
echo "🔍 Checking critical resources for recommendation-engine..."

# PostgreSQL is critical
if command -v resource-postgres >/dev/null 2>&1; then
    if resource-postgres test smoke >/dev/null 2>&1; then
        log::success "✅ PostgreSQL database available and operational"
        testing::phase::add_test passed
    else
        testing::phase::add_error "❌ PostgreSQL database unavailable (CRITICAL)"
    fi
else
    testing::phase::add_error "❌ PostgreSQL resource CLI not found (CRITICAL)"
fi

# Qdrant is optional but recommended
if command -v resource-qdrant >/dev/null 2>&1; then
    if resource-qdrant test smoke >/dev/null 2>&1; then
        log::success "✅ Qdrant vector database available"
        testing::phase::add_test passed
    else
        testing::phase::add_warning "⚠️  Qdrant vector database unavailable (optional - limits similar items feature)"
        testing::phase::add_test skipped
    fi
else
    testing::phase::add_warning "⚠️  Qdrant resource CLI not found (optional - similar items will be limited)"
    testing::phase::add_test skipped
fi

echo ""
echo "🔍 Checking language toolchains..."

# Go is required for API
if command -v go >/dev/null 2>&1; then
    go_version=$(go version | awk '{print $3}')
    log::success "✅ Go available: $go_version"
    testing::phase::add_test passed

    # Check Go module dependencies
    if [ -f "$TESTING_PHASE_SCENARIO_DIR/api/go.mod" ]; then
        if cd "$TESTING_PHASE_SCENARIO_DIR/api" && go mod verify >/dev/null 2>&1; then
            log::success "✅ Go module dependencies verified"
            testing::phase::add_test passed
        else
            testing::phase::add_error "❌ Go module dependencies verification failed"
        fi
    fi
else
    testing::phase::add_error "❌ Go toolchain not found (CRITICAL)"
fi

# Check for UI dependencies if applicable
if [ -f "$TESTING_PHASE_SCENARIO_DIR/ui/package.json" ]; then
    echo ""
    echo "🔍 Checking Node.js and UI dependencies..."

    if command -v node >/dev/null 2>&1; then
        node_version=$(node --version)
        log::success "✅ Node.js available: $node_version"
        testing::phase::add_test passed
    else
        testing::phase::add_error "❌ Node.js runtime not found"
    fi

    if command -v npm >/dev/null 2>&1; then
        npm_version=$(npm --version)
        log::success "✅ npm available: $npm_version"
        testing::phase::add_test passed

        # Check if node_modules exists
        if [ -d "$TESTING_PHASE_SCENARIO_DIR/ui/node_modules" ]; then
            log::success "✅ UI dependencies installed"
            testing::phase::add_test passed
        else
            testing::phase::add_warning "⚠️  UI node_modules not found - run 'npm install' in ui/"
            testing::phase::add_test skipped
        fi
    else
        testing::phase::add_warning "⚠️  npm not found (Node.js tests may fail)"
        testing::phase::add_test skipped
    fi
fi

echo ""
echo "🔍 Checking essential utilities..."
essential_tools=(jq curl)
for tool in "${essential_tools[@]}"; do
    if command -v "$tool" >/dev/null 2>&1; then
        version_output=$("$tool" --version 2>&1 | head -1)
        log::success "✅ $tool available ($version_output)"
        testing::phase::add_test passed
    else
        testing::phase::add_error "❌ Required utility missing: $tool"
    fi
done

echo ""
echo "🔍 Checking recommended utilities..."
recommended_tools=(bc)
for tool in "${recommended_tools[@]}"; do
    if command -v "$tool" >/dev/null 2>&1; then
        log::success "✅ $tool available"
        testing::phase::add_test passed
    else
        testing::phase::add_warning "⚠️  Recommended utility missing: $tool (performance tests may be limited)"
        testing::phase::add_test skipped
    fi
done

echo ""
echo "🔍 Validating Go build requirements..."
if [ -f "$TESTING_PHASE_SCENARIO_DIR/api/main.go" ]; then
    # Check if API can compile
    if cd "$TESTING_PHASE_SCENARIO_DIR/api" && go build -o /tmp/recommendation-engine-test-build . >/dev/null 2>&1; then
        log::success "✅ API builds successfully"
        rm -f /tmp/recommendation-engine-test-build
        testing::phase::add_test passed
    else
        testing::phase::add_error "❌ API build failed"
    fi
else
    testing::phase::add_warning "⚠️  API main.go not found"
    testing::phase::add_test skipped
fi

echo ""
echo "🔍 Validating database connectivity..."
# Check if database connection environment variables are set
db_vars_missing=0
for var in POSTGRES_HOST POSTGRES_PORT POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB; do
    if [ -z "${!var}" ]; then
        testing::phase::add_warning "⚠️  Environment variable $var not set"
        ((db_vars_missing++))
    fi
done

if [ $db_vars_missing -eq 0 ]; then
    log::success "✅ Database environment variables configured"
    testing::phase::add_test passed
else
    testing::phase::add_warning "⚠️  $db_vars_missing database environment variable(s) missing"
    testing::phase::add_test skipped
fi

echo ""
echo "📊 Dependencies Summary:"
echo "═══════════════════════════════"
echo "Tests run: $TESTING_PHASE_TEST_COUNT"
echo "Errors: $TESTING_PHASE_ERROR_COUNT"
echo "Skipped: $TESTING_PHASE_SKIPPED_COUNT"

if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
    log::success "✅ SUCCESS: All critical dependencies available"
else
    log::error "❌ ERROR: $TESTING_PHASE_ERROR_COUNT critical dependency issue(s) found"
fi

# End with summary
testing::phase::end_with_summary "Dependencies validation completed"
