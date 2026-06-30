#!/bin/bash

# Install-time path/dir glue for the Antigravity CLI resource (official binary).
#
# The upstream Antigravity build ships as a per-platform artifact (a tar.gz that
# contains a single binary named `antigravity` on linux/macOS, or a bare `.exe`
# on Windows). This resource lands it on PATH at ~/.local/bin/agy — a
# USER-writable location so install/update never need root — mirroring how the
# grok/codex/opencode resources land their upstream binary and let the
# resource-* Go CLI own only lifecycle + governance. Antigravity's own runtime
# state (config, the native `permissions` grants, conversation trajectories,
# memory) lives under ~/.gemini and is declared via resource.json `durable_data`.
set -euo pipefail

VROOLI_ROOT="${VROOLI_ROOT:-${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}}"
ANTIGRAVITY_DIR="${VROOLI_ROOT}/resources/antigravity"

# Logging utilities (needed when common.sh is sourced standalone).
source "${VROOLI_ROOT}/scripts/lib/utils/log.sh"
# Shared agent-install policy helpers (root-owned-copy refusal + PATH shadow warn).
source "${VROOLI_ROOT}/scripts/resources/lib/agent-install.sh"

# Install target — the real binary goes on PATH (no shim, no indirection),
# mirroring how the grok/codex/opencode resources land their upstream binary.
# The ANTIGRAVITY_SHIM_DIR override exists for tests/sandboxes.
ANTIGRAVITY_BIN_DIR="${ANTIGRAVITY_SHIM_DIR:-${HOME}/.local/bin}"
ANTIGRAVITY_BIN="${ANTIGRAVITY_BIN_DIR}/agy"

# Antigravity's own runtime/durable state directory (settings.json, the native
# permission grants, the jetski conversation/trajectory store, memory). Declared
# for backup via resource.json durable_data; created lazily by `agy` on first run.
ANTIGRAVITY_DATA_DIR="${ANTIGRAVITY_DATA_DIR:-${HOME}/.gemini}"

# Vrooli-owned install marker + staging area. Deliberately kept OUT of the
# durable base (~/.gemini) so the backup target stays clean; ~/.cache/antigravity
# is also where the official installer stages downloads.
ANTIGRAVITY_STATE_DIR="${ANTIGRAVITY_STATE_DIR:-${HOME}/.cache/antigravity}"
ANTIGRAVITY_VERSION_FILE="${ANTIGRAVITY_STATE_DIR}/.vrooli-installed-version"

# Upstream version + artifact source. Antigravity is not on npm/GitHub releases:
# its latest version, artifact URL, and sha512 are served as a per-platform JSON
# manifest from the auto-updater service (the same source the official
# antigravity.google/cli/install.sh probes).
ANTIGRAVITY_MANIFEST_BASE="${ANTIGRAVITY_MANIFEST_BASE:-https://antigravity-cli-auto-updater-974169037036.us-central1.run.app}"

antigravity::ensure_dirs() {
    mkdir -p "${ANTIGRAVITY_BIN_DIR}"
    mkdir -p "${ANTIGRAVITY_STATE_DIR}"
}
