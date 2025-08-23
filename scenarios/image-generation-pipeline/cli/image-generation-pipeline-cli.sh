#!/usr/bin/env bash
# Image Generation Pipeline CLI - Ultra-thin API wrapper
# Enterprise-grade image generation with voice briefings and quality control

set -euo pipefail

# Version information
CLI_VERSION="1.0.0"

# Configuration
API_BASE="${IMAGE_GENERATION_PIPELINE_API_BASE:-http://localhost:${SERVICE_PORT:-8090}}"
API_TOKEN="${IMAGE_GENERATION_PIPELINE_TOKEN:-image_generation_pipeline_cli_default_2024}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

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
        echo -e "${RED}❌ Failed to connect to API at $API_BASE${NC}" >&2
        exit 1
    fi
}

# Pretty print JSON
pretty_json() {
    if command -v jq >/dev/null 2>&1; then
        jq '.'
    else
        cat
    fi
}

# Display help
show_help() {
    cat << 'EOF'
🎨 Image Generation Pipeline CLI

USAGE:
    image-generation-pipeline [COMMAND] [OPTIONS]

COMMANDS:
    campaigns                    List all campaigns
    campaigns create             Create a new campaign
    brands                       List all brands  
    brands create                Create a new brand
    generate                     Generate an image from prompt
    voice-brief                  Process voice briefing to text
    generations                  List image generations
    status                       Show service status
    help                         Show this help message

EXAMPLES:
    image-generation-pipeline campaigns
    image-generation-pipeline generate --campaign-id abc123 --prompt "Modern logo design"
    image-generation-pipeline generations --campaign-id abc123
    image-generation-pipeline voice-brief --file recording.wav

ENVIRONMENT VARIABLES:
    IMAGE_GENERATION_PIPELINE_API_BASE    API base URL (default: http://localhost:8090)
    IMAGE_GENERATION_PIPELINE_TOKEN       API authentication token
    SERVICE_PORT                          Service port (default: 8090)

EOF
}

# Check API health
check_status() {
    echo -e "${BLUE}🔍 Checking Image Generation Pipeline status...${NC}"
    
    if response=$(api_call "GET" "/health" 2>/dev/null); then
        echo -e "${GREEN}✅ Service is healthy${NC}"
        echo "$response" | pretty_json
    else
        echo -e "${RED}❌ Service is not responding${NC}"
        exit 1
    fi
}

# List campaigns
list_campaigns() {
    echo -e "${BLUE}📋 Fetching campaigns...${NC}"
    
    if response=$(api_call "GET" "/api/campaigns"); then
        echo "$response" | pretty_json
    else
        echo -e "${RED}❌ Failed to fetch campaigns${NC}"
        exit 1
    fi
}

# Create campaign
create_campaign() {
    local name="${1:-}"
    local brand_id="${2:-}"
    local description="${3:-}"
    
    if [[ -z "$name" || -z "$brand_id" ]]; then
        echo -e "${RED}❌ Usage: image-generation-pipeline campaigns create <name> <brand_id> [description]${NC}"
        exit 1
    fi
    
    local data
    data=$(cat << EOF
{
    "name": "$name",
    "brand_id": "$brand_id",
    "description": "$description",
    "status": "active"
}
EOF
)
    
    echo -e "${BLUE}📝 Creating campaign: $name${NC}"
    
    if response=$(api_call "POST" "/api/campaigns" "$data"); then
        echo -e "${GREEN}✅ Campaign created successfully${NC}"
        echo "$response" | pretty_json
    else
        echo -e "${RED}❌ Failed to create campaign${NC}"
        exit 1
    fi
}

# List brands
list_brands() {
    echo -e "${BLUE}🏷️ Fetching brands...${NC}"
    
    if response=$(api_call "GET" "/api/brands"); then
        echo "$response" | pretty_json
    else
        echo -e "${RED}❌ Failed to fetch brands${NC}"
        exit 1
    fi
}

# Create brand
create_brand() {
    local name="${1:-}"
    local description="${2:-}"
    
    if [[ -z "$name" ]]; then
        echo -e "${RED}❌ Usage: image-generation-pipeline brands create <name> [description]${NC}"
        exit 1
    fi
    
    local data
    data=$(cat << EOF
{
    "name": "$name",
    "description": "$description",
    "colors": ["#000000", "#FFFFFF"],
    "fonts": ["Arial", "Helvetica"]
}
EOF
)
    
    echo -e "${BLUE}🏷️ Creating brand: $name${NC}"
    
    if response=$(api_call "POST" "/api/brands" "$data"); then
        echo -e "${GREEN}✅ Brand created successfully${NC}"
        echo "$response" | pretty_json
    else
        echo -e "${RED}❌ Failed to create brand${NC}"
        exit 1
    fi
}

# Generate image
generate_image() {
    local campaign_id=""
    local prompt=""
    local style="modern"
    local dimensions="1024x1024"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --campaign-id)
                campaign_id="$2"
                shift 2
                ;;
            --prompt)
                prompt="$2"
                shift 2
                ;;
            --style)
                style="$2"
                shift 2
                ;;
            --dimensions)
                dimensions="$2"
                shift 2
                ;;
            *)
                echo -e "${RED}❌ Unknown option: $1${NC}"
                exit 1
                ;;
        esac
    done
    
    if [[ -z "$campaign_id" || -z "$prompt" ]]; then
        echo -e "${RED}❌ Usage: image-generation-pipeline generate --campaign-id <id> --prompt <prompt> [--style <style>] [--dimensions <dimensions>]${NC}"
        exit 1
    fi
    
    local data
    data=$(cat << EOF
{
    "campaign_id": "$campaign_id",
    "prompt": "$prompt",
    "style": "$style",
    "dimensions": "$dimensions"
}
EOF
)
    
    echo -e "${BLUE}🎨 Generating image: $prompt${NC}"
    
    if response=$(api_call "POST" "/api/generate" "$data"); then
        echo -e "${GREEN}✅ Image generation started${NC}"
        echo "$response" | pretty_json
    else
        echo -e "${RED}❌ Failed to start image generation${NC}"
        exit 1
    fi
}

# Process voice brief
process_voice_brief() {
    local file="${1:-}"
    
    if [[ -z "$file" || ! -f "$file" ]]; then
        echo -e "${RED}❌ Usage: image-generation-pipeline voice-brief --file <audio_file.wav>${NC}"
        exit 1
    fi
    
    # Convert audio file to base64
    local audio_data
    if ! audio_data=$(base64 -w 0 "$file"); then
        echo -e "${RED}❌ Failed to encode audio file${NC}"
        exit 1
    fi
    
    local format="${file##*.}"
    local data
    data=$(cat << EOF
{
    "audio_data": "$audio_data",
    "format": "$format"
}
EOF
)
    
    echo -e "${BLUE}🎙️ Processing voice brief: $file${NC}"
    
    if response=$(api_call "POST" "/api/voice-brief" "$data"); then
        echo -e "${GREEN}✅ Voice brief processed${NC}"
        echo "$response" | pretty_json
    else
        echo -e "${RED}❌ Failed to process voice brief${NC}"
        exit 1
    fi
}

# List generations
list_generations() {
    local campaign_id=""
    local status=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --campaign-id)
                campaign_id="$2"
                shift 2
                ;;
            --status)
                status="$2"
                shift 2
                ;;
            *)
                echo -e "${RED}❌ Unknown option: $1${NC}"
                exit 1
                ;;
        esac
    done
    
    local endpoint="/api/generations"
    local params=()
    
    [[ -n "$campaign_id" ]] && params+=("campaign_id=$campaign_id")
    [[ -n "$status" ]] && params+=("status=$status")
    
    if [[ ${#params[@]} -gt 0 ]]; then
        endpoint+="?$(IFS='&'; echo "${params[*]}")"
    fi
    
    echo -e "${BLUE}🖼️ Fetching image generations...${NC}"
    
    if response=$(api_call "GET" "$endpoint"); then
        echo "$response" | pretty_json
    else
        echo -e "${RED}❌ Failed to fetch generations${NC}"
        exit 1
    fi
}

# Main command dispatcher
main() {
    if [[ $# -eq 0 ]]; then
        show_help
        exit 0
    fi

    case "$1" in
        "help"|"--help"|"-h")
            show_help
            ;;
        "status")
            check_status
            ;;
        "campaigns")
            shift
            if [[ $# -eq 0 ]]; then
                list_campaigns
            elif [[ "$1" == "create" ]]; then
                shift
                create_campaign "$@"
            else
                echo -e "${RED}❌ Unknown campaigns command: $1${NC}"
                exit 1
            fi
            ;;
        "brands")
            shift
            if [[ $# -eq 0 ]]; then
                list_brands
            elif [[ "$1" == "create" ]]; then
                shift
                create_brand "$@"
            else
                echo -e "${RED}❌ Unknown brands command: $1${NC}"
                exit 1
            fi
            ;;
        "generate")
            shift
            generate_image "$@"
            ;;
        "voice-brief")
            shift
            if [[ "$1" == "--file" ]]; then
                shift
                process_voice_brief "$1"
            else
                echo -e "${RED}❌ Usage: image-generation-pipeline voice-brief --file <audio_file>${NC}"
                exit 1
            fi
            ;;
        "generations")
            shift
            list_generations "$@"
            ;;
        *)
            echo -e "${RED}❌ Unknown command: $1${NC}"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

main "$@"