#!/bin/bash
# OpenCode installation and teardown helpers

source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::install::execute() {
    log::info "Installing OpenCode AI CLI"

    if ! opencode::ensure_python; then
        return 1
    fi

    opencode::ensure_dirs
    opencode::load_secrets || true

    if [[ -f "${OPENCODE_CONFIG_FILE}" ]]; then
        log::info "Configuration already exists at ${OPENCODE_CONFIG_FILE}"
    else
        log::info "Creating default configuration"
        local provider="${OPENCODE_DEFAULT_PROVIDER}"
        local chat_model="${OPENCODE_DEFAULT_CHAT_MODEL}"
        local completion_model="${OPENCODE_DEFAULT_COMPLETION_MODEL}"
        local timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

        if command -v jq &>/dev/null; then
            jq -n \
                --arg provider "${provider}" \
                --arg chat "${chat_model}" \
                --arg completion "${completion_model}" \
                --arg created "${timestamp}" \
                --argjson port ${OPENCODE_PORT} \
                '{
                    provider: $provider,
                    chat_model: $chat,
                    completion_model: $completion,
                    port: $port,
                    auto_suggest: true,
                    enable_logging: true,
                    created: $created
                }' > "${OPENCODE_CONFIG_FILE}"
        else
            cat > "${OPENCODE_CONFIG_FILE}" <<'JSON'
{
  "provider": "${provider}",
  "chat_model": "${chat_model}",
  "completion_model": "${completion_model}",
  "port": ${OPENCODE_PORT},
  "auto_suggest": true,
  "enable_logging": true,
  "created": "${timestamp}"
}
JSON
        fi
        log::success "Default configuration created"
    fi

    log::success "OpenCode CLI installation complete"
}

opencode::install::uninstall() {
    log::info "Uninstalling OpenCode AI CLI"
    if [[ -d "${OPENCODE_DATA_DIR}" ]]; then
        rm -rf "${OPENCODE_DATA_DIR}"
        log::success "Removed ${OPENCODE_DATA_DIR}"
    else
        log::info "No data directory to remove"
    fi
}
