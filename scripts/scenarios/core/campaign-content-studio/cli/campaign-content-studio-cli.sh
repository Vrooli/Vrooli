#!/usr/bin/env bash
# Campaign Content Studio CLI - Ultra-thin API wrapper
# This is what the CLI should look like - minimal and focused

set -euo pipefail

# Version information
CLI_VERSION="2.0.0"

# Configuration
API_BASE="${CAMPAIGN_CONTENT_STUDIO_API_BASE:-http://localhost:${SERVICE_PORT:-8090}}"
API_TOKEN="${CAMPAIGN_CONTENT_STUDIO_TOKEN:-campaign_content_studio_cli_default_2024}"

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
        echo "campaign-content-studio CLI version $CLI_VERSION"
        ;;
    health) 
        api_call GET "/health" | format_output
        ;;
    campaigns|list) 
        api_call GET "/campaigns" | format_output
        ;;
    create) 
        [[ -z "${2:-}" ]] && { echo "Usage: $0 create <name> [description]" >&2; exit 1; }
        local data="{\"name\": \"$2\""
        [[ -n "${3:-}" ]] && data+=", \"description\": \"$3\""
        data+="}"
        api_call POST "/campaigns" "$data" | format_output
        ;;
    documents) 
        [[ -z "${2:-}" ]] && { echo "Usage: $0 documents <campaign_id>" >&2; exit 1; }
        api_call GET "/campaigns/$2/documents" | format_output
        ;;
    search) 
        [[ -z "${2:-}" || -z "${3:-}" ]] && { echo "Usage: $0 search <campaign_id> <query>" >&2; exit 1; }
        api_call POST "/campaigns/$2/search" "{\"query\": \"$3\"}" | format_output
        ;;
    generate) 
        [[ -z "${2:-}" || -z "${3:-}" || -z "${4:-}" ]] && { 
            echo "Usage: $0 generate <campaign_id> <content_type> <prompt> [include_images]" >&2
            exit 1
        }
        local data="{\"campaign_id\": \"$2\", \"content_type\": \"$3\", \"prompt\": \"$4\""
        [[ "${5:-}" == "true" ]] && data+=", \"include_images\": true"
        data+="}"
        api_call POST "/generate" "$data" | format_output
        ;;
    api) 
        echo "$API_BASE"
        ;;
    help|--help|-h)
        cat <<-EOF
		Campaign Content Studio CLI - Ultra-thin API wrapper

		Usage: $(basename "$0") <command> [options]

		Commands:
		  health                              Check API health
		  campaigns                           List all campaigns  
		  create <name> [description]         Create new campaign
		  documents <campaign_id>             List campaign documents
		  search <campaign_id> <query>        Search campaign documents
		  generate <campaign_id> <type> <prompt> [images]  Generate content
		  api                                 Show API base URL
		  version                             Show CLI version
		  help                                Show this help

		Content Types:
		  blog_post        Blog post with SEO optimization
		  social_media     Social media posts (Twitter, LinkedIn, Instagram)
		  marketing_copy   Marketing emails, ads, landing pages
		  image            Generated images (requires ComfyUI)

		Examples:
		  $(basename "$0") health
		  $(basename "$0") campaigns
		  $(basename "$0") create "Summer Campaign" "Q3 product launch"
		  $(basename "$0") documents "550e8400-e29b-41d4-a716-446655440001"
		  $(basename "$0") search "550e8400-e29b-41d4-a716-446655440001" "productivity tools"
		  $(basename "$0") generate "550e8400-e29b-41d4-a716-446655440001" "blog_post" "Write about AI productivity tools"
		  $(basename "$0") generate "550e8400-e29b-41d4-a716-446655440001" "social_media" "Create Twitter post" true

		Environment:
		  CAMPAIGN_CONTENT_STUDIO_API_BASE    API base URL (default: $API_BASE)
		  CAMPAIGN_CONTENT_STUDIO_TOKEN       API token
		EOF
        ;;
    *) 
        echo "❌ Unknown command: $1. Use 'help' for usage." >&2
        exit 1
        ;;
esac