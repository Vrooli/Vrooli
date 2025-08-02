#!/bin/bash
# ====================================================================
# Windmill UI Deployment Script for Multi-Modal AI Assistant
# ====================================================================
# 
# This script deploys the Multi-Modal AI Assistant UI components to
# a running Windmill instance as part of the test scenario execution.
#
# Usage:
#   ./deploy-ui.sh [options]
#
# Options:
#   --windmill-url URL     Windmill base URL (default: http://localhost:5681)
#   --workspace WORKSPACE  Windmill workspace (default: demo)
#   --username USERNAME    Windmill username (default: admin@windmill.dev)
#   --password PASSWORD    Windmill password (default: changeme)
#   --app-name NAME        Application name (default: multimodal-assistant)
#   --cleanup             Remove existing app before deployment
#   --validate            Validate deployment after completion
#   --verbose             Enable verbose logging
#
# ====================================================================

set -euo pipefail

# Script directory and paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
UI_SCRIPTS_DIR="$SCRIPT_DIR/scripts"
APP_DEFINITION_FILE="$SCRIPT_DIR/multimodal-assistant-app.json"

# Default configuration
WINDMILL_URL="${WINDMILL_URL:-http://localhost:5681}"
WORKSPACE="${WORKSPACE:-demo}"
USERNAME="${USERNAME:-admin@windmill.dev}"
PASSWORD="${PASSWORD:-changeme}"
APP_NAME="${APP_NAME:-multimodal-assistant}"
CLEANUP=false
VALIDATE=false
VERBOSE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${BLUE}[VERBOSE]${NC} $1"
    fi
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --windmill-url)
                WINDMILL_URL="$2"
                shift 2
                ;;
            --workspace)
                WORKSPACE="$2"
                shift 2
                ;;
            --username)
                USERNAME="$2"
                shift 2
                ;;
            --password)
                PASSWORD="$2"
                shift 2
                ;;
            --app-name)
                APP_NAME="$2"
                shift 2
                ;;
            --cleanup)
                CLEANUP=true
                shift
                ;;
            --validate)
                VALIDATE=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

show_help() {
    echo "Windmill UI Deployment Script for Multi-Modal AI Assistant"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --windmill-url URL     Windmill base URL (default: http://localhost:5681)"
    echo "  --workspace WORKSPACE  Windmill workspace (default: demo)"
    echo "  --username USERNAME    Windmill username (default: admin@windmill.dev)"
    echo "  --password PASSWORD    Windmill password (default: changeme)"
    echo "  --app-name NAME        Application name (default: multimodal-assistant)"
    echo "  --cleanup             Remove existing app before deployment"
    echo "  --validate            Validate deployment after completion"
    echo "  --verbose             Enable verbose logging"
    echo "  --help                Show this help message"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check required tools
    local missing_tools=()
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_tools+=("curl")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing_tools+=("jq")
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_error "Please install the missing tools and try again"
        exit 1
    fi
    
    # Check if files exist
    if [[ ! -f "$APP_DEFINITION_FILE" ]]; then
        log_error "App definition file not found: $APP_DEFINITION_FILE"
        exit 1
    fi
    
    if [[ ! -d "$UI_SCRIPTS_DIR" ]]; then
        log_error "UI scripts directory not found: $UI_SCRIPTS_DIR"
        exit 1
    fi
    
    # Check if Windmill is accessible
    if ! curl -sf "$WINDMILL_URL/api/version" >/dev/null 2>&1; then
        log_error "Cannot connect to Windmill at $WINDMILL_URL"
        log_error "Please ensure Windmill is running and accessible"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Authenticate with Windmill
authenticate() {
    log_info "Authenticating with Windmill..."
    
    local auth_response
    auth_response=$(curl -s -X POST "$WINDMILL_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$USERNAME\",\"password\":\"$PASSWORD\"}" || echo '{"error":"auth_failed"}')
    
    if echo "$auth_response" | jq -e '.token' >/dev/null 2>&1; then
        AUTH_TOKEN=$(echo "$auth_response" | jq -r '.token')
        log_success "Authentication successful"
        log_verbose "Auth token obtained: ${AUTH_TOKEN:0:20}..."
    else
        log_error "Authentication failed"
        log_verbose "Auth response: $auth_response"
        exit 1
    fi
}

# Check if workspace exists, create if needed
ensure_workspace() {
    log_info "Ensuring workspace '$WORKSPACE' exists..."
    
    local workspace_response
    workspace_response=$(curl -s -H "Authorization: Bearer $AUTH_TOKEN" \
        "$WINDMILL_URL/api/w/$WORKSPACE" || echo '{"error":"workspace_check_failed"}')
    
    if echo "$workspace_response" | jq -e '.id' >/dev/null 2>&1; then
        log_success "Workspace '$WORKSPACE' exists"
    else {
        log_info "Creating workspace '$WORKSPACE'..."
        
        local create_response
        create_response=$(curl -s -X POST \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -H "Content-Type: application/json" \
            "$WINDMILL_URL/api/workspaces" \
            -d "{\"id\":\"$WORKSPACE\",\"name\":\"$WORKSPACE\"}" || echo '{"error":"workspace_creation_failed"}')
        
        if echo "$create_response" | jq -e '.id' >/dev/null 2>&1; then
            log_success "Workspace '$WORKSPACE' created"
        else
            log_error "Failed to create workspace '$WORKSPACE'"
            log_verbose "Create response: $create_response"
            exit 1
        fi
    }
}

# Deploy individual TypeScript scripts
deploy_scripts() {
    log_info "Deploying TypeScript scripts..."
    
    local script_files=(
        "whisper-transcribe.ts"
        "ollama-analyze.ts"
        "comfyui-generate.ts"
        "agent-s2-automation.ts"
        "process-multimodal-request.ts"
        "take-screenshot.ts"
        "clear-session.ts"
    )
    
    local deployed_count=0
    
    for script_file in "${script_files[@]}"; do
        local script_path="$UI_SCRIPTS_DIR/$script_file"
        local script_name="${script_file%.ts}"
        
        if [[ ! -f "$script_path" ]]; then
            log_warning "Script file not found: $script_path"
            continue
        fi
        
        log_verbose "Deploying script: $script_name"
        
        # Read script content
        local script_content
        script_content=$(cat "$script_path")
        
        # Create script payload
        local script_payload
        script_payload=$(jq -n \
            --arg path "$script_name" \
            --arg content "$script_content" \
            --arg language "typescript" \
            --arg description "Multi-Modal AI Assistant - $script_name" \
            '{
                path: $path,
                content: $content,
                language: $language,
                description: $description,
                is_template: false
            }')
        
        # Deploy script
        local deploy_response
        deploy_response=$(curl -s -X POST \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -H "Content-Type: application/json" \
            "$WINDMILL_URL/api/w/$WORKSPACE/scripts" \
            -d "$script_payload" || echo '{"error":"script_deployment_failed"}')
        
        if echo "$deploy_response" | jq -e '.path' >/dev/null 2>&1; then
            log_success "Deployed script: $script_name"
            deployed_count=$((deployed_count + 1))
        else
            log_error "Failed to deploy script: $script_name"
            log_verbose "Deploy response: $deploy_response"
        fi
    done
    
    log_info "Deployed $deployed_count/${#script_files[@]} scripts"
    
    if [[ $deployed_count -eq 0 ]]; then
        log_error "No scripts were successfully deployed"
        exit 1
    fi
}

# Cleanup existing app if requested
cleanup_existing_app() {
    if [[ "$CLEANUP" == "true" ]]; then
        log_info "Cleaning up existing app '$APP_NAME'..."
        
        local delete_response
        delete_response=$(curl -s -X DELETE \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            "$WINDMILL_URL/api/w/$WORKSPACE/apps/$APP_NAME" || echo '{"status":"not_found"}')
        
        log_success "Cleanup completed (app may not have existed)"
    fi
}

# Deploy the main application
deploy_app() {
    log_info "Deploying application '$APP_NAME'..."
    
    # Read and validate app definition
    local app_definition
    app_definition=$(cat "$APP_DEFINITION_FILE")
    
    if ! echo "$app_definition" | jq . >/dev/null 2>&1; then
        log_error "Invalid JSON in app definition file: $APP_DEFINITION_FILE"
        exit 1
    fi
    
    # Create app payload
    local app_payload
    app_payload=$(echo "$app_definition" | jq \
        --arg path "$APP_NAME" \
        '. + {path: $path}')
    
    # Deploy app
    local deploy_response
    deploy_response=$(curl -s -X POST \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json" \
        "$WINDMILL_URL/api/w/$WORKSPACE/apps" \
        -d "$app_payload" || echo '{"error":"app_deployment_failed"}')
    
    if echo "$deploy_response" | jq -e '.path' >/dev/null 2>&1; then
        log_success "Application '$APP_NAME' deployed successfully"
        APP_URL="$WINDMILL_URL/apps/$WORKSPACE/$APP_NAME"
        log_info "Application URL: $APP_URL"
    else
        log_error "Failed to deploy application '$APP_NAME'"
        log_verbose "Deploy response: $deploy_response"
        exit 1
    fi
}

# Validate deployment
validate_deployment() {
    if [[ "$VALIDATE" == "true" ]]; then
        log_info "Validating deployment..."
        
        # Check if app is accessible
        local app_response
        app_response=$(curl -s -H "Authorization: Bearer $AUTH_TOKEN" \
            "$WINDMILL_URL/api/w/$WORKSPACE/apps/$APP_NAME" || echo '{"error":"validation_failed"}')
        
        if echo "$app_response" | jq -e '.path' >/dev/null 2>&1; then
            log_success "Application validation passed"
        else
            log_error "Application validation failed"
            log_verbose "Validation response: $app_response"
            exit 1
        fi
        
        # Check if scripts are accessible
        local scripts_valid=true
        local script_names=("whisper-transcribe" "ollama-analyze" "comfyui-generate" "agent-s2-automation" "process-multimodal-request" "take-screenshot" "clear-session")
        
        for script_name in "${script_names[@]}"; do
            local script_response
            script_response=$(curl -s -H "Authorization: Bearer $AUTH_TOKEN" \
                "$WINDMILL_URL/api/w/$WORKSPACE/scripts/get/$script_name" || echo '{"error":"script_validation_failed"}')
            
            if ! echo "$script_response" | jq -e '.path' >/dev/null 2>&1; then
                log_warning "Script validation failed: $script_name"
                scripts_valid=false
            fi
        done
        
        if [[ "$scripts_valid" == "true" ]]; then
            log_success "All scripts validation passed"
        else
            log_warning "Some scripts failed validation - app may not function correctly"
        fi
    fi
}

# Main deployment process
main() {
    log_info "Starting Windmill UI deployment for Multi-Modal AI Assistant"
    log_info "Target: $WINDMILL_URL (workspace: $WORKSPACE)"
    
    parse_args "$@"
    
    check_prerequisites
    authenticate
    ensure_workspace
    cleanup_existing_app
    deploy_scripts
    deploy_app
    validate_deployment
    
    log_success "🎉 Multi-Modal AI Assistant UI deployed successfully!"
    log_info "📱 Application URL: $APP_URL"
    log_info "🔧 Workspace: $WORKSPACE"
    log_info "⚙️ Scripts deployed: 7"
    
    if [[ "$VALIDATE" == "true" ]]; then
        log_info "✅ Deployment validation completed"
    fi
    
    echo ""
    echo "Next steps:"
    echo "1. Open the application URL in your browser"
    echo "2. Test the multi-modal workflow"
    echo "3. Verify AI service integrations"
    echo ""
}

# Export APP_URL for use by the test scenario
export_app_url() {
    echo "$APP_URL"
}

# Export function for test scenario integration
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Script is being run directly
    main "$@"
else
    # Script is being sourced - export functions
    export -f main
    export -f export_app_url
fi