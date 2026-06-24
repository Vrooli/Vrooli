#!/usr/bin/env bash
# Ollama resource — reliability validation harness (Phase 6).
#
# Validates that every TOOL-REQUIRING role's seated model actually tool-calls
# against the live daemon, via the probe SSOT (`resource-ollama models doctor`).
# The suite goes RED if a tool-role model regresses to ungrounded / no-
# structured-tool_calls behavior (the silent-success failure class this plan
# closes). Capability + stub-template are reported too; only a definitive
# behavioral tool-call failure fails the gate (the live smoke is authoritative).
#
# Hermetic + daemon-aware: when Ollama is unreachable the phase SKIPS (passes)
# rather than false-failing on infrastructure.

set -euo pipefail

VROOLI_ROOT="${VROOLI_ROOT:-${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}}"
OLLAMA_DIR="${VROOLI_ROOT}/resources/ollama"

# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

export OLLAMA_MODEL_POLICY_PATH="${OLLAMA_MODEL_POLICY_PATH:-${OLLAMA_DIR}/model-policy.json}"

log::info "Building resource-ollama for the reliability harness..."
bin="$(mktemp -d)/resource-ollama"
if ! (cd "${OLLAMA_DIR}/cli" && go build -o "${bin}" .); then
    log::error "Failed to build resource-ollama"
    exit 1
fi

# Daemon reachability via the SSOT itself.
if ! "${bin}" models list --json >/dev/null 2>&1; then
    log::warning "Ollama daemon unreachable; SKIPPING reliability harness (not a failure)"
    exit 0
fi

log::info "Running tool-role admission doctor (models doctor --all)..."
doctor_json="$("${bin}" models doctor --all --json 2>/dev/null || true)"
if [[ -z "${doctor_json}" ]]; then
    log::error "models doctor produced no output"
    exit 1
fi

# Evaluate the tool-role pass rate. Threshold: every tool-requiring role must
# pass. (A stub template alone is a warning, not a failure — the live tool-call
# smoke is the authority.)
read -r tool_total tool_pass < <(python3 - "$doctor_json" <<'PY'
import json, sys
d = json.loads(sys.argv[1])
tool = [m for m in d.get("models", []) if m.get("requires_tools")]
passed = [m for m in tool if m.get("pass")]
print(len(tool), len(passed))
PY
)

log::info "Tool-requiring roles: ${tool_pass}/${tool_total} passed"
"${bin}" models doctor --all 2>/dev/null || true   # human-readable detail in the log

if [[ "${tool_total}" -eq 0 ]]; then
    log::warning "No tool-requiring roles declared in policy; nothing to gate"
    exit 0
fi
if [[ "${tool_pass}" -lt "${tool_total}" ]]; then
    log::error "Reliability gate FAILED: $((tool_total - tool_pass)) tool-role model(s) cannot tool-call (regression to stub/ungrounded behavior)"
    exit 1
fi

log::success "Reliability gate PASSED: all ${tool_total} tool-role model(s) emit structured tool_calls"
exit 0
