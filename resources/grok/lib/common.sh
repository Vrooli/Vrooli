#!/bin/bash

# Install-time path/dir glue for the Grok Build CLI resource (official binary).
#
# The upstream `grok` binary is a single self-contained download (no archive,
# no npm). This resource lands it on PATH at ~/.local/bin/grok — a USER-writable
# location so install/update never need root — mirroring how the codex/opencode
# resources land their upstream binary and let the resource-* Go CLI own only
# lifecycle + governance. Grok's own runtime state (config, auth, sessions)
# lives under ~/.grok and is declared via resource.json `durable_data`.
set -euo pipefail

VROOLI_ROOT="${VROOLI_ROOT:-${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}}"
GROK_DIR="${VROOLI_ROOT}/resources/grok"

# Logging utilities (needed when common.sh is sourced standalone).
source "${VROOLI_ROOT}/scripts/lib/utils/log.sh"
# Shared agent-install policy helpers (root-owned-copy refusal + PATH shadow warn).
source "${VROOLI_ROOT}/scripts/resources/lib/agent-install.sh"

# Install target — the real binary goes on PATH (no shim, no indirection),
# mirroring how the codex/opencode resources land their upstream binary. The
# GROK_SHIM_DIR override exists for tests/sandboxes.
GROK_BIN_DIR="${GROK_SHIM_DIR:-${HOME}/.local/bin}"
GROK_BIN="${GROK_BIN_DIR}/grok"

# Grok's own runtime state directory (config.toml, auth.json, sessions, …).
# Declared for backup via resource.json durable_data; created lazily by `grok`.
GROK_DATA_DIR="${GROK_DATA_DIR:-${HOME}/.grok}"
GROK_VERSION_FILE="${GROK_DATA_DIR}/.vrooli-installed-version"

# Upstream artifact sources. x.ai is Cloudflare-fronted; GCS is the direct
# fallback the official installer also uses.
GROK_BASE_URL_PRIMARY="https://x.ai/cli"
GROK_BASE_URL_FALLBACK="https://storage.googleapis.com/grok-build-public-artifacts/cli"
# Release channel pointer (stable|alpha|enterprise). A bare GET of
# "${base}/${channel}" returns the latest version string for that channel.
GROK_CHANNEL="${GROK_CHANNEL:-stable}"

grok::ensure_dirs() {
    mkdir -p "${GROK_BIN_DIR}"
    mkdir -p "${GROK_DATA_DIR}"
}
