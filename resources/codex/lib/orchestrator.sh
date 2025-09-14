#!/usr/bin/env bash
################################################################################
# Codex Orchestrator - Central Coordination System
# 
# The orchestrator analyzes requests and coordinates between:
# - Execution contexts (CLI, sandbox, text-only)
# - Model selection (coding, general, reasoning)
# - API endpoints (responses, completions)
# - Capabilities (text-generation, function-calling, reasoning)
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CODEX_LIB_DIR="${APP_ROOT}/resources/codex/lib"

# Source dependencies
source "${CODEX_LIB_DIR}/common.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Request Analysis
################################################################################

#######################################
# Analyze request to determine required capability
# Arguments:
#   $1 - User request/prompt
# Returns:
#   Capability type: "text-generation", "function-calling", "reasoning"
#######################################
orchestrator::analyze_request() {
    local request="$1"
    
    # Function calling indicators (case insensitive)
    local lower_request=$(echo "$request" | tr '[:upper:]' '[:lower:]')
    if [[ "$lower_request" =~ (create|build|make|write|generate|fix|refactor|run) ]]; then
        if [[ "$lower_request" =~ (file|test|app|project|script|command|execute|run|code|program|function|hello.*world) ]]; then
            echo "function-calling"
            return
        fi
    fi
    
    # Reasoning indicators
    if [[ "$lower_request" =~ (solve|prove|analyze|explain.*complex|step.*by.*step|reasoning|logic) ]]; then
        echo "reasoning"
        return
    fi
    
    # Default to text generation
    echo "text-generation"
}

#######################################
# Determine execution preference based on request
# Arguments:
#   $1 - Capability type
#   $2 - User request
# Returns:
#   Preference order: "cli sandbox text"
#######################################
orchestrator::execution_preference() {
    local capability="$1"
    local request="$2"
    
    case "$capability" in
        function-calling)
            # For function calling, prefer CLI then direct system, then sandbox for testing
            echo "cli direct sandbox text"
            ;;
        text-generation)
            # For text generation, prefer text-only (cheapest)
            echo "text direct sandbox cli"
            ;;
        reasoning)
            # For reasoning, prefer text-only with reasoning models
            echo "text cli direct sandbox"
            ;;
        *)
            echo "text direct sandbox cli"
            ;;
    esac
}

################################################################################
# Component Discovery and Selection
################################################################################

#######################################
# Check availability of execution contexts
# Returns:
#   Space-separated list of available contexts
#######################################
orchestrator::available_contexts() {
    local contexts=""
    
    # Check CLI availability
    if [[ -f "${CODEX_LIB_DIR}/execution-contexts/external-cli.sh" ]]; then
        source "${CODEX_LIB_DIR}/execution-contexts/external-cli.sh"
        if type -t cli_context::is_available &>/dev/null && cli_context::is_available; then
            contexts="$contexts cli"
        fi
    fi
    
    # Check direct system availability (prioritized for real work)
    if [[ -f "${CODEX_LIB_DIR}/execution-contexts/direct-system.sh" ]]; then
        source "${CODEX_LIB_DIR}/execution-contexts/direct-system.sh"
        if type -t direct_system_context::is_available &>/dev/null && direct_system_context::is_available; then
            contexts="$contexts direct"
        fi
    fi
    
    # Check sandbox availability
    if [[ -f "${CODEX_LIB_DIR}/execution-contexts/internal-sandbox.sh" ]]; then
        source "${CODEX_LIB_DIR}/execution-contexts/internal-sandbox.sh"
        if type -t sandbox_context::is_available &>/dev/null && sandbox_context::is_available; then
            contexts="$contexts sandbox"
        fi
    fi
    
    # Text-only is always available
    if [[ -f "${CODEX_LIB_DIR}/execution-contexts/text-only.sh" ]]; then
        contexts="$contexts text"
    fi
    
    echo "$contexts"
}

#######################################
# Select best execution context
# Arguments:
#   $1 - Preferred order (e.g., "cli sandbox text")
# Returns:
#   Selected context name
#######################################
orchestrator::select_context() {
    local preference="$1"
    local available=$(orchestrator::available_contexts)
    
    # Find first preferred context that's available
    for preferred in $preference; do
        for avail in $available; do
            if [[ "$preferred" == "$avail" ]]; then
                echo "$preferred"
                return
            fi
        done
    done
    
    # Fallback to first available
    set -- $available
    echo "${1:-text}"
}

#######################################
# Select optimal model for capability and context
# Arguments:
#   $1 - Capability type
#   $2 - Execution context
#   $3 - User request (for analysis)
# Returns:
#   Model configuration as JSON
#######################################
orchestrator::select_model() {
    local capability="$1"
    local context="$2" 
    local request="$3"
    
    # Source model selector
    source "${CODEX_LIB_DIR}/models/model-selector.sh"
    
    # Get model recommendation
    model_selector::get_best_model "$capability" "$context" "$request"
}

################################################################################
# Orchestration Engine
################################################################################

#######################################
# Execute a request through the orchestration system
# Arguments:
#   $1 - User request/prompt
#   $2 - Optional: Force execution context
#   $3 - Optional: Force model
# Returns:
#   0 on success, 1 on failure
# Outputs:
#   Execution result
#######################################
orchestrator::execute() {
    local request="$1"
    local force_context="${2:-}"
    local force_model="${3:-}"
    
    if [[ -z "$request" ]]; then
        log::error "No request provided to orchestrator"
        return 1
    fi
    
    log::debug "Orchestrator analyzing request: $request"
    
    # 1. Analyze request to determine capability
    local capability=$(orchestrator::analyze_request "$request")
    log::debug "Required capability: $capability"
    
    # 2. Determine execution context
    local context
    if [[ -n "$force_context" ]]; then
        context="$force_context"
        log::debug "Using forced context: $context"
    else
        local preference=$(orchestrator::execution_preference "$capability" "$request")
        context=$(orchestrator::select_context "$preference")
        log::debug "Selected context: $context (preference: $preference)"
    fi
    
    # 3. Select model
    local model_config
    if [[ -n "$force_model" ]]; then
        # TODO: Generate config for forced model
        model_config="{\"model_name\":\"$force_model\",\"api_endpoint\":\"completions\"}"
    else
        model_config=$(orchestrator::select_model "$capability" "$context" "$request")
    fi
    
    log::debug "Model config: $model_config"
    
    # 4. Load execution context
    local context_file="${CODEX_LIB_DIR}/execution-contexts/${context}-context.sh"
    if [[ "$context" == "cli" ]]; then
        context_file="${CODEX_LIB_DIR}/execution-contexts/external-cli.sh"
    elif [[ "$context" == "direct" ]]; then
        context_file="${CODEX_LIB_DIR}/execution-contexts/direct-system.sh"
    elif [[ "$context" == "sandbox" ]]; then
        context_file="${CODEX_LIB_DIR}/execution-contexts/internal-sandbox.sh"
    elif [[ "$context" == "text" ]]; then
        context_file="${CODEX_LIB_DIR}/execution-contexts/text-only.sh"
    fi
    
    if [[ ! -f "$context_file" ]]; then
        log::error "Context implementation not found: $context_file"
        return 1
    fi
    
    source "$context_file"
    
    # 5. Load capability
    local capability_file="${CODEX_LIB_DIR}/capabilities/${capability}.sh"
    if [[ ! -f "$capability_file" ]]; then
        log::error "Capability implementation not found: $capability_file"
        return 1
    fi
    
    source "$capability_file"
    
    # 6. Execute through context
    local context_function
    case "$context" in
        cli) context_function="cli_context::execute" ;;
        direct) context_function="direct_system_context::execute" ;;
        sandbox) context_function="sandbox_context::execute" ;;
        text) context_function="text_context::execute" ;;
    esac
    
    if ! type -t "$context_function" &>/dev/null; then
        log::error "Context function not found: $context_function"
        return 1
    fi
    
    log::info "Executing via $context context with $capability capability"
    
    # Execute with context, capability, model config, and request
    "$context_function" "$capability" "$model_config" "$request"
}

#######################################
# Get orchestrator status
# Returns:
#   JSON status of all components
#######################################
orchestrator::status() {
    local available_contexts=$(orchestrator::available_contexts)
    local context_count=$(echo "$available_contexts" | wc -w)
    
    # Check component health
    local models_healthy="true"
    local apis_healthy="true" 
    local capabilities_healthy="true"
    local tools_healthy="true"
    
    # Basic health checks
    if [[ ! -d "${CODEX_LIB_DIR}/models" ]]; then models_healthy="false"; fi
    if [[ ! -d "${CODEX_LIB_DIR}/apis" ]]; then apis_healthy="false"; fi
    if [[ ! -d "${CODEX_LIB_DIR}/capabilities" ]]; then capabilities_healthy="false"; fi
    if [[ ! -d "${CODEX_LIB_DIR}/tools" ]]; then tools_healthy="false"; fi
    
    cat << EOF
{
  "orchestrator": {
    "status": "healthy",
    "available_contexts": "$available_contexts",
    "context_count": $context_count
  },
  "components": {
    "models": "$models_healthy",
    "apis": "$apis_healthy",
    "capabilities": "$capabilities_healthy", 
    "tools": "$tools_healthy"
  }
}
EOF
}

#######################################
# Initialize orchestrator system
# Returns:
#   0 on success, 1 on failure
#######################################
orchestrator::init() {
    log::debug "Initializing orchestrator system"
    
    # Verify directory structure
    local required_dirs=(
        "execution-contexts"
        "models" 
        "apis"
        "capabilities"
        "tools"
        "workspace"
    )
    
    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "${CODEX_LIB_DIR}/$dir" ]]; then
            log::error "Required directory missing: $dir"
            return 1
        fi
    done
    
    log::debug "Orchestrator system initialized successfully"
    return 0
}

# Initialize on source
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    orchestrator::init
fi

################################################################################
# CLI Interface (when run directly)
################################################################################

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        execute)
            shift
            orchestrator::execute "$@"
            ;;
        status)
            orchestrator::status
            ;;
        analyze)
            shift
            orchestrator::analyze_request "$*"
            ;;
        contexts)
            orchestrator::available_contexts
            ;;
        *)
            echo "Usage: $0 {execute <request>|status|analyze <request>|contexts}"
            echo ""
            echo "Examples:"
            echo "  $0 execute 'Create a Python function to sort an array'"
            echo "  $0 analyze 'Build a web app with tests'"
            echo "  $0 status"
            exit 1
            ;;
    esac
fi