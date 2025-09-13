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
    "model_name": "gpt-5-nano",
    "category": "general",
    "api_endpoint": "completions",
    "supports_functions": true,
    "cost_per_1m_tokens": 0.05,
    "output_cost_per_1m_tokens": 0.40,
    "performance_score": 7,
    "reasoning_score": 6,
    "context_window": 400000,
    "max_output": 128000,
    "description": "Most cost-effective GPT-5 model",
    "strengths": ["cost_efficiency", "general_tasks", "basic_reasoning"],
    "best_for": "simple_tasks"
  },
  {
    "model_name": "gpt-5-mini",
    "category": "general",
    "api_endpoint": "completions",
    "supports_functions": true,
    "cost_per_1m_tokens": 0.25,
    "output_cost_per_1m_tokens": 2.00,
    "performance_score": 8,
    "reasoning_score": 7,
    "context_window": 400000,
    "max_output": 128000,
    "description": "Balanced GPT-5 model for most tasks",
    "strengths": ["balanced_performance", "versatile", "good_reasoning"],
    "best_for": "medium_tasks"
  },
  {
    "model_name": "gpt-5",
    "category": "general",
    "api_endpoint": "completions", 
    "supports_functions": true,
    "cost_per_1m_tokens": 1.25,
    "output_cost_per_1m_tokens": 10.00,
    "performance_score": 10,
    "reasoning_score": 9,
    "context_window": 400000,
    "max_output": 128000,
    "description": "Flagship GPT-5 model with best performance",
    "strengths": ["best_performance", "complex_reasoning", "versatile"],
    "best_for": "complex_tasks"
  },
  {
    "model_name": "gpt-4o-mini",
    "category": "general",
    "api_endpoint": "completions",
    "supports_functions": true,
    "cost_per_1m_tokens": 0.15,
    "output_cost_per_1m_tokens": 0.60,
    "performance_score": 6,
    "reasoning_score": 6,
    "context_window": 128000,
    "max_output": 16000,
    "description": "Cost-efficient GPT-4o variant",
    "strengths": ["cost_efficiency", "reliable", "established"],
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
    "max_output": 16000,
    "description": "Previous generation flagship model",
    "strengths": ["proven_performance", "reliable", "multimodal"],
    "best_for": "complex_tasks"
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
        legacy)
            # Prefer GPT-4o for established reliability
            echo "$models" | jq '.[] | select(.model_name | startswith("gpt-4o"))'
            ;;
        latest)
            # Prefer GPT-5 series for latest features
            echo "$models" | jq '.[] | select(.model_name | startswith("gpt-5"))'
            ;;
        *)
            echo "$models" | jq '.[0]'
            ;;
    esac
}

#######################################
# Get general models by generation
# Arguments:
#   $1 - Generation (gpt-4, gpt-5, all)
# Returns:
#   JSON array of matching models
#######################################
general_registry::get_models_by_generation() {
    local generation="$1"
    local models=$(general_registry::get_models)
    
    case "$generation" in
        gpt-4)
            echo "$models" | jq '[.[] | select(.model_name | startswith("gpt-4"))]'
            ;;
        gpt-5)
            echo "$models" | jq '[.[] | select(.model_name | startswith("gpt-5"))]'
            ;;
        latest)
            echo "$models" | jq '[.[] | select(.model_name | startswith("gpt-5"))]'
            ;;
        all)
            echo "$models"
            ;;
        *)
            echo "$models"
            ;;
    esac
}

#######################################
# Get models by capability
# Arguments:
#   $1 - Required capability (functions, high_context, low_cost)
# Returns:
#   JSON array of matching models  
#######################################
general_registry::get_models_by_capability() {
    local capability="$1"
    local models=$(general_registry::get_models)
    
    case "$capability" in
        functions)
            echo "$models" | jq '[.[] | select(.supports_functions == true)]'
            ;;
        high_context)
            echo "$models" | jq '[.[] | select(.context_window >= 300000)]'
            ;;
        low_cost)
            echo "$models" | jq '[.[] | select(.cost_per_1m_tokens <= 0.5)]'
            ;;
        high_performance)
            echo "$models" | jq '[.[] | select(.performance_score >= 8)]'
            ;;
        reasoning)
            echo "$models" | jq '[.[] | select(.reasoning_score >= 7)]'
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
general_registry::get_model() {
    local model_name="$1"
    local models=$(general_registry::get_models)
    
    echo "$models" | jq --arg name "$model_name" '.[] | select(.model_name == $name)'
}

#######################################
# Get recommended model for task type
# Arguments:
#   $1 - Task type (writing, analysis, conversation, creative, technical)
# Returns:
#   JSON model configuration
#######################################
general_registry::get_recommended_model() {
    local task_type="$1"
    local models=$(general_registry::get_models)
    
    case "$task_type" in
        writing|creative|content)
            # Use balanced model for creative tasks
            echo "$models" | jq 'max_by(.performance_score - (.cost_per_1m_tokens * 2))'
            ;;
        analysis|technical|research)
            # Use high-performance model for analysis
            echo "$models" | jq 'max_by(.reasoning_score)'
            ;;
        conversation|chat|simple)
            # Use cost-effective model for simple tasks
            echo "$models" | jq 'min_by(.cost_per_1m_tokens)'
            ;;
        enterprise|production|critical)
            # Use best performance for critical tasks
            echo "$models" | jq 'max_by(.performance_score)'
            ;;
        *)
            # Default to balanced
            general_registry::get_best_model "balanced"
            ;;
    esac
}

#######################################
# Compare models by performance vs cost
# Returns:
#   JSON array of models sorted by value score
#######################################
general_registry::get_models_by_value() {
    local models=$(general_registry::get_models)
    
    # Calculate value score: performance / cost
    echo "$models" | jq 'map(. + {"value_score": (.performance_score / .cost_per_1m_tokens)}) | sort_by(-.value_score)'
}

#######################################
# Get cost estimate for general task
# Arguments:
#   $1 - Model name
#   $2 - Estimated input tokens
#   $3 - Estimated output tokens
# Returns:
#   Cost estimate in USD
#######################################
general_registry::estimate_cost() {
    local model_name="$1"
    local input_tokens="${2:-500}"
    local output_tokens="${3:-1000}"
    
    local model=$(general_registry::get_model "$model_name")
    if [[ "$model" == "null" || -z "$model" ]]; then
        echo "0.001"
        return
    fi
    
    local input_cost=$(echo "$model" | jq -r '.cost_per_1m_tokens')
    local output_cost=$(echo "$model" | jq -r '.output_cost_per_1m_tokens')
    
    # Calculate cost
    local total_cost
    total_cost=$(echo "scale=6; ($input_tokens / 1000000) * $input_cost + ($output_tokens / 1000000) * $output_cost" | bc -l 2>/dev/null || echo "0.001")
    
    echo "$total_cost"
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