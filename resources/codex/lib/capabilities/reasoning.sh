#!/usr/bin/env bash
################################################################################
# Reasoning Capability
# 
# Handles advanced reasoning tasks that require step-by-step thinking
# Optimized for o1 models and responses API with reasoning support
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Reasoning Interface
################################################################################

#######################################
# Execute reasoning-intensive task
# Arguments:
#   $1 - Model configuration (JSON)
#   $2 - User request
#   $3 - Context metadata (JSON, optional)
# Returns:
#   Reasoning result with step-by-step breakdown
#######################################
reasoning::execute() {
    local model_config="$1"
    local request="$2"
    local context_meta="${3:-{}}"
    
    log::debug "Executing reasoning capability"
    
    # Extract model configuration
    local model_name api_endpoint
    model_name=$(echo "$model_config" | jq -r '.model_name // "o1-mini"')
    api_endpoint=$(echo "$model_config" | jq -r '.api_endpoint // "responses"')
    
    log::debug "Using model: $model_name, API: $api_endpoint"
    
    # Check if model supports reasoning
    if ! reasoning::supports_reasoning "$model_name"; then
        log::warn "Model $model_name may not be optimized for reasoning, falling back to enhanced text generation"
        source "${APP_ROOT}/resources/codex/lib/capabilities/text-generation.sh"
        text_generation::execute "$model_config" "$request" "$context_meta"
        return
    fi
    
    # Analyze reasoning complexity
    local complexity
    complexity=$(reasoning::analyze_complexity "$request")
    
    log::debug "Reasoning complexity: $complexity"
    
    # Route to appropriate reasoning method
    case "$api_endpoint" in
        responses)
            reasoning::execute_via_responses "$model_config" "$request" "$complexity"
            ;;
        completions)
            reasoning::execute_via_completions "$model_config" "$request" "$complexity"
            ;;
        *)
            log::error "Unknown API endpoint: $api_endpoint"
            return 1
            ;;
    esac
}

################################################################################
# Complexity Analysis
################################################################################

#######################################
# Analyze reasoning complexity of request
# Arguments:
#   $1 - User request
# Returns:
#   Complexity level: simple, medium, complex, advanced
#######################################
reasoning::analyze_complexity() {
    local request="$1"
    
    # Advanced complexity indicators
    if [[ "$request" =~ (proof|theorem|mathematical|algorithm.*complexity|optimization|game.*theory|decision.*tree) ]]; then
        echo "advanced"
        return
    fi
    
    # Complex reasoning indicators
    if [[ "$request" =~ (analyze|compare.*contrast|evaluate.*pros.*cons|multi-step|logic.*puzzle|strategy|planning) ]]; then
        echo "complex"
        return
    fi
    
    # Medium complexity indicators
    if [[ "$request" =~ (explain.*why|cause.*effect|reason|justify|argue|problem.*solving) ]]; then
        echo "medium"
        return
    fi
    
    # Default to simple
    echo "simple"
}

#######################################
# Check if model supports advanced reasoning
# Arguments:
#   $1 - Model name
# Returns:
#   0 if supports reasoning, 1 if not
#######################################
reasoning::supports_reasoning() {
    local model="$1"
    
    # Known reasoning-optimized models
    local reasoning_models=(
        "o1-mini"
        "o1-preview" 
        "codex-mini-latest"
        "gpt-5"
    )
    
    for supported in "${reasoning_models[@]}"; do
        if [[ "$model" == "$supported" ]]; then
            return 0
        fi
    done
    
    return 1
}

################################################################################
# API-Specific Reasoning
################################################################################

#######################################
# Execute reasoning via responses API
# Arguments:
#   $1 - Model configuration (JSON)
#   $2 - User request
#   $3 - Complexity level
# Returns:
#   Reasoning result with chain of thought
#######################################
reasoning::execute_via_responses() {
    local model_config="$1"
    local request="$2" 
    local complexity="$3"
    
    # Source responses API
    source "${APP_ROOT}/resources/codex/lib/apis/responses.sh"
    
    local model_name
    model_name=$(echo "$model_config" | jq -r '.model_name')
    
    # Check API availability
    if ! type -t responses_api::call &>/dev/null; then
        log::error "Responses API not available"
        return 1
    fi
    
    # Set reasoning parameters based on complexity
    local reasoning_effort options
    case "$complexity" in
        advanced)
            reasoning_effort="high"
            options='{"reasoning_effort": "high", "reasoning_summary": true, "max_completion_tokens": 16384}'
            ;;
        complex)
            reasoning_effort="medium"
            options='{"reasoning_effort": "medium", "reasoning_summary": true, "max_completion_tokens": 12288}'
            ;;
        medium)
            reasoning_effort="medium" 
            options='{"reasoning_effort": "medium", "reasoning_summary": true, "max_completion_tokens": 8192}'
            ;;
        simple)
            reasoning_effort="low"
            options='{"reasoning_effort": "low", "reasoning_summary": true, "max_completion_tokens": 4096}'
            ;;
    esac
    
    # Build reasoning prompt
    local system_message
    system_message=$(reasoning::get_system_message "$complexity")
    
    local enhanced_request
    enhanced_request=$(reasoning::enhance_prompt "$request" "$complexity")
    
    # Build messages
    local messages
    messages=$(jq -n \
        --arg system "$system_message" \
        --arg user "$enhanced_request" \
        '[{"role": "system", "content": $system}, {"role": "user", "content": $user}]')
    
    # Make API call with reasoning
    local response
    response=$(responses_api::call "$model_name" "$messages" "$options")
    
    if ! http_client::has_error "$response"; then
        # Extract both reasoning and content
        local reasoning_chain content
        reasoning_chain=$(responses_api::extract_reasoning "$response")
        content=$(responses_api::extract_content "$response")
        
        # Format output with reasoning chain
        if [[ -n "$reasoning_chain" ]]; then
            echo "## Reasoning Process:"
            echo "$reasoning_chain"
            echo ""
            echo "## Final Answer:"
        fi
        
        echo "$content"
    else
        return 1
    fi
}

#######################################
# Execute reasoning via completions API
# Arguments:
#   $1 - Model configuration (JSON)
#   $2 - User request
#   $3 - Complexity level
# Returns:
#   Reasoning result with explicit chain of thought
#######################################
reasoning::execute_via_completions() {
    local model_config="$1"
    local request="$2"
    local complexity="$3"
    
    # Source completions API
    source "${APP_ROOT}/resources/codex/lib/apis/completions.sh"
    
    local model_name
    model_name=$(echo "$model_config" | jq -r '.model_name')
    
    # For completions API, we need to explicitly request chain of thought
    local system_message
    system_message=$(reasoning::get_system_message "$complexity")
    system_message="$system_message Always show your reasoning step by step before giving the final answer."
    
    local enhanced_request
    enhanced_request=$(reasoning::enhance_prompt "$request" "$complexity")
    enhanced_request="$enhanced_request

Please think through this step by step and show your reasoning process."
    
    # Use the text generation function with reasoning-enhanced prompts
    if type -t completions_api::generate_text &>/dev/null; then
        completions_api::generate_text "$model_name" "$enhanced_request" "$system_message"
    else
        log::error "Completions API not available"
        return 1
    fi
}

################################################################################
# Prompt Engineering for Reasoning
################################################################################

#######################################
# Get appropriate system message for reasoning complexity
# Arguments:
#   $1 - Complexity level
# Returns:
#   System message string
#######################################
reasoning::get_system_message() {
    local complexity="$1"
    
    case "$complexity" in
        advanced)
            echo "You are an expert in advanced reasoning, mathematics, and complex problem solving. Break down complex problems into logical steps. Use rigorous analysis and consider multiple perspectives. Show your work clearly."
            ;;
        complex)
            echo "You are an analytical expert who excels at complex reasoning tasks. Think through problems systematically, consider multiple factors, and provide well-reasoned conclusions with clear justification."
            ;;
        medium)
            echo "You are a logical thinker who approaches problems methodically. Break down the problem, analyze the key factors, and provide reasoned conclusions."
            ;;
        simple)
            echo "You are a helpful assistant who thinks clearly and logically. Provide straightforward reasoning for your answers."
            ;;
        *)
            echo "You are an expert reasoning assistant. Think step by step and provide clear, logical explanations."
            ;;
    esac
}

#######################################
# Enhance user prompt for better reasoning
# Arguments:
#   $1 - Original request
#   $2 - Complexity level
# Returns:
#   Enhanced prompt
#######################################
reasoning::enhance_prompt() {
    local request="$1"
    local complexity="$2"
    
    case "$complexity" in
        advanced)
            echo "$request

Please approach this with rigorous analysis. Consider:
1. What are the key assumptions?
2. What are the logical steps needed?
3. Are there any edge cases or special conditions?
4. What would be alternative approaches?
5. How confident are you in the conclusion?"
            ;;
        complex)
            echo "$request

Please analyze this systematically:
1. Break down the problem into components
2. Consider the relationships between elements
3. Evaluate different perspectives or approaches
4. Provide a well-reasoned conclusion"
            ;;
        medium)
            echo "$request

Please think through this step by step:
1. What is the core question?
2. What factors should be considered?
3. What is the logical reasoning?"
            ;;
        *)
            echo "$request"
            ;;
    esac
}

################################################################################
# Specialized Reasoning Functions
################################################################################

#######################################
# Mathematical reasoning and proofs
# Arguments:
#   $1 - Model configuration (JSON)
#   $2 - Mathematical problem or proof request
# Returns:
#   Step-by-step mathematical reasoning
#######################################
reasoning::mathematical() {
    local model_config="$1"
    local problem="$2"
    
    local enhanced_config
    enhanced_config=$(echo "$model_config" | jq '. + {"complexity": "advanced"}')
    
    local math_prompt="Solve this mathematical problem with complete step-by-step reasoning:

$problem

Show all work, explain each step, and verify the solution."
    
    reasoning::execute "$enhanced_config" "$math_prompt"
}

#######################################
# Logical reasoning and puzzles
# Arguments:
#   $1 - Model configuration (JSON)
#   $2 - Logic problem or puzzle
# Returns:
#   Logical reasoning chain
#######################################
reasoning::logical() {
    local model_config="$1"
    local puzzle="$2"
    
    local enhanced_config
    enhanced_config=$(echo "$model_config" | jq '. + {"complexity": "complex"}')
    
    local logic_prompt="Solve this logic problem step by step:

$puzzle

Use clear logical reasoning, identify any assumptions, and work through the problem systematically."
    
    reasoning::execute "$enhanced_config" "$logic_prompt"
}

#######################################
# Strategic analysis and planning
# Arguments:
#   $1 - Model configuration (JSON)
#   $2 - Strategic question or scenario
# Returns:
#   Strategic analysis with reasoning
#######################################
reasoning::strategic() {
    local model_config="$1"
    local scenario="$2"
    
    local enhanced_config
    enhanced_config=$(echo "$model_config" | jq '. + {"complexity": "complex"}')
    
    local strategy_prompt="Analyze this strategic scenario:

$scenario

Consider:
1. Key stakeholders and their interests
2. Available options and their trade-offs
3. Short-term and long-term implications
4. Risk factors and mitigation strategies
5. Recommended approach with justification"
    
    reasoning::execute "$enhanced_config" "$strategy_prompt"
}

################################################################################
# Quality Enhancement
################################################################################

#######################################
# Validate reasoning chain quality
# Arguments:
#   $1 - Reasoning output
#   $2 - Original request
# Returns:
#   0 if quality is good, 1 if needs improvement
#######################################
reasoning::validate_quality() {
    local output="$1"
    local request="$2"
    
    # Check for reasoning indicators
    local reasoning_indicators=(
        "step"
        "because"
        "therefore"
        "since"
        "analysis"
        "conclusion"
        "reasoning"
    )
    
    local found_indicators=0
    for indicator in "${reasoning_indicators[@]}"; do
        if [[ "$output" =~ $indicator ]]; then
            ((found_indicators++))
        fi
    done
    
    # Need at least 2 reasoning indicators for quality reasoning
    if [[ $found_indicators -ge 2 ]]; then
        return 0
    else
        log::debug "Reasoning quality check failed: only $found_indicators indicators found"
        return 1
    fi
}

#######################################
# Get reasoning capability info
# Returns:
#   JSON object with capability details
#######################################
reasoning::get_info() {
    cat << 'EOF'
{
  "name": "reasoning",
  "description": "Advanced reasoning and step-by-step problem solving",
  "complexity_levels": ["simple", "medium", "complex", "advanced"],
  "supported_models": ["o1-mini", "o1-preview", "codex-mini-latest", "gpt-5"],
  "preferred_api": "responses",
  "features": [
    "Chain of thought reasoning",
    "Mathematical proofs", 
    "Logical puzzles",
    "Strategic analysis",
    "Complex problem decomposition"
  ]
}
EOF
}

# Export functions
export -f reasoning::execute
export -f reasoning::analyze_complexity
export -f reasoning::mathematical
export -f reasoning::logical
export -f reasoning::strategic