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

# Source helper functions early
if [[ -f "$SCRIPT_DIR/framework/helpers/metadata.sh" ]]; then
    source "$SCRIPT_DIR/framework/helpers/metadata.sh"
fi

# Discover all available resources from the resource scripts
discover_all_resources() {
    # If HEALTHY_RESOURCES_STR is already set (from index.sh), use it
    if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
        log_info "Using pre-discovered healthy resources: $HEALTHY_RESOURCES_STR"
        IFS=' ' read -ra HEALTHY_RESOURCES <<< "$HEALTHY_RESOURCES_STR"
        ALL_RESOURCES=("${HEALTHY_RESOURCES[@]}")
        ENABLED_RESOURCES=("${HEALTHY_RESOURCES[@]}")
        log_success "Pre-discovered ${#ALL_RESOURCES[@]} healthy resources: ${ALL_RESOURCES[*]}"
        return 0
    fi
    
    # Fallback to original discovery method for direct test execution
    log_info "Auto-discovering resources..."
    
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
                log_warning "⚠️  Including $resource - running but not configured (auto-discovered)"
            else
                DISABLED_RESOURCES+=("$resource")
                log_debug "⚠️  Skipping $resource - not enabled and not running"
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
    
    # Count auto-discovered resources for summary
    local auto_discovered_count=0
    for resource in "${ENABLED_RESOURCES[@]}"; do
        local enabled
        enabled=$(jq -r --arg resource "$resource" '
            .services | to_entries[] | 
            select(.value | objects | has($resource)) | 
            .value[$resource].enabled // false
        ' "$config_file" 2>/dev/null)
        
        if [[ "$enabled" != "true" ]]; then
            auto_discovered_count=$((auto_discovered_count + 1))
        fi
    done
    
    if [[ $auto_discovered_count -gt 0 ]]; then
        log_info "📝 $auto_discovered_count resources were auto-discovered (running but not in config)"
        log_info "   To configure them: edit ~/.vrooli/resources.local.json"
    fi
}

# Validate health of enabled resources
validate_resource_health() {
    # If resources were pre-discovered and validated, skip re-validation
    if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
        log_info "Resources already validated by discovery system, skipping health checks"
        # HEALTHY_RESOURCES was already populated in discover_all_resources
        return 0
    fi
    
    # Original validation logic for direct test execution
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
    
    # Special handling for CLI tools and non-containerized resources
    case "$resource" in
        "claude-code")
            # Claude Code is a CLI tool, check if it's installed and available
            if which claude-code >/dev/null 2>&1 || npx claude-code --version >/dev/null 2>&1; then
                echo "true"
            else
                echo "false"
            fi
            return
            ;;
        "questdb")
            # QuestDB might be available but not running - check for installation/availability
            # For now, assume it's available if discovered (will be handled by health check)
            echo "true"
            return
            ;;
    esac
    
    # Get all running container names
    local running_containers
    running_containers=$(docker ps --format "{{.Names}}" 2>/dev/null || echo "")
    
    # Check various naming patterns for the resource
    if [[ -n "$running_containers" ]]; then
        # Exact match
        if echo "$running_containers" | grep -q "^${resource}$"; then
            echo "true"
            return
        fi
        
        # Common prefixed patterns (vrooli-, project-)
        if echo "$running_containers" | grep -qE "^(vrooli-|project-)?${resource}(-[0-9]+)?$"; then
            echo "true"
            return
        fi
        
        # Pattern with underscores instead of hyphens
        local resource_underscore="${resource//-/_}"
        if echo "$running_containers" | grep -qE "^(vrooli[_-]|project[_-])?${resource_underscore}([_-][0-9]+)?$"; then
            echo "true"
            return
        fi
    fi
    
    echo "false"
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
            # Windmill uses port 5681 and exposes /api/version endpoint
            if curl -s --max-time 5 "http://localhost:5681/api/version" >/dev/null 2>&1; then
                echo "healthy"
            else
                echo "unreachable"
            fi
            return
            ;;
        "unstructured-io")
            # Unstructured-IO uses port 11450 and exposes /healthcheck endpoint
            if curl -s --max-time 5 "http://localhost:11450/healthcheck" >/dev/null 2>&1; then
                echo "healthy"
            else
                echo "unreachable"
            fi
            return
            ;;
        "claude-code")
            # Claude Code is a CLI tool, check if it responds to basic commands
            if which claude-code >/dev/null 2>&1; then
                if timeout 5 claude-code --version >/dev/null 2>&1; then
                    echo "healthy"
                else
                    echo "unreachable"
                fi
            elif timeout 10 npx claude-code --version >/dev/null 2>&1; then
                echo "healthy"
            else
                echo "unreachable"
            fi
            return
            ;;
        "questdb")
            # QuestDB uses port 9009 and exposes HTTP API
            if curl -s --max-time 5 "http://localhost:9009/status" >/dev/null 2>&1; then
                echo "healthy"
            elif curl -s --max-time 5 "http://localhost:9009/" >/dev/null 2>&1; then
                echo "healthy"
            else
                echo "unreachable"
            fi
            return
            ;;
        "postgres")
            check_postgres_health
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

check_postgres_health() {
    # PostgreSQL doesn't expose HTTP endpoints, so we check via database connection
    # First, try to find running postgres containers
    local running_containers
    running_containers=$(docker ps --format "{{.Names}}" 2>/dev/null | grep -E "(postgres|postgresql)" || echo "")
    
    if [[ -z "$running_containers" ]]; then
        echo "unreachable"
        return
    fi
    
    # Try to connect to PostgreSQL instances on common ports
    local ports=(5433 5434 5435 5436 5437 5438 5439)
    local healthy_found=false
    
    for port in "${ports[@]}"; do
        # Check if port is open
        if timeout 3 bash -c "</dev/tcp/localhost/$port" 2>/dev/null; then
            # Try a simple PostgreSQL connection test
            if which pg_isready >/dev/null 2>&1; then
                if pg_isready -h localhost -p "$port" -U postgres -t 3 >/dev/null 2>&1; then
                    healthy_found=true
                    break
                fi
            else
                # Fallback: if port is open and we have containers, assume healthy
                healthy_found=true
                break
            fi
        fi
    done
    
    if [[ "$healthy_found" == "true" ]]; then
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
            "postgres") host_port="5433" ;;
            *) host_port="" ;;
        esac
        
        if [[ -n "$host_port" ]]; then
            info=$(echo "$info" | jq --arg port "$host_port" '. + {port: $port}')
            log_debug "Using known port mapping for $resource: $host_port" >&2
        fi
    fi
    
    echo "$info"
}

# ====================================================================
# Scenario Discovery Functions
# ====================================================================

# Discover all available scenarios from the scenarios directory
discover_scenarios() {
    log_info "Discovering available business scenarios..."
    
    local scenarios_dir="$SCRIPT_DIR/scenarios"
    
    if [[ ! -d "$scenarios_dir" ]]; then
        log_warning "Scenarios directory not found: $scenarios_dir"
        return 0
    fi
    
    # Find all scenario test files
    local scenario_files
    mapfile -t scenario_files < <(find "$scenarios_dir" -name "*.test.sh" -type f 2>/dev/null | sort)
    
    if [[ ${#scenario_files[@]} -eq 0 ]]; then
        log_warning "No scenario test files found"
        return 0
    fi
    
    # Process each scenario file to extract metadata
    declare -g -A SCENARIO_METADATA
    declare -g -a ALL_SCENARIOS
    
    for scenario_file in "${scenario_files[@]}"; do
        local scenario_name
        scenario_name=$(basename "$scenario_file" .test.sh)
        
        # Extract metadata from the scenario file
        local metadata_file="/tmp/scenario_metadata_$$"
        if extract_test_metadata "$scenario_file" "$metadata_file"; then
            source "$metadata_file"
            
            # Store scenario metadata
            SCENARIO_METADATA["$scenario_name:scenario"]="${SCENARIO:-$scenario_name}"
            SCENARIO_METADATA["$scenario_name:category"]="${CATEGORY:-unknown}"
            SCENARIO_METADATA["$scenario_name:complexity"]="${COMPLEXITY:-unknown}"
            SCENARIO_METADATA["$scenario_name:services"]="${SERVICES:-}"
            SCENARIO_METADATA["$scenario_name:business_value"]="${BUSINESS_VALUE:-unknown}"
            SCENARIO_METADATA["$scenario_name:market_demand"]="${MARKET_DEMAND:-unknown}"
            SCENARIO_METADATA["$scenario_name:revenue_potential"]="${REVENUE_POTENTIAL:-unknown}"
            SCENARIO_METADATA["$scenario_name:file"]="$scenario_file"
            
            ALL_SCENARIOS+=("$scenario_name")
            
            log_debug "Found scenario: $scenario_name (${CATEGORY:-unknown}, ${COMPLEXITY:-unknown})"
            
            rm -f "$metadata_file"
        else
            log_warning "Failed to extract metadata from $scenario_file"
        fi
    done
    
    log_success "Discovered ${#ALL_SCENARIOS[@]} business scenarios: ${ALL_SCENARIOS[*]}"
}

# Filter scenarios based on available resources
filter_runnable_scenarios() {
    log_info "Filtering scenarios based on available resources..."
    
    if [[ ${#ALL_SCENARIOS[@]} -eq 0 ]]; then
        log_warning "No scenarios to filter"
        return 0
    fi
    
    declare -g -a RUNNABLE_SCENARIOS
    declare -g -a BLOCKED_SCENARIOS
    
    local available_resources_str="${HEALTHY_RESOURCES[*]}"
    
    for scenario in "${ALL_SCENARIOS[@]}"; do
        local required_services="${SCENARIO_METADATA["$scenario:services"]}"
        
        if [[ -z "$required_services" ]]; then
            log_debug "Scenario $scenario has no service requirements - marking as runnable"
            RUNNABLE_SCENARIOS+=("$scenario")
            continue
        fi
        
        # Check if all required services are available
        local missing_services=""
        IFS=',' read -ra services <<< "$required_services"
        
        local can_run=true
        for service in "${services[@]}"; do
            service=$(echo "$service" | xargs)  # Trim whitespace
            if [[ -n "$service" && " $available_resources_str " != *" $service "* ]]; then
                can_run=false
                if [[ -n "$missing_services" ]]; then
                    missing_services="$missing_services,$service"
                else
                    missing_services="$service"
                fi
            fi
        done
        
        if [[ "$can_run" == "true" ]]; then
            RUNNABLE_SCENARIOS+=("$scenario")
            log_debug "✅ Scenario $scenario is runnable (services: $required_services)"
        else
            BLOCKED_SCENARIOS+=("$scenario")
            log_debug "⚠️  Scenario $scenario blocked - missing: $missing_services"
        fi
    done
    
    log_success "Found ${#RUNNABLE_SCENARIOS[@]} runnable scenarios: ${RUNNABLE_SCENARIOS[*]}"
    
    if [[ ${#BLOCKED_SCENARIOS[@]} -gt 0 ]]; then
        log_info "Blocked scenarios (${#BLOCKED_SCENARIOS[@]}): ${BLOCKED_SCENARIOS[*]}"
    fi
}

# Filter scenarios by metadata criteria
filter_scenarios_by_criteria() {
    local filter_criteria="$1"  # e.g., "category=customer-service,complexity=intermediate"
    
    log_info "Filtering scenarios by criteria: $filter_criteria"
    
    if [[ ${#RUNNABLE_SCENARIOS[@]} -eq 0 ]]; then
        log_warning "No runnable scenarios to filter"
        return 0
    fi
    
    # Parse filter criteria
    declare -A filters
    if [[ -n "$filter_criteria" ]]; then
        IFS=',' read -ra criteria <<< "$filter_criteria"
        for criterion in "${criteria[@]}"; do
            if [[ "$criterion" =~ ^([^=]+)=(.+)$ ]]; then
                local key="${BASH_REMATCH[1]}"
                local value="${BASH_REMATCH[2]}"
                key=$(echo "$key" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
                filters["$key"]="$value"
            fi
        done
    fi
    
    declare -g -a FILTERED_SCENARIOS
    
    for scenario in "${RUNNABLE_SCENARIOS[@]}"; do
        local matches=true
        
        for key in "${!filters[@]}"; do
            local expected_value="${filters[$key]}"
            local actual_value="${SCENARIO_METADATA["$scenario:${key,,}"]}"  # Convert to lowercase
            
            if [[ "$actual_value" != *"$expected_value"* ]]; then
                matches=false
                break
            fi
        done
        
        if [[ "$matches" == "true" ]]; then
            FILTERED_SCENARIOS+=("$scenario")
        fi
    done
    
    if [[ ${#FILTERED_SCENARIOS[@]} -gt 0 ]]; then
        log_success "Filtered to ${#FILTERED_SCENARIOS[@]} scenarios: ${FILTERED_SCENARIOS[*]}"
    else
        log_warning "No scenarios match the specified criteria"
    fi
}

# Get scenario metadata for reporting
get_scenario_metadata() {
    local scenario="$1"
    local field="$2"
    
    echo "${SCENARIO_METADATA["$scenario:$field"]:-}"
}

# List scenarios with their metadata
list_scenarios() {
    local format="${1:-table}"  # table or json
    
    if [[ ${#ALL_SCENARIOS[@]} -eq 0 ]]; then
        echo "No scenarios discovered"
        return
    fi
    
    if [[ "$format" == "json" ]]; then
        echo "["
        local first=true
        for scenario in "${ALL_SCENARIOS[@]}"; do
            if [[ "$first" != "true" ]]; then
                echo ","
            fi
            echo "  {"
            echo "    \"name\": \"$scenario\","
            echo "    \"scenario\": \"$(get_scenario_metadata "$scenario" "scenario")\","
            echo "    \"category\": \"$(get_scenario_metadata "$scenario" "category")\","
            echo "    \"complexity\": \"$(get_scenario_metadata "$scenario" "complexity")\","
            echo "    \"services\": \"$(get_scenario_metadata "$scenario" "services")\","
            echo "    \"business_value\": \"$(get_scenario_metadata "$scenario" "business_value")\","
            echo "    \"market_demand\": \"$(get_scenario_metadata "$scenario" "market_demand")\","
            echo "    \"revenue_potential\": \"$(get_scenario_metadata "$scenario" "revenue_potential")\""
            echo -n "  }"
            first=false
        done
        echo ""
        echo "]"
    else
        printf "%-30s %-20s %-15s %-30s %-20s\n" "Scenario" "Category" "Complexity" "Services" "Revenue Potential"
        printf "%-30s %-20s %-15s %-30s %-20s\n" "--------" "--------" "----------" "--------" "-----------------"
        
        for scenario in "${ALL_SCENARIOS[@]}"; do
            local category=$(get_scenario_metadata "$scenario" "category")
            local complexity=$(get_scenario_metadata "$scenario" "complexity")
            local services=$(get_scenario_metadata "$scenario" "services")
            local revenue=$(get_scenario_metadata "$scenario" "revenue_potential")
            
            printf "%-30s %-20s %-15s %-30s %-20s\n" \
                "$scenario" \
                "${category:-unknown}" \
                "${complexity:-unknown}" \
                "${services:-none}" \
                "${revenue:-unknown}"
        done
    fi
}