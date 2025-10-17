#!/bin/bash
# OpenRouter model selection and fallback library

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_LIB_DIR="${APP_ROOT}/resources/openrouter/lib"
OPENROUTER_RESOURCE_DIR="${APP_ROOT}/resources/openrouter"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${OPENROUTER_LIB_DIR}/core.sh"
source "${OPENROUTER_RESOURCE_DIR}/config/defaults.sh"

# Source rate limit handling if available
if [[ -f "${OPENROUTER_LIB_DIR}/ratelimit.sh" ]]; then
    source "${OPENROUTER_LIB_DIR}/ratelimit.sh"
fi

# Source routing rules if available
if [[ -f "${OPENROUTER_LIB_DIR}/routing.sh" ]]; then
    source "${OPENROUTER_LIB_DIR}/routing.sh"
fi

openrouter::models::require_jq() {
    if ! command -v jq &>/dev/null; then
        log::error "jq is required to list OpenRouter models. Install jq and retry."
        return 1
    fi
    return 0
}

openrouter::models::fetch_catalog() {
    local timeout="${OPENROUTER_TIMEOUT:-30}"

    # Initialize credentials if available (but listing works without a key)
    openrouter::init >/dev/null 2>&1 || true

    local api_url
    api_url=$(openrouter::cloudflare::get_gateway_url "${OPENROUTER_API_BASE}" "")

    local -a curl_args=(-sS "${api_url}/models")
    if [[ -n "${OPENROUTER_API_KEY:-}" && "${OPENROUTER_API_KEY}" != auto-null-* ]]; then
        curl_args=(-H "Authorization: Bearer ${OPENROUTER_API_KEY}" "${curl_args[@]}")
    fi

    local response
    if ! response=$(timeout "${timeout}" curl "${curl_args[@]}" 2>/dev/null); then
        return 1
    fi

    if [[ -z "${response}" ]]; then
        return 1
    fi

    echo "${response}"
    return 0
}

openrouter::models::filter_catalog() {
    local raw_json="${1:-}"
    local provider_prefix="${2:-}"
    local search_term="${3:-}"
    local limit_value="${4:-null}"

    if ! openrouter::models::require_jq; then
        return 1
    fi

    local normalized
    normalized=$(jq \
        --arg provider "${provider_prefix}" \
        --arg search "${search_term}" \
        --argjson limit "${limit_value}" \
        '
        def parse_price($p):
          if ($p == null or $p == "") then null else
            ($p | try (tonumber) catch (try (sub(","; "") | tonumber) catch null))
          end;

        def matches_provider($prefix):
          ($prefix == "" or (.id | startswith($prefix)));

        def matches_search($term):
          ($term == "" or ((.id // "") | test($term; "i") or (.name // "") | test($term; "i") or (.description // "") | test($term; "i")));

        (.data // [])
        | map(select(matches_provider($provider) and matches_search($search))
            | . as $model
            | {
                id: ($model.id // null),
                name: ($model.name // null),
                display_name: ($model.name // $model.id),
                provider: (($model.id // "") | split("/") | .[0] // "unknown"),
                description: ($model.description // null),
                pricing: {
                    prompt: parse_price($model.pricing.prompt),
                    completion: parse_price($model.pricing.completion),
                    request: parse_price($model.pricing.request),
                    image: parse_price($model.pricing.image)
                },
                context_length: ($model.top_provider.context_length // $model.context_length // null),
                max_completion_tokens: ($model.top_provider.max_completion_tokens // null),
                architecture: {
                    modality: ($model.architecture.modality // null),
                    input: ($model.architecture.input_modalities // []),
                    output: ($model.architecture.output_modalities // [])
                },
                supported_parameters: ($model.supported_parameters // [])
            }
        )
        | (if ($limit != null and $limit > 0) then .[:$limit] else . end)
        ' <<<"${raw_json}") || return 1

    echo "${normalized}"
}

openrouter::models::build_response() {
    local normalized_json="${1:-[]}";
    local default_model="${OPENROUTER_DEFAULT_MODEL:-openai/gpt-3.5-turbo}";

    if ! openrouter::models::require_jq; then
        return 1
    fi

    jq \
        --arg source "openrouter" \
        --arg fetched "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
        --arg default_model "${default_model}" \
        '
        . as $models
        | {
            source: $source,
            fetched_at: $fetched,
            default_model: $default_model,
            provider_count: ($models | map(.provider) | unique | length),
            count: ($models | length),
            models: $models
        }
        ' <<<"${normalized_json}"
}

# Model selection strategies
openrouter::models::select_auto() {
    local task_type="${1:-general}"
    local budget="${2:-0.10}"
    local prompt="${3:-}"
    
    # Check if routing rules should be used
    if [[ -n "$prompt" ]] && command -v openrouter::routing::evaluate >/dev/null 2>&1; then
        # Use routing rules to select model
        local selected_model=$(openrouter::routing::evaluate "$prompt")
        if [[ -n "$selected_model" ]]; then
            echo "$selected_model"
            return 0
        fi
    fi
    
    # Fall back to original auto-select based on task type
    case "$task_type" in
        code|programming)
            echo "anthropic/claude-3-sonnet"
            ;;
        creative|writing)
            echo "anthropic/claude-3-opus"
            ;;
        simple|quick)
            echo "openai/gpt-3.5-turbo"
            ;;
        analysis|reasoning)
            echo "openai/gpt-4-turbo"
            ;;
        budget|cheap)
            echo "mistralai/mistral-7b"
            ;;
        *)
            echo "${OPENROUTER_DEFAULT_MODEL:-openai/gpt-3.5-turbo}"
            ;;
    esac
}

# Select cheapest model that meets requirements
openrouter::models::select_cheapest() {
    local min_capability="${1:-basic}"
    
    # Ordered from cheapest to most expensive
    local cheap_models=(
        "mistralai/mistral-7b"
        "openai/gpt-3.5-turbo"
        "anthropic/claude-3-haiku"
        "openai/gpt-4-turbo"
        "anthropic/claude-3-sonnet"
        "anthropic/claude-3-opus"
    )
    
    # For now, return the cheapest that's available
    # In a full implementation, we'd check actual pricing via API
    echo "${cheap_models[0]}"
}

# Select fastest model
openrouter::models::select_fastest() {
    # Models ordered by typical response speed
    local fast_models=(
        "openai/gpt-3.5-turbo"
        "mistralai/mistral-7b"
        "anthropic/claude-3-haiku"
    )
    
    echo "${fast_models[0]}"
}

# Select highest quality model
openrouter::models::select_quality() {
    # Models ordered by quality/capability
    local quality_models=(
        "anthropic/claude-3-opus"
        "openai/gpt-4-turbo"
        "anthropic/claude-3-sonnet"
    )
    
    echo "${quality_models[0]}"
}

# Execute request with fallback chain
openrouter::models::execute_with_fallback() {
    local primary_model="${1}"
    local prompt="${2}"
    local max_tokens="${3:-1000}"
    shift 3
    local fallback_models=("$@")
    
    # Add default fallbacks if none provided
    if [[ ${#fallback_models[@]} -eq 0 ]]; then
        fallback_models=(
            "openai/gpt-3.5-turbo"
            "mistralai/mistral-7b"
        )
    fi
    
    # Initialize API key if needed
    if [[ -z "${OPENROUTER_API_KEY:-}" ]]; then
        if ! openrouter::init; then
            log::error "Failed to initialize OpenRouter API key"
            return 1
        fi
    fi
    
    # Try primary model first
    local models_to_try=("$primary_model" "${fallback_models[@]}")
    local timeout="${OPENROUTER_TIMEOUT:-30}"
    
    for model in "${models_to_try[@]}"; do
        log::info "Trying model: $model"
        
        # Check rate limiting before making request
        if command -v openrouter::ratelimit::check >/dev/null 2>&1; then
            if ! openrouter::ratelimit::check; then
                log::warn "Rate limited, queuing request"
                local request_data=$(jq -n \
                    --arg model "$model" \
                    --arg content "$prompt" \
                    --argjson max_tokens "$max_tokens" \
                    '{
                        model: $model,
                        messages: [{role: "user", content: $content}],
                        max_tokens: $max_tokens
                    }')
                openrouter::ratelimit::queue_request "$request_data" "normal"
                continue
            fi
        fi
        
        local response
        local temp_headers=$(mktemp)
        response=$(timeout "$timeout" curl -s -D "$temp_headers" -X POST \
            -H "Authorization: Bearer ${OPENROUTER_API_KEY}" \
            -H "Content-Type: application/json" \
            -d "$(jq -n \
                --arg model "$model" \
                --arg content "$prompt" \
                --argjson max_tokens "$max_tokens" \
                '{
                    model: $model,
                    messages: [{role: "user", content: $content}],
                    max_tokens: $max_tokens
                }')" \
            "${OPENROUTER_API_BASE}/chat/completions" 2>/dev/null)
        
        # Get response code
        local response_code=$(grep "^HTTP/" "$temp_headers" 2>/dev/null | tail -1 | cut -d' ' -f2)
        
        # Update rate limit tracking if available
        if command -v openrouter::ratelimit::update >/dev/null 2>&1; then
            openrouter::ratelimit::update "$(<"$temp_headers")" "$response_code"
        fi
        
        rm -f "$temp_headers"
        
        # Check if successful
        if [[ -n "$response" ]] && ! echo "$response" | jq -e '.error' >/dev/null 2>&1; then
            # Track usage if response includes usage data
            local usage=$(echo "$response" | jq -r '.usage // empty')
            if [[ -n "$usage" ]]; then
                local prompt_tokens=$(echo "$usage" | jq -r '.prompt_tokens // 0')
                local completion_tokens=$(echo "$usage" | jq -r '.completion_tokens // 0')
                local total_cost=$(echo "$usage" | jq -r '.total_cost // 0')
                
                # Track the usage
                openrouter::models::track_usage "$model" "$prompt_tokens" "$completion_tokens" "$total_cost"
            fi
            
            # Success - return the response with model info
            echo "$response" | jq --arg model "$model" '. + {actual_model: $model}'
            return 0
        fi
        
        # Log the error and try next model
        local error_msg=$(echo "$response" | jq -r '.error.message // .error // "Unknown error"' 2>/dev/null || echo "Request failed")
        log::warn "Model $model failed: $error_msg"
    done
    
    log::error "All models failed. Tried: ${models_to_try[*]}"
    return 1
}

# Get model pricing information
openrouter::models::get_pricing() {
    local model="${1}"
    local timeout="${OPENROUTER_TIMEOUT:-30}"
    
    if [[ -z "${OPENROUTER_API_KEY:-}" ]]; then
        if ! openrouter::init; then
            return 1
        fi
    fi
    
    # Fetch model details including pricing
    local response
    response=$(timeout "$timeout" curl -s \
        -H "Authorization: Bearer ${OPENROUTER_API_KEY}" \
        "${OPENROUTER_API_BASE}/models" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        return 1
    fi
    
    # Extract pricing for specific model
    echo "$response" | jq -r --arg model "$model" '
        .data[] | 
        select(.id == $model) | 
        {
            id: .id,
            name: .name,
            pricing: .pricing,
            context_length: .context_length,
            per_request_limits: .per_request_limits
        }'
}

# Track usage and costs
openrouter::models::track_usage() {
    local model="${1}"
    local prompt_tokens="${2}"
    local completion_tokens="${3}"
    local cost="${4:-0}"
    
    # Create usage tracking directory if it doesn't exist
    local usage_dir="${var_ROOT_DIR}/data/openrouter/usage"
    mkdir -p "$usage_dir"
    
    # Create daily usage file
    local date=$(date +%Y-%m-%d)
    local usage_file="$usage_dir/usage-$date.json"
    
    # Initialize file if it doesn't exist
    if [[ ! -f "$usage_file" ]]; then
        echo '{"total_cost": 0, "total_prompt_tokens": 0, "total_completion_tokens": 0, "requests": []}' > "$usage_file"
    fi
    
    # Add new usage entry
    local timestamp=$(date -Iseconds)
    local new_entry=$(jq -n \
        --arg model "$model" \
        --arg timestamp "$timestamp" \
        --argjson prompt_tokens "$prompt_tokens" \
        --argjson completion_tokens "$completion_tokens" \
        --argjson cost "$cost" \
        '{
            model: $model,
            timestamp: $timestamp,
            prompt_tokens: $prompt_tokens,
            completion_tokens: $completion_tokens,
            cost: $cost
        }')
    
    # Update the usage file
    jq --argjson entry "$new_entry" \
       --argjson prompt_tokens "$prompt_tokens" \
       --argjson completion_tokens "$completion_tokens" \
       --argjson cost "$cost" \
       '.requests += [$entry] |
        .total_prompt_tokens += $prompt_tokens |
        .total_completion_tokens += $completion_tokens |
        .total_cost += $cost' "$usage_file" > "$usage_file.tmp" && mv "$usage_file.tmp" "$usage_file"
}

# Get usage analytics
openrouter::models::get_usage_analytics() {
    local period="${1:-today}"  # today, week, month, all
    local usage_dir="${var_ROOT_DIR}/data/openrouter/usage"
    
    if [[ ! -d "$usage_dir" ]]; then
        echo '{"error": "No usage data available"}'
        return 1
    fi
    
    case "$period" in
        today)
            local date=$(date +%Y-%m-%d)
            local usage_file="$usage_dir/usage-$date.json"
            if [[ -f "$usage_file" ]]; then
                cat "$usage_file"
            else
                echo '{"total_cost": 0, "total_prompt_tokens": 0, "total_completion_tokens": 0, "requests": []}'
            fi
            ;;
        week)
            # Aggregate last 7 days
            local total_cost=0
            local total_prompt=0
            local total_completion=0
            local total_requests=0
            
            for i in {0..6}; do
                local date=$(date -d "$i days ago" +%Y-%m-%d 2>/dev/null || date -v -${i}d +%Y-%m-%d)
                local usage_file="$usage_dir/usage-$date.json"
                if [[ -f "$usage_file" ]]; then
                    local day_data=$(cat "$usage_file")
                    total_cost=$(echo "$total_cost + $(echo "$day_data" | jq -r '.total_cost')" | bc)
                    total_prompt=$((total_prompt + $(echo "$day_data" | jq -r '.total_prompt_tokens')))
                    total_completion=$((total_completion + $(echo "$day_data" | jq -r '.total_completion_tokens')))
                    total_requests=$((total_requests + $(echo "$day_data" | jq -r '.requests | length')))
                fi
            done
            
            jq -n \
                --argjson cost "$total_cost" \
                --argjson prompt "$total_prompt" \
                --argjson completion "$total_completion" \
                --argjson requests "$total_requests" \
                '{
                    period: "week",
                    total_cost: $cost,
                    total_prompt_tokens: $prompt,
                    total_completion_tokens: $completion,
                    total_requests: $requests
                }'
            ;;
        month)
            # Aggregate last 30 days
            local total_cost=0
            local total_prompt=0
            local total_completion=0
            local total_requests=0
            
            for i in {0..29}; do
                local date=$(date -d "$i days ago" +%Y-%m-%d 2>/dev/null || date -v -${i}d +%Y-%m-%d)
                local usage_file="$usage_dir/usage-$date.json"
                if [[ -f "$usage_file" ]]; then
                    local day_data=$(cat "$usage_file")
                    total_cost=$(echo "$total_cost + $(echo "$day_data" | jq -r '.total_cost')" | bc)
                    total_prompt=$((total_prompt + $(echo "$day_data" | jq -r '.total_prompt_tokens')))
                    total_completion=$((total_completion + $(echo "$day_data" | jq -r '.total_completion_tokens')))
                    total_requests=$((total_requests + $(echo "$day_data" | jq -r '.requests | length')))
                fi
            done
            
            jq -n \
                --argjson cost "$total_cost" \
                --argjson prompt "$total_prompt" \
                --argjson completion "$total_completion" \
                --argjson requests "$total_requests" \
                '{
                    period: "month",
                    total_cost: $cost,
                    total_prompt_tokens: $prompt,
                    total_completion_tokens: $completion,
                    total_requests: $requests
                }'
            ;;
        all)
            # Aggregate all available data
            local total_cost=0
            local total_prompt=0
            local total_completion=0
            local total_requests=0
            
            for file in "$usage_dir"/usage-*.json; do
                if [[ -f "$file" ]]; then
                    local day_data=$(cat "$file")
                    total_cost=$(echo "$total_cost + $(echo "$day_data" | jq -r '.total_cost')" | bc)
                    total_prompt=$((total_prompt + $(echo "$day_data" | jq -r '.total_prompt_tokens')))
                    total_completion=$((total_completion + $(echo "$day_data" | jq -r '.total_completion_tokens')))
                    total_requests=$((total_requests + $(echo "$day_data" | jq -r '.requests | length')))
                fi
            done
            
            jq -n \
                --argjson cost "$total_cost" \
                --argjson prompt "$total_prompt" \
                --argjson completion "$total_completion" \
                --argjson requests "$total_requests" \
                '{
                    period: "all",
                    total_cost: $cost,
                    total_prompt_tokens: $prompt,
                    total_completion_tokens: $completion,
                    total_requests: $requests
                }'
            ;;
        *)
            echo '{"error": "Invalid period. Use: today, week, month, or all"}'
            return 1
            ;;
    esac
}

# List models by category
openrouter::models::list_by_category() {
    local category="${1:-all}"
    local timeout="${OPENROUTER_TIMEOUT:-30}"
    
    if [[ -z "${OPENROUTER_API_KEY:-}" ]]; then
        if ! openrouter::init; then
            return 1
        fi
    fi
    
    local response
    response=$(timeout "$timeout" curl -s \
        -H "Authorization: Bearer ${OPENROUTER_API_KEY}" \
        "${OPENROUTER_API_BASE}/models" 2>/dev/null)
    
    if [[ -z "$response" ]]; then
        return 1
    fi
    
    case "$category" in
        code|coding|programming)
            echo "$response" | jq -r '.data[] | select(.id | contains("code") or contains("codestral") or contains("deepseek")) | .id'
            ;;
        chat|conversation)
            echo "$response" | jq -r '.data[] | select(.id | contains("chat") or contains("turbo") or contains("claude")) | .id'
            ;;
        vision|image)
            echo "$response" | jq -r '.data[] | select(.id | contains("vision") or contains("gpt-4o")) | .id'
            ;;
        cheap|budget)
            echo "$response" | jq -r '.data[] | select(.pricing.prompt < 0.0001) | .id'
            ;;
        fast|speed)
            echo "$response" | jq -r '.data[] | select(.id | contains("turbo") or contains("haiku") or contains("mistral-7b")) | .id'
            ;;
        all|*)
            echo "$response" | jq -r '.data[].id'
            ;;
    esac
}

# Test model with sample prompt
openrouter::models::test() {
    local model="${1}"
    local test_prompt="${2:-Hello, please respond with 'OK' if you receive this message.}"
    
    log::info "Testing model: $model"
    
    local response
    response=$(openrouter::models::execute_with_fallback "$model" "$test_prompt" 10)
    
    if [[ $? -eq 0 ]]; then
        local content=$(echo "$response" | jq -r '.choices[0].message.content // "No response"')
        log::success "Model test successful: $content"
        return 0
    else
        log::error "Model test failed"
        return 1
    fi
}
