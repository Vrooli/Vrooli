#!/bin/bash
# OpenCode status helpers

source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::status() {
    opencode::ensure_dirs
    opencode::load_secrets || true

    if ! opencode::ensure_cli; then
        return 1
    fi

    opencode::ensure_config
    opencode::export_runtime_env

    local version="unknown"
    if version_output=$("${OPENCODE_BIN}" --version 2>/dev/null); then
        version="${version_output}" 
    fi

    local model="(not configured)"
    local small_model="(not configured)"
    if [[ -f "${OPENCODE_CONFIG_FILE}" ]]; then
        if command -v jq &>/dev/null; then
            model=$(jq -r '.model // "(not configured)"' "${OPENCODE_CONFIG_FILE}" 2>/dev/null)
            small_model=$(jq -r '.small_model // "(not configured)"' "${OPENCODE_CONFIG_FILE}" 2>/dev/null)
        else
            model=$(grep '"model"' "${OPENCODE_CONFIG_FILE}" | head -n1 | sed -E 's/.*"model"[[:space:]]*:[[:space:]]*"([^"]*)".*/\1/' || echo "(not configured)")
            small_model=$(grep '"small_model"' "${OPENCODE_CONFIG_FILE}" | head -n1 | sed -E 's/.*"small_model"[[:space:]]*:[[:space:]]*"([^"]*)".*/\1/' || echo "(not configured)")
        fi
    fi

    local has_openrouter="false"
    local has_cloudflare="false"
    [[ -n "${OPENROUTER_API_KEY:-}" ]] && has_openrouter="true"
    [[ -n "${CLOUDFLARE_API_TOKEN:-}" ]] && has_cloudflare="true"

    local server_url
    server_url="$(opencode::server::base_url)"
    local server_status
    if opencode::server::is_running; then
        server_status="running (${server_url})"
    else
        server_status="stopped (${server_url})"
    fi

    echo "OpenCode CLI Status"
    echo "===================="
    echo "Binary Path   : ${OPENCODE_BIN}"
    echo "Version       : ${version}"
    echo "Config Path   : ${OPENCODE_CONFIG_FILE}"
    echo "Model         : ${model}"
    echo "Small Model   : ${small_model}"
    echo "Server        : ${server_status}"
    echo "Secrets       : { OPENROUTER_API_KEY: ${has_openrouter}, CLOUDFLARE_API_TOKEN: ${has_cloudflare} }"
    echo "Auth Storage  : ${OPENCODE_XDG_DATA_HOME}/opencode/auth.json"
    echo "Logs Directory: ${OPENCODE_LOG_DIR}"
}

opencode::status::check() {
    if ! opencode::run_cli --version >/dev/null 2>&1; then
        log::error "OpenCode CLI health check failed"
        return 1
    fi
    log::success "OpenCode CLI health check passed"
}
