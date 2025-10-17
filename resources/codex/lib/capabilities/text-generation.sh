#!/usr/bin/env bash
################################################################################
# Text Generation Capability
# 
# Handles pure text generation tasks without tool execution
# Works with any execution context and API endpoint
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Text Generation Interface
################################################################################

#######################################
# Execute text generation
# Arguments:
#   $1 - Model configuration (JSON)
#   $2 - User request
#   $3 - Context metadata (JSON, optional)
# Returns:
#   Generated text
#######################################
text_generation::execute() {
    local model_config="$1"
    local request="$2"
    local context_meta="${3:-{}}"
    
    log::debug "Executing text generation capability"
    
    # Extract model configuration
    local model_name
    local api_endpoint
    model_name=$(echo "$model_config" | jq -r '.model_name // "gpt-5-nano"')
    api_endpoint=$(echo "$model_config" | jq -r '.api_endpoint // "completions"')
    
    log::debug "Using model: $model_name, API: $api_endpoint"
    
    # Analyze request to determine generation type
    local generation_type
    generation_type=$(text_generation::analyze_request "$request")
    
    log::debug "Generation type: $generation_type"
    
    # Route to appropriate API and generation method
    case "$api_endpoint" in
        responses)
            text_generation::generate_via_responses "$model_config" "$request" "$generation_type"
            ;;
        completions)
            text_generation::generate_via_completions "$model_config" "$request" "$generation_type"
            ;;
        *)
            log::error "Unknown API endpoint: $api_endpoint"
            return 1
            ;;
    esac
}

################################################################################
# Request Analysis
################################################################################

#######################################
# Analyze request to determine generation type
# Arguments:
#   $1 - User request
# Returns:
#   Generation type: code, documentation, explanation, conversion, general
#######################################
text_generation::analyze_request() {
    local request="$1"
    
    # Code generation indicators
    if [[ "$request" =~ (write|create|generate|implement|build).*(function|class|module|script|program|code) ]]; then
        echo "code"
        return
    fi
    
    # Documentation indicators
    if [[ "$request" =~ (document|readme|doc|comment|docstring) ]]; then
        echo "documentation"
        return
    fi
    
    # Explanation indicators
    if [[ "$request" =~ (explain|describe|what.*does|how.*works) ]]; then
        echo "explanation"
        return
    fi
    
    # Conversion indicators
    if [[ "$request" =~ (convert|translate|transform|port) ]]; then
        echo "conversion"
        return
    fi
    
    # Default to general
    echo "general"
}

################################################################################
# API-Specific Generation
################################################################################

#######################################
# Generate text via completions API
# Arguments:
#   $1 - Model configuration (JSON)
#   $2 - User request
#   $3 - Generation type
# Returns:
#   Generated text
#######################################
text_generation::generate_via_completions() {
    local model_config="$1"
    local request="$2"
    local generation_type="$3"
    
    # Source completions API
    source "${APP_ROOT}/resources/codex/lib/apis/completions.sh"
    
    local model_name
    model_name=$(echo "$model_config" | jq -r '.model_name')
    
    # Get appropriate system message for generation type
    local system_message
    system_message=$(text_generation::get_system_message "$generation_type")
    
    # Route to specialized function based on generation type
    case "$generation_type" in
        code)
            # Detect programming language if possible
            local language
            language=$(text_generation::detect_language "$request")
            completions_api::generate_code "$model_name" "$request" "$language"
            ;;
        explanation)
            completions_api::explain_code "$model_name" "$request"
            ;;
        conversion)
            # Extract source and target from request
            local target_language
            target_language=$(text_generation::extract_target_language "$request")
            if [[ -n "$target_language" ]]; then
                completions_api::convert_code "$model_name" "$request" "$target_language"
            else
                completions_api::generate_text "$model_name" "$request" "$system_message"
            fi
            ;;
        *)
            completions_api::generate_text "$model_name" "$request" "$system_message"
            ;;
    esac
}

#######################################
# Generate text via responses API
# Arguments:
#   $1 - Model configuration (JSON)
#   $2 - User request
#   $3 - Generation type
# Returns:
#   Generated text
#######################################
text_generation::generate_via_responses() {
    local model_config="$1"
    local request="$2"
    local generation_type="$3"
    
    # Source responses API
    source "${APP_ROOT}/resources/codex/lib/apis/responses.sh"
    
    local model_name
    model_name=$(echo "$model_config" | jq -r '.model_name')
    
    # Check if model supports responses API
    if ! responses_api::check_model "$model_name"; then
        log::warn "Model $model_name may not support responses API, falling back to completions"
        text_generation::generate_via_completions "$model_config" "$request" "$generation_type"
        return
    fi
    
    # For code generation with responses API, include reasoning
    if [[ "$generation_type" == "code" ]]; then
        responses_api::generate_code_with_reasoning "$model_name" "$request" "false"
    else
        # Build messages for responses API
        local system_message
        system_message=$(text_generation::get_system_message "$generation_type")
        
        local messages
        messages=$(jq -n \
            --arg system "$system_message" \
            --arg user "$request" \
            '[{"role": "system", "content": $system}, {"role": "user", "content": $user}]')
        
        local response
        response=$(responses_api::call "$model_name" "$messages")
        
        if ! http_client::has_error "$response"; then
            responses_api::extract_content "$response"
        else
            return 1
        fi
    fi
}

################################################################################
# Helper Functions
################################################################################

#######################################
# Get appropriate system message for generation type
# Arguments:
#   $1 - Generation type
# Returns:
#   System message string
#######################################
text_generation::get_system_message() {
    local generation_type="$1"
    
    case "$generation_type" in
        code)
            echo "You are an expert programmer. Generate clean, efficient, well-commented code following best practices."
            ;;
        documentation)
            echo "You are a technical writer. Create clear, comprehensive documentation that helps users understand and use the code effectively."
            ;;
        explanation)
            echo "You are an expert programmer and teacher. Explain code and technical concepts clearly, breaking down complex ideas into understandable parts."
            ;;
        conversion)
            echo "You are an expert programmer fluent in multiple languages. Convert code accurately while maintaining functionality and following target language best practices."
            ;;
        general)
            echo "You are a helpful AI assistant. Provide accurate, helpful responses to user questions."
            ;;
        *)
            echo "You are an expert programmer and helpful AI assistant."
            ;;
    esac
}

#######################################
# Detect programming language from request
# Arguments:
#   $1 - User request
# Returns:
#   Language name or empty string
#######################################
text_generation::detect_language() {
    local request="$1"
    
    # Direct language mentions
    if [[ "$request" =~ [Pp]ython ]]; then echo "python"; return; fi
    if [[ "$request" =~ [Jj]ava[Ss]cript|[Jj][Ss] ]]; then echo "javascript"; return; fi
    if [[ "$request" =~ [Tt]ype[Ss]cript|[Tt][Ss] ]]; then echo "typescript"; return; fi
    if [[ "$request" =~ [Gg]o[[:space:]]|[Gg]olang ]]; then echo "go"; return; fi
    if [[ "$request" =~ [Rr]ust ]]; then echo "rust"; return; fi
    if [[ "$request" =~ [Jj]ava[[:space:]] ]]; then echo "java"; return; fi
    if [[ "$request" =~ [Cc]\+\+|[Cc]pp ]]; then echo "cpp"; return; fi
    if [[ "$request" =~ [Bb]ash|[Ss]hell ]]; then echo "bash"; return; fi
    if [[ "$request" =~ SQL|[Dd]atabase ]]; then echo "sql"; return; fi
    if [[ "$request" =~ PHP ]]; then echo "php"; return; fi
    if [[ "$request" =~ Ruby ]]; then echo "ruby"; return; fi
    
    # Context clues
    if [[ "$request" =~ (django|flask|pandas|numpy|pip) ]]; then echo "python"; return; fi
    if [[ "$request" =~ (react|node|express|npm|yarn) ]]; then echo "javascript"; return; fi
    if [[ "$request" =~ (gin|gorm|goroutine) ]]; then echo "go"; return; fi
    if [[ "$request" =~ (spring|maven|gradle) ]]; then echo "java"; return; fi
    
    echo ""
}

#######################################
# Extract target language from conversion request
# Arguments:
#   $1 - User request
# Returns:
#   Target language name or empty string
#######################################
text_generation::extract_target_language() {
    local request="$1"
    
    # Look for "to [language]" patterns
    if [[ "$request" =~ to[[:space:]]+([Pp]ython) ]]; then echo "python"; return; fi
    if [[ "$request" =~ to[[:space:]]+([Jj]ava[Ss]cript) ]]; then echo "javascript"; return; fi
    if [[ "$request" =~ to[[:space:]]+([Gg]o) ]]; then echo "go"; return; fi
    if [[ "$request" =~ to[[:space:]]+([Rr]ust) ]]; then echo "rust"; return; fi
    if [[ "$request" =~ to[[:space:]]+([Jj]ava) ]]; then echo "java"; return; fi
    
    # Look for "in [language]" patterns
    if [[ "$request" =~ in[[:space:]]+([Pp]ython) ]]; then echo "python"; return; fi
    if [[ "$request" =~ in[[:space:]]+([Jj]ava[Ss]cript) ]]; then echo "javascript"; return; fi
    if [[ "$request" =~ in[[:space:]]+([Gg]o) ]]; then echo "go"; return; fi
    
    echo ""
}

################################################################################
# Quality Enhancement Functions
################################################################################

#######################################
# Post-process generated text
# Arguments:
#   $1 - Generated text
#   $2 - Generation type
#   $3 - Programming language (optional)
# Returns:
#   Enhanced text
#######################################
text_generation::post_process() {
    local text="$1"
    local generation_type="$2"
    local language="${3:-}"
    
    case "$generation_type" in
        code)
            text_generation::enhance_code_output "$text" "$language"
            ;;
        *)
            echo "$text"
            ;;
    esac
}

#######################################
# Enhance code output with formatting and metadata
# Arguments:
#   $1 - Generated code
#   $2 - Programming language (optional)
# Returns:
#   Enhanced code output
#######################################
text_generation::enhance_code_output() {
    local code="$1"
    local language="${2:-}"
    
    # Add language identifier if detected
    if [[ -n "$language" ]]; then
        echo "# $language code:"
        echo ""
    fi
    
    # Output the code
    echo "$code"
    
    # Add usage notes for certain languages
    case "$language" in
        python)
            if [[ "$code" =~ def.*main ]]; then
                echo ""
                echo "# Run with: python script.py"
            fi
            ;;
        javascript)
            if [[ "$code" =~ require\( ]]; then
                echo ""
                echo "# Run with: node script.js"
            fi
            ;;
        go)
            echo ""
            echo "# Run with: go run main.go"
            ;;
    esac
}

################################################################################
# Utility Functions
################################################################################

#######################################
# Estimate token count for request
# Arguments:
#   $1 - Text content
# Returns:
#   Estimated token count
#######################################
text_generation::estimate_tokens() {
    local text="$1"
    
    # Rough estimation: ~4 characters per token
    local char_count=${#text}
    local token_estimate=$((char_count / 4))
    
    echo "$token_estimate"
}

#######################################
# Validate generation request
# Arguments:
#   $1 - User request
# Returns:
#   0 if valid, 1 if invalid
#######################################
text_generation::validate_request() {
    local request="$1"
    
    # Check for minimum length
    if [[ ${#request} -lt 5 ]]; then
        log::error "Request too short"
        return 1
    fi
    
    # Check for maximum length
    if [[ ${#request} -gt 10000 ]]; then
        log::error "Request too long"
        return 1
    fi
    
    # Basic content validation
    if [[ "$request" =~ ^[[:space:]]*$ ]]; then
        log::error "Request is empty or only whitespace"
        return 1
    fi
    
    return 0
}

# Export functions
export -f text_generation::execute
export -f text_generation::analyze_request
export -f text_generation::detect_language