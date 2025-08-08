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
source "${var_APP_UTILS_DIR}/env.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/docker.sh"

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
    
    # Process service.json for runtime template resolution if inheritance is used
    local service_json="${var_ROOT_DIR}/.vrooli/service.json"
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

    dockerOnly::cleanup() {
        log::info "🔧 Cleaning up development environment at $var_ROOT_DIR..."
        
        # Use instance manager for cleanup if available
        if command -v instance::shutdown_target >/dev/null 2>&1; then
            instance::shutdown_target "docker"
        else
            # Fallback to traditional cleanup
            cd "$var_ROOT_DIR"
            docker::compose down
        fi
        
        cd "$ORIGINAL_DIR"
        exit "$EXIT_USER_INTERRUPT"
    }
    if ! flow::is_yes "$detached"; then
        trap dockerOnly::cleanup SIGINT SIGTERM
    fi
    log::info "Starting all services in detached mode (Postgres, Redis, server, jobs, UI)..."
    
    # Docker Compose needs to see the environment variables
    # Load environment file properly to ensure all variables are available
    env::load_env_file
    
    if flow::is_yes "$detached"; then
        docker::compose up -d
    else
        docker::compose up
    fi

    log::success "✅ Docker only development environment started successfully."
    log::info "You can view logs with 'docker::compose logs -f' from this script's context."
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    dockerOnly::start_development_docker_only "$@"
fi
