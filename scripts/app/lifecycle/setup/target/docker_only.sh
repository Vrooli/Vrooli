#!/usr/bin/env bash
# Posix-compliant script to setup the project for Docker only development/production
set -euo pipefail

SETUP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${SETUP_TARGET_DIR}/../../../../lib/utils/var.sh"

# Now use the variables for cleaner paths
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/pnpm_tools.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/docker.sh"

docker_only::setup_docker_only() {
    log::header "Setting up Docker only development/production..."

    pnpm_tools::setup

    log::info "Building Docker images for all services..."
    docker::compose build

    log::success "✅ Docker images built successfully."
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    docker_only::setup_docker_only "$@"
fi 