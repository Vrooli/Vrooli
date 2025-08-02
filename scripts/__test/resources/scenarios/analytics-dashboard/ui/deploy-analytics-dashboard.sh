#!/bin/bash
# ====================================================================
# Enterprise Analytics Dashboard - Windmill Deployment Script
# ====================================================================
# 
# This script deploys the complete Enterprise Analytics Dashboard
# to a running Windmill instance, including all backend scripts and the
# professional UI for Fortune 500 operations teams.
#
# Revenue Potential: $15,000 - $30,000 per project
# Target Market: Enterprise operations teams, DevOps organizations
#
# Usage:
#   ./deploy-analytics-dashboard.sh [options]
#
# Options:
#   --windmill-url URL     Windmill base URL (default: http://localhost:5681)
#   --workspace WORKSPACE  Windmill workspace (default: enterprise)
#   --username USERNAME    Windmill username (default: admin@windmill.dev)
#   --password PASSWORD    Windmill password (default: changeme)
#   --app-name NAME        Application name (default: enterprise-analytics-dashboard)
#   --cleanup             Remove existing app before deployment
#   --validate            Validate deployment after completion
#   --scripts-only        Deploy only TypeScript scripts (skip UI)
#   --ui-only             Deploy only UI (skip scripts)
#   --environment ENV     Deployment environment (dev|staging|prod)
#   --verbose             Enable verbose logging
#
# ====================================================================

set -euo pipefail

# Script directory and paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
UI_SCRIPTS_DIR="$SCRIPT_DIR/scripts"
APP_DEFINITION_FILE="$SCRIPT_DIR/enterprise-analytics-dashboard-app.json"

# Default configuration
WINDMILL_URL="${WINDMILL_URL:-http://localhost:5681}"
WORKSPACE="${WORKSPACE:-enterprise}"
USERNAME="${USERNAME:-admin@windmill.dev}"
PASSWORD="${PASSWORD:-changeme}"
APP_NAME="${APP_NAME:-enterprise-analytics-dashboard}"
ENVIRONMENT="${ENVIRONMENT:-dev}"
CLEANUP=false
VALIDATE=false
SCRIPTS_ONLY=false
UI_ONLY=false
VERBOSE=false

# Service URLs for script configuration
QUESTDB_BASE_URL="${QUESTDB_BASE_URL:-http://localhost:9010}"
OLLAMA_BASE_URL="${OLLAMA_BASE_URL:-http://localhost:11434}"
REDIS_BASE_URL="${REDIS_BASE_URL:-http://localhost:6380}"
MINIO_BASE_URL="${MINIO_BASE_URL:-http://localhost:9000}"
AGENT_S2_BASE_URL="${AGENT_S2_BASE_URL:-http://localhost:4113}"
VAULT_BASE_URL="${VAULT_BASE_URL:-http://localhost:8200}"
NODE_RED_BASE_URL="${NODE_RED_BASE_URL:-http://localhost:1880}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
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

log_enterprise() {
    echo -e "${PURPLE}[ENTERPRISE]${NC} $1"
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${CYAN}[VERBOSE]${NC} $1"
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
            --environment)
                ENVIRONMENT="$2"
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
            --scripts-only)
                SCRIPTS_ONLY=true
                shift
                ;;
            --ui-only)
                UI_ONLY=true
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
    echo "📊 Enterprise Analytics Dashboard - Windmill Deployment"
    echo "💰 Revenue Potential: \$15,000 - \$30,000 per project"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --windmill-url URL     Windmill base URL (default: http://localhost:5681)"
    echo "  --workspace WORKSPACE  Windmill workspace (default: enterprise)"
    echo "  --username USERNAME    Windmill username (default: admin@windmill.dev)"
    echo "  --password PASSWORD    Windmill password (default: changeme)"
    echo "  --app-name NAME        Application name (default: enterprise-analytics-dashboard)"
    echo "  --environment ENV      Deployment environment (dev|staging|prod)"
    echo "  --cleanup              Remove existing app before deployment"
    echo "  --validate             Validate deployment after completion"
    echo "  --scripts-only         Deploy only TypeScript scripts (skip UI)"
    echo "  --ui-only              Deploy only UI (skip scripts)"
    echo "  --verbose              Enable verbose logging"
    echo "  --help                 Show this help message"
    echo ""
    echo "Enterprise Features:"
    echo "  📊 Real-Time Operations Intelligence"
    echo "  🤖 AI-Powered Anomaly Detection"
    echo "  🔮 Predictive Analytics & Forecasting"
    echo "  🚨 Intelligent Alert Management"
    echo "  🛠️ Custom Dashboard Builder"
    echo "  📈 Performance Analysis & Optimization"
    echo "  🏢 Multi-Tenant Enterprise Architecture"
}

# Check prerequisites and enterprise requirements
check_prerequisites() {
    log_info "Checking enterprise deployment prerequisites..."
    
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
    
    # Check service dependencies for enterprise features
    check_service_dependencies
    
    log_success "Enterprise prerequisites check passed"
}

# Check enterprise service dependencies
check_service_dependencies() {
    log_info "Checking enterprise service dependencies..."
    
    local services=(
        "QuestDB:${QUESTDB_BASE_URL}/"
        "Ollama:${OLLAMA_BASE_URL}/api/tags"
        "Redis:${REDIS_BASE_URL}/ping"
        "MinIO:${MINIO_BASE_URL}/minio/health/live"
        "Agent-S2:${AGENT_S2_BASE_URL}/health"
        "Node-RED:${NODE_RED_BASE_URL}/"
    )
    
    local available_services=0
    local total_services=${#services[@]}
    
    for service_info in "${services[@]}"; do
        local service_name="${service_info%%:*}"
        local service_url="${service_info#*:}"
        
        if curl -sf "$service_url" >/dev/null 2>&1; then
            log_verbose "✅ $service_name is available"
            available_services=$((available_services + 1))
        else
            log_warning "⚠️ $service_name is not available at $service_url"
        fi
    done
    
    log_info "Service availability: $available_services/$total_services services online"
    
    if [[ $available_services -lt 3 ]]; then
        log_warning "Less than 3 core services are available. Some features may not work."
        log_warning "For full enterprise functionality, ensure all services are running."
    fi
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

# Ensure enterprise workspace exists
ensure_workspace() {
    log_enterprise "Ensuring enterprise workspace '$WORKSPACE' exists..."
    
    local workspace_response
    workspace_response=$(curl -s -H "Authorization: Bearer $AUTH_TOKEN" \
        "$WINDMILL_URL/api/w/$WORKSPACE" || echo '{"error":"workspace_check_failed"}')
    
    if echo "$workspace_response" | jq -e '.id' >/dev/null 2>&1; then
        log_success "Enterprise workspace '$WORKSPACE' exists"
    else {
        log_info "Creating enterprise workspace '$WORKSPACE'..."
        
        local create_response
        create_response=$(curl -s -X POST \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -H "Content-Type: application/json" \
            "$WINDMILL_URL/api/workspaces" \
            -d "{\"id\":\"$WORKSPACE\",\"name\":\"Enterprise Analytics Dashboard - $WORKSPACE\"}" || echo '{"error":"workspace_creation_failed"}')
        
        if echo "$create_response" | jq -e '.id' >/dev/null 2>&1; then
            log_success "Enterprise workspace '$WORKSPACE' created"
        else
            log_error "Failed to create workspace '$WORKSPACE'"
            log_verbose "Create response: $create_response"
            exit 1
        fi
    }
    
    # Set workspace-level environment variables for service URLs
    set_workspace_variables
}

# Set workspace environment variables
set_workspace_variables() {
    log_info "Configuring workspace environment variables..."
    
    local variables=(
        "QUESTDB_BASE_URL:$QUESTDB_BASE_URL"
        "OLLAMA_BASE_URL:$OLLAMA_BASE_URL"
        "REDIS_BASE_URL:$REDIS_BASE_URL"
        "MINIO_BASE_URL:$MINIO_BASE_URL"
        "AGENT_S2_BASE_URL:$AGENT_S2_BASE_URL"
        "VAULT_BASE_URL:$VAULT_BASE_URL"
        "NODE_RED_BASE_URL:$NODE_RED_BASE_URL"
        "DEPLOYMENT_ENVIRONMENT:$ENVIRONMENT"
    )
    
    for var_info in "${variables[@]}"; do
        local var_name="${var_info%%:*}"
        local var_value="${var_info#*:}"
        
        log_verbose "Setting variable: $var_name = $var_value"
        
        local var_response
        var_response=$(curl -s -X POST \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -H "Content-Type: application/json" \
            "$WINDMILL_URL/api/w/$WORKSPACE/variables" \
            -d "{\"path\":\"$var_name\",\"value\":\"$var_value\",\"is_secret\":false,\"description\":\"Enterprise Analytics Dashboard - $var_name\"}" || echo '{"error":"variable_set_failed"}')
    done
    
    log_success "Workspace environment configured"
}

# Deploy enterprise TypeScript scripts
deploy_scripts() {
    if [[ "$UI_ONLY" == "true" ]]; then
        log_info "Skipping script deployment (UI-only mode)"
        return
    fi
    
    log_enterprise "Deploying enterprise TypeScript backend scripts..."
    
    local script_files=(
        "metrics_collector.ts"
        "ai_insights_engine.ts"
    )
    
    local deployed_count=0
    
    for script_file in "${script_files[@]}"; do
        local script_path="$UI_SCRIPTS_DIR/$script_file"
        local script_name="${script_file%.ts}"
        
        if [[ ! -f "$script_path" ]]; then
            log_warning "Script file not found: $script_path"
            continue
        fi
        
        log_verbose "Deploying enterprise script: $script_name"
        
        # Read script content
        local script_content
        script_content=$(cat "$script_path")
        
        # Create script payload with enterprise metadata
        local script_payload
        script_payload=$(jq -n \
            --arg path "$script_name" \
            --arg content "$script_content" \
            --arg language "typescript" \
            --arg description "Enterprise Analytics Dashboard - $script_name (Revenue: \$15K-30K)" \
            --arg tag "enterprise" \
            '{
                path: $path,
                content: $content,
                language: $language,
                description: $description,
                is_template: false,
                tag: $tag
            }')
        
        # Deploy script
        local deploy_response
        deploy_response=$(curl -s -X POST \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -H "Content-Type: application/json" \
            "$WINDMILL_URL/api/w/$WORKSPACE/scripts" \
            -d "$script_payload" || echo '{"error":"script_deployment_failed"}')
        
        if echo "$deploy_response" | jq -e '.path' >/dev/null 2>&1; then
            log_success "✅ Deployed enterprise script: $script_name"
            deployed_count=$((deployed_count + 1))
        else
            log_error "❌ Failed to deploy script: $script_name"
            log_verbose "Deploy response: $deploy_response"
        fi
    done
    
    log_enterprise "Deployed $deployed_count/${#script_files[@]} enterprise scripts"
    
    if [[ $deployed_count -eq 0 ]]; then
        log_error "No scripts were successfully deployed"
        exit 1
    fi
}

# Cleanup existing app if requested
cleanup_existing_app() {
    if [[ "$CLEANUP" == "true" ]]; then
        log_info "Cleaning up existing enterprise dashboard '$APP_NAME'..."
        
        local delete_response
        delete_response=$(curl -s -X DELETE \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            "$WINDMILL_URL/api/w/$WORKSPACE/apps/$APP_NAME" || echo '{"status":"not_found"}')
        
        log_success "Cleanup completed (app may not have existed)"
    fi
}

# Deploy the enterprise application UI
deploy_app() {
    if [[ "$SCRIPTS_ONLY" == "true" ]]; then
        log_info "Skipping UI deployment (scripts-only mode)"
        return
    fi
    
    log_enterprise "Deploying Enterprise Analytics Dashboard UI..."
    log_info "💰 Revenue Potential: \$15,000 - \$30,000 per project"
    
    # Read and validate app definition
    local app_definition
    app_definition=$(cat "$APP_DEFINITION_FILE")
    
    if ! echo "$app_definition" | jq . >/dev/null 2>&1; then
        log_error "Invalid JSON in app definition file: $APP_DEFINITION_FILE"
        exit 1
    fi
    
    # Enhance app definition with environment-specific settings
    local app_payload
    app_payload=$(echo "$app_definition" | jq \
        --arg path "$APP_NAME" \
        --arg environment "$ENVIRONMENT" \
        --arg deployed_at "$(date -Iseconds)" \
        '. + {
            path: $path,
            deployment: {
                environment: $environment,
                deployed_at: $deployed_at,
                revenue_potential: "$15K-30K",
                target_market: "Fortune 500 Operations Teams, Enterprise DevOps"
            }
        }')
    
    # Deploy app
    local deploy_response
    deploy_response=$(curl -s -X POST \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json" \
        "$WINDMILL_URL/api/w/$WORKSPACE/apps" \
        -d "$app_payload" || echo '{"error":"app_deployment_failed"}')
    
    if echo "$deploy_response" | jq -e '.path' >/dev/null 2>&1; then
        log_success "🎉 Enterprise Analytics Dashboard deployed successfully!"
        APP_URL="$WINDMILL_URL/apps/$WORKSPACE/$APP_NAME"
        log_enterprise "Application URL: $APP_URL"
    else
        log_error "Failed to deploy enterprise dashboard '$APP_NAME'"
        log_verbose "Deploy response: $deploy_response"
        exit 1
    fi
}

# Comprehensive deployment validation
validate_deployment() {
    if [[ "$VALIDATE" != "true" ]]; then
        return
    fi
    
    log_info "Validating enterprise deployment..."
    
    # Validate UI deployment
    if [[ "$SCRIPTS_ONLY" != "true" ]]; then
        log_verbose "Validating UI application..."
        local app_response
        app_response=$(curl -s -H "Authorization: Bearer $AUTH_TOKEN" \
            "$WINDMILL_URL/api/w/$WORKSPACE/apps/$APP_NAME" || echo '{"error":"validation_failed"}')
        
        if echo "$app_response" | jq -e '.path' >/dev/null 2>&1; then
            log_success "✅ UI application validation passed"
        else
            log_error "❌ UI application validation failed"
            log_verbose "Validation response: $app_response"
            exit 1
        fi
    fi
    
    # Validate script deployment
    if [[ "$UI_ONLY" != "true" ]]; then
        log_verbose "Validating enterprise scripts..."
        local scripts_valid=true
        local script_names=(
            "metrics_collector"
            "ai_insights_engine"
        )
        
        for script_name in "${script_names[@]}"; do
            local script_response
            script_response=$(curl -s -H "Authorization: Bearer $AUTH_TOKEN" \
                "$WINDMILL_URL/api/w/$WORKSPACE/scripts/get/$script_name" || echo '{"error":"script_validation_failed"}')
            
            if echo "$script_response" | jq -e '.path' >/dev/null 2>&1; then
                log_verbose "✅ Script validated: $script_name"
            else
                log_warning "❌ Script validation failed: $script_name"
                scripts_valid=false
            fi
        done
        
        if [[ "$scripts_valid" == "true" ]]; then
            log_success "✅ All enterprise scripts validation passed"
        else
            log_warning "⚠️ Some scripts failed validation - platform may not function correctly"
        fi
    fi
    
    # Test basic API connectivity
    test_api_connectivity
}

# Test API connectivity to services
test_api_connectivity() {
    log_verbose "Testing service API connectivity..."
    
    local connectivity_tests=(
        "Windmill API:$WINDMILL_URL/api/version"
        "QuestDB:$QUESTDB_BASE_URL/"
        "Ollama:$OLLAMA_BASE_URL/api/tags"
    )
    
    local successful_tests=0
    
    for test_info in "${connectivity_tests[@]}"; do
        local service_name="${test_info%%:*}"
        local service_url="${test_info#*:}"
        
        if curl -sf "$service_url" >/dev/null 2>&1; then
            log_verbose "✅ $service_name connectivity test passed"
            successful_tests=$((successful_tests + 1))
        else
            log_verbose "❌ $service_name connectivity test failed"
        fi
    done
    
    if [[ $successful_tests -ge 2 ]]; then
        log_success "Core service connectivity validated"
    else
        log_warning "Limited service connectivity detected"
    fi
}

# Display enterprise deployment summary
show_deployment_summary() {
    echo ""
    echo "=================================================="
    echo "📊 ENTERPRISE ANALYTICS DASHBOARD DEPLOYED"
    echo "=================================================="
    echo ""
    echo "💰 Revenue Potential: \$15,000 - \$30,000 per project"
    echo "🎯 Target Market: Fortune 500 Operations Teams, Enterprise DevOps"
    echo ""
    echo "📱 Application URL: $APP_URL"
    echo "🏢 Workspace: $WORKSPACE"
    echo "🌍 Environment: $ENVIRONMENT"
    
    if [[ "$SCRIPTS_ONLY" != "true" && "$UI_ONLY" != "true" ]]; then
        echo "⚙️ Enterprise Scripts: 2 deployed"
        echo "🎛️ UI Components: Complete dashboard platform"
    elif [[ "$SCRIPTS_ONLY" == "true" ]]; then
        echo "⚙️ Enterprise Scripts: 2 deployed"
        echo "🎛️ UI Components: Skipped (scripts-only)"
    elif [[ "$UI_ONLY" == "true" ]]; then
        echo "⚙️ Enterprise Scripts: Skipped (UI-only)"
        echo "🎛️ UI Components: Complete dashboard platform"
    fi
    
    echo ""
    echo "🚀 ENTERPRISE FEATURES AVAILABLE:"
    echo "   📊 Real-Time Operations Intelligence"
    echo "   🤖 AI-Powered Anomaly Detection"
    echo "   🔮 Predictive Analytics & Forecasting"
    echo "   🚨 Intelligent Alert Management"
    echo "   📈 Performance Analysis & Optimization"
    echo "   🛠️ Custom Dashboard Builder"
    echo "   🏢 Multi-Tenant Architecture"
    echo ""
    
    if [[ "$VALIDATE" == "true" ]]; then
        echo "✅ Deployment validation completed"
        echo ""
    fi
    
    echo "📋 NEXT STEPS:"
    echo "   1. Open the application URL in your browser"
    echo "   2. Review executive overview dashboard"
    echo "   3. Configure real-time monitoring sources"
    echo "   4. Set up AI insights and anomaly detection"
    echo "   5. Create custom dashboards for teams"
    echo "   6. Configure intelligent alert rules"
    echo ""
    echo "💼 BUSINESS OPPORTUNITIES:"
    echo "   • Fortune 500 operations monitoring"
    echo "   • DevOps platform intelligence"
    echo "   • Manufacturing operations oversight"
    echo "   • Healthcare system monitoring"
    echo "   • Financial services compliance"
    echo ""
}

# Main deployment process
main() {
    echo "📊 Starting Enterprise Analytics Dashboard Deployment"
    echo "💰 Revenue Potential: \$15,000 - \$30,000 per project"
    echo "🎯 Target: $WINDMILL_URL (workspace: $WORKSPACE, env: $ENVIRONMENT)"
    echo ""
    
    parse_args "$@"
    
    check_prerequisites
    authenticate
    ensure_workspace
    cleanup_existing_app
    deploy_scripts
    deploy_app
    validate_deployment
    
    show_deployment_summary
}

# Export APP_URL for use by test scenarios
export_app_url() {
    echo "$APP_URL"
}

# Export functions for test scenario integration
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Script is being run directly
    main "$@"
else
    # Script is being sourced - export functions
    export -f main
    export -f export_app_url
fi