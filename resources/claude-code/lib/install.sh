#!/usr/bin/env bash
# Claude Code upstream-CLI install (owned, sudo-free).
#
# Claude Code ships as a self-contained native binary installed by Anthropic's
# official installer into ~/.local (NOT via npm). This resource drives that
# installer so the managed install matches how Claude Code is actually
# distributed, lands in ~/.local/bin/claude, and needs no root for any update.
# The pin in resource.json (upstream_cli.version_pinned) is honoured.
#
# If a root-owned system claude (e.g. a legacy `npm install -g` into /usr) is
# already on PATH, the install refuses with an actionable message rather than
# silently shadowing or failing — removing a root-owned copy needs sudo, which
# only the privileged setup step may use.
set -euo pipefail

CLAUDE_RESOURCE_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${CLAUDE_RESOURCE_DIR}/../.." && builtin pwd)"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/resources/lib/agent-install.sh"

CLAUDE_BINARY="claude"
CLAUDE_INSTALLER_URL="${CLAUDE_INSTALLER_URL:-https://claude.ai/install.sh}"
# Anthropic's native installer lands the binary here.
CLAUDE_BIN_DIR="${CLAUDE_BIN_DIR:-${HOME}/.local/bin}"

claude_code::install::pinned_version() {
    local manifest="${CLAUDE_RESOURCE_DIR}/resource.json"
    if command -v jq >/dev/null 2>&1 && [[ -f "${manifest}" ]]; then
        jq -r '.upstream_cli.version_pinned // empty' "${manifest}" 2>/dev/null
    fi
}

claude_code::install::execute() {
    log::info "Installing Claude Code CLI (native installer, user-owned, sudo-free)"

    local downloader=""
    if command -v curl >/dev/null 2>&1; then
        downloader="curl"
    elif command -v wget >/dev/null 2>&1; then
        downloader="wget"
    fi
    if [[ -z "${downloader}" ]]; then
        log::error "curl or wget is required to run the Claude Code installer"
        return 1
    fi

    local blocker
    blocker="$(agent_install::blocking_system_install "${CLAUDE_BINARY}" "${CLAUDE_BIN_DIR}")"
    if [[ -n "${blocker}" ]]; then
        log::error "Refusing to install: a system Claude CLI already exists at ${blocker}"
        log::error "  It lives in a root-owned location, so replacing it requires sudo —"
        log::error "  which only the privileged 'make setup' step may use."
        log::error "  This resource installs sudo-free into ${CLAUDE_BIN_DIR}. To migrate once:"
        log::error "    1) remove the system copy (needs sudo), e.g. sudo npm remove -g @anthropic-ai/claude-code"
        log::error "    2) re-run: vrooli resource install claude-code   (lands in ${CLAUDE_BIN_DIR}; no sudo afterwards)"
        return 1
    fi

    # The installer accepts [stable|latest|VERSION]; the pin is a bare semver.
    local version target
    version="${CLAUDE_VERSION:-$(claude_code::install::pinned_version)}"
    target="${version#v}"
    if [[ -z "${target}" ]]; then
        target="stable"
    fi

    log::info "Running Claude Code native installer (target: ${target})"
    local script=""
    if [[ "${downloader}" == "curl" ]]; then
        script="$(curl -fsSL "${CLAUDE_INSTALLER_URL}")" || { log::error "Failed to download ${CLAUDE_INSTALLER_URL}"; return 1; }
    else
        script="$(wget -qO- "${CLAUDE_INSTALLER_URL}")" || { log::error "Failed to download ${CLAUDE_INSTALLER_URL}"; return 1; }
    fi
    if ! printf '%s' "${script}" | bash -s -- "${target}"; then
        log::error "Claude Code native installer failed"
        return 1
    fi

    agent_install::warn_if_shadowed "${CLAUDE_BINARY}" "${CLAUDE_BIN_DIR}"
    log::success "Installed Claude Code CLI (${target}) to ${CLAUDE_BIN_DIR}/${CLAUDE_BINARY}"
}

# update reinstalls to the pin/latest via the same native installer.
claude_code::install::update() { claude_code::install::execute "$@"; }

claude_code::install::uninstall() {
    log::info "Claude Code is managed by its native installer under ~/.local/share/claude"
    log::info "To remove it, delete ${CLAUDE_BIN_DIR}/${CLAUDE_BINARY} and ~/.local/share/claude"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-install}" in
        install) shift || true; claude_code::install::execute "$@" ;;
        update) shift || true; claude_code::install::update "$@" ;;
        uninstall) shift || true; claude_code::install::uninstall "$@" ;;
        *) claude_code::install::execute ;;
    esac
fi
