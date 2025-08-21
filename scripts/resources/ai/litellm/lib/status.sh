#!/bin/bash
# LiteLLM status functionality

# Get script directory
LITELLM_STATUS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${LITELLM_STATUS_DIR}/core.sh"
source "${LITELLM_STATUS_DIR}/docker.sh"
# shellcheck disable=SC1091
source "${LITELLM_STATUS_DIR}/../../lib/status-args.sh"

#######################################
# Collect LiteLLM status data
# Arguments:
#   --fast: Skip expensive operations
# Returns:
#   Key-value pairs (one per line)
#######################################
litellm::status::collect_data() {
    local fast="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Initialize
    litellm::init >/dev/null 2>&1
    
    local status="unknown"
    local message=""
    local container_status
    local api_base="$LITELLM_API_BASE"
    local proxy_version="N/A"
    local master_key_status="not_configured"
    
    # Check container status
    container_status=$(litellm::container_status)
    
    case "$container_status" in
        "running")
            # Container is running, check API health
            if [[ "$fast" == "true" ]]; then
                status="running"
                message="LiteLLM proxy server is running (fast mode)"
            elif litellm::test_connection 5 false; then
                status="running"
                message="LiteLLM proxy server is healthy"
            else
                status="degraded"
                message="LiteLLM container running but API not responding"
            fi
            ;;
        "stopped")
            status="stopped"
            message="LiteLLM container exists but is stopped"
            ;;
        "not_created")
            status="not_installed"
            message="LiteLLM container not found (run install first)"
            ;;
        *)
            status="unknown"
            message="Unable to determine container status"
            ;;
    esac
    
    # Check master key configuration
    if litellm::get_master_key >/dev/null 2>&1; then
        master_key_status="configured"
    fi
    
    # Determine running status
    local running="false"
    if [[ "$status" == "running" || "$status" == "degraded" ]]; then
        running="true"
    fi
    
    # Determine health status
    local healthy="false"
    if [[ "$status" == "running" ]]; then
        healthy="true"
    fi
    
    # Get proxy version and metrics (skip in fast mode or if not running)
    local model_count="N/A"
    local provider_count="N/A"
    local uptime="N/A"
    
    if [[ "$fast" == "false" && "$status" == "running" ]]; then
        # Get version from container
        proxy_version=$(docker exec "$LITELLM_CONTAINER_NAME" python -c "import litellm; print(litellm.__version__)" 2>/dev/null || echo "unknown")
        
        # Get model count
        local models
        models=$(litellm::list_models 10 false 2>/dev/null)
        if [[ -n "$models" ]]; then
            model_count=$(echo "$models" | wc -l)
        fi
        
        # Get provider count from config
        if [[ -f "$LITELLM_CONFIG_FILE" ]]; then
            provider_count=$(yq eval '.model_list | length' "$LITELLM_CONFIG_FILE" 2>/dev/null || echo "unknown")
        fi
        
        # Get container uptime
        local start_time
        start_time=$(docker inspect --format='{{.State.StartedAt}}' "$LITELLM_CONTAINER_NAME" 2>/dev/null)
        if [[ -n "$start_time" ]]; then
            local start_epoch
            start_epoch=$(date -d "$start_time" +%s 2>/dev/null)
            if [[ -n "$start_epoch" ]]; then
                local current_epoch
                current_epoch=$(date +%s)
                local uptime_seconds=$((current_epoch - start_epoch))
                uptime="${uptime_seconds}s"
            fi
        fi
    fi
    
    # Check test results
    local test_status="not_run"
    local test_timestamp="N/A"
    local test_file="${var_ROOT_DIR}/data/test-results/litellm-test.json"
    
    if [[ -f "$test_file" ]]; then
        test_status=$(jq -r '.status // "unknown"' "$test_file" 2>/dev/null || echo "unknown")
        test_timestamp=$(jq -r '.timestamp // "N/A"' "$test_file" 2>/dev/null || echo "N/A")
    fi
    
    # Get configuration status
    local config_status="not_found"
    if [[ -f "$LITELLM_CONFIG_FILE" ]]; then
        config_status="found"
    fi
    
    # Check provider key availability (skip in fast mode)
    local providers_configured="N/A"
    if [[ "$fast" == "false" && -f "$LITELLM_ENV_FILE" ]]; then
        local configured_count=0
        local providers=("OPENAI_API_KEY" "ANTHROPIC_API_KEY" "OPENROUTER_API_KEY" "GOOGLE_API_KEY")
        
        for provider in "${providers[@]}"; do
            if grep -q "^${provider}=" "$LITELLM_ENV_FILE" && ! grep -q "^${provider}=.*PLACEHOLDER" "$LITELLM_ENV_FILE"; then
                configured_count=$((configured_count + 1))
            fi
        done
        
        providers_configured="$configured_count/4"
    fi
    
    # Output data as key-value pairs
    echo "status"
    echo "$status"
    echo "running"
    echo "$running"
    echo "healthy"
    echo "$healthy"
    echo "message"
    echo "$message"
    echo "container_status"
    echo "$container_status"
    echo "api_base"
    echo "$api_base"
    echo "proxy_version"
    echo "$proxy_version"
    echo "master_key_status"
    echo "$master_key_status"
    echo "config_status"
    echo "$config_status"
    echo "model_count"
    echo "$model_count"
    echo "provider_count"
    echo "$provider_count"
    echo "providers_configured"
    echo "$providers_configured"
    echo "uptime"
    echo "$uptime"
    echo "test_status"
    echo "$test_status"
    echo "test_timestamp"
    echo "$test_timestamp"
}

#######################################
# Display LiteLLM status in text format
# Arguments:
#   data_array: Array of key-value pairs
#######################################
litellm::status::display_text() {
    local -a data_array=("$@")
    
    # Convert array to associative array for easier access
    local -A data
    for ((i=0; i<${#data_array[@]}; i+=2)); do
        data["${data_array[i]}"]="${data_array[i+1]}"
    done
    
    echo "Status: ${data[status]}"
    echo "Running: ${data[running]}"
    echo "Healthy: ${data[healthy]}"
    echo "Message: ${data[message]}"
    echo "Container Status: ${data[container_status]}"
    echo "API Base: ${data[api_base]}"
    echo "Proxy Version: ${data[proxy_version]}"
    echo "Master Key: ${data[master_key_status]}"
    echo "Configuration: ${data[config_status]}"
    echo "Models Available: ${data[model_count]}"
    echo "Providers Configured: ${data[provider_count]}"
    echo "Providers with Keys: ${data[providers_configured]}"
    echo "Uptime: ${data[uptime]}"
    echo "Test Status: ${data[test_status]}"
    echo "Test Timestamp: ${data[test_timestamp]}"
}

#######################################
# Check LiteLLM status
# Arguments:
#   --format: Output format (text/json)
#   --fast: Skip expensive operations
# Returns:
#   0 if healthy, 1 otherwise
#######################################
litellm::status() {
    status::run_standard "litellm" "litellm::status::collect_data" "litellm::status::display_text" "$@"
}

# Get detailed status with extended information
litellm::status::detailed() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Getting detailed LiteLLM status"
    
    # Basic status
    litellm::status --format text
    
    echo
    echo "=== Extended Information ==="
    
    # Configuration details
    echo "Configuration File: $LITELLM_CONFIG_FILE"
    echo "Environment File: $LITELLM_ENV_FILE"
    echo "Data Directory: $LITELLM_DATA_DIR"
    echo "Log Directory: $LITELLM_LOG_DIR"
    
    # Container details
    if litellm::is_running; then
        echo
        echo "=== Container Information ==="
        echo "Container Name: $LITELLM_CONTAINER_NAME"
        echo "Image: $LITELLM_IMAGE"
        echo "Network: $LITELLM_NETWORK"
        echo "Port Mapping: ${LITELLM_PORT}:${LITELLM_INTERNAL_PORT}"
        
        # Resource usage
        echo
        echo "=== Resource Usage ==="
        litellm::stats 2>/dev/null || echo "Unable to get resource statistics"
        
        # Recent logs
        echo
        echo "=== Recent Logs (last 10 lines) ==="
        litellm::logs 10 false 2>/dev/null || echo "Unable to get logs"
    fi
    
    # Provider key status
    echo
    echo "=== Provider Configuration ==="
    local providers=("OPENAI_API_KEY" "ANTHROPIC_API_KEY" "OPENROUTER_API_KEY" "GOOGLE_API_KEY")
    
    for provider in "${providers[@]}"; do
        local status="❌ Not configured"
        
        if [[ -f "$LITELLM_ENV_FILE" ]] && grep -q "^${provider}=" "$LITELLM_ENV_FILE"; then
            if grep -q "^${provider}=.*PLACEHOLDER" "$LITELLM_ENV_FILE"; then
                status="⚠️  Placeholder value"
            else
                status="✅ Configured"
            fi
        fi
        
        echo "$provider: $status"
    done
    
    # Model availability (if running)
    if litellm::is_running; then
        echo
        echo "=== Available Models ==="
        local models
        models=$(litellm::list_models 10 false 2>/dev/null)
        
        if [[ -n "$models" ]]; then
            echo "$models" | head -10
            local total_count
            total_count=$(echo "$models" | wc -l)
            if [[ $total_count -gt 10 ]]; then
                echo "... and $((total_count - 10)) more models"
            fi
        else
            echo "No models available or unable to fetch model list"
        fi
    fi
}

# Export function
export -f litellm::status