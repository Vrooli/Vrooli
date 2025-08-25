#!/usr/bin/env bats
# Tests for Browserless messages.sh configuration
bats_require_minimum_version 1.5.0

# Setup paths and source var.sh first
SCRIPT_DIR="$(builtin cd "${BATS_TEST_FILENAME%/*}" && builtin pwd)"
# shellcheck disable=SC1091
source "$(builtin cd "${SCRIPT_DIR%/*/*/*/*/*}" && builtin pwd)/lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "browserless"
    
    # Load the messages once per file
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/messages.sh"
    
    # Export the setup_file directory for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
}

# Lightweight per-test setup
setup() {
    # Use the directory from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    
    # Re-source messages to ensure functions are available in test scope
    source "${SCRIPT_DIR}/messages.sh"
}

# Test message export
@test "browserless::export_messages sets all success messages" {
    browserless::export_messages
    
    # Test success messages
    [ "$MSG_INSTALL_SUCCESS" = "✅ Browserless installed successfully" ]
    [ "$MSG_START_SUCCESS" = "✅ Browserless started successfully" ]
    [ "$MSG_STOP_SUCCESS" = "✅ Browserless stopped successfully" ]
    [ "$MSG_HEALTHY" = "✅ Browserless API is healthy" ]
    [ "$MSG_RUNNING" = "✅ Browserless container is running" ]
}

# Test error messages
@test "browserless::export_messages sets all error messages" {
    browserless::export_messages
    
    # Test error messages
    [ "$MSG_DOCKER_NOT_FOUND" = "Docker is not installed" ]
    [ "$MSG_DOCKER_NOT_RUNNING" = "Docker daemon is not running" ]
    [ "$MSG_INSTALL_FAILED" = "❌ Browserless installation failed" ]
    [ "$MSG_NOT_INSTALLED" = "❌ Browserless is not installed" ]
    [ "$MSG_NOT_HEALTHY" = "Browserless is not running or healthy" ]
}

# Test info messages
@test "browserless::export_messages sets all info messages" {
    browserless::export_messages
    
    # Test info messages
    [[ "$MSG_CREATING_DIRS" == *"Creating Browserless data directory"* ]]
    [[ "$MSG_STARTING_CONTAINER" == *"Starting Browserless container"* ]]
    [[ "$MSG_WAITING_STARTUP" == *"Waiting for Browserless to start"* ]]
}

# Test usage example messages
@test "browserless::export_messages sets usage example messages" {
    browserless::export_messages
    
    # Test usage messages
    [ "$MSG_USAGE_SCREENSHOT" = "📸 Testing Browserless Screenshot API" ]
    [ "$MSG_USAGE_PDF" = "📄 Testing Browserless PDF API" ]
    [ "$MSG_USAGE_SCRAPE" = "🕷️ Testing Browserless Content Scraping" ]
    [ "$MSG_USAGE_PRESSURE" = "📊 Checking Browserless Pool Status" ]
    [ "$MSG_USAGE_ALL" = "🎭 Running All Browserless Usage Examples" ]
}

# Test Docker hint messages
@test "browserless::export_messages sets Docker hint messages" {
    browserless::export_messages
    
    # Test Docker hints
    [[ "$MSG_DOCKER_INSTALL_HINT" == *"https://docs.docker.com/get-docker/"* ]]
    [[ "$MSG_DOCKER_START_HINT" == *"sudo systemctl start docker"* ]]
    [[ "$MSG_DOCKER_PERMISSIONS_HINT" == *"sudo usermod -aG docker"* ]]
}
