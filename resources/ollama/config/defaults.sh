#!/usr/bin/env bash

# Ollama Resource Configuration Defaults
# This file contains all configuration constants and defaults for the Ollama resource

# Ollama version configuration
readonly OLLAMA_VERSION="${OLLAMA_CUSTOM_VERSION:-v0.11.7}"  # Specific version for consistency

# Ollama service configuration
readonly OLLAMA_PORT="${OLLAMA_CUSTOM_PORT:-$(resources::get_default_port "ollama" 2>/dev/null || echo "11434")}"
readonly OLLAMA_BASE_URL="http://localhost:${OLLAMA_PORT}"
readonly OLLAMA_SERVICE_NAME="ollama"
readonly OLLAMA_INSTALL_DIR="/usr/local/bin"
readonly OLLAMA_USER="ollama"

# Ollama performance configuration
readonly OLLAMA_NUM_PARALLEL="${OLLAMA_CUSTOM_NUM_PARALLEL:-16}"     # Parallel request processing
readonly OLLAMA_MAX_LOADED_MODELS="${OLLAMA_CUSTOM_MAX_LOADED_MODELS:-3}"  # Models in memory
readonly OLLAMA_FLASH_ATTENTION="${OLLAMA_CUSTOM_FLASH_ATTENTION:-1}"     # Enable flash attention
readonly OLLAMA_ORIGINS="${OLLAMA_CUSTOM_ORIGINS:-*}"                    # CORS origins

# Model catalog with metadata
# Format: ["model:tag"]="size_gb|capabilities|description"
declare -A MODEL_CATALOG=(
    # Current/Recommended Models
    ["llama3.1:8b"]="4.9|general,chat,reasoning|Latest general-purpose model from Meta"
    ["deepseek-r1:8b"]="4.7|reasoning,math,code,chain-of-thought|Advanced reasoning model with explicit thinking process"
    ["qwen2.5-coder:7b"]="4.1|code,programming,debugging|Superior code generation model, replaces CodeLlama"
    
    # Alternative Sizes
    ["llama3.3:8b"]="4.9|general,chat,reasoning|Very latest from Meta (Dec 2024)"
    ["deepseek-r1:14b"]="8.1|reasoning,math,code,chain-of-thought|Larger reasoning model for complex problems"
    ["deepseek-r1:1.5b"]="0.9|reasoning,lightweight|Smallest reasoning model for resource-constrained environments"
    
    # Embedding Models (lightweight, high parallelism supported)
    ["mxbai-embed-large:latest"]="1.2|embedding,semantic-search,1024-dim|High-quality embeddings model with 1024 dimensions"
    ["nomic-embed-text:latest"]="0.8|embedding,semantic-search,768-dim|Efficient embeddings model with 768 dimensions"
    ["bge-m3:latest"]="1.5|embedding,multilingual,1024-dim|Multilingual embeddings model with 1024 dimensions"
    
    # Specialized Models
    ["phi-4:14b"]="8.2|general,multilingual,math,function-calling|Microsoft's efficient model with multilingual support"
    ["qwen2.5:14b"]="8.0|general,multilingual,reasoning|Strong multilingual model with excellent reasoning"
    ["mistral-small:22b"]="13.2|general,balanced,multilingual|Excellent balanced performance model"
    
    # Code-Focused Models
    ["qwen2.5-coder:32b"]="19.1|code,programming,architecture|Large code model for complex projects"
    ["deepseek-coder:6.7b"]="3.8|code,programming,documentation|Specialized programming model"
    
    # Vision/Multimodal
    ["llava:13b"]="7.3|vision,image-understanding,multimodal|Image understanding and visual reasoning"
    ["qwen2-vl:7b"]="4.2|vision,image-understanding,multimodal|Vision-language model for image analysis"
    
    # Legacy Models (for reference)
    ["llama2:7b"]="3.8|general,legacy|Legacy model, superseded by llama3.1"
    ["codellama:7b"]="3.8|code,legacy|Legacy code model, superseded by qwen2.5-coder"
)

# Default models to install (updated for 2025)
readonly DEFAULT_MODELS=(
    "llama3.1:8b"      # General purpose - proven and reliable
    "deepseek-r1:8b"   # Advanced reasoning - breakthrough model for complex thinking
    "qwen2.5-coder:7b" # Modern code generation - superior to CodeLlama
)

#######################################
# Export configuration variables
#######################################
ollama::export_config() {
    export OLLAMA_VERSION OLLAMA_PORT OLLAMA_BASE_URL OLLAMA_SERVICE_NAME
    export OLLAMA_INSTALL_DIR OLLAMA_USER
    export OLLAMA_NUM_PARALLEL OLLAMA_MAX_LOADED_MODELS OLLAMA_FLASH_ATTENTION OLLAMA_ORIGINS
    # Note: Arrays MODEL_CATALOG and DEFAULT_MODELS are already available to sourcing scripts
}

# Export function for subshell availability
export -f ollama::export_config