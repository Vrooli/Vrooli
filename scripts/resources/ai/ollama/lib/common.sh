#!/usr/bin/env bash

# Ollama Common Utilities
# This file contains argument parsing, usage display, and other common utility functions

#######################################
# Parse command line arguments
#######################################
ollama::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|models|available|info|prompt" \
        --default ""
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if Ollama appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "models" \
        --flag "m" \
        --desc "Comma-separated list of models to install (empty = default models)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "skip-models" \
        --desc "Skip model installation during setup" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "text" \
        --flag "t" \
        --desc "Text prompt to send to Ollama (for prompt action)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "model" \
        --desc "Specific model to use (for prompt action)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "type" \
        --desc "Model type for prompt action (general|code|reasoning|vision, defaults to general)" \
        --type "value" \
        --options "general|code|reasoning|vision" \
        --default "general"
    
    args::register \
        --name "format" \
        --desc "Output format (text|json, defaults to text)" \
        --type "value" \
        --options "text|json" \
        --default "text"
    
    args::register \
        --name "temperature" \
        --desc "Temperature for randomness (0.0-2.0, defaults to 0.8)" \
        --type "value" \
        --default "0.8"
    
    args::register \
        --name "max-tokens" \
        --desc "Maximum tokens to generate (defaults to model limit)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "top-p" \
        --desc "Top-p nucleus sampling (0.0-1.0, defaults to 0.9)" \
        --type "value" \
        --default "0.9"
    
    args::register \
        --name "top-k" \
        --desc "Top-k sampling (defaults to 40)" \
        --type "value" \
        --default "40"
    
    args::register \
        --name "seed" \
        --desc "Random seed for reproducibility" \
        --type "value" \
        --default ""
    
    args::register \
        --name "system" \
        --desc "System prompt to set context" \
        --type "value" \
        --default ""
    
    if args::is_asking_for_help "$@"; then
        ollama::usage
        exit 0
    fi
    
    # Handle version request (check arguments manually)
    for arg in "$@"; do
        if [[ "$arg" == "--version" ]]; then
            ollama::version
            exit 0
        fi
    done
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export MODELS_INPUT=$(args::get "models")
    export SKIP_MODELS=$(args::get "skip-models")
    export PROMPT_TEXT=$(args::get "text")
    export PROMPT_MODEL=$(args::get "model")
    export PROMPT_TYPE=$(args::get "type")
    export OUTPUT_FORMAT=$(args::get "format")
    export TEMPERATURE=$(args::get "temperature")
    export MAX_TOKENS=$(args::get "max-tokens")
    export TOP_P=$(args::get "top-p")
    export TOP_K=$(args::get "top-k")
    export SEED=$(args::get "seed")
    export SYSTEM_PROMPT=$(args::get "system")
}

#######################################
# Display usage information
#######################################
ollama::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                              # Install Ollama with default models (llama3.1:8b, deepseek-r1:8b, qwen2.5-coder:7b)"
    echo "  $0 --action install --skip-models               # Install Ollama without models"
    echo "  $0 --action install --models 'llama3.1:8b,phi-4:14b'  # Install with specific models"
    echo "  $0 --action prompt --text 'What is the capital of France?'  # Send prompt using general model type (default)"
    echo "  $0 --action prompt --text 'Write a function' --type code  # Use code-specialized model"
    echo "  $0 --action prompt --text 'Solve 2x+5=15' --type reasoning  # Use reasoning model"
    echo "  $0 --action prompt --text 'Hello' --model qwen2.5-coder:7b  # Force specific model"
    echo "  $0 --action prompt --text 'Explain quantum physics' --format json  # Get JSON formatted response"
    echo "  $0 --action prompt --text 'Be creative' --temperature 1.5  # Higher temperature for creativity"
    echo "  $0 --action prompt --text 'Write a haiku' --max-tokens 50  # Limit response length"
    echo "  $0 --action prompt --text 'Code review' --system 'You are a senior developer'  # Set context"
    echo "  $0 --action available                           # Show available models from catalog"
    echo "  $0 --action status                               # Check Ollama status"
    echo "  $0 --action models                               # List installed models"
    echo "  $0 --action uninstall                           # Remove Ollama"
}

#######################################
# Display version information
#######################################
ollama::version() {
    echo "ollama resource manager v1.0"
    echo "Vrooli AI resource for local LLM inference"
    
    # Show installed Ollama version if available
    if system::is_command "ollama"; then
        echo "Installed Ollama version:"
        ollama version 2>/dev/null || echo "  Unable to determine Ollama version"
    else
        echo "Ollama binary not found - run '$0 --action install' to install"
    fi
}

#######################################
# Update Vrooli configuration with Ollama settings
#######################################
ollama::update_config() {
    # Create properly formatted JSON in a single line for jq compatibility
    local additional_config='{"models":{"defaultModel":"llama3.1:8b","supportsFunctionCalling":true},"api":{"version":"v1","modelsEndpoint":"/api/tags","chatEndpoint":"/api/chat","generateEndpoint":"/api/generate"}}'
    
    resources::update_config "ai" "ollama" "$OLLAMA_BASE_URL" "$additional_config"
}

#######################################
# Validation helper functions
#######################################

#######################################
# Validate temperature parameter
# Arguments:
#   $1 - temperature value
# Returns: 0 if valid, 1 if invalid
#######################################
ollama::validate_temperature() {
    local temp="$1"
    
    # Check if it's a valid number (including negative for proper rejection)
    if ! [[ "$temp" =~ ^-?[0-9]*\.?[0-9]+$ ]]; then
        return 1
    fi
    
    # Use awk for floating point comparison (more portable than bc)
    if awk -v temp="$temp" 'BEGIN { if (temp >= 0.0 && temp <= 2.0) exit 0; else exit 1 }'; then
        return 0
    else
        return 1
    fi
}

#######################################
# Validate top-p parameter
# Arguments:
#   $1 - top-p value
# Returns: 0 if valid, 1 if invalid
#######################################
ollama::validate_top_p() {
    local top_p="$1"
    
    # Check if it's a valid number (including negative for proper rejection)
    if ! [[ "$top_p" =~ ^-?[0-9]*\.?[0-9]+$ ]]; then
        return 1
    fi
    
    # Use awk for floating point comparison (more portable than bc)
    if awk -v top_p="$top_p" 'BEGIN { if (top_p >= 0.0 && top_p <= 1.0) exit 0; else exit 1 }'; then
        return 0
    else
        return 1
    fi
}

#######################################
# Validate top-k parameter
# Arguments:
#   $1 - top-k value
# Returns: 0 if valid, 1 if invalid
#######################################
ollama::validate_top_k() {
    local top_k="$1"
    
    # Check if it's a valid positive integer
    if [[ "$top_k" =~ ^[1-9][0-9]*$ ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Validate max-tokens parameter
# Arguments:
#   $1 - max-tokens value
# Returns: 0 if valid, 1 if invalid
#######################################
ollama::validate_max_tokens() {
    local max_tokens="$1"
    
    # Empty is valid (uses model default)
    if [[ -z "$max_tokens" ]]; then
        return 0
    fi
    
    # Check if it's a valid positive integer
    if [[ "$max_tokens" =~ ^[1-9][0-9]*$ ]]; then
        return 0
    else
        return 1
    fi
}

# Export functions for use in tests and subshells
export -f ollama::parse_arguments ollama::usage ollama::version ollama::update_config
export -f ollama::validate_temperature ollama::validate_top_p
export -f ollama::validate_top_k ollama::validate_max_tokens