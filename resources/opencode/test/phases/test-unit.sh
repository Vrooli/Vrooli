#!/usr/bin/env bash
# OpenCode Resource Unit Tests - config-write & secret-resolution sanity checks

set -euo pipefail

VROOLI_ROOT="${VROOLI_ROOT:-${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}}"
OPENCODE_DIR="${VROOLI_ROOT}/resources/opencode"

# Run hermetically: point XDG at a throwaway dir so config/auth writes never
# touch the operator's real ~/.config / ~/.local/share.
opencode_test_xdg=$(mktemp -d)
trap 'rm -rf "${opencode_test_xdg}"' EXIT
export XDG_CONFIG_HOME="${opencode_test_xdg}/config"
export XDG_DATA_HOME="${opencode_test_xdg}/data"

# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh"
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

default_payload="$(opencode::default_config_payload)"
assert_equals "1" "$(grep -c '"model": "openrouter/x-ai/grok-code-fast-1"' <(printf '%s' "${default_payload}"))" "Default config uses updated chat model slug"
assert_equals "1" "$(grep -c '"small_model": "openrouter/x-ai/grok-code-fast-1"' <(printf '%s' "${default_payload}"))" "Default config uses updated completion model slug"

tmp_config=$(mktemp)
cat <<'EOF' >"${tmp_config}"
{
  "model": "openrouter/qwen/qwen3-coder",
  "small_model": "openrouter/qwen/qwen3-coder"
}
EOF
opencode::config::migrate_legacy_models "${tmp_config}"

assert_equals "openrouter/x-ai/grok-code-fast-1" "$(jq -r '.model' "${tmp_config}")" "Migrates legacy chat model slug"
assert_equals "openrouter/x-ai/grok-code-fast-1" "$(jq -r '.small_model' "${tmp_config}")" "Migrates legacy completion model slug"
rm -f "${tmp_config}"

# ensure_config must merge a default model into a pre-existing file that only
# carries a permission.bash map (as written by the Go permissions adapter)
# without clobbering the permissions.
mkdir -p "${OPENCODE_CONFIG_DIR}"
cat <<'EOF' >"${OPENCODE_CONFIG_FILE}"
{
  "permission": {
    "bash": {
      "git stash*": "deny"
    }
  }
}
EOF
OPENROUTER_API_KEY="sk-or-v1-testkey" opencode::ensure_config
assert_equals "deny" "$(jq -r '.permission.bash["git stash*"]' "${OPENCODE_CONFIG_FILE}")" "ensure_config preserves existing permission.bash entries"
assert_equals "openrouter/x-ai/grok-code-fast-1" "$(jq -r '.model' "${OPENCODE_CONFIG_FILE}")" "ensure_config merges default model into existing config"
rm -f "${OPENCODE_CONFIG_FILE}"

# Ollama host normalization: a bare host must default to port 11434.
assert_equals "http://localhost:11434" "$(OLLAMA_HOST=localhost opencode::ollama::base_url)" "Bare OLLAMA_HOST gets default port"
assert_equals "http://127.0.0.1:11500" "$(OLLAMA_HOST=127.0.0.1:11500 opencode::ollama::base_url)" "host:port OLLAMA_HOST preserved"
assert_equals "https://ollama.example.com:443" "$(OLLAMA_HOST=https://ollama.example.com:443 opencode::ollama::base_url)" "Full URL OLLAMA_HOST preserved"
assert_equals "http://localhost:11434" "$(unset OLLAMA_HOST; opencode::ollama::base_url)" "Unset OLLAMA_HOST defaults to localhost:11434"

# ensure_config writes a usable Ollama provider (with models map) when no
# OpenRouter key is present and the daemon is reachable.
ollama_cfg="${OPENCODE_CONFIG_DIR}/opencode.json"
rm -f "${ollama_cfg}"
if OLLAMA_HOST=localhost opencode::ollama::reachable; then
    (unset OPENROUTER_API_KEY; OPENCODE_OLLAMA_DEFAULT_MODEL="qwen3:1.7b" opencode::ensure_config)
    assert_equals "ollama/qwen3:1.7b" "$(jq -r '.model' "${ollama_cfg}")" "ensure_config selects Ollama model when no key + daemon reachable"
    assert_equals "@ai-sdk/openai-compatible" "$(jq -r '.provider.ollama.npm' "${ollama_cfg}")" "ensure_config writes Ollama provider npm"
    assert_equals "http://localhost:11434/v1" "$(jq -r '.provider.ollama.options.baseURL' "${ollama_cfg}")" "ensure_config writes Ollama baseURL with port"
    assert_equals "{}" "$(jq -c '.provider.ollama.models["qwen3:1.7b"]' "${ollama_cfg}")" "ensure_config declares Ollama model in models map"
else
    log::info "Ollama not reachable on this host; skipping live Ollama-provider assertions"
fi
rm -f "${ollama_cfg}"

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
# Point the credential-file fallback at our fixture so resolution is
# deterministic regardless of any real vault on the host.
export OPENROUTER_CREDENTIALS_FILE="${tmp_root}/data/credentials/openrouter-credentials.json"
previous_root="${var_ROOT_DIR:-}"
previous_data="${var_DATA_DIR:-}"
var_ROOT_DIR="${tmp_root}"
var_DATA_DIR="${tmp_root}/data"
opencode::load_secrets
unset OPENROUTER_CREDENTIALS_FILE
rm -rf "${tmp_root}"
if [[ "${OPENROUTER_API_KEY:-}" == auto-null-* || -z "${OPENROUTER_API_KEY:-}" ]]; then
    log::error "Failed to resolve OpenRouter API key"
    fail_count=$((fail_count + 1))
else
    log::success "Resolves OpenRouter API key from fallback sources"
    pass_count=$((pass_count + 1))
fi

if [[ -f "${OPENCODE_AUTH_FILE}" ]]; then
    stored_key=$(jq -r '.openrouter.key // empty' "${OPENCODE_AUTH_FILE}")
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
