#\!/usr/bin/env bash
# Codex Injection Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CODEX_INJECT_DIR="${APP_ROOT}/resources/codex/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CODEX_INJECT_DIR}/common.sh"

#######################################
# Inject a Python script for Codex processing
# Arguments:
#   file - Path to the Python script
# Returns:
#   0 on success, 1 on failure
#######################################
codex::inject() {
    local file="${1:-}"
    
    if [[ -z "${file}" ]]; then
        log::error "No file specified"
        return 1
    fi
    
    if [[ ! -f "${file}" ]]; then
        log::error "File not found: ${file}"
        return 1
    fi
    
    # Create scripts directory if needed
    mkdir -p "${CODEX_SCRIPTS_DIR}" "${CODEX_INJECTED_DIR}"
    
    # Copy to scripts directory
    local basename
    basename=$(basename "${file}")
    local target="${CODEX_SCRIPTS_DIR}/${basename}"
    
    log::info "Injecting script: ${basename}"
    cp "${file}" "${target}"
    
    # Also copy to injected directory for tracking
    cp "${file}" "${CODEX_INJECTED_DIR}/${basename}"
    
    log::success "Script injected successfully: ${basename}"
    log::info "Script location: ${target}"
    
    return 0
}

#######################################
# Run a script with Codex
# Arguments:
#   script - Name of the script to run
# Returns:
#   0 on success, 1 on failure
#######################################
codex::run() {
    local script="${1:-}"
    
    if [[ -z "${script}" ]]; then
        log::error "No script specified"
        return 1
    fi
    
    local script_path="${CODEX_SCRIPTS_DIR}/${script}"
    
    if [[ ! -f "${script_path}" ]]; then
        log::error "Script not found: ${script}"
        return 1
    fi
    
    local api_key
    api_key=$(codex::get_api_key)
    
    if [[ -z "${api_key}" ]]; then
        log::error "No API key configured"
        return 1
    fi
    
    log::info "Running script with Codex: ${script}"
    
    # Read the script content
    local content
    content=$(cat "${script_path}")
    
    # Create a completion request
    local request_data
    request_data=$(jq -n \
        --arg model "${CODEX_DEFAULT_MODEL}" \
        --arg prompt "${content}" \
        --arg temperature "${CODEX_DEFAULT_TEMPERATURE}" \
        --arg max_tokens "${CODEX_DEFAULT_MAX_TOKENS}" \
        '{
            model: $model,
            prompt: $prompt,
            temperature: ($temperature | tonumber),
            max_tokens: ($max_tokens | tonumber),
            top_p: 1,
            frequency_penalty: 0,
            presence_penalty: 0
        }')
    
    # Make the API call
    local response
    response=$(curl -s -X POST "${CODEX_API_ENDPOINT}/completions" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${api_key}" \
        -d "${request_data}" \
        --max-time "${CODEX_TIMEOUT}")
    
    # Check for errors
    if echo "${response}" | jq -e '.error' &>/dev/null; then
        local error_msg
        error_msg=$(echo "${response}" | jq -r '.error.message')
        log::error "API error: ${error_msg}"
        return 1
    fi
    
    # Extract completion
    local completion
    completion=$(echo "${response}" | jq -r '.choices[0].text')
    
    if [[ -z "${completion}" || "${completion}" == "null" ]]; then
        log::error "No completion generated"
        return 1
    fi
    
    # Save output
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)
    local output_file="${CODEX_OUTPUT_DIR}/${script%.py}_${timestamp}.py"
    
    echo "${completion}" > "${output_file}"
    
    log::success "Completion saved to: ${output_file}"
    echo ""
    echo "Generated code:"
    echo "----------------"
    echo "${completion}"
    
    return 0
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ "$1" == "inject" ]]; then
        shift
        codex::inject "$@"
    elif [[ "$1" == "run" ]]; then
        shift
        codex::run "$@"
    fi
}
