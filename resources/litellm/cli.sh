#!/bin/bash
# LiteLLM CLI interface

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
LITELLM_CLI_DIR="${APP_ROOT}/resources/litellm"

# Simple configuration
LITELLM_URL="http://localhost:11435"
LITELLM_MASTER_KEY="sk-vrooli-litellm-de9e1bde85234cd9"
LITELLM_CONTAINER="vrooli-litellm"

# Function to make completions
litellm_complete() {
    local message="${1:-Hello, how are you?}"
    local model="${2:-llama2-local}"
    local max_tokens="${3:-100}"
    
    # Make the API request
    local response=$(curl -s -X POST "${LITELLM_URL}/chat/completions" \
        -H "Authorization: Bearer ${LITELLM_MASTER_KEY}" \
        -H "Content-Type: application/json" \
        -d "{
            \"model\": \"${model}\",
            \"messages\": [{\"role\": \"user\", \"content\": \"${message}\"}],
            \"max_tokens\": ${max_tokens}
        }" 2>/dev/null)
    
    # Extract the content from response
    if [[ -n "$response" ]]; then
        # Check for error
        local error=$(echo "$response" | jq -r '.error.message // empty' 2>/dev/null)
        if [[ -n "$error" ]]; then
            echo "Error: $error"
            return 1
        fi
        
        # Extract the actual response
        local content=$(echo "$response" | jq -r '.choices[0].message.content // empty' 2>/dev/null)
        if [[ -n "$content" ]]; then
            echo "$content"
        else
            echo "No response received. Response was: $response"
        fi
    else
        echo "Failed to connect to LiteLLM at ${LITELLM_URL}"
        return 1
    fi
}

# Main CLI handler
case "${1:-help}" in
    complete|chat|c)
        shift
        litellm_complete "$@"
        ;;
    
    status|s)
        # Support JSON output format
        output_format="${2:-text}"
        if [[ "$2" == "--json" ]] || [[ "$2" == "json" ]]; then
            output_format="json"
        elif [[ "$2" == "--verbose" ]]; then
            output_format="verbose"  
        fi
        
        # Check if container is running
        is_running="false"
        container_status="Not running"
        
        if docker ps --format '{{.Names}}' | grep -q "^${LITELLM_CONTAINER}$"; then
            is_running="true"
            container_status="Running"
        fi
        
        # Initialize status data
        api_healthy="false"
        available_models="[]"
        endpoint_health="{}"
        api_response_time=""
        
        if [[ "$is_running" == "true" ]]; then
            # Test API health with timing (with timeout to prevent hanging)
            start_time=$(date +%s%N)
            if timeout 15 curl -s -f -H "Authorization: Bearer ${LITELLM_MASTER_KEY}" "${LITELLM_URL}/health" >/dev/null 2>&1; then
                api_healthy="true"
                end_time=$(date +%s%N)
                api_response_time=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
            fi
            
            # Get available models
            models_response=""
            if models_response=$(curl -s -H "Authorization: Bearer ${LITELLM_MASTER_KEY}" "${LITELLM_URL}/v1/models" 2>/dev/null); then
                if echo "$models_response" | jq -e '.data' >/dev/null 2>&1; then
                    available_models=$(echo "$models_response" | jq '.data')
                fi
            fi
            
            # Get endpoint health details (with timeout to prevent hanging)
            health_response=""
            if health_response=$(timeout 15 curl -s -H "Authorization: Bearer ${LITELLM_MASTER_KEY}" "${LITELLM_URL}/health" 2>/dev/null); then
                if echo "$health_response" | jq -e '.healthy_endpoints' >/dev/null 2>&1; then
                    endpoint_health="$health_response"
                fi
            fi
        fi
        
        # Output based on format
        case "$output_format" in
            json)
                # JSON output
                jq -n \
                    --arg container_running "$is_running" \
                    --arg container_status "$container_status" \
                    --arg api_healthy "$api_healthy" \
                    --arg url "$LITELLM_URL" \
                    --arg master_key "$LITELLM_MASTER_KEY" \
                    --arg container_name "$LITELLM_CONTAINER" \
                    --arg response_time "$api_response_time" \
                    --argjson models "$available_models" \
                    --argjson health "$endpoint_health" \
                    '{
                        "container_running": ($container_running == "true"),
                        "container_status": $container_status,
                        "api_healthy": ($api_healthy == "true"),
                        "api_url": $url,
                        "master_key": $master_key,
                        "container_name": $container_name,
                        "api_response_time_ms": ($response_time | tonumber? // null),
                        "available_models": $models,
                        "endpoint_health": $health,
                        "timestamp": now | strftime("%Y-%m-%dT%H:%M:%SZ")
                    }'
                ;;
                
            verbose)
                # Verbose text output
                echo "🔍 Checking LiteLLM status..."
                echo "📦 Container: $LITELLM_CONTAINER ($container_status)"
                echo "📍 URL: $LITELLM_URL"
                echo "🔑 Master Key: $LITELLM_MASTER_KEY"
                
                if [[ "$is_running" == "true" ]]; then
                    if [[ "$api_healthy" == "true" ]]; then
                        echo "💚 API is healthy"
                        if [[ -n "$api_response_time" ]]; then
                            echo "⚡ Response time: ${api_response_time}ms"
                        fi
                        
                        # Show available models
                        model_count=$(echo "$available_models" | jq '. | length' 2>/dev/null || echo "0")
                        echo ""
                        echo "🤖 Available Models ($model_count):"
                        if [[ $model_count -gt 0 ]]; then
                            echo "$available_models" | jq -r '.[] | "   • " + .id + " (owned by: " + .owned_by + ")"' 2>/dev/null
                        else
                            echo "   No models found"
                        fi
                        
                        # Show endpoint health
                        if echo "$endpoint_health" | jq -e '.healthy_endpoints' >/dev/null 2>&1; then
                            healthy_count=$(echo "$endpoint_health" | jq -r '.healthy_count // 0')
                            unhealthy_count=$(echo "$endpoint_health" | jq -r '.unhealthy_count // 0')
                            echo ""
                            echo "💚 Endpoint Health:"
                            echo "   Healthy: $healthy_count, Unhealthy: $unhealthy_count"
                            
                            if [[ $healthy_count -gt 0 ]]; then
                                echo "   Healthy Endpoints:"
                                echo "$endpoint_health" | jq -r '.healthy_endpoints[]? | "   ✅ " + .model + (if .api_base then " (" + .api_base + ")" else "" end)' 2>/dev/null
                            fi
                            
                            if [[ $unhealthy_count -gt 0 ]]; then
                                echo "   Unhealthy Endpoints:"
                                echo "$endpoint_health" | jq -r '.unhealthy_endpoints[]? | "   ❌ " + .model + (if .api_base then " (" + .api_base + ")" else "" end)' 2>/dev/null
                            fi
                        fi
                    else
                        echo "⚠️  API is not responding"
                        echo "   Try: resource-litellm restart"
                    fi
                else
                    echo "❌ LiteLLM is not running"
                    echo "   Run: resource-litellm start"
                fi
                ;;
                
            *)
                # Default text output (backward compatible)
                echo "🔍 Checking LiteLLM status..."
                if [[ "$is_running" == "true" ]]; then
                    echo "✅ LiteLLM is running"
                    echo "📍 URL: ${LITELLM_URL}"
                    echo "🔑 Master Key: ${LITELLM_MASTER_KEY}"
                    
                    if [[ "$api_healthy" == "true" ]]; then
                        echo "💚 API is healthy"
                    else
                        echo "⚠️  API is not responding"
                    fi
                else
                    echo "❌ LiteLLM is not running"
                    echo "Run: resource-litellm start"
                fi
                ;;
        esac
        ;;
    
    start)
        echo "🚀 Starting LiteLLM..."
        if docker ps --format '{{.Names}}' | grep -q "^${LITELLM_CONTAINER}$"; then
            echo "Already running!"
        else
            docker start "${LITELLM_CONTAINER}" 2>/dev/null || {
                echo "Container doesn't exist. Creating new container..."
                docker run -d \
                    --name "${LITELLM_CONTAINER}" \
                    --network host \
                    -v "/home/matthalloran8/Vrooli/data/litellm/config/minimal-config.yaml:/app/config.yaml:ro" \
                    -e LITELLM_MASTER_KEY="${LITELLM_MASTER_KEY}" \
                    ghcr.io/berriai/litellm:main-stable \
                    --config /app/config.yaml \
                    --port 11435
            }
            echo "✅ LiteLLM started"
        fi
        ;;
    
    stop)
        echo "🛑 Stopping LiteLLM..."
        docker stop "${LITELLM_CONTAINER}" 2>/dev/null && echo "✅ Stopped" || echo "Not running"
        ;;
    
    restart)
        $0 stop
        sleep 2
        $0 start
        ;;
    
    logs|l)
        shift
        if [[ "$1" == "-f" || "$1" == "--follow" ]]; then
            docker logs -f "${LITELLM_CONTAINER}"
        else
            docker logs "${LITELLM_CONTAINER}" --tail 50
        fi
        ;;
    
    models|list-models)
        echo "📋 Available models:"
        curl -s -H "Authorization: Bearer ${LITELLM_MASTER_KEY}" \
            "${LITELLM_URL}/models" | jq -r '.data[].id' 2>/dev/null || echo "Failed to fetch models"
        ;;
    
    test)
        echo "🧪 Testing LiteLLM..."
        response=$(litellm_complete "Say 'LiteLLM is working!' if you can hear me" "llama2-local" "20")
        if [[ $? -eq 0 ]]; then
            echo "✅ Test successful!"
            echo "Response: $response"
        else
            echo "❌ Test failed"
            echo "$response"
        fi
        ;;
    
    help|h|--help|-h|*)
        cat <<EOF
🤖 LiteLLM Resource CLI
======================

Usage: resource-litellm <command> [options]

Commands:
  complete, chat, c <message> [model] [max_tokens]
                        Send a completion request to LiteLLM
                        Examples:
                          resource-litellm complete "Hello world"
                          resource-litellm c "Explain quantum physics" gpt-3.5-turbo 200
                          resource-litellm chat "What is 2+2?"

  status, s [--json|--verbose] 
                        Check if LiteLLM is running
                        --json: Output detailed status as JSON
                        --verbose: Show detailed models and endpoint health
  start                 Start LiteLLM service
  stop                  Stop LiteLLM service
  restart               Restart LiteLLM service
  logs, l [-f]          View container logs (use -f to follow)
  models, list-models   List available models
  test                  Test the LiteLLM connection
  help, h               Show this help message

Configuration:
  URL: ${LITELLM_URL}
  Master Key: ${LITELLM_MASTER_KEY}
  Container: ${LITELLM_CONTAINER}

Examples:
  # Simple chat
  resource-litellm chat "Hello, how are you?"
  
  # Use specific model
  resource-litellm complete "Write a poem" gpt-3.5-turbo 100
  
  # Check status
  resource-litellm status
  
  # View logs
  resource-litellm logs -f
EOF
        ;;
esac
