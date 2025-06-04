#!/usr/bin/env bash
# Posix-compliant script to setup the project for native Mac development
set -euo pipefail

DEVELOP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/log.sh"
# shellcheck disable=SC1091
source "${DEVELOP_TARGET_DIR}/../../utils/system.sh"

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
    docker-compose up -d
}

nativeMac::setup_native_mac() {
    log::header "Setting up native Mac development/production..."
    nativeMac::brew_install
    nativeMac::volta_install
    nativeMac::node_pnpm_setup
    nativeMac::gnu_tools_install
    nativeMac::docker_compose_infra
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    nativeMac::setup_native_mac "$@"
fi
