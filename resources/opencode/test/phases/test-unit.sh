#!/usr/bin/env bash
# OpenCode Resource Unit Tests.
#
# Config generation + secret resolution were migrated from bash to the Go CLI
# (cli/internal/{config,secrets}); their logic is covered by Go table tests run
# below. This phase additionally runs a hermetic integration smoke of
# `resource-opencode config ensure` to prove the binary writes a usable
# opencode.json and preserves the governed permission map.

set -euo pipefail

VROOLI_ROOT="${VROOLI_ROOT:-${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}}"
OPENCODE_DIR="${VROOLI_ROOT}/resources/opencode"

# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

pass_count=0
fail_count=0

assert_equals() {
    local expected="$1" actual="$2" description="$3"
    if [[ "$expected" == "$actual" ]]; then
        log::success "${description}"
        pass_count=$((pass_count + 1))
    else
        log::error "${description} (expected='${expected}' actual='${actual}')"
        fail_count=$((fail_count + 1))
    fi
}

log::info "Running OpenCode Go unit tests..."
if (cd "${OPENCODE_DIR}/cli" && go test ./internal/... >/tmp/opencode-gotest.log 2>&1); then
    log::success "Go unit tests passed (cli/internal/...)"
    pass_count=$((pass_count + 1))
else
    log::error "Go unit tests failed:"
    cat /tmp/opencode-gotest.log
    fail_count=$((fail_count + 1))
fi

log::info "Running hermetic 'config ensure' integration smoke..."
smoke_bin="$(mktemp -d)/resource-opencode"
if ! (cd "${OPENCODE_DIR}/cli" && go build -o "${smoke_bin}" .); then
    log::error "Failed to build resource-opencode for the smoke test"
    fail_count=$((fail_count + 1))
else
    # Hermetic XDG so writes never touch the operator's real config/auth.
    smoke_xdg=$(mktemp -d)
    export XDG_CONFIG_HOME="${smoke_xdg}/config"
    export XDG_DATA_HOME="${smoke_xdg}/data"
    cfg="${XDG_CONFIG_HOME}/opencode/opencode.json"

    # Pre-seed a permission map (as the Go permissions adapter writes) so we can
    # assert the config writer preserves it rather than clobbering it.
    mkdir -p "${XDG_CONFIG_HOME}/opencode"
    cat <<'EOF' >"${cfg}"
{
  "permission": {
    "bash": {
      "git push*": "deny"
    }
  }
}
EOF

    # No daemon dependency: with no usable key and Ollama maybe-absent the writer
    # still produces a coherent cloud default; assert the managed keys + that the
    # permission map survives.
    if "${smoke_bin}" config ensure >/dev/null 2>&1; then
        assert_equals "deny" "$(jq -r '.permission.bash["git push*"]' "${cfg}")" "config ensure preserves existing permission.bash entries"
        model="$(jq -r '.model' "${cfg}")"
        if [[ -n "${model}" && "${model}" != "null" ]]; then
            log::success "config ensure writes a model (${model})"
            pass_count=$((pass_count + 1))
        else
            log::error "config ensure did not write a model"
            fail_count=$((fail_count + 1))
        fi
        assert_equals "https://opencode.ai/config.json" "$(jq -r '.["$schema"]' "${cfg}")" "config ensure writes the opencode schema"
    else
        log::error "config ensure exited non-zero"
        fail_count=$((fail_count + 1))
    fi
    rm -rf "${smoke_xdg}"
fi

if [[ ${fail_count} -gt 0 ]]; then
    log::error "OpenCode unit tests failed (${fail_count} failure(s))"
    exit 1
fi

log::success "OpenCode unit tests passed (${pass_count} checks)"

# The runner-owned policy protocol is always part of the resource suite. Live
# credentials may be unavailable in CI; the phase records that as a skip.
bash "${OPENCODE_DIR}/test/phases/test-model-policy.sh"
