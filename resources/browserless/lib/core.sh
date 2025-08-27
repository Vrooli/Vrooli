#!/usr/bin/env bash
# Browserless Core Functions - Consolidated browserless-specific logic
# All generic operations delegated to shared libraries

# Source shared libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSERLESS_LIB_DIR="${APP_ROOT}/resources/browserless/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/status-engine.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/health-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/backup-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/init-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/wait-utils.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"

#######################################
# Browserless Configuration Constants
# These are set in config/defaults.sh as readonly
# Only set non-readonly variables here
#######################################
: "${MAX_BROWSERS:=5}"
: "${HEADLESS:=yes}"
: "${TIMEOUT:=30000}"

#######################################
# Get Browserless initialization configuration
# Returns: JSON configuration for init framework
#######################################
browserless::get_init_config() {
    local max_browsers="${1:-$MAX_BROWSERS}"
    local timeout="${2:-$TIMEOUT}"
    local headless="${3:-$HEADLESS}"
    
    # Build volumes list
    local volumes_array="[\"${BROWSERLESS_DATA_DIR}:/workspace\"]"
    
    # Build environment variables
    local config='{
        "resource_name": "browserless",
        "container_name": "'$BROWSERLESS_CONTAINER_NAME'",
        "data_dir": "'$BROWSERLESS_DATA_DIR'",
        "host_port": '$BROWSERLESS_PORT',
        "container_port": 3000,
        "image": "'$BROWSERLESS_IMAGE'",
        "env_vars": {
            "CONCURRENT": "'$max_browsers'",
            "TIMEOUT": "'$timeout'",
            "ENABLE_DEBUGGER": "false",
            "PREBOOT_CHROME": "true",
            "KEEP_ALIVE": "true",
            "PRE_REQUEST_HEALTH_CHECK": "false",
            "FUNCTION_ENABLE_INCOGNITO_MODE": "true",
            "WORKSPACE_DELETE_EXPIRED": "true",
            "WORKSPACE_EXPIRE_DAYS": "7"
        },
        "volumes": '$volumes_array',
        "networks": ["'$BROWSERLESS_NETWORK_NAME'"],
        "shm_size": "'$BROWSERLESS_DOCKER_SHM_SIZE'",
        "cap_add": ["'$BROWSERLESS_DOCKER_CAPS'"],
        "security_opt": ["seccomp='$BROWSERLESS_DOCKER_SECCOMP'"],
        "first_run_check": "browserless::is_first_run",
        "setup_func": "browserless::first_time_setup",
        "wait_for_ready": "browserless::wait_for_ready"
    }'
    
    echo "$config"
}

#######################################
# Install Browserless using init framework
#######################################
browserless::install() {
    local init_config
    init_config=$(browserless::get_init_config)
    
    if ! init::setup_resource "$init_config"; then
        log::error "Failed to install Browserless"
        return 1
    fi
    
    # Perform any additional Browserless-specific setup
    browserless::post_install_setup
    
    return 0
}

#######################################
# Uninstall Browserless
#######################################
browserless::uninstall() {
    log::info "Uninstalling Browserless..."
    
    # Stop container
    docker::stop_container "$BROWSERLESS_CONTAINER_NAME"
    
    # Remove container
    docker::remove_container "$BROWSERLESS_CONTAINER_NAME"
    
    # Optional: Remove data directory
    if [[ "$FORCE" == "yes" ]]; then
        log::warn "Force flag set - removing data directory"
        rm -rf "$BROWSERLESS_DATA_DIR"
    fi
    
    # Remove network
    docker::remove_network "$BROWSERLESS_NETWORK_NAME"
    
    log::success "Browserless uninstalled successfully"
    return 0
}

#######################################
# Start Browserless container
#######################################
browserless::start() {
    if docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
        log::info "Browserless is already running"
        return 0
    fi
    
    if ! docker::container_exists "$BROWSERLESS_CONTAINER_NAME"; then
        log::error "Browserless container does not exist. Run install first."
        return 1
    fi
    
    docker::start_container "$BROWSERLESS_CONTAINER_NAME"
    browserless::wait_for_ready
}

#######################################
# Stop Browserless container
#######################################
browserless::stop() {
    docker::stop_container "$BROWSERLESS_CONTAINER_NAME"
}

#######################################
# Restart Browserless container
#######################################
browserless::restart() {
    browserless::stop
    sleep 2
    browserless::start
}

#######################################
# Show Browserless logs
#######################################
browserless::logs() {
    local lines="${LINES:-50}"
    
    if ! docker::container_exists "$BROWSERLESS_CONTAINER_NAME"; then
        log::error "Browserless container does not exist"
        return 1
    fi
    
    log::info "Showing Browserless logs (last $lines lines)..."
    docker logs --tail "$lines" "$BROWSERLESS_CONTAINER_NAME"
}

#######################################
# Get Browserless info
#######################################
browserless::info() {
    log::info "Browserless Information"
    echo "========================"
    echo "Container: $BROWSERLESS_CONTAINER_NAME"
    echo "Port: $BROWSERLESS_PORT"
    echo "Data Directory: $BROWSERLESS_DATA_DIR"
    echo "Max Browsers: $MAX_BROWSERS"
    echo "Timeout: ${TIMEOUT}ms"
    echo "Headless: $HEADLESS"
    echo
    
    if docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
        echo "Status: Running"
        echo "URL: http://localhost:$BROWSERLESS_PORT"
        echo
        
        # Show container stats
        docker stats --no-stream "$BROWSERLESS_CONTAINER_NAME" 2>/dev/null || true
    else
        echo "Status: Not running"
    fi
}

#######################################
# Get Browserless version
#######################################
browserless::version() {
    if docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
        local version_info
        version_info=$(http::get "http://localhost:$BROWSERLESS_PORT/config" 2>/dev/null || echo "{}")
        
        if [[ -n "$version_info" ]] && [[ "$version_info" != "{}" ]]; then
            echo "$version_info" | jq -r '.version // "Unknown"' 2>/dev/null || echo "Unknown"
        else
            echo "Unable to retrieve version"
        fi
    else
        echo "Browserless is not running"
        return 1
    fi
}

#######################################
# Test Browserless functionality
#######################################
browserless::test() {
    log::info "Testing Browserless functionality..."
    
    if ! docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
        log::error "Browserless is not running"
        return 1
    fi
    
    local test_results=0
    
    # Test health endpoint
    log::info "Testing health endpoint..."
    if http::check_endpoint "http://localhost:$BROWSERLESS_PORT/pressure"; then
        log::success "Health endpoint: OK"
    else
        log::error "Health endpoint: FAILED"
        ((test_results++))
    fi
    
    # Test metrics endpoint
    log::info "Testing metrics endpoint..."
    if http::check_endpoint "http://localhost:$BROWSERLESS_PORT/metrics"; then
        log::success "Metrics endpoint: OK"
    else
        log::error "Metrics endpoint: FAILED"
        ((test_results++))
    fi
    
    # Test config endpoint
    log::info "Testing config endpoint..."
    if http::check_endpoint "http://localhost:$BROWSERLESS_PORT/config"; then
        log::success "Config endpoint: OK"
    else
        log::error "Config endpoint: FAILED"
        ((test_results++))
    fi
    
    # Test screenshot capability
    log::info "Testing screenshot capability..."
    if browserless::test_screenshot; then
        log::success "Screenshot API: OK"
    else
        log::error "Screenshot API: FAILED"
        ((test_results++))
    fi
    
    if [[ $test_results -eq 0 ]]; then
        log::success "All tests passed!"
        return 0
    else
        log::error "$test_results test(s) failed"
        return 1
    fi
}

#######################################
# Check if this is first run
#######################################
browserless::is_first_run() {
    # Check if data directory exists and has content
    if [[ -d "$BROWSERLESS_DATA_DIR" ]] && [[ "$(ls -A "$BROWSERLESS_DATA_DIR" 2>/dev/null)" ]]; then
        return 1  # Not first run
    fi
    return 0  # First run
}

#######################################
# First time setup for Browserless
#######################################
browserless::first_time_setup() {
    log::info "Performing first-time setup for Browserless..."
    
    # Create workspace directory structure
    mkdir -p "$BROWSERLESS_DATA_DIR"/{downloads,uploads,screenshots}
    
    # Set permissions
    chmod -R 755 "$BROWSERLESS_DATA_DIR"
    
    log::success "First-time setup completed"
    return 0
}

#######################################
# Wait for Browserless to be ready
#######################################
browserless::wait_for_ready() {
    log::info "Waiting for Browserless to be ready..."
    
    # Use the configured host port (4110)
    if wait::for_http "http://localhost:${BROWSERLESS_PORT}/pressure" 60; then
        log::success "Browserless is ready!"
        return 0
    else
        log::error "Browserless failed to become ready"
        return 1
    fi
}

#######################################
# Post-install setup
#######################################
browserless::post_install_setup() {
    log::info "Performing post-install setup..."
    
    # Display connection information
    echo
    log::success "Browserless installed successfully!"
    echo "=================================="
    echo "Access URL: http://localhost:${BROWSERLESS_PORT}"
    echo "Max Browsers: $MAX_BROWSERS"
    echo "Timeout: ${TIMEOUT}ms"
    echo "Data Directory: $BROWSERLESS_DATA_DIR"
    echo
    echo "To test the installation:"
    echo "  $0 --action test"
    echo
    echo "To view usage examples:"
    echo "  $0 --action usage"
    echo
    
    # Auto-install CLI if available
    # shellcheck disable=SC1091
    "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "$BROWSERLESS_SCRIPT_DIR" 2>/dev/null || true
}

#######################################
# Test screenshot functionality
#######################################
browserless::test_screenshot() {
    local test_url="https://example.com"
    local response
    
    response=$(curl -s -X POST \
        "http://localhost:$BROWSERLESS_PORT/chrome/screenshot" \
        -H "Content-Type: application/json" \
        -d "{\"url\":\"$test_url\"}" \
        -w "\n%{http_code}" \
        2>/dev/null | tail -n1)
    
    [[ "$response" == "200" ]]
}

#######################################
# Get URLs for Browserless
#######################################
browserless::get_urls() {
    echo "Browserless URLs:"
    echo "================="
    echo "Main UI: http://localhost:$BROWSERLESS_PORT"
    echo "Metrics: http://localhost:$BROWSERLESS_PORT/metrics"
    echo "Config: http://localhost:$BROWSERLESS_PORT/config"
    echo "Pressure: http://localhost:$BROWSERLESS_PORT/pressure"
    echo
    echo "API Endpoints:"
    echo "Screenshot: http://localhost:$BROWSERLESS_PORT/chrome/screenshot"
    echo "PDF: http://localhost:$BROWSERLESS_PORT/chrome/pdf"
    echo "Content: http://localhost:$BROWSERLESS_PORT/chrome/content"
    echo "Function: http://localhost:$BROWSERLESS_PORT/chrome/function"
}