#!/usr/bin/env bash
# Agent Metareasoning Manager CLI - Ultra-thin API wrapper
# This is what the CLI should look like - minimal and focused

set -euo pipefail

# Version information
CLI_VERSION="2.0.0"

# Configuration
API_BASE="${AGENT_METAREASONING_MANAGER_API_BASE:-http://localhost:${API_PORT:-8090}}"
API_TOKEN="${AGENT_METAREASONING_MANAGER_TOKEN:-agent_metareasoning_manager_cli_default_2024}"

# API request helper
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    
    local curl_args=("-s" "-X" "$method")
    curl_args+=("-H" "Content-Type: application/json")
    curl_args+=("-H" "Authorization: Bearer $API_TOKEN")
    
    [[ -n "$data" ]] && curl_args+=("-d" "$data")
    
    if ! curl "${curl_args[@]}" "$API_BASE$endpoint" 2>/dev/null; then
        echo "❌ Failed to connect to API at $API_BASE" >&2
        exit 1
    fi
}

# Format output with jq if available, otherwise raw
format_output() {
    if command -v jq >/dev/null 2>&1; then
        jq '.' 2>/dev/null || cat
    else
        cat
    fi
}

# Main command dispatcher
case "${1:-help}" in
    version|--version|-v)
        echo "agent-metareasoning-manager CLI version $CLI_VERSION"
        ;;
    health) 
        api_call GET "/health" | format_output
        ;;
    list|workflows) 
        api_call GET "/workflows" | format_output
        ;;
    search) 
        [[ -z "${2:-}" ]] && { echo "Usage: $0 search <query>" >&2; exit 1; }
        api_call POST "/workflows/search" "{\"query\": \"$2\"}" | format_output
        ;;
    analyze) 
        [[ -z "${2:-}" || -z "${3:-}" ]] && { echo "Usage: $0 analyze <type> <input> [context]" >&2; exit 1; }
        local data="{\"type\": \"$2\", \"input\": \"$3\""
        [[ -n "${4:-}" ]] && data+=", \"context\": \"$4\""
        data+="}"
        api_call POST "/analyze" "$data" | format_output
        ;;
    api) 
        echo "$API_BASE"
        ;;
    help|--help|-h)
        cat <<-EOF
		Agent Metareasoning Manager CLI - Ultra-thin API wrapper

		Usage: $(basename "$0") <command> [options]

		Commands:
		  health                    Check API health
		  list                      List discovered workflows  
		  search <query>            Search workflows
		  analyze <type> <input> [context]  Run analysis
		  api                       Show API base URL
		  version                   Show CLI version
		  help                      Show this help

		Examples:
		  $(basename "$0") health
		  $(basename "$0") list
		  $(basename "$0") search "risk assessment"
		  $(basename "$0") analyze pros-cons "Should we migrate to cloud?"
		  $(basename "$0") analyze swot "New product" "competitive market"

		Environment:
		  AGENT_METAREASONING_MANAGER_API_BASE    API base URL (default: $API_BASE)
		  AGENT_METAREASONING_MANAGER_TOKEN       API token
		EOF
        ;;
    *) 
        echo "❌ Unknown command: $1. Use 'help' for usage." >&2
        exit 1
        ;;
esac