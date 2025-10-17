#!/usr/bin/env bash
# Windmill App Deployment Tool
# Future-ready script for programmatic app deployment

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

#######################################
# Display usage information
#######################################
usage() {
    cat << EOF
Windmill App Deployment Tool

This tool is prepared for future use when Windmill adds programmatic app creation API.
Currently, it helps prepare apps for manual import.

Usage:
  $0 check-api                    # Check if app API is available
  $0 prepare <app-name>          # Prepare app for manual import
  $0 deploy <app-json> <workspace>  # Deploy app (when API available)

Environment Variables:
  WINDMILL_API_URL   - Windmill API URL (default: http://localhost:5681)
  WINDMILL_API_TOKEN - API authentication token

Examples:
  $0 check-api
  $0 prepare admin-dashboard
  WINDMILL_API_TOKEN=xxx $0 deploy app.json my-workspace

Note: App deployment API is not yet available in Windmill.
      Monitor https://github.com/windmill-labs/windmill for updates.
EOF
}

#######################################
# Check if Windmill has app API
#######################################
check_api() {
    local api_url="${WINDMILL_API_URL:-http://localhost:5681}"
    local api_token="${WINDMILL_API_TOKEN:-}"
    
    echo -e "${YELLOW}Checking for Windmill app management API...${NC}"
    echo
    
    if [[ -z "$api_token" ]]; then
        echo -e "${YELLOW}Warning: No API token set. Some endpoints may require authentication.${NC}"
        echo "Set WINDMILL_API_TOKEN environment variable for authenticated checks."
        echo
    fi
    
    # Test potential app endpoints
    local endpoints=(
        "/api/apps"
        "/api/apps/list"
        "/api/w/apps"
        "/api/w/apps/list"
        "/api/workspaces/apps"
    )
    
    local found=false
    for endpoint in "${endpoints[@]}"; do
        echo -n "Testing ${api_url}${endpoint}... "
        
        local response
        local http_code
        
        if [[ -n "$api_token" ]]; then
            http_code=$(curl -s -o /dev/null -w "%{http_code}" \
                -H "Authorization: Bearer $api_token" \
                "${api_url}${endpoint}" 2>/dev/null || echo "000")
        else
            http_code=$(curl -s -o /dev/null -w "%{http_code}" \
                "${api_url}${endpoint}" 2>/dev/null || echo "000")
        fi
        
        case "$http_code" in
            200|201)
                echo -e "${GREEN}✓ Found (HTTP $http_code)${NC}"
                found=true
                ;;
            401|403)
                echo -e "${YELLOW}Authentication required (HTTP $http_code)${NC}"
                ;;
            404)
                echo -e "${RED}Not found (HTTP $http_code)${NC}"
                ;;
            000)
                echo -e "${RED}Connection failed${NC}"
                ;;
            *)
                echo -e "HTTP $http_code"
                ;;
        esac
    done
    
    echo
    if [[ "$found" == "true" ]]; then
        echo -e "${GREEN}App API endpoints detected!${NC}"
        echo "Windmill may have added app management API support."
        echo "Check the official documentation for usage."
    else
        echo -e "${YELLOW}No app API endpoints found.${NC}"
        echo "App creation still requires manual import through the UI."
        echo
        echo "Current workflow:"
        echo "1. Use 'prepare' command to export app definition"
        echo "2. Import manually through Windmill UI"
    fi
}

#######################################
# Prepare app for import
#######################################
prepare_app() {
    local app_name="$1"
    if [[ -z "$app_name" ]]; then
        echo -e "${RED}Error: App name required${NC}"
        echo "Usage: $0 prepare <app-name>"
        exit 1
    fi
    
    # Use the windmill manage script
    if [[ -x "${APP_ROOT}/resources/windmill/manage.sh" ]]; then
        "${APP_ROOT}/resources/windmill/manage.sh" --action prepare-app --app-name "$app_name"
    else
        echo -e "${RED}Error: Windmill manage script not found${NC}"
        exit 1
    fi
}

#######################################
# Deploy app (future functionality)
#######################################
deploy_app() {
    local app_file="$1"
    local workspace="$2"
    local api_url="${WINDMILL_API_URL:-http://localhost:5681}"
    local api_token="${WINDMILL_API_TOKEN:-}"
    
    echo -e "${YELLOW}App Deployment (Future Feature)${NC}"
    echo "================================"
    echo
    
    if [[ -z "$app_file" || -z "$workspace" ]]; then
        echo -e "${RED}Error: App file and workspace required${NC}"
        echo "Usage: $0 deploy <app-json> <workspace>"
        exit 1
    fi
    
    if [[ ! -f "$app_file" ]]; then
        echo -e "${RED}Error: App file not found: $app_file${NC}"
        exit 1
    fi
    
    if [[ -z "$api_token" ]]; then
        echo -e "${RED}Error: WINDMILL_API_TOKEN environment variable required${NC}"
        exit 1
    fi
    
    # Parse app name from file
    local app_name
    if command -v jq >/dev/null 2>&1; then
        app_name=$(jq -r '.name // "Unknown"' "$app_file")
    else
        app_name=$(basename "$app_file" .json)
    fi
    
    echo "App to deploy: $app_name"
    echo "Workspace: $workspace"
    echo "API URL: $api_url"
    echo
    
    echo -e "${YELLOW}Note: App deployment API is not yet available in Windmill.${NC}"
    echo
    echo "When available, the API call might look like:"
    echo
    echo "curl -X POST \\"
    echo "  -H \"Authorization: Bearer \$WINDMILL_API_TOKEN\" \\"
    echo "  -H \"Content-Type: application/json\" \\"
    echo "  -d @$app_file \\"
    echo "  ${api_url}/api/w/${workspace}/apps/create"
    echo
    echo "For now, please:"
    echo "1. Use 'prepare' command to export the app"
    echo "2. Import manually through Windmill UI at ${api_url}"
    echo
    echo "Monitor these resources for updates:"
    echo "- https://docs.windmill.dev"
    echo "- https://github.com/windmill-labs/windmill/releases"
}

#######################################
# Main execution
#######################################
main() {
    case "${1:-}" in
        check-api)
            check_api
            ;;
        prepare)
            prepare_app "${2:-}"
            ;;
        deploy)
            deploy_app "${2:-}" "${3:-}"
            ;;
        -h|--help|help)
            usage
            ;;
        *)
            echo -e "${RED}Error: Unknown command '${1:-}'${NC}"
            echo
            usage
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"