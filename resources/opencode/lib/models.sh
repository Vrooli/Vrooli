#!/bin/bash
# OpenCode model discovery helpers

source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::models::list() {
    local output_format="text"
    local provider_filter=""
    local search_term=""
    local limit_value=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                output_format="json"
                ;;
            --provider|-p)
                provider_filter="${2:-}"
                shift
                ;;
            --search|--contains)
                search_term="${2:-}"
                shift
                ;;
            --limit|-n)
                limit_value="${2:-}"
                shift
                ;;
            --help|-h)
                cat <<'HELP'
Usage: resource-opencode models [options]

Options:
  --json            Output structured JSON including metadata for each model
  --provider <id>   Filter by provider (e.g. openrouter, ollama)
  --search <term>   Filter by substring in id, display name, or description
  --limit <n>       Limit the number of models returned
  -h, --help        Show this help message

Examples:
  resource-opencode models --limit 15
  resource-opencode models --provider openrouter --json
  resource-opencode models --search embed
HELP
                return 0
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
        shift
    done

    if [[ -n "${limit_value}" && ! "${limit_value}" =~ ^[0-9]+$ ]]; then
        log::error "Invalid limit: ${limit_value}. Provide a positive integer."
        return 1
    fi

    if ! command -v jq &>/dev/null; then
        log::error "jq is required to list OpenCode models. Install jq and retry."
        return 1
    fi

    opencode::ensure_dirs
    opencode::load_secrets || true

    local openrouter_data=""
    local openrouter_models="[]"
    local openrouter_status="unavailable"
    if command -v resource-openrouter &>/dev/null; then
        if openrouter_data=$(resource-openrouter content models --json 2>/dev/null); then
            openrouter_models=$(echo "${openrouter_data}" |                 jq '(.models // [])
                    | map(
                        .backend = (.provider // null)
                        | .provider = "openrouter"
                        | .model = (.id // null)
                        | .id = (if .model then "openrouter/" + .model else .id end)
                        | .source = "openrouter"
                        | .origin = "remote"
                    )')
            openrouter_status="ok"
        else
            openrouter_status="error"
        fi
    fi

    local ollama_models="[]"
    local ollama_status="unavailable"
    if command -v ollama &>/dev/null; then
        local ollama_raw
        if ollama_raw=$(ollama list 2>/dev/null); then
            ollama_models=$(echo "${ollama_raw}"                 | tail -n +2                 | sed -E 's/[[:space:]]+$//'                 | sed -E 's/[[:space:]]{2,}/	/g'                 | jq -Rn '[inputs | select(length>0) | split("	") | {
                        id: ("ollama/" + (.[0] // "")),
                        model: (.[0] // ""),
                        provider: "ollama",
                        backend: "ollama",
                        source: "ollama",
                        origin: "local",
                        display_name: (.[0] // ""),
                        metadata: {
                            digest: (.[1] // null),
                            size: (.[2] // null),
                            modified: (.[3] // null)
                        }
                    }]')
            ollama_status="ok"
        else
            ollama_status="error"
        fi
    fi

    local combined
    combined=$(printf '%s
%s
' "${openrouter_models}" "${ollama_models}" | jq -s 'add')

    local openrouter_count
    openrouter_count=$(jq 'length' <<<"${openrouter_models}" 2>/dev/null || echo "0")

    local ollama_count
    ollama_count=$(jq 'length' <<<"${ollama_models}" 2>/dev/null || echo "0")

    local provider_filter_value="${provider_filter}"
    local search_value="${search_term}"
    local limit_json="null"
    if [[ -n "${limit_value}" ]]; then
        limit_json="${limit_value}"
    fi

    local filtered
    filtered=$(jq         --arg provider "${provider_filter_value}"         --arg search "${search_value}"         --argjson limit "${limit_json}"         '
        def matches_provider($prefix):
          ($prefix == "" or (.provider // "" | startswith($prefix)) or (.id // "" | startswith($prefix)));

        def matches_search($term):
          ($term == "" or (
            (.id // "" | test($term; "i")) or
            (.display_name // "" | test($term; "i")) or
            (.provider // "" | test($term; "i")) or
            (.description // "" | test($term; "i"))
          ));

        map(select(matches_provider($provider) and matches_search($search)))
        | (if $limit != null and $limit > 0 then .[:$limit] else . end)
        ' <<<"${combined}")

    if [[ "${output_format}" == "json" ]]; then
        local default_model="${OPENCODE_DEFAULT_PROVIDER}/${OPENCODE_DEFAULT_CHAT_MODEL}"
        local filtered_file
        filtered_file=$(mktemp)
        printf '%s' "${filtered}" > "${filtered_file}"

        jq -n             --arg fetched "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"             --arg default "${default_model}"             --arg openrouter_status "${openrouter_status}"             --arg ollama_status "${ollama_status}"             --argjson openrouter_count "${openrouter_count}"             --argjson ollama_count "${ollama_count}"             --slurpfile models "${filtered_file}"             '
            {
                source: "opencode",
                fetched_at: $fetched,
                default_model: $default,
                count: ($models[0] | length),
                sources: [
                    { name: "openrouter", status: $openrouter_status, count: $openrouter_count },
                    { name: "ollama", status: $ollama_status, count: $ollama_count }
                ],
                models: $models[0]
            }
            '
        local jq_status=$?
        rm -f "${filtered_file}"
        return ${jq_status}
    fi

    if [[ "${filtered}" == "[]" ]]; then
        echo "${OPENCODE_DEFAULT_PROVIDER}/${OPENCODE_DEFAULT_CHAT_MODEL}"
        return 0
    fi

    if command -v column &>/dev/null; then
        jq -r '.[] | "\(.id)	\(.provider)	\(.source)"' <<<"${filtered}" | column -t -s $'	'
    else
        jq -r '.[] | "\(.id)	\(.provider)	\(.source)"' <<<"${filtered}"
    fi
}

export -f opencode::models::list
