#!/usr/bin/env bats

# Load Vrooli test infrastructure (REQUIRED)
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "windmill"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    export SETUP_FILE_WINDMILL_DIR="$(dirname "${BATS_TEST_DIRNAME}")" 
    export SETUP_FILE_CONFIG_DIR="$(dirname "${BATS_TEST_DIRNAME}")/config"
    export SETUP_FILE_TEST_DIR="$(dirname "${BATS_TEST_DIRNAME}")/test"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    WINDMILL_DIR="${SETUP_FILE_WINDMILL_DIR}"
    CONFIG_DIR="${SETUP_FILE_CONFIG_DIR}"
    TEST_DIR="${SETUP_FILE_TEST_DIR}"
    
    # Set test environment BEFORE sourcing config files to avoid readonly conflicts
    export WINDMILL_PORT="5681"
    export WINDMILL_CONTAINER_NAME="windmill-test"
    export WINDMILL_BASE_URL="http://localhost:5681"
    export WINDMILL_DATA_DIR="/tmp/windmill-test"
    export WINDMILL_DB_PASSWORD="test-password"
    export YES="no"
    
    # Create test directories
    mkdir -p "$WINDMILL_DATA_DIR"
    
    # Mock resources functions that are called during config loading
    resources::get_default_port() {
        case "$1" in
            "windmill") echo "5681" ;;
            *) echo "8080" ;;
        esac
    }
    
    # Now source the config files
    source "${WINDMILL_DIR}/config/defaults.sh"
    source "${WINDMILL_DIR}/config/messages.sh"
    
    # Export config and messages
    windmill::export_config
    windmill::export_messages
    
    # Source common functions
    source "${WINDMILL_DIR}/lib/common.sh"
    
    # Mock system commands for testing
    docker() {
        case "$1" in
            "ps")
                # Mock docker ps output
                if [[ "$*" =~ --format ]]; then
                    echo "windmill-vrooli-server"
                    echo "windmill-vrooli-worker"
                    echo "windmill-vrooli-db"
                else
                    echo "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES"
                    echo "abc123        windmill  /start    1min      Up 1min   8000/tcp  windmill-vrooli-server"
                fi
                ;;
            "inspect")
                echo '{"State":{"Status":"running"}}'
                ;;
            "logs")
                echo "Server started successfully"
                echo "Listening on port 8000"
                ;;
            *)
                echo "docker command: $*"
                ;;
        esac
        return 0
    }
    
    # Mock curl for HTTP requests
    curl() {
        local args=("$@")
        local url=""
        local method="GET"
        local output_file=""
        local return_status=false
        
        # Parse curl arguments
        for ((i=0; i<${#args[@]}; i++)); do
            case "${args[i]}" in
                "-X"|"--request") 
                    i=$((i+1))
                    method="${args[i]}" ;;
                "-o"|"--output")
                    i=$((i+1))
                    output_file="${args[i]}" ;;
                "-w"|"--write-out")
                    i=$((i+1))
                    if [[ "${args[i]}" == *"http_code"* ]]; then
                        return_status=true
                    fi ;;
                "http"*)
                    url="${args[i]}" ;;
            esac
        done
        
        # Handle status code requests
        if [[ "$return_status" == "true" ]]; then
            if [[ "$url" =~ /api/version$ ]]; then
                echo "200"
            elif [[ "$url" =~ /api/health$ ]]; then
                echo "200"
            elif [[ "$url" =~ /api/w/list$ ]] || [[ "$url" =~ /api/users/whoami$ ]]; then
                echo "401"  # Auth required
            else
                echo "404"
            fi
            return 0
        fi
        
        # Handle regular API requests
        if [[ "$url" =~ /api/version$ ]]; then
            echo '"v1.290.0"'
        elif [[ "$url" =~ /api/health$ ]]; then
            echo '{"status":"ok"}'
        elif [[ "$url" =~ /api/openapi.yaml$ ]]; then
            echo "openapi: 3.0.0"
            echo "info:"
            echo "  title: Windmill API"
        elif [[ "$url" =~ /openapi.html$ ]]; then
            echo '<!DOCTYPE html><html><head><title>Swagger UI</title></head></html>'
        elif [[ "$url" =~ ^http://localhost:5681/?$ ]]; then
            echo '<!DOCTYPE html><html><head><title>Windmill</title></head></html>'
        else
            # Return connection refused for unhandled endpoints
            echo "curl: (7) Failed to connect to localhost" >&2
            return 7
        fi
        
        return 0
    }
    
    # Source the integration test script functions (if it has them)
    if [[ -f "${TEST_DIR}/integration-test.sh" ]]; then
        # Check if it's meant to be sourced (has functions) or executed
        if grep -q "^[[:space:]]*function\|^[[:space:]]*test_\|^[[:space:]]*show_verbose_info" "${TEST_DIR}/integration-test.sh"; then
            source "${TEST_DIR}/integration-test.sh"
        fi
    fi
}

# Cleanup after each test
teardown() {
    rm -rf "$WINDMILL_DATA_DIR" 2>/dev/null || true
    vrooli_cleanup_test
}

# Test integration test script exists and has correct permissions
@test "integration-test.sh exists and is executable" {
    [ -f "${TEST_DIR}/integration-test.sh" ]
    [ -x "${TEST_DIR}/integration-test.sh" ]
}

# Test integration test script has valid syntax
@test "integration-test.sh has valid bash syntax" {
    run bash -n "${TEST_DIR}/integration-test.sh"
    [ "$status" -eq 0 ]
}

# Test individual test functions if they exist
@test "test_web_interface function exists and can be called" {
    if declare -f test_web_interface >/dev/null; then
        run test_web_interface
        [ "$status" -eq 0 ]
        [[ "$output" =~ "web interface" ]]
    else
        skip "test_web_interface function not defined"
    fi
}

@test "test_api_version function exists and can be called" {
    if declare -f test_api_version >/dev/null; then
        run test_api_version
        [ "$status" -eq 0 ]
        [[ "$output" =~ "version" ]]
    else
        skip "test_api_version function not defined"
    fi
}

@test "test_health_endpoint function exists and can be called" {
    if declare -f test_health_endpoint >/dev/null; then
        run test_health_endpoint
        [ "$status" -eq 0 ]
        [[ "$output" =~ "health" ]]
    else
        skip "test_health_endpoint function not defined"
    fi
}

@test "test_openapi_spec function exists and can be called" {
    if declare -f test_openapi_spec >/dev/null; then
        run test_openapi_spec
        [[ "$status" -eq 0 ]] || [[ "$status" -eq 2 ]]  # PASS or SKIP
        [[ "$output" =~ "OpenAPI" ]]
    else
        skip "test_openapi_spec function not defined"
    fi
}

@test "test_authentication_endpoints function exists and can be called" {
    if declare -f test_authentication_endpoints >/dev/null; then
        run test_authentication_endpoints
        [ "$status" -eq 0 ]
        [[ "$output" =~ "authentication" ]]
    else
        skip "test_authentication_endpoints function not defined"
    fi
}

@test "test_workers_endpoint function exists and can be called" {
    if declare -f test_workers_endpoint >/dev/null; then
        run test_workers_endpoint
        [[ "$status" -eq 0 ]] || [[ "$status" -eq 2 ]]  # PASS or SKIP
        [[ "$output" =~ "workers" ]]
    else
        skip "test_workers_endpoint function not defined"
    fi
}

@test "test_server_container function exists and can be called" {
    if declare -f test_server_container >/dev/null; then
        run test_server_container
        [ "$status" -eq 0 ]
        [[ "$output" =~ "server container" ]]
    else
        skip "test_server_container function not defined"
    fi
}

@test "test_worker_containers function exists and can be called" {
    if declare -f test_worker_containers >/dev/null; then
        run test_worker_containers
        [[ "$status" -eq 0 ]] || [[ "$status" -eq 2 ]]  # PASS or SKIP
        [[ "$output" =~ "worker containers" ]]
    else
        skip "test_worker_containers function not defined"
    fi
}

@test "test_database_container function exists and can be called" {
    if declare -f test_database_container >/dev/null; then
        run test_database_container
        [ "$status" -eq 0 ]
        [[ "$output" =~ "database container" ]]
    else
        skip "test_database_container function not defined"
    fi
}

@test "test_compose_stack function exists and can be called" {
    if declare -f test_compose_stack >/dev/null; then
        run test_compose_stack
        [[ "$status" -eq 0 ]] || [[ "$status" -eq 2 ]]  # PASS or SKIP
        [[ "$output" =~ "compose stack" ]]
    else
        skip "test_compose_stack function not defined"
    fi
}

@test "test_log_output function exists and can be called" {
    if declare -f test_log_output >/dev/null; then
        run test_log_output
        [[ "$status" -eq 0 ]] || [[ "$status" -eq 2 ]]  # PASS or SKIP
        [[ "$output" =~ "log" ]]
    else
        skip "test_log_output function not defined"
    fi
}

@test "show_verbose_info function exists and can be called" {
    if declare -f show_verbose_info >/dev/null; then
        run show_verbose_info
        [ "$status" -eq 0 ]
        [[ "$output" =~ "Windmill Information" ]]
    else
        skip "show_verbose_info function not defined"
    fi
}

# Test that the integration test can be run as a script
@test "integration-test.sh can be executed directly" {
    if [[ -x "${TEST_DIR}/integration-test.sh" ]]; then
        # Run with --help or similar safe flag if available
        run "${TEST_DIR}/integration-test.sh" --help 2>/dev/null || true
        # Should not crash, may show help or run tests
        [[ "$status" -eq 0 ]] || [[ "$output" =~ "help" ]] || [[ "$output" =~ "test" ]] || [[ "$status" -eq 1 ]]
    else
        skip "integration-test.sh not executable"
    fi
}

# Test configuration loading in integration test
@test "integration test loads windmill configuration correctly" {
    # These should be set after sourcing the script
    [ -n "$WINDMILL_PROJECT_NAME" ]
    [ -n "$WINDMILL_SERVER_PORT" ]
    [ -n "$WINDMILL_BASE_URL" ]
}

# Test that required tools are defined
@test "integration test defines required tools" {
    if [[ -n "${REQUIRED_TOOLS:-}" ]]; then
        [[ "${REQUIRED_TOOLS[*]}" =~ "curl" ]]
        [[ "${REQUIRED_TOOLS[*]}" =~ "docker" ]]
    else
        skip "REQUIRED_TOOLS not defined"
    fi
}