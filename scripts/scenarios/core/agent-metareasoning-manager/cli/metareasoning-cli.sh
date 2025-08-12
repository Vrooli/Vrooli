#!/usr/bin/env bash
# Metareasoning CLI - Thin wrapper for the API
# Ultra-minimal implementation focusing on API calls only

set -euo pipefail

# Configuration
CLI_VERSION="2.0.0"
API_BASE="${METAREASONING_API_BASE:-http://localhost:${SERVICE_PORT:-8093}}"
API_TOKEN="${METAREASONING_TOKEN:-metareasoning_cli_default_2024}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Helper functions
error() { echo -e "${RED}❌ $*${NC}" >&2; exit 1; }
success() { echo -e "${GREEN}✅ $*${NC}"; }
info() { echo -e "${BLUE}ℹ️  $*${NC}"; }

# API request wrapper
api_request() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    
    local curl_args=("-s" "-X" "$method")
    curl_args+=("-H" "Content-Type: application/json")
    curl_args+=("-H" "Authorization: Bearer $API_TOKEN")
    
    [[ -n "$data" ]] && curl_args+=("-d" "$data")
    
    local response
    if ! response=$(curl "${curl_args[@]}" "$API_BASE$endpoint" 2>/dev/null); then
        error "Failed to connect to API at $API_BASE"
    fi
    
    echo "$response"
}

# Format JSON output
format_json() {
    if command -v jq >/dev/null 2>&1; then
        jq '.' 2>/dev/null || cat
    else
        cat
    fi
}

# Format table output
format_table() {
    if command -v jq >/dev/null 2>&1; then
        jq -r '.workflows[] | [.name, .type, .platform, .usage_count] | @tsv' 2>/dev/null | \
            column -t -s $'\t' 2>/dev/null || format_json
    else
        format_json
    fi
}

# Commands
cmd_health() {
    local response
    response=$(api_request GET "/health")
    echo "$response" | format_json
}

cmd_list() {
    local query=""
    [[ -n "${1:-}" ]] && query="?platform=$1"
    [[ -n "${2:-}" ]] && query="${query}&type=$2"
    
    local response
    response=$(api_request GET "/workflows$query")
    
    if [[ "${FORMAT:-table}" == "json" ]]; then
        echo "$response" | format_json
    else
        echo -e "${BLUE}Available Workflows:${NC}"
        echo "$response" | format_table
    fi
}

cmd_get() {
    [[ -z "${1:-}" ]] && error "Workflow ID required"
    api_request GET "/workflows/$1" | format_json
}

cmd_create() {
    [[ -z "${1:-}" ]] && error "Workflow JSON required"
    api_request POST "/workflows" "$1" | format_json
}

cmd_update() {
    [[ -z "${1:-}" ]] && error "Workflow ID required"
    [[ -z "${2:-}" ]] && error "Workflow JSON required"
    api_request PUT "/workflows/$1" "$2" | format_json
}

cmd_delete() {
    [[ -z "${1:-}" ]] && error "Workflow ID required"
    api_request DELETE "/workflows/$1" | format_json
    success "Workflow deactivated"
}

cmd_execute() {
    [[ -z "${1:-}" ]] && error "Workflow ID required"
    [[ -z "${2:-}" ]] && error "Input JSON required"
    
    info "Executing workflow..."
    api_request POST "/workflows/$1/execute" "$2" | format_json
}

cmd_analyze() {
    [[ -z "${1:-}" ]] && error "Analysis type required"
    [[ -z "${2:-}" ]] && error "Input required"
    
    local data="{\"input\": \"$2\", \"context\": \"${3:-}\", \"model\": \"${MODEL:-llama3.2}\"}"
    
    info "Running $1 analysis..."
    api_request POST "/analyze/$1" "$data" | format_json
}

cmd_search() {
    [[ -z "${1:-}" ]] && error "Search query required"
    api_request GET "/workflows/search?q=$1" | format_json
}

cmd_generate() {
    [[ -z "${1:-}" ]] && error "Prompt required"
    local platform="${2:-n8n}"
    local data="{\"prompt\": \"$1\", \"platform\": \"$platform\"}"
    
    info "Generating workflow..."
    api_request POST "/workflows/generate" "$data" | format_json
}

cmd_import() {
    [[ -z "${1:-}" ]] && error "Platform required"
    [[ -z "${2:-}" ]] && error "Workflow data required"
    
    local data="{\"platform\": \"$1\", \"data\": $2}"
    api_request POST "/workflows/import" "$data" | format_json
}

cmd_export() {
    [[ -z "${1:-}" ]] && error "Workflow ID required"
    local format="${2:-json}"
    api_request GET "/workflows/$1/export?format=$format" | format_json
}

cmd_clone() {
    [[ -z "${1:-}" ]] && error "Workflow ID required"
    local name="${2:-}"
    local data="{\"name\": \"$name\"}"
    api_request POST "/workflows/$1/clone" "$data" | format_json
}

cmd_history() {
    [[ -z "${1:-}" ]] && error "Workflow ID required"
    local limit="${2:-50}"
    api_request GET "/workflows/$1/history?limit=$limit" | format_json
}

cmd_metrics() {
    [[ -z "${1:-}" ]] && error "Workflow ID required"
    api_request GET "/workflows/$1/metrics" | format_json
}

cmd_models() {
    api_request GET "/models" | format_json
}

cmd_platforms() {
    api_request GET "/platforms" | format_json
}

cmd_stats() {
    api_request GET "/stats" | format_json
}

# Show usage
show_usage() {
    cat << EOF
Metareasoning CLI v3.0.0

Usage: $(basename "$0") <command> [args]

Core Commands:
  health                      Check API health
  list [platform] [type]      List workflows
  get <id>                    Get workflow details
  create <json>               Create new workflow
  update <id> <json>          Update workflow
  delete <id>                 Delete workflow (soft)
  execute <id> <json>         Execute workflow

Advanced Commands:
  search <query>              Search workflows
  generate <prompt> [plat]    Generate workflow from prompt
  import <platform> <data>    Import workflow
  export <id> [format]        Export workflow
  clone <id> [name]           Clone workflow
  history <id> [limit]        Get execution history
  metrics <id>                Get performance metrics

System Commands:
  models                      List available AI models
  platforms                   List execution platforms
  stats                       System statistics
  analyze <type> <input>      Quick analysis

Environment:
  API_BASE:    $API_BASE
  API_TOKEN:   ${API_TOKEN:0:10}...

Examples:
  $(basename "$0") search "risk assessment"
  $(basename "$0") generate "Create a workflow that analyzes sentiment"
  $(basename "$0") export abc-123 n8n
  $(basename "$0") history abc-123 100

EOF
}

# Main
main() {
    local cmd="${1:-help}"
    shift || true
    
    case "$cmd" in
        # Core commands
        health) cmd_health "$@" ;;
        list)   cmd_list "$@" ;;
        get)    cmd_get "$@" ;;
        create) cmd_create "$@" ;;
        update) cmd_update "$@" ;;
        delete) cmd_delete "$@" ;;
        execute) cmd_execute "$@" ;;
        
        # Advanced commands
        search) cmd_search "$@" ;;
        generate) cmd_generate "$@" ;;
        import) cmd_import "$@" ;;
        export) cmd_export "$@" ;;
        clone) cmd_clone "$@" ;;
        history) cmd_history "$@" ;;
        metrics) cmd_metrics "$@" ;;
        
        # System commands
        models) cmd_models "$@" ;;
        platforms) cmd_platforms "$@" ;;
        stats) cmd_stats "$@" ;;
        analyze) cmd_analyze "$@" ;;
        
        help|--help|-h) show_usage ;;
        *) error "Unknown command: $cmd" ;;
    esac
}

main "$@"