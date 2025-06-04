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

dockerOnly::start_development_docker_only() {
    local detached=${DETACHED:-No}

    log::header "🚀 Starting Docker only development environment..."
    cd "$var_ROOT_DIR"

    dockerOnly::cleanup() {
        log::info "🔧 Cleaning up development environment at $var_ROOT_DIR..."
        cd "$var_ROOT_DIR"
        docker-compose down
        cd "$ORIGINAL_DIR"
        exit 0
    }
    if ! flow::is_yes "$detached"; then
        trap dockerOnly::cleanup SIGINT SIGTERM
    fi
    log::info "Starting all services in detached mode (Postgres, Redis, server, jobs, UI)..."
    if flow::is_yes "$detached"; then
        docker-compose up -d
    else
        docker-compose up
    fi

    log::success "✅ Docker only development environment started successfully."
    log::info "You can view logs with 'docker-compose logs -f'."
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    dockerOnly::start_development_docker_only "$@"
fi
