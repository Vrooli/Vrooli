#!/usr/bin/env bash
################################################################################
# Coding Models Registry
# 
# Registry of models optimized for coding tasks
################################################################################

#######################################
# Get all coding models
# Returns:
#   JSON array of coding models with their configurations
#######################################
coding_registry::get_models() {
    cat << 'EOF'
[
  {
    "model_name": "gpt-4o-mini",
    "category": "coding",
    "api_endpoint": "completions",
    "supports_functions": true,
    "cost_per_1m_tokens": 0.15,
    "output_cost_per_1m_tokens": 0.60,
    "performance_score": 8,
    "reasoning_score": 7,
    "context_window": 128000,
    "max_output": 16384,
    "description": "Cost-effective model good for basic coding tasks",
    "strengths": ["cost_efficiency", "basic_coding", "simple_functions"],
    "best_for": "simple_coding_tasks"
  },
  {
    "model_name": "gpt-4o",
    "category": "coding",
    "api_endpoint": "completions",
    "supports_functions": true,
    "cost_per_1m_tokens": 2.50,
    "output_cost_per_1m_tokens": 10.00,
    "performance_score": 9,
    "reasoning_score": 8,
    "context_window": 128000,
    "max_output": 16384,
    "description": "High-performance model for complex coding tasks",
    "strengths": ["code_generation", "debugging", "complex_reasoning"],
    "best_for": "complex_coding_tasks"
  },
  {
    "model_name": "gpt-4",
    "category": "coding", 
    "api_endpoint": "completions",
    "supports_functions": true,
    "cost_per_1m_tokens": 30.00,
    "output_cost_per_1m_tokens": 60.00,
    "performance_score": 9,
    "reasoning_score": 9,
    "context_window": 8192,
    "max_output": 4096,
    "description": "GPT-4 model with excellent coding and reasoning capabilities",
    "strengths": ["best_reasoning", "complex_algorithms", "architecture_design"],
    "best_for": "complex_reasoning_tasks"
  }
]
EOF
}

#######################################
# Get best coding model for specific criteria
# Arguments:
#   $1 - Optimization criteria (cost, performance, balanced)
# Returns:
#   JSON model configuration
#######################################
coding_registry::get_best_model() {
    local criteria="${1:-balanced}"
    local models=$(coding_registry::get_models)
    
    case "$criteria" in
        cost)
            echo "$models" | jq 'min_by(.cost_per_1m_tokens)'
            ;;
        performance)
            echo "$models" | jq 'max_by(.performance_score)'
            ;;
        balanced)
            echo "$models" | jq 'max_by(.performance_score - (.cost_per_1m_tokens * 4))'
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
coding_registry::get_model() {
    local model_name="$1"
    local models=$(coding_registry::get_models)
    
    echo "$models" | jq --arg name "$model_name" '.[] | select(.model_name == $name)'
}

# Export functions
export -f coding_registry::get_models
export -f coding_registry::get_best_model
export -f coding_registry::get_model