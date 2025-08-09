#!/usr/bin/env bash
# Posix-compliant script for native Windows development
set -euo pipefail

ORIGINAL_DIR=$(pwd)
APP_LIFECYCLE_DEVELOP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEVELOP_TARGET_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/docker.sh"

nativeWin::start_development_native_win() {
    log::header "🚀 Starting native Windows development environment..."
    cd "$var_ROOT_DIR"

    # Note: Windows detection adjusted for WSL/Git Bash/Cygwin
    # This script can run on Windows through WSL, Git Bash, or native Windows with bash

    nativeWin::cleanup() {
        log::info "🔧 Cleaning up development environment at $var_ROOT_DIR..."
        cd "$var_ROOT_DIR"
        docker::compose down
        cd "$ORIGINAL_DIR"
        exit "$EXIT_USER_INTERRUPT"
    }
    if ! flow::is_yes "$DETACHED"; then
        trap nativeWin::cleanup SIGINT SIGTERM
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

    log::info "Starting watchers and development servers (server, jobs, UI)..."
    # Export all environment variables for child processes
    export REDIS_URL="${REDIS_URL}"
    export DB_URL="${DB_URL}"
    export NODE_ENV="${NODE_ENV}"
    export PORT_API="${PORT_API}"
    export PORT_UI="${PORT_UI}"
    
    # Define development commands using the existing start.sh scripts
    local watchers=(
        # Server uses start.sh which handles build, migrations, and starts nodemon
        "cd packages/server && PROJECT_DIR=$var_ROOT_DIR NODE_ENV=development DB_URL=${DB_URL} REDIS_URL=${REDIS_URL} npm_package_name=@vrooli/server bash ../../scripts/package/server/start.sh"
        # Jobs service
        "cd packages/jobs && PROJECT_DIR=$var_ROOT_DIR NODE_ENV=development DB_URL=${DB_URL} REDIS_URL=${REDIS_URL} npm_package_name=@vrooli/jobs bash ../../scripts/package/jobs/start.sh"
        # UI development server
        "pnpm --filter @vrooli/ui run start-development -- --port ${PORT_UI:-3000}"
    )
    if flow::is_yes "$DETACHED"; then
        log::info "Detached mode: launching individual watchers in background"
        # Start each watcher in background and track PIDs
        local pids=()
        for cmd in "${watchers[@]}"; do
            # Windows-compatible background execution
            bash -c "$cmd" > /dev/null 2>&1 &
            pids+=("$!")
        done
        log::info "Launched watchers (PIDs: ${pids[*]})"
        return 0
    else
        # Foreground mode: use concurrently to run all watchers together
        pnpm exec concurrently \
            --names "SWC-SVR,SWC-JOB,ASSET-SVR,ASSET-JOB,NODE-SVR,NODE-JOB,UI" \
            -c "yellow,blue,gray,gray,magenta,cyan,green" \
            "${watchers[@]}"
    fi
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    nativeWin::start_development_native_win "$@"
fi
