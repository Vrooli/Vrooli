#!/usr/bin/env bats

# Tests for Ollama configuration defaults

# Load test helper
setup() {
    # Set test environment to avoid port conflicts
    export OLLAMA_CUSTOM_PORT="9999"
    
    # Mock the resources::get_default_port function for testing
    resources::get_default_port() {
        echo "11434"
    }
    
    # Source the defaults
    source "${BATS_TEST_DIRNAME}/defaults.sh"
}

@test "configuration constants are set correctly" {
    [ "$OLLAMA_PORT" = "9999" ]  # Should use custom port
    [ "$OLLAMA_BASE_URL" = "http://localhost:9999" ]
    [ "$OLLAMA_SERVICE_NAME" = "ollama" ]
    [ "$OLLAMA_INSTALL_DIR" = "/usr/local/bin" ]
    [ "$OLLAMA_USER" = "ollama" ]
}

@test "model catalog is loaded" {
    # Test that MODEL_CATALOG is an associative array with expected entries
    [ "${MODEL_CATALOG['llama3.1:8b']}" ]
    [ "${MODEL_CATALOG['deepseek-r1:8b']}" ]
    [ "${MODEL_CATALOG['qwen2.5-coder:7b']}" ]
    
    # Test catalog format (size|capabilities|description)
    local model_info="${MODEL_CATALOG['llama3.1:8b']}"
    [[ "$model_info" =~ ^[0-9]+\.[0-9]+\|.*\|.* ]]
}

@test "default models are defined" {
    [ "${#DEFAULT_MODELS[@]}" -gt 0 ]
    
    # Check that default models are in the catalog
    for model in "${DEFAULT_MODELS[@]}"; do
        [ "${MODEL_CATALOG[$model]}" ]
    done
}

@test "model catalog contains expected categories" {
    # Test that we have models from different categories
    local has_general=false
    local has_code=false
    local has_reasoning=false
    
    for model in "${!MODEL_CATALOG[@]}"; do
        local capabilities=$(echo "${MODEL_CATALOG[$model]}" | cut -d'|' -f2)
        
        [[ "$capabilities" =~ general ]] && has_general=true
        [[ "$capabilities" =~ code ]] && has_code=true
        [[ "$capabilities" =~ reasoning ]] && has_reasoning=true
    done
    
    [ "$has_general" = true ]
    [ "$has_code" = true ]
    [ "$has_reasoning" = true ]
}

@test "model catalog has consistent format" {
    for model in "${!MODEL_CATALOG[@]}"; do
        local info="${MODEL_CATALOG[$model]}"
        
        # Should have exactly 2 pipe separators (3 parts: size|capabilities|description)
        local pipe_count=$(echo "$info" | tr -cd '|' | wc -c)
        [ "$pipe_count" -eq 2 ]
        
        # Size should be a number with decimal
        local size=$(echo "$info" | cut -d'|' -f1)
        [[ "$size" =~ ^[0-9]+\.[0-9]+$ ]]
        
        # Should have non-empty capabilities and description
        local capabilities=$(echo "$info" | cut -d'|' -f2)
        local description=$(echo "$info" | cut -d'|' -f3)
        [ -n "$capabilities" ]
        [ -n "$description" ]
    done
}

@test "ollama::export_config function works" {
    # Clear any existing exports
    unset OLLAMA_PORT OLLAMA_BASE_URL OLLAMA_SERVICE_NAME
    unset OLLAMA_INSTALL_DIR OLLAMA_USER
    
    # Call export function
    ollama::export_config
    
    # Verify exports
    [ "$OLLAMA_PORT" = "9999" ]
    [ "$OLLAMA_BASE_URL" = "http://localhost:9999" ]
    [ "$OLLAMA_SERVICE_NAME" = "ollama" ]
    [ "$OLLAMA_INSTALL_DIR" = "/usr/local/bin" ]
    [ "$OLLAMA_USER" = "ollama" ]
}

@test "model names follow expected format" {
    for model in "${!MODEL_CATALOG[@]}"; do
        # Model names should follow format: name:tag
        [[ "$model" =~ ^[a-zA-Z0-9._-]+:[a-zA-Z0-9._-]+$ ]]
    done
}

@test "default models have reasonable sizes" {
    for model in "${DEFAULT_MODELS[@]}"; do
        local size=$(echo "${MODEL_CATALOG[$model]}" | cut -d'|' -f1)
        local size_num=$(echo "$size" | cut -d'.' -f1)
        
        # Default models should be reasonably sized (under 20GB)
        [ "$size_num" -lt 20 ]
    done
}