#!/bin/bash
# Structure tests for scenario-to-extension

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::info "Starting structure tests for scenario-to-extension"

# Required files
REQUIRED_FILES=(
    ".vrooli/service.json"
    "PRD.md"
    "README.md"
    "api/main.go"
    "api/go.mod"
    "api/test_helpers.go"
    "api/test_patterns.go"
    "api/main_test.go"
    "cli/install.sh"
    "templates/vanilla/manifest.json"
    "templates/vanilla/background.js"
    "templates/vanilla/content.js"
    "templates/vanilla/popup.html"
    "templates/vanilla/popup.js"
    "test/phases/test-unit.sh"
)

# Required directories
REQUIRED_DIRS=(
    "api"
    "cli"
    "templates"
    "templates/vanilla"
    "test"
    "test/phases"
    "ui"
)

# Check required files
testing::phase::step "Checking required files"
MISSING_FILES=()
for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        MISSING_FILES+=("$file")
    fi
done

if [ ${#MISSING_FILES[@]} -gt 0 ]; then
    testing::phase::error "Missing required files:"
    for file in "${MISSING_FILES[@]}"; do
        testing::phase::error "  - $file"
    done
    testing::phase::end_with_summary "Structure tests failed"
    exit 1
fi
testing::phase::success "All required files present"

# Check required directories
testing::phase::step "Checking required directories"
MISSING_DIRS=()
for dir in "${REQUIRED_DIRS[@]}"; do
    if [ ! -d "$dir" ]; then
        MISSING_DIRS+=("$dir")
    fi
done

if [ ${#MISSING_DIRS[@]} -gt 0 ]; then
    testing::phase::error "Missing required directories:"
    for dir in "${MISSING_DIRS[@]}"; do
        testing::phase::error "  - $dir"
    done
    testing::phase::end_with_summary "Structure tests failed"
    exit 1
fi
testing::phase::success "All required directories present"

# Validate service.json structure
testing::phase::step "Validating service.json structure"
if ! jq -e '.service.name == "scenario-to-extension"' .vrooli/service.json &>/dev/null; then
    testing::phase::error "Invalid service.json: missing or incorrect service.name"
    testing::phase::end_with_summary "Structure tests failed"
    exit 1
fi

if ! jq -e '.api.paths."/api/v1/extension/generate"' .vrooli/service.json &>/dev/null; then
    testing::phase::error "Invalid service.json: missing API path definition"
    testing::phase::end_with_summary "Structure tests failed"
    exit 1
fi

if ! jq -e '.cli.commands.generate' .vrooli/service.json &>/dev/null; then
    testing::phase::error "Invalid service.json: missing CLI command definition"
    testing::phase::end_with_summary "Structure tests failed"
    exit 1
fi
testing::phase::success "service.json structure is valid"

# Validate Go module structure
testing::phase::step "Validating Go module structure"
if ! grep -q "module github.com/vrooli/vrooli/scenarios/scenario-to-extension" api/go.mod; then
    testing::phase::error "Invalid go.mod: incorrect module path"
    testing::phase::end_with_summary "Structure tests failed"
    exit 1
fi
testing::phase::success "Go module structure is valid"

# Validate template structure
testing::phase::step "Validating template structure"
if ! jq -e '.manifest_version' templates/vanilla/manifest.json &>/dev/null; then
    testing::phase::error "Invalid manifest.json: missing manifest_version"
    testing::phase::end_with_summary "Structure tests failed"
    exit 1
fi

if ! grep -q "{{APP_NAME}}" templates/vanilla/manifest.json; then
    testing::phase::error "manifest.json missing template variables"
    testing::phase::end_with_summary "Structure tests failed"
    exit 1
fi
testing::phase::success "Template structure is valid"

# Check test file organization
testing::phase::step "Validating test file organization"
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
        testing::phase::warn "Recommended test file missing: $test_file"
    fi
done
testing::phase::success "Test file organization checked"

# Validate CLI structure
testing::phase::step "Validating CLI structure"
if [ ! -x "cli/install.sh" ]; then
    testing::phase::error "CLI install script is not executable"
    testing::phase::end_with_summary "Structure tests failed"
    exit 1
fi
testing::phase::success "CLI structure is valid"

# Check for documentation
testing::phase::step "Checking documentation"
DOCS=(
    "PRD.md"
    "README.md"
)

for doc in "${DOCS[@]}"; do
    if [ ! -f "$doc" ]; then
        testing::phase::warn "Missing documentation: $doc"
    elif [ ! -s "$doc" ]; then
        testing::phase::warn "Empty documentation file: $doc"
    fi
done
testing::phase::success "Documentation checked"

testing::phase::info "All structure tests passed"
testing::phase::end_with_summary "Structure tests completed successfully"
