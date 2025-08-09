#!/usr/bin/env bash
# Posix-compliant script to setup the project for native Windows development/production
set -euo pipefail

SETUP_TARGET_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_TARGET_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"

native_win::setup_native_win() {
    log::header "Setting up native Windows development/production..."

    # Check if running on Windows
    if [[ "$(uname)" != *"NT"* ]] && [[ "$(uname)" != *"MINGW"* ]] && [[ "$(uname)" != *"MSYS"* ]] && [[ "$(uname)" != *"CYGWIN"* ]]; then
        log::error "This script must be run on Windows"
        exit 1
    fi

    # Check if nvm-windows is installed
    if ! system::is_command "nvm"; then
        log::info "Installing nvm-windows..."
        # Download and install nvm-windows
        # shellcheck disable=SC2154
        powershell -Command "
            \$nvmUrl = 'https://github.com/coreybutler/nvm-windows/releases/latest/download/nvm-setup.exe';
            \$nvmInstaller = [System.IO.Path]::Combine(\$env:TEMP, 'nvm-setup.exe');
            Invoke-WebRequest -Uri \$nvmUrl -OutFile \$nvmInstaller;
            Start-Process -FilePath \$nvmInstaller -Wait;
        "
        
        log::warning "nvm-windows has been installed. You may need to restart your terminal for it to be available."
    fi

    # Setup pnpm and generate Prisma client (this will install Node if needed)
    # shellcheck disable=SC1091
    source "${SETUP_TARGET_DIR}/../../utils/pnpm_tools.sh"
    pnpm_tools::setup

    log::success "Native Windows setup completed successfully!"
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    native_win::setup_native_win "$@"
fi 