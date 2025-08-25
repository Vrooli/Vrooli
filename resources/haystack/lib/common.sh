#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HAYSTACK_LIB_DIR="${APP_ROOT}/resources/haystack/lib"

# Source the shared var utility FIRST
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Set up paths
HAYSTACK_BASE_DIR="${HAYSTACK_LIB_DIR}/.."
HAYSTACK_CONFIG_DIR="${HAYSTACK_BASE_DIR}/config"
HAYSTACK_DATA_DIR="${HOME}/.vrooli/haystack"
HAYSTACK_VENV_DIR="${HAYSTACK_DATA_DIR}/venv"
HAYSTACK_LOG_DIR="${HAYSTACK_DATA_DIR}/logs"
HAYSTACK_LOG_FILE="${HAYSTACK_LOG_DIR}/haystack.log"
HAYSTACK_PID_FILE="${HAYSTACK_DATA_DIR}/haystack.pid"
HAYSTACK_SCRIPTS_DIR="${HAYSTACK_DATA_DIR}/scripts"

# Source port registry
source "${APP_ROOT}/scripts/resources/port_registry.sh"

# Get port for Haystack
haystack::get_port() {
    ports::get_resource_port "haystack"
}

# Check if Haystack is installed
haystack::is_installed() {
    [[ -d "${HAYSTACK_VENV_DIR}" ]] && [[ -f "${HAYSTACK_VENV_DIR}/bin/pip" ]]
}

# Check if Haystack is running
haystack::is_running() {
    if [[ -f "${HAYSTACK_PID_FILE}" ]]; then
        local pid
        pid=$(cat "${HAYSTACK_PID_FILE}")
        if kill -0 "${pid}" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}