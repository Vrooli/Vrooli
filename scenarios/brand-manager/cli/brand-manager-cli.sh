#!/usr/bin/env bash
# Brand Manager CLI - Ultra-thin API wrapper
# Manages AI-powered brand generation and app integration

set -euo pipefail

# Version information
CLI_VERSION="2.0.0"

# Configuration
API_BASE="${BRAND_MANAGER_API_BASE:-http://localhost:${API_PORT:-8090}}"
API_TOKEN="${BRAND_MANAGER_TOKEN:-brand_manager_cli_default_2024}"

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
        echo "brand-manager CLI version $CLI_VERSION"
        ;;
    health) 
        api_call GET "/health" | format_output
        ;;
    list|brands) 
        local limit="${2:-20}"
        local offset="${3:-0}"
        api_call GET "/api/brands?limit=$limit&offset=$offset" | format_output
        ;;
    create|generate) 
        [[ -z "${2:-}" || -z "${3:-}" ]] && { 
            echo "Usage: $0 create <brand_name> <industry> [template] [logo_style] [color_scheme]" >&2
            exit 1
        }
        local data="{\"brand_name\": \"$2\", \"industry\": \"$3\""
        [[ -n "${4:-}" ]] && data+=", \"template\": \"$4\""
        [[ -n "${5:-}" ]] && data+=", \"logo_style\": \"$5\""
        [[ -n "${6:-}" ]] && data+=", \"color_scheme\": \"$6\""
        data+="}"
        api_call POST "/api/brands" "$data" | format_output
        ;;
    get|show) 
        [[ -z "${2:-}" ]] && { echo "Usage: $0 get <brand_id>" >&2; exit 1; }
        api_call GET "/api/brands/$2" | format_output
        ;;
    status) 
        [[ -z "${2:-}" ]] && { echo "Usage: $0 status <brand_name>" >&2; exit 1; }
        api_call GET "/api/brands/status/$2" | format_output
        ;;
    integrations) 
        local limit="${2:-20}"
        local offset="${3:-0}"
        api_call GET "/api/integrations?limit=$limit&offset=$offset" | format_output
        ;;
    integrate) 
        [[ -z "${2:-}" || -z "${3:-}" ]] && { 
            echo "Usage: $0 integrate <brand_id> <target_app_path> [integration_type] [create_backup]" >&2
            exit 1
        }
        local data="{\"brand_id\": \"$2\", \"target_app_path\": \"$3\""
        [[ -n "${4:-}" ]] && data+=", \"integration_type\": \"$4\""
        [[ -n "${5:-}" ]] && data+=", \"create_backup\": $5"
        data+="}"
        api_call POST "/api/integrations" "$data" | format_output
        ;;
    services|urls) 
        api_call GET "/api/services" | format_output
        ;;
    api) 
        echo "$API_BASE"
        ;;
    help|--help|-h)
        cat <<-EOF
		Brand Manager CLI - AI-powered brand generation and app integration

		Usage: $(basename "$0") <command> [options]

		Commands:
		  health                    Check API health
		  list [limit] [offset]     List all brands
		  create <name> <industry> [template] [logo_style] [color_scheme]
		                           Generate new brand
		  get <brand_id>           Get specific brand details
		  status <brand_name>      Check brand generation status
		  integrations [limit] [offset]  List integration requests
		  integrate <brand_id> <app_path> [type] [backup]
		                           Integrate brand into app
		  services                 Show service URLs and dashboards
		  api                      Show API base URL
		  version                  Show CLI version
		  help                     Show this help

		Examples:
		  $(basename "$0") health
		  $(basename "$0") list
		  $(basename "$0") create "TechCorp" "technology" "modern-tech" "minimalist" "primary"
		  $(basename "$0") get 123e4567-e89b-12d3-a456-426614174000
		  $(basename "$0") status "TechCorp"
		  $(basename "$0") integrations
		  $(basename "$0") integrate 123e4567-e89b-12d3-a456-426614174000 "/path/to/app" "full" true
		  $(basename "$0") services

		Brand Generation Templates:
		  - modern-tech           Modern technology company
		  - creative-agency       Creative design agency
		  - professional-services Professional services business
		  - startup-bold          Bold startup company
		  - healthcare-trust      Healthcare and medical services

		Logo Styles:
		  - minimalist       Clean, minimal design
		  - modern           Contemporary styling
		  - classic          Traditional approach
		  - bold             Strong, impactful design

		Color Schemes:
		  - primary          Brand primary colors
		  - monochrome       Black and white
		  - vibrant          Bright, energetic colors
		  - subtle           Muted, professional colors

		Integration Types:
		  - full             Complete brand integration
		  - partial          Selected components only
		  - theme-only       Theme and colors only

		Environment Variables:
		  BRAND_MANAGER_API_BASE    API base URL (default: http://localhost:8090)
		  BRAND_MANAGER_TOKEN       API authentication token (default: brand_manager_cli_default_2024)
		  API_PORT              Override default API port (8090)
		EOF
        ;;
    *) 
        echo "❌ Unknown command: $1. Use 'help' for usage." >&2
        exit 1
        ;;
esac