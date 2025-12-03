#!/bin/bash
# OpenCode installation and teardown helpers (official binary)

source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::install::detect_platform() {
    local uname_s uname_m
    uname_s=$(uname -s | tr '[:upper:]' '[:lower:]')
    uname_m=$(uname -m)

    case "${uname_s}" in
        linux)
            OPENCODE_INSTALL_OS="linux"
            ;;
        darwin)
            OPENCODE_INSTALL_OS="darwin"
            ;;
        msys*|mingw*|cygwin*)
            OPENCODE_INSTALL_OS="windows"
            ;;
        *)
            log::error "Unsupported operating system: ${uname_s}"
            return 1
            ;;
    esac

    case "${uname_m}" in
        x86_64|amd64)
            OPENCODE_INSTALL_ARCH="x64"
            ;;
        arm64|aarch64)
            OPENCODE_INSTALL_ARCH="arm64"
            ;;
        *)
            log::error "Unsupported architecture: ${uname_m}"
            return 1
            ;;
    esac

    if [[ "${OPENCODE_INSTALL_OS}" == "windows" && "${OPENCODE_INSTALL_ARCH}" != "x64" ]]; then
        log::error "OpenCode only ships Windows builds for x64"
        return 1
    fi

    return 0
}

opencode::install::determine_version() {
    local requested_version="${OPENCODE_VERSION:-}" api_json
    if [[ -n "${requested_version}" ]]; then
        OPENCODE_INSTALL_VERSION="${requested_version}"
        local default_ext="zip"
        [[ "${OPENCODE_INSTALL_OS}" == "linux" ]] && default_ext="tar.gz"
        OPENCODE_INSTALL_URL="https://github.com/sst/opencode/releases/download/v${requested_version}/opencode-${OPENCODE_INSTALL_OS}-${OPENCODE_INSTALL_ARCH}.${default_ext}"
        return 0
    fi

    if ! command -v curl &>/dev/null; then
        log::error "curl is required to discover the latest OpenCode release"
        return 1
    fi

    api_json=$(curl -fsSL "https://api.github.com/repos/sst/opencode/releases/latest" || true)
    if [[ -z "${api_json}" ]]; then
        log::error "Failed to query the GitHub releases API"
        return 1
    fi

    if command -v jq &>/dev/null; then
        OPENCODE_INSTALL_VERSION=$(printf '%s' "${api_json}" | jq -r '.tag_name // ""' 2>/dev/null | sed 's/^v//')
    else
        OPENCODE_INSTALL_VERSION=$(printf '%s' "${api_json}" | awk -F'"' '/"tag_name"/ {gsub(/^v/, "", $4); print $4; exit}')
    fi
    if [[ -z "${OPENCODE_INSTALL_VERSION}" || "${OPENCODE_INSTALL_VERSION}" == "null" ]]; then
        log::error "Unable to determine the latest OpenCode version"
        return 1
    fi

    local default_ext="zip"
    [[ "${OPENCODE_INSTALL_OS}" == "linux" ]] && default_ext="tar.gz"
    OPENCODE_INSTALL_URL="https://github.com/sst/opencode/releases/download/v${OPENCODE_INSTALL_VERSION}/opencode-${OPENCODE_INSTALL_OS}-${OPENCODE_INSTALL_ARCH}.${default_ext}"
    return 0
}

opencode::install::download() {
    local tmp_dir archive

    if ! command -v curl &>/dev/null; then
        log::error "curl is required to install OpenCode"
        return 1
    fi

    tmp_dir=$(mktemp -d)
    local base_url="https://github.com/sst/opencode/releases/download/v${OPENCODE_INSTALL_VERSION}/opencode-${OPENCODE_INSTALL_OS}-${OPENCODE_INSTALL_ARCH}"
    local candidates=()

    if [[ "${OPENCODE_INSTALL_OS}" == "linux" ]]; then
        candidates+=(
            "${base_url}.tar.gz"
            "${base_url}.zip"
            "${base_url}-musl.tar.gz"
            "${base_url}-baseline.tar.gz"
            "${base_url}-baseline-musl.tar.gz"
        )
    else
        candidates+=(
            "${base_url}.zip"
            "${base_url}-baseline.zip"
        )
    fi

    local downloaded=0
    for url in "${candidates[@]}"; do
        archive="${tmp_dir}/$(basename "${url}")"
        log::info "Downloading OpenCode ${OPENCODE_INSTALL_VERSION} (${url##*/})"
        if ! curl -fsSL -o "${archive}" "${url}"; then
            log::warning "Download failed: ${url}"
            continue
        fi

        local extension="${archive##*.}"
        if [[ "${extension}" == "zip" ]]; then
            if ! command -v unzip &>/dev/null; then
                log::error "unzip is required to install OpenCode (missing for ${url##*/})"
                continue
            fi
            if ! unzip -q "${archive}" -d "${tmp_dir}"; then
                log::warning "Failed to extract zip archive from ${url}"
                continue
            fi
        else
            if ! command -v tar &>/dev/null; then
                log::error "tar is required to install OpenCode (missing for ${url##*/})"
                continue
            fi
            if ! tar -xzf "${archive}" -C "${tmp_dir}"; then
                log::warning "Failed to extract tar archive from ${url}"
                continue
            fi
        fi

        if [[ ! -f "${tmp_dir}/opencode" ]]; then
            log::warning "Archive ${url##*/} did not contain expected 'opencode' binary"
            continue
        fi

        downloaded=1
        break
    done

    if [[ "${downloaded}" -ne 1 ]]; then
        rm -rf "${tmp_dir}"
        log::error "Failed to download a compatible OpenCode archive (tried: ${candidates[*]})"
        return 1
    fi

    mkdir -p "${OPENCODE_BIN_DIR}"
    rm -f "${OPENCODE_BIN}"
    mv "${tmp_dir}/opencode" "${OPENCODE_BIN}"
    chmod +x "${OPENCODE_BIN}"
    rm -rf "${tmp_dir}"

    printf '%s' "${OPENCODE_INSTALL_VERSION}" >"${OPENCODE_VERSION_FILE}"
    log::success "Installed OpenCode ${OPENCODE_INSTALL_VERSION} to ${OPENCODE_BIN}"
}

opencode::install::write_shim() {
    # Install a wrapper so `opencode` works outside resource-opencode while
    # still using the resource-scoped config and secrets.
    local shim_dir="${OPENCODE_SHIM_DIR:-${HOME}/.local/bin}"
    local shim_path="${shim_dir}/opencode"

    mkdir -p "${shim_dir}"

    if [[ -e "${shim_path}" && ! -L "${shim_path}" ]]; then
        log::warning "Skipping shim creation because ${shim_path} already exists"
        return 0
    fi

    cat >"${shim_path}" <<EOF
#!/usr/bin/env bash
APP_ROOT="${APP_ROOT}"
# shellcheck disable=SC1090
source "${OPENCODE_DIR}/lib/common.sh"
opencode::run_cli "\$@"
EOF
    chmod +x "${shim_path}" 2>/dev/null || true

    log::success "Installed global OpenCode shim at ${shim_path}"
    if ! printf '%s' "${PATH}" | tr ':' '\n' | grep -Fx "${shim_dir}" >/dev/null 2>&1; then
        log::info "Add ${shim_dir} to your PATH to call 'opencode' directly."
    fi
}

opencode::install::remove_shim() {
    local shim_dir="${OPENCODE_SHIM_DIR:-${HOME}/.local/bin}"
    local shim_path="${shim_dir}/opencode"

    if [[ -f "${shim_path}" ]] && grep -q "${OPENCODE_DIR}/lib/common.sh" "${shim_path}" 2>/dev/null; then
        rm -f "${shim_path}" 2>/dev/null || true
        log::info "Removed OpenCode shim at ${shim_path}"
    fi
}

opencode::install::execute() {
    log::info "Installing OpenCode AI CLI"

    opencode::ensure_dirs

    if ! opencode::install::detect_platform; then
        return 1
    fi
    if ! opencode::install::determine_version; then
        return 1
    fi
    if ! opencode::install::download; then
        return 1
    fi

    opencode::ensure_config
    opencode::install::write_shim
    log::success "OpenCode CLI installation complete"
}

opencode::install::uninstall() {
    log::info "Uninstalling OpenCode AI CLI"
    opencode::install::remove_shim
    if [[ -d "${OPENCODE_DATA_DIR}" ]]; then
        rm -rf "${OPENCODE_DATA_DIR}"
        log::success "Removed ${OPENCODE_DATA_DIR}"
    else
        log::info "No data directory to remove"
    fi
}
