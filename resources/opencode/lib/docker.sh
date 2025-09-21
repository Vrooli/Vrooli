#!/bin/bash
# OpenCode resource does not run containers; keep helpers for compatibility.

source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::docker::start() {
    opencode::ensure_dirs
    opencode::ensure_cli || return 1
    opencode::ensure_config
    opencode::load_secrets || true
    opencode::export_runtime_env

    if opencode::server::is_running; then
        log::info "OpenCode server already running on $(opencode::server::base_url)"
        return 0
    fi

    mkdir -p "${OPENCODE_LOG_DIR}"
    local log_file="${OPENCODE_SERVER_LOG_FILE}"
    : >"${log_file}"

    log::info "Starting OpenCode server (${OPENCODE_SERVER_HOST}:${OPENCODE_SERVER_PORT})"

    opencode::run_cli serve --hostname "${OPENCODE_SERVER_HOST}" --port "${OPENCODE_SERVER_PORT}" --print-logs >>"${log_file}" 2>&1 &
    local pid=$!
    disown "${pid}" 2>/dev/null || true
    opencode::server::record_pid "${pid}"

    if opencode::server::wait_until_ready 20; then
        log::success "OpenCode server is ready at $(opencode::server::base_url)"
        return 0
    fi

    log::error "OpenCode server failed to start"
    kill "${pid}" >/dev/null 2>&1 || true
    opencode::server::clear_pid
    return 1
}

opencode::docker::stop() {
    if ! opencode::server::is_running; then
        log::info "OpenCode server is not running"
        return 0
    fi

    local pid
    pid=$(opencode::server::pid)
    if [[ -z "${pid}" ]]; then
        log::warning "Could not determine OpenCode server PID"
        return 1
    fi

    log::info "Stopping OpenCode server (pid ${pid})"
    if kill "${pid}" >/dev/null 2>&1; then
        sleep 1
        if kill -0 "${pid}" >/dev/null 2>&1; then
            log::warning "OpenCode server may still be shutting down (pid ${pid})"
            return 1
        fi
        log::success "OpenCode server stopped"
        opencode::server::clear_pid
        return 0
    fi

    log::warning "Failed to signal OpenCode server"
    return 1
}

opencode::docker::restart() {
    opencode::docker::stop || true
    sleep 1
    opencode::docker::start
}

opencode::docker::logs() {
    if [[ -f "${OPENCODE_SERVER_LOG_FILE}" ]]; then
        cat "${OPENCODE_SERVER_LOG_FILE}"
    else
        log::info "No server logs yet. Start the server first."
    fi
}
