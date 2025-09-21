#!/bin/bash

# Core helpers for the OpenCode AI CLI resource (official binary integration)
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENCODE_DIR="${APP_ROOT}/resources/opencode"

# Load defaults
source "${OPENCODE_DIR}/config/defaults.sh"

# Data directories (var_DATA_DIR is exported by the global resource runtime)
OPENCODE_DATA_DIR="${var_DATA_DIR:-${APP_ROOT}/data}/opencode"
OPENCODE_BIN_DIR="${OPENCODE_DATA_DIR}/bin"
OPENCODE_BIN="${OPENCODE_BIN_DIR}/opencode"
OPENCODE_CACHE_DIR="${OPENCODE_DATA_DIR}/cache"
OPENCODE_LOG_DIR="${OPENCODE_DATA_DIR}/logs"
OPENCODE_VERSION_FILE="${OPENCODE_DATA_DIR}/VERSION"

# Config paths (we pin them via env vars before invoking the CLI)
OPENCODE_CONFIG_DIR="${OPENCODE_DATA_DIR}/config"
OPENCODE_CONFIG_FILE="${OPENCODE_CONFIG_DIR}/opencode.json"
OPENCODE_XDG_CONFIG_HOME="${OPENCODE_DATA_DIR}/xdg-config"
OPENCODE_XDG_DATA_HOME="${OPENCODE_DATA_DIR}/xdg-data"

# Secrets loading cache flag
OPENCODE_SECRETS_LOADED=${OPENCODE_SECRETS_LOADED:-0}

# Server runtime defaults
OPENCODE_SERVER_HOST="${OPENCODE_SERVER_HOST:-127.0.0.1}"
OPENCODE_SERVER_PORT="${OPENCODE_SERVER_PORT:-4096}"
OPENCODE_SERVER_PID_FILE="${OPENCODE_DATA_DIR}/opencode-serve.pid"
OPENCODE_SERVER_LOG_FILE="${OPENCODE_LOG_DIR}/server.log"

# Global API status helpers (exported for callers)
OPENCODE_API_HTTP_CODE=0
OPENCODE_CURL_EXIT_CODE=0
REPLY=""
OPENCODE_EVENT_LISTENER=""

opencode::ensure_dirs() {
    mkdir -p "${OPENCODE_DATA_DIR}"
    mkdir -p "${OPENCODE_BIN_DIR}"
    mkdir -p "${OPENCODE_CACHE_DIR}"
    mkdir -p "${OPENCODE_LOG_DIR}"
    mkdir -p "${OPENCODE_CONFIG_DIR}"
    mkdir -p "${OPENCODE_XDG_CONFIG_HOME}"
    mkdir -p "${OPENCODE_XDG_CONFIG_HOME}/opencode"
    mkdir -p "${OPENCODE_XDG_DATA_HOME}"
    mkdir -p "${OPENCODE_XDG_DATA_HOME}/opencode"
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

    if [[ -z "${OPENROUTER_API_KEY:-}" || "${OPENROUTER_API_KEY}" == auto-null-* ]]; then
        if ! declare -f secrets::resolve >/dev/null 2>&1; then
            if [[ -f "${APP_ROOT}/scripts/lib/service/secrets.sh" ]]; then
                # shellcheck disable=SC1091
                source "${APP_ROOT}/scripts/lib/service/secrets.sh"
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

    if command -v jq &>/dev/null; then
        local vrooli_root="${VROOLI_ROOT:-"$HOME/Vrooli"}"
        local secrets_file="${vrooli_root}/.vrooli/secrets.json"
        if [[ -f "${secrets_file}" ]]; then
            local vars=(
                OPENROUTER_API_KEY
                CLOUDFLARE_API_TOKEN
                CLOUDFLARE_ACCOUNT_ID
                CLOUDFLARE_AI_GATEWAY_SLUG
            )
            for var_name in "${vars[@]}"; do
                if [[ -z "${!var_name:-}" ]]; then
                    local value
                    value=$(jq -r --arg key "${var_name}" '.[$key] // empty' "${secrets_file}" 2>/dev/null)
                    if [[ -n "${value}" && "${value}" != "null" ]]; then
                        export "${var_name}"="${value}"
                    fi
                fi
            done
        fi
    fi

    if [[ -z "${OPENROUTER_API_KEY:-}" || "${OPENROUTER_API_KEY}" == auto-null-* ]]; then
        local credentials_file="${var_ROOT_DIR:-${APP_ROOT}}/data/credentials/openrouter-credentials.json"
        if [[ -f "${credentials_file}" ]]; then
            local credential_key
            credential_key=$(jq -r '.data.apiKey // empty' "${credentials_file}" 2>/dev/null || true)
            if [[ -n "${credential_key}" && "${credential_key}" != "null" && "${credential_key}" != auto-null-* ]]; then
                export OPENROUTER_API_KEY="${credential_key}"
            fi
        fi
    fi

    if [[ -z "${OPENROUTER_API_KEY:-}" || "${OPENROUTER_API_KEY}" == auto-null-* ]]; then
        local openrouter_core="${APP_ROOT}/resources/openrouter/lib/core.sh"
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

    OPENCODE_SECRETS_LOADED=1
    return 0
}

opencode::auth::sync_openrouter() {
    local key="${OPENROUTER_API_KEY:-}"
    if [[ -z "${key}" || "${key}" == auto-null-* ]]; then
        return 0
    fi

    local auth_dir="${OPENCODE_XDG_DATA_HOME}/opencode"
    local auth_file="${auth_dir}/auth.json"
    mkdir -p "${auth_dir}"

    local tmp
    tmp=$(mktemp "${TMPDIR:-/tmp}/opencode-auth.XXXXXX")

    if command -v jq >/dev/null 2>&1 && [[ -f "${auth_file}" ]]; then
        if jq \
            --arg key "${key}" \
            '.openrouter = {"type":"api","key":$key}' \
            "${auth_file}" >"${tmp}" 2>/dev/null; then
            mv "${tmp}" "${auth_file}"
            chmod 600 "${auth_file}" 2>/dev/null || true
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
    mv "${tmp}" "${auth_file}"
    chmod 600 "${auth_file}" 2>/dev/null || true
}

opencode::supported_provider() {
    local provider="${1:-}"
    for candidate in "${OPENCODE_SUPPORTED_PROVIDERS[@]}"; do
        if [[ "${candidate}" == "${provider}" ]]; then
            return 0
        fi
    done
    return 1
}

opencode::ensure_cli() {
    if [[ ! -x "${OPENCODE_BIN}" ]]; then
        log::error "OpenCode CLI binary not found at ${OPENCODE_BIN}"
        log::info "Run 'resource-opencode manage install' to download the official CLI."
        return 1
    fi
    return 0
}

opencode::export_runtime_env() {
    export OPENCODE_CONFIG="${OPENCODE_CONFIG_FILE}"
    export XDG_CONFIG_HOME="${OPENCODE_XDG_CONFIG_HOME}"
    export XDG_DATA_HOME="${OPENCODE_XDG_DATA_HOME}"
    export XDG_CACHE_HOME="${OPENCODE_CACHE_DIR}"
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

opencode::ensure_config() {
    if [[ -f "${OPENCODE_CONFIG_FILE}" ]]; then
        opencode::config::migrate_legacy_models "${OPENCODE_CONFIG_FILE}"
        return 0
    fi
    log::info "Creating default OpenCode config at ${OPENCODE_CONFIG_FILE}"
    mkdir -p "${OPENCODE_CONFIG_DIR}"
    opencode::default_config_payload >"${OPENCODE_CONFIG_FILE}"
}

opencode::config::migrate_legacy_models() {
    local config_path="${1:-${OPENCODE_CONFIG_FILE}}"
    if [[ ! -f "${config_path}" ]]; then
        return 0
    fi

    local legacy_slug="openrouter/qwen3-coder"
    local target_chat="openrouter/${OPENCODE_DEFAULT_CHAT_MODEL}"
    local target_small="openrouter/${OPENCODE_DEFAULT_COMPLETION_MODEL}"
    local updated=0

    if command -v jq >/dev/null 2>&1; then
        local tmp
        tmp=$(mktemp "${TMPDIR:-/tmp}/opencode-config.XXXXXX")
        if jq \
            --arg legacy "${legacy_slug}" \
            --arg chat "${target_chat}" \
            --arg small "${target_small}" \
            'if (.model // "") == $legacy then .model = $chat else . end
             | if (.small_model // "") == $legacy then .small_model = $small else . end' \
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
    else
        if grep -q "${legacy_slug}" "${config_path}"; then
            local tmp
            tmp=$(mktemp "${TMPDIR:-/tmp}/opencode-config.XXXXXX")
            sed \
                -e "s#\"model\"[[:space:]]*:[[:space:]]*\"${legacy_slug}\"#\"model\": \"${target_chat}\"#" \
                -e "s#\"small_model\"[[:space:]]*:[[:space:]]*\"${legacy_slug}\"#\"small_model\": \"${target_small}\"#" \
                "${config_path}" >"${tmp}" && mv "${tmp}" "${config_path}" && updated=1 || rm -f "${tmp}"
        fi
    fi

    if [[ ${updated} -eq 1 ]]; then
        log::info "Updated OpenCode default model slugs in ${config_path}"
    fi
}

opencode::run_cli() {
    opencode::ensure_dirs
    opencode::ensure_cli || return 1
    opencode::ensure_config
    opencode::load_secrets || true
    opencode::export_runtime_env

    PATH="${OPENCODE_BIN_DIR}:${PATH}" \
    OPENCODE_LOG_DIR="${OPENCODE_LOG_DIR}" \
        "${OPENCODE_BIN}" "$@"
}

opencode::require_config() {
    if [[ ! -f "${OPENCODE_CONFIG_FILE}" ]]; then
        log::error "OpenCode configuration not found. Run 'resource-opencode manage install' first."
        return 1
    fi
    return 0
}

################################################################################
# Server helpers (align with https://opencode.ai/docs/server/)
################################################################################

opencode::server::_trim_pid() {
    if [[ -f "${OPENCODE_SERVER_PID_FILE}" ]]; then
        local pid
        pid=$(<"${OPENCODE_SERVER_PID_FILE}")
        pid=${pid//[^0-9]/}
        if [[ -n "${pid}" ]]; then
            printf '%s' "${pid}"
            return 0
        fi
    fi
    return 1
}

opencode::server::pid() {
    opencode::server::_trim_pid || true
}

opencode::server::record_pid() {
    local pid="${1:-}"
    if [[ -n "${pid}" ]]; then
        printf '%s' "${pid}" >"${OPENCODE_SERVER_PID_FILE}"
    fi
}

opencode::server::clear_pid() {
    rm -f "${OPENCODE_SERVER_PID_FILE}" 2>/dev/null || true
}

opencode::server::is_running() {
    local pid
    pid=$(opencode::server::pid) || return 1
    if [[ -z "${pid}" ]]; then
        return 1
    fi
    if kill -0 "${pid}" >/dev/null 2>&1; then
        return 0
    fi
    opencode::server::clear_pid
    return 1
}

opencode::server::base_url() {
    printf 'http://%s:%s' "${OPENCODE_SERVER_HOST}" "${OPENCODE_SERVER_PORT}"
}

opencode::server::wait_until_ready() {
    local timeout="${1:-20}"
    local interval=1
    local elapsed=0
    local url="$(opencode::server::base_url)/doc"

    while [[ ${elapsed} -lt ${timeout} ]]; do
        if curl -fsS --max-time 2 "${url}" >/dev/null 2>&1; then
            return 0
        fi
        sleep "${interval}"
        elapsed=$((elapsed + interval))
    done
    return 1
}

opencode::server::ensure_running() {
    if opencode::server::is_running; then
        return 0
    fi
    if type -t opencode::docker::start >/dev/null 2>&1; then
        opencode::docker::start
        return $?
    fi
    log::error "OpenCode server is not running and start helper is unavailable"
    return 1
}

################################################################################
# API helpers (doc reference: https://opencode.ai/docs/sdk/)
################################################################################

opencode::api::default_directory() {
    if [[ -n "${VROOLI_ROOT:-}" ]]; then
        printf '%s' "${VROOLI_ROOT}"
    else
        printf '%s' "${APP_ROOT}"
    fi
}

opencode::api::build_url() {
    local path="${1:-}"
    local directory="${2:-}"
    if [[ -z "${path}" ]]; then
        return 1
    fi
    local base="$(opencode::server::base_url)${path}"
    if [[ -z "${directory}" ]]; then
        printf '%s' "${base}"
        return 0
    fi
    local encoded
    if command -v jq >/dev/null 2>&1; then
        encoded=$(printf '%s' "${directory}" | jq -sRr @uri 2>/dev/null || printf '%s' "${directory}")
    else
        encoded="${directory}"
    fi
    if [[ "${base}" == *\?* ]]; then
        printf '%s&directory=%s' "${base}" "${encoded}"
    else
        printf '%s?directory=%s' "${base}" "${encoded}"
    fi
}

opencode::api::request_raw() {
    local method="${1:-}" path="${2:-}" payload="${3:-}" directory="${4:-}"
    if [[ -z "${method}" || -z "${path}" ]]; then
        log::error "Method and path are required for API requests"
        return 1
    fi

    local url
    url="$(opencode::api::build_url "${path}" "${directory}")" || return 1

    local curl_args=("--silent" "--show-error" "--no-progress-meter" "-X" "${method}" "${url}" "--write-out" $'\n%{http_code}')
    if [[ -n "${OPENCODE_REQUEST_TIMEOUT_SECS:-}" && "${OPENCODE_REQUEST_TIMEOUT_SECS}" -gt 0 ]]; then
        curl_args+=("--max-time" "${OPENCODE_REQUEST_TIMEOUT_SECS}")
    fi
    if [[ -n "${payload}" ]]; then
        curl_args+=("-H" "Content-Type: application/json" "--data" "${payload}")
    fi

    local raw
    if ! raw=$(curl "${curl_args[@]}"); then
        OPENCODE_CURL_EXIT_CODE=$?
        log::error "API transport failed: ${method} ${path}"
        return 1
    fi
    OPENCODE_CURL_EXIT_CODE=0

    local http_code="${raw##*$'\n'}"
    local body="${raw%$'\n'*}"

    OPENCODE_API_HTTP_CODE="${http_code}"
    REPLY="${body}"

    if [[ "${http_code}" -lt 200 || "${http_code}" -ge 300 ]]; then
        log::error "API request failed (${http_code}): ${method} ${path}"
        if [[ -n "${body}" ]]; then
            if command -v jq >/dev/null 2>&1; then
                printf '%s' "${body}" | jq . >&2 2>/dev/null || printf '%s\n' "${body}" >&2
            else
                printf '%s\n' "${body}" >&2
            fi
        fi
        return 1
    fi

    return 0
}

opencode::api::request() {
    if ! opencode::api::request_raw "$@"; then
        return 1
    fi
    if command -v jq >/dev/null 2>&1; then
        printf '%s' "${REPLY}" | jq .
    else
        printf '%s\n' "${REPLY}"
    fi
}

################################################################################
# Permission helpers & SSE listener
################################################################################

opencode::permissions::normalize_csv() {
    local csv="${1:-}"
    [[ -z "${csv}" ]] && return 0
    IFS=',' read -r -a __opencode_perm_entries <<<"${csv}"
    local cleaned=()
    for entry_raw in "${__opencode_perm_entries[@]}"; do
        local entry
        entry=$(printf '%s' "${entry_raw}" | sed -e 's/^ *//' -e 's/ *$//' | tr '[:upper:]' '[:lower:]')
        if [[ -n "${entry}" ]]; then
            cleaned+=("${entry}")
        fi
    done
    printf '%s\n' "${cleaned[*]}"
}

opencode::permissions::decide() {
    local allowed_csv="${1:-}"
    local skip_flag="${2:-}"
    local tool="${3:-}"
    local patterns_data="${4:-}"

    local tool_lower
    tool_lower=$(printf '%s' "${tool}" | tr '[:upper:]' '[:lower:]')

    case "${skip_flag}" in
        yes|true|on|1)
            printf 'allow\n'
            return 0
            ;;
    esac

    if [[ -z "${allowed_csv}" ]]; then
        printf 'allow\n'
        return 0
    fi

    IFS=',' read -r -a __opencode_allowed_entries <<<"${allowed_csv}"
    for entry_raw in "${__opencode_allowed_entries[@]}"; do
        local entry
        entry=$(printf '%s' "${entry_raw}" | sed -e 's/^ *//' -e 's/ *$//' | tr '[:upper:]' '[:lower:]')
        if [[ -z "${entry}" ]]; then
            continue
        fi
        if [[ "${entry}" == "*" || "${entry}" == "${tool_lower}" ]]; then
            printf 'allow\n'
            return 0
        fi
        if [[ "${entry}" == ${tool_lower}\(*\) ]]; then
            local requested="${entry#${tool_lower}(}"
            requested="${requested%)}"
            if [[ -z "${requested}" ]]; then
                continue
            fi
            if [[ -n "${patterns_data}" ]]; then
                while IFS= read -r pattern_line; do
                    [[ -z "${pattern_line}" ]] && continue
                    local pattern_lower
                    pattern_lower=$(printf '%s' "${pattern_line}" | tr '[:upper:]' '[:lower:]')
                    if [[ "${pattern_lower}" == "${requested}" ]]; then
                        printf 'allow\n'
                        return 0
                    fi
                done <<<"${patterns_data}"
            fi
        fi
    done

    printf 'reject\n'
}

opencode::permissions::patterns_from_payload() {
    local payload="${1:-}"
    if [[ -z "${payload}" ]]; then
        return 0
    fi
    printf '%s' "${payload}" | jq -r '
        (.properties.pattern // empty)
        | if type == "array" then (.[] | tostring)
          elif type == "string" or type == "number" then tostring
          else empty end
    ' 2>/dev/null || true
}

opencode::events::loop() {
    local session_id="${1:-}"
    local directory="${2:-}"
    local allowed_tools="${3:-}"
    local skip_permissions="${4:-}"
    local max_turns="${5:-0}"
    local abort_file="${6:-}"
    local fifo_path="${7:-}"
    local curl_pid="${8:-}"

    trap 'kill "${curl_pid}" 2>/dev/null || true; rm -f "${fifo_path}" 2>/dev/null || true' EXIT

    local turns=0
    declare -A processed_tool_parts=()
    declare -A acknowledged_permissions=()

    while IFS= read -r line; do
        [[ -z "${line}" ]] && continue
        [[ "${line}" != data:* ]] && continue

        local payload="${line#data: }"

        local event_type
        event_type=$(printf '%s' "${payload}" | jq -r '.type // empty' 2>/dev/null || true)
        [[ -z "${event_type}" ]] && continue

        case "${event_type}" in
            permission.updated)
                local permission_id
                permission_id=$(printf '%s' "${payload}" | jq -r '.properties.id // empty' 2>/dev/null || true)
                if [[ -n "${permission_id}" && -n "${acknowledged_permissions[${permission_id}]+x}" ]]; then
                    continue
                fi

                local permission_session
                permission_session=$(printf '%s' "${payload}" | jq -r '.properties.sessionID // empty' 2>/dev/null || true)
                [[ "${permission_session}" != "${session_id}" ]] && continue

                local tool
                tool=$(printf '%s' "${payload}" | jq -r '.properties.type // empty' 2>/dev/null || true)
                local patterns_output
                patterns_output=$(opencode::permissions::patterns_from_payload "${payload}")

                local decision
                decision=$(opencode::permissions::decide "${allowed_tools}" "${skip_permissions}" "${tool}" "${patterns_output}")

                local response_body
                if [[ "${decision}" == "allow" ]]; then
                    response_body='{"response":"always"}'
                    log::info "Auto-approved ${tool} permission (${permission_id})"
                else
                    response_body='{"response":"reject"}'
                    log::warning "Denied ${tool} permission (${permission_id})"
                    if [[ -n "${abort_file}" ]]; then
                        printf '%s' "permission_rejected" >"${abort_file}"
                    fi
                fi

                OPENCODE_REQUEST_TIMEOUT_SECS= opencode::api::request_raw POST "/session/${session_id}/permissions/${permission_id}" "${response_body}" "${directory}" >/dev/null || true
                acknowledged_permissions["${permission_id}"]=1
                ;;
            message\.part\.updated)
                if [[ -z "${max_turns}" || "${max_turns}" -le 0 ]]; then
                    continue
                fi

                local part_session
                part_session=$(printf '%s' "${payload}" | jq -r '.properties.part.sessionID // empty' 2>/dev/null || true)
                [[ "${part_session}" != "${session_id}" ]] && continue

                local part_type
                part_type=$(printf '%s' "${payload}" | jq -r '.properties.part.type // empty' 2>/dev/null || true)
                [[ "${part_type}" != "tool" ]] && continue

                local part_id
                part_id=$(printf '%s' "${payload}" | jq -r '.properties.part.id // empty' 2>/dev/null || true)
                if [[ -z "${part_id}" || -n "${processed_tool_parts[${part_id}]+x}" ]]; then
                    continue
                fi

                local state_status
                state_status=$(printf '%s' "${payload}" | jq -r '.properties.part.state.status // empty' 2>/dev/null || true)
                if [[ "${state_status}" != "completed" && "${state_status}" != "error" ]]; then
                    continue
                fi

                processed_tool_parts["${part_id}"]=1
                turns=$((turns + 1))
                if [[ "${max_turns}" -gt 0 && "${turns}" -ge "${max_turns}" ]]; then
                    if [[ -n "${abort_file}" ]]; then
                        printf '%s' "max_turns" >"${abort_file}"
                    fi
                    opencode::api::request_raw POST "/session/${session_id}/abort" '{}' "${directory}" >/dev/null || true
                    break
                fi
                ;;
            *)
                continue
                ;;
        esac
    done <"${fifo_path}"
}

opencode::events::start_listener() {
    local session_id="${1:-}"
    local directory="${2:-}"
    local allowed_tools="${3:-${OPENCODE_ALLOWED_TOOLS:-${ALLOWED_TOOLS:-}}}"
    local skip_permissions="${4:-${OPENCODE_SKIP_PERMISSIONS:-${SKIP_PERMISSIONS:-}}}"
    local max_turns="${5:-${OPENCODE_MAX_TURNS:-${MAX_TURNS:-0}}}"

    OPENCODE_EVENT_LISTENER=""

    if [[ -z "${session_id}" ]]; then
        return 0
    fi
    if ! command -v jq >/dev/null 2>&1; then
        log::warning "jq not available; skipping permission auto-response"
        return 0
    fi

    local fifo_path
    fifo_path=$(mktemp -u "${TMPDIR:-/tmp}/opencode-events.XXXXXX")
    mkfifo "${fifo_path}"

    local url
    url="$(opencode::api::build_url "/event" "${directory}")" || {
        rm -f "${fifo_path}"
        return 1
    }

    curl --silent --show-error --no-buffer "${url}" >"${fifo_path}" &
    local curl_pid=$!
    disown "${curl_pid}" 2>/dev/null || true

    local abort_file
    abort_file=$(mktemp "${TMPDIR:-/tmp}/opencode-abort.XXXXXX")
    : >"${abort_file}"

    opencode::events::loop "${session_id}" "${directory}" "${allowed_tools}" "${skip_permissions}" "${max_turns}" "${abort_file}" "${fifo_path}" "${curl_pid}" &
    local loop_pid=$!
    disown "${loop_pid}" 2>/dev/null || true

    OPENCODE_EVENT_LISTENER="${loop_pid}:${curl_pid}:${fifo_path}:${abort_file}"
}

opencode::events::stop_listener() {
    local token="${1:-}"
    if [[ -z "${token}" ]]; then
        return 0
    fi

    local loop_pid curl_pid fifo_path abort_file
    IFS=':' read -r loop_pid curl_pid fifo_path abort_file <<<"${token}"

    if [[ -n "${loop_pid}" ]]; then
        kill "${loop_pid}" 2>/dev/null || true
        wait "${loop_pid}" 2>/dev/null || true
    fi
    if [[ -n "${curl_pid}" ]]; then
        kill "${curl_pid}" 2>/dev/null || true
        wait "${curl_pid}" 2>/dev/null || true
    fi
    if [[ -n "${fifo_path}" ]]; then
        rm -f "${fifo_path}" 2>/dev/null || true
    fi

    if [[ -n "${abort_file}" && -f "${abort_file}" ]]; then
        local reason
        reason=$(<"${abort_file}")
        rm -f "${abort_file}" 2>/dev/null || true
        printf '%s' "${reason}"
    fi
}

################################################################################
# Agent helpers
################################################################################

opencode::agents::tools_payload() {
    local allowed_csv="${1:-}"
    [[ -z "${allowed_csv}" ]] && return 0

    declare -A map=(
        [edit]=false
        [write]=false
        [read]=false
        [bash]=false
        [webfetch]=false
    )

    IFS=',' read -r -a entries <<<"${allowed_csv}"
    for entry_raw in "${entries[@]}"; do
        local entry
        entry=$(printf '%s' "${entry_raw}" | sed -e 's/^ *//' -e 's/ *$//' | tr '[:upper:]' '[:lower:]')
        [[ -z "${entry}" ]] && continue
        if [[ "${entry}" == "*" ]]; then
            for key in "${!map[@]}"; do
                map["${key}"]=true
            done
            break
        fi
        if [[ -n "${map[${entry}]+x}" ]]; then
            map["${entry}"]=true
        fi
    done

    printf '{"edit":%s,"write":%s,"read":%s,"bash":%s,"webfetch":%s}' \
        "${map[edit]}" "${map[write]}" "${map[read]}" "${map[bash]}" "${map[webfetch]}"
}

opencode::agents::message_internal() {
    local session_id="$1"
    local prompt="$2"
    local provider="$3"
    local model="$4"
    local agent_name="$5"
    local system_text="$6"
    local directory="$7"
    local tools_csv="$8"

    if ! command -v jq >/dev/null 2>&1; then
        log::error "jq is required to build OpenCode requests"
        return 1
    fi

    local provider_id="${provider}" model_id="${model}"
    if [[ -n "${model_id}" && "${model_id}" == */* ]]; then
        if [[ -z "${provider_id}" ]]; then
            provider_id="${model_id%%/*}"
        fi
        model_id="${model_id#*/}"
    fi

    if [[ -n "${provider_id}" && -z "${model_id}" ]]; then
        log::error "Model ID is required when specifying a provider"
        return 1
    fi

    local tools_json=""
    if [[ -n "${tools_csv}" ]]; then
        tools_json=$(opencode::agents::tools_payload "${tools_csv}")
    fi
    [[ -z "${tools_json}" ]] && tools_json="null"

    local payload
    payload=$(jq -n \
        --arg text "${prompt}" \
        --arg provider "${provider_id}" \
        --arg model "${model_id}" \
        --arg agent "${agent_name}" \
        --arg system "${system_text}" \
        --argjson tools "${tools_json}" \
        '{ parts: [ { type: "text", text: $text } ] }
         | if ($agent | length) > 0 then .agent = $agent else . end
         | if ($system | length) > 0 then .system = $system else . end
         | if ($provider | length) > 0 and ($model | length) > 0 then .model = { providerID: $provider, modelID: $model } else . end
         | if ($tools | type) == "object" then .tools = $tools else . end')

    if ! opencode::api::request_raw POST "/session/${session_id}/message" "${payload}" "${directory}"; then
        return 1
    fi

    local body="${REPLY}"

    if command -v jq >/dev/null 2>&1; then
        log::info "Assistant response:"
        printf '%s' "${body}" | jq -r '.parts[]? | select(.type=="text") | .text'
        printf '\n'
        printf '%s' "${body}" | jq .
    else
        printf '%s\n' "${body}"
    fi

    return 0
}

opencode::agents::dispatch_message() {
    local session_id="$1"
    shift

    if type -t agents::with_agent >/dev/null 2>&1; then
        agents::with_agent "opencode-message" "opencode::agents::message_internal" "${session_id}" "$@"
    else
        opencode::agents::message_internal "${session_id}" "$@"
    fi
}

opencode::agents::create_session() {
    local title="$1"
    local parent_id="$2"
    local directory="$3"

    if ! command -v jq >/dev/null 2>&1; then
        log::error "jq is required to build OpenCode requests"
        return 1
    fi

    local payload
    payload=$(jq -n \
        --arg title "${title}" \
        --arg parent "${parent_id}" \
        '. | if ($title | length) > 0 then .title = $title else . end
           | if ($parent | length) > 0 then .parentID = $parent else . end')

    if ! opencode::api::request_raw POST "/session" "${payload}" "${directory}"; then
        return 1
    fi

    local session_json="${REPLY}"
    local session_id
    session_id=$(printf '%s' "${session_json}" | jq -r '.id // empty')
    if [[ -z "${session_id}" ]]; then
        log::error "Failed to parse session ID from response"
        return 1
    fi

    log::info "Created OpenCode session: ${session_id}"
    REPLY="${session_id}"
    return 0
}

opencode::agents::run() {
    local prompt="" prompt_file="" session_id="" title="" provider="" model="" agent_name="" system_text="" directory=""
    local allowed_tools="${OPENCODE_ALLOWED_TOOLS:-${ALLOWED_TOOLS:-}}"
    local skip_permissions="${OPENCODE_SKIP_PERMISSIONS:-${SKIP_PERMISSIONS:-}}"
    local max_turns="${OPENCODE_MAX_TURNS:-${MAX_TURNS:-}}"
    local task_timeout="${OPENCODE_TASK_TIMEOUT:-${TASK_TIMEOUT:-${TIMEOUT:-}}}"

    while [[ $# -gt 0 ]]; do
        case "${1}" in
            --prompt)
                prompt="${2:-}"
                shift 2
                ;;
            --prompt-file)
                prompt_file="${2:-}"
                shift 2
                ;;
            --session)
                session_id="${2:-}"
                shift 2
                ;;
            --title)
                title="${2:-}"
                shift 2
                ;;
            --provider)
                provider="${2:-}"
                shift 2
                ;;
            --model)
                model="${2:-}"
                shift 2
                ;;
            --agent)
                agent_name="${2:-}"
                shift 2
                ;;
            --system)
                system_text="${2:-}"
                shift 2
                ;;
            --directory)
                directory="${2:-}"
                shift 2
                ;;
            --allowed-tools)
                allowed_tools="${2:-}"
                shift 2
                ;;
            --skip-permissions)
                skip_permissions="yes"
                shift
                ;;
            --max-turns)
                max_turns="${2:-}"
                shift 2
                ;;
            --task-timeout|--timeout)
                task_timeout="${2:-}"
                shift 2
                ;;
            -h|--help)
                opencode::agents::run_usage
                return 0
                ;;
            *)
                log::error "Unknown option: ${1}"
                opencode::agents::run_usage
                return 1
                ;;
        esac
    done

    if [[ -n "${prompt_file}" ]]; then
        if [[ ! -f "${prompt_file}" ]]; then
            log::error "Prompt file not found: ${prompt_file}"
            return 1
        fi
        prompt="$(<"${prompt_file}")"
    fi

    if [[ -z "${prompt}" ]]; then
        log::error "Prompt is required (--prompt or --prompt-file)"
        opencode::agents::run_usage
        return 1
    fi

    if [[ -z "${directory}" ]]; then
        directory="$(opencode::api::default_directory)"
    fi

    if [[ -n "${max_turns}" && ! "${max_turns}" =~ ^[0-9]+$ ]]; then
        log::warning "Ignoring invalid max-turns value: ${max_turns}"
        max_turns=""
    fi
    if [[ -n "${task_timeout}" && ! "${task_timeout}" =~ ^[0-9]+$ ]]; then
        log::warning "Ignoring invalid task timeout value: ${task_timeout}"
        task_timeout=""
    fi

    opencode::server::ensure_running || return 1

    if [[ -z "${session_id}" ]]; then
        local generated_title="${title}"
        if [[ -z "${generated_title}" ]]; then
            generated_title="Vrooli session $(date -u +%Y-%m-%dT%H:%M:%SZ)"
        fi
        if ! opencode::agents::create_session "${generated_title}" "" "${directory}"; then
            return 1
        fi
        session_id="${REPLY}"
    else
        log::info "Using existing session: ${session_id}"
    fi

    local normalized_allowed=""
    if [[ -n "${allowed_tools}" ]]; then
        normalized_allowed=$(opencode::permissions::normalize_csv "${allowed_tools}")
        normalized_allowed=${normalized_allowed// /,}
        normalized_allowed=${normalized_allowed#,}
        normalized_allowed=${normalized_allowed%,}
    fi

    local listener_token=""
    if opencode::events::start_listener "${session_id}" "${directory}" "${normalized_allowed}" "${skip_permissions}" "${max_turns}"; then
        listener_token="${OPENCODE_EVENT_LISTENER:-}"
    fi

    local previous_timeout_marker="${OPENCODE_REQUEST_TIMEOUT_SECS+set}"
    local previous_timeout_value="${OPENCODE_REQUEST_TIMEOUT_SECS:-}"
    if [[ -n "${task_timeout}" && "${task_timeout}" -gt 0 ]]; then
        export OPENCODE_REQUEST_TIMEOUT_SECS="${task_timeout}"
    else
        unset OPENCODE_REQUEST_TIMEOUT_SECS
    fi

    opencode::agents::dispatch_message "${session_id}" "${prompt}" "${provider}" "${model}" \
        "${agent_name}" "${system_text}" "${directory}" "${normalized_allowed}"
    local dispatch_status=$?

    if [[ "${previous_timeout_marker}" == "set" ]]; then
        export OPENCODE_REQUEST_TIMEOUT_SECS="${previous_timeout_value}"
    else
        unset OPENCODE_REQUEST_TIMEOUT_SECS
    fi

    local abort_reason=""
    abort_reason=$(opencode::events::stop_listener "${listener_token}") || true

    if [[ ${dispatch_status} -ne 0 ]]; then
        if [[ "${OPENCODE_CURL_EXIT_CODE:-0}" -eq 28 ]]; then
            log::error "OpenCode request timed out after ${task_timeout:-?} seconds"
            return 3
        fi
        return ${dispatch_status}
    fi

    if [[ -n "${abort_reason}" ]]; then
        case "${abort_reason}" in
            max_turns)
                log::error "OpenCode session aborted after reaching max turns (${max_turns})"
                return 2
                ;;
            permission_rejected)
                log::error "OpenCode permission was rejected; session halted"
                return 3
                ;;
        esac
    fi

    return 0
}

opencode::agents::usage() {
    cat <<'EOF'
Usage: resource-opencode agents <subcommand> [options]

Custom subcommands:
  run      Execute a prompt via the OpenCode server (creates a session if needed)
  session  Manage OpenCode server sessions (list)

Standard agent manager subcommands:
  list | stop | info | logs | monitor | metrics | cleanup | help

Examples:
  resource-opencode agents run --prompt "Fix the failing tests" --model openrouter/qwen3-coder
  resource-opencode agents session list
  resource-opencode agents list --json
EOF
}

opencode::agents::run_usage() {
    cat <<'EOF'
Usage: resource-opencode agents run --prompt <text> [options]

Options:
  --prompt <text>          Prompt to send to the OpenCode server (required)
  --prompt-file <path>     Read prompt from file
  --session <id>           Reuse an existing session (skip creation)
  --title <title>          Title to apply when creating a new session
  --provider <id>          Provider ID (e.g. openrouter)
  --model <id|provider/model>
                           Model identifier; accepts provider/model or model ID when --provider is set
  --agent <name>           Agent name override for OpenCode
  --system <text>          Additional system instructions
  --directory <path>       Project directory scope (default: repo root)
  --allowed-tools <list>   Comma-separated allowlist (e.g. "edit,write,bash(ls *)")
  --skip-permissions       Auto-approve permission prompts (dangerous)
  --max-turns <n>          Abort after N completed tool calls (0 = unlimited)
  --task-timeout <sec>     Abort the request if it exceeds N seconds (alias: --timeout)
  -h, --help               Show this help message
EOF
}

opencode::agents::session_usage() {
    cat <<'EOF'
Usage: resource-opencode agents session <command> [options]

Commands:
  list [--directory <path>]   List OpenCode server sessions for the directory
  help                        Show this message
EOF
}

opencode::agents::session_list() {
    local directory=""
    while [[ $# -gt 0 ]]; do
        case "${1}" in
            --directory)
                directory="${2:-}"
                shift 2
                ;;
            -h|--help)
                opencode::agents::session_usage
                return 0
                ;;
            *)
                log::error "Unknown option: ${1}"
                return 1
                ;;
        esac
    done

    opencode::server::ensure_running || return 1
    opencode::api::request GET "/session" "" "${directory}"
}

opencode::agents::session_command() {
    local subcommand="${1:-list}"
    shift || true

    case "${subcommand}" in
        list)
            opencode::agents::session_list "$@"
            ;;
        help|-h|--help)
            opencode::agents::session_usage
            ;;
        *)
            log::error "Unknown session subcommand: ${subcommand}"
            opencode::agents::session_usage
            return 1
            ;;
    esac
}
