#!/bin/bash
# Structure test phase for scenario-to-android
# Validates scenario directory structure and file organization

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "30s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# ============================================================================
# Required Directories
# ============================================================================

log::info "Validating directory structure"

REQUIRED_DIRS=(
    ".vrooli"
    "cli"
    "initialization"
    "initialization/templates"
    "initialization/templates/android"
    "initialization/prompts"
    "test"
    "test/phases"
    "test/cli"
)

MISSING_DIRS=()
for dir in "${REQUIRED_DIRS[@]}"; do
    if [ ! -d "$dir" ]; then
        MISSING_DIRS+=("$dir")
    fi
done

if [ ${#MISSING_DIRS[@]} -eq 0 ]; then
    log::success "All required directories present"
else
    testing::phase::add_error "Missing directories: ${MISSING_DIRS[*]}"
fi

# ============================================================================
# Required Files
# ============================================================================

log::info "Validating required files"

REQUIRED_FILES=(
    "PRD.md"
    "README.md"
    "Makefile"
    ".vrooli/service.json"
    "cli/scenario-to-android"
    "cli/convert.sh"
    "cli/install.sh"
)

MISSING_FILES=()
for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$file" ]; then
        MISSING_FILES+=("$file")
    fi
done

if [ ${#MISSING_FILES[@]} -eq 0 ]; then
    log::success "All required files present"
else
    testing::phase::add_error "Missing files: ${MISSING_FILES[*]}"
fi

# ============================================================================
# Android Template Structure
# ============================================================================

log::info "Validating Android template structure"

TEMPLATE_FILES=(
    "initialization/templates/android/app/build.gradle"
    "initialization/templates/android/build.gradle"
    "initialization/templates/android/settings.gradle"
    "initialization/templates/android/app/src/main/AndroidManifest.xml"
)

MISSING_TEMPLATES=()
for template in "${TEMPLATE_FILES[@]}"; do
    if [ ! -f "$template" ]; then
        MISSING_TEMPLATES+=("$template")
    fi
done

if [ ${#MISSING_TEMPLATES[@]} -eq 0 ]; then
    log::success "Android template structure is complete"
else
    testing::phase::add_error "Missing template files: ${MISSING_TEMPLATES[*]}"
fi

# ============================================================================
# Executable Permissions
# ============================================================================

log::info "Validating executable permissions"

EXECUTABLES=(
    "cli/scenario-to-android"
    "cli/convert.sh"
    "cli/install.sh"
    "test/phases/test-unit.sh"
    "test/phases/test-integration.sh"
    "test/phases/test-performance.sh"
    "test/phases/test-structure.sh"
)

PERMISSION_ERRORS=()
for exe in "${EXECUTABLES[@]}"; do
    if [ -f "$exe" ]; then
        if [ ! -x "$exe" ]; then
            PERMISSION_ERRORS+=("$exe")
        fi
    fi
done

if [ ${#PERMISSION_ERRORS[@]} -eq 0 ]; then
    log::success "All executables have correct permissions"
else
    testing::phase::add_error "Missing execute permissions: ${PERMISSION_ERRORS[*]}"
fi

# ============================================================================
# File Naming Conventions
# ============================================================================

log::info "Validating file naming conventions"

# Check for consistent naming
NAMING_ISSUES=()

# CLI scripts should use kebab-case
if find cli -name "*.sh" -type f | grep -E "[A-Z_]" &> /dev/null; then
    NAMING_ISSUES+=("CLI scripts should use kebab-case")
fi

# Test files should follow test-*.sh pattern
if find test/phases -name "*.sh" -type f | grep -v "^test-" &> /dev/null; then
    # This is expected, so we don't fail, just note
    log::debug "Test phase files use test-*.sh pattern"
fi

if [ ${#NAMING_ISSUES[@]} -eq 0 ]; then
    log::success "File naming conventions followed"
else
    for issue in "${NAMING_ISSUES[@]}"; do
        testing::phase::add_warning "$issue"
    done
fi

# ============================================================================
# JSON Configuration Validation
# ============================================================================

log::info "Validating JSON configuration files"

if command -v jq &> /dev/null; then
    JSON_FILES=(
        ".vrooli/service.json"
    )

    JSON_ERRORS=()
    for json_file in "${JSON_FILES[@]}"; do
        if [ -f "$json_file" ]; then
            if ! jq empty "$json_file" &> /dev/null; then
                JSON_ERRORS+=("$json_file")
            fi
        fi
    done

    if [ ${#JSON_ERRORS[@]} -eq 0 ]; then
        log::success "All JSON files are valid"
    else
        testing::phase::add_error "Invalid JSON files: ${JSON_ERRORS[*]}"
    fi
else
    testing::phase::add_warning "jq not installed, cannot validate JSON"
fi

# ============================================================================
# Documentation Structure
# ============================================================================

log::info "Validating documentation structure"

# Check PRD.md structure
if [ -f "PRD.md" ]; then
    REQUIRED_SECTIONS=(
        "Core Capability"
        "Success Metrics"
        "Technical Architecture"
        "CLI Interface Contract"
    )

    MISSING_SECTIONS=()
    for section in "${REQUIRED_SECTIONS[@]}"; do
        if ! grep -q "$section" PRD.md; then
            MISSING_SECTIONS+=("$section")
        fi
    done

    if [ ${#MISSING_SECTIONS[@]} -eq 0 ]; then
        log::success "PRD.md has required sections"
    else
        testing::phase::add_warning "PRD.md missing sections: ${MISSING_SECTIONS[*]}"
    fi
else
    testing::phase::add_error "PRD.md not found"
fi

# Check README.md
if [ -f "README.md" ]; then
    if grep -q "scenario-to-android\|Android" README.md; then
        log::success "README.md exists and has content"
    else
        testing::phase::add_warning "README.md may be missing scenario-specific content"
    fi
else
    testing::phase::add_error "README.md not found"
fi

# ============================================================================
# Template File Organization
# ============================================================================

log::info "Validating template organization"

TEMPLATES_DIR="initialization/templates/android"

if [ -d "$TEMPLATES_DIR" ]; then
    # Check for Android project structure
    ANDROID_DIRS=(
        "$TEMPLATES_DIR/app"
        "$TEMPLATES_DIR/app/src"
        "$TEMPLATES_DIR/app/src/main"
        "$TEMPLATES_DIR/app/src/main/java"
        "$TEMPLATES_DIR/app/src/main/res"
    )

    MISSING_ANDROID_DIRS=()
    for android_dir in "${ANDROID_DIRS[@]}"; do
        if [ ! -d "$android_dir" ]; then
            MISSING_ANDROID_DIRS+=("$android_dir")
        fi
    done

    if [ ${#MISSING_ANDROID_DIRS[@]} -eq 0 ]; then
        log::success "Android template directory structure is correct"
    else
        testing::phase::add_warning "Missing Android template directories: ${MISSING_ANDROID_DIRS[*]}"
    fi
else
    testing::phase::add_error "Templates directory not found"
fi

# ============================================================================
# Test Suite Organization
# ============================================================================

log::info "Validating test suite organization"

# Check for test phases
TEST_PHASES=(
    "test/phases/test-unit.sh"
    "test/phases/test-integration.sh"
    "test/phases/test-performance.sh"
    "test/phases/test-structure.sh"
)

MISSING_PHASES=()
for phase in "${TEST_PHASES[@]}"; do
    if [ ! -f "$phase" ]; then
        MISSING_PHASES+=("$phase")
    fi
done

if [ ${#MISSING_PHASES[@]} -eq 0 ]; then
    log::success "All test phases present"
else
    testing::phase::add_warning "Missing test phases: ${MISSING_PHASES[*]}"
fi

# Check for BATS tests
if [ -d "test/cli" ]; then
    bats_count=$(find test/cli -name "*.bats" -type f | wc -l)
    if [ "$bats_count" -gt 0 ]; then
        log::success "BATS CLI tests present ($bats_count files)"
    else
        testing::phase::add_warning "No BATS test files found in test/cli"
    fi
else
    testing::phase::add_warning "test/cli directory not found"
fi

# ============================================================================
# Makefile Targets
# ============================================================================

log::info "Validating Makefile targets"

if [ -f "Makefile" ]; then
    REQUIRED_TARGETS=(
        "help"
        "run"
        "stop"
        "test"
        "clean"
    )

    MISSING_TARGETS=()
    for target in "${REQUIRED_TARGETS[@]}"; do
        if ! grep -q "^$target:" Makefile; then
            MISSING_TARGETS+=("$target")
        fi
    done

    if [ ${#MISSING_TARGETS[@]} -eq 0 ]; then
        log::success "Makefile has all required targets"
    else
        testing::phase::add_warning "Makefile missing targets: ${MISSING_TARGETS[*]}"
    fi
else
    testing::phase::add_error "Makefile not found"
fi

# ============================================================================
# File Size Validation
# ============================================================================

log::info "Validating file sizes"

# Check that files aren't unreasonably large
if [ -f "PRD.md" ]; then
    prd_size=$(wc -c < PRD.md)
    if [ "$prd_size" -gt 100000 ]; then
        testing::phase::add_warning "PRD.md is very large (> 100KB)"
    else
        log::success "PRD.md size is reasonable"
    fi
fi

# End with summary
testing::phase::end_with_summary "Structure tests completed"
