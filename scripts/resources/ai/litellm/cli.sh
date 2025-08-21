#!/bin/bash
# LiteLLM CLI interface

# Get script directory
LITELLM_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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
        echo "🔍 Checking LiteLLM status..."
        if docker ps --format '{{.Names}}' | grep -q "^${LITELLM_CONTAINER}$"; then
            echo "✅ LiteLLM is running"
            echo "📍 URL: ${LITELLM_URL}"
            echo "🔑 Master Key: ${LITELLM_MASTER_KEY}"
            
            # Test the health endpoint
            if curl -s -f -H "Authorization: Bearer ${LITELLM_MASTER_KEY}" "${LITELLM_URL}/health" >/dev/null 2>&1; then
                echo "💚 API is healthy"
            else
                echo "⚠️  API is not responding"
            fi
        else
            echo "❌ LiteLLM is not running"
            echo "Run: resource-litellm start"
        fi
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

  status, s             Check if LiteLLM is running
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