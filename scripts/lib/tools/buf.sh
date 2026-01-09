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
source "${var_LIB_UTILS_DIR}/retry.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"

BUF_VERSION="${BUF_VERSION:-1.37.0}"

buf::version_ge() {
    local current="$1" target="$2"
    if [[ "$current" == "$target" ]]; then
        return 0
    fi
    # shellcheck disable=SC2016
    [[ "$(printf '%s\n%s\n' "$target" "$current" | sort -V | head -n1)" == "$target" ]]
}

buf::ensure_installed() {
    if system::is_command buf; then
        local current
        current="$(buf --version 2>/dev/null || true)"
        if [[ -n "$current" ]] && buf::version_ge "$current" "$BUF_VERSION"; then
            log::info "buf already installed (version $current)"
            return 0
        fi
        log::info "buf found (version ${current:-unknown}); upgrading to ${BUF_VERSION}"
    else
        log::header "📦 Installing buf ${BUF_VERSION}"
    fi

    local platform arch
    platform="$(system::detect_platform)"
    arch="$(system::detect_arch)"

    local buf_platform buf_arch
    case "$platform" in
        linux) buf_platform="Linux" ;;
        darwin) buf_platform="Darwin" ;;
        *)
            log::error "Unsupported platform for buf: $platform"
            return 1
            ;;
    esac

    case "$arch" in
        amd64) buf_arch="x86_64" ;;
        arm64) buf_arch="arm64" ;;
        *)
            log::error "Unsupported architecture for buf: $arch"
            return 1
            ;;
    esac

    local filename="buf-${buf_platform}-${buf_arch}"
    local url="https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}/${filename}"

    local install_dir="/usr/local/bin"
    local use_sudo=true
    if [[ "${SUDO_MODE:-ask}" == "skip" ]] || ! sudo::can_use_sudo; then
        use_sudo=false
        install_dir="${HOME}/.local/bin"
        mkdir -p "$install_dir"
        if [[ ":$PATH:" != *":${install_dir}:"* ]]; then
            export PATH="${install_dir}:$PATH"
            log::info "Added ${install_dir} to PATH for current session"
        fi
    fi

    local tmpdir
    tmpdir="$(mktemp -d)"
    local tmpfile="${tmpdir}/${filename}"

    if ! retry::exponential_backoff 3 2 curl -fsSL "$url" -o "$tmpfile"; then
        log::error "Failed to download buf from ${url}"
        rm -rf "$tmpdir"
        return 1
    fi

    chmod +x "$tmpfile"

    if [[ "$use_sudo" == true ]]; then
        sudo::exec_with_fallback "cp '$tmpfile' '${install_dir}/buf'"
    else
        cp "$tmpfile" "${install_dir}/buf"
    fi
    rm -rf "$tmpdir"

    if buf --version >/dev/null 2>&1; then
        log::success "buf $(buf --version) installed to ${install_dir}"
        return 0
    fi

    log::error "buf installation completed but command not available"
    return 1
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    buf::ensure_installed "$@"
fi
