#!/usr/bin/env bash
# OpenCode resource — local-model reliability tie (Phase 6).
#
# OpenCode's local coding path resolves the `code.local` Ollama role. This
# phase asserts that role's seated model actually tool-calls, via the probe
# SSOT (`resource-ollama models doctor --role code.local`) — so a regression to
# a model that narrates tool calls as text (the silent-success failure class)
# turns the suite RED rather than surfacing only at agent run time.
#
# Daemon-aware: SKIPS (passes) when Ollama or the resource-ollama CLI is
# unavailable, so it never false-fails on infrastructure.

set -euo pipefail

VROOLI_ROOT="${VROOLI_ROOT:-${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}}"
OLLAMA_DIR="${VROOLI_ROOT}/resources/ollama"

# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

export OLLAMA_MODEL_POLICY_PATH="${OLLAMA_MODEL_POLICY_PATH:-${OLLAMA_DIR}/model-policy.json}"

# Prefer an installed resource-ollama; otherwise build from source.
if command -v resource-ollama >/dev/null 2>&1; then
    ssot="resource-ollama"
else
    log::info "resource-ollama not on PATH; building from source..."
    ssot="$(mktemp -d)/resource-ollama"
    if ! (cd "${OLLAMA_DIR}/cli" && go build -o "${ssot}" .); then
        log::warning "Could not build resource-ollama; SKIPPING opencode local-model reliability tie"
        exit 0
    fi
fi

if ! "${ssot}" models list --json >/dev/null 2>&1; then
    log::warning "Ollama daemon unreachable; SKIPPING opencode local-model reliability tie (not a failure)"
    exit 0
fi

log::info "Validating opencode's local role (code.local) tool-calls via the SSOT..."
if "${ssot}" models doctor --role code.local; then
    log::success "code.local model emits structured tool_calls — opencode local path is reliable"
    exit 0
fi

log::error "code.local model failed the tool-call smoke — opencode local runs would silently no-op"
exit 1
