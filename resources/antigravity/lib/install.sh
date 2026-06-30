#!/bin/bash
# Antigravity CLI installation and teardown helpers (official upstream binary).
#
# Lands the real `agy` binary on PATH (~/.local/bin/agy) so operators (and, when
# later wired, agent-manager) invoke it directly — no wrapper shim. Mirrors the
# grok/codex/opencode resources, which install their upstream binary into a
# user-writable prefix and let the resource-* Go CLI handle only lifecycle +
# governance.
#
# Strategy: query the upstream per-platform JSON manifest, then download the
# artifact it points at directly (the opencode/grok pattern) rather than running
# the official `curl | bash` installer. We do NOT run `agy install` afterwards —
# that step mutates the operator's shell profile (PATH append + alias purge); we
# never touch the operator's rc, and ~/.local/bin is already on PATH. The
# manifest contract is:
#   GET ${MANIFEST_BASE}/manifests/<platform>.json
#     -> { "version": "...", "url": "<artifact>", "sha512": "<hex>" }
#   platform = {linux,darwin}_{amd64,arm64} | windows_{amd64,arm64}
#   linux/darwin artifacts are *.tar.gz containing a binary named `antigravity`;
#   windows artifacts are a bare *.exe. (No musl build is published.)
#
# Reproducible/pinned/air-gapped installs: set ANTIGRAVITY_ARTIFACT_URL (and
# optionally ANTIGRAVITY_ARTIFACT_SHA512) to bypass the manifest lookup. Note
# that `agy` self-updates in the background during normal runs, so the installed
# version is a floor, not a ceiling; resource.json upstream_cli.version_pinned is
# the known-good baseline used for drift reporting (see internal/upstream).

source "${BASH_SOURCE[0]%/*}/common.sh"

antigravity::install::detect_platform() {
    local uname_s uname_m
    uname_s=$(uname -s | tr '[:upper:]' '[:lower:]')
    uname_m=$(uname -m)

    case "${uname_s}" in
        linux)
            ANTIGRAVITY_INSTALL_OS="linux"
            ;;
        darwin)
            ANTIGRAVITY_INSTALL_OS="darwin"
            ;;
        msys*|mingw*|cygwin*)
            ANTIGRAVITY_INSTALL_OS="windows"
            ;;
        *)
            log::error "Unsupported operating system: ${uname_s}"
            return 1
            ;;
    esac

    case "${uname_m}" in
        x86_64|amd64)
            ANTIGRAVITY_INSTALL_ARCH="amd64"
            ;;
        arm64|aarch64)
            ANTIGRAVITY_INSTALL_ARCH="arm64"
            ;;
        *)
            log::error "Unsupported architecture: ${uname_m}"
            return 1
            ;;
    esac

    # Antigravity publishes no musl build; refuse early with an actionable
    # message rather than downloading a glibc artifact that won't run.
    if [[ "${ANTIGRAVITY_INSTALL_OS}" == "linux" ]]; then
        if [[ -f /lib/libc.musl-x86_64.so.1 || -f /lib/libc.musl-aarch64.so.1 ]] || ldd /bin/ls 2>&1 | grep -q musl; then
            log::error "Antigravity CLI does not publish a musl build; this host's libc is musl. Unsupported."
            return 1
        fi
    fi

    ANTIGRAVITY_INSTALL_PLATFORM="${ANTIGRAVITY_INSTALL_OS}_${ANTIGRAVITY_INSTALL_ARCH}"
    return 0
}

# antigravity::install::pinned_version reads upstream_cli.version_pinned from
# resource.json — the known-good baseline reported by `upstream-check`. (It does
# not force the install version: the manifest only serves "latest", and the
# artifact URL embeds an opaque build id that cannot be reconstructed from a
# version string alone.)
antigravity::install::pinned_version() {
    local manifest="${ANTIGRAVITY_DIR}/resource.json"
    if command -v jq >/dev/null 2>&1 && [[ -f "${manifest}" ]]; then
        jq -r '.upstream_cli.version_pinned // empty' "${manifest}" 2>/dev/null
    fi
}

# POSIX-compliant single-key extractor for the flat manifest JSON (no jq dep,
# matching the official installer's parser).
antigravity::install::parse_manifest_key() {
    local payload="$1" key="$2"
    printf '%s' "${payload}" | sed -n 's/.*"'"${key}"'"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1
}

# antigravity::install::resolve_artifact populates ANTIGRAVITY_INSTALL_VERSION,
# ANTIGRAVITY_ARTIFACT_URL, ANTIGRAVITY_ARTIFACT_SHA512 — from explicit env
# overrides if set, otherwise from the upstream per-platform manifest.
antigravity::install::resolve_artifact() {
    if [[ -n "${ANTIGRAVITY_ARTIFACT_URL:-}" ]]; then
        ANTIGRAVITY_INSTALL_VERSION="${ANTIGRAVITY_VERSION:-$(antigravity::install::pinned_version)}"
        ANTIGRAVITY_INSTALL_VERSION="${ANTIGRAVITY_INSTALL_VERSION:-override}"
        ANTIGRAVITY_ARTIFACT_SHA512="${ANTIGRAVITY_ARTIFACT_SHA512:-}"
        log::info "Using ANTIGRAVITY_ARTIFACT_URL override (${ANTIGRAVITY_ARTIFACT_URL})"
        return 0
    fi

    if ! command -v curl >/dev/null 2>&1; then
        log::error "curl is required to query the Antigravity release manifest"
        return 1
    fi

    local manifest_url manifest_json
    manifest_url="${ANTIGRAVITY_MANIFEST_BASE}/manifests/${ANTIGRAVITY_INSTALL_PLATFORM}.json"
    log::info "Querying Antigravity release manifest (${manifest_url})"
    manifest_json=$(curl -fsSL "${manifest_url}" 2>/dev/null || true)
    if [[ -z "${manifest_json}" ]]; then
        log::error "Could not fetch the Antigravity release manifest for ${ANTIGRAVITY_INSTALL_PLATFORM}"
        return 1
    fi

    ANTIGRAVITY_INSTALL_VERSION="$(antigravity::install::parse_manifest_key "${manifest_json}" version)"
    ANTIGRAVITY_ARTIFACT_URL="$(antigravity::install::parse_manifest_key "${manifest_json}" url)"
    ANTIGRAVITY_ARTIFACT_SHA512="$(antigravity::install::parse_manifest_key "${manifest_json}" sha512)"

    if [[ -z "${ANTIGRAVITY_ARTIFACT_URL}" ]]; then
        log::error "Antigravity manifest for ${ANTIGRAVITY_INSTALL_PLATFORM} did not contain an artifact url"
        return 1
    fi

    local pinned
    pinned="$(antigravity::install::pinned_version)"
    if [[ -n "${pinned}" && -n "${ANTIGRAVITY_INSTALL_VERSION}" && "${pinned}" != "${ANTIGRAVITY_INSTALL_VERSION}" ]]; then
        log::warning "Upstream manifest serves ${ANTIGRAVITY_INSTALL_VERSION}; resource.json pins ${pinned} as the known-good baseline. The upstream manifest only serves latest — installing ${ANTIGRAVITY_INSTALL_VERSION}. (agy also self-updates in the background.)"
    fi
    return 0
}

antigravity::install::download() {
    if ! command -v curl >/dev/null 2>&1; then
        log::error "curl is required to install Antigravity"
        return 1
    fi

    local tmp_dir payload_tmp binary_tmp
    tmp_dir=$(mktemp -d)
    # shellcheck disable=SC2064
    trap "rm -rf '${tmp_dir}'" RETURN

    local is_tar_gz=0
    case "${ANTIGRAVITY_ARTIFACT_URL}" in
        *.tar.gz*) is_tar_gz=1 ;;
    esac

    if [[ "${is_tar_gz}" -eq 1 ]]; then
        payload_tmp="${tmp_dir}/agy.tar.gz"
    else
        payload_tmp="${tmp_dir}/agy"
    fi

    log::info "Downloading Antigravity ${ANTIGRAVITY_INSTALL_VERSION} (${ANTIGRAVITY_ARTIFACT_URL})"
    if ! curl -fsSL -o "${payload_tmp}" "${ANTIGRAVITY_ARTIFACT_URL}"; then
        log::error "Failed to download Antigravity artifact from ${ANTIGRAVITY_ARTIFACT_URL}"
        return 1
    fi

    # Verify the sha512 from the manifest when we have it (the official
    # installer treats a mismatch as a hard security halt).
    if [[ -n "${ANTIGRAVITY_ARTIFACT_SHA512:-}" ]]; then
        local actual_hash
        if [[ "${ANTIGRAVITY_INSTALL_OS}" == "darwin" ]]; then
            actual_hash=$(shasum -a 512 "${payload_tmp}" | cut -d' ' -f1 || true)
        else
            actual_hash=$(sha512sum "${payload_tmp}" | cut -d' ' -f1 || true)
        fi
        if [[ "${actual_hash}" != "${ANTIGRAVITY_ARTIFACT_SHA512}" ]]; then
            log::error "Antigravity artifact checksum mismatch — refusing to install (expected ${ANTIGRAVITY_ARTIFACT_SHA512}, got ${actual_hash})."
            return 1
        fi
        log::info "Verified Antigravity artifact sha512 checksum"
    else
        log::warning "No sha512 available for the Antigravity artifact; skipping checksum verification"
    fi

    if [[ "${is_tar_gz}" -eq 1 ]]; then
        # The tarball contains a single binary named `antigravity`.
        if ! tar -xzf "${payload_tmp}" -C "${tmp_dir}" antigravity 2>/dev/null; then
            # Fall back to extracting whatever single file the archive holds.
            tar -xzf "${payload_tmp}" -C "${tmp_dir}" 2>/dev/null || true
        fi
        binary_tmp="${tmp_dir}/antigravity"
        if [[ ! -f "${binary_tmp}" ]]; then
            binary_tmp=$(find "${tmp_dir}" -maxdepth 2 -type f ! -name '*.tar.gz' | head -n1)
        fi
    else
        binary_tmp="${payload_tmp}"
    fi

    if [[ -z "${binary_tmp}" || ! -f "${binary_tmp}" ]]; then
        log::error "Could not locate the Antigravity binary inside the downloaded artifact"
        return 1
    fi

    chmod +x "${binary_tmp}"
    # Verify the artifact actually runs before clobbering any existing install.
    if [[ "${ANTIGRAVITY_INSTALL_OS}" != "windows" ]]; then
        if ! "${binary_tmp}" --version </dev/null >/dev/null 2>&1; then
            log::error "Downloaded agy failed to run; keeping the existing install."
            return 1
        fi
    fi

    local suffix=""
    [[ "${ANTIGRAVITY_INSTALL_OS}" == "windows" ]] && suffix=".exe"
    mkdir -p "${ANTIGRAVITY_BIN_DIR}"
    rm -f "${ANTIGRAVITY_BIN}${suffix}"
    mv "${binary_tmp}" "${ANTIGRAVITY_BIN}${suffix}"
    chmod +x "${ANTIGRAVITY_BIN}${suffix}"

    mkdir -p "${ANTIGRAVITY_STATE_DIR}"
    printf '%s' "${ANTIGRAVITY_INSTALL_VERSION}" >"${ANTIGRAVITY_VERSION_FILE}"
    log::success "Installed Antigravity ${ANTIGRAVITY_INSTALL_VERSION} to ${ANTIGRAVITY_BIN}${suffix}"
}

antigravity::install::execute() {
    log::info "Installing Antigravity CLI"

    # Refuse to fight a root-owned `agy` already on PATH: clobbering it would
    # need sudo and is not our job here (privileged setup vacates it first).
    local blocker
    blocker="$(agent_install::blocking_system_install agy "${ANTIGRAVITY_BIN_DIR}")"
    if [[ -n "${blocker}" ]]; then
        log::error "A root-owned agy already exists on PATH at ${blocker}."
        log::error "Remove it (e.g. 'sudo rm ${blocker}') or vacate it via privileged setup, then re-run; this resource never installs agy with sudo."
        return 1
    fi

    antigravity::ensure_dirs

    if ! antigravity::install::detect_platform; then
        return 1
    fi
    if ! antigravity::install::resolve_artifact; then
        return 1
    fi
    if ! antigravity::install::download; then
        return 1
    fi

    # Surface a pre-existing agy earlier on PATH that would shadow our install.
    agent_install::warn_if_shadowed agy "${ANTIGRAVITY_BIN_DIR}"
    log::success "Antigravity CLI installation complete"
}

# antigravity::install::update reinstalls agy to the current upstream version.
# Idempotent reinstall (mirrors codex/opencode `update`); never runs
# automatically. (agy also self-updates in the background.)
antigravity::install::update() { antigravity::install::execute "$@"; }

antigravity::install::uninstall() {
    log::info "Uninstalling Antigravity CLI"
    local removed=0
    for f in "${ANTIGRAVITY_BIN}" "${ANTIGRAVITY_BIN}.exe"; do
        if [[ -f "${f}" ]]; then
            rm -f "${f}"
            log::info "Removed ${f}"
            removed=1
        fi
    done
    rm -f "${ANTIGRAVITY_VERSION_FILE}" 2>/dev/null || true
    if [[ "${removed}" -eq 0 ]]; then
        log::info "No user-owned agy binary to remove"
    fi
    # Intentionally leave ${ANTIGRAVITY_DATA_DIR} (~/.gemini) in place: it holds
    # durable config/permission/conversation state declared for backup. Removing
    # it is the operator's deliberate choice, not an uninstall side effect.
    log::info "Left ${ANTIGRAVITY_DATA_DIR} intact (durable config/permission/conversation state)."
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-install}" in
        install) shift || true; antigravity::install::execute "$@" ;;
        update) shift || true; antigravity::install::update "$@" ;;
        uninstall) shift || true; antigravity::install::uninstall "$@" ;;
        *) antigravity::install::execute ;;
    esac
fi
