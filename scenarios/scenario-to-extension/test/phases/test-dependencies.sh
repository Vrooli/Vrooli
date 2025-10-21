#!/bin/bash
# Dependency tests for scenario-to-extension
source "$(dirname "${BASH_SOURCE[0]}")/../../../../scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

# Check Go dependencies
echo "🔍 Checking Go dependencies..."
cd api || testing::phase::add_error "❌ Failed to cd into api directory"
if ! go mod verify &>/dev/null; then
    testing::phase::add_error "❌ Go module verification failed"
else
    log::success "✅ Go dependencies verified"
fi

# Check for required Go packages
REQUIRED_PACKAGES=(
    "github.com/gorilla/mux"
    "github.com/rs/cors"
)

for pkg in "${REQUIRED_PACKAGES[@]}"; do
    if ! grep -q "$pkg" go.mod; then
        testing::phase::add_error "❌ Required package missing: $pkg"
    else
        log::success "✅ Package present: $pkg"
    fi
done

cd ..

# Check UI dependencies (if Node.js UI exists)
if [ -f "ui/package.json" ]; then
    echo "🔍 Checking UI dependencies..."
    if [ ! -d "ui/node_modules" ]; then
        testing::phase::add_warning "⚠️  UI dependencies not installed"
    else
        log::success "✅ UI dependencies present"
    fi
fi

# Test browserless resource availability (optional dependency)
echo "🔍 Checking browserless resource availability..."
BROWSERLESS_URL="${BROWSERLESS_URL:-http://localhost:3000}"
if curl -sf "$BROWSERLESS_URL/pressure" &>/dev/null; then
    log::success "✅ Browserless resource is available at $BROWSERLESS_URL"
else
    testing::phase::add_warning "⚠️  Browserless resource not available at $BROWSERLESS_URL (optional dependency)"
fi

# Check template dependencies
echo "🔍 Validating template file dependencies..."
testing::phase::check_files \
    "templates/vanilla/manifest.json" \
    "templates/vanilla/background.js" \
    "templates/vanilla/content.js" \
    "templates/vanilla/popup.html" \
    "templates/vanilla/popup.js" \
    "templates/vanilla/build.js"

# Check CLI dependencies
if [ ! -x "cli/install.sh" ]; then
    testing::phase::add_error "❌ CLI install script not executable"
else
    log::success "✅ CLI install script is executable"
fi

# Verify API binary can be built
echo "🔍 Testing API binary compilation..."
(
    cd api || testing::phase::add_error "❌ Failed to cd into api directory"
    if ! go build -o scenario-to-extension-api-test . &>/dev/null; then
        testing::phase::add_error "❌ Failed to compile API binary"
    else
        log::success "✅ API binary compiles successfully"
        rm -f scenario-to-extension-api-test
    fi
)

testing::phase::end_with_summary "Dependency tests completed"
