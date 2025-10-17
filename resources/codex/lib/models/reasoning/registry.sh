#!/usr/bin/env bash
################################################################################
# Reasoning Models Registry
# 
# Registry of models specialized for reasoning and complex problem solving
################################################################################

#######################################
# Get all reasoning models
# Returns:
#   JSON array of reasoning models with their configurations
#######################################
reasoning_registry::get_models() {
    cat << 'EOF'
[
  {
    "model_name": "o1-mini",
    "category": "reasoning",
    "api_endpoint": "completions",
    "supports_functions": false,
    "cost_per_1m_tokens": 3.00,
    "output_cost_per_1m_tokens": 12.00,
    "performance_score": 8,
    "reasoning_score": 10,
    "context_window": 128000,
    "max_output": 65536,
    "description": "Cost-efficient reasoning model with chain-of-thought",
    "strengths": ["deep_reasoning", "mathematical_proofs", "step_by_step"],
    "best_for": "reasoning_tasks"
  },
  {
    "model_name": "o1-preview", 
    "category": "reasoning",
    "api_endpoint": "completions",
    "supports_functions": false,
    "cost_per_1m_tokens": 15.00,
    "output_cost_per_1m_tokens": 60.00,
    "performance_score": 9,
    "reasoning_score": 10,
    "context_window": 128000,
    "max_output": 32768,
    "description": "Advanced reasoning model for hardest problems",
    "strengths": ["complex_reasoning", "problem_solving", "logic"],
    "best_for": "complex_reasoning_tasks"
  },
  {
    "model_name": "gpt-5",
    "category": "reasoning",
    "api_endpoint": "completions",
    "supports_functions": true,
    "cost_per_1m_tokens": 1.25,
    "output_cost_per_1m_tokens": 10.00,
    "performance_score": 10,
    "reasoning_score": 9,
    "context_window": 400000,
    "max_output": 128000,
    "description": "General model with strong reasoning capabilities",
    "strengths": ["balanced_reasoning", "versatile", "function_calling"],
    "best_for": "general_reasoning_tasks"
  }
]
EOF
}

#######################################
# Get best reasoning model for specific criteria
# Arguments:
#   $1 - Optimization criteria (cost, reasoning, balanced)
# Returns:
#   JSON model configuration
#######################################
reasoning_registry::get_best_model() {
    local criteria="${1:-reasoning}"
    local models=$(reasoning_registry::get_models)
    
    case "$criteria" in
        cost)
            echo "$models" | jq 'min_by(.cost_per_1m_tokens)'
            ;;
        reasoning)
            echo "$models" | jq 'max_by(.reasoning_score)'
            ;;
        performance)
            echo "$models" | jq 'max_by(.performance_score)'
            ;;
        balanced)
            echo "$models" | jq 'max_by(.reasoning_score - (.cost_per_1m_tokens * 0.5))'
            ;;
        functions)
            # Get best model that supports functions
            echo "$models" | jq 'map(select(.supports_functions == true)) | max_by(.reasoning_score)'
            ;;
        *)
            echo "$models" | jq '.[0]'
            ;;
    esac
}

#######################################
# Get reasoning models by type
# Arguments:
#   $1 - Model type (o1, gpt, all)
# Returns:
#   JSON array of matching models
#######################################
reasoning_registry::get_models_by_type() {
    local model_type="$1"
    local models=$(reasoning_registry::get_models)
    
    case "$model_type" in
        o1)
            echo "$models" | jq '[.[] | select(.model_name | startswith("o1"))]'
            ;;
        gpt)
            echo "$models" | jq '[.[] | select(.model_name | startswith("gpt"))]'
            ;;
        specialized)
            echo "$models" | jq '[.[] | select(.model_name | startswith("o1"))]'
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
# Get models by reasoning complexity
# Arguments:
#   $1 - Complexity level (simple, medium, complex, extreme)
# Returns:
#   JSON model configuration
#######################################
reasoning_registry::get_model_by_complexity() {
    local complexity="$1"
    local models=$(reasoning_registry::get_models)
    
    case "$complexity" in
        simple|basic)
            # Use cheapest reasoning model for simple problems
            echo "$models" | jq 'min_by(.cost_per_1m_tokens)'
            ;;
        medium)
            # Use balanced approach
            echo "$models" | jq 'max_by(.reasoning_score - (.cost_per_1m_tokens * 0.5))'
            ;;
        complex|hard)
            # Use best reasoning model regardless of cost
            echo "$models" | jq 'max_by(.reasoning_score)'
            ;;
        extreme|research)
            # Use the most advanced model available
            echo "$models" | jq '.[] | select(.model_name == "o1-preview")'
            ;;
        *)
            # Default to balanced
            reasoning_registry::get_best_model "balanced"
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
reasoning_registry::get_model() {
    local model_name="$1"
    local models=$(reasoning_registry::get_models)
    
    echo "$models" | jq --arg name "$model_name" '.[] | select(.model_name == $name)'
}

#######################################
# Get recommended model for reasoning task type
# Arguments:
#   $1 - Task type (math, logic, proof, analysis, planning, debugging)
# Returns:
#   JSON model configuration
#######################################
reasoning_registry::get_recommended_model() {
    local task_type="$1"
    local models=$(reasoning_registry::get_models)
    
    case "$task_type" in
        math|mathematical|proof|theorem)
            # Use specialized reasoning model for math
            echo "$models" | jq '.[] | select(.model_name | startswith("o1"))'
            ;;
        logic|logical|deduction|inference)
            # Use best reasoning model for logic
            echo "$models" | jq 'max_by(.reasoning_score)'
            ;;
        analysis|complex_analysis|research)
            # Use most advanced model for analysis
            echo "$models" | jq '.[] | select(.model_name == "o1-preview")'
            ;;
        planning|strategy|problem_solving)
            # Use balanced reasoning model
            echo "$models" | jq 'max_by(.reasoning_score - (.cost_per_1m_tokens * 0.5))'
            ;;
        debugging|troubleshooting)
            # Prefer model with functions if available
            local func_model=$(echo "$models" | jq 'map(select(.supports_functions == true)) | max_by(.reasoning_score)')
            if [[ "$func_model" != "null" && -n "$func_model" ]]; then
                echo "$func_model"
            else
                echo "$models" | jq 'max_by(.reasoning_score)'
            fi
            ;;
        simple|basic|explain)
            # Use cost-effective model for simple reasoning
            echo "$models" | jq 'min_by(.cost_per_1m_tokens)'
            ;;
        *)
            # Default to best reasoning
            reasoning_registry::get_best_model "reasoning"
            ;;
    esac
}

#######################################
# Check if reasoning model supports function calling
# Arguments:
#   $1 - Model name
# Returns:
#   0 if supports functions, 1 if not
#######################################
reasoning_registry::supports_functions() {
    local model_name="$1"
    local model=$(reasoning_registry::get_model "$model_name")
    
    if [[ "$model" != "null" && -n "$model" ]]; then
        local supports=$(echo "$model" | jq -r '.supports_functions')
        if [[ "$supports" == "true" ]]; then
            return 0
        fi
    fi
    
    return 1
}

#######################################
# Get reasoning models that support function calling
# Returns:
#   JSON array of models with function support
#######################################
reasoning_registry::get_function_capable_models() {
    local models=$(reasoning_registry::get_models)
    echo "$models" | jq '[.[] | select(.supports_functions == true)]'
}

#######################################
# Get cost estimate for reasoning task
# Arguments:
#   $1 - Model name
#   $2 - Estimated input tokens (reasoning uses more)
#   $3 - Estimated output tokens
# Returns:
#   Cost estimate in USD
#######################################
reasoning_registry::estimate_cost() {
    local model_name="$1"
    local input_tokens="${2:-2000}"
    local output_tokens="${3:-1000}"
    
    local model=$(reasoning_registry::get_model "$model_name")
    if [[ "$model" == "null" || -z "$model" ]]; then
        echo "0.010"
        return
    fi
    
    local input_cost=$(echo "$model" | jq -r '.cost_per_1m_tokens')
    local output_cost=$(echo "$model" | jq -r '.output_cost_per_1m_tokens')
    
    # Reasoning models often use more tokens due to chain-of-thought
    local reasoning_multiplier=1.5
    input_tokens=$(echo "$input_tokens * $reasoning_multiplier" | bc -l 2>/dev/null || echo "$input_tokens")
    
    # Calculate cost
    local total_cost
    total_cost=$(echo "scale=6; ($input_tokens / 1000000) * $input_cost + ($output_tokens / 1000000) * $output_cost" | bc -l 2>/dev/null || echo "0.010")
    
    echo "$total_cost"
}

#######################################
# Compare reasoning models by effectiveness
# Returns:
#   JSON array sorted by reasoning effectiveness per dollar
#######################################
reasoning_registry::get_models_by_reasoning_value() {
    local models=$(reasoning_registry::get_models)
    
    # Calculate reasoning value: reasoning_score / cost
    echo "$models" | jq 'map(. + {"reasoning_value": (.reasoning_score / .cost_per_1m_tokens)}) | sort_by(-.reasoning_value)'
}

# Export functions
export -f reasoning_registry::get_models
export -f reasoning_registry::get_best_model
export -f reasoning_registry::get_model
export -f reasoning_registry::supports_functions
export -f reasoning_registry::get_recommended_model