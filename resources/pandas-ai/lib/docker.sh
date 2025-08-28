#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
PANDAS_AI_LIB_DIR="${APP_ROOT}/resources/pandas-ai/lib"

# Source dependencies
source "${PANDAS_AI_LIB_DIR}/common.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${PANDAS_AI_LIB_DIR}/status.sh"
source "${PANDAS_AI_LIB_DIR}/lifecycle.sh"

# Start pandas-ai service
pandas_ai::docker::start() {
    pandas_ai::start "$@"
}

# Stop pandas-ai service
pandas_ai::docker::stop() {
    pandas_ai::stop "$@"
}

# Restart pandas-ai service
pandas_ai::docker::restart() {
    log::info "Restarting Pandas AI service"
    pandas_ai::stop
    sleep 2
    pandas_ai::start
}

# Show logs
pandas_ai::docker::logs() {
    local follow="${1:-false}"
    
    if [[ ! -f "${PANDAS_AI_LOG_FILE}" ]]; then
        log::warning "Log file not found: ${PANDAS_AI_LOG_FILE}"
        return 1
    fi
    
    if [[ "${follow}" == "--follow" || "${follow}" == "-f" ]]; then
        log::info "Following Pandas AI logs (Ctrl+C to stop):"
        tail -f "${PANDAS_AI_LOG_FILE}"
    else
        log::info "Pandas AI logs:"
        tail -50 "${PANDAS_AI_LOG_FILE}"
    fi
}

# Export functions for use by CLI
export -f pandas_ai::docker::start
export -f pandas_ai::docker::stop
export -f pandas_ai::docker::restart
export -f pandas_ai::docker::logs