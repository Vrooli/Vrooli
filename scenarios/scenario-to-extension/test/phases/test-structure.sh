#!/bin/bash
# Structure tests for scenario-to-extension
source "$(dirname "${BASH_SOURCE[0]}")/../../../../scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

# Check required files
testing::phase::check_files \
    ".vrooli/service.json" \
    "PRD.md" \
    "README.md" \
    "api/main.go" \
    "api/go.mod" \
    "api/test_helpers.go" \
    "api/test_patterns.go" \
    "api/main_test.go" \
    "cli/install.sh" \
    "templates/vanilla/manifest.json" \
    "templates/vanilla/background.js" \
    "templates/vanilla/content.js" \
    "templates/vanilla/popup.html" \
    "templates/vanilla/popup.js" \
    "test/phases/test-api.sh"

# Check required directories
testing::phase::check_directories \
    "api" \
    "cli" \
    "templates" \
    "templates/vanilla" \
    "test" \
    "test/phases" \
    "ui"

# Validate service.json structure
echo "🔍 Validating service.json structure..."
if ! jq -e '.service.name == "scenario-to-extension"' .vrooli/service.json &>/dev/null; then
    testing::phase::add_error "❌ Invalid service.json: missing or incorrect service.name"
else
    log::success "✅ service.json has correct service name"
fi

if ! jq -e '.api.paths."/api/v1/extension/generate"' .vrooli/service.json &>/dev/null; then
    testing::phase::add_error "❌ Invalid service.json: missing API path definition"
else
    log::success "✅ service.json has API path definitions"
fi

if ! jq -e '.cli.commands.generate' .vrooli/service.json &>/dev/null; then
    testing::phase::add_error "❌ Invalid service.json: missing CLI command definition"
else
    log::success "✅ service.json has CLI command definitions"
fi

# Validate Go module structure
echo "🔍 Validating Go module structure..."
if ! grep -q "module github.com/vrooli/vrooli/scenarios/scenario-to-extension" api/go.mod; then
    testing::phase::add_error "❌ Invalid go.mod: incorrect module path"
else
    log::success "✅ Go module structure is valid"
fi

# Validate template structure
echo "🔍 Validating template structure..."
if ! grep -q '"manifest_version"' templates/vanilla/manifest.json; then
    testing::phase::add_error "❌ Invalid manifest.json: missing manifest_version"
else
    log::success "✅ Template manifest has manifest_version"
fi

if ! grep -q "{{APP_NAME}}" templates/vanilla/manifest.json; then
    testing::phase::add_error "❌ manifest.json missing template variables"
else
    log::success "✅ Template variables present in manifest.json"
fi

# Check test file organization
echo "🔍 Validating test file organization..."
TEST_FILES=(
    "api/main_test.go"
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/comprehensive_test.go"
    "api/performance_test.go"
    "api/business_test.go"
)

for test_file in "${TEST_FILES[@]}"; do
    if [ ! -f "$test_file" ]; then
        testing::phase::add_warning "⚠️  Recommended test file missing: $test_file"
    fi
done

# Validate CLI structure
if [ ! -x "cli/install.sh" ]; then
    testing::phase::add_error "❌ CLI install script is not executable"
else
    log::success "✅ CLI install script is executable"
fi

# Check for documentation
DOCS=("PRD.md" "README.md")
for doc in "${DOCS[@]}"; do
    if [ ! -f "$doc" ]; then
        testing::phase::add_warning "⚠️  Missing documentation: $doc"
    elif [ ! -s "$doc" ]; then
        testing::phase::add_warning "⚠️  Empty documentation file: $doc"
    else
        log::success "✅ Documentation present: $doc"
    fi
done

testing::phase::end_with_summary "Structure tests completed"
