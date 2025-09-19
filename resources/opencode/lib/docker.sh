#!/bin/bash
# OpenCode resource does not run containers; keep helpers for compatibility.

source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::docker::start() {
    log::info "OpenCode CLI does not require background services."
    log::info "Use \"resource-opencode run --help\" or \"resource-opencode run run 'Hello'\" to issue requests."
}

opencode::docker::stop() {
    log::info "No background process to stop for OpenCode CLI."
}

opencode::docker::restart() {
    log::info "Restart is a no-op; rerun your CLI command instead."
}

opencode::docker::logs() {
    log::info "Logs are written to ${OPENCODE_LOG_DIR} when commands are executed."
    if [[ -d "${OPENCODE_LOG_DIR}" ]]; then
        find "${OPENCODE_LOG_DIR}" -type f -maxdepth 1 -printf ' - %P\n'
    else
        log::info "Log directory not created yet."
    fi
}
