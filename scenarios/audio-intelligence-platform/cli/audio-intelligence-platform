#!/usr/bin/env bash
# Audio Intelligence Platform CLI - Ultra-thin API wrapper

set -euo pipefail

# Version information
CLI_VERSION="1.0.0"

# Configuration
API_BASE="${AUDIO_INTELLIGENCE_PLATFORM_API_BASE:-http://localhost:${API_PORT:-8090}}"
API_TOKEN="${AUDIO_INTELLIGENCE_PLATFORM_TOKEN:-audio_intelligence_platform_cli_default_2024}"

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

# File upload helper
upload_file() {
    local file_path="$1"
    
    if [[ ! -f "$file_path" ]]; then
        echo "❌ File not found: $file_path" >&2
        exit 1
    fi
    
    local curl_args=("-s" "-X" "POST")
    curl_args+=("-H" "Authorization: Bearer $API_TOKEN")
    curl_args+=("-F" "audio=@$file_path")
    
    if ! curl "${curl_args[@]}" "$API_BASE/api/upload" 2>/dev/null; then
        echo "❌ Failed to upload file to API at $API_BASE" >&2
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
        echo "audio-intelligence-platform CLI version $CLI_VERSION"
        ;;
    health) 
        api_call GET "/health" | format_output
        ;;
    list|transcriptions) 
        api_call GET "/api/transcriptions" | format_output
        ;;
    get) 
        [[ -z "${2:-}" ]] && { echo "Usage: $0 get <transcription-id>" >&2; exit 1; }
        api_call GET "/api/transcriptions/$2" | format_output
        ;;
    upload) 
        [[ -z "${2:-}" ]] && { echo "Usage: $0 upload <audio-file>" >&2; exit 1; }
        upload_file "$2" | format_output
        ;;
    analyze) 
        [[ -z "${2:-}" || -z "${3:-}" ]] && { echo "Usage: $0 analyze <transcription-id> <analysis-type> [custom-prompt]" >&2; exit 1; }
        local data="{\"analysis_type\": \"$3\""
        [[ -n "${4:-}" ]] && data+=", \"custom_prompt\": \"$4\""
        data+="}"
        api_call POST "/api/transcriptions/$2/analyze" "$data" | format_output
        ;;
    analyses) 
        [[ -z "${2:-}" ]] && { echo "Usage: $0 analyses <transcription-id>" >&2; exit 1; }
        api_call GET "/api/transcriptions/$2/analyses" | format_output
        ;;
    search) 
        [[ -z "${2:-}" ]] && { echo "Usage: $0 search <query> [limit]" >&2; exit 1; }
        local data="{\"query\": \"$2\""
        [[ -n "${3:-}" ]] && data+=", \"limit\": $3"
        data+="}"
        api_call POST "/api/search" "$data" | format_output
        ;;
    api) 
        echo "$API_BASE"
        ;;
    help|--help|-h)
        cat <<-EOF
		Audio Intelligence Platform CLI - Ultra-thin API wrapper

		Usage: $(basename "$0") <command> [options]

		Commands:
		  health                              Check API health
		  list                                List all transcriptions  
		  get <transcription-id>              Get specific transcription
		  upload <audio-file>                 Upload audio file for transcription
		  analyze <id> <type> [prompt]        Analyze transcription
		  analyses <transcription-id>         Get all analyses for transcription
		  search <query> [limit]              Search transcriptions semantically
		  api                                 Show API base URL
		  version                             Show CLI version
		  help                                Show this help

		Analysis Types:
		  summary                             Generate concise summary
		  insights                            Extract key insights
		  custom                              Use custom prompt

		Examples:
		  $(basename "$0") health
		  $(basename "$0") list
		  $(basename "$0") upload meeting.mp3
		  $(basename "$0") get 123e4567-e89b-12d3-a456-426614174000
		  $(basename "$0") analyze 123e4567-e89b-12d3-a456-426614174000 summary
		  $(basename "$0") analyze 123e4567-e89b-12d3-a456-426614174000 custom "What are the action items?"
		  $(basename "$0") search "budget discussion" 5

		Environment:
		  AUDIO_INTELLIGENCE_PLATFORM_API_BASE    API base URL (default: $API_BASE)
		  AUDIO_INTELLIGENCE_PLATFORM_TOKEN       API token
		EOF
        ;;
    *) 
        echo "❌ Unknown command: $1. Use 'help' for usage." >&2
        exit 1
        ;;
esac