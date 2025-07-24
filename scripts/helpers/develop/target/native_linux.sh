#!/usr/bin/env bash
set -euo pipefail

ORIGINAL_DIR=$(pwd)
DEVELOP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/flow.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/log.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/var.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/exit_codes.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/docker.sh"

nativeLinux::start_development_native_linux() {
    log::header "🚀 Starting native Linux development environment..."
    cd "$var_ROOT_DIR"

    nativeLinux::cleanup() {
        log::info "🔧 Cleaning up development environment at $var_ROOT_DIR..."
        cd "$var_ROOT_DIR"
        docker::compose down
        cd "$ORIGINAL_DIR"
        exit "$EXIT_USER_INTERRUPT"
    }
    if ! flow::is_yes "$DETACHED"; then
        trap nativeLinux::cleanup SIGINT SIGTERM
    fi

    log::info "Starting database containers (Postgres and Redis)..."
    docker::compose up -d postgres redis
    
    # Wait for containers to be healthy before continuing
    log::info "Waiting for database containers to be healthy..."
    local max_attempts=30
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if docker::run ps | grep "postgres" | grep -q "healthy" && docker::run ps | grep "redis" | grep -q "healthy"; then
            log::info "Database containers are healthy!"
            break
        fi
        attempt=$((attempt + 1))
        if [ $attempt -eq $max_attempts ]; then
            log::error "Database containers failed to become healthy after $max_attempts attempts"
            return 1
        fi
        sleep 2
    done
    
    # Additional delay to ensure Redis is fully ready to accept connections
    log::info "Giving Redis extra time to fully initialize..."
    sleep 10
    
    # Source environment URL utilities
    # shellcheck disable=SC1091
    source "${DEVELOP_TARGET_DIR}/../../utils/env_urls.sh"
    
    # Configure environment variables for native mode
    log::info "Configuring environment variables for native mode..."
    env_urls::setup_native_environment
    
    # Validate connectivity (but don't fail if validation tools aren't available)
    if ! env_urls::validate_native_environment; then
        log::warning "Some connectivity tests failed, but continuing anyway..."
    fi
    
    # Test database connectivity with a simple connection test
    log::info "Testing PostgreSQL connectivity with Prisma..."
    cd "$var_ROOT_DIR/packages/server"
    if DB_URL="${DB_URL}" npx prisma db execute --stdin <<< "SELECT 1;" >/dev/null 2>&1; then
        log::info "PostgreSQL connectivity test successful"
    else
        log::error "PostgreSQL connectivity test failed. DB_URL: ${DB_URL}"
        log::error "This suggests the database is not accessible. Check containers and network."
    fi
    cd "$var_ROOT_DIR"

    log::info "Starting watchers and development servers (server, jobs, UI)..."
    # Export all environment variables for child processes
    export REDIS_URL="${REDIS_URL}"
    export DB_URL="${DB_URL}"
    export NODE_ENV="${NODE_ENV}"
    export PORT_API="${PORT_API}"
    export PORT_UI="${PORT_UI}"
    export VITE_SERVER_LOCATION="${SERVER_LOCATION}"
    
    # Define development commands using the existing start.sh scripts
    # Add staggered startup to prevent database connection race conditions
    local watchers=(
        # Server starts first (handles migrations)
        "cd packages/server && PROJECT_DIR=$var_ROOT_DIR NODE_ENV=development DB_URL=${DB_URL} REDIS_URL=${REDIS_URL} npm_package_name=@vrooli/server bash ../../scripts/package/server/start.sh"
        # Jobs service starts after a delay to avoid DB connection race
        "sleep 10 && cd packages/jobs && PROJECT_DIR=$var_ROOT_DIR NODE_ENV=development DB_URL=${DB_URL} REDIS_URL=${REDIS_URL} npm_package_name=@vrooli/jobs bash ../../scripts/package/jobs/start.sh"
        # UI development server (starts immediately, no DB dependency)
        "pnpm --filter @vrooli/ui run start-development -- --port ${PORT_UI:-3000}"
    )
    if flow::is_yes "$DETACHED"; then
        log::info "Detached mode: launching individual watchers in background"
        # Start each watcher in background using nohup and track PIDs
        local pids=()
        for cmd in "${watchers[@]}"; do
            # Export variables for the subshell
            REDIS_URL="${REDIS_URL}" \
            DB_URL="${DB_URL}" \
            NODE_ENV="${NODE_ENV}" \
            PORT_API="${PORT_API}" \
            PORT_UI="${PORT_UI}" \
            ADMIN_WALLET="${ADMIN_WALLET}" \
            ADMIN_PASSWORD="${ADMIN_PASSWORD}" \
            SITE_EMAIL_USERNAME="${SITE_EMAIL_USERNAME}" \
            VALYXA_PASSWORD="${VALYXA_PASSWORD}" \
            nohup bash -c "$cmd" > /dev/null 2>&1 &
            pids+=("$!")
        done
        log::info "Launched watchers (PIDs: ${pids[*]})"
        return 0
    else
        # Foreground mode: use concurrently to run all watchers together
        # Pass environment variables explicitly to ensure they're available
        REDIS_URL="${REDIS_URL}" \
        DB_URL="${DB_URL}" \
        NODE_ENV="${NODE_ENV}" \
        PORT_API="${PORT_API}" \
        PORT_UI="${PORT_UI}" \
        ADMIN_WALLET="${ADMIN_WALLET}" \
        ADMIN_PASSWORD="${ADMIN_PASSWORD}" \
        SITE_EMAIL_USERNAME="${SITE_EMAIL_USERNAME}" \
        VALYXA_PASSWORD="${VALYXA_PASSWORD}" \
        pnpm exec concurrently \
            --names "SERVER,JOBS,UI" \
            -c "yellow,blue,green" \
            "${watchers[@]}"
    fi
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    nativeLinux::start_development_native_linux "$@"
fi
