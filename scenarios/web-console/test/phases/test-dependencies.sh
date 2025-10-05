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
        return
    fi

    if "$cli_name" status >/dev/null 2>&1; then
        log::success "✅ ${resource_name^} resource status OK"
        return
    fi

    if [ "$required_flag" = "true" ]; then
        testing::phase::add_error "❌ Required resource '$resource_name' is unavailable"
    else
        testing::phase::add_warning "⚠️  Optional resource '$resource_name' could not be verified"
    fi
}

if [ -f "$SERVICE_JSON" ] && command -v jq >/dev/null 2>&1; then
    echo "🔍 Inspecting declared resources..."
    mapfile -t RESOURCE_ROWS < <(jq -r '.resources // {} | to_entries[] | "\(.key)|\(.value.required // false)|\(.value.enabled // false)"' "$SERVICE_JSON")

    if [ ${#RESOURCE_ROWS[@]} -eq 0 ]; then
        log::info "ℹ️  No resources declared in service.json"
    else
        for row in "${RESOURCE_ROWS[@]}"; do
            IFS='|' read -r resource_name resource_required resource_enabled <<< "$row"
            if [ "$resource_enabled" = "true" ]; then
                resource_required="true"
            fi
            check_resource_cli "$resource_name" "$resource_required"
        done
    fi
else
    testing::phase::add_warning "⚠️  Unable to parse resources from service.json (missing file or jq)"
fi

echo "🔍 Checking language toolchains..."
if command -v go >/dev/null 2>&1; then
    go_version=$(go version | awk '{print $3}')
    log::success "✅ Go available: $go_version"
else
    testing::phase::add_error "❌ Go toolchain not found"
fi

# Node.js is optional for web-console (static files only)
if command -v node >/dev/null 2>&1; then
    node_version=$(node --version)
    log::success "✅ Node.js available: $node_version"
else
    log::info "ℹ️  Node.js not found (optional for static UI)"
fi

echo "🔍 Checking essential utilities..."
essential_tools=(jq curl)
for tool in "${essential_tools[@]}"; do
    if command -v "$tool" >/dev/null 2>&1; then
        version_output=$("$tool" --version 2>&1 | head -1)
        log::success "✅ $tool available ($version_output)"
    else
        testing::phase::add_error "❌ Required utility missing: $tool"
    fi
done

echo "🔍 Checking Go dependencies..."
cd "$TESTING_PHASE_SCENARIO_DIR/api" || exit 1
if go mod verify &> /dev/null; then
    log::success "✅ Go module dependencies verified"
else
    testing::phase::add_error "❌ Go module verification failed"
fi

# End with summary
testing::phase::end_with_summary "Dependencies validation completed"
