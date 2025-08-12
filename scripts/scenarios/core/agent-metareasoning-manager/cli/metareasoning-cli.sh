#!/usr/bin/env bash
# Metareasoning CLI - Thin wrapper for the simplified coordinator API
# Discovery-based implementation (no CRUD operations)

set -euo pipefail

# Configuration
CLI_VERSION="3.0.0"
API_BASE="${METAREASONING_API_BASE:-http://localhost:${SERVICE_PORT:-8090}}"
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

# Commands
cmd_health() {
    local response
    response=$(api_request GET "/health")
    echo "$response" | format_json
}

cmd_list() {
    local response
    response=$(api_request GET "/workflows")
    
    if [[ "${FORMAT:-table}" == "json" ]]; then
        echo "$response" | format_json
    else
        echo -e "${BLUE}Discovered Workflows:${NC}"
        if command -v jq >/dev/null 2>&1; then
            echo "$response" | jq -r '.[] | [.name, .platform, .category // "general", .usage_count // 0] | @tsv' 2>/dev/null | \
                column -t -s $'\t' 2>/dev/null || echo "$response" | format_json
        else
            echo "$response" | format_json
        fi
    fi
}

cmd_search() {
    [[ -z "${1:-}" ]] && error "Search query required"
    local query_json="{\"query\": \"$1\"}"
    api_request POST "/workflows/search" "$query_json" | format_json
}

cmd_execute() {
    [[ -z "${1:-}" ]] && error "Platform required (n8n or windmill)"
    [[ -z "${2:-}" ]] && error "Workflow ID required"
    [[ -z "${3:-}" ]] && error "Input data required"
    
    info "Executing workflow on $1..."
    api_request POST "/execute/$1/$2" "$3" | format_json
}

cmd_analyze() {
    [[ -z "${1:-}" ]] && error "Analysis type required (e.g., pros-cons, swot)"
    [[ -z "${2:-}" ]] && error "Input required"
    
    # Map analysis type to workflow ID
    local workflow_id=""
    case "$1" in
        pros-cons) workflow_id="pros-cons-analyzer" ;;
        swot) workflow_id="swot-analysis" ;;
        risk*) workflow_id="risk-assessment" ;;
        self-review) workflow_id="self-review" ;;
        reasoning-chain) workflow_id="reasoning-chain" ;;
        decision) workflow_id="decision-analyzer" ;;
        *) workflow_id="$1" ;;
    esac
    
    # Determine platform (most are n8n, decision is windmill)
    local platform="n8n"
    [[ "$1" == "decision" ]] && platform="windmill"
    
    local data="{\"input\": \"$2\", \"context\": \"${3:-}\", \"model\": \"${MODEL:-llama3.2}\"}"
    
    info "Running $1 analysis..."
    api_request POST "/execute/$platform/$workflow_id" "$data" | format_json
}

# Main command dispatcher
case "${1:-help}" in
    health) 
        cmd_health 
        ;;
    list|workflows|workflow) 
        shift
        cmd_list "$@" 
        ;;
    search) 
        shift
        cmd_search "$@" 
        ;;
    execute) 
        shift
        cmd_execute "$@" 
        ;;
    analyze) 
        shift
        cmd_analyze "$@" 
        ;;
    api) 
        echo "$API_BASE" 
        ;;
    version) 
        echo "Metareasoning Coordinator CLI v$CLI_VERSION" 
        ;;
    help|--help|-h)
        cat <<-EOF
		${BLUE}Metareasoning Coordinator CLI v$CLI_VERSION${NC}
		
		Usage: $(basename "$0") <command> [options]
		
		Commands:
		  health                    Check API health
		  list                      List discovered workflows
		  search <query>            Search workflows
		  execute <platform> <id> <input>  Execute workflow directly
		  analyze <type> <input> [context]  Run analysis workflow
		  api                       Show API base URL
		  version                   Show version
		  help                      Show this help
		
		Analysis Types:
		  pros-cons     Weighted pros and cons analysis
		  swot          SWOT analysis  
		  risk          Risk assessment
		  self-review   Self-review loop
		  decision      Decision analysis (Windmill)
		  reasoning-chain  Multi-step reasoning
		
		Examples:
		  $(basename "$0") health
		  $(basename "$0") list
		  $(basename "$0") search "risk assessment"
		  $(basename "$0") analyze pros-cons "Should we migrate to cloud?"
		  $(basename "$0") analyze swot "New product launch" "B2B market"
		  $(basename "$0") execute n8n pros-cons-analyzer '{"input": "Remote work policy"}'
		
		Environment:
		  METAREASONING_API_BASE  API base URL (default: $API_BASE)
		  METAREASONING_TOKEN     API token
		  FORMAT                  Output format (json|table)
		  MODEL                   AI model to use (default: llama3.2)
		EOF
        ;;
    *) 
        error "Unknown command: $1. Use 'help' for usage." 
        ;;
esac