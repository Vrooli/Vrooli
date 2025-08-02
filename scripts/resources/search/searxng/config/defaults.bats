#!/usr/bin/env bats
# Tests for SearXNG defaults.sh configuration

# Setup for each test
setup() {
    # Enable test mode to prevent readonly variables
    export SEARXNG_TEST_MODE=\"yes\"
    
    # Set test environment variables
    export SEARXNG_CUSTOM_PORT="8200"
    export SEARXNG_SECRET_KEY="test-secret-key-32-characters-long"
    export SEARXNG_BIND_ADDRESS="0.0.0.0"
    export SEARXNG_ENABLE_REDIS="yes"
    export SEARXNG_REDIS_HOST="test-redis"
    export SEARXNG_REDIS_PORT="6380"
    export SEARXNG_RATE_LIMIT="20"
    
    # Mock resources function
    resources::get_default_port() {
        echo "9200"
    }
    
    # Load the defaults
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    source "${SCRIPT_DIR}/defaults.sh"
}

# ============================================================================
# Configuration Export Tests
# ============================================================================

@test "searxng::export_config sets all required variables" {
    searxng::export_config
    
    # Test basic configuration
    [ "$SEARXNG_PORT" = "8200" ]  # Custom port should override default
    [ "$SEARXNG_BASE_URL" = "http://localhost:8200" ]
    [ "$SEARXNG_CONTAINER_NAME" = "searxng" ]
    [ "$SEARXNG_IMAGE" = "searxng/searxng:2025.1.31-157c9267e" ]
    [ "$SEARXNG_NETWORK_NAME" = "searxng-network" ]
}

@test "searxng::export_config uses default port when custom not set" {
    unset SEARXNG_CUSTOM_PORT
    searxng::export_config
    
    [ "$SEARXNG_PORT" = "9200" ]  # Should use default from mock function
    [ "$SEARXNG_BASE_URL" = "http://localhost:9200" ]
}

@test "searxng::export_config sets data directory correctly" {
    searxng::export_config
    
    [[ "$SEARXNG_DATA_DIR" =~ "searxng" ]]
    [ -n "$SEARXNG_DATA_DIR" ]
}

@test "searxng::export_config generates secret key when not provided" {
    unset SEARXNG_SECRET_KEY
    searxng::export_config
    
    [ -n "$SEARXNG_SECRET_KEY" ]
    [ ${#SEARXNG_SECRET_KEY} -ge 32 ]
}

@test "searxng::export_config uses provided secret key" {
    searxng::export_config
    
    [ "$SEARXNG_SECRET_KEY" = "test-secret-key-32-characters-long" ]
}

@test "searxng::export_config sets bind address correctly" {
    searxng::export_config
    
    [ "$SEARXNG_BIND_ADDRESS" = "0.0.0.0" ]
}

@test "searxng::export_config uses default bind address when not set" {
    unset SEARXNG_BIND_ADDRESS
    searxng::export_config
    
    [ "$SEARXNG_BIND_ADDRESS" = "127.0.0.1" ]  # Should default to localhost only
}

# ============================================================================
# Search Configuration Tests
# ============================================================================

@test "searxng::export_config sets search defaults" {
    searxng::export_config
    
    [ -n "$SEARXNG_DEFAULT_ENGINES" ]
    [ -n "$SEARXNG_SAFE_SEARCH" ]
    # SEARXNG_AUTOCOMPLETE can be empty (no autocomplete) - this is valid
    [ "$SEARXNG_AUTOCOMPLETE" = "" ]
    [ -n "$SEARXNG_DEFAULT_LANG" ]
    [ "$SEARXNG_DEFAULT_LANG" = "en" ]
}

@test "searxng::export_config sets instance information" {
    searxng::export_config
    
    [ -n "$SEARXNG_INSTANCE_NAME" ]
    [ -n "$SEARXNG_CONTACT_URL" ]
    [[ "$SEARXNG_INSTANCE_NAME" =~ "SearXNG" ]]
}

# ============================================================================
# Performance Configuration Tests
# ============================================================================

@test "searxng::export_config sets performance parameters" {
    searxng::export_config
    
    [ -n "$SEARXNG_REQUEST_TIMEOUT" ]
    [ -n "$SEARXNG_MAX_REQUEST_TIMEOUT" ]
    [ -n "$SEARXNG_POOL_CONNECTIONS" ]
    [ -n "$SEARXNG_POOL_MAXSIZE" ]
    [ -n "$SEARXNG_API_TIMEOUT" ]
    
    # Test numeric values
    [[ "$SEARXNG_REQUEST_TIMEOUT" =~ ^[0-9]+$ ]]
    [[ "$SEARXNG_MAX_REQUEST_TIMEOUT" =~ ^[0-9]+$ ]]
    [[ "$SEARXNG_POOL_CONNECTIONS" =~ ^[0-9]+$ ]]
    [[ "$SEARXNG_POOL_MAXSIZE" =~ ^[0-9]+$ ]]
    [[ "$SEARXNG_API_TIMEOUT" =~ ^[0-9]+$ ]]
}

@test "searxng::export_config validates timeout relationships" {
    searxng::export_config
    
    # Max timeout should be greater than regular timeout
    [ "$SEARXNG_MAX_REQUEST_TIMEOUT" -gt "$SEARXNG_REQUEST_TIMEOUT" ]
}

# ============================================================================
# Redis Configuration Tests
# ============================================================================

@test "searxng::export_config handles Redis configuration when enabled" {
    searxng::export_config
    
    [ "$SEARXNG_ENABLE_REDIS" = "yes" ]
    [ "$SEARXNG_REDIS_HOST" = "test-redis" ]
    [ "$SEARXNG_REDIS_PORT" = "6380" ]
}

@test "searxng::export_config handles Redis configuration when disabled" {
    export SEARXNG_ENABLE_REDIS="no"
    searxng::export_config
    
    [ "$SEARXNG_ENABLE_REDIS" = "no" ]
    # Should still have default values even when disabled
    [ -n "$SEARXNG_REDIS_HOST" ]
    [ -n "$SEARXNG_REDIS_PORT" ]
}

@test "searxng::export_config uses default Redis settings when not specified" {
    unset SEARXNG_REDIS_HOST
    unset SEARXNG_REDIS_PORT
    searxng::export_config
    
    [ "$SEARXNG_REDIS_HOST" = "redis" ]
    [ "$SEARXNG_REDIS_PORT" = "6379" ]
}

# ============================================================================
# Security Configuration Tests
# ============================================================================

@test "searxng::export_config sets security defaults" {
    searxng::export_config
    
    [ -n "$SEARXNG_LIMITER_ENABLED" ]
    [ -n "$SEARXNG_RATE_LIMIT" ]
    [ -n "$SEARXNG_ENABLE_PUBLIC_ACCESS" ]
    
    # Test that rate limit is numeric
    [[ "$SEARXNG_RATE_LIMIT" =~ ^[0-9]+$ ]]
}

@test "searxng::export_config uses custom rate limit" {
    searxng::export_config
    
    [ "$SEARXNG_RATE_LIMIT" = "20" ]
}

@test "searxng::export_config uses default rate limit when not set" {
    unset SEARXNG_RATE_LIMIT
    searxng::export_config
    
    [ "$SEARXNG_RATE_LIMIT" = "10" ]  # Default rate limit
}

@test "searxng::export_config sets public access to no by default" {
    unset SEARXNG_ENABLE_PUBLIC_ACCESS
    searxng::export_config
    
    [ "$SEARXNG_ENABLE_PUBLIC_ACCESS" = "no" ]  # Should default to secure
}

# ============================================================================
# Readonly Variable Tests
# ============================================================================

@test "all critical configuration variables are readonly" {
    searxng::export_config
    
    # Test that these variables are declared as readonly
    run bash -c "declare -p SEARXNG_PORT"
    [[ "$output" =~ "readonly" ]]
    
    run bash -c "declare -p SEARXNG_CONTAINER_NAME"
    [[ "$output" =~ "readonly" ]]
    
    run bash -c "declare -p SEARXNG_IMAGE"
    [[ "$output" =~ "readonly" ]]
}

# ============================================================================
# Variable Validation Tests
# ============================================================================

@test "searxng::export_config validates port range" {
    export SEARXNG_CUSTOM_PORT="8200"
    searxng::export_config
    
    [ "$SEARXNG_PORT" -ge 1024 ]
    [ "$SEARXNG_PORT" -le 65535 ]
}

@test "searxng::export_config validates secret key length" {
    searxng::export_config
    
    [ ${#SEARXNG_SECRET_KEY} -ge 32 ]
}

@test "searxng::export_config validates boolean values" {
    searxng::export_config
    
    [[ "$SEARXNG_ENABLE_REDIS" =~ ^(yes|no)$ ]]
    [[ "$SEARXNG_LIMITER_ENABLED" =~ ^(yes|no)$ ]]
    [[ "$SEARXNG_ENABLE_PUBLIC_ACCESS" =~ ^(yes|no)$ ]]
    # SEARXNG_AUTOCOMPLETE is not a boolean - it's a string that can be empty or contain provider name
    [[ "$SEARXNG_ENABLE_METRICS" =~ ^(yes|no)$ ]]
}

# ============================================================================
# Function Existence Tests
# ============================================================================

@test "searxng::export_config function exists and is callable" {
    run type searxng::export_config
    [ "$status" -eq 0 ]
    [[ "$output" =~ "function" ]]
}

@test "searxng::export_config can be called multiple times safely" {
    searxng::export_config
    local first_secret="$SEARXNG_SECRET_KEY"
    
    searxng::export_config
    local second_secret="$SEARXNG_SECRET_KEY"
    
    # Secret should remain the same when called multiple times
    [ "$first_secret" = "$second_secret" ]
}

# ============================================================================
# Environment Integration Tests
# ============================================================================

@test "searxng::export_config respects all environment overrides" {
    export SEARXNG_CUSTOM_PORT="9999"
    export SEARXNG_SECRET_KEY="custom-secret-key-for-testing-purposes"
    export SEARXNG_BIND_ADDRESS="192.168.1.100"
    export SEARXNG_ENABLE_REDIS="yes"
    export SEARXNG_RATE_LIMIT="50"
    export SEARXNG_ENABLE_PUBLIC_ACCESS="yes"
    
    searxng::export_config
    
    [ "$SEARXNG_PORT" = "9999" ]
    [ "$SEARXNG_SECRET_KEY" = "custom-secret-key-for-testing-purposes" ]
    [ "$SEARXNG_BIND_ADDRESS" = "192.168.1.100" ]
    [ "$SEARXNG_ENABLE_REDIS" = "yes" ]
    [ "$SEARXNG_RATE_LIMIT" = "50" ]
    [ "$SEARXNG_ENABLE_PUBLIC_ACCESS" = "yes" ]
}