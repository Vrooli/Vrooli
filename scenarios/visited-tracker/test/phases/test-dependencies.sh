#!/bin/bash
# Dependencies validation phase - <30 seconds
# Resolves required resources from service.json and verifies language/tooling prerequisites
set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

echo "=== Dependencies Phase (Target: <30s) ==="
start_time=$(date +%s)

error_count=0
warning_count=0

SCENARIO_NAME=$(basename "$SCENARIO_DIR")
SERVICE_JSON="$SCENARIO_DIR/.vrooli/service.json"

check_resource_cli() {
    local resource_name="$1"
    local required_flag="$2"
    local cli_name="resource-${resource_name}"

    if ! command -v "$cli_name" >/dev/null 2>&1; then
        if [ "$required_flag" = "true" ]; then
            log::error "❌ Required resource CLI missing: $cli_name"
            ((error_count++))
        else
            log::warning "⚠️  Optional resource CLI missing: $cli_name"
            ((warning_count++))
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
        log::error "❌ Required resource '$resource_name' is unavailable"
        ((error_count++))
    else
        log::warning "⚠️  Optional resource '$resource_name' could not be verified"
        ((warning_count++))
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
    log::warning "⚠️  Unable to parse resources from service.json (missing file or jq)"
    if [ ! -f "$SERVICE_JSON" ]; then
        ((warning_count++))
    fi
fi

echo "🔍 Checking language toolchains..."
if command -v go >/dev/null 2>&1; then
    go_version=$(go version | awk '{print $3}')
    log::success "✅ Go available: $go_version"
else
    log::error "❌ Go toolchain not found"
    ((error_count++))
fi

if command -v node >/dev/null 2>&1; then
    node_version=$(node --version)
    log::success "✅ Node.js available: $node_version"
else
    log::error "❌ Node.js runtime not found"
    ((error_count++))
fi

if command -v npm >/dev/null 2>&1; then
    npm_version=$(npm --version)
    log::success "✅ npm available: $npm_version"
else
    log::warning "⚠️  npm not found (Node.js tests may fail)"
    ((warning_count++))
fi

echo "🔍 Checking essential utilities..."
essential_tools=(jq curl)
for tool in "${essential_tools[@]}"; do
    if command -v "$tool" >/dev/null 2>&1; then
        version_output=$("$tool" --version 2>&1 | head -1)
        log::success "✅ $tool available ($version_output)"
    else
        log::error "❌ Required utility missing: $tool"
        ((error_count++))
    fi
done

end_time=$(date +%s)
duration=$((end_time - start_time))
echo ""

if [ $error_count -eq 0 ]; then
    if [ $warning_count -eq 0 ]; then
        log::success "✅ Dependencies validation completed successfully in ${duration}s"
    else
        log::success "✅ Dependencies validation completed with $warning_count warnings in ${duration}s"
    fi
else
    log::error "❌ Dependencies validation failed with $error_count errors and $warning_count warnings in ${duration}s"
fi

if [ $duration -gt 30 ]; then
    log::warning "⚠️  Dependencies phase exceeded 30s target"
fi

if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi
