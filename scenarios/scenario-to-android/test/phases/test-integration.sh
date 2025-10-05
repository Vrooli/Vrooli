#!/bin/bash
# Integration test phase for scenario-to-android
# Tests end-to-end conversion workflows

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 5-minute target (build can take time)
testing::phase::init --target-time "300s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# Test output directory
TEST_OUTPUT_DIR="/tmp/scenario-to-android-integration-test-$$"
mkdir -p "$TEST_OUTPUT_DIR"

# Cleanup function
cleanup_integration_tests() {
    log::info "Cleaning up integration test artifacts"
    rm -rf "$TEST_OUTPUT_DIR"
}
trap cleanup_integration_tests EXIT

# ============================================================================
# CLI Installation Test
# ============================================================================

log::info "Testing CLI installation"

if [ -f "cli/install.sh" ]; then
    # Check that install script is executable
    if [ -x "cli/install.sh" ]; then
        log::success "Install script is executable"
    else
        testing::phase::add_error "Install script is not executable"
    fi

    # Verify CLI binary exists
    if [ -x "cli/scenario-to-android" ]; then
        log::success "CLI binary is present and executable"
    else
        testing::phase::add_error "CLI binary not found or not executable"
    fi
else
    testing::phase::add_error "Install script not found"
fi

# ============================================================================
# CLI Command Integration Tests
# ============================================================================

log::info "Testing CLI command integration"

CLI_PATH="$TESTING_PHASE_SCENARIO_DIR/cli/scenario-to-android"

if [ -x "$CLI_PATH" ]; then
    # Test help command
    if "$CLI_PATH" help &> /dev/null; then
        log::success "CLI help command works"
    else
        testing::phase::add_error "CLI help command failed"
    fi

    # Test version command
    if "$CLI_PATH" version &> /dev/null; then
        log::success "CLI version command works"
    else
        testing::phase::add_error "CLI version command failed"
    fi

    # Test status command
    if "$CLI_PATH" status &> /dev/null; then
        log::success "CLI status command works"
    else
        testing::phase::add_error "CLI status command failed"
    fi

    # Test templates command
    if "$CLI_PATH" templates &> /dev/null; then
        log::success "CLI templates command works"
    else
        testing::phase::add_error "CLI templates command failed"
    fi
else
    testing::phase::add_error "CLI binary not available for integration testing"
fi

# ============================================================================
# Template Integrity Test
# ============================================================================

log::info "Testing template integrity"

TEMPLATES_DIR="$TESTING_PHASE_SCENARIO_DIR/initialization/templates/android"

if [ -d "$TEMPLATES_DIR" ]; then
    # Test that templates have required files
    REQUIRED_FILES=(
        "app/build.gradle"
        "build.gradle"
        "settings.gradle"
        "app/src/main/AndroidManifest.xml"
    )

    MISSING_FILES=()
    for file in "${REQUIRED_FILES[@]}"; do
        if [ ! -f "$TEMPLATES_DIR/$file" ]; then
            MISSING_FILES+=("$file")
        fi
    done

    if [ ${#MISSING_FILES[@]} -eq 0 ]; then
        log::success "All required template files present"
    else
        testing::phase::add_error "Missing template files: ${MISSING_FILES[*]}"
    fi

    # Validate Gradle files are valid (basic syntax check)
    if find "$TEMPLATES_DIR" -name "*.gradle" -type f | xargs grep -l "android" &> /dev/null; then
        log::success "Gradle templates contain Android configuration"
    else
        testing::phase::add_warning "Gradle templates may be missing Android configuration"
    fi

    # Validate AndroidManifest.xml
    MANIFEST="$TEMPLATES_DIR/app/src/main/AndroidManifest.xml"
    if [ -f "$MANIFEST" ]; then
        if grep -q "<manifest" "$MANIFEST" && grep -q "<application" "$MANIFEST"; then
            log::success "AndroidManifest.xml has required XML structure"
        else
            testing::phase::add_error "AndroidManifest.xml missing required tags"
        fi
    fi
else
    testing::phase::add_error "Templates directory not found"
fi

# ============================================================================
# Test Scenario Creation
# ============================================================================

log::info "Testing scenario project creation"

# Create a minimal test scenario
TEST_SCENARIO_DIR="$TEST_OUTPUT_DIR/test-scenario"
mkdir -p "$TEST_SCENARIO_DIR/ui"
cat > "$TEST_SCENARIO_DIR/ui/index.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Test Scenario</title>
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
    <h1>Test Android App</h1>
    <p>This is a test scenario for conversion.</p>
</body>
</html>
EOF

cat > "$TEST_SCENARIO_DIR/ui/app.js" << 'EOF'
console.log('Test app initialized');
EOF

if [ -f "$TEST_SCENARIO_DIR/ui/index.html" ]; then
    log::success "Test scenario created successfully"
else
    testing::phase::add_error "Failed to create test scenario"
fi

# ============================================================================
# Conversion Script Integration Test
# ============================================================================

log::info "Testing conversion script with minimal scenario"

CONVERT_SCRIPT="$TESTING_PHASE_SCENARIO_DIR/cli/convert.sh"

if [ -x "$CONVERT_SCRIPT" ]; then
    # Test conversion (will likely fail without Android SDK, but we test the process)
    log::info "Running conversion script (may skip build without Android SDK)"

    # Set up scenario in expected location
    SCENARIOS_DIR="$TEST_OUTPUT_DIR/scenarios"
    mkdir -p "$SCENARIOS_DIR"
    cp -r "$TEST_SCENARIO_DIR" "$SCENARIOS_DIR/test-scenario"

    # Run conversion with custom output
    ANDROID_OUTPUT="$TEST_OUTPUT_DIR/android-build"

    # Capture output and status
    if "$CONVERT_SCRIPT" \
        --scenario test-scenario \
        --output "$ANDROID_OUTPUT" \
        --app-name "Test App" \
        --version "1.0.0" 2>&1 | tee "$TEST_OUTPUT_DIR/convert.log"; then

        log::success "Conversion script completed successfully"

        # Check if project directory was created
        if [ -d "$ANDROID_OUTPUT/test-scenario-android" ]; then
            log::success "Android project directory created"

            # Verify project structure
            PROJECT_DIR="$ANDROID_OUTPUT/test-scenario-android"

            if [ -f "$PROJECT_DIR/build.gradle" ]; then
                log::success "Root build.gradle created"
            else
                testing::phase::add_warning "Root build.gradle not found"
            fi

            if [ -f "$PROJECT_DIR/app/build.gradle" ]; then
                log::success "App build.gradle created"
            else
                testing::phase::add_warning "App build.gradle not found"
            fi

            if [ -d "$PROJECT_DIR/app/src/main/assets" ]; then
                log::success "Assets directory created"

                # Check if UI files were copied
                if [ -f "$PROJECT_DIR/app/src/main/assets/index.html" ]; then
                    log::success "UI files copied to assets"
                else
                    testing::phase::add_warning "UI files not found in assets"
                fi
            else
                testing::phase::add_warning "Assets directory not found"
            fi
        else
            testing::phase::add_warning "Android project directory not created (may be due to missing Android SDK)"
        fi
    else
        # Conversion may fail without Android SDK, which is acceptable
        if grep -q "Gradle not found\|Android SDK" "$TEST_OUTPUT_DIR/convert.log"; then
            log::warning "Conversion skipped build (Android SDK or Gradle not available)"
        else
            testing::phase::add_error "Conversion script failed unexpectedly"
        fi
    fi
else
    testing::phase::add_error "Conversion script not found or not executable"
fi

# ============================================================================
# Prepare-Store Integration Test
# ============================================================================

log::info "Testing Play Store preparation"

# Create a dummy APK
DUMMY_APK="$TEST_OUTPUT_DIR/test.apk"
touch "$DUMMY_APK"

if "$CLI_PATH" prepare-store "$DUMMY_APK" &> "$TEST_OUTPUT_DIR/prepare-store.log"; then
    log::success "Prepare-store command executed"

    # Check if assets directory was created
    ASSETS_DIR="$TEST_OUTPUT_DIR/play-store-assets"
    if [ -d "$ASSETS_DIR" ]; then
        log::success "Play Store assets directory created"

        # Check for generated files
        if [ -f "$ASSETS_DIR/listing.txt" ]; then
            log::success "Store listing template generated"
        else
            testing::phase::add_warning "Store listing not found"
        fi

        if [ -f "$ASSETS_DIR/README.md" ]; then
            log::success "Store submission checklist generated"
        else
            testing::phase::add_warning "Submission checklist not found"
        fi
    else
        testing::phase::add_warning "Play Store assets directory not created"
    fi
else
    testing::phase::add_error "Prepare-store command failed"
fi

# ============================================================================
# Template Variable Substitution Test
# ============================================================================

log::info "Testing template variable substitution"

if [ -f "$ANDROID_OUTPUT/test-scenario-android/app/build.gradle" ]; then
    APP_BUILD_GRADLE="$ANDROID_OUTPUT/test-scenario-android/app/build.gradle"

    # Check if variables were substituted
    if ! grep -q "{{SCENARIO_NAME}}" "$APP_BUILD_GRADLE"; then
        log::success "SCENARIO_NAME variable was substituted"
    else
        testing::phase::add_error "SCENARIO_NAME variable not substituted"
    fi

    # Check for expected substitutions
    if grep -q "test.scenario\|test_scenario" "$APP_BUILD_GRADLE"; then
        log::success "Package name substitution appears correct"
    else
        testing::phase::add_warning "Package name substitution may be incorrect"
    fi
else
    testing::phase::add_warning "Cannot verify template substitution (build.gradle not found)"
fi

# ============================================================================
# End-to-End Workflow Test
# ============================================================================

log::info "Testing complete workflow"

# This tests the full workflow: scenario → template → project
if [ -d "$ANDROID_OUTPUT/test-scenario-android" ]; then
    PROJECT_DIR="$ANDROID_OUTPUT/test-scenario-android"

    # Verify all expected components exist
    COMPONENTS=(
        "build.gradle:Root Gradle build file"
        "settings.gradle:Gradle settings file"
        "app/build.gradle:App module build file"
        "app/src/main/AndroidManifest.xml:Android manifest"
        "app/src/main/assets/index.html:UI assets"
    )

    WORKFLOW_COMPLETE=true
    for component in "${COMPONENTS[@]}"; do
        FILE="${component%%:*}"
        DESC="${component##*:}"

        if [ -e "$PROJECT_DIR/$FILE" ]; then
            log::debug "✓ $DESC"
        else
            log::warning "✗ $DESC not found"
            WORKFLOW_COMPLETE=false
        fi
    done

    if $WORKFLOW_COMPLETE; then
        log::success "End-to-end workflow completed successfully"
    else
        testing::phase::add_warning "End-to-end workflow incomplete (some components missing)"
    fi
else
    testing::phase::add_warning "Cannot verify end-to-end workflow (project not created)"
fi

# End with summary
testing::phase::end_with_summary "Integration tests completed"
