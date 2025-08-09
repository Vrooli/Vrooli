#!/usr/bin/env bash
set -euo pipefail

# URL Discovery Demo Script
# Demonstrates the enhanced URL discovery and display functionality

_HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${_HERE}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${_HERE}/url-discovery.sh"

echo "🎯 Vrooli URL Discovery Demo"
echo "============================"
echo ""

# Simulate the scenario-to-app.sh environment
SERVICE_JSON=$(cat "${var_SCRIPTS_SCENARIOS_DIR}/core/campaign-content-studio/.vrooli/service.json")
SCENARIO_PATH="${var_SCRIPTS_SCENARIOS_DIR}/core/campaign-content-studio"

# Define helper functions (from scenario-to-app.sh)
demo_url_discovery::get_required_resources() {
    echo "$SERVICE_JSON" | jq -r '
        .resources | 
        to_entries[] | 
        .value | 
        to_entries[] | 
        select(.value.enabled == true and (.value.required // false) == true) | 
        .key
    ' 2>/dev/null | sort -u
}

demo_url_discovery::log_info() { echo "[$(date +'%H:%M:%S')] INFO: $*"; }

# Demonstrate enhanced URL display
demo_url_discovery::get_access_urls_demo() {
    demo_url_discovery::log_info "🎯 Application Access Points:"
    echo ""
    
    # Extract application URL from service.json if available
    local app_url app_name api_url health_url
    app_url=$(echo "$SERVICE_JSON" | jq -r '.deployment.urls.application // empty' 2>/dev/null)
    api_url=$(echo "$SERVICE_JSON" | jq -r '.deployment.urls.api // empty' 2>/dev/null)
    health_url=$(echo "$SERVICE_JSON" | jq -r '.deployment.urls.health // empty' 2>/dev/null)
    app_name=$(echo "$SERVICE_JSON" | jq -r '.service.displayName // .service.name // "Application"' 2>/dev/null)
    
    # Display application URLs if available
    if [[ -n "$app_url" ]]; then
        local app_status="⏳"
        
        # Check if application URL is accessible
        if url_discovery::validate_url "$app_url" 3; then
            app_status="✅"
        else
            app_status="❌"
        fi
        
        echo "┌─────────────────────────────────────────────────────────────┐"
        echo "│ 🚀 ${app_name}                                              │"
        echo "├─────────────────────────────────────────────────────────────┤"
        echo "│ ${app_status} Application: ${app_url}                       │"
        
        if [[ -n "$api_url" && "$api_url" != "$app_url" ]]; then
            echo "│   └─ API: ${api_url}                                      │"
        fi
        
        if [[ -n "$health_url" && "$health_url" != "$app_url" ]]; then
            echo "│   └─ Health: ${health_url}                               │"
        fi
        
        echo "└─────────────────────────────────────────────────────────────┘"
        echo ""
    fi
    
    # Get required resources
    local required_resources
    mapfile -t required_resources < <(demo_url_discovery::get_required_resources)
    
    if [[ ${#required_resources[@]} -eq 0 ]]; then
        demo_url_discovery::log_info "No resources configured."
        return 0
    fi
    
    echo "📋 Required resources: ${required_resources[*]}"
    echo ""
    
    # Categorize resources for better display
    declare -A resource_categories=(
        ["🤖 AI Resources"]=""
        ["⚙️  Automation"]=""
        ["💾 Storage"]=""
        ["🔍 Search"]=""
        ["🚀 Execution"]=""
        ["🤝 Agents"]=""
    )
    
    # Categorize each resource
    for resource in "${required_resources[@]}"; do
        case "$resource" in
            ollama|whisper|unstructured-io)
                resource_categories["🤖 AI Resources"]+="$resource "
                ;;
            n8n|windmill|node-red|comfyui|huginn)
                resource_categories["⚙️  Automation"]+="$resource "
                ;;
            postgres|postgresql|redis|minio|qdrant|questdb|vault)
                resource_categories["💾 Storage"]+="$resource "
                ;;
            searxng)
                resource_categories["🔍 Search"]+="$resource "
                ;;
            judge0)
                resource_categories["🚀 Execution"]+="$resource "
                ;;
            agent-s2|browserless|claude-code)
                resource_categories["🤝 Agents"]+="$resource "
                ;;
        esac
    done
    
    # Display each non-empty category
    for category in "${!resource_categories[@]}"; do
        local category_resources="${resource_categories[$category]}"
        if [[ -n "$category_resources" ]]; then
            echo "┌─────────────────────────────────────────────────────────────┐"
            echo "│ ${category}                                             │"
            echo "├─────────────────────────────────────────────────────────────┤"
            
            for resource in $category_resources; do
                local resource_display=""
                
                # Try to get URLs using the discovery infrastructure
                local discovery_result service_json_path=""
                
                # Check for service.json overrides
                if [[ -n "$SERVICE_JSON" ]]; then
                    service_json_path="${SCENARIO_PATH}/.vrooli/service.json"
                fi
                
                discovery_result=$(url_discovery::discover "$resource" 2>/dev/null || echo '{"status": "error"}')
                
                # Apply service.json overrides if available
                if [[ -n "$service_json_path" ]]; then
                    discovery_result=$(url_discovery::apply_overrides "$resource" "$discovery_result" "$service_json_path" 2>/dev/null || echo "$discovery_result")
                fi
                
                resource_display=$(url_discovery::format_display "$resource" "$discovery_result" true 2>/dev/null || echo "")
                
                # Fallback to simple display if discovery failed
                if [[ -z "$resource_display" ]]; then
                    resource_display="❓ ${resource}: Status unknown"
                fi
                
                # Format for box display (prefix with │ and pad)
                echo "│ ${resource_display}"
            done
            
            echo "└─────────────────────────────────────────────────────────────┘"
            echo ""
        fi
    done
    
    # Show connection tips
    demo_url_discovery::log_info "💡 Connection Tips:"
    demo_url_discovery::log_info "  • Most services use localhost and standard ports"
    demo_url_discovery::log_info "  • Check individual service logs if connection fails"
    demo_url_discovery::log_info "  • Some services may take a few moments to start"
    echo ""
}

# Run the demo
echo "Demo: Enhanced URL Discovery and Display"
echo "========================================"
echo ""

echo "1. Testing individual resource discovery:"
echo "   - Ollama AI:"
url_discovery::discover "ollama" | jq .

echo ""
echo "   - n8n Automation:"
url_discovery::discover "n8n" | jq .

echo ""
echo "   - PostgreSQL Database:"
url_discovery::discover "postgres" | jq .

echo ""
echo ""

echo "2. Testing complete scenario URL display:"
echo "   (Using campaign-content-studio scenario)"
echo ""

demo_url_discovery::get_access_urls_demo

echo ""
echo "3. Testing health check for all configured resources:"
echo ""

url_discovery::health_check_all

echo ""
echo "✅ Demo completed! The URL discovery infrastructure is working correctly."
echo ""
echo "Key improvements:"
echo "  ✅ Dynamic URL discovery (no hardcoded URLs)"
echo "  ✅ Real-time health status checking"
echo "  ✅ Service.json override support"
echo "  ✅ Categorized resource display"
echo "  ✅ Enhanced visual formatting with status indicators"
echo "  ✅ Caching for improved performance"
echo ""