#!/bin/bash
# ====================================================================
# Resource Discovery and Health Validation
# ====================================================================
#
# Discovers available resources, checks configuration, and validates
# health status to determine which resources can be tested.
#
# Functions:
#   - discover_all_resources()     - Find all available resources
#   - filter_enabled_resources()   - Check configuration for enabled resources
#   - validate_resource_health()   - Test resource health and connectivity
#   - check_resource_health()      - Check individual resource health
#   - get_resource_info()          - Get detailed resource information
#
# ====================================================================

# Discover all available resources from the resource scripts
discover_all_resources() {
    log_info "Discovering available resources..."
    
    local discovery_output
    discovery_output=$("$RESOURCES_DIR/index.sh" --action list 2>&1) || {
        log_error "Failed to discover resources"
        exit 1
    }
    
    # Parse the discovery output to extract resource names and status
    while IFS= read -r line; do
        if [[ "$line" =~ ^[[:space:]]*-[[:space:]]+([^:]+):[[:space:]]*(.+)$ ]]; then
            local resource_name="${BASH_REMATCH[1]}"
            local resource_status="${BASH_REMATCH[2]}"
            
            ALL_RESOURCES+=("$resource_name")
            
            log_debug "Found resource: $resource_name ($resource_status)"
        fi
    done <<< "$discovery_output"
    
    log_success "Discovered ${#ALL_RESOURCES[@]} total resources: ${ALL_RESOURCES[*]}"
}

# Filter resources based on configuration
filter_enabled_resources() {
    log_info "Checking enabled resources in configuration..."
    
    local config_file="$HOME/.vrooli/resources.local.json"
    
    if [[ ! -f "$config_file" ]]; then
        log_warning "Configuration file not found: $config_file"
        log_info "Treating all discovered resources as enabled"
        ENABLED_RESOURCES=("${ALL_RESOURCES[@]}")
        return
    fi
    
    # Check each resource in configuration
    for resource in "${ALL_RESOURCES[@]}"; do
        if [[ -n "$SPECIFIC_RESOURCE" && "$resource" != "$SPECIFIC_RESOURCE" ]]; then
            DISABLED_RESOURCES+=("$resource")
            log_debug "Skipping $resource - not the specified resource"
            continue
        fi
        
        # Check if resource is enabled in config
        local enabled
        enabled=$(jq -r --arg resource "$resource" '
            .services | to_entries[] | 
            select(.value | objects | has($resource)) | 
            .value[$resource].enabled // false
        ' "$config_file" 2>/dev/null)
        
        if [[ "$enabled" == "true" ]]; then
            ENABLED_RESOURCES+=("$resource")
            log_debug "✅ $resource is enabled in configuration"
        else
            # Check if resource is running anyway (might be enabled differently)
            local is_running
            is_running=$(check_resource_running "$resource")
            
            if [[ "$is_running" == "true" ]]; then
                ENABLED_RESOURCES+=("$resource")
                log_warning "⚠️  $resource is running but not explicitly enabled in config"
            else
                DISABLED_RESOURCES+=("$resource")
                log_warning "⚠️  Skipping $resource - not enabled in configuration"
            fi
        fi
    done
    
    if [[ ${#ENABLED_RESOURCES[@]} -eq 0 ]]; then
        log_error "No enabled resources found"
        exit 1
    fi
    
    log_success "Found ${#ENABLED_RESOURCES[@]} enabled resources: ${ENABLED_RESOURCES[*]}"
    
    if [[ ${#DISABLED_RESOURCES[@]} -gt 0 ]]; then
        log_info "Disabled resources (${#DISABLED_RESOURCES[@]}): ${DISABLED_RESOURCES[*]}"
    fi
}

# Validate health of enabled resources
validate_resource_health() {
    log_info "Validating resource health..."
    
    for resource in "${ENABLED_RESOURCES[@]}"; do
        local health_status
        health_status=$(check_resource_health "$resource")
        
        case "$health_status" in
            "healthy")
                HEALTHY_RESOURCES+=("$resource")
                log_success "✅ $resource is healthy and ready for testing"
                ;;
            "starting")
                UNHEALTHY_RESOURCES+=("$resource")
                log_warning "⚠️  $resource is starting - will retry in health check"
                ;;
            "unhealthy"|"timeout"|"unreachable")
                UNHEALTHY_RESOURCES+=("$resource")
                log_error "❌ Skipping $resource - health check failed: $health_status"
                ;;
            *)
                UNHEALTHY_RESOURCES+=("$resource")
                log_error "❌ Skipping $resource - unknown health status: $health_status"
                ;;
        esac
    done
    
    # Retry resources that were starting
    if [[ ${#UNHEALTHY_RESOURCES[@]} -gt 0 ]]; then
        log_info "Retrying health checks for starting resources..."
        sleep 5
        
        local retry_resources=("${UNHEALTHY_RESOURCES[@]}")
        UNHEALTHY_RESOURCES=()
        
        for resource in "${retry_resources[@]}"; do
            local health_status
            health_status=$(check_resource_health "$resource")
            
            if [[ "$health_status" == "healthy" ]]; then
                HEALTHY_RESOURCES+=("$resource")
                log_success "✅ $resource is now healthy after retry"
            else
                UNHEALTHY_RESOURCES+=("$resource")
                log_error "❌ $resource still unhealthy after retry: $health_status"
            fi
        done
    fi
    
    if [[ ${#HEALTHY_RESOURCES[@]} -eq 0 ]]; then
        log_error "No healthy resources available for testing"
        exit 1
    fi
    
    log_success "Found ${#HEALTHY_RESOURCES[@]} healthy resources ready for testing"
    
    if [[ ${#UNHEALTHY_RESOURCES[@]} -gt 0 ]]; then
        log_info "Unhealthy resources (${#UNHEALTHY_RESOURCES[@]}): ${UNHEALTHY_RESOURCES[*]}"
    fi
}

# Check if a resource is currently running (basic check)
check_resource_running() {
    local resource="$1"
    
    # Use docker ps to check if resource container is running
    if docker ps --format "{{.Names}}" | grep -q "^${resource}$" 2>/dev/null; then
        echo "true"
    else
        echo "false"
    fi
}

# Check detailed health status of a specific resource
check_resource_health() {
    local resource="$1"
    
    # Redirect debug output to stderr so it doesn't interfere with return value
    log_debug "Checking health of $resource..." >&2
    
    # Special case handling for resources without standard port mappings
    case "$resource" in
        "agent-s2")
            check_agent_s2_health
            return
            ;;
        "windmill")
            # Windmill has complex deployment, check if main service exists
            if docker ps --format "{{.Names}}" | grep -q "windmill.*server" 2>/dev/null; then
                echo "healthy"
            else
                echo "unreachable"
            fi
            return
            ;;
    esac
    
    # Get resource info from discovery
    local resource_info
    resource_info=$(get_resource_info "$resource")
    
    if [[ -z "$resource_info" ]]; then
        echo "unknown"
        return
    fi
    
    # Extract port and health endpoint information
    local port
    port=$(echo "$resource_info" | jq -r '.port // empty' 2>/dev/null)
    
    if [[ -z "$port" ]]; then
        # Try to get port from docker if not in discovery
        port=$(docker port "$resource" 2>/dev/null | head -n1 | cut -d':' -f2)
    fi
    
    if [[ -z "$port" ]]; then
        log_debug "No port information found for $resource" >&2
        echo "unreachable"
        return
    fi
    
    # Perform health check based on resource type
    case "$resource" in
        "ollama")
            check_ollama_health "$port"
            ;;
        "whisper")
            check_whisper_health "$port"
            ;;
        "n8n")
            check_n8n_health "$port"
            ;;
        "browserless")
            check_browserless_health "$port"
            ;;
        "minio")
            check_minio_health "$port"
            ;;
        "qdrant")
            check_qdrant_health "$port"
            ;;
        "searxng")
            check_searxng_health "$port"
            ;;
        "node-red")
            check_node_red_health "$port"
            ;;
        "huginn")
            check_huginn_health "$port"
            ;;
        "comfyui")
            check_comfyui_health "$port"
            ;;
        "vault")
            check_vault_health "$port"
            ;;
        *)
            # Generic HTTP health check
            check_generic_health "$resource" "$port"
            ;;
    esac
}

# Resource-specific health check functions
check_ollama_health() {
    local port="$1"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/api/tags" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        if echo "$response" | jq -e '.models' >/dev/null 2>&1; then
            echo "healthy"
        else
            echo "unhealthy"
        fi
    else
        echo "unreachable"
    fi
}

check_whisper_health() {
    local port="$1"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/health" 2>/dev/null)
    if [[ $? -eq 0 ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_n8n_health() {
    local port="$1"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/healthz" 2>/dev/null)
    if [[ $? -eq 0 ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_browserless_health() {
    local port="$1"
    local response
    
    # Browserless provides a /pressure endpoint for health checks
    response=$(curl -s --max-time 10 "http://localhost:${port}/pressure" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        # Check if response contains expected fields
        if echo "$response" | jq -e '.pressure' >/dev/null 2>&1; then
            echo "healthy"
        else
            echo "unhealthy"
        fi
    else
        echo "unreachable"
    fi
}

check_agent_s2_health() {
    # Agent-S2 doesn't expose a port, check docker health
    local health_status
    health_status=$(docker inspect agent-s2 --format='{{.State.Health.Status}}' 2>/dev/null)
    
    case "$health_status" in
        "healthy")
            echo "healthy"
            ;;
        "starting")
            echo "starting"
            ;;
        *)
            echo "unhealthy"
            ;;
    esac
}

check_minio_health() {
    local port="$1"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/minio/health/live" 2>/dev/null)
    if [[ $? -eq 0 ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_qdrant_health() {
    local port="$1"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_searxng_health() {
    local port="$1"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/healthz" 2>/dev/null)
    if [[ $? -eq 0 ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_node_red_health() {
    local port="$1"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_huginn_health() {
    local port="$1"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_comfyui_health() {
    local port="$1"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_vault_health() {
    local port="$1"
    local response
    
    response=$(curl -s --max-time 10 "http://localhost:${port}/v1/sys/health" 2>/dev/null)
    if [[ $? -eq 0 && -n "$response" ]]; then
        echo "healthy"
    else
        echo "unreachable"
    fi
}

check_generic_health() {
    local resource="$1"
    local port="$2"
    local response
    
    # Try common health endpoints
    for endpoint in "/" "/health" "/healthz" "/ping"; do
        response=$(curl -s --max-time 10 "http://localhost:${port}${endpoint}" 2>/dev/null)
        if [[ $? -eq 0 && -n "$response" ]]; then
            echo "healthy"
            return
        fi
    done
    
    echo "unreachable"
}

# Get detailed resource information
get_resource_info() {
    local resource="$1"
    
    # Try to get resource info from various sources
    local info="{}"
    
    # First, try to get port mapping from Docker
    local docker_port
    docker_port=$(docker port "$resource" 2>/dev/null | head -n1)
    
    if [[ -n "$docker_port" ]]; then
        # Extract host port from format: 3000/tcp -> 0.0.0.0:4110
        local host_port
        host_port=$(echo "$docker_port" | grep -oP '(?<=:)\d+$')
        
        if [[ -n "$host_port" ]]; then
            info=$(echo "$info" | jq --arg port "$host_port" '. + {port: $port}')
            log_debug "Found Docker port mapping for $resource: $host_port" >&2
        fi
    else
        # Fallback to known port mappings
        case "$resource" in
            "ollama") host_port="11434" ;;
            "whisper") host_port="8090" ;;
            "browserless") host_port="4110" ;;
            "agent-s2") host_port="4113" ;;
            "n8n") host_port="5678" ;;
            "node-red") host_port="1880" ;;
            "minio") host_port="9000" ;;
            "vault") host_port="8200" ;;
            "qdrant") host_port="6333" ;;
            "searxng") host_port="8100" ;;
            "huginn") host_port="4111" ;;
            "comfyui") host_port="8188" ;;
            *) host_port="" ;;
        esac
        
        if [[ -n "$host_port" ]]; then
            info=$(echo "$info" | jq --arg port "$host_port" '. + {port: $port}')
            log_debug "Using known port mapping for $resource: $host_port" >&2
        fi
    fi
    
    echo "$info"
}