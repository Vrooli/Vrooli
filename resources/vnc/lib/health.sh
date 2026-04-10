#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

vnc_health() {
    local healthy=true

    if command -v x11vnc &>/dev/null; then
        log::success "x11vnc is available"
    else
        log::error "x11vnc is NOT installed"
        healthy=false
    fi

    if command -v websockify &>/dev/null; then
        log::success "websockify is available"
    else
        log::error "websockify is NOT installed"
        healthy=false
    fi

    $healthy
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    vnc_health "$@"
fi
