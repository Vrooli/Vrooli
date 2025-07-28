#!/usr/bin/env bash
# SearXNG Configuration Defaults
# All configuration constants and default values

# Initialize all variables early to prevent unbound variable errors in messages.sh
# These are set with default values and will be overridden by export_config function
SEARXNG_PORT="${SEARXNG_CUSTOM_PORT:-8100}"
SEARXNG_BASE_URL="http://localhost:${SEARXNG_PORT}"
SEARXNG_DATA_DIR="${HOME}/.searxng"
SEARXNG_CONTAINER_NAME="searxng"
SEARXNG_DEFAULT_ENGINES="google,bing,duckduckgo,startpage"
SEARXNG_REQUEST_TIMEOUT="3"
SEARXNG_POOL_MAXSIZE="20"
SEARXNG_RATE_LIMIT="10"

# Function to make variables readonly for production safety
searxng::make_config_readonly() {
    # Only make readonly if not in test environment
    if [[ "${BATS_TEST_NAME:-}" == "" && "${SEARXNG_TEST_MODE:-}" != "yes" ]]; then
        readonly SEARXNG_PORT SEARXNG_BASE_URL SEARXNG_CONTAINER_NAME SEARXNG_IMAGE SEARXNG_DATA_DIR
        readonly SEARXNG_NETWORK_NAME SEARXNG_BIND_ADDRESS
        readonly SEARXNG_SECRET_KEY SEARXNG_ENABLE_PUBLIC_ACCESS
        readonly SEARXNG_DEFAULT_ENGINES SEARXNG_SAFE_SEARCH SEARXNG_AUTOCOMPLETE SEARXNG_DEFAULT_LANG
        readonly SEARXNG_REQUEST_TIMEOUT SEARXNG_MAX_REQUEST_TIMEOUT SEARXNG_POOL_CONNECTIONS SEARXNG_POOL_MAXSIZE
        readonly SEARXNG_LIMITER_ENABLED SEARXNG_RATE_LIMIT SEARXNG_RATE_LIMIT_WINDOW
        readonly SEARXNG_HEALTH_CHECK_INTERVAL SEARXNG_HEALTH_CHECK_MAX_ATTEMPTS SEARXNG_HEALTH_ENDPOINT
        readonly SEARXNG_API_TIMEOUT SEARXNG_ENABLE_METRICS
        readonly SEARXNG_LOG_LEVEL SEARXNG_LOG_LINES
        readonly SEARXNG_ENABLE_REDIS SEARXNG_REDIS_HOST SEARXNG_REDIS_PORT
        readonly SEARXNG_INSTANCE_NAME SEARXNG_CONTACT_URL SEARXNG_DONATION_URL
    fi
}

# Export function to make configuration available
searxng::export_config() {
    # Set all configuration variables (allows for dynamic reconfiguration)
    # SearXNG port configuration
    SEARXNG_PORT="${SEARXNG_CUSTOM_PORT:-8100}"
    SEARXNG_BASE_URL="${SEARXNG_BASE_URL:-http://localhost:${SEARXNG_PORT}}"
    
    # Container configuration
    SEARXNG_CONTAINER_NAME="searxng"
    SEARXNG_IMAGE="${SEARXNG_CUSTOM_IMAGE:-searxng/searxng:latest}"
    SEARXNG_DATA_DIR="${HOME}/.searxng"
    
    # Network configuration
    SEARXNG_NETWORK_NAME="searxng-network"
    SEARXNG_BIND_ADDRESS="${SEARXNG_BIND_ADDRESS:-127.0.0.1}"  # Local only by default
    
    # Security configuration
    SEARXNG_SECRET_KEY="${SEARXNG_SECRET_KEY:-$(openssl rand -hex 32 2>/dev/null || date +%s | sha256sum | head -c 32)}"
    SEARXNG_ENABLE_PUBLIC_ACCESS="${SEARXNG_ENABLE_PUBLIC_ACCESS:-no}"
    
    # Search engine configuration
    SEARXNG_DEFAULT_ENGINES="${SEARXNG_DEFAULT_ENGINES:-google,bing,duckduckgo,startpage}"
    SEARXNG_SAFE_SEARCH="${SEARXNG_SAFE_SEARCH:-1}"
    SEARXNG_AUTOCOMPLETE="${SEARXNG_AUTOCOMPLETE:-}"  # Empty string for no autocomplete
    SEARXNG_DEFAULT_LANG="${SEARXNG_DEFAULT_LANG:-en}"
    
    # Performance configuration - using integers instead of floats for better compatibility
    SEARXNG_REQUEST_TIMEOUT="${SEARXNG_REQUEST_TIMEOUT:-3}"
    SEARXNG_MAX_REQUEST_TIMEOUT="${SEARXNG_MAX_REQUEST_TIMEOUT:-10}"
    SEARXNG_POOL_CONNECTIONS="${SEARXNG_POOL_CONNECTIONS:-100}"
    SEARXNG_POOL_MAXSIZE="${SEARXNG_POOL_MAXSIZE:-20}"
    
    # Rate limiting configuration
    SEARXNG_LIMITER_ENABLED="${SEARXNG_LIMITER_ENABLED:-yes}"
    SEARXNG_RATE_LIMIT="${SEARXNG_RATE_LIMIT:-10}"  # requests per minute
    SEARXNG_RATE_LIMIT_WINDOW="${SEARXNG_RATE_LIMIT_WINDOW:-60}"  # seconds
    
    # Health check configuration
    SEARXNG_HEALTH_CHECK_INTERVAL=5
    SEARXNG_HEALTH_CHECK_MAX_ATTEMPTS=30
    SEARXNG_HEALTH_ENDPOINT="/stats"
    
    # API configuration
    SEARXNG_API_TIMEOUT=30
    SEARXNG_ENABLE_METRICS="${SEARXNG_ENABLE_METRICS:-yes}"
    
    # Log configuration
    SEARXNG_LOG_LEVEL="${SEARXNG_LOG_LEVEL:-INFO}"
    SEARXNG_LOG_LINES=100
    
    # Redis configuration (optional caching)
    SEARXNG_ENABLE_REDIS="${SEARXNG_ENABLE_REDIS:-no}"
    SEARXNG_REDIS_HOST="${SEARXNG_REDIS_HOST:-redis}"
    SEARXNG_REDIS_PORT="${SEARXNG_REDIS_PORT:-6379}"
    
    # Instance configuration - adding proper defaults
    SEARXNG_INSTANCE_NAME="${SEARXNG_INSTANCE_NAME:-Vrooli SearXNG}"
    SEARXNG_CONTACT_URL="${SEARXNG_CONTACT_URL:-https://github.com/Vrooli/Vrooli}"
    SEARXNG_DONATION_URL="${SEARXNG_DONATION_URL:-}"
    
    # Export all configuration variables
    export SEARXNG_PORT SEARXNG_BASE_URL SEARXNG_CONTAINER_NAME SEARXNG_IMAGE SEARXNG_DATA_DIR
    export SEARXNG_NETWORK_NAME SEARXNG_BIND_ADDRESS
    export SEARXNG_SECRET_KEY SEARXNG_ENABLE_PUBLIC_ACCESS
    export SEARXNG_DEFAULT_ENGINES SEARXNG_SAFE_SEARCH SEARXNG_AUTOCOMPLETE SEARXNG_DEFAULT_LANG
    export SEARXNG_REQUEST_TIMEOUT SEARXNG_MAX_REQUEST_TIMEOUT SEARXNG_POOL_CONNECTIONS SEARXNG_POOL_MAXSIZE
    export SEARXNG_LIMITER_ENABLED SEARXNG_RATE_LIMIT SEARXNG_RATE_LIMIT_WINDOW
    export SEARXNG_HEALTH_CHECK_INTERVAL SEARXNG_HEALTH_CHECK_MAX_ATTEMPTS SEARXNG_HEALTH_ENDPOINT
    export SEARXNG_API_TIMEOUT SEARXNG_ENABLE_METRICS
    export SEARXNG_LOG_LEVEL SEARXNG_LOG_LINES
    export SEARXNG_ENABLE_REDIS SEARXNG_REDIS_HOST SEARXNG_REDIS_PORT
    export SEARXNG_INSTANCE_NAME SEARXNG_CONTACT_URL SEARXNG_DONATION_URL
    
    # Make variables readonly for production safety (unless in test mode)
    searxng::make_config_readonly
}