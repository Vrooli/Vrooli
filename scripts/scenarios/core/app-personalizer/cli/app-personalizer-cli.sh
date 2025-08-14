#!/usr/bin/env bash
# App Personalizer CLI - Ultra-thin API wrapper

set -euo pipefail

# Version information
CLI_VERSION="1.0.0"

# Configuration
API_BASE="${APP_PERSONALIZER_API_BASE:-http://localhost:${SERVICE_PORT:-8300}}"
API_TOKEN="${APP_PERSONALIZER_TOKEN:-app_personalizer_cli_default_2024}"

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
        echo "app-personalizer CLI version $CLI_VERSION"
        ;;
    health) 
        api_call GET "/health" | format_output
        ;;
    list|apps) 
        api_call GET "/api/apps" | format_output
        ;;
    register) 
        [[ -z "${2:-}" || -z "${3:-}" || -z "${4:-}" || -z "${5:-}" ]] && { 
            echo "Usage: $0 register <app_name> <app_path> <app_type> <framework> [version]" >&2; 
            exit 1; 
        }
        local data="{\"app_name\": \"$2\", \"app_path\": \"$3\", \"app_type\": \"$4\", \"framework\": \"$5\""
        [[ -n "${6:-}" ]] && data+=", \"version\": \"$6\""
        data+="}"
        api_call POST "/api/apps/register" "$data" | format_output
        ;;
    analyze) 
        [[ -z "${2:-}" ]] && { echo "Usage: $0 analyze <app_id>" >&2; exit 1; }
        api_call POST "/api/apps/analyze" "{\"app_id\": \"$2\"}" | format_output
        ;;
    personalize) 
        [[ -z "${2:-}" || -z "${3:-}" || -z "${4:-}" ]] && { 
            echo "Usage: $0 personalize <app_id> <type> <deployment_mode> [persona_id] [brand_id]" >&2; 
            echo "Types: ui_theme, content, branding, behavior, full" >&2;
            echo "Modes: copy, patch, multi_tenant" >&2;
            exit 1; 
        }
        local data="{\"app_id\": \"$2\", \"personalization_type\": \"$3\", \"deployment_mode\": \"$4\""
        [[ -n "${5:-}" ]] && data+=", \"persona_id\": \"$5\""
        [[ -n "${6:-}" ]] && data+=", \"brand_id\": \"$6\""
        data+="}"
        api_call POST "/api/personalize" "$data" | format_output
        ;;
    backup) 
        [[ -z "${2:-}" ]] && { echo "Usage: $0 backup <app_path> [backup_type]" >&2; exit 1; }
        local backup_type="${3:-full}"
        api_call POST "/api/backup" "{\"app_path\": \"$2\", \"backup_type\": \"$backup_type\"}" | format_output
        ;;
    validate) 
        [[ -z "${2:-}" ]] && { echo "Usage: $0 validate <app_path> [test1,test2,...]" >&2; exit 1; }
        local tests="${3:-build,lint}"
        local tests_array="[\"$(echo "$tests" | sed 's/,/","/g')\"]"
        api_call POST "/api/validate" "{\"app_path\": \"$2\", \"tests\": $tests_array}" | format_output
        ;;
    api) 
        echo "$API_BASE"
        ;;
    help|--help|-h)
        cat <<-EOF
		App Personalizer CLI - Ultra-thin API wrapper

		Usage: $(basename "$0") <command> [options]

		Commands:
		  health                                      Check API health
		  list                                        List registered apps  
		  register <name> <path> <type> <framework> [version]  Register app
		  analyze <app_id>                           Analyze app for personalization points
		  personalize <app_id> <type> <mode> [persona_id] [brand_id]  Start personalization
		  backup <app_path> [type]                   Create app backup
		  validate <app_path> [tests]                Validate app (build,lint,test,startup)
		  api                                        Show API base URL
		  version                                    Show CLI version
		  help                                       Show this help

		Personalization Types:
		  ui_theme      Theme and styling changes
		  content       Content and text modifications
		  branding      Logo, colors, and brand assets
		  behavior      AI personality and interaction style
		  full          Complete personalization (all types)

		Deployment Modes:
		  copy          Create a copy of the app with modifications
		  patch         Apply modifications directly to app
		  multi_tenant  Configure for multi-tenant deployment

		Examples:
		  $(basename "$0") health
		  $(basename "$0") list
		  $(basename "$0") register "MyApp" "/apps/myapp" "generated" "react" "1.0.0"
		  $(basename "$0") analyze "123e4567-e89b-12d3-a456-426614174000"
		  $(basename "$0") personalize "123e4567-e89b-12d3-a456-426614174000" "ui_theme" "copy"
		  $(basename "$0") backup "/apps/myapp" "full"
		  $(basename "$0") validate "/apps/myapp" "build,lint"

		Environment:
		  APP_PERSONALIZER_API_BASE    API base URL (default: $API_BASE)
		  APP_PERSONALIZER_TOKEN       API token
		EOF
        ;;
    *) 
        echo "❌ Unknown command: $1. Use 'help' for usage." >&2
        exit 1
        ;;
esac