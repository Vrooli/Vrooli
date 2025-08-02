#!/bin/bash
# ====================================================================
# Automation Resource Health Checks
# ====================================================================
#
# Category-specific health checks for automation resources including
# workflow engine status, UI responsiveness, and trigger reliability.
#
# Supported Automation Resources:
# - n8n: Visual workflow automation
# - Node-RED: Real-time flow programming
# - Windmill: Code-first workflows
# - Huginn: Agent-based event processing  
# - ComfyUI: AI image generation workflows
#
# ====================================================================

# Automation resource health check implementations
check_n8n_health() {
    local port="${1:-5678}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        echo "unreachable"
        return 1
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local ui_responsive="false"
    local api_available="false"
    local workflows_count="unknown"
    
    # Check if main UI loads
    local ui_response
    ui_response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ -n "$ui_response" ]] && echo "$ui_response" | grep -q "n8n"; then
        ui_responsive="true"
    fi
    
    # Check API endpoints
    if curl -s --max-time 5 "http://localhost:${port}/rest/workflows" >/dev/null 2>&1; then
        api_available="true"
        
        # Try to get workflow count (may require authentication)
        local workflows_response
        workflows_response=$(curl -s --max-time 10 "http://localhost:${port}/rest/workflows" 2>/dev/null)
        if [[ -n "$workflows_response" ]] && echo "$workflows_response" | jq . >/dev/null 2>&1; then
            workflows_count=$(echo "$workflows_response" | jq 'length' 2>/dev/null || echo "auth_required")
        fi
    fi
    
    if [[ "$ui_responsive" == "true" && "$api_available" == "true" ]]; then
        echo "healthy:ui_responsive:api_available:workflows:$workflows_count"
    elif [[ "$ui_responsive" == "true" ]]; then
        echo "degraded:ui_responsive:api_unavailable"
    else
        echo "degraded:ui_unresponsive"
    fi
    
    return 0
}

check_node_red_health() {
    local port="${1:-1880}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        echo "unreachable"
        return 1
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local ui_responsive="false"
    local flows_api="false"
    local flows_count="0"
    
    # Check if main UI loads
    local ui_response
    ui_response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ -n "$ui_response" ]] && echo "$ui_response" | grep -q "Node-RED"; then
        ui_responsive="true"
    fi
    
    # Check flows API
    if curl -s --max-time 5 "http://localhost:${port}/flows" >/dev/null 2>&1; then
        flows_api="true"
        
        local flows_response
        flows_response=$(curl -s --max-time 10 "http://localhost:${port}/flows" 2>/dev/null)
        if [[ -n "$flows_response" ]] && echo "$flows_response" | jq . >/dev/null 2>&1; then
            flows_count=$(echo "$flows_response" | jq 'length' 2>/dev/null || echo "0")
        fi
    fi
    
    if [[ "$ui_responsive" == "true" && "$flows_api" == "true" ]]; then
        echo "healthy:ui_responsive:flows_api_available:flows:$flows_count"
    elif [[ "$ui_responsive" == "true" ]]; then
        echo "degraded:ui_responsive:flows_api_unavailable"
    else
        echo "degraded:ui_unresponsive"
    fi
    
    return 0
}

check_windmill_health() {
    local port="${1:-5681}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/api/version" >/dev/null 2>&1; then
        # Try alternative endpoints
        if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
            echo "healthy"
            return 0
        else
            echo "unreachable"
            return 1
        fi
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local version_info="unknown"
    local ui_responsive="false"
    local api_available="false"
    
    # Get version information
    local version_response
    version_response=$(curl -s --max-time 10 "http://localhost:${port}/api/version" 2>/dev/null)
    if [[ -n "$version_response" ]]; then
        version_info=$(echo "$version_response" | tr -d '"' || echo "unknown")
        api_available="true"
    fi
    
    # Check if main UI loads
    local ui_response
    ui_response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ -n "$ui_response" ]] && echo "$ui_response" | grep -q "Windmill"; then
        ui_responsive="true"
    fi
    
    if [[ "$ui_responsive" == "true" && "$api_available" == "true" ]]; then
        echo "healthy:ui_responsive:api_available:version:$version_info"
    elif [[ "$api_available" == "true" ]]; then
        echo "degraded:api_available:ui_check_failed:version:$version_info"
    else
        echo "degraded:api_unavailable"
    fi
    
    return 0
}

check_huginn_health() {
    local port="${1:-4111}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        echo "unreachable"
        return 1
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local ui_responsive="false"
    local agents_accessible="false"
    
    # Check if main UI loads  
    local ui_response
    ui_response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ -n "$ui_response" ]] && echo "$ui_response" | grep -q "Huginn"; then
        ui_responsive="true"
    fi
    
    # Check agents endpoint (may require authentication)
    if curl -s --max-time 5 "http://localhost:${port}/agents" >/dev/null 2>&1; then
        agents_accessible="true"
    fi
    
    if [[ "$ui_responsive" == "true" && "$agents_accessible" == "true" ]]; then
        echo "healthy:ui_responsive:agents_accessible"
    elif [[ "$ui_responsive" == "true" ]]; then
        echo "degraded:ui_responsive:agents_require_auth"
    else
        echo "degraded:ui_unresponsive"
    fi
    
    return 0
}

check_comfyui_health() {
    local port="${1:-8188}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        echo "unreachable"
        return 1
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local ui_responsive="false"
    local api_available="false"
    local queue_available="false"
    
    # Check if main UI loads
    local ui_response
    ui_response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ -n "$ui_response" ]] && echo "$ui_response" | grep -q "ComfyUI"; then
        ui_responsive="true"
    fi
    
    # Check API endpoints
    if curl -s --max-time 5 "http://localhost:${port}/api/prompt" >/dev/null 2>&1; then
        api_available="true"
    fi
    
    # Check queue endpoint
    if curl -s --max-time 5 "http://localhost:${port}/api/queue" >/dev/null 2>&1; then
        queue_available="true"
    fi
    
    if [[ "$ui_responsive" == "true" && "$api_available" == "true" && "$queue_available" == "true" ]]; then
        echo "healthy:ui_responsive:api_available:queue_available"
    elif [[ "$ui_responsive" == "true" && "$api_available" == "true" ]]; then
        echo "degraded:ui_responsive:api_available:queue_unavailable"
    elif [[ "$ui_responsive" == "true" ]]; then
        echo "degraded:ui_responsive:api_unavailable"
    else
        echo "degraded:ui_unresponsive"
    fi
    
    return 0
}

# Generic automation health check dispatcher
check_automation_resource_health() {
    local resource_name="$1"
    local port="$2"
    local health_level="${3:-basic}"
    
    case "$resource_name" in
        "n8n")
            check_n8n_health "$port" "$health_level"
            ;;
        "node-red")
            check_node_red_health "$port" "$health_level"
            ;;
        "windmill")
            check_windmill_health "$port" "$health_level"
            ;;
        "huginn")
            check_huginn_health "$port" "$health_level"
            ;;
        "comfyui")
            check_comfyui_health "$port" "$health_level"
            ;;
        *)
            # Fallback to generic HTTP health check
            if curl -s --max-time 5 "http://localhost:${port}/health" >/dev/null 2>&1; then
                echo "healthy"
            elif curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
                echo "healthy"
            else
                echo "unreachable"
            fi
            ;;
    esac
}

# Automation resource capability testing
test_automation_resource_capabilities() {
    local resource_name="$1"
    local port="$2"
    
    case "$resource_name" in
        "n8n")
            test_n8n_capabilities "$port"
            ;;
        "node-red")
            test_node_red_capabilities "$port"
            ;;
        "windmill")
            test_windmill_capabilities "$port"
            ;;
        "huginn")
            test_huginn_capabilities "$port"
            ;;
        "comfyui")
            test_comfyui_capabilities "$port"
            ;;
        *)
            echo "capability_testing_not_implemented"
            ;;
    esac
}

test_n8n_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        capabilities+=("web_interface")
        capabilities+=("visual_workflows")
    fi
    
    if curl -s --max-time 5 "http://localhost:${port}/rest/workflows" >/dev/null 2>&1; then
        capabilities+=("rest_api")
        capabilities+=("workflow_management")
    fi
    
    if curl -s --max-time 5 "http://localhost:${port}/webhook/" >/dev/null 2>&1; then
        capabilities+=("webhook_triggers")
    fi
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_node_red_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        capabilities+=("web_interface")
        capabilities+=("flow_programming")
    fi
    
    if curl -s --max-time 5 "http://localhost:${port}/flows" >/dev/null 2>&1; then
        capabilities+=("flows_api")
        capabilities+=("real_time_processing")
    fi
    
    capabilities+=("mqtt_support")
    capabilities+=("iot_integration")
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_windmill_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    if curl -s --max-time 5 "http://localhost:${port}/api/version" >/dev/null 2>&1; then
        capabilities+=("rest_api")
        capabilities+=("code_workflows")
    fi
    
    if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        capabilities+=("web_interface")
        capabilities+=("script_execution")
    fi
    
    capabilities+=("typescript_support")
    capabilities+=("python_support")
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_huginn_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        capabilities+=("web_interface")
        capabilities+=("agent_based_automation")
    fi
    
    capabilities+=("web_scraping")
    capabilities+=("event_monitoring")
    capabilities+=("data_aggregation")
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

test_comfyui_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        capabilities+=("web_interface")
        capabilities+=("workflow_ui")
    fi
    
    if curl -s --max-time 5 "http://localhost:${port}/api/prompt" >/dev/null 2>&1; then
        capabilities+=("generation_api")
        capabilities+=("image_processing")
    fi
    
    if curl -s --max-time 5 "http://localhost:${port}/api/queue" >/dev/null 2>&1; then
        capabilities+=("job_queue")
        capabilities+=("batch_processing")
    fi
    
    capabilities+=("stable_diffusion")
    capabilities+=("ai_image_generation")
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

# Export functions
export -f check_automation_resource_health
export -f test_automation_resource_capabilities
export -f check_n8n_health
export -f check_node_red_health
export -f check_windmill_health
export -f check_huginn_health
export -f check_comfyui_health