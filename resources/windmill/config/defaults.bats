#!/usr/bin/env bats
# Tests for Windmill defaults.sh configuration

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Setup for each test
setup() {
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    WINDMILL_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Set up test environment variables that might be referenced
    export HOME="/tmp/test-home"
    export USER="testuser"
    export VROOLI_DIR="/tmp/vrooli"
    
    # Create test directories
    mkdir -p "$HOME/.vrooli"
    mkdir -p "$VROOLI_DIR"
    
    # Load the defaults configuration to test
    source "${WINDMILL_DIR}/config/defaults.sh"
}

# Cleanup after each test
teardown() {
    trash::safe_remove "/tmp/test-home" --test-cleanup
    trash::safe_remove "/tmp/vrooli" --test-cleanup
}

# Test core service configuration
@test "WINDMILL_SERVICE_NAME is defined and valid" {
    [ -n "$WINDMILL_SERVICE_NAME" ]
    [[ "$WINDMILL_SERVICE_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]
}


@test "WINDMILL_CONTAINER_NAME is defined and valid" {
    [ -n "$WINDMILL_CONTAINER_NAME" ]
    [[ "$WINDMILL_CONTAINER_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]
}


@test "WINDMILL_DB_CONTAINER_NAME is defined and valid" {
    [ -n "$WINDMILL_DB_CONTAINER_NAME" ]
    [[ "$WINDMILL_DB_CONTAINER_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]
}


# Test port configuration
@test "WINDMILL_DEFAULT_PORT is defined and valid" {
    [ -n "$WINDMILL_DEFAULT_PORT" ]
    [[ "$WINDMILL_DEFAULT_PORT" =~ ^[0-9]+$ ]]
    [ "$WINDMILL_DEFAULT_PORT" -gt 1024 ]
    [ "$WINDMILL_DEFAULT_PORT" -lt 65536 ]
}


@test "WINDMILL_DB_PORT is defined and valid" {
    [ -n "$WINDMILL_DB_PORT" ]
    [[ "$WINDMILL_DB_PORT" =~ ^[0-9]+$ ]]
    [ "$WINDMILL_DB_PORT" -gt 1024 ]
    [ "$WINDMILL_DB_PORT" -lt 65536 ]
}


# Test URL configuration
@test "WINDMILL_BASE_URL is defined and valid" {
    [ -n "$WINDMILL_BASE_URL" ]
    [[ "$WINDMILL_BASE_URL" =~ ^http://localhost:[0-9]+$ ]]
}


@test "WINDMILL_API_BASE is defined and valid" {
    [ -n "$WINDMILL_API_BASE" ]
    [[ "$WINDMILL_API_BASE" =~ /api$ ]]
}


# Test Docker image configuration
@test "WINDMILL_DEFAULT_IMAGE is defined and valid" {
    [ -n "$WINDMILL_DEFAULT_IMAGE" ]
    [[ "$WINDMILL_DEFAULT_IMAGE" =~ ^[a-zA-Z0-9._/-]+:[a-zA-Z0-9._-]+$ ]]
}


@test "WINDMILL_DB_IMAGE is defined and valid" {
    [ -n "$WINDMILL_DB_IMAGE" ]
    [[ "$WINDMILL_DB_IMAGE" =~ ^[a-zA-Z0-9._/-]+:[a-zA-Z0-9._-]+$ ]]
    [[ "$WINDMILL_DB_IMAGE" =~ postgres ]]
}


# Test directory configuration
@test "WINDMILL_DATA_DIR is defined and valid" {
    [ -n "$WINDMILL_DATA_DIR" ]
    [[ "$WINDMILL_DATA_DIR" =~ ^/.+ ]]  # Absolute path
}


@test "WINDMILL_LOG_DIR is defined and valid" {
    [ -n "$WINDMILL_LOG_DIR" ]
    [[ "$WINDMILL_LOG_DIR" =~ ^/.+ ]]  # Absolute path
}


@test "WINDMILL_BACKUP_DIR is defined and valid" {
    [ -n "$WINDMILL_BACKUP_DIR" ]
    [[ "$WINDMILL_BACKUP_DIR" =~ ^/.+ ]]  # Absolute path
}


@test "WINDMILL_CONFIG_DIR is defined and valid" {
    [ -n "$WINDMILL_CONFIG_DIR" ]
    [[ "$WINDMILL_CONFIG_DIR" =~ ^/.+ ]]  # Absolute path
}


# Test database configuration
@test "WINDMILL_DB_NAME is defined and valid" {
    [ -n "$WINDMILL_DB_NAME" ]
    [[ "$WINDMILL_DB_NAME" =~ ^[a-zA-Z0-9_]+$ ]]
}


@test "WINDMILL_DB_USER is defined and valid" {
    [ -n "$WINDMILL_DB_USER" ]
    [[ "$WINDMILL_DB_USER" =~ ^[a-zA-Z0-9_]+$ ]]
}


@test "WINDMILL_DEFAULT_DB_PASSWORD is defined and valid" {
    [ -n "$WINDMILL_DEFAULT_DB_PASSWORD" ]
    [[ ${#WINDMILL_DEFAULT_DB_PASSWORD} -ge 8 ]]  # At least 8 characters
}


# Test admin user configuration
@test "WINDMILL_DEFAULT_ADMIN_EMAIL is defined and valid" {
    [ -n "$WINDMILL_DEFAULT_ADMIN_EMAIL" ]
    [[ "$WINDMILL_DEFAULT_ADMIN_EMAIL" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]
}


@test "WINDMILL_DEFAULT_ADMIN_PASSWORD is defined and valid" {
    [ -n "$WINDMILL_DEFAULT_ADMIN_PASSWORD" ]
    [[ ${#WINDMILL_DEFAULT_ADMIN_PASSWORD} -ge 8 ]]  # At least 8 characters
}


# Test worker configuration
@test "WINDMILL_DEFAULT_WORKERS is defined and valid" {
    [ -n "$WINDMILL_DEFAULT_WORKERS" ]
    [[ "$WINDMILL_DEFAULT_WORKERS" =~ ^[0-9]+$ ]]
    [ "$WINDMILL_DEFAULT_WORKERS" -gt 0 ]
    [ "$WINDMILL_DEFAULT_WORKERS" -le 32 ]
}


# Test health check configuration
@test "WINDMILL_HEALTH_CHECK_TIMEOUT is defined and valid" {
    [ -n "$WINDMILL_HEALTH_CHECK_TIMEOUT" ]
    [[ "$WINDMILL_HEALTH_CHECK_TIMEOUT" =~ ^[0-9]+$ ]]
    [ "$WINDMILL_HEALTH_CHECK_TIMEOUT" -gt 0 ]
}


@test "WINDMILL_HEALTH_CHECK_INTERVAL is defined and valid" {
    [ -n "$WINDMILL_HEALTH_CHECK_INTERVAL" ]
    [[ "$WINDMILL_HEALTH_CHECK_INTERVAL" =~ ^[0-9]+$ ]]
    [ "$WINDMILL_HEALTH_CHECK_INTERVAL" -gt 0 ]
}


@test "WINDMILL_STARTUP_TIMEOUT is defined and valid" {
    [ -n "$WINDMILL_STARTUP_TIMEOUT" ]
    [[ "$WINDMILL_STARTUP_TIMEOUT" =~ ^[0-9]+$ ]]
    [ "$WINDMILL_STARTUP_TIMEOUT" -gt 0 ]
}


# Test backup configuration
@test "WINDMILL_BACKUP_RETENTION_DAYS is defined and valid" {
    [ -n "$WINDMILL_BACKUP_RETENTION_DAYS" ]
    [[ "$WINDMILL_BACKUP_RETENTION_DAYS" =~ ^[0-9]+$ ]]
    [ "$WINDMILL_BACKUP_RETENTION_DAYS" -gt 0 ]
}


# Test logging configuration
@test "WINDMILL_LOG_LEVEL is defined and valid" {
    [ -n "$WINDMILL_LOG_LEVEL" ]
    [[ "$WINDMILL_LOG_LEVEL" =~ ^(DEBUG|INFO|WARN|ERROR)$ ]]
}


# Test resource limits
@test "WINDMILL_MEMORY_LIMIT is defined and valid" {
    [ -n "$WINDMILL_MEMORY_LIMIT" ]
    [[ "$WINDMILL_MEMORY_LIMIT" =~ ^[0-9]+[GMg]?$ ]]
}


@test "WINDMILL_CPU_LIMIT is defined and valid" {
    [ -n "$WINDMILL_CPU_LIMIT" ]
    [[ "$WINDMILL_CPU_LIMIT" =~ ^[0-9]*\.?[0-9]+$ ]]
}


@test "WINDMILL_DB_MEMORY_LIMIT is defined and valid" {
    [ -n "$WINDMILL_DB_MEMORY_LIMIT" ]
    [[ "$WINDMILL_DB_MEMORY_LIMIT" =~ ^[0-9]+[GMg]?$ ]]
}


# Test network configuration
@test "WINDMILL_NETWORK_NAME is defined and valid" {
    [ -n "$WINDMILL_NETWORK_NAME" ]
    [[ "$WINDMILL_NETWORK_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]
}


# Test volume configuration
@test "WINDMILL_VOLUME_PREFIX is defined and valid" {
    [ -n "$WINDMILL_VOLUME_PREFIX" ]
    [[ "$WINDMILL_VOLUME_PREFIX" =~ ^[a-zA-Z0-9_-]+$ ]]
}


# Test environment variables array
@test "WINDMILL_ENV_VARS is defined as array" {
    [ -n "$WINDMILL_ENV_VARS" ]
    # Check if it's an array
    declare -p WINDMILL_ENV_VARS | grep -q "declare -a"
}


# Test volumes array
@test "WINDMILL_VOLUMES is defined as array" {
    [ -n "$WINDMILL_VOLUMES" ]
    # Check if it's an array
    declare -p WINDMILL_VOLUMES | grep -q "declare -a"
}


# Test Docker Compose configuration
@test "WINDMILL_COMPOSE_PROJECT_NAME is defined and valid" {
    [ -n "$WINDMILL_COMPOSE_PROJECT_NAME" ]
    [[ "$WINDMILL_COMPOSE_PROJECT_NAME" =~ ^[a-zA-Z0-9_-]+$ ]]
}


# Test workspace configuration
@test "WINDMILL_DEFAULT_WORKSPACE is defined and valid" {
    [ -n "$WINDMILL_DEFAULT_WORKSPACE" ]
    [[ "$WINDMILL_DEFAULT_WORKSPACE" =~ ^[a-zA-Z0-9_-]+$ ]]
}


# Test SSL/TLS configuration
@test "WINDMILL_SSL_ENABLED is defined and valid" {
    [ -n "$WINDMILL_SSL_ENABLED" ]
    [[ "$WINDMILL_SSL_ENABLED" =~ ^(true|false)$ ]]
}


@test "WINDMILL_SSL_CERT_PATH is defined when SSL enabled" {
    if [[ "$WINDMILL_SSL_ENABLED" == "true" ]]; then
        [ -n "$WINDMILL_SSL_CERT_PATH" ]
        [[ "$WINDMILL_SSL_CERT_PATH" =~ ^/.+ ]]  # Absolute path
    fi
}


@test "WINDMILL_SSL_KEY_PATH is defined when SSL enabled" {
    if [[ "$WINDMILL_SSL_ENABLED" == "true" ]]; then
        [ -n "$WINDMILL_SSL_KEY_PATH" ]
        [[ "$WINDMILL_SSL_KEY_PATH" =~ ^/.+ ]]  # Absolute path
    fi
}


# Test authentication configuration
@test "WINDMILL_AUTH_TYPE is defined and valid" {
    [ -n "$WINDMILL_AUTH_TYPE" ]
    [[ "$WINDMILL_AUTH_TYPE" =~ ^(local|ldap|oauth|saml)$ ]]
}


# Test OAuth configuration (if applicable)
@test "OAuth configuration is valid when enabled" {
    if [[ "$WINDMILL_AUTH_TYPE" == "oauth" ]]; then
        [ -n "$WINDMILL_OAUTH_CLIENT_ID" ]
        [ -n "$WINDMILL_OAUTH_CLIENT_SECRET" ]
        [ -n "$WINDMILL_OAUTH_REDIRECT_URI" ]
    fi
}


# Test LDAP configuration (if applicable)
@test "LDAP configuration is valid when enabled" {
    if [[ "$WINDMILL_AUTH_TYPE" == "ldap" ]]; then
        [ -n "$WINDMILL_LDAP_SERVER" ]
        [ -n "$WINDMILL_LDAP_BASE_DN" ]
    fi
}


# Test configuration export function
@test "windmill::export_config (CORE - needs implementation)" {
    skip "windmill::export_config is a core function but not yet implemented"
}


# Test custom configuration override
@test "custom configuration can override defaults" {
    # Set custom values
    export WINDMILL_CUSTOM_PORT="9999"
    export WINDMILL_CUSTOM_IMAGE="custom/windmill:test"
    
    # Reload configuration
    source "${WINDMILL_DIR}/config/defaults.sh"
    
    # Check that custom values are used
    [[ "$WINDMILL_BASE_URL" =~ 9999 ]]
}


# Test directory path consistency
@test "all data directories are under WINDMILL_DATA_DIR" {
    [[ "$WINDMILL_LOG_DIR" =~ ^$WINDMILL_DATA_DIR ]] || [[ "$WINDMILL_LOG_DIR" =~ ^/var/log ]]
    [[ "$WINDMILL_BACKUP_DIR" =~ ^$WINDMILL_DATA_DIR ]] || [[ "$WINDMILL_BACKUP_DIR" =~ ^/opt ]]
}


# Test port uniqueness
@test "all ports are unique" {
    [ "$WINDMILL_DEFAULT_PORT" != "$WINDMILL_DB_PORT" ]
}


# Test resource limit validation
@test "resource limits are reasonable" {
    # Memory limit should be at least 1GB
    if [[ "$WINDMILL_MEMORY_LIMIT" =~ ^([0-9]+)G$ ]]; then
        local mem_gb="${BASH_REMATCH[1]}"
        [ "$mem_gb" -ge 1 ]
    fi
    
    # CPU limit should be reasonable (0.1 to 16.0)
    if [[ "$WINDMILL_CPU_LIMIT" =~ ^([0-9]*\.?[0-9]+)$ ]]; then
        local cpu_val="${BASH_REMATCH[1]}"
        # Use awk for floating point comparison
        awk -v cpu="$cpu_val" 'BEGIN { exit (cpu >= 0.1 && cpu <= 16.0) ? 0 : 1 }'
    fi
}


# Test environment variables format
@test "environment variables have valid format" {
    for env_var in "${WINDMILL_ENV_VARS[@]}"; do
        [[ "$env_var" =~ ^[A-Z_]+=.* ]]
    done
}


# Test volume mounts format
@test "volume mounts have valid format" {
    for volume in "${WINDMILL_VOLUMES[@]}"; do
        [[ "$volume" =~ ^/.+:/.+ ]]
    done
}


# Test configuration validation function

# Test configuration with missing required variables

# Test configuration file generation

# Test Docker Compose file generation

# Test configuration backup

# Test configuration restoration

# Test database configuration validation
@test "database configuration is consistent" {
    # Database container and service should use same credentials
    [ -n "$WINDMILL_DB_NAME" ]
    [ -n "$WINDMILL_DB_USER" ]
    [ -n "$WINDMILL_DEFAULT_DB_PASSWORD" ]
}


# Test worker configuration validation
@test "worker configuration is reasonable" {
    # Worker count should be reasonable for typical systems
    [ "$WINDMILL_DEFAULT_WORKERS" -ge 1 ]
    [ "$WINDMILL_DEFAULT_WORKERS" -le 16 ]
}


# Test timeout configuration validation
@test "timeout values are reasonable" {
    # Health check timeout should be less than interval
    [ "$WINDMILL_HEALTH_CHECK_TIMEOUT" -lt "$WINDMILL_HEALTH_CHECK_INTERVAL" ]
    
    # Startup timeout should be greater than health check timeout
    [ "$WINDMILL_STARTUP_TIMEOUT" -gt "$WINDMILL_HEALTH_CHECK_TIMEOUT" ]
}


# Test default workspace validation
@test "default workspace name is valid" {
    # Should be a valid workspace identifier
    [[ "$WINDMILL_DEFAULT_WORKSPACE" =~ ^[a-z][a-z0-9_-]*$ ]]
    [[ ${#WINDMILL_DEFAULT_WORKSPACE} -ge 2 ]]
}


# Test backup retention policy
@test "backup retention policy is reasonable" {
    # Should retain backups for at least 7 days
    [ "$WINDMILL_BACKUP_RETENTION_DAYS" -ge 7 ]
    
    # Should not retain backups for more than 1 year
    [ "$WINDMILL_BACKUP_RETENTION_DAYS" -le 365 ]
}


# Test security configuration
@test "security configuration is appropriate" {
    # Default passwords should be complex enough
    [[ ${#WINDMILL_DEFAULT_DB_PASSWORD} -ge 12 ]]
    [[ ${#WINDMILL_DEFAULT_ADMIN_PASSWORD} -ge 12 ]]
    
    # Passwords should contain mixed characters
    [[ "$WINDMILL_DEFAULT_DB_PASSWORD" =~ [A-Z] ]]
    [[ "$WINDMILL_DEFAULT_DB_PASSWORD" =~ [a-z] ]]
    [[ "$WINDMILL_DEFAULT_DB_PASSWORD" =~ [0-9] ]]
