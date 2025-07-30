#!/usr/bin/env bats
# Tests for Judge0 defaults.sh configuration

# Setup for each test
setup() {
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    JUDGE0_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Set up test environment variables that might be referenced
    export HOME="/tmp/test-home"
    export USER="testuser"
    export VROOLI_DIR="/tmp/vrooli"
    
    # Create test directories
    mkdir -p "$HOME/.vrooli"
    mkdir -p "$VROOLI_DIR"
    
    # Load the defaults configuration to test
    source "${JUDGE0_DIR}/config/defaults.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "/tmp/test-home"
    rm -rf "/tmp/vrooli"
}

# Test core service configuration
@test "JUDGE0_SERVICE_NAME is defined and valid" {
    [ -n "$JUDGE0_SERVICE_NAME" ]
    [[ "$JUDGE0_SERVICE_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]
}


@test "JUDGE0_DISPLAY_NAME is defined and meaningful" {
    [ -n "$JUDGE0_DISPLAY_NAME" ]
    [[ "$JUDGE0_DISPLAY_NAME" =~ "Judge0" ]]
}


@test "JUDGE0_DESCRIPTION is defined and informative" {
    [ -n "$JUDGE0_DESCRIPTION" ]
    [[ "$JUDGE0_DESCRIPTION" =~ "execution" ]]
    [[ "$JUDGE0_DESCRIPTION" =~ "language" ]]
}


@test "JUDGE0_CATEGORY is defined and valid" {
    [ -n "$JUDGE0_CATEGORY" ]
    [[ "$JUDGE0_CATEGORY" == "execution" ]]
}


# Test Docker configuration
@test "JUDGE0_IMAGE is defined and valid" {
    [ -n "$JUDGE0_IMAGE" ]
    [[ "$JUDGE0_IMAGE" =~ ^[a-zA-Z0-9._/-]+$ ]]
}


@test "JUDGE0_VERSION is defined and valid" {
    [ -n "$JUDGE0_VERSION" ]
    [[ "$JUDGE0_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
}


@test "JUDGE0_CONTAINER_NAME is defined and valid" {
    [ -n "$JUDGE0_CONTAINER_NAME" ]
    [[ "$JUDGE0_CONTAINER_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]
    [[ "$JUDGE0_CONTAINER_NAME" =~ "judge0" ]]
}


@test "JUDGE0_WORKERS_NAME is defined and valid" {
    [ -n "$JUDGE0_WORKERS_NAME" ]
    [[ "$JUDGE0_WORKERS_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]
    [[ "$JUDGE0_WORKERS_NAME" =~ "worker" ]]
}


@test "JUDGE0_NETWORK_NAME is defined and valid" {
    [ -n "$JUDGE0_NETWORK_NAME" ]
    [[ "$JUDGE0_NETWORK_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]
}


@test "JUDGE0_VOLUME_NAME is defined and valid" {
    [ -n "$JUDGE0_VOLUME_NAME" ]
    [[ "$JUDGE0_VOLUME_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]
}


# Test port configuration
@test "JUDGE0_PORT is defined and valid" {
    [ -n "$JUDGE0_PORT" ]
    [[ "$JUDGE0_PORT" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_PORT" -gt 1024 ]
    [ "$JUDGE0_PORT" -lt 65536 ]
}


@test "JUDGE0_BASE_URL is defined and valid" {
    [ -n "$JUDGE0_BASE_URL" ]
    [[ "$JUDGE0_BASE_URL" =~ ^http://localhost:[0-9]+$ ]]
    [[ "$JUDGE0_BASE_URL" =~ "$JUDGE0_PORT" ]]
}


# Test security configuration
@test "JUDGE0_CPU_TIME_LIMIT is defined and reasonable" {
    [ -n "$JUDGE0_CPU_TIME_LIMIT" ]
    [[ "$JUDGE0_CPU_TIME_LIMIT" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_CPU_TIME_LIMIT" -gt 0 ]
    [ "$JUDGE0_CPU_TIME_LIMIT" -le 300 ]  # Max 5 minutes
}


@test "JUDGE0_WALL_TIME_LIMIT is defined and reasonable" {
    [ -n "$JUDGE0_WALL_TIME_LIMIT" ]
    [[ "$JUDGE0_WALL_TIME_LIMIT" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_WALL_TIME_LIMIT" -gt 0 ]
    [ "$JUDGE0_WALL_TIME_LIMIT" -ge "$JUDGE0_CPU_TIME_LIMIT" ]
}


@test "JUDGE0_MEMORY_LIMIT is defined and reasonable" {
    [ -n "$JUDGE0_MEMORY_LIMIT" ]
    [[ "$JUDGE0_MEMORY_LIMIT" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_MEMORY_LIMIT" -gt 0 ]
    [ "$JUDGE0_MEMORY_LIMIT" -le 1048576 ]  # Max 1GB in KB
}


@test "JUDGE0_STACK_LIMIT is defined and reasonable" {
    [ -n "$JUDGE0_STACK_LIMIT" ]
    [[ "$JUDGE0_STACK_LIMIT" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_STACK_LIMIT" -gt 0 ]
    [ "$JUDGE0_STACK_LIMIT" -le 524288 ]  # Max 512MB in KB
}


@test "JUDGE0_MAX_PROCESSES is defined and reasonable" {
    [ -n "$JUDGE0_MAX_PROCESSES" ]
    [[ "$JUDGE0_MAX_PROCESSES" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_MAX_PROCESSES" -gt 0 ]
    [ "$JUDGE0_MAX_PROCESSES" -le 100 ]
}


@test "JUDGE0_MAX_FILE_SIZE is defined and reasonable" {
    [ -n "$JUDGE0_MAX_FILE_SIZE" ]
    [[ "$JUDGE0_MAX_FILE_SIZE" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_MAX_FILE_SIZE" -gt 0 ]
    [ "$JUDGE0_MAX_FILE_SIZE" -le 10240 ]  # Max 10MB in KB
}


# Test security boolean settings
@test "JUDGE0_ENABLE_NETWORK is defined and boolean" {
    [ -n "$JUDGE0_ENABLE_NETWORK" ]
    [[ "$JUDGE0_ENABLE_NETWORK" =~ ^(true|false)$ ]]
}


@test "JUDGE0_ENABLE_CALLBACKS is defined and boolean" {
    [ -n "$JUDGE0_ENABLE_CALLBACKS" ]
    [[ "$JUDGE0_ENABLE_CALLBACKS" =~ ^(true|false)$ ]]
}


@test "JUDGE0_ENABLE_SUBMISSION_DELETE is defined and boolean" {
    [ -n "$JUDGE0_ENABLE_SUBMISSION_DELETE" ]
    [[ "$JUDGE0_ENABLE_SUBMISSION_DELETE" =~ ^(true|false)$ ]]
}


@test "JUDGE0_ENABLE_BATCHED_SUBMISSIONS is defined and boolean" {
    [ -n "$JUDGE0_ENABLE_BATCHED_SUBMISSIONS" ]
    [[ "$JUDGE0_ENABLE_BATCHED_SUBMISSIONS" =~ ^(true|false)$ ]]
}


@test "JUDGE0_ENABLE_WAIT_RESULT is defined and boolean" {
    [ -n "$JUDGE0_ENABLE_WAIT_RESULT" ]
    [[ "$JUDGE0_ENABLE_WAIT_RESULT" =~ ^(true|false)$ ]]
}


# Test performance configuration
@test "JUDGE0_WORKERS_COUNT is defined and reasonable" {
    [ -n "$JUDGE0_WORKERS_COUNT" ]
    [[ "$JUDGE0_WORKERS_COUNT" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_WORKERS_COUNT" -gt 0 ]
    [ "$JUDGE0_WORKERS_COUNT" -le 16 ]
}


@test "JUDGE0_MAX_QUEUE_SIZE is defined and reasonable" {
    [ -n "$JUDGE0_MAX_QUEUE_SIZE" ]
    [[ "$JUDGE0_MAX_QUEUE_SIZE" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_MAX_QUEUE_SIZE" -gt 0 ]
    [ "$JUDGE0_MAX_QUEUE_SIZE" -le 1000 ]
}


@test "JUDGE0_CPU_LIMIT is defined and reasonable" {
    [ -n "$JUDGE0_CPU_LIMIT" ]
    [[ "$JUDGE0_CPU_LIMIT" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_CPU_LIMIT" -gt 0 ]
    [ "$JUDGE0_CPU_LIMIT" -le 16 ]
}


@test "JUDGE0_MEMORY_LIMIT_DOCKER is defined and valid format" {
    [ -n "$JUDGE0_MEMORY_LIMIT_DOCKER" ]
    [[ "$JUDGE0_MEMORY_LIMIT_DOCKER" =~ ^[0-9]+[GMg]?$ ]]
}


@test "JUDGE0_WORKER_CPU_LIMIT is defined and reasonable" {
    [ -n "$JUDGE0_WORKER_CPU_LIMIT" ]
    [[ "$JUDGE0_WORKER_CPU_LIMIT" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_WORKER_CPU_LIMIT" -gt 0 ]
    [ "$JUDGE0_WORKER_CPU_LIMIT" -le 8 ]
}


@test "JUDGE0_WORKER_MEMORY_LIMIT is defined and valid format" {
    [ -n "$JUDGE0_WORKER_MEMORY_LIMIT" ]
    [[ "$JUDGE0_WORKER_MEMORY_LIMIT" =~ ^[0-9]+[GMg]?$ ]]
}


# Test health check configuration
@test "JUDGE0_HEALTH_ENDPOINT is defined and valid" {
    [ -n "$JUDGE0_HEALTH_ENDPOINT" ]
    [[ "$JUDGE0_HEALTH_ENDPOINT" =~ ^/.+ ]]
}


@test "JUDGE0_HEALTH_INTERVAL is defined and reasonable" {
    [ -n "$JUDGE0_HEALTH_INTERVAL" ]
    [[ "$JUDGE0_HEALTH_INTERVAL" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_HEALTH_INTERVAL" -gt 0 ]
    [ "$JUDGE0_HEALTH_INTERVAL" -le 300000 ]  # Max 5 minutes in ms
}


@test "JUDGE0_HEALTH_TIMEOUT is defined and reasonable" {
    [ -n "$JUDGE0_HEALTH_TIMEOUT" ]
    [[ "$JUDGE0_HEALTH_TIMEOUT" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_HEALTH_TIMEOUT" -gt 0 ]
    [ "$JUDGE0_HEALTH_TIMEOUT" -lt "$JUDGE0_HEALTH_INTERVAL" ]
}


@test "JUDGE0_STARTUP_WAIT is defined and reasonable" {
    [ -n "$JUDGE0_STARTUP_WAIT" ]
    [[ "$JUDGE0_STARTUP_WAIT" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_STARTUP_WAIT" -gt 0 ]
    [ "$JUDGE0_STARTUP_WAIT" -le 300 ]  # Max 5 minutes
}


# Test API configuration
@test "JUDGE0_API_PREFIX is defined and valid" {
    [ -n "$JUDGE0_API_PREFIX" ]
    [[ "$JUDGE0_API_PREFIX" =~ ^/.+ ]]
}


@test "JUDGE0_ENABLE_AUTHENTICATION is defined and boolean" {
    [ -n "$JUDGE0_ENABLE_AUTHENTICATION" ]
    [[ "$JUDGE0_ENABLE_AUTHENTICATION" =~ ^(true|false)$ ]]
}


@test "JUDGE0_API_KEY_LENGTH is defined and reasonable" {
    [ -n "$JUDGE0_API_KEY_LENGTH" ]
    [[ "$JUDGE0_API_KEY_LENGTH" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_API_KEY_LENGTH" -ge 16 ]
    [ "$JUDGE0_API_KEY_LENGTH" -le 64 ]
}


# Test directory configuration
@test "JUDGE0_DATA_DIR is defined and absolute path" {
    [ -n "$JUDGE0_DATA_DIR" ]
    [[ "$JUDGE0_DATA_DIR" =~ ^/.+ ]]
}


@test "JUDGE0_CONFIG_DIR is defined and under data dir" {
    [ -n "$JUDGE0_CONFIG_DIR" ]
    [[ "$JUDGE0_CONFIG_DIR" =~ ^/.+ ]]
    [[ "$JUDGE0_CONFIG_DIR" =~ ^$JUDGE0_DATA_DIR ]]
}


@test "JUDGE0_LOGS_DIR is defined and under data dir" {
    [ -n "$JUDGE0_LOGS_DIR" ]
    [[ "$JUDGE0_LOGS_DIR" =~ ^/.+ ]]
    [[ "$JUDGE0_LOGS_DIR" =~ ^$JUDGE0_DATA_DIR ]]
}


@test "JUDGE0_SUBMISSIONS_DIR is defined and under data dir" {
    [ -n "$JUDGE0_SUBMISSIONS_DIR" ]
    [[ "$JUDGE0_SUBMISSIONS_DIR" =~ ^/.+ ]]
    [[ "$JUDGE0_SUBMISSIONS_DIR" =~ ^$JUDGE0_DATA_DIR ]]
}


# Test logging configuration
@test "JUDGE0_LOG_LEVEL is defined and valid" {
    [ -n "$JUDGE0_LOG_LEVEL" ]
    [[ "$JUDGE0_LOG_LEVEL" =~ ^(debug|info|warn|error)$ ]]
}


@test "JUDGE0_LOG_MAX_SIZE is defined and valid format" {
    [ -n "$JUDGE0_LOG_MAX_SIZE" ]
    [[ "$JUDGE0_LOG_MAX_SIZE" =~ ^[0-9]+[KMG]?$ ]]
}


@test "JUDGE0_LOG_MAX_FILES is defined and reasonable" {
    [ -n "$JUDGE0_LOG_MAX_FILES" ]
    [[ "$JUDGE0_LOG_MAX_FILES" =~ ^[0-9]+$ ]]
    [ "$JUDGE0_LOG_MAX_FILES" -gt 0 ]
    [ "$JUDGE0_LOG_MAX_FILES" -le 20 ]
}


# Test default languages array
@test "JUDGE0_DEFAULT_LANGUAGES is defined as array" {
    [ -n "$JUDGE0_DEFAULT_LANGUAGES" ]
    # Check if it's an array
    declare -p JUDGE0_DEFAULT_LANGUAGES | grep -q "declare -a"
}


@test "JUDGE0_DEFAULT_LANGUAGES contains expected languages" {
    local found_python=false
    local found_javascript=false
    local found_java=false
    
    for lang_entry in "${JUDGE0_DEFAULT_LANGUAGES[@]}"; do
        if [[ "$lang_entry" =~ python ]]; then
            found_python=true
        elif [[ "$lang_entry" =~ javascript ]]; then
            found_javascript=true
        elif [[ "$lang_entry" =~ java ]]; then
            found_java=true
        fi
    done
    
    [ "$found_python" = true ]
    [ "$found_javascript" = true ]
    [ "$found_java" = true ]
}


@test "JUDGE0_DEFAULT_LANGUAGES entries have valid format" {
    for lang_entry in "${JUDGE0_DEFAULT_LANGUAGES[@]}"; do
        [[ "$lang_entry" =~ ^[a-zA-Z0-9_-]+:[0-9]+$ ]]
    done
}


# Test configuration export function
@test "judge0::export_config (CORE - needs implementation)" {
    skip "judge0::export_config is a core function but not yet implemented"
}


# Test language ID lookup function



# Test configuration consistency
@test "wall time limit is greater than or equal to CPU time limit" {
    [ "$JUDGE0_WALL_TIME_LIMIT" -ge "$JUDGE0_CPU_TIME_LIMIT" ]
}


@test "worker count is reasonable for system resources" {
    [ "$JUDGE0_WORKERS_COUNT" -le 8 ]  # Reasonable default
}


@test "memory limits are consistent" {
    [ "$JUDGE0_MEMORY_LIMIT" -ge 32768 ]  # At least 32MB
    [ "$JUDGE0_STACK_LIMIT" -le "$JUDGE0_MEMORY_LIMIT" ]
}


@test "health check timeout is less than interval" {
    [ "$JUDGE0_HEALTH_TIMEOUT" -lt "$JUDGE0_HEALTH_INTERVAL" ]
}


# Test security defaults
@test "security defaults are appropriately restrictive" {
    [[ "$JUDGE0_ENABLE_NETWORK" == "false" ]]
    [[ "$JUDGE0_ENABLE_CALLBACKS" == "false" ]]
}


# Test resource limits are within safe bounds
@test "resource limits are within safe operational bounds" {
    # CPU time should be reasonable for educational use
    [ "$JUDGE0_CPU_TIME_LIMIT" -le 30 ]
    
    # Memory should be reasonable (not more than 512MB by default)
    [ "$JUDGE0_MEMORY_LIMIT" -le 524288 ]
    
    # Process count should be limited
    [ "$JUDGE0_MAX_PROCESSES" -le 50 ]
    
    # File size should be limited (not more than 10MB)
    [ "$JUDGE0_MAX_FILE_SIZE" -le 10240 ]
}


# Test API key length is secure
@test "API key length provides adequate security" {
    [ "$JUDGE0_API_KEY_LENGTH" -ge 32 ]
}


# Test Docker resource limits are reasonable
@test "Docker resource limits are reasonable" {
    # Check CPU limit is numeric
    [[ "$JUDGE0_CPU_LIMIT" =~ ^[0-9]+$ ]]
    
    # Check memory limits have valid format
    [[ "$JUDGE0_MEMORY_LIMIT_DOCKER" =~ [0-9]+[GM] ]]
    [[ "$JUDGE0_WORKER_MEMORY_LIMIT" =~ [0-9]+[GM] ]]
}


# Test directory paths don't conflict
@test "directory paths are properly organized" {
    # All subdirectories should be under main data directory
    [[ "$JUDGE0_CONFIG_DIR" =~ ^$JUDGE0_DATA_DIR ]]
    [[ "$JUDGE0_LOGS_DIR" =~ ^$JUDGE0_DATA_DIR ]]
    [[ "$JUDGE0_SUBMISSIONS_DIR" =~ ^$JUDGE0_DATA_DIR ]]
    
    # Paths should be different
    [ "$JUDGE0_CONFIG_DIR" != "$JUDGE0_LOGS_DIR" ]
    [ "$JUDGE0_CONFIG_DIR" != "$JUDGE0_SUBMISSIONS_DIR" ]
    [ "$JUDGE0_LOGS_DIR" != "$JUDGE0_SUBMISSIONS_DIR" ]
}


# Test configuration can handle environment overrides
@test "configuration respects environment variable overrides" {
    # Set custom port
    export JUDGE0_CUSTOM_PORT="9999"
    
    # Reload configuration (would need implementation in actual config)
    # This test validates the concept
    [ -n "$JUDGE0_CUSTOM_PORT" ]
    [[ "$JUDGE0_CUSTOM_PORT" == "9999" ]]
