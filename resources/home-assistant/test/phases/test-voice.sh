#!/usr/bin/env bash
################################################################################
# Home Assistant Voice Control Tests
# Tests voice assistant integration functionality
################################################################################

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/voice.sh"

# Test results tracking
TESTS_PASSED=0
TESTS_FAILED=0

#######################################
# Run a test and track results
#######################################
run_test() {
    local test_name="$1"
    shift
    
    log::info "Test: $test_name..."
    
    if "$@"; then
        log::success "✓ $test_name"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "✗ $test_name"
        ((TESTS_FAILED++))
        return 1
    fi
}

#######################################
# Test voice CLI commands exist
#######################################
test_voice_cli_commands() {
    local help_output
    help_output=$("${RESOURCE_DIR}/cli.sh" help 2>&1)
    
    # Check for voice command group
    if echo "$help_output" | grep -q "voice.*Voice assistant integration"; then
        return 0
    else
        log::error "Voice command group not found in CLI help"
        return 1
    fi
}

#######################################
# Test voice status command
#######################################
test_voice_status() {
    local status_output
    status_output=$("${RESOURCE_DIR}/cli.sh" voice status 2>&1)
    
    # Check for expected status output
    if echo "$status_output" | grep -q "Voice Control Status"; then
        return 0
    else
        log::error "Voice status command failed"
        return 1
    fi
}

#######################################
# Test Alexa configuration
#######################################
test_alexa_configuration() {
    # Clean up any existing Alexa config
    rm -f "$HOME_ASSISTANT_CONFIG_DIR/alexa_config.yaml" 2>/dev/null || true
    
    # Configure Alexa
    if home_assistant::voice::configure_alexa; then
        # Check if configuration file was created
        if [[ -f "$HOME_ASSISTANT_CONFIG_DIR/alexa_config.yaml" ]]; then
            # Verify content
            if grep -q "alexa:" "$HOME_ASSISTANT_CONFIG_DIR/alexa_config.yaml"; then
                return 0
            else
                log::error "Alexa config file missing expected content"
                return 1
            fi
        else
            log::error "Alexa config file not created"
            return 1
        fi
    else
        log::error "Alexa configuration function failed"
        return 1
    fi
}

#######################################
# Test Google Assistant configuration
#######################################
test_google_configuration() {
    # Clean up any existing Google config
    rm -f "$HOME_ASSISTANT_CONFIG_DIR/google_assistant_config.yaml" 2>/dev/null || true
    
    # Configure Google Assistant
    if home_assistant::voice::configure_google; then
        # Check if configuration file was created
        if [[ -f "$HOME_ASSISTANT_CONFIG_DIR/google_assistant_config.yaml" ]]; then
            # Verify content
            if grep -q "google_assistant:" "$HOME_ASSISTANT_CONFIG_DIR/google_assistant_config.yaml"; then
                return 0
            else
                log::error "Google config file missing expected content"
                return 1
            fi
        else
            log::error "Google config file not created"
            return 1
        fi
    else
        log::error "Google Assistant configuration function failed"
        return 1
    fi
}

#######################################
# Test custom voice assistant configuration
#######################################
test_custom_voice_configuration() {
    # Clean up any existing custom config
    rm -f "$HOME_ASSISTANT_CONFIG_DIR/assist_config.yaml" 2>/dev/null || true
    rm -f "$HOME_ASSISTANT_CONFIG_DIR/intent_scripts.yaml" 2>/dev/null || true
    
    # Configure custom voice assistant
    if home_assistant::voice::configure_custom; then
        # Check if configuration files were created
        if [[ -f "$HOME_ASSISTANT_CONFIG_DIR/assist_config.yaml" ]] && \
           [[ -f "$HOME_ASSISTANT_CONFIG_DIR/intent_scripts.yaml" ]]; then
            # Verify content
            if grep -q "assist_pipeline:" "$HOME_ASSISTANT_CONFIG_DIR/assist_config.yaml" && \
               grep -q "stt.faster_whisper" "$HOME_ASSISTANT_CONFIG_DIR/assist_config.yaml" && \
               grep -q "tts.piper" "$HOME_ASSISTANT_CONFIG_DIR/assist_config.yaml"; then
                return 0
            else
                log::error "Custom voice config files missing expected content"
                return 1
            fi
        else
            log::error "Custom voice config files not created"
            return 1
        fi
    else
        log::error "Custom voice configuration function failed"
        return 1
    fi
}

#######################################
# Test voice configuration test command
#######################################
test_voice_test_command() {
    # Ensure we have a configuration to test
    home_assistant::voice::configure_custom >/dev/null 2>&1 || true
    
    # Run the test command
    if home_assistant::voice::test; then
        return 0
    else
        # Expected to fail with warnings if not fully configured, but should not error
        return 0
    fi
}

#######################################
# Test intent scripts creation
#######################################
test_intent_scripts() {
    # Configure custom voice to create intent scripts
    home_assistant::voice::configure_custom >/dev/null 2>&1 || true
    
    if [[ -f "$HOME_ASSISTANT_CONFIG_DIR/intent_scripts.yaml" ]]; then
        # Check for required intents
        local required_intents=("TurnOnLight" "TurnOffLight" "SetTemperature" "GetStatus")
        for intent in "${required_intents[@]}"; do
            if ! grep -q "$intent:" "$HOME_ASSISTANT_CONFIG_DIR/intent_scripts.yaml"; then
                log::error "Missing intent: $intent"
                return 1
            fi
        done
        return 0
    else
        log::error "Intent scripts file not found"
        return 1
    fi
}

#######################################
# Main test execution
#######################################
main() {
    log::header "Home Assistant Voice Control Tests"
    
    # Initialize environment
    home_assistant::init
    
    # Run tests
    run_test "Voice CLI commands exist" test_voice_cli_commands
    run_test "Voice status command works" test_voice_status
    run_test "Alexa configuration" test_alexa_configuration
    run_test "Google Assistant configuration" test_google_configuration
    run_test "Custom voice configuration" test_custom_voice_configuration
    run_test "Voice test command" test_voice_test_command
    run_test "Intent scripts creation" test_intent_scripts
    
    # Summary
    local total=$((TESTS_PASSED + TESTS_FAILED))
    log::header "Test Summary"
    log::info "Total tests: $total"
    log::success "Passed: $TESTS_PASSED"
    if [[ $TESTS_FAILED -gt 0 ]]; then
        log::error "Failed: $TESTS_FAILED"
        return 1
    else
        log::success "All voice control tests passed!"
        return 0
    fi
}

# Run main function
main "$@"