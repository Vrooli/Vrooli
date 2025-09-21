#!/usr/bin/env bash
# OpenCode Resource Unit Tests - permission helpers & argument parsing sanity checks

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
OPENCODE_DIR="${APP_ROOT}/resources/opencode"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${OPENCODE_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
source "${OPENCODE_DIR}/lib/common.sh"

pass_count=0
fail_count=0

assert_equals() {
    local expected="$1"
    local actual="$2"
    local description="$3"
    if [[ "$expected" == "$actual" ]]; then
        log::success "${description}"
        pass_count=$((pass_count + 1))
    else
        log::error "${description} (expected='${expected}' actual='${actual}')"
        fail_count=$((fail_count + 1))
    fi
}

log::info "Running OpenCode unit checks..."

assert_equals "allow" "$(opencode::permissions::decide "" "" "edit" "")" "Default allow when no allowlist configured"
assert_equals "allow" "$(opencode::permissions::decide "edit,write" "yes" "bash" "")" "Skip flag overrides allowlist"
patterns=$'ls *\nrm *'
assert_equals "allow" "$(opencode::permissions::decide "bash(ls *)" "" "bash" "$patterns")" "Pattern allowlist matches"
assert_equals "reject" "$(opencode::permissions::decide "bash(ls *)" "" "bash" $'rm *')" "Pattern allowlist rejects mismatched command"
assert_equals "reject" "$(opencode::permissions::decide "edit" "" "write" "")" "Allowlist enforces edit-only"

if [[ ${fail_count} -gt 0 ]]; then
    log::error "OpenCode unit tests failed (${fail_count} failure(s))"
    exit 1
fi

log::success "OpenCode unit tests passed (${pass_count} checks)"
