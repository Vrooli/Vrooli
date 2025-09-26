#!/usr/bin/env bash
################################################################################
# Kernel Configuration Utilities
#
# Centralized helpers for inspecting and tuning kernel parameters required by
# Vrooli scenarios and shared development tooling (e.g. inotify limits for Vite,
# Judge0 sandbox requirements).
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"

# Internal state flag used to let callers know if a function actually changed a
# kernel parameter.
KERNEL_CONFIG_DID_CHANGE=false

kernel_config::reset_change_flag() {
    KERNEL_CONFIG_DID_CHANGE=false
}

kernel_config::mark_changed() {
    KERNEL_CONFIG_DID_CHANGE=true
}

kernel_config::check_parameter() {
    local param="${1:-}"
    local expected="${2:-}"

    if [[ -z "$param" || -z "$expected" ]]; then
        return 1
    fi

    if ! command -v sysctl >/dev/null 2>&1; then
        log::warning "sysctl not available; cannot read $param"
        return 1
    fi

    local current
    if ! current=$(sysctl -n "$param" 2>/dev/null); then
        return 1
    fi

    [[ "$current" == "$expected" ]]
}

kernel_config::apply_parameter() {
    local param="$1"
    local value="$2"
    local description="${3:-$param}"

    if kernel_config::check_parameter "$param" "$value"; then
        log::debug "$param already set to $value"
        return 0
    fi

    if ! flow::can_run_sudo "configure kernel parameter $param"; then
        log::warning "Cannot modify $param without sudo; please rerun setup with elevated privileges"
        return 1
    fi

    log::info "Setting $param to $value ($description)"
    if ! sudo::exec_with_fallback "sysctl -w ${param}=${value}" >/dev/null 2>&1; then
        log::warning "Failed to set $param via sysctl"
        return 1
    fi

    kernel_config::mark_changed
    kernel_config::make_persistent "$param" "$value" "$description" || true
    return 0
}

kernel_config::make_persistent() {
    local param="$1"
    local value="$2"
    local description="${3:-Vrooli kernel tuning}"
    local config_file="/etc/sysctl.d/99-vrooli.conf"
    local escaped_param

    if ! flow::can_run_sudo "persist kernel parameter $param"; then
        log::warning "Cannot persist $param without sudo; apply manually by adding '${param} = ${value}' to $config_file"
        return 1
    fi

    escaped_param=${param//\//\\/}

    sudo::exec_with_fallback "mkdir -p /etc/sysctl.d"
    sudo::exec_with_fallback "touch '${config_file}'"
    sudo::exec_with_fallback "sed -i \"/^${escaped_param} = /d\" '${config_file}' 2>/dev/null || true"
    sudo::exec_with_fallback "echo '${param} = ${value}' >> '${config_file}'"
    sudo::exec_with_fallback "sysctl -p '${config_file}' >/dev/null 2>&1 || true"
    return 0
}

kernel_config::cleanup() {
    local config_file="/etc/sysctl.d/99-vrooli.conf"

    if ! flow::can_run_sudo "cleanup Vrooli kernel configuration"; then
        log::info "Skipping kernel configuration cleanup (no sudo permissions)"
        return 0
    fi

    log::info "Cleaning up Vrooli kernel configurations"
    sudo::exec_with_fallback "rm -f '${config_file}'"
    sudo::exec_with_fallback "sysctl -p >/dev/null 2>&1 || true"
    return 0
}

kernel_config::configure_judge0() {
    local param="kernel.apparmor_restrict_unprivileged_userns"
    local desired="0"

    if kernel_config::check_parameter "$param" "$desired"; then
        log::info "Judge0 kernel parameter already configured"
        return 0
    fi

    log::info "Configuring kernel parameters for Judge0"
    kernel_config::apply_parameter "$param" "$desired" "Allow unprivileged user namespaces"
}

kernel_config::ensure_inotify_limits() {
    local min_watches=${1:-524288}
    local min_instances=${2:-1024}
    local changed=false
    local current

    if ! command -v sysctl >/dev/null 2>&1; then
        log::warning "sysctl not available; cannot adjust inotify limits"
        return 1
    fi

    if current=$(sysctl -n fs.inotify.max_user_watches 2>/dev/null); then
        if [[ "${current}" =~ ^[0-9]+$ && ${current} -ge ${min_watches} ]]; then
            log::debug "fs.inotify.max_user_watches already ${current}" 
        else
            if kernel_config::apply_parameter "fs.inotify.max_user_watches" "$min_watches" "Increase inotify watchers for dev servers"; then
                changed=true
            fi
        fi
    else
        log::warning "Unable to read fs.inotify.max_user_watches"
    fi

    if current=$(sysctl -n fs.inotify.max_user_instances 2>/dev/null); then
        if [[ "${current}" =~ ^[0-9]+$ && ${current} -ge ${min_instances} ]]; then
            log::debug "fs.inotify.max_user_instances already ${current}" 
        else
            if kernel_config::apply_parameter "fs.inotify.max_user_instances" "$min_instances" "Increase inotify instances for dev servers"; then
                changed=true
            fi
        fi
    else
        log::warning "Unable to read fs.inotify.max_user_instances"
    fi

    if [[ "$changed" == true ]]; then
        kernel_config::mark_changed
    fi

    return 0
}

kernel_config::configure_for_resources() {
    local service_json="${var_SERVICE_JSON_FILE:-}" 
    local any_changes=false

    kernel_config::reset_change_flag
    kernel_config::ensure_inotify_limits || true
    if [[ "$KERNEL_CONFIG_DID_CHANGE" == true ]]; then
        any_changes=true
    fi

    if [[ -n "$service_json" && -f "$service_json" ]]; then
        local has_judge0="false"
        if command -v jq >/dev/null 2>&1; then
            has_judge0=$(jq -r '.resources.judge0.enabled // .judge0.enabled // false' "$service_json" 2>/dev/null || echo "false")
        fi

        if [[ "$has_judge0" == "true" ]]; then
            kernel_config::reset_change_flag
            kernel_config::configure_judge0 || true
            if [[ "$KERNEL_CONFIG_DID_CHANGE" == true ]]; then
                any_changes=true
            fi
        fi
    fi

    if [[ "$any_changes" == false ]]; then
        log::info "No kernel parameter changes required"
    else
        log::success "Kernel parameters verified"
    fi

    return 0
}

export -f kernel_config::check_parameter
export -f kernel_config::apply_parameter
export -f kernel_config::make_persistent
export -f kernel_config::cleanup
export -f kernel_config::configure_judge0
export -f kernel_config::ensure_inotify_limits
export -f kernel_config::configure_for_resources
