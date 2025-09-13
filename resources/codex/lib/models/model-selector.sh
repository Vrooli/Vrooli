#!/usr/bin/env bash
################################################################################
# Model Selector - Intelligent Model Selection System
# 
# Selects the optimal model based on:
# - Required capability (text-generation, function-calling, reasoning)
# - Execution context (cli, sandbox, text)
# - User request analysis
# - Cost optimization
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
MODELS_DIR="${APP_ROOT}/resources/codex/lib/models"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Source model registries
source "${MODELS_DIR}/coding/registry.sh" 2>/dev/null || true
source "${MODELS_DIR}/general/registry.sh" 2>/dev/null || true
source "${MODELS_DIR}/reasoning/registry.sh" 2>/dev/null || true

################################################################################
# Model Selection Logic
################################################################################

#######################################
# Get best model for given parameters
# Arguments:
#   $1 - Capability type (text-generation, function-calling, reasoning)
#   $2 - Execution context (cli, sandbox, text)
#   $3 - User request (for analysis)
# Returns:
#   JSON model configuration
#######################################
model_selector::get_best_model() {
    local capability="$1"
    local context="$2"
    local request="$3"
    
    log::debug "Selecting model for capability=$capability, context=$context"
    
    # Analyze request characteristics
    local complexity=$(model_selector::analyze_complexity "$request")
    local language=$(model_selector::detect_language "$request")
    local task_type=$(model_selector::detect_task_type "$request")
    
    log::debug "Request analysis: complexity=$complexity, language=$language, task_type=$task_type"
    
    # Get model recommendations by category
    local coding_models general_models reasoning_models
    
    if type -t coding_registry::get_models &>/dev/null; then
        coding_models=$(coding_registry::get_models)
    else
        coding_models="[]"
    fi
    
    if type -t general_registry::get_models &>/dev/null; then
        general_models=$(general_registry::get_models)
    else
        general_models="[]"
    fi
    
    if type -t reasoning_registry::get_models &>/dev/null; then
        reasoning_models=$(reasoning_registry::get_models)
    else
        reasoning_models="[]"
    fi
    
    # Select based on capability and context
    case "$capability" in
        function-calling)
            model_selector::select_for_function_calling "$context" "$complexity" "$task_type" "$coding_models" "$general_models"
            ;;
        reasoning)
            model_selector::select_for_reasoning "$context" "$complexity" "$reasoning_models" "$general_models"
            ;;
        text-generation)
            model_selector::select_for_text_generation "$context" "$complexity" "$language" "$coding_models" "$general_models"
            ;;
        *)
            log::error "Unknown capability: $capability"
            echo "{\"error\": \"Unknown capability: $capability\"}"
            return 1
            ;;
    esac
}

#######################################
# Select model for function calling tasks
# Arguments:
#   $1 - Execution context
#   $2 - Complexity level
#   $3 - Task type
#   $4 - Coding models JSON
#   $5 - General models JSON
# Returns:
#   JSON model configuration
#######################################
model_selector::select_for_function_calling() {
    local context="$1"
    local complexity="$2"
    local task_type="$3"
    local coding_models="$4"
    local general_models="$5"
    
    # For function calling, prefer coding models
    local preferred_models="$coding_models"
    
    # CLI context gets the best available
    if [[ "$context" == "cli" ]]; then
        # CLI uses its own model selection, return a config that indicates this
        echo '{
            "model_name": "codex-mini-latest",
            "api_endpoint": "external",
            "supports_functions": true,
            "context": "cli",
            "cost_per_1m_tokens": 1.50
        }'
        return
    fi
    
    # For sandbox/text contexts, select based on complexity and task
    case "$complexity" in
        high)
            # High complexity: use best coding model
            model_selector::select_from_models "$preferred_models" "performance" "$context"
            ;;
        medium)
            # Medium complexity: balance performance and cost
            model_selector::select_from_models "$preferred_models" "balanced" "$context"
            ;;
        low)
            # Low complexity: optimize for cost
            model_selector::select_from_models "$preferred_models" "cost" "$context"
            ;;
        *)
            # Default to balanced
            model_selector::select_from_models "$preferred_models" "balanced" "$context"
            ;;
    esac
}

#######################################
# Select model for reasoning tasks
# Arguments:
#   $1 - Execution context
#   $2 - Complexity level
#   $3 - Reasoning models JSON
#   $4 - General models JSON
# Returns:
#   JSON model configuration
#######################################
model_selector::select_for_reasoning() {
    local context="$1"
    local complexity="$2"
    local reasoning_models="$3"
    local general_models="$4"
    
    # For reasoning, prefer reasoning-specific models if available
    local preferred_models="$reasoning_models"
    if [[ $(echo "$preferred_models" | jq 'length') -eq 0 ]]; then
        preferred_models="$general_models"
    fi
    
    # Reasoning tasks generally benefit from higher-end models
    case "$complexity" in
        high)
            model_selector::select_from_models "$preferred_models" "reasoning" "$context"
            ;;
        *)
            model_selector::select_from_models "$preferred_models" "balanced" "$context"
            ;;
    esac
}

#######################################
# Select model for text generation tasks
# Arguments:
#   $1 - Execution context
#   $2 - Complexity level
#   $3 - Language
#   $4 - Coding models JSON
#   $5 - General models JSON
# Returns:
#   JSON model configuration
#######################################
model_selector::select_for_text_generation() {
    local context="$1"
    local complexity="$2"
    local language="$3"
    local coding_models="$4"
    local general_models="$5"
    
    # If it's code-related, prefer coding models
    local preferred_models
    if [[ "$language" =~ ^(python|javascript|go|rust|java|cpp|c|sql|bash)$ ]]; then
        preferred_models="$coding_models"
    else
        preferred_models="$general_models"
    fi
    
    # For text generation, cost is often more important
    case "$complexity" in
        high)
            model_selector::select_from_models "$preferred_models" "performance" "$context"
            ;;
        *)
            model_selector::select_from_models "$preferred_models" "cost" "$context"
            ;;
    esac
}

################################################################################
# Model Selection Helpers
################################################################################

#######################################
# Select from available models based on optimization criteria
# Arguments:
#   $1 - Models JSON array
#   $2 - Optimization criteria (cost, performance, balanced, reasoning)
#   $3 - Execution context
# Returns:
#   JSON model configuration
#######################################
model_selector::select_from_models() {
    local models="$1"
    local criteria="$2"
    local context="$3"
    
    if [[ $(echo "$models" | jq 'length') -eq 0 ]]; then
        # Fallback to default model
        echo '{
            "model_name": "gpt-5-nano",
            "api_endpoint": "completions",
            "supports_functions": false,
            "cost_per_1m_tokens": 0.05,
            "optimization": "fallback"
        }'
        return
    fi
    
    # Filter models by context compatibility
    local compatible_models
    case "$context" in
        sandbox)
            # Sandbox can use both APIs
            compatible_models="$models"
            ;;
        text)
            # Text context prefers completions API
            compatible_models=$(echo "$models" | jq '[.[] | select(.api_endpoint == "completions" or .api_endpoint == "both")]')
            ;;
        *)
            compatible_models="$models"
            ;;
    esac
    
    # Select based on criteria
    local selected_model
    case "$criteria" in
        cost)
            selected_model=$(echo "$compatible_models" | jq 'min_by(.cost_per_1m_tokens)')
            ;;
        performance)
            selected_model=$(echo "$compatible_models" | jq 'max_by(.performance_score // 5)')
            ;;
        reasoning)
            selected_model=$(echo "$compatible_models" | jq 'max_by(.reasoning_score // 5)')
            ;;
        balanced)
            selected_model=$(echo "$compatible_models" | jq 'max_by((.performance_score // 5) - (.cost_per_1m_tokens * 10))')
            ;;
        *)
            selected_model=$(echo "$compatible_models" | jq '.[0]')
            ;;
    esac
    
    if [[ "$selected_model" == "null" || -z "$selected_model" ]]; then
        # Final fallback
        echo '{
            "model_name": "gpt-5-nano",
            "api_endpoint": "completions",
            "supports_functions": false,
            "cost_per_1m_tokens": 0.05,
            "optimization": "emergency_fallback"
        }'
    else
        echo "$selected_model"
    fi
}

################################################################################
# Request Analysis
################################################################################

#######################################
# Analyze request complexity
# Arguments:
#   $1 - User request
# Returns:
#   Complexity level: low, medium, high
#######################################
model_selector::analyze_complexity() {
    local request="$1"
    local complexity="low"
    
    # High complexity indicators
    if [[ "$request" =~ (complex|advanced|enterprise|production|optimize|algorithm|architecture|system|framework|complete|full|comprehensive) ]]; then
        complexity="high"
    # Medium complexity indicators
    elif [[ "$request" =~ (application|app|project|multiple|several|integrate|connect|database|api|test|refactor) ]]; then
        complexity="medium"
    # Low complexity indicators (function, simple, basic, etc.)
    elif [[ "$request" =~ (function|simple|basic|example|demo|hello|quick|small) ]]; then
        complexity="low"
    fi
    
    # Consider length as a factor
    local word_count=$(echo "$request" | wc -w)
    if [[ $word_count -gt 50 ]]; then
        complexity="high"
    elif [[ $word_count -gt 20 ]]; then
        if [[ "$complexity" == "low" ]]; then
            complexity="medium"
        fi
    fi
    
    echo "$complexity"
}

#######################################
# Detect programming language from request
# Arguments:
#   $1 - User request
# Returns:
#   Language identifier or "unknown"
#######################################
model_selector::detect_language() {
    local request="$1"
    
    # Direct language mentions
    if [[ "$request" =~ [Pp]ython ]]; then echo "python"; return; fi
    if [[ "$request" =~ [Jj]ava[Ss]cript|[Jj][Ss] ]]; then echo "javascript"; return; fi
    if [[ "$request" =~ [Gg]o[[:space:]] ]]; then echo "go"; return; fi
    if [[ "$request" =~ [Rr]ust ]]; then echo "rust"; return; fi
    if [[ "$request" =~ [Jj]ava[[:space:]] ]]; then echo "java"; return; fi
    if [[ "$request" =~ [Cc]\+\+|[Cc]pp ]]; then echo "cpp"; return; fi
    if [[ "$request" =~ [Bb]ash|[Ss]hell ]]; then echo "bash"; return; fi
    if [[ "$request" =~ SQL|[Dd]atabase ]]; then echo "sql"; return; fi
    if [[ "$request" =~ [Tt]ype[Ss]cript|[Tt][Ss] ]]; then echo "typescript"; return; fi
    
    # Context clues
    if [[ "$request" =~ (django|flask|pandas|numpy) ]]; then echo "python"; return; fi
    if [[ "$request" =~ (react|node|express|npm) ]]; then echo "javascript"; return; fi
    if [[ "$request" =~ (gin|gorm|goroutine) ]]; then echo "go"; return; fi
    
    echo "unknown"
}

#######################################
# Detect task type from request
# Arguments:
#   $1 - User request
# Returns:
#   Task type: coding, analysis, generation, transformation
#######################################
model_selector::detect_task_type() {
    local request="$1"
    
    # Coding tasks
    if [[ "$request" =~ (write|create|build|implement|develop|code|program|function|class|module) ]]; then
        echo "coding"
        return
    fi
    
    # Analysis tasks
    if [[ "$request" =~ (analyze|review|explain|understand|debug|find|check) ]]; then
        echo "analysis"
        return
    fi
    
    # Transformation tasks
    if [[ "$request" =~ (convert|translate|transform|refactor|migrate|port) ]]; then
        echo "transformation"
        return
    fi
    
    # Default to generation
    echo "generation"
}

################################################################################
# Utility Functions
################################################################################

#######################################
# Get all available models across categories
# Returns:
#   JSON array of all models
#######################################
model_selector::get_all_models() {
    local all_models="[]"
    
    if type -t coding_registry::get_models &>/dev/null; then
        local coding=$(coding_registry::get_models)
        all_models=$(echo "$all_models $coding" | jq -s 'add')
    fi
    
    if type -t general_registry::get_models &>/dev/null; then
        local general=$(general_registry::get_models)
        all_models=$(echo "$all_models $general" | jq -s 'add')
    fi
    
    if type -t reasoning_registry::get_models &>/dev/null; then
        local reasoning=$(reasoning_registry::get_models)
        all_models=$(echo "$all_models $reasoning" | jq -s 'add')
    fi
    
    echo "$all_models"
}

#######################################
# Get model by name
# Arguments:
#   $1 - Model name
# Returns:
#   JSON model configuration or null
#######################################
model_selector::get_model_by_name() {
    local model_name="$1"
    local all_models=$(model_selector::get_all_models)
    
    echo "$all_models" | jq --arg name "$model_name" '.[] | select(.model_name == $name)'
}

# Export functions
export -f model_selector::get_best_model
export -f model_selector::get_all_models
export -f model_selector::get_model_by_name