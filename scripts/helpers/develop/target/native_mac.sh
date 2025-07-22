#!/usr/bin/env bash
# Posix-compliant script to setup the project for native Mac development
set -euo pipefail

ORIGINAL_DIR=$(pwd)
DEVELOP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/flow.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/log.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/system.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/var.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/exit_codes.sh"

nativeMac::brew_install() {
    if ! system::is_command "brew"; then
        echo "Homebrew not found, installing..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    else
        echo "Homebrew is already installed."
    fi
}

nativeMac::volta_install() {
    if ! system::is_command "volta"; then
        echo "Volta not found, installing..."
        brew install volta
    else
        echo "Volta is already installed."
    fi
}

nativeMac::node_pnpm_setup() {
    volta install node
    volta install pnpm
}

nativeMac::gnu_tools_install() {
    brew install coreutils findutils
}

nativeMac::docker_compose_infra() {
    docker-compose up -d postgres redis
}

nativeMac::setup_native_mac() {
    log::header "Setting up native Mac development/production..."
    nativeMac::brew_install
    nativeMac::volta_install
    nativeMac::node_pnpm_setup
    nativeMac::gnu_tools_install
}

nativeMac::start_development_native_mac() {
    log::header "🚀 Starting native Mac development environment..."
    cd "$var_ROOT_DIR"

    nativeMac::cleanup() {
        log::info "🔧 Cleaning up development environment at $var_ROOT_DIR..."
        cd "$var_ROOT_DIR"
        docker-compose down
        cd "$ORIGINAL_DIR"
        exit "$EXIT_USER_INTERRUPT"
    }
    if ! flow::is_yes "$DETACHED"; then
        trap nativeMac::cleanup SIGINT SIGTERM
    fi

    log::info "Starting database containers (Postgres and Redis)..."
    docker-compose up -d postgres redis
    
    # Wait for containers to be healthy before continuing
    log::info "Waiting for database containers to be healthy..."
    local max_attempts=30
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if docker ps | grep "postgres" | grep -q "healthy" && docker ps | grep "redis" | grep -q "healthy"; then
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
    source "${DEVELOP_TARGET_DIR}/../../utils/env_urls.sh"
    
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
        # Start each watcher in background using nohup and track PIDs
        local pids=()
        for cmd in "${watchers[@]}"; do
            nohup bash -c "$cmd" > /dev/null 2>&1 &
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
    # Check if we should run setup or development based on arguments
    if [[ "${1:-}" == "--setup" ]]; then
        nativeMac::setup_native_mac "$@"
    else
        nativeMac::start_development_native_mac "$@"
    fi
fi
