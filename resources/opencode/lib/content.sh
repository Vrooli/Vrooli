#!/bin/bash
# OpenCode Content Functions - manage resource-owned configs

source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::content::add() {
    local content_type="${1:-config}"

    case "${content_type}" in
        config|configuration)
            opencode::content::add_config "$@"
            ;;
        *)
            log::error "Unknown content type: ${content_type}"
            log::info "Supported types: config"
            return 1
            ;;
    esac
}

opencode::content::add_config() {
    local config_name="${2:-default}"
    local first_arg="${3:-}"
    local second_arg="${4:-}"
    local third_arg="${5:-}"

    local model=""
    local small_model=""

    if [[ -z "${first_arg}" ]]; then
        model="${OPENCODE_DEFAULT_PROVIDER}/${OPENCODE_DEFAULT_CHAT_MODEL}"
        small_model="${OPENCODE_DEFAULT_PROVIDER}/${OPENCODE_DEFAULT_COMPLETION_MODEL}"
    elif [[ "${first_arg}" == */* ]]; then
        model="${first_arg}"
        if [[ -n "${second_arg}" ]]; then
            small_model="${second_arg}"
        else
            small_model="${model}"
        fi
    else
        local provider="${first_arg}"
        local chat_model="${second_arg:-${OPENCODE_DEFAULT_CHAT_MODEL}}"
        local completion_model="${third_arg:-${OPENCODE_DEFAULT_COMPLETION_MODEL}}"
        model="${provider}/${chat_model}"
        small_model="${provider}/${completion_model}"
    fi

    opencode::ensure_dirs

    local config_file="${OPENCODE_CONFIG_DIR}/config-${config_name}.json"
    cat <<EOF >"${config_file}"
{
  "\$schema": "https://opencode.ai/config.json",
  "model": "${model}",
  "small_model": "${small_model}",
  "instructions": [
    "AGENTS.md"
  ]
}
EOF

    log::success "Configuration '${config_name}' saved to ${config_file}"
    log::info "To activate this configuration, run: resource-opencode content execute ${config_name}"
}

opencode::content::list() {
    local content_type="${1:-config}"

    case "${content_type}" in
        config|configurations)
            opencode::content::list_configs
            ;;
        *)
            log::error "Unknown content type: ${content_type}"
            log::info "Supported types: config"
            return 1
            ;;
    esac
}

opencode::content::list_configs() {
    log::info "OpenCode configurations:"

    if [[ -f "${OPENCODE_CONFIG_FILE}" ]]; then
        echo "📋 Active configuration:"
        if command -v jq &>/dev/null; then
            jq '{model, small_model}' "${OPENCODE_CONFIG_FILE}" 2>/dev/null || cat "${OPENCODE_CONFIG_FILE}"
        else
            cat "${OPENCODE_CONFIG_FILE}"
        fi
        echo
    fi

    local glob_pattern="${OPENCODE_CONFIG_DIR}/config-"*
    local config_paths=(${glob_pattern})

    echo "💾 Saved configurations:"
    if [[ ${#config_paths[@]} -eq 0 || ! -e ${config_paths[0]} ]]; then
        echo "   (none)"
        return 0
    fi

    for cfg in "${config_paths[@]}"; do
        local display_name
        display_name=$(basename "${cfg}" .json | sed 's/^config-//')
        if command -v jq &>/dev/null; then
            local summary
            summary=$(jq -r '.model // ""' "${cfg}" 2>/dev/null)
            echo " - ${display_name}: ${summary:-see file}"
        else
            echo " - ${display_name}: ${cfg}"
        fi
    done
}

opencode::content::get() {
    local content_type="${1:-}"
    local name="${2:-}"

    if [[ -z "${content_type}" ]]; then
        log::error "Content type is required"
        log::info "Usage: opencode content get <type> <name>"
        return 1
    fi

    case "${content_type}" in
        config|configuration)
            opencode::content::get_config "${name}"
            ;;
        *)
            log::error "Unknown content type: ${content_type}"
            return 1
            ;;
    esac
}

opencode::content::get_config() {
    local config_name="${1:-active}"

    if [[ "${config_name}" == "active" ]]; then
        if [[ -f "${OPENCODE_CONFIG_FILE}" ]]; then
            cat "${OPENCODE_CONFIG_FILE}"
        else
            log::error "No active configuration found"
            return 1
        fi
        return 0
    fi

    local config_file="${OPENCODE_CONFIG_DIR}/config-${config_name}.json"
    if [[ -f "${config_file}" ]]; then
        cat "${config_file}"
    else
        log::error "Configuration '${config_name}' not found"
        return 1
    fi
}

opencode::content::remove() {
    local content_type="${1:-}"
    local name="${2:-}"

    if [[ -z "${content_type}" || -z "${name}" ]]; then
        log::error "Content type and name are required"
        log::info "Usage: opencode content remove <type> <name>"
        return 1
    fi

    case "${content_type}" in
        config|configuration)
            opencode::content::remove_config "${name}"
            ;;
        *)
            log::error "Unknown content type: ${content_type}"
            return 1
            ;;
    esac
}

opencode::content::remove_config() {
    local config_name="${1}"

    if [[ "${config_name}" == "active" ]]; then
        log::error "Cannot remove active configuration"
        log::info "Create a new configuration and activate it first"
        return 1
    fi

    local config_file="${OPENCODE_CONFIG_DIR}/config-${config_name}.json"
    if [[ -f "${config_file}" ]]; then
        rm "${config_file}"
        log::success "Configuration '${config_name}' removed"
    else
        log::error "Configuration '${config_name}' not found"
        return 1
    fi
}

opencode::content::activate() {
    local config_name="${1:-}"

    if [[ -z "${config_name}" ]]; then
        log::error "Configuration name is required"
        log::info "Usage: opencode content activate <config_name>"
        return 1
    fi

    local config_file="${OPENCODE_CONFIG_DIR}/config-${config_name}.json"
    if [[ -f "${config_file}" ]]; then
        cp "${config_file}" "${OPENCODE_CONFIG_FILE}"
        log::success "Configuration '${config_name}' activated"
    else
        log::error "Configuration '${config_name}' not found"
        return 1
    fi
}
