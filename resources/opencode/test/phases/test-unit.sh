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

default_payload="$(opencode::default_config_payload)"
assert_equals "1" "$(grep -c '"model": "openrouter/qwen/qwen3-coder"' <(printf '%s' "${default_payload}"))" "Default config uses updated chat model slug"
assert_equals "1" "$(grep -c '"small_model": "openrouter/qwen/qwen3-coder"' <(printf '%s' "${default_payload}"))" "Default config uses updated completion model slug"

tmp_config=$(mktemp)
cat <<'EOF' >"${tmp_config}"
{
  "model": "openrouter/qwen3-coder",
  "small_model": "openrouter/qwen3-coder"
}
EOF
opencode::config::migrate_legacy_models "${tmp_config}"

assert_equals "openrouter/qwen/qwen3-coder" "$(jq -r '.model' "${tmp_config}")" "Migrates legacy chat model slug"
assert_equals "openrouter/qwen/qwen3-coder" "$(jq -r '.small_model' "${tmp_config}")" "Migrates legacy completion model slug"
rm -f "${tmp_config}"

OPENCODE_SECRETS_LOADED=0
OPENROUTER_API_KEY="auto-null-placeholder"
tmp_root=$(mktemp -d)
mkdir -p "${tmp_root}/data/credentials"
cat <<'EOF' >"${tmp_root}/data/credentials/openrouter-credentials.json"
{
  "data": {
    "apiKey": "sk-or-v1-testkey1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
  }
}
EOF
previous_root="${var_ROOT_DIR:-}"
previous_data="${var_DATA_DIR:-}"
var_ROOT_DIR="${tmp_root}"
var_DATA_DIR="${tmp_root}/data"
opencode::load_secrets
rm -rf "${tmp_root}"
if [[ "${OPENROUTER_API_KEY}" == auto-null-* || -z "${OPENROUTER_API_KEY}" ]]; then
    log::error "Failed to resolve OpenRouter API key"
    fail_count=$((fail_count + 1))
else
    log::success "Resolves OpenRouter API key from fallback sources"
    pass_count=$((pass_count + 1))
fi

auth_file="${OPENCODE_XDG_DATA_HOME}/opencode/auth.json"
if [[ -f "${auth_file}" ]]; then
    stored_key=$(jq -r '.openrouter.key // empty' "${auth_file}")
    if [[ -z "${stored_key}" || "${stored_key}" == auto-null-* ]]; then
        log::error "Auth store missing usable OpenRouter key"
        fail_count=$((fail_count + 1))
    else
        log::success "Auth store contains OpenRouter API key"
        pass_count=$((pass_count + 1))
    fi
else
    log::error "Auth store not created"
    fail_count=$((fail_count + 1))
fi
var_ROOT_DIR="${previous_root}"
var_DATA_DIR="${previous_data}"

if [[ ${fail_count} -gt 0 ]]; then
    log::error "OpenCode unit tests failed (${fail_count} failure(s))"
    exit 1
fi

log::success "OpenCode unit tests passed (${pass_count} checks)"
