#\!/bin/bash

# Start functions for AutoGPT

autogpt_start() {
    echo "[HEADER]  Starting AutoGPT..."
    
    # Check if already running
    if autogpt_container_running; then
        echo "[INFO]    AutoGPT is already running"
        return 0
    fi
    
    # Ensure directories exist
    autogpt_ensure_directories
    
    # Get LLM configuration
    local llm_provider=$(autogpt_get_llm_config)
    local api_key=""
    
    if [ "$llm_provider" == "openrouter" ]; then
        api_key=$(autogpt_get_api_key "openrouter")
        if [ -z "$api_key" ]; then
            echo "[WARNING] No OpenRouter API key found in vault"
        fi
    fi
    
    # Prepare environment variables
    local env_vars=""
    [ -n "$api_key" ] && env_vars="$env_vars -e OPENAI_API_KEY=$api_key"
    env_vars="$env_vars -e AI_PROVIDER=$llm_provider"
    env_vars="$env_vars -e WORKSPACE_PATH=/workspace"
    env_vars="$env_vars -e LOG_DIR=/logs"
    
    # Check if container exists but is stopped
    if autogpt_container_exists; then
        echo "[INFO]    Starting existing AutoGPT container..."
        docker start "$AUTOGPT_CONTAINER_NAME"
    else
        echo "[INFO]    Creating and starting AutoGPT container..."
        docker run -d \
            --name "$AUTOGPT_CONTAINER_NAME" \
            -p "${AUTOGPT_PORT_API}:8000" \
            -p "${AUTOGPT_PORT_UI}:8501" \
            -v "$AUTOGPT_CONFIG_DIR:/app/config" \
            -v "$AUTOGPT_AGENTS_DIR:/app/agents" \
            -v "$AUTOGPT_TOOLS_DIR:/app/tools" \
            -v "$AUTOGPT_WORKSPACE_DIR:/workspace" \
            -v "$AUTOGPT_LOGS_DIR:/logs" \
            $env_vars \
            "$AUTOGPT_IMAGE"
    fi
    
    # Wait for service to be ready
    if autogpt_wait_ready; then
        echo "[SUCCESS] AutoGPT started successfully"
        echo "[INFO]    API available at: http://localhost:${AUTOGPT_PORT_API}"
        echo "[INFO]    UI available at: http://localhost:${AUTOGPT_PORT_UI}"
        return 0
    else
        echo "[ERROR]   AutoGPT failed to start properly"
        return 1
    fi
}
