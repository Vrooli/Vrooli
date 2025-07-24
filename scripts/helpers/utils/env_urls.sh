#!/usr/bin/env bash
# Utility functions for generating service URLs based on deployment mode
set -euo pipefail

# Get the directory of this script
ENV_URLS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source docker utilities for container commands
# shellcheck disable=SC1091
source "${ENV_URLS_DIR}/docker.sh"

# Generate Redis URL based on deployment mode
env_urls::generate_redis_url() {
    local mode="$1"  # "docker" or "native"
    local redis_password="$2"
    local redis_host="$3"  # "redis" for docker, "127.0.0.1" for native
    local redis_port="${4:-6379}"
    
    # Handle empty password case
    if [ -z "$redis_password" ]; then
        echo "redis://${redis_host}:${redis_port}"
    else
        echo "redis://:${redis_password}@${redis_host}:${redis_port}"
    fi
}

# Generate PostgreSQL URL based on deployment mode
env_urls::generate_db_url() {
    local mode="$1"  # "docker" or "native"
    local db_user="$2"
    local db_password="$3"
    local db_host="$4"  # "postgres" for docker, "127.0.0.1" for native
    local db_port="${5:-5432}"
    local db_name="$6"
    
    echo "postgresql://${db_user}:${db_password}@${db_host}:${db_port}/${db_name}"
}

# Extract environment variables from docker-compose config
env_urls::extract_docker_compose_env() {
    local var_name="$1"
    local service="${2:-}"  # optional service name
    local default_value="${3:-}"  # optional default value
    
    local value=""
    
    # Try to get from docker-compose config
    if [ -n "$service" ]; then
        # Look for service-specific environment variable
        value=$(docker::compose config 2>/dev/null | grep -A 50 "^\\s*${service}:" | grep -E "^\\s*${var_name}:" | head -1 | cut -d: -f2- | xargs || echo "")
    else
        # Look for any occurrence of the variable
        value=$(docker::compose config 2>/dev/null | grep -E "^\\s*${var_name}:" | head -1 | cut -d: -f2- | xargs || echo "")
    fi
    
    # If not found in docker-compose config, try environment
    if [ -z "$value" ]; then
        value="${!var_name:-}"
    fi
    
    # Use default if still empty
    if [ -z "$value" ] && [ -n "$default_value" ]; then
        value="$default_value"
    fi
    
    echo "$value"
}

# Setup environment variables for native mode
env_urls::setup_native_environment() {
    local redis_password db_user db_password db_name redis_port db_port

    # IMPORTANT: Source the .env file first to get all environment variables
    local env_file="${PROJECT_DIR:-/root/Vrooli}/.env"
    if [ -f "$env_file" ]; then
        echo "Loading environment variables from $env_file"
        set -a  # Automatically export all variables
        # shellcheck source=/dev/null
        . "$env_file"
        set +a
    else
        echo "WARNING: Environment file $env_file not found"
    fi
    
    # Extract Redis configuration
    redis_password=$(env_urls::extract_docker_compose_env "REDIS_PASSWORD" "redis" "${REDIS_PASSWORD:-redispassword}")
    
    # Discover actual Redis port from running container
    redis_port=$(docker::run ps --format "{{.Ports}}" | grep "127.0.0.1.*->6379" | head -1 | grep -o "127.0.0.1:[0-9]*" | cut -d: -f2 || echo "")
    if [ -z "$redis_port" ]; then
        # Try alternative format
        redis_port=$(docker::run ps --format "{{.Ports}}" | grep "0.0.0.0.*->6379" | head -1 | grep -o "0.0.0.0:[0-9]*" | cut -d: -f2 || echo "")
    fi
    if [ -z "$redis_port" ]; then
        # Fallback to configured port if discovery fails
        redis_port=$(env_urls::extract_docker_compose_env "PORT_REDIS" "" "${PORT_REDIS:-6379}")
    fi
    
    # Extract PostgreSQL configuration
    db_user=$(env_urls::extract_docker_compose_env "POSTGRES_USER" "postgres" "${DB_USER:-site}")
    db_password=$(env_urls::extract_docker_compose_env "POSTGRES_PASSWORD" "postgres" "${DB_PASSWORD:-databasepassword}")
    db_name=$(env_urls::extract_docker_compose_env "DB_NAME" "" "${DB_NAME:-vrooli}")
    
    # Discover actual PostgreSQL port from running container
    db_port=$(docker::run ps --format "{{.Ports}}" | grep "127.0.0.1.*->5432" | head -1 | grep -o "127.0.0.1:[0-9]*" | cut -d: -f2 || echo "")
    if [ -z "$db_port" ]; then
        # Try alternative format
        db_port=$(docker::run ps --format "{{.Ports}}" | grep "0.0.0.0.*->5432" | head -1 | grep -o "0.0.0.0:[0-9]*" | cut -d: -f2 || echo "")
    fi
    if [ -z "$db_port" ]; then
        # Fallback to configured port if discovery fails
        db_port=$(env_urls::extract_docker_compose_env "PORT_DB" "" "${PORT_DB:-5432}")
    fi
    
    # Also check for DB_USER and DB_PASSWORD as alternatives
    if [ -z "$db_user" ]; then
        db_user="${DB_USER:-site}"
    fi
    if [ -z "$db_password" ]; then
        db_password="${DB_PASSWORD:-databasepassword}"
    fi
    
    # Generate native URLs (using localhost/127.0.0.1)
    local redis_url db_url
    redis_url=$(env_urls::generate_redis_url "native" "$redis_password" "127.0.0.1" "$redis_port")
    db_url=$(env_urls::generate_db_url "native" "$db_user" "$db_password" "127.0.0.1" "$db_port" "$db_name")
    
    # Export for child processes
    export REDIS_URL="$redis_url"
    export DB_URL="$db_url"
    
    # Also export individual components for other uses
    export REDIS_PASSWORD="$redis_password"
    export DB_USER="$db_user"
    export DB_PASSWORD="$db_password"
    export DB_NAME="$db_name"
    export DB_HOST="127.0.0.1"
    export REDIS_HOST="127.0.0.1"
    export PORT_DB="$db_port"
    export PORT_REDIS="$redis_port"
    
    # Export other essential environment variables (already loaded from .env file above)
    # Note: Variables are already loaded and exported from .env, these are fallbacks
    export NODE_ENV="${NODE_ENV:-development}"
    export JWT_PRIV="${JWT_PRIV:-dev-jwt-private-key-placeholder}"
    export JWT_PUB="${JWT_PUB:-dev-jwt-public-key-placeholder}"
    export EXTERNAL_SITE_KEY="${EXTERNAL_SITE_KEY:-abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz}"
    export OPENAI_API_KEY="${OPENAI_API_KEY:-}"
    export ANTHROPIC_API_KEY="${ANTHROPIC_API_KEY:-}"
    export MISTRAL_API_KEY="${MISTRAL_API_KEY:-}"
    export CREATE_MOCK_DATA="${CREATE_MOCK_DATA:-true}"
    export SITE_EMAIL_FROM="${SITE_EMAIL_FROM:-}"
    export VAPID_PUBLIC_KEY="${VAPID_PUBLIC_KEY:-the_public_key_generated_from_generate-vapid-keys}"
    export VAPID_PRIVATE_KEY="${VAPID_PRIVATE_KEY:-the_private_key_generated_from_generate-vapid-keys}"
    export SITE_EMAIL_USERNAME="${SITE_EMAIL_USERNAME:-}"
    export SITE_EMAIL_PASSWORD="${SITE_EMAIL_PASSWORD:-}"
    export SITE_EMAIL_ALIAS="${SITE_EMAIL_ALIAS:-}"
    export SITE_IP="${SITE_IP:-localhost}"
    export STRIPE_SECRET_KEY="${STRIPE_SECRET_KEY:-}"
    export STRIPE_WEBHOOK_SECRET="${STRIPE_WEBHOOK_SECRET:-}"
    export API_URL="${API_URL:-http://localhost:5329/api}"
    export UI_URL="${UI_URL:-http://localhost:3000}"
    export TWILIO_ACCOUNT_SID="${TWILIO_ACCOUNT_SID:-}"
    export TWILIO_AUTH_TOKEN="${TWILIO_AUTH_TOKEN:-}"
    export TWILIO_PHONE_NUMBER="${TWILIO_PHONE_NUMBER:-}"
    export WORKER_ID="${WORKER_ID:-dev-worker-1}"
    export ADMIN_WALLET="${ADMIN_WALLET:-}"
    export ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
    export VALYXA_PASSWORD="${VALYXA_PASSWORD:-}"
    export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-}"
    export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-}"
    export PROJECT_DIR="${PROJECT_DIR:-/root/Vrooli}"
    export VITE_SERVER_LOCATION="${VITE_SERVER_LOCATION:-http://localhost:5329}"
    export SERVER_LOCATION="${SERVER_LOCATION:-http://localhost:5329}"
    export VITE_PORT_API="${VITE_PORT_API:-5329}"
    export PORT_API="${PORT_API:-5329}"
    export PORT_UI="${PORT_UI:-3000}"
    export DB_PULL="${DB_PULL:-false}"
}

# Validate native environment connectivity
env_urls::validate_native_environment() {
    local all_good=true
    
    # Check that containers are running
    if ! docker::run ps | grep "redis" | grep -q "healthy"; then
        echo "ERROR: Redis container not running or unhealthy" >&2
        all_good=false
    fi
    
    if ! docker::run ps | grep "postgres" | grep -q "healthy"; then
        echo "ERROR: PostgreSQL container not running or unhealthy" >&2
        all_good=false
    fi
    
    # Test Redis connectivity (if redis-cli is available)
    if command -v redis-cli >/dev/null 2>&1; then
        if [ -n "${REDIS_PASSWORD}" ]; then
            if ! redis-cli -h 127.0.0.1 -p "${PORT_REDIS:-6379}" -a "${REDIS_PASSWORD}" ping >/dev/null 2>&1; then
                echo "WARNING: Cannot connect to Redis with generated URL" >&2
                all_good=false
            fi
        else
            if ! redis-cli -h 127.0.0.1 -p "${PORT_REDIS:-6379}" ping >/dev/null 2>&1; then
                echo "WARNING: Cannot connect to Redis with generated URL" >&2
                all_good=false
            fi
        fi
    fi
    
    # Test PostgreSQL connectivity (if pg_isready is available)
    if command -v pg_isready >/dev/null 2>&1; then
        if ! pg_isready -h 127.0.0.1 -p "${PORT_DB:-5432}" -U "${DB_USER}" >/dev/null 2>&1; then
            echo "WARNING: Cannot connect to PostgreSQL with generated URL" >&2
            all_good=false
        fi
    fi
    
    if [ "$all_good" = true ]; then
        echo "All service connectivity tests passed"
        return 0
    else
        echo "Some service connectivity tests failed" >&2
        return 1
    fi
}

# Display environment configuration (for debugging)
env_urls::display_native_config() {
    echo "Native mode environment configuration:"
    echo "  REDIS_URL=${REDIS_URL}"
    echo "  DB_URL=${DB_URL}"
    echo "  NODE_ENV=${NODE_ENV}"
    echo "  API_URL=${API_URL}"
    echo "  UI_URL=${UI_URL}"
    echo "  PORT_API=${PORT_API}"
    echo "  PORT_UI=${PORT_UI}"
}