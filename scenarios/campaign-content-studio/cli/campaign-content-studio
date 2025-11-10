#!/bin/bash
################################################################################
# Campaign Content Studio CLI - Thin Wrapper Version
# 
# A lightweight CLI that delegates all logic to the scenario's API.
# Port discovery uses ultra-fast file-based lookup.
################################################################################

set -e

# Configuration
SCENARIO_NAME="campaign-content-studio"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

################################################################################
# Port Discovery - Ultra-fast file-based lookup
################################################################################
get_api_url() {
    # Use the new ultra-fast port command
    local api_port
    api_port=$(vrooli scenario port "${SCENARIO_NAME}" API_PORT 2>/dev/null)
    
    if [[ -z "$api_port" ]]; then
        echo -e "${RED}❌ Error: ${SCENARIO_NAME} is not running${NC}" >&2
        echo "   Start it with: vrooli scenario run ${SCENARIO_NAME}" >&2
        exit 1
    fi
    
    echo "http://localhost:${api_port}"
}

################################################################################
# Helper Functions
################################################################################
usage() {
    echo -e "${CYAN}Campaign Content Studio CLI v2.0.0${NC}"
    echo "AI-powered content creation and campaign management"
    echo ""
    echo "Usage: campaign-content-studio [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo -e "  ${GREEN}campaigns${NC}   List all campaigns"
    echo -e "  ${GREEN}create${NC}      Create new campaign"
    echo -e "  ${GREEN}documents${NC}   List campaign documents"
    echo -e "  ${GREEN}search${NC}      Search campaign documents"
    echo -e "  ${GREEN}generate${NC}    Generate content for campaign"
    echo -e "  ${GREEN}health${NC}      Check service health"
    echo -e "  ${GREEN}status${NC}      Show system status"
    echo ""
    echo "Content Types:"
    echo "  blog_post        Blog post with SEO optimization"
    echo "  social_media     Social media posts (Twitter, LinkedIn, Instagram)"
    echo "  marketing_copy   Marketing emails, ads, landing pages"
    echo "  image           Generated images (requires ComfyUI)"
    echo ""
    echo "Examples:"
    echo "  campaign-content-studio campaigns"
    echo "  campaign-content-studio create \"Summer Campaign\" \"Q3 product launch\""
    echo "  campaign-content-studio documents \"550e8400-e29b-41d4-a716-446655440001\""
    echo "  campaign-content-studio search \"550e8400-e29b-41d4-a716-446655440001\" \"productivity tools\""
    echo "  campaign-content-studio generate \"550e8400-e29b-41d4-a716-446655440001\" \"blog_post\" \"Write about AI productivity tools\""
    echo ""
    echo "Options:"
    echo "  --help, -h       Show help for any command"
    echo "  --json           Output in JSON format (most commands)"
    echo ""
    echo "For more information: campaign-content-studio <command> --help"
}

# Format JSON output if jq is available
format_json() {
    if command -v jq >/dev/null 2>&1; then
        jq .
    else
        cat
    fi
}

# Make API request
api_request() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    local expected_status="${4:-200}"
    local api_url
    
    # Get the current API URL
    api_url=$(get_api_url)
    
    local curl_args=(-s -w "%{http_code}")
    
    if [[ "$method" != "GET" ]]; then
        curl_args+=(-X "$method")
    fi
    
    if [[ -n "$data" ]]; then
        curl_args+=(-H "Content-Type: application/json" -d "$data")
    fi
    
    local response
    response=$(curl "${curl_args[@]}" "${api_url}${endpoint}")
    
    local http_code="${response: -3}"
    local body="${response%???}"
    
    if [[ "$http_code" != "$expected_status" ]]; then
        echo -e "${RED}❌ API call failed: HTTP $http_code${NC}" >&2
        if command -v jq >/dev/null 2>&1 && echo "$body" | jq . >/dev/null 2>&1; then
            echo "$body" | jq . >&2
        else
            echo "$body" >&2
        fi
        return 1
    fi
    
    echo "$body"
}

################################################################################
# Command Implementations - All delegated to API
################################################################################

# List campaigns command
cmd_campaigns() {
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: campaign-content-studio campaigns [OPTIONS]"
                echo ""
                echo "List all campaigns"
                echo ""
                echo "Options:"
                echo "  --json    Output in JSON format"
                echo ""
                echo "Examples:"
                echo "  campaign-content-studio campaigns"
                echo "  campaign-content-studio campaigns --json"
                return 0
                ;;
            *)
                echo -e "${RED}❌ Unknown option: $1${NC}" >&2
                return 1
                ;;
        esac
    done
    
    echo -e "${BLUE}📋 Fetching campaigns...${NC}"
    
    local response
    response=$(api_request "GET" "/campaigns" "" "200")
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if command -v jq >/dev/null 2>&1; then
            local count
            count=$(echo "$response" | jq 'length' 2>/dev/null || echo "0")
            
            echo -e "${GREEN}✅ Found $count campaigns${NC}"
            
            if [[ $count -gt 0 ]]; then
                echo ""
                echo -e "${CYAN}📊 Campaigns:${NC}"
                echo "$response" | jq -r '.[] | "  • \(.name) (\(.id))\n    Description: \(.description)\n    Created: \(.created_at)"' 2>/dev/null
            fi
        else
            echo "$response" | format_json
        fi
    fi
}

# Create campaign command
cmd_create() {
    local name=""
    local description=""
    local json_output=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --description)
                description="$2"
                shift 2
                ;;
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: campaign-content-studio create <name> [OPTIONS]"
                echo ""
                echo "Create new campaign"
                echo ""
                echo "Options:"
                echo "  --description <desc>  Campaign description"
                echo "  --json                Output in JSON format"
                echo ""
                echo "Examples:"
                echo "  campaign-content-studio create \"Summer Campaign\""
                echo "  campaign-content-studio create \"Q3 Launch\" --description \"Product launch campaign\""
                return 0
                ;;
            -*)
                echo -e "${RED}❌ Unknown option: $1${NC}" >&2
                return 1
                ;;
            *)
                if [[ -z "$name" ]]; then
                    name="$1"
                elif [[ -z "$description" ]]; then
                    description="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo -e "${RED}❌ Error: Campaign name required${NC}" >&2
        echo "   Usage: campaign-content-studio create <name> [options]" >&2
        return 1
    fi
    
    echo -e "${BLUE}📝 Creating campaign: $name${NC}"
    
    # Build request JSON
    local request_body=$(cat <<EOF
{
    "name": "$name",
    "description": "$description",
    "settings": {}
}
EOF
)
    
    local response
    response=$(api_request "POST" "/campaigns" "$request_body" "201")
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if command -v jq >/dev/null 2>&1; then
            local campaign_id
            campaign_id=$(echo "$response" | jq -r '.id')
            
            echo -e "${GREEN}✅ Campaign created successfully!${NC}"
            echo "  Name: $name"
            echo "  ID: $campaign_id"
            [[ -n "$description" ]] && echo "  Description: $description"
            echo ""
            echo -e "${CYAN}💡 Next steps:${NC}"
            echo "  • Upload documents: Use the web interface or API"
            echo "  • Generate content: campaign-content-studio generate $campaign_id <type> <prompt>"
        else
            echo -e "${GREEN}✅ Campaign created successfully!${NC}"
            echo "$response" | format_json
        fi
    fi
}

# List documents command
cmd_documents() {
    local campaign_id=""
    local json_output=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: campaign-content-studio documents <campaign_id> [OPTIONS]"
                echo ""
                echo "List campaign documents"
                echo ""
                echo "Options:"
                echo "  --json    Output in JSON format"
                echo ""
                echo "Examples:"
                echo "  campaign-content-studio documents \"550e8400-e29b-41d4-a716-446655440001\""
                return 0
                ;;
            -*)
                echo -e "${RED}❌ Unknown option: $1${NC}" >&2
                return 1
                ;;
            *)
                if [[ -z "$campaign_id" ]]; then
                    campaign_id="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$campaign_id" ]]; then
        echo -e "${RED}❌ Error: Campaign ID required${NC}" >&2
        echo "   Usage: campaign-content-studio documents <campaign_id>" >&2
        return 1
    fi
    
    echo -e "${BLUE}📄 Fetching documents for campaign: $campaign_id${NC}"
    
    local response
    response=$(api_request "GET" "/campaigns/$campaign_id/documents" "" "200")
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if command -v jq >/dev/null 2>&1; then
            local count
            count=$(echo "$response" | jq 'length' 2>/dev/null || echo "0")
            
            echo -e "${GREEN}✅ Found $count documents${NC}"
            
            if [[ $count -gt 0 ]]; then
                echo ""
                echo -e "${CYAN}📄 Documents:${NC}"
                echo "$response" | jq -r '.[] | "  • \(.filename) (\(.content_type))\n    ID: \(.id)\n    Uploaded: \(.upload_date)"' 2>/dev/null
            else
                echo ""
                echo -e "${YELLOW}💡 No documents found. Upload documents using the web interface or API.${NC}"
            fi
        else
            echo "$response" | format_json
        fi
    fi
}

# Search documents command
cmd_search() {
    local campaign_id=""
    local query=""
    local limit=10
    local json_output=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --limit)
                limit="$2"
                shift 2
                ;;
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: campaign-content-studio search <campaign_id> <query> [OPTIONS]"
                echo ""
                echo "Search campaign documents using semantic search"
                echo ""
                echo "Options:"
                echo "  --limit <num>  Maximum number of results (default: 10)"
                echo "  --json         Output in JSON format"
                echo ""
                echo "Examples:"
                echo "  campaign-content-studio search \"550e8400-e29b-41d4-a716-446655440001\" \"productivity tools\""
                echo "  campaign-content-studio search \"550e8400-e29b-41d4-a716-446655440001\" \"AI automation\" --limit 5"
                return 0
                ;;
            -*)
                echo -e "${RED}❌ Unknown option: $1${NC}" >&2
                return 1
                ;;
            *)
                if [[ -z "$campaign_id" ]]; then
                    campaign_id="$1"
                elif [[ -z "$query" ]]; then
                    query="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$campaign_id" || -z "$query" ]]; then
        echo -e "${RED}❌ Error: Campaign ID and search query required${NC}" >&2
        echo "   Usage: campaign-content-studio search <campaign_id> <query>" >&2
        return 1
    fi
    
    echo -e "${BLUE}🔍 Searching documents in campaign: $campaign_id${NC}"
    echo -e "${CYAN}Query: $query${NC}"
    
    # Build request JSON
    local request_body=$(cat <<EOF
{
    "query": "$query",
    "limit": $limit
}
EOF
)
    
    local response
    response=$(api_request "POST" "/campaigns/$campaign_id/search" "$request_body" "200")
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        if command -v jq >/dev/null 2>&1; then
            # The response format depends on the API implementation
            # This is a basic interpretation - adjust based on actual API response
            echo -e "${GREEN}✅ Search completed${NC}"
            echo ""
            echo -e "${CYAN}🔍 Results:${NC}"
            echo "$response" | format_json
        else
            echo "$response" | format_json
        fi
    fi
}

# Generate content command
cmd_generate() {
    local campaign_id=""
    local content_type=""
    local prompt=""
    local include_images=false
    local json_output=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --include-images|--images)
                include_images=true
                shift
                ;;
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: campaign-content-studio generate <campaign_id> <content_type> <prompt> [OPTIONS]"
                echo ""
                echo "Generate content for a campaign"
                echo ""
                echo "Content Types:"
                echo "  blog_post        Blog post with SEO optimization"
                echo "  social_media     Social media posts (Twitter, LinkedIn, Instagram)"
                echo "  marketing_copy   Marketing emails, ads, landing pages"
                echo "  image           Generated images (requires ComfyUI)"
                echo ""
                echo "Options:"
                echo "  --include-images  Include image generation"
                echo "  --json            Output in JSON format"
                echo ""
                echo "Examples:"
                echo "  campaign-content-studio generate \"550e8400-e29b-41d4-a716-446655440001\" \"blog_post\" \"Write about AI productivity tools\""
                echo "  campaign-content-studio generate \"550e8400-e29b-41d4-a716-446655440001\" \"social_media\" \"Create Twitter post\" --include-images"
                return 0
                ;;
            -*)
                echo -e "${RED}❌ Unknown option: $1${NC}" >&2
                return 1
                ;;
            *)
                if [[ -z "$campaign_id" ]]; then
                    campaign_id="$1"
                elif [[ -z "$content_type" ]]; then
                    content_type="$1"
                elif [[ -z "$prompt" ]]; then
                    prompt="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$campaign_id" || -z "$content_type" || -z "$prompt" ]]; then
        echo -e "${RED}❌ Error: Campaign ID, content type, and prompt required${NC}" >&2
        echo "   Usage: campaign-content-studio generate <campaign_id> <content_type> <prompt>" >&2
        return 1
    fi
    
    # Validate content type
    case $content_type in
        blog_post|social_media|marketing_copy|image)
            ;;
        *)
            echo -e "${RED}❌ Error: Invalid content type: $content_type${NC}" >&2
            echo "   Valid types: blog_post, social_media, marketing_copy, image" >&2
            return 1
            ;;
    esac
    
    echo -e "${BLUE}🎨 Generating content...${NC}"
    echo -e "${CYAN}Campaign: $campaign_id${NC}"
    echo -e "${CYAN}Type: $content_type${NC}"
    echo -e "${CYAN}Prompt: $prompt${NC}"
    [[ "$include_images" == true ]] && echo -e "${CYAN}Including images: Yes${NC}"
    
    # Build request JSON
    local request_body=$(cat <<EOF
{
    "campaign_id": "$campaign_id",
    "content_type": "$content_type",
    "prompt": "$prompt",
    "include_images": $include_images
}
EOF
)
    
    local response
    response=$(api_request "POST" "/generate" "$request_body" "200")
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    if [[ $json_output == true ]]; then
        echo "$response" | format_json
    else
        echo -e "${GREEN}✅ Content generated successfully!${NC}"
        echo ""
        echo -e "${CYAN}📝 Generated Content:${NC}"
        echo "$response" | format_json
    fi
}

# Health check command
cmd_health() {
    local detailed=false
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --detailed)
                detailed=true
                shift
                ;;
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: campaign-content-studio health [OPTIONS]"
                echo ""
                echo "Check service health"
                echo ""
                echo "Options:"
                echo "  --detailed    Show detailed health information"
                echo "  --json        Output in JSON format"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo -e "${BLUE}🏥 Checking health...${NC}"
    
    local api_url
    api_url=$(get_api_url)
    
    local health_response
    health_response=$(curl -s "${api_url}/health")
    
    if [[ $json_output == true ]]; then
        echo "$health_response" | format_json
    else
        if echo "$health_response" | grep -q "healthy"; then
            echo -e "${GREEN}✅ Campaign Content Studio is healthy${NC}"
            echo "   API: ${api_url}"
            
            if [[ "$detailed" == true ]]; then
                echo ""
                echo -e "${CYAN}📋 Health Details:${NC}"
                if command -v jq >/dev/null 2>&1; then
                    echo "$health_response" | jq -r '
                        "  Status: \(.status)",
                        "  Service: \(.service)",
                        "  Version: \(.version)"
                    ' 2>/dev/null
                fi
            fi
        else
            echo -e "${RED}❌ Health check failed${NC}"
            echo "   Response: $health_response"
            return 1
        fi
    fi
}

# Status command (system overview)
cmd_status() {
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --json)
                json_output=true
                shift
                ;;
            --help|-h)
                echo "Usage: campaign-content-studio status [OPTIONS]"
                echo ""
                echo "Show system status"
                echo ""
                echo "Options:"
                echo "  --json    Output in JSON format"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    echo -e "${CYAN}📊 Campaign Content Studio Status${NC}"
    echo ""
    
    # Get API URL and check if running
    local api_url
    if api_url=$(get_api_url 2>/dev/null); then
        echo -e "${GREEN}✅ Service: Running${NC}"
        echo "   API: $api_url"
        
        # Get health for more details
        local health_response
        health_response=$(curl -s "${api_url}/health" 2>/dev/null)
        
        if [[ -n "$health_response" ]]; then
            echo ""
            echo -e "${CYAN}Service Details:${NC}"
            if command -v jq >/dev/null 2>&1; then
                echo "$health_response" | jq -r '
                    "  Status: \(.status)",
                    "  Service: \(.service)",
                    "  Version: \(.version)"
                ' 2>/dev/null || echo "  Status information unavailable"
            fi
        fi
        
        # Try to get campaign count
        local campaigns_response
        campaigns_response=$(curl -s "${api_url}/campaigns" 2>/dev/null)
        if [[ -n "$campaigns_response" ]] && command -v jq >/dev/null 2>&1; then
            local campaign_count
            campaign_count=$(echo "$campaigns_response" | jq 'length' 2>/dev/null || echo "0")
            echo "  Campaigns: $campaign_count"
        fi
    else
        echo -e "${RED}❌ Service: Not Running${NC}"
        echo "   Start with: vrooli scenario run campaign-content-studio"
    fi
}

################################################################################
# Main Command Router
################################################################################
main() {
    local command="${1:-}"
    
    if [[ -z "$command" ]]; then
        usage
        exit 0
    fi
    
    shift
    
    case "$command" in
        campaigns|list)
            cmd_campaigns "$@"
            ;;
        create)
            cmd_create "$@"
            ;;
        documents)
            cmd_documents "$@"
            ;;
        search)
            cmd_search "$@"
            ;;
        generate)
            cmd_generate "$@"
            ;;
        health)
            cmd_health "$@"
            ;;
        status)
            cmd_status "$@"
            ;;
        --help|-h|help)
            usage
            ;;
        --version|-v|version)
            echo "Campaign Content Studio CLI v2.0.0 (thin wrapper)"
            ;;
        *)
            echo -e "${RED}❌ Unknown command: $command${NC}" >&2
            usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"