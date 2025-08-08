#!/usr/bin/env bash
set -euo pipefail

ORIGINAL_DIR=$(pwd)
APP_LIFECYCLE_DEVELOP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEVELOP_TARGET_DIR}/../../../../lib/utils/var.sh"

# Now use the variables for cleaner paths
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/docker.sh"

nativeLinux::start_development_native_linux() {
    log::header "🚀 Starting native Linux development environment..."
    cd "$var_ROOT_DIR"

    nativeLinux::cleanup() {
        log::info "🔧 Cleaning up development environment at $var_ROOT_DIR..."
        
        # Temporarily disable unbound variable checking during cleanup
        # This prevents "ui_pid: unbound variable" errors from external scripts
        set +u
        
        # Use instance manager for cleanup if available
        if command -v instance::shutdown_target >/dev/null 2>&1; then
            instance::shutdown_target "native-linux" 2>&1 | grep -v "unbound variable" || true
        else
            # Fallback to traditional cleanup
            cd "$var_ROOT_DIR"
            docker::compose down 2>&1 | grep -v "unbound variable" || true
        fi
        
        # Re-enable strict mode
        set -u
        
        cd "$ORIGINAL_DIR"
        exit "$EXIT_USER_INTERRUPT"
    }
    if ! flow::is_yes "$DETACHED"; then
        trap nativeLinux::cleanup SIGINT SIGTERM
    fi

    # Ensure environment variables are loaded for docker-compose
    # The main setup script should have already called env::load_secrets()
    # but we need to make sure the environment is available in this process
    if [[ -f "${var_APP_UTILS_DIR}/env.sh" ]]; then
        # shellcheck disable=SC1091
        source "${var_APP_UTILS_DIR}/env.sh"
        env::load_env_file
    fi
    
    # Process service.json for runtime template resolution if inheritance is used
    local service_json="${var_SERVICE_JSON_FILE}"
    local service_config_util="${var_APP_UTILS_DIR}/service_config.sh"
    
    if [[ -f "$service_json" ]] && [[ -f "$service_config_util" ]]; then
        # shellcheck disable=SC1090
        source "$service_config_util"
        
        # Check if service.json has inheritance - if so, export resolved resource URLs
        if service_config::has_inheritance "$service_json"; then
            log::info "Resolving service configuration templates at runtime..."
            service_config::export_resource_urls "$service_json"
        fi
    fi
    
    log::info "Starting database containers (Postgres and Redis)..."
    docker::compose up -d postgres redis
    
    # Wait for containers to be healthy before continuing
    log::info "Waiting for database containers to be healthy..."
    local max_attempts=30
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        # Check health status more reliably
        # Get the full status field by extracting everything after the container name
        local postgres_status=$(docker::run ps --format "table {{.Names}}\t{{.Status}}" | grep -E "^postgres" | sed 's/^postgres[[:space:]]*//')
        local redis_status=$(docker::run ps --format "table {{.Names}}\t{{.Status}}" | grep -E "^redis" | sed 's/^redis[[:space:]]*//')
        
        # Debug output
        log::debug "Postgres status: $postgres_status"
        log::debug "Redis status: $redis_status"
        
        # Check if both containers are healthy (status contains "healthy")
        if [[ "$postgres_status" == *"healthy"* ]] && [[ "$redis_status" == *"healthy"* ]]; then
            log::info "Database containers are healthy!"
            break
        fi
        
        # Check for unhealthy or restarting containers
        if [[ "$postgres_status" == *"unhealthy"* ]] || [[ "$redis_status" == *"unhealthy"* ]]; then
            log::warning "One or more containers are unhealthy. Postgres: $postgres_status, Redis: $redis_status"
        fi
        
        if [[ "$postgres_status" == *"Restarting"* ]] || [[ "$redis_status" == *"Restarting"* ]]; then
            log::warning "One or more containers are restarting. This may indicate a configuration issue."
        fi
        
        attempt=$((attempt + 1))
        if [ $attempt -eq $max_attempts ]; then
            log::error "Database containers failed to become healthy after $max_attempts attempts"
            log::error "Final status - Postgres: $postgres_status, Redis: $redis_status"
            log::info "You can check container logs with: docker logs postgres"
            log::info "And: docker logs redis"
            return 1
        fi
        sleep 2
    done
    
    # Additional delay to ensure Redis is fully ready to accept connections
    log::info "Giving Redis extra time to fully initialize..."
    sleep 10
    
    # Source environment URL utilities
    # shellcheck disable=SC1091
    source "${var_APP_UTILS_DIR}/env_urls.sh"
    
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

    # Auto-convert enabled scenarios to standalone apps
    log::info "Auto-converting enabled scenarios to apps..."
    if [[ -f "${var_ROOT_DIR}/scripts/scenarios/auto-converter.sh" ]]; then
        # Run with verbose output if in debug mode
        local converter_opts=""
        if [[ "${DEBUG:-}" == "true" ]]; then
            converter_opts="--verbose"
        fi
        
        if "${var_ROOT_DIR}/scripts/scenarios/auto-converter.sh" $converter_opts; then
            log::success "Scenario auto-conversion completed"
        else
            log::warning "Some scenarios failed to convert (check logs above)"
        fi
    else
        log::warning "Scenario auto-converter not found, skipping..."
    fi

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
        "cd packages/server && PROJECT_DIR=$var_ROOT_DIR NODE_ENV=development DB_URL=${DB_URL} REDIS_URL=${REDIS_URL} npm_package_name=@vrooli/server bash ../../scripts/app/package/server/start.sh"
        # Jobs service starts after a delay to avoid DB connection race
        "sleep 10 && cd packages/jobs && PROJECT_DIR=$var_ROOT_DIR NODE_ENV=development DB_URL=${DB_URL} REDIS_URL=${REDIS_URL} npm_package_name=@vrooli/jobs bash ../../scripts/app/package/jobs/start.sh"
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
