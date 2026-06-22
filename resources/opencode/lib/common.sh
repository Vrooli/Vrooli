#!/bin/bash

# Core helpers for the OpenCode AI CLI resource (official binary integration).
#
# Source of truth: the raw `opencode` binary reads its config from the
# default XDG location (~/.config/opencode/opencode.json) and its auth from
# ~/.local/share/opencode/auth.json. The Go `permissions` adapter
# (resource-opencode permissions …) writes the same opencode.json, and
# agent-manager invokes the raw binary directly — so every surface shares
# one config/auth location. This file owns only the install-time and
# config-write helpers reachable from lib/install.sh and the resource test
# phase; the legacy `run`/server/SSE-permission wrappers were deleted when
# direct-invocation became the contract.
set -euo pipefail

VROOLI_ROOT="${VROOLI_ROOT:-${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}}"
OPENCODE_DIR="${VROOLI_ROOT}/resources/opencode"

# Logging utilities (needed when common.sh is sourced standalone).
source "${VROOLI_ROOT}/scripts/lib/utils/log.sh"
source "${VROOLI_ROOT}/scripts/lib/service/secrets.sh" 2>/dev/null || true

# Load defaults (default provider/model slugs).
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

# Secrets loading cache flag.
OPENCODE_SECRETS_LOADED=${OPENCODE_SECRETS_LOADED:-0}

opencode::secret_value_invalid() {
    local value="${1:-}"
    if [[ -z "${value}" ]]; then
        return 0
    fi
    if [[ "${value}" == auto-null-* ]]; then
        return 0
    fi
    if [[ "${value}" == *"[ERROR]"* ]] || [[ "${value}" == *"Failed to retrieve secret"* ]] || [[ "${value}" == *"❌"* ]]; then
        return 0
    fi
    return 1
}

opencode::secret_value_valid() {
    ! opencode::secret_value_invalid "$1"
}

# opencode::openrouter::key_usable reports whether a resolved OpenRouter key
# looks like a real credential rather than a placeholder or truncated value.
# OpenRouter keys are `sk-or-...` and far longer than the short placeholders
# the secrets backend can return (e.g. a 14-char stub), so we gate on the
# prefix plus a conservative length floor. A key that fails this is treated as
# "no auth" by ensure_config so the provider self-heal can fall back to Ollama.
opencode::openrouter::key_usable() {
    local key="${1:-}"
    [[ -n "${key}" ]] || return 1
    opencode::secret_value_valid "${key}" || return 1
    [[ "${key}" == sk-or-* ]] || return 1
    [[ "${#key}" -ge 40 ]] || return 1
    return 0
}

opencode::sanitize_secret_var() {
    local var_name="${1:-}" label="${2:-${1:-secret}}"
    if [[ -z "${var_name}" ]]; then
        return 0
    fi
    local current_value="${!var_name:-}"
    if [[ -z "${current_value}" ]]; then
        return 0
    fi
    if opencode::secret_value_invalid "${current_value}"; then
        log::warning "Ignoring invalid ${label} value from secrets backend"
        unset "${var_name}"
        return 0
    fi
    return 0
}

opencode::ensure_dirs() {
    mkdir -p "${OPENCODE_CONFIG_DIR}"
    mkdir -p "${OPENCODE_DATA_DIR}"
    mkdir -p "${OPENCODE_LOG_DIR}"
    mkdir -p "${OPENCODE_BIN_DIR}"
}

opencode::load_secrets() {
    if [[ "${OPENCODE_SECRETS_LOADED}" -eq 1 ]]; then
        return 0
    fi

    if command -v resource-vault &>/dev/null; then
        local export_cmd
        if export_cmd=$(resource-vault secrets export opencode 2>/dev/null); then
            if [[ -n "${export_cmd}" ]]; then
                eval "${export_cmd}"
            fi
        fi
        if [[ -z "${OPENROUTER_API_KEY:-}" ]]; then
            if export_cmd=$(resource-vault secrets export openrouter 2>/dev/null); then
                if [[ -n "${export_cmd}" ]]; then
                    eval "${export_cmd}"
                fi
            fi
        fi
    fi
    opencode::sanitize_secret_var OPENROUTER_API_KEY "OpenRouter API key"

    if [[ -z "${OPENROUTER_API_KEY:-}" || "${OPENROUTER_API_KEY}" == auto-null-* ]] || opencode::secret_value_invalid "${OPENROUTER_API_KEY:-}"; then
        if ! declare -f secrets::resolve >/dev/null 2>&1; then
            if [[ -f "${VROOLI_ROOT}/scripts/lib/service/secrets.sh" ]]; then
                # shellcheck disable=SC1091
                source "${VROOLI_ROOT}/scripts/lib/service/secrets.sh"
            fi
        fi
        if declare -f secrets::resolve >/dev/null 2>&1; then
            local resolved_key=""
            resolved_key=$(secrets::resolve "OPENROUTER_API_KEY" 2>/dev/null || true)
            if [[ -z "${resolved_key}" || "${resolved_key}" == auto-null-* ]]; then
                resolved_key=$(secrets::resolve "openrouter_api_key" "resources/openrouter/api/main" 2>/dev/null || true)
            fi
            if [[ -n "${resolved_key}" && "${resolved_key}" != auto-null-* ]]; then
                export OPENROUTER_API_KEY="${resolved_key}"
            fi
        fi
    fi

    local vars=(
        OPENROUTER_API_KEY
        CLOUDFLARE_API_TOKEN
        CLOUDFLARE_ACCOUNT_ID
        CLOUDFLARE_AI_GATEWAY_SLUG
    )
    for var_name in "${vars[@]}"; do
        if [[ -z "${!var_name:-}" ]] || opencode::secret_value_invalid "${!var_name:-}"; then
            local value=""
            if declare -f secrets::read_project_secret >/dev/null 2>&1; then
                value=$(secrets::read_project_secret "${var_name}" 2>/dev/null || true)
            fi
            if [[ -n "${value}" ]] && opencode::secret_value_valid "${value}"; then
                export "${var_name}"="${value}"
            fi
        fi
    done

    if [[ -z "${OPENROUTER_API_KEY:-}" || "${OPENROUTER_API_KEY}" == auto-null-* ]] || opencode::secret_value_invalid "${OPENROUTER_API_KEY:-}"; then
        local credentials_file="${OPENROUTER_CREDENTIALS_FILE:-${opencode_xdg_config_home}/vrooli/resources/openrouter/openrouter-credentials.json}"
        if [[ -f "${credentials_file}" ]]; then
            local credential_key
            credential_key=$(jq -r '.data.apiKey // empty' "${credentials_file}" 2>/dev/null || true)
            if [[ -n "${credential_key}" && "${credential_key}" != "null" && "${credential_key}" != auto-null-* ]] && opencode::secret_value_valid "${credential_key}"; then
                export OPENROUTER_API_KEY="${credential_key}"
            fi
        fi
    fi

    if [[ -z "${OPENROUTER_API_KEY:-}" || "${OPENROUTER_API_KEY}" == auto-null-* ]] || opencode::secret_value_invalid "${OPENROUTER_API_KEY:-}"; then
        local openrouter_core="${VROOLI_ROOT}/resources/openrouter/lib/core.sh"
        if [[ -f "${openrouter_core}" ]]; then
            # shellcheck disable=SC1090
            source "${openrouter_core}" 2>/dev/null || true
            if declare -f openrouter::get_api_key >/dev/null 2>&1; then
                local derived_key
                derived_key=$(openrouter::get_api_key 2>/dev/null || true)
                if [[ -z "${derived_key}" || "${derived_key}" == auto-null-* ]]; then
                    if openrouter::init >/dev/null 2>&1; then
                        derived_key="${OPENROUTER_API_KEY:-}"
                    fi
                fi
                if [[ -n "${derived_key}" && "${derived_key}" != auto-null-* ]]; then
                    export OPENROUTER_API_KEY="${derived_key}"
                fi
            fi
        fi
    fi

    opencode::auth::sync_openrouter

    opencode::sanitize_secret_var OPENROUTER_API_KEY "OpenRouter API key"
    opencode::sanitize_secret_var CLOUDFLARE_API_TOKEN "Cloudflare API token"
    opencode::sanitize_secret_var CLOUDFLARE_ACCOUNT_ID "Cloudflare account ID"
    opencode::sanitize_secret_var CLOUDFLARE_AI_GATEWAY_SLUG "Cloudflare gateway slug"

    OPENCODE_SECRETS_LOADED=1
    return 0
}

# opencode::auth::sync_openrouter writes the resolved OpenRouter key into
# the auth store raw `opencode` reads (~/.local/share/opencode/auth.json),
# merging into any existing providers. Idempotent.
opencode::auth::sync_openrouter() {
    local key="${OPENROUTER_API_KEY:-}"
    if ! opencode::openrouter::key_usable "${key}"; then
        return 0
    fi

    local auth_dir
    auth_dir="$(dirname "${OPENCODE_AUTH_FILE}")"
    mkdir -p "${auth_dir}"

    local tmp
    tmp=$(mktemp "${TMPDIR:-/tmp}/opencode-auth.XXXXXX")

    if command -v jq >/dev/null 2>&1 && [[ -f "${OPENCODE_AUTH_FILE}" ]]; then
        if jq \
            --arg key "${key}" \
            '.openrouter = {"type":"api","key":$key}' \
            "${OPENCODE_AUTH_FILE}" >"${tmp}" 2>/dev/null; then
            mv "${tmp}" "${OPENCODE_AUTH_FILE}"
            chmod 600 "${OPENCODE_AUTH_FILE}" 2>/dev/null || true
            return 0
        fi
    fi

    cat <<EOF >"${tmp}"
{
  "openrouter": {
    "type": "api",
    "key": "${key}"
  }
}
EOF
    mv "${tmp}" "${OPENCODE_AUTH_FILE}"
    chmod 600 "${OPENCODE_AUTH_FILE}" 2>/dev/null || true
}

# opencode::ollama::base_url normalizes OLLAMA_HOST into a full
# scheme://host:port URL. OLLAMA_HOST follows Ollama's own convention and
# may be a bare host ("localhost"), host:port, or a full URL; a missing
# port defaults to 11434, a missing scheme to http.
opencode::ollama::base_url() {
    local raw="${OLLAMA_HOST:-localhost:11434}"
    local scheme="http://"
    if [[ "${raw}" == https://* ]]; then
        scheme="https://"
        raw="${raw#https://}"
    elif [[ "${raw}" == http://* ]]; then
        raw="${raw#http://}"
    fi
    local hostport="${raw%%/*}"
    if [[ "${hostport}" != *:* ]]; then
        hostport="${hostport}:11434"
    fi
    printf '%s%s' "${scheme}" "${hostport}"
}

# opencode::ollama::reachable probes a local Ollama daemon. Used to decide
# whether to pre-configure Ollama as a keyless local provider.
opencode::ollama::reachable() {
    command -v curl >/dev/null 2>&1 || return 1
    curl -fsS --max-time 2 "$(opencode::ollama::base_url)/api/tags" >/dev/null 2>&1
}

# opencode::ollama::supports_tools reports whether a pulled Ollama model
# actually returns STRUCTURED tool calls rather than narrating a tool call as
# plain text. Ollama advertises a "tools" capability for models whose chat
# template cannot be parsed back into tool_calls on a given runtime (e.g.
# qwen2.5-coder on older Ollama), so the capability flag alone is unreliable —
# the only sound signal is a live probe. Best-effort and fail-OPEN: if the
# probe cannot run (no curl/jq, daemon error, empty reply) it returns success
# so discovery is never blocked on the probe itself. It returns failure only
# when the daemon answered AND no structured tool_calls came back.
opencode::ollama::supports_tools() {
    local model="$1" base body resp
    command -v jq >/dev/null 2>&1 || return 0
    base="$(opencode::ollama::base_url)"
    body="$(jq -cn --arg m "${model}" '{
        model: $m,
        stream: false,
        messages: [{role: "user", content: "Create a file named probe.txt containing the text ok. Use the write tool."}],
        tools: [{type: "function", function: {name: "write", description: "write content to a file", parameters: {type: "object", properties: {filePath: {type: "string"}, content: {type: "string"}}, required: ["filePath", "content"]}}}]
    }')" || return 0
    resp="$(curl -fsS --max-time 30 "${base}/v1/chat/completions" -H 'Content-Type: application/json' -d "${body}" 2>/dev/null || true)"
    [[ -z "${resp}" ]] && return 0
    printf '%s' "${resp}" | jq -e '(.choices[0].message.tool_calls // []) | length > 0' >/dev/null 2>&1
}

# opencode::ollama::pick_model echoes a usable local chat model id from the
# reachable Ollama daemon. It honours OPENCODE_OLLAMA_DEFAULT_MODEL when that
# model is pulled and tool-capable, otherwise prefers a tool-calling coder
# model (newest families first), then a general chat model, excluding
# embedding-only models that cannot serve a chat. Each candidate is probed for
# STRUCTURED tool calls before selection so a model that only narrates tool
# calls (e.g. qwen2.5-coder on an old runtime) is skipped rather than silently
# chosen. Falls back to the configured default when discovery is unavailable.
opencode::ollama::pick_model() {
    local preferred="${OPENCODE_OLLAMA_DEFAULT_MODEL:-qwen3-coder}"
    local tags names
    if ! command -v jq >/dev/null 2>&1; then
        printf '%s' "${preferred}"
        return 0
    fi
    tags="$(curl -fsS --max-time 3 "$(opencode::ollama::base_url)/api/tags" 2>/dev/null || true)"
    if [[ -z "${tags}" ]]; then
        printf '%s' "${preferred}"
        return 0
    fi
    names="$(printf '%s' "${tags}" | jq -r '.models[].name' 2>/dev/null)"
    # Drop embedding-only models — they can't serve chat/completions.
    names="$(printf '%s\n' "${names}" | grep -viE 'embed|minilm|bge' || true)"
    if [[ -z "${names}" ]]; then
        printf '%s' "${preferred}"
        return 0
    fi
    # Honour the configured default when it is present AND tool-capable.
    if grep -qxF "${preferred}" <<<"${names}" && opencode::ollama::supports_tools "${preferred}"; then
        printf '%s' "${preferred}"
        return 0
    fi
    # Prefer tool-calling-capable models, newest coder families first. Demote
    # qwen2.5-coder/coder beneath proven tool-callers; the probe is the real
    # gate, the order is just the search priority.
    local pref match
    for pref in 'qwen3-coder' 'qwen3' 'llama3.1' 'llama3' 'mistral' 'qwen2.5-coder' 'coder' 'qwen' 'phi'; do
        match="$(grep -m1 -iF "${pref}" <<<"${names}" || true)"
        if [[ -n "${match}" ]] && opencode::ollama::supports_tools "${match}"; then
            printf '%s' "${match}"
            return 0
        fi
    done
    # Last resort: first tool-capable model, else the first chat model.
    local name
    while IFS= read -r name; do
        [[ -z "${name}" ]] && continue
        if opencode::ollama::supports_tools "${name}"; then
            printf '%s' "${name}"
            return 0
        fi
    done <<<"${names}"
    printf '%s' "$(head -n1 <<<"${names}")"
}

opencode::default_config_payload() {
    local provider="${1:-${OPENCODE_DEFAULT_PROVIDER}}"
    local chat_model="${2:-${OPENCODE_DEFAULT_CHAT_MODEL}}"
    local completion_model="${3:-${OPENCODE_DEFAULT_COMPLETION_MODEL}}"

    cat <<EOF
{
  "\$schema": "https://opencode.ai/config.json",
  "model": "${provider}/${chat_model}",
  "small_model": "${provider}/${completion_model}",
  "instructions": [
    "AGENTS.md"
  ]
}
EOF
}

# opencode::ensure_config guarantees ~/.config/opencode/opencode.json carries
# a usable model, preserving any permission.bash map and unknown keys already
# written by the Go permissions adapter. It is the provider self-heal:
#
#   - No OpenRouter key + local Ollama reachable → use Ollama (keyless local
#     provider) as the model, AND re-point an existing dead OpenRouter/empty
#     model to Ollama (drift self-heal), not just fresh configs.
#   - Provider needs a key we can't resolve + Ollama NOT reachable → keep the
#     config but emit a loud warning naming the missing key, so the failure is
#     diagnosable instead of silent.
#
# A model already pinned to a working local (ollama/*) provider is left alone.
# Re-runnable (idempotent).
opencode::ensure_config() {
    mkdir -p "${OPENCODE_CONFIG_DIR}"

    local have_openrouter=0
    if opencode::openrouter::key_usable "${OPENROUTER_API_KEY:-}"; then
        have_openrouter=1
    fi

    local provider="${OPENCODE_DEFAULT_PROVIDER}"
    local chat_model="${OPENCODE_DEFAULT_CHAT_MODEL}"
    local completion_model="${OPENCODE_DEFAULT_COMPLETION_MODEL}"
    local use_ollama=0
    if [[ "${have_openrouter}" -eq 0 ]] && opencode::ollama::reachable; then
        use_ollama=1
        provider="ollama"
        chat_model="$(opencode::ollama::pick_model)"
        completion_model="${chat_model}"
    fi

    if [[ ! -f "${OPENCODE_CONFIG_FILE}" ]]; then
        log::info "Creating default OpenCode config at ${OPENCODE_CONFIG_FILE}"
        opencode::default_config_payload "${provider}" "${chat_model}" "${completion_model}" >"${OPENCODE_CONFIG_FILE}"
        if [[ "${use_ollama}" -eq 1 ]]; then
            opencode::config::ensure_ollama_provider "${OPENCODE_CONFIG_FILE}" "${chat_model}" "${completion_model}"
        fi
        return 0
    fi

    opencode::config::migrate_legacy_models "${OPENCODE_CONFIG_FILE}"

    command -v jq >/dev/null 2>&1 || return 0

    local current_model current_provider
    current_model="$(jq -r '.model // ""' "${OPENCODE_CONFIG_FILE}" 2>/dev/null || true)"
    current_provider="${current_model%%/*}"

    # Decide whether to re-point the active model. We only manage the cloud
    # default (openrouter) and an empty/missing model; a model already pinned
    # to ollama/* or another provider the operator chose is left untouched.
    local repoint=0
    if [[ "${use_ollama}" -eq 1 && ( -z "${current_provider}" || "${current_provider}" == "openrouter" ) ]]; then
        repoint=1
    fi

    local tmp
    tmp=$(mktemp "${TMPDIR:-/tmp}/opencode-config.XXXXXX")
    if [[ "${repoint}" -eq 1 ]]; then
        # Self-heal: force model/small_model onto the reachable local provider.
        jq \
            --arg schema "https://opencode.ai/config.json" \
            --arg model "${provider}/${chat_model}" \
            --arg small "${provider}/${completion_model}" \
            '.["$schema"] = (.["$schema"] // $schema)
             | .model = $model
             | .small_model = $small
             | .instructions = (.instructions // ["AGENTS.md"])' \
            "${OPENCODE_CONFIG_FILE}" >"${tmp}" 2>/dev/null || { rm -f "${tmp}"; tmp=""; }
    else
        # No drift action: only fill in absent defaults without clobbering.
        jq \
            --arg schema "https://opencode.ai/config.json" \
            --arg model "${provider}/${chat_model}" \
            --arg small "${provider}/${completion_model}" \
            '.["$schema"] = (.["$schema"] // $schema)
             | .model = (.model // $model)
             | .small_model = (.small_model // $small)
             | .instructions = (.instructions // ["AGENTS.md"])' \
            "${OPENCODE_CONFIG_FILE}" >"${tmp}" 2>/dev/null || { rm -f "${tmp}"; tmp=""; }
    fi
    if [[ -n "${tmp}" ]]; then
        if ! cmp -s "${OPENCODE_CONFIG_FILE}" "${tmp}"; then
            mv "${tmp}" "${OPENCODE_CONFIG_FILE}"
            if [[ "${repoint}" -eq 1 ]]; then
                log::info "Self-healed OpenCode model → ${provider}/${chat_model} (no OpenRouter key; local Ollama reachable)"
            else
                log::info "Merged default model into ${OPENCODE_CONFIG_FILE}"
            fi
        else
            rm -f "${tmp}"
        fi
    fi

    if [[ "${use_ollama}" -eq 1 ]]; then
        opencode::config::ensure_ollama_provider "${OPENCODE_CONFIG_FILE}" "${chat_model}" "${completion_model}"
    fi

    # Loud warning when the active provider needs a key we can't resolve and
    # there is no local fallback — otherwise the failure is silent until a run.
    if [[ "${use_ollama}" -eq 0 && "${current_provider}" == "openrouter" && "${have_openrouter}" -eq 0 ]]; then
        log::warning "OpenCode model '${current_model}' uses OpenRouter but no OPENROUTER_API_KEY is resolvable and no local Ollama is reachable — runs will fail. Set the key (resource-vault) or start Ollama."
    fi
}

# opencode::config::ensure_ollama_provider writes an OpenAI-compatible
# Ollama provider block (npm @ai-sdk/openai-compatible, baseURL
# http://localhost:11434/v1) into opencode.json, declaring the requested
# models so `opencode run -m ollama/<model>` resolves. Idempotent: only
# writes when the block is absent or differs.
opencode::config::ensure_ollama_provider() {
    local config_path="${1:-${OPENCODE_CONFIG_FILE}}"
    local chat_model="${2:-${OPENCODE_OLLAMA_DEFAULT_MODEL}}"
    local completion_model="${3:-${chat_model}}"
    command -v jq >/dev/null 2>&1 || return 0
    [[ -f "${config_path}" ]] || return 0

    local base_url
    base_url="$(opencode::ollama::base_url)/v1"

    local tmp
    tmp=$(mktemp "${TMPDIR:-/tmp}/opencode-config.XXXXXX")
    if jq \
        --arg base "${base_url}" \
        --arg chat "${chat_model}" \
        --arg small "${completion_model}" \
        '.provider = (.provider // {})
         | .provider.ollama = (.provider.ollama // {})
         | .provider.ollama.npm = "@ai-sdk/openai-compatible"
         | .provider.ollama.name = "Ollama (local)"
         | .provider.ollama.options = (.provider.ollama.options // {})
         | .provider.ollama.options.baseURL = $base
         | .provider.ollama.models = (.provider.ollama.models // {})
         | .provider.ollama.models[$chat] = (.provider.ollama.models[$chat] // {})
         | .provider.ollama.models[$small] = (.provider.ollama.models[$small] // {})' \
        "${config_path}" >"${tmp}" 2>/dev/null; then
        if ! cmp -s "${config_path}" "${tmp}"; then
            mv "${tmp}" "${config_path}"
            log::info "Configured local Ollama provider in ${config_path}"
        else
            rm -f "${tmp}"
        fi
    else
        rm -f "${tmp}"
    fi
}

opencode::config::migrate_legacy_models() {
    local config_path="${1:-${OPENCODE_CONFIG_FILE}}"
    if [[ ! -f "${config_path}" ]]; then
        return 0
    fi

    local legacy_slug_short="openrouter/qwen3-coder"
    local legacy_slug_full="openrouter/qwen/qwen3-coder"
    local target_chat="openrouter/${OPENCODE_DEFAULT_CHAT_MODEL}"
    local target_small="openrouter/${OPENCODE_DEFAULT_COMPLETION_MODEL}"
    local updated=0

    if command -v jq >/dev/null 2>&1; then
        local tmp
        tmp=$(mktemp "${TMPDIR:-/tmp}/opencode-config.XXXXXX")
        if jq \
            --arg legacy_short "${legacy_slug_short}" \
            --arg legacy_full "${legacy_slug_full}" \
            --arg chat "${target_chat}" \
            --arg small "${target_small}" \
            'if (.model // "") == $legacy_short or (.model // "") == $legacy_full then .model = $chat else . end
             | if (.small_model // "") == $legacy_short or (.small_model // "") == $legacy_full then .small_model = $small else . end' \
            "${config_path}" >"${tmp}" 2>/dev/null; then
            if ! cmp -s "${config_path}" "${tmp}"; then
                mv "${tmp}" "${config_path}"
                updated=1
            else
                rm -f "${tmp}"
            fi
        else
            rm -f "${tmp}"
        fi
    fi

    if [[ ${updated} -eq 1 ]]; then
        log::info "Updated OpenCode default model slugs in ${config_path}"
    fi
}
