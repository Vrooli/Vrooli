#!/bin/bash
# OpenRouter usage tracking and analytics

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_LIB_DIR="${APP_ROOT}/resources/openrouter/lib"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${OPENROUTER_LIB_DIR}/models.sh"

# Main usage command
openrouter::usage() {
    local period="${1:-today}"
    
    case "$period" in
        today|week|month|all)
            openrouter::usage::show "$period"
            ;;
        --help|help)
            openrouter::usage::help
            ;;
        *)
            log::error "Invalid period: $period"
            openrouter::usage::help
            return 1
            ;;
    esac
}

# Show usage for specific period
openrouter::usage::show() {
    local period="${1:-today}"
    
    # Capitalize period for display
    local period_display="${period^}"
    log::info "OpenRouter Usage Analytics - $period_display"
    echo
    
    local analytics=$(openrouter::models::get_usage_analytics "$period")
    
    if echo "$analytics" | jq -e '.error' >/dev/null 2>&1; then
        log::warn "No usage data available for period: $period"
        return 1
    fi
    
    # Format and display the analytics
    local total_cost=$(echo "$analytics" | jq -r '.total_cost // 0')
    local total_prompt=$(echo "$analytics" | jq -r '.total_prompt_tokens // 0')
    local total_completion=$(echo "$analytics" | jq -r '.total_completion_tokens // 0')
    local total_requests=$(echo "$analytics" | jq -r '.total_requests // .requests // [] | if type == "array" then length else . end')
    
    echo "┌─────────────────────────────────────────┐"
    echo "│         Usage Summary                   │"
    echo "├─────────────────────────────────────────┤"
    printf "│ Period:     %-28s│\n" "$period"
    printf "│ Requests:   %-28s│\n" "$total_requests"
    printf "│ Cost:       \$%-27.4f│\n" "$total_cost"
    printf "│ Prompt:     %-28s│\n" "$total_prompt tokens"
    printf "│ Completion: %-28s│\n" "$total_completion tokens"
    echo "└─────────────────────────────────────────┘"
    
    # Show top models used if data is available
    if [[ "$period" == "today" ]] && [[ -f "${var_ROOT_DIR}/data/openrouter/usage/usage-$(date +%Y-%m-%d).json" ]]; then
        echo
        echo "Top Models Used:"
        jq -r '.requests | group_by(.model) | map({model: .[0].model, count: length}) | sort_by(.count) | reverse | limit(5;.) | .[] | "  • \(.model): \(.count) requests"' \
            "${var_ROOT_DIR}/data/openrouter/usage/usage-$(date +%Y-%m-%d).json" 2>/dev/null || true
    fi
}

# Usage for today
openrouter::usage::today() {
    openrouter::usage::show "today"
}

# Usage for week
openrouter::usage::week() {
    openrouter::usage::show "week"
}

# Usage for month
openrouter::usage::month() {
    openrouter::usage::show "month"
}

# Usage for all time
openrouter::usage::all() {
    openrouter::usage::show "all"
}

# Show help
openrouter::usage::help() {
    cat <<EOF
OpenRouter Usage Analytics

USAGE:
    resource-openrouter usage [period]

PERIODS:
    today    Show usage for today
    week     Show usage for last 7 days
    month    Show usage for last 30 days
    all      Show all-time usage

EXAMPLES:
    resource-openrouter usage           # Show today's usage
    resource-openrouter usage week      # Show last 7 days
    resource-openrouter usage month     # Show last 30 days

NOTES:
    - Usage data is tracked automatically for all API calls
    - Costs are estimated based on OpenRouter pricing
    - Data is stored locally in JSON format
EOF
}

# Export functions
export -f openrouter::usage
export -f openrouter::usage::show
export -f openrouter::usage::today
export -f openrouter::usage::week
export -f openrouter::usage::month
export -f openrouter::usage::all
export -f openrouter::usage::help