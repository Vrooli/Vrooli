#!/usr/bin/env bash
# Codex upstream-CLI install (owned, sudo-free).
#
# Installs the @openai/codex npm package into a USER-writable prefix
# (~/.local by default) so the binary lands in ~/.local/bin/codex and every
# subsequent update/reinstall needs no root — mirroring the opencode and
# claude-code resources. The pin in resource.json (upstream_cli.version_pinned)
# is honoured so installs are reproducible rather than silently tracking
# whatever npm tags `latest`.
#
# If a root-owned system Codex (e.g. an `npm install -g` into /usr) is already
# on PATH, the install refuses with an actionable message rather than failing
# with a cryptic EACCES — replacing a root-owned copy needs sudo, which only
# the privileged setup step may use.
set -euo pipefail

CODEX_RESOURCE_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${CODEX_RESOURCE_DIR}/../.." && builtin pwd)"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/resources/lib/agent-install.sh"

CODEX_NPM_PACKAGE="${CODEX_NPM_PACKAGE:-@openai/codex}"
CODEX_BINARY="codex"
# User-writable npm prefix; binaries land in $CODEX_NPM_PREFIX/bin.
CODEX_NPM_PREFIX="${CODEX_NPM_PREFIX:-${HOME}/.local}"
CODEX_BIN_DIR="${CODEX_NPM_PREFIX}/bin"

codex::install::pinned_version() {
    local manifest="${CODEX_RESOURCE_DIR}/resource.json"
    if command -v jq >/dev/null 2>&1 && [[ -f "${manifest}" ]]; then
        jq -r '.upstream_cli.version_pinned // empty' "${manifest}" 2>/dev/null
    fi
}

codex::install::execute() {
    log::info "Installing OpenAI Codex CLI (user-owned, sudo-free)"

    if ! command -v npm >/dev/null 2>&1; then
        log::error "npm is required to install ${CODEX_NPM_PACKAGE} (install Node.js first)"
        return 1
    fi

    local blocker
    blocker="$(agent_install::blocking_system_install "${CODEX_BINARY}" "${CODEX_BIN_DIR}")"
    if [[ -n "${blocker}" ]]; then
        local npm_prefix
        npm_prefix="$(npm config get prefix 2>/dev/null || echo '?')"
        log::error "Refusing to install: a system Codex CLI already exists at ${blocker}"
        log::error "  It lives in a root-owned location (npm global prefix: ${npm_prefix}), so replacing"
        log::error "  it requires sudo — which only the privileged 'make setup' step may use."
        log::error "  This resource installs sudo-free into ${CODEX_BIN_DIR}. To migrate once:"
        log::error "    1) remove the system copy (needs sudo): sudo npm remove -g ${CODEX_NPM_PACKAGE}"
        log::error "    2) re-run: vrooli resource install codex   (lands in ${CODEX_BIN_DIR}; no sudo afterwards)"
        return 1
    fi

    local version spec
    version="${CODEX_VERSION:-$(codex::install::pinned_version)}"
    spec="${CODEX_NPM_PACKAGE}"
    if [[ -n "${version}" ]]; then
        spec="${CODEX_NPM_PACKAGE}@${version#v}"
    fi

    mkdir -p "${CODEX_BIN_DIR}"
    log::info "npm install -g ${spec} --prefix ${CODEX_NPM_PREFIX}"
    if ! npm install -g "${spec}" --prefix "${CODEX_NPM_PREFIX}"; then
        log::error "npm install failed for ${spec}"
        return 1
    fi

    agent_install::warn_if_shadowed "${CODEX_BINARY}" "${CODEX_BIN_DIR}"
    log::success "Installed Codex CLI (${spec}) to ${CODEX_BIN_DIR}/${CODEX_BINARY}"
}

# update reinstalls to the pin/latest; identical to a fresh install because
# the npm prefix install is idempotent.
codex::install::update() { codex::install::execute "$@"; }

codex::install::uninstall() {
    log::info "Removing user-owned Codex CLI from ${CODEX_NPM_PREFIX}"
    if command -v npm >/dev/null 2>&1; then
        npm uninstall -g "${CODEX_NPM_PACKAGE}" --prefix "${CODEX_NPM_PREFIX}" >/dev/null 2>&1 || true
    fi
    log::success "Codex CLI removed from ${CODEX_BIN_DIR} (system copies untouched)"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-install}" in
        install) shift || true; codex::install::execute "$@" ;;
        update) shift || true; codex::install::update "$@" ;;
        uninstall) shift || true; codex::install::uninstall "$@" ;;
        *) codex::install::execute ;;
    esac
fi
