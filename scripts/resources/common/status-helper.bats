#!/usr/bin/env bats

# Test suite for status-helper.sh
# Tests the standardized status helper utilities

# Setup test directory and source var.sh first
COMMON_DIR="$BATS_TEST_DIRNAME"
_HERE="$COMMON_DIR"

# shellcheck disable=SC1091
source "${_HERE}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/test_helper.bash"

# Load mocks for testing
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/docker.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/jq.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/http.sh"

# Setup and teardown
setup() {
    # Reset mock states
    if command -v mock::docker::reset &>/dev/null; then
        mock::docker::reset
    fi
    if command -v mock::jq::reset &>/dev/null; then
        mock::jq::reset
    fi
    if command -v mock::http::reset &>/dev/null; then
        mock::http::reset
    fi
}

teardown() {
    # Clean up test artifacts
    rm -rf /tmp/test_status_* 2>/dev/null || true
}

# ============================================================================
# Loading and basic tests
# ============================================================================

@test "status-helper.sh can be sourced" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        echo 'SUCCESS'
    "
    
    assert_success
    [[ "$output" =~ "SUCCESS" ]]
}

@test "status constants are defined correctly" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        echo \"HEALTHY: \$STATUS_HEALTHY\"
        echo \"DEGRADED: \$STATUS_DEGRADED\"
        echo \"ERROR: \$STATUS_ERROR\"
        echo \"UNKNOWN: \$STATUS_UNKNOWN\"
    "
    
    assert_success
    [[ "$output" =~ "HEALTHY: healthy" ]]
    [[ "$output" =~ "DEGRADED: degraded" ]]
    [[ "$output" =~ "ERROR: error" ]]
    [[ "$output" =~ "UNKNOWN: unknown" ]]
}

# ============================================================================
# Function existence tests
# ============================================================================

@test "status_helper::format_response function exists" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        type -t status_helper::format_response
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "status_helper::is_port_listening function exists" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        type -t status_helper::is_port_listening
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "status_helper::check_http_health function exists" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        type -t status_helper::check_http_health
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "status_helper::check_docker_container function exists" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        type -t status_helper::check_docker_container
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "status_helper::get_docker_details function exists" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        type -t status_helper::get_docker_details
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "status_helper::check_service function exists" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        type -t status_helper::check_service
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "status_helper::check_integration function exists" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        type -t status_helper::check_integration
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "status_helper::display_human function exists" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        type -t status_helper::display_human
    "
    
    assert_success
    [ "$output" = "function" ]
}

@test "status_helper::validate_dependencies function exists" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        type -t status_helper::validate_dependencies
    "
    
    assert_success
    [ "$output" = "function" ]
}

# ============================================================================
# Format response tests
# ============================================================================

@test "format_response generates valid JSON" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        status_helper::format_response 'test-service' 'healthy' 'Service is running' | jq -e '.'
    "
    
    assert_success
}

@test "format_response includes all required fields" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        response=\$(status_helper::format_response 'test-service' 'healthy' 'Service is running')
        echo \"\$response\" | jq -r '.service'
        echo \"\$response\" | jq -r '.status'
        echo \"\$response\" | jq -r '.message'
        echo \"\$response\" | jq -r '.integration_ready'
        echo \"\$response\" | jq -r '.timestamp'
    "
    
    assert_success
    [[ "$output" =~ "test-service" ]]
    [[ "$output" =~ "healthy" ]]
    [[ "$output" =~ "Service is running" ]]
    [[ "$output" =~ "false" ]]
    [[ "$output" =~ [0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2} ]]
}

@test "format_response handles optional details" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        details='{\"port\": 8080, \"version\": \"1.0.0\"}'
        response=\$(status_helper::format_response 'test-service' 'healthy' 'Running' \"\$details\")
        echo \"\$response\" | jq -r '.details.port'
        echo \"\$response\" | jq -r '.details.version'
    "
    
    assert_success
    [[ "$output" =~ "8080" ]]
    [[ "$output" =~ "1.0.0" ]]
}

@test "format_response handles invalid JSON in details" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        status_helper::format_response 'test-service' 'healthy' 'Running' 'invalid json' | jq -e '.details.raw'
    "
    
    assert_success
    [[ "$output" =~ "invalid json" ]]
}

# ============================================================================
# Port checking tests
# ============================================================================

@test "is_port_listening detects open port with netstat" {
    run bash -c "
        # Mock netstat command
        netstat() {
            echo 'tcp        0      0 0.0.0.0:8080            0.0.0.0:*               LISTEN'
            return 0
        }
        export -f netstat
        command() {
            if [[ \"\$1\" == '-v' && \"\$2\" == 'netstat' ]]; then
                return 0
            fi
            builtin command \"\$@\"
        }
        export -f command
        
        source '$COMMON_DIR/status-helper.sh'
        status_helper::is_port_listening 8080
    "
    
    assert_success
}

@test "is_port_listening detects closed port" {
    run bash -c "
        # Mock netstat to return no results
        netstat() {
            return 0
        }
        export -f netstat
        lsof() {
            return 1
        }
        export -f lsof
        curl() {
            return 7
        }
        export -f curl
        command() {
            if [[ \"\$1\" == '-v' ]]; then
                return 1
            fi
            builtin command \"\$@\"
        }
        export -f command
        
        source '$COMMON_DIR/status-helper.sh'
        status_helper::is_port_listening 9999
    "
    
    [ "$status" -eq 1 ]
}

# ============================================================================
# HTTP health check tests
# ============================================================================

@test "check_http_health succeeds with 200 response" {
    run bash -c "
        # Mock curl to return 200
        curl() {
            echo 'response_body200'
            return 0
        }
        export -f curl
        
        source '$COMMON_DIR/status-helper.sh'
        status_helper::check_http_health 'http://localhost:8080' '/health' '200'
    "
    
    assert_success
}

@test "check_http_health fails with wrong status code" {
    run bash -c "
        # Mock curl to return 404
        curl() {
            echo 'not_found404'
            return 0
        }
        export -f curl
        
        source '$COMMON_DIR/status-helper.sh'
        status_helper::check_http_health 'http://localhost:8080' '/health' '200'
    "
    
    [ "$status" -eq 1 ]
}

# ============================================================================
# Docker container tests
# ============================================================================

@test "check_docker_container detects running container" {
    # Use the mock docker infrastructure
    mock::docker::add_container "test-container" "running" "test:latest"
    
    run bash -c "
        source '$COMMON_DIR/../../../scripts/__test/fixtures/mocks/docker.sh'
        source '$COMMON_DIR/status-helper.sh'
        
        # Override docker command to use mock
        docker() {
            if [[ \"\$1\" == 'ps' ]]; then
                echo 'test-container'
                return 0
            fi
            return 1
        }
        export -f docker
        command() {
            if [[ \"\$1\" == '-v' && \"\$2\" == 'docker' ]]; then
                return 0
            fi
            builtin command \"\$@\"
        }
        export -f command
        
        status_helper::check_docker_container 'test-container'
    "
    
    assert_success
}

@test "check_docker_container fails when container not running" {
    run bash -c "
        # Mock docker ps to return empty
        docker() {
            if [[ \"\$1\" == 'ps' ]]; then
                echo ''
                return 0
            fi
            return 1
        }
        export -f docker
        command() {
            if [[ \"\$1\" == '-v' && \"\$2\" == 'docker' ]]; then
                return 0
            fi
            builtin command \"\$@\"
        }
        export -f command
        
        source '$COMMON_DIR/status-helper.sh'
        status_helper::check_docker_container 'nonexistent-container'
    "
    
    [ "$status" -eq 1 ]
}

# ============================================================================
# Service check tests
# ============================================================================

@test "check_service returns healthy status when all checks pass" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        
        # Override check functions to succeed
        status_helper::is_port_listening() { return 0; }
        status_helper::check_docker_container() { return 0; }
        status_helper::get_docker_details() { echo '{}'; }
        status_helper::check_http_health() { return 0; }
        curl() { echo '{}'; return 0; }
        export -f curl
        
        response=\$(status_helper::check_service 'test-service' '8080' '/health' 'test-container')
        echo \"\$response\" | jq -r '.status'
    "
    
    assert_success
    [ "$output" = "healthy" ]
}

@test "check_service returns degraded when port is open but health check fails" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        
        # Port is listening but health check fails
        status_helper::is_port_listening() { return 0; }
        status_helper::check_docker_container() { return 1; }
        status_helper::get_docker_details() { echo '{}'; }
        status_helper::check_http_health() { return 1; }
        
        response=\$(status_helper::check_service 'test-service' '8080' '/health')
        echo \"\$response\" | jq -r '.status'
    "
    
    assert_success
    [ "$output" = "degraded" ]
}

@test "check_service returns error when service is not running" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        
        # All checks fail
        status_helper::is_port_listening() { return 1; }
        status_helper::check_docker_container() { return 1; }
        status_helper::get_docker_details() { echo '{}'; }
        status_helper::check_http_health() { return 1; }
        
        response=\$(status_helper::check_service 'test-service' '8080' '/health')
        echo \"\$response\" | jq -r '.status'
    "
    
    assert_success
    [ "$output" = "error" ]
}

# ============================================================================
# Human display tests
# ============================================================================

@test "display_human shows healthy status with emoji" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        
        json='{
            \"service\": \"test-service\",
            \"status\": \"healthy\",
            \"message\": \"Service is running\",
            \"integration_ready\": true,
            \"next_actions\": []
        }'
        
        status_helper::display_human \"\$json\"
    "
    
    assert_success
    [[ "$output" =~ "✅" ]]
    [[ "$output" =~ "test-service" ]]
    [[ "$output" =~ "Service is running" ]]
    [[ "$output" =~ "Integration ready" ]]
}

@test "display_human shows next actions when available" {
    run bash -c "
        source '$COMMON_DIR/status-helper.sh'
        
        json='{
            \"service\": \"test-service\",
            \"status\": \"error\",
            \"message\": \"Service failed\",
            \"integration_ready\": false,
            \"next_actions\": [\"Check logs\", \"Restart service\"]
        }'
        
        status_helper::display_human \"\$json\"
    "
    
    assert_success
    [[ "$output" =~ "❌" ]]
    [[ "$output" =~ "Check logs" ]]
    [[ "$output" =~ "Restart service" ]]
}

# ============================================================================
# Dependency validation tests
# ============================================================================

@test "validate_dependencies succeeds when all tools available" {
    run bash -c "
        # Mock command to say jq and curl exist
        command() {
            if [[ \"\$1\" == '-v' ]]; then
                return 0
            fi
            builtin command \"\$@\"
        }
        export -f command
        
        source '$COMMON_DIR/status-helper.sh'
        status_helper::validate_dependencies
    "
    
    assert_success
}

@test "validate_dependencies fails when jq is missing" {
    run bash -c "
        # Mock command to say jq is missing
        command() {
            if [[ \"\$1\" == '-v' && \"\$2\" == 'jq' ]]; then
                return 1
            elif [[ \"\$1\" == '-v' && \"\$2\" == 'curl' ]]; then
                return 0
            fi
            builtin command \"\$@\"
        }
        export -f command
        
        source '$COMMON_DIR/status-helper.sh'
        status_helper::validate_dependencies 2>&1
    "
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Missing required dependencies" ]]
    [[ "$output" =~ "jq" ]]
}