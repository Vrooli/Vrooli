#!/usr/bin/env bash
################################################################################
# General Models Registry
# 
# Registry of general-purpose models for text generation and mixed tasks
################################################################################

#######################################
# Get all general models
# Returns:
#   JSON array of general models with their configurations
#######################################
general_registry::get_models() {
    cat << 'EOF'
[
  {
    "model_name": "gpt-4o-mini",
    "category": "general",
    "api_endpoint": "completions",
    "supports_functions": true,
    "cost_per_1m_tokens": 0.15,
    "output_cost_per_1m_tokens": 0.60,
    "performance_score": 8,
    "reasoning_score": 7,
    "context_window": 128000,
    "max_output": 16384,
    "description": "Cost-effective GPT-4o mini model",
    "strengths": ["cost_efficiency", "general_tasks", "function_calling"],
    "best_for": "simple_tasks"
  },
  {
    "model_name": "gpt-4o",
    "category": "general",
    "api_endpoint": "completions",
    "supports_functions": true,
    "cost_per_1m_tokens": 2.50,
    "output_cost_per_1m_tokens": 10.00,
    "performance_score": 9,
    "reasoning_score": 8,
    "context_window": 128000,
    "max_output": 16384,
    "description": "Flagship GPT-4o model with best performance",
    "strengths": ["high_performance", "complex_reasoning", "multimodal"],
    "best_for": "complex_tasks"
  },
  {
    "model_name": "gpt-4",
    "category": "general",
    "api_endpoint": "completions", 
    "supports_functions": true,
    "cost_per_1m_tokens": 30.00,
    "output_cost_per_1m_tokens": 60.00,
    "performance_score": 9,
    "reasoning_score": 9,
    "context_window": 8192,
    "max_output": 4096,
    "description": "GPT-4 model for complex reasoning",
    "strengths": ["excellent_reasoning", "complex_tasks", "reliable"],
    "best_for": "complex_reasoning"
  }
]
EOF
}

#######################################
# Get best general model for specific criteria
# Arguments:
#   $1 - Optimization criteria (cost, performance, balanced)
# Returns:
#   JSON model configuration
#######################################
general_registry::get_best_model() {
    local criteria="${1:-balanced}"
    local models=$(general_registry::get_models)
    
    case "$criteria" in
        cost)
            echo "$models" | jq 'min_by(.cost_per_1m_tokens)'
            ;;
        performance)
            echo "$models" | jq 'max_by(.performance_score)'
            ;;
        balanced)
            echo "$models" | jq 'max_by(.performance_score - (.cost_per_1m_tokens * 3))'
            ;;
        *)
            echo "$models" | jq '.[0]'
            ;;
    esac
}

#######################################
# Get model configuration by name
# Arguments:
#   $1 - Model name
# Returns:
#   JSON model configuration or null
#######################################
general_registry::get_model() {
    local model_name="$1"
    local models=$(general_registry::get_models)
    
    echo "$models" | jq --arg name "$model_name" '.[] | select(.model_name == $name)'
}

#######################################
# Check if model supports function calling
# Arguments:
#   $1 - Model name
# Returns:
#   0 if supports functions, 1 if not
#######################################
general_registry::supports_functions() {
    local model_name="$1"
    local model=$(general_registry::get_model "$model_name")
    
    if [[ "$model" != "null" && -n "$model" ]]; then
        local supports=$(echo "$model" | jq -r '.supports_functions')
        if [[ "$supports" == "true" ]]; then
            return 0
        fi
    fi
    
    return 1
}

# Export functions
export -f general_registry::get_models
export -f general_registry::get_best_model
export -f general_registry::get_model
export -f general_registry::supports_functions