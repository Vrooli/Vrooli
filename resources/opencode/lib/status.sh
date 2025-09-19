#!/bin/bash
# OpenCode status helpers

source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::status() {
    opencode::ensure_dirs
    opencode::load_secrets || true

    if ! opencode::ensure_python; then
        return 1
    fi

    local py
    py=$(opencode::python_bin)

    if [[ ! -f "${OPENCODE_CONFIG_FILE}" ]]; then
        log::warn "Configuration not found at ${OPENCODE_CONFIG_FILE}"
        echo "Create one with: resource-opencode manage install"
        return 1
    fi

    local info_json
    if ! info_json=$("${py}" "${OPENCODE_CLI_ENTRYPOINT}" --config "${OPENCODE_CONFIG_FILE}" info 2>/dev/null); then
        log::error "Failed to gather CLI status"
        return 1
    fi

    if command -v jq &>/dev/null; then
        echo "OpenCode CLI Status"
        echo "===================="
        echo "Provider      : $(echo "${info_json}" | jq -r '.provider')"
        echo "Chat Model    : $(echo "${info_json}" | jq -r '.chat_model')"
        echo "Completion    : $(echo "${info_json}" | jq -r '.completion_model')"
        echo "Config Path   : ${OPENCODE_CONFIG_FILE}"
        echo "Secrets       : $(echo "${info_json}" | jq -r '.secrets')"
        local model_count
        model_count=$(echo "${info_json}" | jq '[.models | to_entries[] | .value | length] | add // 0')
        echo "Models        : ${model_count} found"
        if [[ "${model_count}" -gt 0 ]]; then
            echo "--------------------"
            echo "Available Models:"
            echo "${info_json}" | jq -r '.models | to_entries[] | select(.value | length > 0) | "\(.key):"' | sed 's/^/ - /'
            echo "${info_json}" | jq -r '.models | to_entries[] | select(.value | length > 0) | .value[]' | sed 's/^/   • /'
        fi
    else
        echo "Provider: $(echo "${info_json}" | grep -o '"provider": "[^"]*"' | sed 's/.*"\(.*\)"/\1/')"
        echo "Config : ${OPENCODE_CONFIG_FILE}"
    fi
}

opencode::status::check() {
    if ! opencode::ensure_python; then
        return 1
    fi
    if [[ ! -f "${OPENCODE_CONFIG_FILE}" ]]; then
        log::error "No OpenCode configuration found"
        return 1
    fi
    if ! opencode::run_cli info >/dev/null; then
        log::error "OpenCode CLI health check failed"
        return 1
    fi
    log::success "OpenCode CLI health check passed"
}
