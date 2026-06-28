#!/bin/bash
# Grok Build CLI installation and teardown helpers (official upstream binary).
#
# Lands the real `grok` binary on PATH (~/.local/bin/grok) so agent-manager and
# operators invoke it directly — no wrapper shim. Mirrors the codex/opencode
# resources, which install their upstream binary into a user-writable prefix and
# let the resource-* Go CLI handle only lifecycle + governance.
#
# Strategy: download the pinned single-file artifact directly (the opencode
# pattern) rather than running the official `curl | bash` installer, so we honor
# the resource.json version pin, never mutate the operator's shell rc, and stay
# fully re-runnable. The artifact URL matrix mirrors x.ai/cli/install.sh:
#   https://x.ai/cli/grok-<version>-<os>-<arch>   (os: linux|macos|windows,
#                                                  arch: x86_64|aarch64)
# with a direct-GCS fallback when x.ai is unreachable.

source "${BASH_SOURCE[0]%/*}/common.sh"

grok::install::detect_platform() {
    local uname_s uname_m
    uname_s=$(uname -s | tr '[:upper:]' '[:lower:]')
    uname_m=$(uname -m)

    case "${uname_s}" in
        linux)
            GROK_INSTALL_OS="linux"
            ;;
        darwin)
            GROK_INSTALL_OS="macos"
            ;;
        msys*|mingw*|cygwin*)
            GROK_INSTALL_OS="windows"
            ;;
        *)
            log::error "Unsupported operating system: ${uname_s}"
            return 1
            ;;
    esac

    case "${uname_m}" in
        x86_64|amd64)
            GROK_INSTALL_ARCH="x86_64"
            ;;
        arm64|aarch64)
            GROK_INSTALL_ARCH="aarch64"
            ;;
        *)
            log::error "Unsupported architecture: ${uname_m}"
            return 1
            ;;
    esac

    GROK_INSTALL_PLATFORM="${GROK_INSTALL_OS}-${GROK_INSTALL_ARCH}"
    return 0
}

# grok::install::pinned_version reads upstream_cli.version_pinned from
# resource.json so a plain install is reproducible (honours the pin) rather
# than silently tracking whatever upstream tags latest.
grok::install::pinned_version() {
    local manifest="${GROK_DIR}/resource.json"
    if command -v jq >/dev/null 2>&1 && [[ -f "${manifest}" ]]; then
        jq -r '.upstream_cli.version_pinned // empty' "${manifest}" 2>/dev/null
    fi
}

# grok::install::channel_latest probes the release channel pointer to discover
# the latest version when no pin/override is set. Tries x.ai first, then GCS.
grok::install::channel_latest() {
    local result
    result=$(curl -fsSL "${GROK_BASE_URL_PRIMARY}/${GROK_CHANNEL}" 2>/dev/null || true)
    if [[ -z "${result}" ]]; then
        result=$(curl -fsSL "${GROK_BASE_URL_FALLBACK}/${GROK_CHANNEL}" 2>/dev/null || true)
    fi
    printf '%s' "${result}" | tr -d '[:space:]'
}

grok::install::determine_version() {
    local requested_version="${GROK_VERSION:-}"
    if [[ -z "${requested_version}" ]]; then
        requested_version="$(grok::install::pinned_version)"
    fi
    if [[ -n "${requested_version}" ]]; then
        GROK_INSTALL_VERSION="${requested_version#v}"
        return 0
    fi

    if ! command -v curl >/dev/null 2>&1; then
        log::error "curl is required to discover the latest Grok release"
        return 1
    fi
    GROK_INSTALL_VERSION="$(grok::install::channel_latest)"
    if [[ -z "${GROK_INSTALL_VERSION}" ]]; then
        log::error "Unable to determine the latest Grok version from ${GROK_BASE_URL_PRIMARY}/${GROK_CHANNEL}"
        return 1
    fi
    return 0
}

grok::install::download() {
    if ! command -v curl >/dev/null 2>&1; then
        log::error "curl is required to install Grok"
        return 1
    fi

    local suffix=""
    [[ "${GROK_INSTALL_OS}" == "windows" ]] && suffix=".exe"

    local artifact="grok-${GROK_INSTALL_VERSION}-${GROK_INSTALL_PLATFORM}${suffix}"
    local tmp_dir binary_tmp
    tmp_dir=$(mktemp -d)
    binary_tmp="${tmp_dir}/grok${suffix}"

    local downloaded=0 url
    for base in "${GROK_BASE_URL_PRIMARY}" "${GROK_BASE_URL_FALLBACK}"; do
        url="${base}/${artifact}"
        log::info "Downloading Grok ${GROK_INSTALL_VERSION} (${url})"
        if curl -fsSL -o "${binary_tmp}" "${url}"; then
            downloaded=1
            break
        fi
        log::warning "Download failed: ${url}"
    done

    if [[ "${downloaded}" -ne 1 ]]; then
        rm -rf "${tmp_dir}"
        log::error "Failed to download Grok ${GROK_INSTALL_VERSION} for ${GROK_INSTALL_PLATFORM} (tried x.ai and GCS)"
        return 1
    fi

    chmod +x "${binary_tmp}"
    # Verify the artifact actually runs before clobbering any existing install.
    if [[ "${GROK_INSTALL_OS}" != "windows" ]]; then
        if ! "${binary_tmp}" --version </dev/null >/dev/null 2>&1; then
            log::error "Downloaded grok failed to run; keeping the existing install."
            rm -rf "${tmp_dir}"
            return 1
        fi
    fi

    mkdir -p "${GROK_BIN_DIR}"
    rm -f "${GROK_BIN}${suffix}"
    mv "${binary_tmp}" "${GROK_BIN}${suffix}"
    chmod +x "${GROK_BIN}${suffix}"
    rm -rf "${tmp_dir}"

    mkdir -p "${GROK_DATA_DIR}"
    printf '%s' "${GROK_INSTALL_VERSION}" >"${GROK_VERSION_FILE}"
    log::success "Installed Grok ${GROK_INSTALL_VERSION} to ${GROK_BIN}${suffix}"
}

grok::install::execute() {
    log::info "Installing Grok Build CLI"

    # Refuse to fight a root-owned `grok` already on PATH: clobbering it would
    # need sudo and is not our job here (privileged setup vacates it first).
    local blocker
    blocker="$(agent_install::blocking_system_install grok "${GROK_BIN_DIR}")"
    if [[ -n "${blocker}" ]]; then
        log::error "A root-owned grok already exists on PATH at ${blocker}."
        log::error "Remove it (e.g. 'sudo rm ${blocker}') or vacate it via privileged setup, then re-run; this resource never installs grok with sudo."
        return 1
    fi

    grok::ensure_dirs

    if ! grok::install::detect_platform; then
        return 1
    fi
    if ! grok::install::determine_version; then
        return 1
    fi
    if ! grok::install::download; then
        return 1
    fi

    # Surface a pre-existing grok earlier on PATH that would shadow our install.
    agent_install::warn_if_shadowed grok "${GROK_BIN_DIR}"
    log::success "Grok Build CLI installation complete"
}

# grok::install::update reinstalls grok to the pinned version. The opt-in
# self-update surface (mirrors codex/opencode `update`); never runs
# automatically. `determine_version` reads the pin, so this catches up a
# `behind` binary and is fully idempotent.
grok::install::update() { grok::install::execute "$@"; }

grok::install::uninstall() {
    log::info "Uninstalling Grok Build CLI"
    local removed=0
    for f in "${GROK_BIN}" "${GROK_BIN}.exe"; do
        if [[ -f "${f}" ]]; then
            rm -f "${f}"
            log::info "Removed ${f}"
            removed=1
        fi
    done
    rm -f "${GROK_VERSION_FILE}" 2>/dev/null || true
    if [[ "${removed}" -eq 0 ]]; then
        log::info "No user-owned grok binary to remove"
    fi
    # Intentionally leave ${GROK_DATA_DIR} (~/.grok) in place: it holds durable
    # config/auth/session state declared for backup. Removing it is the
    # operator's deliberate choice, not an uninstall side effect.
    log::info "Left ${GROK_DATA_DIR} intact (durable config/auth/session state)."
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-install}" in
        install) shift || true; grok::install::execute "$@" ;;
        update) shift || true; grok::install::update "$@" ;;
        uninstall) shift || true; grok::install::uninstall "$@" ;;
        *) grok::install::execute ;;
    esac
fi
