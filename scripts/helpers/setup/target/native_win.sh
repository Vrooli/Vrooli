#!/usr/bin/env bash
# Posix-compliant script to setup the project for native Windows development/production
set -euo pipefail

SETUP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_TARGET_DIR}/../../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_TARGET_DIR}/../../utils/system.sh"

native_win::setup_native_win() {
    log::header "Setting up native Windows development/production..."

    # Check if running on Windows
    if [[ "$(uname)" != *"NT"* ]]; then
        log::error "This script must be run on Windows"
        exit 1
    fi

    # Check if nvm-windows is installed
    if ! system::is_command "nvm"; then
        log::info "Installing nvm-windows..."
        # Download and install nvm-windows
        powershell -Command "
            $nvmUrl = 'https://github.com/coreybutler/nvm-windows/releases/latest/download/nvm-setup.exe';
            $nvmInstaller = [System.IO.Path]::Combine($env:TEMP, 'nvm-setup.exe');
            Invoke-WebRequest -Uri $nvmUrl -OutFile $nvmInstaller;
            Start-Process -FilePath $nvmInstaller -Wait;
        "
    fi

    # Install Node.js LTS version
    log::info "Installing Node.js LTS version..."
    nvm install lts
    nvm use lts

    # Install essential global packages
    log::info "Installing essential global packages..."
    pnpm add -g typescript@latest
    pnpm add -g @types/node
    pnpm add -g ts-node
    pnpm add -g eslint
    pnpm add -g prettier

    # Install project dependencies
    log::info "Installing project dependencies..."
    pnpm install

    # Setup Git hooks
    log::info "Setting up Git hooks..."
    npm run prepare

    log::success "Native Windows setup completed successfully!"
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    native_win::setup_native_win "$@"
fi 