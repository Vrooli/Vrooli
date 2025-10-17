#!/bin/bash
# Dependency tests for scenario-to-extension

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::info "Starting dependency tests for scenario-to-extension"

# Check Go dependencies
testing::phase::step "Checking Go dependencies"
cd api
if ! go mod verify &>/dev/null; then
    testing::phase::error "Go module verification failed"
    testing::phase::end_with_summary "Dependency tests failed"
    exit 1
fi
testing::phase::success "Go dependencies verified"

# Check for required Go packages
testing::phase::step "Checking required Go packages"
REQUIRED_PACKAGES=(
    "github.com/gorilla/mux"
    "github.com/rs/cors"
)

for pkg in "${REQUIRED_PACKAGES[@]}"; do
    if ! grep -q "$pkg" go.mod; then
        testing::phase::error "Required package missing: $pkg"
        testing::phase::end_with_summary "Dependency tests failed"
        exit 1
    fi
done
testing::phase::success "All required Go packages present"

cd ..

# Check UI dependencies (if Node.js UI exists)
if [ -f "ui/package.json" ]; then
    testing::phase::step "Checking UI dependencies"
    cd ui
    if [ ! -d "node_modules" ]; then
        testing::phase::warn "UI dependencies not installed, running npm install..."
        npm install &>/dev/null || {
            testing::phase::error "Failed to install UI dependencies"
            testing::phase::end_with_summary "Dependency tests failed"
            exit 1
        }
    fi
    testing::phase::success "UI dependencies checked"
    cd ..
fi

# Test browserless resource availability (optional dependency)
testing::phase::step "Checking browserless resource availability"
BROWSERLESS_URL="${BROWSERLESS_URL:-http://localhost:3000}"
if curl -sf "$BROWSERLESS_URL/pressure" &>/dev/null; then
    testing::phase::success "Browserless resource is available at $BROWSERLESS_URL"
else
    testing::phase::warn "Browserless resource not available at $BROWSERLESS_URL (optional dependency)"
fi

# Check template dependencies
testing::phase::step "Validating template file dependencies"
if [ ! -f "templates/vanilla/manifest.json" ]; then
    testing::phase::error "Missing required template: templates/vanilla/manifest.json"
    testing::phase::end_with_summary "Dependency tests failed"
    exit 1
fi

TEMPLATE_FILES=(
    "templates/vanilla/background.js"
    "templates/vanilla/content.js"
    "templates/vanilla/popup.html"
    "templates/vanilla/popup.js"
    "templates/vanilla/build.js"
)

MISSING_TEMPLATES=()
for template in "${TEMPLATE_FILES[@]}"; do
    if [ ! -f "$template" ]; then
        MISSING_TEMPLATES+=("$template")
    fi
done

if [ ${#MISSING_TEMPLATES[@]} -gt 0 ]; then
    testing::phase::error "Missing template files:"
    for template in "${MISSING_TEMPLATES[@]}"; do
        testing::phase::error "  - $template"
    done
    testing::phase::end_with_summary "Dependency tests failed"
    exit 1
fi
testing::phase::success "All template dependencies present"

# Check CLI dependencies
testing::phase::step "Checking CLI dependencies"
if [ ! -x "cli/install.sh" ]; then
    testing::phase::error "CLI install script not executable"
    testing::phase::end_with_summary "Dependency tests failed"
    exit 1
fi
testing::phase::success "CLI dependencies verified"

# Verify API binary can be built
testing::phase::step "Testing API binary compilation"
cd api
if ! go build -o scenario-to-extension-api-test . &>/dev/null; then
    testing::phase::error "Failed to compile API binary"
    testing::phase::end_with_summary "Dependency tests failed"
    exit 1
fi
rm -f scenario-to-extension-api-test
testing::phase::success "API binary compiles successfully"
cd ..

testing::phase::info "All dependency tests passed"
testing::phase::end_with_summary "Dependency tests completed successfully"
