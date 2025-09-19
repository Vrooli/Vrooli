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
        OPENCODE_INSTALL_URL="https://github.com/sst/opencode/releases/download/v${requested_version}/opencode-${OPENCODE_INSTALL_OS}-${OPENCODE_INSTALL_ARCH}.zip"
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

    OPENCODE_INSTALL_URL="https://github.com/sst/opencode/releases/download/v${OPENCODE_INSTALL_VERSION}/opencode-${OPENCODE_INSTALL_OS}-${OPENCODE_INSTALL_ARCH}.zip"
    return 0
}

opencode::install::download() {
    local tmp_dir archive

    if ! command -v unzip &>/dev/null; then
        log::error "unzip is required to install OpenCode"
        return 1
    fi
    if ! command -v curl &>/dev/null; then
        log::error "curl is required to install OpenCode"
        return 1
    fi

    tmp_dir=$(mktemp -d)
    archive="${tmp_dir}/opencode.zip"

    log::info "Downloading OpenCode ${OPENCODE_INSTALL_VERSION} for ${OPENCODE_INSTALL_OS}/${OPENCODE_INSTALL_ARCH}"
    if ! curl -fsSL -o "${archive}" "${OPENCODE_INSTALL_URL}"; then
        log::error "Failed to download ${OPENCODE_INSTALL_URL}"
        rm -rf "${tmp_dir}"
        return 1
    fi

    if ! unzip -q "${archive}" -d "${tmp_dir}"; then
        log::error "Failed to extract OpenCode archive"
        rm -rf "${tmp_dir}"
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
