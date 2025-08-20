#!/bin/bash

# Installation functions for AutoGPT

autogpt_install() {
    echo "[HEADER]  Installing AutoGPT..."
    
    # Ensure directories exist
    autogpt_ensure_directories
    
    # Pull the Docker image
    echo "[INFO]    Pulling AutoGPT Docker image..."
    if ! docker pull "$AUTOGPT_IMAGE"; then
        echo "[ERROR]   Failed to pull AutoGPT image"
        return 1
    fi
    
    # Create default configuration
    autogpt_create_default_config
    
    # Install CLI
    local cli_script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    "$cli_script_dir/../../../lib/resources/install-resource-cli.sh" "execution/autogpt"
    
    echo "[SUCCESS] AutoGPT installed successfully"
    return 0
}

autogpt_uninstall() {
    echo "[HEADER]  Uninstalling AutoGPT..."
    
    # Stop container if running
    if autogpt_container_running; then
        autogpt_stop
    fi
    
    # Remove container
    if autogpt_container_exists; then
        echo "[INFO]    Removing AutoGPT container..."
        docker rm -f "$AUTOGPT_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    # Uninstall CLI
    local cli_script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    "$cli_script_dir/../../../lib/resources/uninstall-resource-cli.sh" autogpt
    
    echo "[SUCCESS] AutoGPT uninstalled"
    return 0
}

autogpt_create_default_config() {
    echo "[INFO]    Creating default configuration..."
    
    # Create base config file
    cat > "$AUTOGPT_CONFIG_DIR/config.yaml" << YAML
ai_settings:
  provider: auto
  model: auto
  temperature: 0.7
  max_tokens: 4000

memory:
  backend: local
  vector_store: qdrant

workspace:
  path: /workspace

plugins:
  enabled:
    - web_browser
    - file_operations
    - code_execution

limits:
  max_iterations: 25
  max_cost_per_run: 1.0
  
debug:
  enabled: false
  log_level: INFO
YAML
    
    echo "[SUCCESS] Default configuration created"
}
