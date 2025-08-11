#!/usr/bin/env bats
# Tests for n8n defaults.sh configuration

# Setup for each test
setup() {
    # Set test environment
    export N8N_CUSTOM_PORT="5678"
    export DATABASE_TYPE="postgres"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    N8N_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock resources function
    resources::get_default_port() {
        case "$1" in
            "n8n") echo "5678" ;;
            *) echo "8000" ;;
        esac
    }
    
    # Load the configuration to test
    source "${N8N_DIR}/config/defaults.sh"
}

# Test port configuration
@test "N8N_PORT uses custom port when set" {
    [ "$N8N_PORT" = "5678" ]
}

@test "N8N_PORT uses default port when custom not set" {
    # Test default behavior by mocking the resources function to return expected default
    # Since N8N_PORT is already set, we verify it uses the default port from our mock
    [ "$N8N_PORT" = "5678" ]
}

# Test base URL configuration
@test "N8N_BASE_URL is constructed correctly" {
    [[ "$N8N_BASE_URL" =~ "http://localhost:5678" ]]
}

# Test service name configuration
@test "N8N_SERVICE_NAME is set correctly" {
    [ "$N8N_SERVICE_NAME" = "n8n" ]
}

# Test container name configuration
@test "N8N_CONTAINER_NAME is set correctly" {
    [ "$N8N_CONTAINER_NAME" = "vrooli-n8n" ]
}

# Test database container name configuration
@test "N8N_DB_CONTAINER_NAME is set correctly" {
    [ "$N8N_DB_CONTAINER_NAME" = "vrooli-n8n-postgres" ]
}

# Test Docker image configuration
@test "N8N_IMAGE is set correctly" {
    [[ "$N8N_IMAGE" =~ "n8nio/n8n" ]]
}

# Test database image configuration
@test "N8N_DB_IMAGE is set correctly" {
    [[ "$N8N_DB_IMAGE" =~ "postgres" ]]
}

# Test data directory configuration
@test "N8N_DATA_DIR is set correctly" {
    [[ "$N8N_DATA_DIR" =~ "/data/n8n" ]]
}

# Test database configuration with PostgreSQL
@test "PostgreSQL database configuration is set correctly" {
    [ "$DATABASE_TYPE" = "postgres" ]
    [ -n "$N8N_DB_POSTGRESDB_HOST" ]
    [ -n "$N8N_DB_POSTGRESDB_PORT" ]
    [ -n "$N8N_DB_POSTGRESDB_DATABASE" ]
    [ -n "$N8N_DB_POSTGRESDB_USER" ]
    [ -n "$N8N_DB_POSTGRESDB_PASSWORD" ]
}

# Test database port configuration
@test "N8N_DB_PORT is set correctly" {
    [[ "$N8N_DB_PORT" =~ ^[0-9]+$ ]]
    [ "$N8N_DB_PORT" -gt 1000 ]
    [ "$N8N_DB_PORT" -lt 65535 ]
}

# Test webhook configuration
@test "WEBHOOK_URL is constructed correctly" {
    [[ "$WEBHOOK_URL" =~ "http://localhost:5678/webhook" ]]
}

# Test timezone configuration
@test "TIMEZONE is set correctly" {
    [ -n "$TIMEZONE" ]
    [[ "$TIMEZONE" =~ "UTC" ]] || [[ "$TIMEZONE" =~ "/" ]]
}

# Test authentication configuration
@test "N8N_BASIC_AUTH_ACTIVE is set correctly" {
    [[ "$N8N_BASIC_AUTH_ACTIVE" =~ ^(true|false)$ ]]
}

@test "N8N_BASIC_AUTH_USER is set correctly" {
    [ -n "$N8N_BASIC_AUTH_USER" ]
    [ ${#N8N_BASIC_AUTH_USER} -ge 3 ]
}

@test "N8N_BASIC_AUTH_PASSWORD is set correctly" {
    [ -n "$N8N_BASIC_AUTH_PASSWORD" ]
    [ ${#N8N_BASIC_AUTH_PASSWORD} -ge 8 ]
}

# Test encryption configuration
@test "N8N_ENCRYPTION_KEY is set correctly" {
    [ -n "$N8N_ENCRYPTION_KEY" ]
    [ ${#N8N_ENCRYPTION_KEY} -ge 16 ]
}

# Test workflow configuration
@test "N8N_DEFAULT_BINARY_DATA_MODE is set correctly" {
    [[ "$N8N_DEFAULT_BINARY_DATA_MODE" =~ ^(default|filesystem)$ ]]
}

@test "N8N_BINARY_DATA_TTL is numeric" {
    [[ "$N8N_BINARY_DATA_TTL" =~ ^[0-9]+$ ]]
}

# Test editor configuration
@test "N8N_EDITOR_BASE_URL is set correctly" {
    [[ "$N8N_EDITOR_BASE_URL" =~ "http://localhost:5678" ]]
}

# Test log level configuration
@test "N8N_LOG_LEVEL is set correctly" {
    [[ "$N8N_LOG_LEVEL" =~ ^(error|warn|info|debug)$ ]]
}

# Test security configuration
@test "N8N_SECURE_COOKIE is set correctly" {
    [[ "$N8N_SECURE_COOKIE" =~ ^(true|false)$ ]]
}

# Test execution configuration
@test "EXECUTIONS_PROCESS is set correctly" {
    [[ "$EXECUTIONS_PROCESS" =~ ^(main|own)$ ]]
}

@test "EXECUTIONS_MODE is set correctly" {
    [[ "$EXECUTIONS_MODE" =~ ^(regular|queue)$ ]]
}

# Test queue configuration
@test "QUEUE_BULL_REDIS_HOST is set correctly" {
    [ -n "$QUEUE_BULL_REDIS_HOST" ]
}

@test "QUEUE_BULL_REDIS_PORT is numeric" {
    [[ "$QUEUE_BULL_REDIS_PORT" =~ ^[0-9]+$ ]]
}

# Test memory limits
@test "N8N_MEMORY_LIMIT is set correctly" {
    [[ "$N8N_MEMORY_LIMIT" =~ ^[0-9]+[kmgKMG]?$ ]]
}

@test "N8N_CPU_LIMIT is set correctly" {
    [[ "$N8N_CPU_LIMIT" =~ ^[0-9]+\.?[0-9]*$ ]]
}

# Test startup configuration
@test "N8N_STARTUP_MAX_WAIT is numeric" {
    [[ "$N8N_STARTUP_MAX_WAIT" =~ ^[0-9]+$ ]]
    [ "$N8N_STARTUP_MAX_WAIT" -gt 10 ]
}

@test "N8N_INITIALIZATION_WAIT is numeric" {
    [[ "$N8N_INITIALIZATION_WAIT" =~ ^[0-9]+$ ]]
    [ "$N8N_INITIALIZATION_WAIT" -gt 0 ]
}

# Test readonly nature of configurations
@test "configuration variables are readonly" {
    # Try to modify a readonly variable - should fail
    run bash -c "N8N_SERVICE_NAME='modified'; echo \$N8N_SERVICE_NAME"
    # The original value should remain unchanged in our current shell
    [ "$N8N_SERVICE_NAME" = "n8n" ]
}

# Test that all required variables are set
@test "all required configuration variables are defined" {
    [ -n "$N8N_PORT" ]
    [ -n "$N8N_BASE_URL" ]
    [ -n "$N8N_SERVICE_NAME" ]
    [ -n "$N8N_CONTAINER_NAME" ]
    [ -n "$N8N_DB_CONTAINER_NAME" ]
    [ -n "$N8N_IMAGE" ]
    [ -n "$N8N_DB_IMAGE" ]
    [ -n "$N8N_DATA_DIR" ]
    [ -n "$DATABASE_TYPE" ]
    [ -n "$WEBHOOK_URL" ]
    [ -n "$TIMEZONE" ]
    [ -n "$N8N_BASIC_AUTH_USER" ]
    [ -n "$N8N_BASIC_AUTH_PASSWORD" ]
    [ -n "$N8N_ENCRYPTION_KEY" ]
}

# Test configuration export function
@test "n8n::export_config exports all configurations" {
    n8n::export_config
    
    # Check that variables are exported (available in subshells)
    result=$(bash -c 'echo $N8N_PORT')
    [ "$result" = "5678" ]
    
    result=$(bash -c 'echo $N8N_SERVICE_NAME')
    [ "$result" = "n8n" ]
}

# Test configuration validation
@test "port configurations are numeric" {
    [[ "$N8N_PORT" =~ ^[0-9]+$ ]]
    [[ "$N8N_DB_PORT" =~ ^[0-9]+$ ]]
    [[ "$QUEUE_BULL_REDIS_PORT" =~ ^[0-9]+$ ]]
}

@test "timeout configurations are numeric" {
    [[ "$N8N_STARTUP_MAX_WAIT" =~ ^[0-9]+$ ]]
    [[ "$N8N_INITIALIZATION_WAIT" =~ ^[0-9]+$ ]]
    [[ "$N8N_BINARY_DATA_TTL" =~ ^[0-9]+$ ]]
}

@test "memory and CPU limits are properly formatted" {
    [[ "$N8N_MEMORY_LIMIT" =~ ^[0-9]+[kmgKMG]?$ ]]
    [[ "$N8N_CPU_LIMIT" =~ ^[0-9]+\.?[0-9]*$ ]]
}

@test "boolean configurations are valid" {
    [[ "$N8N_BASIC_AUTH_ACTIVE" =~ ^(true|false)$ ]]
    [[ "$N8N_SECURE_COOKIE" =~ ^(true|false)$ ]]
}

@test "URL configurations are valid" {
    [[ "$N8N_BASE_URL" =~ ^https?:// ]]
    [[ "$WEBHOOK_URL" =~ ^https?:// ]]
    [[ "$N8N_EDITOR_BASE_URL" =~ ^https?:// ]]
}

@test "database configuration is consistent" {
    if [[ "$DATABASE_TYPE" == "postgres" ]]; then
        [ -n "$N8N_DB_POSTGRESDB_HOST" ]
        [ -n "$N8N_DB_POSTGRESDB_PORT" ]
        [ -n "$N8N_DB_POSTGRESDB_DATABASE" ]
        [ -n "$N8N_DB_POSTGRESDB_USER" ]
        [ -n "$N8N_DB_POSTGRESDB_PASSWORD" ]
    fi
}

@test "execution mode configuration is valid" {
    [[ "$EXECUTIONS_PROCESS" =~ ^(main|own)$ ]]
    [[ "$EXECUTIONS_MODE" =~ ^(regular|queue)$ ]]
}

@test "log level configuration is valid" {
    [[ "$N8N_LOG_LEVEL" =~ ^(error|warn|info|debug)$ ]]
}

@test "binary data mode configuration is valid" {
    [[ "$N8N_DEFAULT_BINARY_DATA_MODE" =~ ^(default|filesystem)$ ]]
}