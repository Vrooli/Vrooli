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
source "${DEVELOP_TARGET_DIR}/../../utils/env.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/exit_codes.sh"

dockerOnly::start_development_docker_only() {
    local detached=${DETACHED:-No}

    log::header "🚀 Starting Docker only development environment..."
    cd "$var_ROOT_DIR"

    # Ensure environment variables are loaded for docker-compose
    # The main setup script should have already called env::load_secrets() and env::construct_derived_secrets()
    # but let's make sure the derived secrets are available in this subprocess
    if [[ -z "${DB_URL:-}" || -z "${REDIS_URL:-}" ]]; then
        log::info "Constructing derived secrets for docker-compose..."
        env::construct_derived_secrets
    fi

    dockerOnly::cleanup() {
        log::info "🔧 Cleaning up development environment at $var_ROOT_DIR..."
        cd "$var_ROOT_DIR"
        docker-compose down
        cd "$ORIGINAL_DIR"
        exit "$EXIT_USER_INTERRUPT"
    }
    if ! flow::is_yes "$detached"; then
        trap dockerOnly::cleanup SIGINT SIGTERM
    fi
    log::info "Starting all services in detached mode (Postgres, Redis, server, jobs, UI)..."
    
    # Docker Compose needs to see the environment variables
    # Export all variables from the environment file for docker-compose
    export $(grep -v '^#' "$var_ENV_DEV_FILE" | grep -v '^$' | xargs)
    
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
