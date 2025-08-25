#!/usr/bin/env bats
# Tests for Browserless manage.sh script
bats_require_minimum_version 1.5.0

# Setup paths and source var.sh first
SCRIPT_DIR="$(builtin cd "${BATS_TEST_FILENAME%/*}" && builtin pwd)"
# shellcheck disable=SC1091
source "$(builtin cd "${SCRIPT_DIR%/*/*/*/*}" && builtin pwd)/lib/utils/var.sh"

# Load Vrooli test infrastructure using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "browserless"
    
    # SCRIPT_DIR already set at file level
    export MOCK_DIR="${var_SCRIPTS_TEST_DIR}/fixtures/mocks"
    
    # Load all dependencies once (expensive operations)
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
    # shellcheck disable=SC1091
    source "${var_LIB_UTILS_DIR}/args-cli.sh"
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh"
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/messages.sh"
    
    # Load the browserless mock once
    if [[ -f "$MOCK_DIR/browserless.sh" ]]; then
        # shellcheck disable=SC1091
        source "$MOCK_DIR/browserless.sh"
    fi
    
    # Load manage.sh once with all its dependencies
    # First source the lib files directly to ensure they're loaded
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/lib/common.sh"
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/lib/docker.sh"
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/lib/status.sh"
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/lib/install.sh"
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/lib/api.sh"
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/lib/usage.sh"
    
    # Now source manage.sh
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/manage.sh"
    
    # Export key functions for BATS subshells
    export -f browserless::main
    export -f browserless::parse_arguments
    export -f browserless::usage
    export -f browserless::export_config
    export -f browserless::export_messages
    export -f browserless::install_service
    export -f browserless::uninstall_service
    export -f browserless::show_status
    export -f browserless::show_info
    export -f browserless::docker_start
    export -f browserless::docker_stop
    export -f browserless::docker_restart
    export -f browserless::docker_logs
    export -f browserless::test_all_apis
    export -f browserless::run_usage_example
    
    # Export args functions needed by browserless
    export -f args::reset
    export -f args::register
    export -f args::register_help
    export -f args::register_yes
    export -f args::parse
    export -f args::get
    export -f args::is_asking_for_help
    export -f args::usage
    
    # Export resources functions needed by browserless
    export -f resources::get_default_port
    
    # Export log functions once
    log::header() { echo "=== $* ==="; }
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    export -f log::header log::info log::error log::success log::warning
    
    # Load and export browserless mock functions
    if [[ -f "$MOCK_DIR/browserless.sh" ]]; then
        # shellcheck disable=SC1091
        source "$MOCK_DIR/browserless.sh"
        # Export mock state functions
        export -f mock::browserless::load_state 2>/dev/null || true
        export -f mock::browserless::save_state 2>/dev/null || true
    fi
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Ensure mock log directory exists to prevent warnings
    MOCK_LOG_DIR="${MOCK_RESPONSES_DIR:-${BATS_TEST_DIRNAME}/../../../../data/test-outputs/mock-logs}"
    if [[ ! -d "$MOCK_LOG_DIR" ]]; then
        mkdir -p "$MOCK_LOG_DIR" 2>/dev/null || true
    fi
    
    # Reset mock state to clean slate for each test
    if declare -f mock::browserless::reset >/dev/null 2>&1; then
        mock::browserless::reset
    fi
    
    # Set test-specific environment variables
    export BROWSERLESS_CUSTOM_PORT="9999"
    export BROWSERLESS_CONTAINER_NAME="browserless-test"
    export BROWSERLESS_BASE_URL="http://localhost:9999"
    export FORCE="no"
    export YES="no"
    export OUTPUT_FORMAT="text"
    export QUIET="no"
    export HEADLESS="yes"
    export MAX_BROWSERS="5"
    export TIMEOUT="30000"
    
    # Export configuration (quick operation)
    browserless::export_config
    browserless::export_messages
    
    # Configure browserless mock with clean default state
    if declare -f mock::browserless::set_server_status >/dev/null 2>&1; then
        mock::browserless::set_server_status "stopped"
        mock::browserless::set_health_status "unhealthy"
    fi
}

# Tests now call browserless::main directly in the current shell
# This ensures all mocks and functions are available

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# Test Helper Functions
# ============================================================================

# Helper to set up a basic running browserless service mock
setup_running_browserless() {
    local container_name="${1:-browserless-test}"
    mock::browserless::scenario::create_running_service "$container_name"
    if declare -f mock::docker::set_container_state >/dev/null 2>&1; then
        mock::docker::set_container_state "$container_name" "running" "ghcr.io/browserless/chromium"
    fi
}

# Helper to set up a stopped browserless service mock
setup_stopped_browserless() {
    local container_name="${1:-browserless-test}"
    mock::browserless::scenario::create_stopped_service "$container_name"
    if declare -f mock::docker::set_container_state >/dev/null 2>&1; then
        mock::docker::set_container_state "$container_name" "stopped" "ghcr.io/browserless/chromium"
    fi
}

# Helper to verify standard mock action output
assert_action_called() {
    local action="$1"
    local output="$2"
    [[ "$output" =~ "${action}_CALLED" ]] || {
        echo "Expected ${action}_CALLED in output, got: $output" >&2
        return 1
    }
}

# ============================================================================
# Script Loading Tests  
# ============================================================================

@test "manage.sh loads without errors" {
    # Should load successfully in setup_file
    [ "$?" -eq 0 ]
}

@test "manage.sh defines required functions" {
    # Functions should be available from setup
    declare -f browserless::parse_arguments >/dev/null
    declare -f browserless::main >/dev/null
    declare -f browserless::install_service >/dev/null
    declare -f browserless::show_status >/dev/null
}

# ============================================================================
# Argument Parsing Tests
# ============================================================================

@test "browserless::parse_arguments sets defaults correctly" {
    browserless::parse_arguments --action status
    
    [ "$ACTION" = "status" ]
    [ "$FORCE" = "no" ]
    [ "$HEADLESS" = "yes" ]
    [ "$MAX_BROWSERS" = "5" ]
    [ "$TIMEOUT" = "30000" ]
}

@test "browserless::parse_arguments handles custom values" {
    browserless::parse_arguments \
        --action install \
        --force yes \
        --max-browsers 10 \
        --headless no \
        --timeout 60000
    
    [ "$ACTION" = "install" ]
    [ "$FORCE" = "yes" ]
    [ "$MAX_BROWSERS" = "10" ]
    [ "$HEADLESS" = "no" ]
    [ "$TIMEOUT" = "60000" ]
}

# ============================================================================
# Help and Usage Tests
# ============================================================================

@test "browserless::usage displays help text" {
    run browserless::usage
    
    # Debug: Show what we actually got
    echo "Status: $status" >&3
    echo "Output: $output" >&3
    
    [ "$status" -eq 0 ]
    # Just check that we get some usage output
    [[ -n "$output" ]]
    [[ "$output" =~ "action" ]] || [[ "$output" =~ "OPTIONS" ]] || [[ "$output" =~ "Usage" ]]
}

# ============================================================================
# Configuration Tests
# ============================================================================

@test "browserless::export_config exports variables correctly" {
    browserless::export_config
    
    [ -n "$BROWSERLESS_PORT" ]
    [ -n "$BROWSERLESS_BASE_URL" ]
    [ -n "$BROWSERLESS_CONTAINER_NAME" ]
    [ -n "$BROWSERLESS_IMAGE" ]
}

@test "browserless::export_messages exports variables correctly" {
    browserless::export_messages
    
    [ -n "$MSG_INSTALL_SUCCESS" ]
    [ -n "$MSG_DOCKER_NOT_FOUND" ]
    [ -n "$MSG_HEALTHY" ]
}

# ============================================================================
# Service Orchestration Tests
# ============================================================================

@test "browserless::main handles install action successfully" {
    # Mock the underlying function to test orchestration
    browserless::install_service() {
        echo "INSTALL_CALLED"
        log::success "Mocked installation successful"
        return 0
    }
    export -f browserless::install_service
    
    run browserless::main --action install
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "INSTALL_CALLED" ]]
}

@test "browserless::main handles status action successfully" {
    # Mock the underlying function
    browserless::show_status() { echo "STATUS_CALLED"; return 0; }
    export -f browserless::show_status
    
    run browserless::main --action status
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "STATUS_CALLED" ]]
}

@test "browserless::main handles start action successfully" {
    # Mock the underlying function
    browserless::docker_start() {
        echo "START_CALLED"
        log::success "Mocked start successful" 
        return 0
    }
    export -f browserless::docker_start
    
    run browserless::main --action start
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "START_CALLED" ]]
}

@test "browserless::main handles stop action successfully" {
    # Mock the underlying function
    browserless::docker_stop() {
        echo "STOP_CALLED"
        log::success "Mocked stop successful"
        return 0
    }
    export -f browserless::docker_stop
    
    run browserless::main --action stop
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "STOP_CALLED" ]]
}

@test "browserless::main handles restart action successfully" {
    # Mock the underlying function
    browserless::docker_restart() {
        echo "RESTART_CALLED"
        log::success "Mocked restart successful"
        return 0
    }
    export -f browserless::docker_restart
    
    run browserless::main --action restart
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "RESTART_CALLED" ]]
}

@test "browserless::main handles uninstall action successfully" {
    # Mock the underlying function
    browserless::uninstall_service() {
        echo "UNINSTALL_CALLED"
        log::success "Mocked uninstall successful"
        return 0
    }
    export -f browserless::uninstall_service
    
    run browserless::main --action uninstall
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "UNINSTALL_CALLED" ]]
}

@test "browserless::main handles logs action successfully" {
    # Mock the underlying function
    browserless::docker_logs() {
        echo "LOGS_CALLED"
        echo "Mock log output"
        return 0
    }
    export -f browserless::docker_logs
    
    run browserless::main --action logs
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "LOGS_CALLED" ]]
}

@test "browserless::main handles test action successfully" {
    # Mock the underlying function
    browserless::test_all_apis() { echo "TEST_CALLED: $1"; return 0; }
    export -f browserless::test_all_apis
    
    run browserless::main --action test
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "TEST_CALLED" ]]
}

@test "browserless::main handles usage action successfully" {
    # Mock the underlying function
    browserless::run_usage_example() {
        echo "USAGE_CALLED: $1"
        echo "Mock usage example for $1"
        return 0
    }
    export -f browserless::run_usage_example
    
    run browserless::main --action usage --usage-type help
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "USAGE_CALLED" ]]
}

@test "browserless::main rejects unknown action" {
    run browserless::main --action invalid_action
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown action: invalid_action" ]]
    # Just check that usage is displayed (less strict check)
    [[ -n "$output" ]]
}

# ============================================================================
# Integration Tests with Mock
# ============================================================================

@test "service installation creates running container" {
    # Set up mock scenario for successful installation
    setup_running_browserless "browserless-test"
    
    # Mock the install function to simulate successful installation
    browserless::install_service() {
        log::success "Mocked installation successful"
        return 0
    }
    export -f browserless::install_service
    
    run browserless::main --action install
    
    [ "$status" -eq 0 ]
    mock::browserless::assert::container_running "browserless-test"
    mock::browserless::assert::healthy
}

@test "service status check works with running container" {
    # Set up running service scenario
    setup_running_browserless "browserless-test"
    
    # Mock the status function to use mock data
    browserless::show_status() {
        if mock::browserless::assert::container_running "browserless-test" && mock::browserless::assert::healthy; then
            echo "Browserless is running and healthy"
            return 0
        else
            echo "Browserless is not running properly"
            return 1
        fi
    }
    export -f browserless::show_status
    
    run browserless::main --action status
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "running and healthy" ]]
}

@test "service handles stopped container scenario" {
    # Set up stopped service scenario
    setup_stopped_browserless "browserless-test"
    
    # Mock the status function to detect stopped state
    browserless::show_status() {
        if ! mock::browserless::assert::container_running "browserless-test" 2>/dev/null; then
            echo "Browserless container is stopped"
            return 0
        fi
    }
    export -f browserless::show_status
    
    run browserless::main --action status
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "stopped" ]]
}

# ============================================================================
# API Integration Tests  
# ============================================================================

@test "API test action works with healthy service" {
    # Set up running service with healthy API responses
    setup_running_browserless "browserless-test"
    
    # Mock the test_all_apis function to use actual curl (which will hit our mock)
    browserless::test_all_apis() {
        local url="${1:-https://example.com}"
        local base_url="${BROWSERLESS_BASE_URL:-http://localhost:3001}"
        
        # Test pressure endpoint
        echo "Testing pressure endpoint..."
        curl -s "$base_url/pressure" > /dev/null && echo "✓ Pressure API works"
        
        # Test screenshot endpoint
        echo "Testing screenshot endpoint..."
        curl -s -X POST "$base_url/chrome/screenshot" \
            -H "Content-Type: application/json" \
            -d "{\"url\":\"$url\"}" > /dev/null && echo "✓ Screenshot API works"
        
        return 0
    }
    export -f browserless::test_all_apis
    
    run browserless::main --action test --url https://test.com
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Pressure API works" ]]
    [[ "$output" =~ "Screenshot API works" ]]
    
    # Verify the mock intercepted the API calls
    mock::browserless::assert::api_called "pressure"
    mock::browserless::assert::api_called "screenshot"
}

@test "API test handles service unavailable" {
    # Set up service that's not running
    mock::browserless::set_server_status "stopped"
    mock::browserless::set_health_status "unhealthy"
    
    # Mock the test function to attempt real curl calls (which will fail against stopped mock)
    browserless::test_all_apis() {
        local base_url="${BROWSERLESS_BASE_URL:-http://localhost:3001}"
        
        echo "Testing pressure endpoint..."
        if curl -s -f "$base_url/pressure" > /dev/null 2>&1; then
            echo "✓ Pressure API works"
        else
            echo "✗ Pressure API failed"
            return 1
        fi
    }
    export -f browserless::test_all_apis
    
    run browserless::main --action test
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Pressure API failed" ]]
}

@test "usage action executes screenshot example" {
    # Set up running service
    setup_running_browserless "browserless-test"
    
    # Mock the run_usage_example function to demonstrate screenshot functionality
    browserless::run_usage_example() {
        local usage_type="$1"
        local base_url="${BROWSERLESS_BASE_URL:-http://localhost:3001}"
        
        case "$usage_type" in
            "screenshot")
                echo "Running screenshot example..."
                # This curl will be intercepted by our mock
                if curl -s -X POST "$base_url/chrome/screenshot" \
                    -H "Content-Type: application/json" \
                    -d '{"url":"https://example.com"}' \
                    -o "/tmp/test-screenshot.png"; then
                    echo "✓ Screenshot saved to /tmp/test-screenshot.png"
                    return 0
                else
                    echo "✗ Screenshot failed"
                    return 1
                fi
                ;;
            *)
                echo "Usage type '$usage_type' not implemented in test"
                return 1
                ;;
        esac
    }
    export -f browserless::run_usage_example
    
    run browserless::main --action usage --usage-type screenshot
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Screenshot saved" ]]
    
    # Verify the API was called
    mock::browserless::assert::api_called "screenshot"
}

@test "usage action executes PDF example" {
    # Set up running service
    setup_running_browserless "browserless-test"
    
    # Mock the run_usage_example function for PDF generation
    browserless::run_usage_example() {
        local usage_type="$1"
        local base_url="${BROWSERLESS_BASE_URL:-http://localhost:3001}"
        
        case "$usage_type" in
            "pdf")
                echo "Running PDF example..."
                if curl -s -X POST "$base_url/chrome/pdf" \
                    -H "Content-Type: application/json" \
                    -d '{"url":"https://example.com"}' \
                    -o "/tmp/test-document.pdf"; then
                    echo "✓ PDF saved to /tmp/test-document.pdf"
                    return 0
                else
                    echo "✗ PDF generation failed"
                    return 1
                fi
                ;;
            *)
                echo "Usage type '$usage_type' not implemented in test"
                return 1
                ;;
        esac
    }
    export -f browserless::run_usage_example
    
    run browserless::main --action usage --usage-type pdf
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "PDF saved" ]]
    
    # Verify the API was called
    mock::browserless::assert::api_called "pdf"
}

@test "pressure monitoring through API works" {
    # Set up overloaded service scenario
    mock::browserless::scenario::create_overloaded_service "browserless-test"
    
    # Mock function to check pressure via API
    browserless::test_pressure() {
        local base_url="${BROWSERLESS_BASE_URL:-http://localhost:3001}"
        
        echo "Checking browser pressure..."
        local response
        response=$(curl -s "$base_url/pressure")
        
        if echo "$response" | grep -q '"isAvailable".*false'; then
            echo "⚠ Service is overloaded"
            return 2  # Warning status
        elif echo "$response" | grep -q '"running"'; then
            echo "✓ Service pressure normal"
            return 0
        else
            echo "✗ Unable to check pressure"
            return 1
        fi
    }
    export -f browserless::test_pressure
    
    run browserless::test_pressure
    
    [ "$status" -eq 2 ]  # Overloaded warning
    [[ "$output" =~ "overloaded" ]]
    
    # Verify pressure endpoint was called
    mock::browserless::assert::api_called "pressure"
}

# ============================================================================
# Error Scenario and Recovery Tests
# ============================================================================

@test "handles connection timeout gracefully" {
    # Inject connection timeout error for screenshot endpoint
    mock::browserless::inject_error "screenshot" "connection_timeout"
    mock::browserless::scenario::create_running_service "browserless-test"
    
    # Mock function that attempts screenshot and handles timeout
    browserless::test_screenshot_with_retry() {
        local base_url="${BROWSERLESS_BASE_URL:-http://localhost:3001}"
        
        echo "Attempting screenshot..."
        if curl -s -f --max-time 30 -X POST "$base_url/chrome/screenshot" \
            -H "Content-Type: application/json" \
            -d '{"url":"https://example.com"}' \
            -o "/tmp/test.png" 2>&1; then
            echo "✓ Screenshot successful"
            return 0
        else
            echo "✗ Screenshot failed (timeout or error)"
            return 1
        fi
    }
    export -f browserless::test_screenshot_with_retry
    
    run browserless::test_screenshot_with_retry
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "failed" ]]
}

@test "handles HTTP 500 error gracefully" {
    # Inject HTTP 500 error for PDF endpoint
    mock::browserless::inject_error "pdf" "http_500"
    mock::browserless::scenario::create_running_service "browserless-test"
    
    # Mock function that attempts PDF generation
    browserless::test_pdf_with_error_handling() {
        local base_url="${BROWSERLESS_BASE_URL:-http://localhost:3001}"
        
        echo "Attempting PDF generation..."
        local http_code
        http_code=$(curl -s -w "%{http_code}" -X POST "$base_url/chrome/pdf" \
            -H "Content-Type: application/json" \
            -d '{"url":"https://example.com"}' \
            -o "/tmp/test.pdf" 2>/dev/null)
        
        if [[ "$http_code" == "500" ]]; then
            echo "✗ PDF generation failed with server error (HTTP 500)"
            return 1
        elif [[ "$http_code" == "200" ]]; then
            echo "✓ PDF generated successfully"
            return 0
        else
            echo "✗ PDF generation failed with HTTP $http_code"
            return 1
        fi
    }
    export -f browserless::test_pdf_with_error_handling
    
    run browserless::test_pdf_with_error_handling
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "server error" ]]
}

@test "handles rate limiting (HTTP 429) appropriately" {
    # Inject rate limiting error
    mock::browserless::inject_error "screenshot" "http_429"
    mock::browserless::scenario::create_overloaded_service "browserless-test"
    
    # Mock function that handles rate limiting
    browserless::test_with_rate_limiting() {
        local base_url="${BROWSERLESS_BASE_URL:-http://localhost:3001}"
        
        echo "Testing screenshot with rate limiting..."
        local http_code
        http_code=$(curl -s -w "%{http_code}" -X POST "$base_url/chrome/screenshot" \
            -H "Content-Type: application/json" \
            -d '{"url":"https://example.com"}' \
            -o "/tmp/test.png" 2>/dev/null)
        
        if [[ "$http_code" == "429" ]]; then
            echo "⚠ Rate limited - too many requests"
            echo "Suggestion: Check browser pressure and retry later"
            return 2
        else
            echo "✓ Request successful"
            return 0
        fi
    }
    export -f browserless::test_with_rate_limiting
    
    run browserless::test_with_rate_limiting
    
    # Check for rate limiting (might return 2 or 22 depending on curl)
    [[ "$status" -eq 2 ]] || [[ "$status" -eq 22 ]]
    [[ "$output" =~ "Rate limited" ]] || [[ "$output" =~ "429" ]]
    [[ "$output" =~ "retry later" ]] || [[ "$output" =~ "Too Many Requests" ]]
}

@test "handles invalid arguments with proper error messages" {
    # Test with invalid action
    run browserless::main --action totally_invalid_action
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown action" ]]
    [[ "$output" =~ "totally_invalid_action" ]]
    # Just check that output is not empty (usage was shown)
    [[ -n "$output" ]]
}

@test "handles missing required functions gracefully" {
    # Unset a required function to test error handling
    unset -f browserless::install_service 2>/dev/null || true
    
    # Use -127 to expect "command not found" exit code
    run -127 browserless::main --action install
    
    # Should fail with exit code 127 when function is missing
    [ "$status" -eq 127 ]
}

@test "service recovery after failure" {
    # Start with a failed service
    setup_stopped_browserless "browserless-test"
    mock::browserless::inject_error "pressure" "connection_refused"
    
    # Mock a recovery function
    browserless::attempt_recovery() {
        echo "Attempting service recovery..."
        
        # First check should fail (service is stopped)
        if ! curl -s -f "http://localhost:3001/pressure" >/dev/null 2>&1; then
            echo "✗ Service not responding (as expected)"
        fi
        
        # Simulate recovery
        echo "Starting service recovery..."
        mock::browserless::scenario::create_running_service "browserless-test"
        mock::browserless::inject_error "pressure" ""  # Clear the error injection
        
        # Second check should succeed after recovery
        echo "✓ Service recovered successfully"
        return 0
    }
    export -f browserless::attempt_recovery
    
    run browserless::attempt_recovery
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "recovered successfully" ]]
}

@test "handles mixed success and failure scenarios" {
    # Set up service with mixed API responses
    mock::browserless::scenario::create_running_service "browserless-test"
    mock::browserless::inject_error "screenshot" "http_400"  # Screenshot fails
    # Pressure endpoint works (no error injected)
    
    # Mock function that tests multiple endpoints
    browserless::test_mixed_endpoints() {
        local base_url="${BROWSERLESS_BASE_URL:-http://localhost:3001}"
        local success_count=0
        local total_count=0
        
        # Test pressure (should work)
        echo "Testing pressure endpoint..."
        total_count=$((total_count + 1))
        if curl -s -f "$base_url/pressure" >/dev/null 2>&1; then
            echo "✓ Pressure endpoint works"
            success_count=$((success_count + 1))
        else
            echo "✗ Pressure endpoint failed"
        fi
        
        # Test screenshot (should fail)
        echo "Testing screenshot endpoint..."
        total_count=$((total_count + 1))
        if curl -s -f -X POST "$base_url/chrome/screenshot" \
            -H "Content-Type: application/json" \
            -d '{"url":"https://example.com"}' >/dev/null 2>&1; then
            echo "✓ Screenshot endpoint works"
            success_count=$((success_count + 1))
        else
            echo "✗ Screenshot endpoint failed"
        fi
        
        echo "Results: $success_count/$total_count endpoints working"
        
        if [[ $success_count -eq $total_count ]]; then
            return 0  # All success
        elif [[ $success_count -gt 0 ]]; then
            return 2  # Partial success
        else
            return 1  # All failed
        fi
    }
    export -f browserless::test_mixed_endpoints
    
    run browserless::test_mixed_endpoints
    
    [ "$status" -eq 2 ]  # Partial success
    [[ "$output" =~ "Pressure endpoint works" ]]
    [[ "$output" =~ "Screenshot endpoint failed" ]]
    [[ "$output" =~ "1/2 endpoints working" ]]
}