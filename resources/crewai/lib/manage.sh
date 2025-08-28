#!/usr/bin/env bash

# ⚠️ DEPRECATION NOTICE: This script is deprecated as of v2.0 (January 2025)
# Please use cli.sh instead: resource-crewai <command>
# This file will be removed in v3.0 (target: December 2025)
#
# Migration guide:
#   OLD: ./manage.sh --action <command>
#   NEW: ./cli.sh <command>  OR  resource-crewai <command>

# Show deprecation warning
if [[ "${VROOLI_SUPPRESS_DEPRECATION:-}" != "true" ]]; then
    echo "⚠️  WARNING: manage.sh is deprecated. Please use cli.sh instead." >&2
    echo "   This script will be removed in v3.0 (December 2025)" >&2
    echo "   To suppress this warning: export VROOLI_SUPPRESS_DEPRECATION=true" >&2
    echo "" >&2
fi

# CrewAI Management Functions

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"

# Helper functions for uninstall and restart
crewai::install::uninstall() {
    log::info "Uninstalling CrewAI..."
    stop_crewai
    if [[ -d "${CREWAI_DATA_DIR}" ]]; then
        rm -rf "${CREWAI_DATA_DIR}"
        log::success "CrewAI data directory removed"
    fi
    log::success "CrewAI uninstalled"
}

crewai::manage::restart() {
    log::info "Restarting CrewAI..."
    stop_crewai
    sleep 1
    start_crewai
}

crewai::status::check() {
    if is_running; then
        log::success "CrewAI is running (port ${CREWAI_PORT})"
        return 0
    else
        log::error "CrewAI is not running"
        return 1
    fi
}

crewai::logs() {
    if [[ -f "${CREWAI_LOG_FILE}" ]]; then
        tail -f "${CREWAI_LOG_FILE}"
    else
        log::error "No log file found at ${CREWAI_LOG_FILE}"
        return 1
    fi
}