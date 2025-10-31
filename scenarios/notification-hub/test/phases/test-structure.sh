#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Testing scenario structure..."

# Test required files
testing::phase::log "Checking required files..."

required_files=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "Makefile"
    "api/main.go"
    "cli/notification-hub"
    "cli/install.sh"
)

for file in "${required_files[@]}"; do
    if [ -f "$file" ] || [ -L "$file" ]; then
        testing::phase::success "  ✓ $file"
    else
        testing::phase::error "  ✗ $file missing"
        testing::phase::end_with_summary "Required file missing: $file" 1
    fi
done

# Test required directories
testing::phase::log "Checking required directories..."

required_dirs=(
    "api"
    "cli"
    "ui"
    "test"
    "initialization"
)

for dir in "${required_dirs[@]}"; do
    if [ -d "$dir" ]; then
        testing::phase::success "  ✓ $dir/"
    else
        testing::phase::warn "  ✗ $dir/ missing (may be optional)"
    fi
done

# Test service.json structure
testing::phase::log "Validating service.json structure..."

if [ -f .vrooli/service.json ]; then
    required_fields=("version" "service" "ports" "resources" "lifecycle")
    for field in "${required_fields[@]}"; do
        if grep -q "\"$field\"" .vrooli/service.json; then
            testing::phase::success "  ✓ $field field present"
        else
            testing::phase::error "  ✗ $field field missing"
            testing::phase::end_with_summary "Invalid service.json structure" 1
        fi
    done
else
    testing::phase::error "service.json not found"
    testing::phase::end_with_summary "Missing service configuration" 1
fi

# Test Makefile targets
testing::phase::log "Checking Makefile targets..."

if [ -f Makefile ]; then
    required_targets=("help" "start" "stop" "test" "logs" "status")
    for target in "${required_targets[@]}"; do
        if grep -q "^$target:" Makefile; then
            testing::phase::success "  ✓ make $target"
        else
            testing::phase::error "  ✗ make $target missing"
            testing::phase::end_with_summary "Invalid Makefile structure" 1
        fi
    done
else
    testing::phase::error "Makefile not found"
    testing::phase::end_with_summary "Missing Makefile" 1
fi

# Test CLI executable
testing::phase::log "Checking CLI executable..."

if [ -x cli/notification-hub ]; then
    testing::phase::success "CLI binary is executable"
else
    testing::phase::error "CLI binary not executable"
    testing::phase::end_with_summary "CLI setup incomplete" 1
fi

# Test initialization structure
testing::phase::log "Checking initialization structure..."

if [ -d initialization/storage/postgres ]; then
    testing::phase::success "PostgreSQL initialization directory exists"
else
    testing::phase::warn "PostgreSQL initialization directory missing"
fi

testing::phase::end_with_summary "Structure tests completed"
