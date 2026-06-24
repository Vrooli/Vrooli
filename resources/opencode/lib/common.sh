#!/bin/bash

# Install-time glue for the OpenCode AI CLI resource (official binary).
#
# Config generation and secret resolution were migrated to the Go CLI
# (`resource-opencode config ensure`): see cli/internal/config (opencode.json
# builder — provider self-heal, native Ollama block, num_ctx + role sampling,
# legacy-slug migration; preserves the permission map) and cli/internal/secrets
# (OpenRouter key resolution + auth.json sync). Ollama probing now flows through
# the resource-ollama SSOT, not bash. This file owns ONLY the path/dir glue the
# install path and test phase need.
set -euo pipefail

VROOLI_ROOT="${VROOLI_ROOT:-${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}}"
OPENCODE_DIR="${VROOLI_ROOT}/resources/opencode"

# Logging utilities (needed when common.sh is sourced standalone).
source "${VROOLI_ROOT}/scripts/lib/utils/log.sh"

# Load defaults (default provider/model slugs; documents the env overrides the
# Go writer also honors).
source "${OPENCODE_DIR}/config/defaults.sh"

# Default-XDG locations — the single source of truth raw `opencode` reads.
opencode_xdg_config_home="${XDG_CONFIG_HOME:-${HOME}/.config}"
opencode_xdg_data_home="${XDG_DATA_HOME:-${HOME}/.local/share}"

OPENCODE_CONFIG_DIR="${opencode_xdg_config_home}/opencode"
OPENCODE_CONFIG_FILE="${OPENCODE_CONFIG_DIR}/opencode.json"
OPENCODE_DATA_DIR="${opencode_xdg_data_home}/opencode"
OPENCODE_AUTH_FILE="${OPENCODE_DATA_DIR}/auth.json"
OPENCODE_LOG_DIR="${OPENCODE_DATA_DIR}/log"
OPENCODE_VERSION_FILE="${OPENCODE_DATA_DIR}/VERSION"

# Install target — the real binary goes on PATH (no shim, no indirection),
# mirroring how the codex/claude-code resources land their upstream binary.
OPENCODE_BIN_DIR="${OPENCODE_SHIM_DIR:-${HOME}/.local/bin}"
OPENCODE_BIN="${OPENCODE_BIN_DIR}/opencode"

opencode::ensure_dirs() {
    mkdir -p "${OPENCODE_CONFIG_DIR}"
    mkdir -p "${OPENCODE_DATA_DIR}"
    mkdir -p "${OPENCODE_LOG_DIR}"
    mkdir -p "${OPENCODE_BIN_DIR}"
}

# opencode::ensure_config delegates to the Go config writer. That verb resolves
# the OpenRouter key, decides the provider (cloud vs local Ollama self-heal via
# the resource-ollama SSOT), writes opencode.json (preserving the permission
# map and unknown keys), and syncs the OpenRouter auth store. Best-effort: a
# cold first install before resource-opencode lands is tolerated and self-heals
# on a later configure/health pass.
opencode::ensure_config() {
    if command -v resource-opencode >/dev/null 2>&1; then
        resource-opencode config ensure \
            || log::warning "resource-opencode config ensure failed; opencode.json may be stale"
    else
        log::warning "resource-opencode not yet installed; skipping opencode.json generation (self-heals on next configure)"
    fi
}
