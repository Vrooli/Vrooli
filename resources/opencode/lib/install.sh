#!/bin/bash
# OpenCode installation and teardown helpers (official upstream binary).
#
# Lands the real `opencode` binary on PATH (~/.local/bin/opencode) so
# agent-manager and operators invoke it directly — no wrapper shim. Mirrors
# the codex/claude-code resources, which install their upstream binary
# globally and let the resource-* Go CLI handle only lifecycle + governance.

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

# opencode::install::pinned_version reads upstream_cli.version_pinned from
# resource.json so a plain install is reproducible (honours the pin) rather
# than silently tracking whatever upstream tags latest.
opencode::install::pinned_version() {
    local manifest="${OPENCODE_DIR}/resource.json"
    if command -v jq >/dev/null 2>&1 && [[ -f "${manifest}" ]]; then
        jq -r '.upstream_cli.version_pinned // empty' "${manifest}" 2>/dev/null
    fi
}

opencode::install::determine_version() {
    local requested_version="${OPENCODE_VERSION:-}" api_json
    if [[ -z "${requested_version}" ]]; then
        requested_version="$(opencode::install::pinned_version)"
    fi
    if [[ -n "${requested_version}" ]]; then
        OPENCODE_INSTALL_VERSION="${requested_version#v}"
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

    mkdir -p "${OPENCODE_DATA_DIR}"
    printf '%s' "${OPENCODE_INSTALL_VERSION}" >"${OPENCODE_VERSION_FILE}"
    log::success "Installed OpenCode ${OPENCODE_INSTALL_VERSION} to ${OPENCODE_BIN}"

    if ! printf '%s' "${PATH}" | tr ':' '\n' | grep -Fx "${OPENCODE_BIN_DIR}" >/dev/null 2>&1; then
        log::info "Add ${OPENCODE_BIN_DIR} to your PATH to call 'opencode' directly."
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

    # Generate opencode.json + sync OpenRouter auth via the Go config writer
    # (resolves secrets, decides provider via the resource-ollama SSOT, and
    # preserves the governed permission map). See cli/internal/{config,secrets}.
    opencode::ensure_config
    # Heal any pre-1.0 opencode.json that still carries the retired inline
    # `x-vrooli-managed-permissions` key (opencode rejects unknown top-level
    # keys → startup fails). Best-effort: resource-opencode is installed by
    # the cli.install step and the verb is idempotent + ungated.
    if command -v resource-opencode >/dev/null 2>&1; then
        resource-opencode permissions migrate >/dev/null 2>&1 || true
    fi
    log::success "OpenCode CLI installation complete"
}

# opencode::install::update reinstalls opencode to the pinned version. The
# opt-in self-update surface (mirrors codex/claude's `update`); never runs
# automatically. `determine_version` reads the pin, so this catches up a
# `behind` binary.
opencode::install::update() { opencode::install::execute "$@"; }

opencode::install::uninstall() {
    log::info "Uninstalling OpenCode AI CLI"
    if [[ -f "${OPENCODE_BIN}" ]]; then
        rm -f "${OPENCODE_BIN}"
        log::info "Removed ${OPENCODE_BIN}"
    fi
    if [[ -d "${OPENCODE_DATA_DIR}" ]]; then
        rm -rf "${OPENCODE_DATA_DIR}"
        log::success "Removed ${OPENCODE_DATA_DIR}"
    else
        log::info "No data directory to remove"
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-install}" in
        install) shift || true; opencode::install::execute "$@" ;;
        update) shift || true; opencode::install::update "$@" ;;
        uninstall) shift || true; opencode::install::uninstall "$@" ;;
        *) opencode::install::execute ;;
    esac
fi
