#\!/bin/bash

# Common variables and functions for AutoGPT resource

# Container configuration
AUTOGPT_CONTAINER_NAME="autogpt-main"
AUTOGPT_IMAGE="significantgravitas/auto-gpt:latest"

# Use port registry for port allocation
AUTOGPT_PORT_API="${AUTOGPT_PORT_API:-8501}"
AUTOGPT_PORT_UI="${AUTOGPT_PORT_UI:-8502}"

# Data directories
AUTOGPT_DATA_DIR="${var_DATA_DIR:-/home/matthalloran8/Vrooli/data}/resources/autogpt"
AUTOGPT_AGENTS_DIR="$AUTOGPT_DATA_DIR/agents"
AUTOGPT_TOOLS_DIR="$AUTOGPT_DATA_DIR/tools"
AUTOGPT_WORKSPACE_DIR="$AUTOGPT_DATA_DIR/workspace"
AUTOGPT_LOGS_DIR="$AUTOGPT_DATA_DIR/logs"
AUTOGPT_CONFIG_DIR="$AUTOGPT_DATA_DIR/config"

# Resource metadata
AUTOGPT_RESOURCE_NAME="autogpt"
AUTOGPT_RESOURCE_CATEGORY="execution"
AUTOGPT_RESOURCE_DESCRIPTION="Autonomous AI agent framework for task automation"

# Ensure data directories exist
autogpt_ensure_directories() {
    mkdir -p "$AUTOGPT_AGENTS_DIR"
    mkdir -p "$AUTOGPT_TOOLS_DIR"
    mkdir -p "$AUTOGPT_WORKSPACE_DIR"
    mkdir -p "$AUTOGPT_LOGS_DIR"
    mkdir -p "$AUTOGPT_CONFIG_DIR"
}

# Check if container exists
autogpt_container_exists() {
    docker ps -a --format "{{.Names}}" | grep -q "^${AUTOGPT_CONTAINER_NAME}$"
}

# Check if container is running
autogpt_container_running() {
    docker ps --format "{{.Names}}" | grep -q "^${AUTOGPT_CONTAINER_NAME}$"
}

# Get container ID
autogpt_get_container_id() {
    docker ps -aq -f "name=${AUTOGPT_CONTAINER_NAME}"
}

# Wait for service to be ready
autogpt_wait_ready() {
    local max_attempts=30
    local attempt=0
    
    echo "[INFO]    Waiting for AutoGPT to be ready..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "http://localhost:${AUTOGPT_PORT_API}/health" >/dev/null 2>&1; then
            echo "[SUCCESS] AutoGPT is ready"
            return 0
        fi
        
        attempt=$((attempt + 1))
        sleep 2
    done
    
    echo "[ERROR]   AutoGPT failed to become ready"
    return 1
}

# Get LLM provider configuration
autogpt_get_llm_config() {
    # Check for available providers in order of preference
    if resource-openrouter status --format json 2>/dev/null | jq -e '.healthy == true' >/dev/null 2>&1; then
        echo "openrouter"
    elif resource-ollama status --format json 2>/dev/null | jq -e '.healthy == true' >/dev/null 2>&1; then
        echo "ollama"
    else
        echo "none"
    fi
}

# Get API key from vault if available
autogpt_get_api_key() {
    local provider="$1"
    
    if [ "$provider" == "openrouter" ] && command -v resource-vault >/dev/null 2>&1; then
        resource-vault get "openrouter/api_key" 2>/dev/null || echo ""
    else
        echo ""
    fi
}
