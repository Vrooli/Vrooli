#!/usr/bin/env bash
# Posix-compliant script to setup the project for native Mac development/production
set -euo pipefail

SETUP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_TARGET_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_APP_UTILS_DIR}/docker.sh"

native_mac::brew_install() {
    if ! system::is_command "brew"; then
        log::info "Homebrew not found, installing..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    else
        log::info "Homebrew is already installed."
    fi
}

native_mac::gnu_tools_install() {
    brew install coreutils findutils
}

native_mac::docker_compose_infra() {
    docker::compose up -d
}

native_mac::setup_native_mac() {
    log::header "Setting up native Mac development/production..."
    native_mac::brew_install
    # Volta, Node, and pnpm are now handled by pnpm_tools::setup
    # which calls nodejs::check_and_install
    native_mac::gnu_tools_install
    
    # Setup pnpm and generate Prisma client (this will install Node/Volta if needed)
    # shellcheck disable=SC1091
    source "${SETUP_TARGET_DIR}/../../../../lib/utils/pnpm_tools.sh"
    pnpm_tools::setup
    
    native_mac::docker_compose_infra
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    native_mac::setup_native_mac "$@"
fi 