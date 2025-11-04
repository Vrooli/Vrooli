#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"

_js_yaml::npm_prefix_install() {
    local prefix="$1"
    mkdir -p "${prefix}/bin"
    NPM_CONFIG_PREFIX="$prefix" npm install -g js-yaml >/dev/null 2>&1
    if [[ ":$PATH:" != *":${prefix}/bin:"* ]]; then
        export PATH="${prefix}/bin:$PATH"
        log::info "Temporarily added ${prefix}/bin to PATH"
    fi
}

js_yaml::install() {
    if system::is_command "js-yaml"; then
        log::info "js-yaml CLI already available"
        return 0
    fi

    if ! system::is_command npm; then
        if system::is_command pnpm; then
            log::warning "npm missing; attempting pnpm installation for js-yaml"
        else
            log::error "Cannot install js-yaml: npm is not available"
            return 1
        fi
    fi

    log::header "🧩 Installing js-yaml CLI"

    if system::is_command npm; then
        local can_use_sudo=1
        set +e
        flow::can_run_sudo "npm install -g js-yaml"
        can_use_sudo=$?
        set -euo pipefail
        if [ "$can_use_sudo" -eq 0 ]; then
            sudo npm install -g js-yaml >/dev/null 2>&1 || true
        else
            local user_prefix="${HOME}/.local"
            _js_yaml::npm_prefix_install "$user_prefix"
        fi
    else
        # Fallback to pnpm global install
        local pnpm_can_sudo=1
        set +e
        flow::can_run_sudo "pnpm add -g js-yaml"
        pnpm_can_sudo=$?
        set -euo pipefail
        if [ "$pnpm_can_sudo" -eq 0 ]; then
            sudo pnpm add -g js-yaml >/dev/null 2>&1 || true
        else
            local pnpm_home="${HOME}/.local"
            mkdir -p "${pnpm_home}/bin"
            PNPM_HOME="$pnpm_home" pnpm add -g js-yaml >/dev/null 2>&1 || true
            if [[ ":$PATH:" != *":${pnpm_home}/bin:"* ]]; then
                export PATH="${pnpm_home}/bin:$PATH"
                log::info "Temporarily added ${pnpm_home}/bin to PATH"
            fi
        fi
    fi

    if js-yaml --version >/dev/null 2>&1; then
        log::success "js-yaml CLI installed"
        return 0
    fi

    log::error "js-yaml installation completed but command still missing"
    return 1
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    js_yaml::install "$@"
fi
