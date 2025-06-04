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

nativeLinux::start_development_native_linux() {
    log::header "🚀 Starting native Linux development environment..."
    cd "$var_ROOT_DIR"

    nativeLinux::cleanup() {
        log::info "🔧 Cleaning up development environment at $var_ROOT_DIR..."
        cd "$var_ROOT_DIR"
        docker-compose down
        cd "$ORIGINAL_DIR"
        exit 0
    }
    if ! flow::is_yes "$DETACHED"; then
        trap nativeLinux::cleanup SIGINT SIGTERM
    fi

    log::info "Starting database containers (Postgres and Redis)..."
    docker-compose up -d postgres redis

    log::info "Starting watchers and development servers (server, jobs, UI)..."
    # Define development watcher commands
    local watchers=(
        "pnpm exec tsc -b packages/server --watch --preserveWatchOutput"
        "pnpm exec tsc -b packages/jobs --watch --preserveWatchOutput"
        "pnpm exec node --watch packages/server/dist/index.js"
        "pnpm exec node --watch packages/jobs/dist/index.js"
        "pnpm --filter @vrooli/ui run dev -- --port 3000"
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
            --names "TSC-SVR,TSC-JOB,NODE-SVR,NODE-JOB,UI" \
            -c "yellow,blue,magenta,cyan,green" \
            --kill-others-on-fail \
            "${watchers[@]}"
    fi
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    nativeLinux::start_development_native_linux "$@"
fi
