#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

vnc_install() {
    log::header "Installing VNC resource (x11vnc + websockify)"

    # Check if already installed
    if command -v x11vnc &>/dev/null && command -v websockify &>/dev/null; then
        log::success "x11vnc and websockify already installed"
        return 0
    fi

    sudo apt-get update -qq
    sudo apt-get install -y -qq x11vnc websockify

    # Verify
    if ! command -v x11vnc &>/dev/null; then
        log::error "x11vnc installation failed"
        return 1
    fi
    if ! command -v websockify &>/dev/null; then
        log::error "websockify installation failed"
        return 1
    fi

    log::success "VNC resource installed successfully"
}

# Allow direct invocation
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    vnc_install "$@"
fi
