#!/bin/bash
# Unit test phase for scenario-to-android
# Uses centralized testing library

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 60-second target
testing::phase::init --target-time "60s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# ============================================================================
# CLI Testing with BATS
# ============================================================================

log::info "Running CLI tests with BATS"

if ! command -v bats &> /dev/null; then
    testing::phase::add_warning "BATS not installed, skipping CLI tests"
    log::warning "Install BATS: npm install -g bats"
else
    # Run BATS tests for CLI
    if [ -d "test/cli" ]; then
        log::info "Testing CLI commands..."

        if bats test/cli/scenario-to-android.bats; then
            log::success "CLI main command tests passed"
        else
            testing::phase::add_error "CLI main command tests failed"
        fi

        if bats test/cli/convert.bats; then
            log::success "CLI convert script tests passed"
        else
            testing::phase::add_error "CLI convert script tests failed"
        fi
    else
        testing::phase::add_warning "No CLI test directory found"
    fi
fi

# ============================================================================
# Shell Script Linting
# ============================================================================

log::info "Linting shell scripts"

if command -v shellcheck &> /dev/null; then
    # Lint CLI scripts
    if find cli -name "*.sh" -type f | xargs shellcheck; then
        log::success "Shell scripts passed linting"
    else
        testing::phase::add_warning "Shell script linting found issues"
    fi
else
    testing::phase::add_warning "shellcheck not installed"
fi

# ============================================================================
# Template Validation
# ============================================================================

log::info "Validating Android templates"

TEMPLATES_DIR="initialization/templates/android"

if [ -d "$TEMPLATES_DIR" ]; then
    # Check for required template files
    REQUIRED_TEMPLATES=(
        "app/build.gradle"
        "build.gradle"
        "settings.gradle"
        "app/src/main/AndroidManifest.xml"
    )

    MISSING_TEMPLATES=()
    for template in "${REQUIRED_TEMPLATES[@]}"; do
        if [ ! -f "$TEMPLATES_DIR/$template" ]; then
            MISSING_TEMPLATES+=("$template")
        fi
    done

    if [ ${#MISSING_TEMPLATES[@]} -eq 0 ]; then
        log::success "All required templates present"
    else
        testing::phase::add_error "Missing templates: ${MISSING_TEMPLATES[*]}"
    fi

    # Validate template variables
    log::info "Checking template variable syntax"
    if grep -r "{{SCENARIO_NAME}}" "$TEMPLATES_DIR" &> /dev/null; then
        log::success "Template variables found"
    else
        testing::phase::add_warning "No template variables found (may be intentional)"
    fi
else
    testing::phase::add_error "Templates directory not found: $TEMPLATES_DIR"
fi

# ============================================================================
# Service Configuration Validation
# ============================================================================

log::info "Validating service.json"

if [ -f ".vrooli/service.json" ]; then
    if command -v jq &> /dev/null; then
        # Validate JSON syntax
        if jq empty .vrooli/service.json &> /dev/null; then
            log::success "service.json is valid JSON"

            # Check for required fields
            if jq -e '.service.name' .vrooli/service.json &> /dev/null; then
                log::success "service.json has required fields"
            else
                testing::phase::add_error "service.json missing required fields"
            fi
        else
            testing::phase::add_error "service.json is invalid JSON"
        fi
    else
        testing::phase::add_warning "jq not installed, cannot validate service.json"
    fi
else
    testing::phase::add_error "service.json not found"
fi

# ============================================================================
# File Permissions Check
# ============================================================================

log::info "Checking executable permissions"

EXECUTABLES=(
    "cli/scenario-to-android"
    "cli/convert.sh"
    "cli/install.sh"
)

PERMISSION_ERRORS=()
for exe in "${EXECUTABLES[@]}"; do
    if [ -f "$exe" ]; then
        if [ ! -x "$exe" ]; then
            PERMISSION_ERRORS+=("$exe")
        fi
    else
        testing::phase::add_warning "Expected executable not found: $exe"
    fi
done

if [ ${#PERMISSION_ERRORS[@]} -eq 0 ]; then
    log::success "All executables have correct permissions"
else
    testing::phase::add_error "Missing execute permissions: ${PERMISSION_ERRORS[*]}"
fi

# ============================================================================
# Documentation Validation
# ============================================================================

log::info "Validating documentation"

REQUIRED_DOCS=(
    "PRD.md"
    "README.md"
)

MISSING_DOCS=()
for doc in "${REQUIRED_DOCS[@]}"; do
    if [ ! -f "$doc" ]; then
        MISSING_DOCS+=("$doc")
    fi
done

if [ ${#MISSING_DOCS[@]} -eq 0 ]; then
    log::success "Required documentation present"
else
    testing::phase::add_error "Missing documentation: ${MISSING_DOCS[*]}"
fi

# End with summary
testing::phase::end_with_summary "Unit tests completed"
