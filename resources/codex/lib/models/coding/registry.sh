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
    "model_name": "codex-mini-latest",
    "category": "coding",
    "api_endpoint": "responses",
    "supports_functions": true,
    "cost_per_1m_tokens": 1.50,
    "output_cost_per_1m_tokens": 6.00,
    "performance_score": 9,
    "reasoning_score": 7,
    "context_window": 200000,
    "max_output": 100000,
    "description": "Optimized coding model, fine-tuned o4-mini for Codex CLI",
    "strengths": ["code_generation", "debugging", "code_completion"],
    "best_for": "complex_coding_tasks"
  },
  {
    "model_name": "gpt-5-nano",
    "category": "coding",
    "api_endpoint": "completions",
    "supports_functions": true,
    "cost_per_1m_tokens": 0.05,
    "output_cost_per_1m_tokens": 0.40,
    "performance_score": 7,
    "reasoning_score": 6,
    "context_window": 400000,
    "max_output": 128000,
    "description": "Cheapest GPT-5 model, good for simple coding tasks",
    "strengths": ["cost_efficiency", "basic_coding", "simple_functions"],
    "best_for": "simple_coding_tasks"
  },
  {
    "model_name": "gpt-5-mini",
    "category": "coding",
    "api_endpoint": "completions",
    "supports_functions": true,
    "cost_per_1m_tokens": 0.25,
    "output_cost_per_1m_tokens": 2.00,
    "performance_score": 8,
    "reasoning_score": 7,
    "context_window": 400000,
    "max_output": 128000,
    "description": "Balanced GPT-5 model for coding tasks",
    "strengths": ["balanced_performance", "good_coding", "function_calling"],
    "best_for": "medium_coding_tasks"
  },
  {
    "model_name": "gpt-5",
    "category": "coding", 
    "api_endpoint": "completions",
    "supports_functions": true,
    "cost_per_1m_tokens": 1.25,
    "output_cost_per_1m_tokens": 10.00,
    "performance_score": 10,
    "reasoning_score": 9,
    "context_window": 400000,
    "max_output": 128000,
    "description": "Flagship GPT-5 model with excellent coding capabilities",
    "strengths": ["best_coding", "complex_algorithms", "architecture_design"],
    "best_for": "complex_coding_tasks"
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
# Get coding models by capability
# Arguments:
#   $1 - Required capability (functions, high_context, reasoning)
# Returns:
#   JSON array of matching models
#######################################
coding_registry::get_models_by_capability() {
    local capability="$1"
    local models=$(coding_registry::get_models)
    
    case "$capability" in
        functions)
            echo "$models" | jq '[.[] | select(.supports_functions == true)]'
            ;;
        high_context)
            echo "$models" | jq '[.[] | select(.context_window >= 300000)]'
            ;;
        reasoning)
            echo "$models" | jq '[.[] | select(.reasoning_score >= 7)]'
            ;;
        low_cost)
            echo "$models" | jq '[.[] | select(.cost_per_1m_tokens <= 0.5)]'
            ;;
        *)
            echo "$models"
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

#######################################
# Check if model is available
# Arguments:
#   $1 - Model name
# Returns:
#   0 if available, 1 if not
#######################################
coding_registry::is_available() {
    local model_name="$1"
    local model=$(coding_registry::get_model "$model_name")
    
    if [[ "$model" != "null" && -n "$model" ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Get recommended model for task type
# Arguments:
#   $1 - Task type (simple, medium, complex, debugging, refactoring)
# Returns:
#   JSON model configuration
#######################################
coding_registry::get_recommended_model() {
    local task_type="$1"
    local models=$(coding_registry::get_models)
    
    case "$task_type" in
        simple|basic|hello_world|snippet)
            # Use cheapest model for simple tasks
            echo "$models" | jq 'min_by(.cost_per_1m_tokens)'
            ;;
        debugging|analysis|review)
            # Use high reasoning model for debugging
            echo "$models" | jq 'max_by(.reasoning_score)'
            ;;
        complex|architecture|enterprise|production)
            # Use best performance model for complex tasks
            echo "$models" | jq 'max_by(.performance_score)'
            ;;
        medium|application|project|refactoring)
            # Use balanced model for medium tasks
            echo "$models" | jq 'max_by(.performance_score - (.cost_per_1m_tokens * 4))'
            ;;
        *)
            # Default to balanced
            coding_registry::get_best_model "balanced"
            ;;
    esac
}

#######################################
# Get cost estimate for coding task
# Arguments:
#   $1 - Model name
#   $2 - Estimated input tokens
#   $3 - Estimated output tokens
# Returns:
#   Cost estimate in USD
#######################################
coding_registry::estimate_cost() {
    local model_name="$1"
    local input_tokens="${2:-1000}"
    local output_tokens="${3:-2000}"
    
    local model=$(coding_registry::get_model "$model_name")
    if [[ "$model" == "null" || -z "$model" ]]; then
        echo "0.001"
        return
    fi
    
    local input_cost=$(echo "$model" | jq -r '.cost_per_1m_tokens')
    local output_cost=$(echo "$model" | jq -r '.output_cost_per_1m_tokens')
    
    # Calculate: (tokens / 1,000,000) * cost_per_million
    local total_cost
    total_cost=$(echo "scale=6; ($input_tokens / 1000000) * $input_cost + ($output_tokens / 1000000) * $output_cost" | bc -l 2>/dev/null || echo "0.001")
    
    echo "$total_cost"
}

# Export functions
export -f coding_registry::get_models
export -f coding_registry::get_best_model
export -f coding_registry::get_model