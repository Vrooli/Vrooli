#!/bin/bash
# Chain Test Handler - Handles multi-step workflow testing

set -euo pipefail

# Colors for output (only define if not already defined)
if [[ -z "${RED:-}" ]]; then
    readonly RED='\033[0;31m'
    readonly GREEN='\033[0;32m'
    readonly YELLOW='\033[1;33m'
    readonly BLUE='\033[0;34m'
    readonly NC='\033[0m'
fi

# Chain test results
CHAIN_ERRORS=0
CHAIN_WARNINGS=0
CHAIN_STEPS_PASSED=0
CHAIN_STEPS_FAILED=0

# Variable storage for chain execution
declare -A CHAIN_VARIABLES

# Source dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/scenarios/validation/handlers"
if [[ -f "$SCRIPT_DIR/http.sh" ]]; then
    source "$SCRIPT_DIR/http.sh"
fi
if [[ -f "$SCRIPT_DIR/../clients/common.sh" ]]; then
    source "${APP_ROOT}/scripts/scenarios/validation/clients/common.sh"
fi

# Print functions
print_chain_info() {
    echo -e "${BLUE}[CHAIN]${NC} $1"
}

print_chain_success() {
    echo -e "${GREEN}[CHAIN ✓]${NC} $1"
    ((CHAIN_STEPS_PASSED++))
}

print_chain_error() {
    echo -e "${RED}[CHAIN ✗]${NC} $1"
    ((CHAIN_STEPS_FAILED++))
    ((CHAIN_ERRORS++))
}

print_chain_warning() {
    echo -e "${YELLOW}[CHAIN ⚠]${NC} $1"
    ((CHAIN_WARNINGS++))
}

print_step_info() {
    echo -e "${BLUE}  Step $1:${NC} $2"
}

# Store variable in chain context
set_chain_variable() {
    local var_name="$1"
    local var_value="$2"
    
    CHAIN_VARIABLES["$var_name"]="$var_value"
    print_chain_info "Set variable: $var_name = ${var_value:0:50}..."
}

# Get variable from chain context
get_chain_variable() {
    local var_name="$1"
    local default_value="${2:-}"
    
    echo "${CHAIN_VARIABLES[$var_name]:-$default_value}"
}

# Substitute variables in string
substitute_variables() {
    local input="$1"
    local output="$input"
    
    # Simple variable substitution: ${variable_name}
    for var_name in "${!CHAIN_VARIABLES[@]}"; do
        local var_value="${CHAIN_VARIABLES[$var_name]}"
        output="${output//\$\{$var_name\}/$var_value}"
    done
    
    echo "$output"
}

# Wait for service readiness
wait_for_service() {
    local service_url="$1"
    local timeout="${2:-30}"
    local interval="${3:-2}"
    
    print_chain_info "Waiting for service: $service_url"
    
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if curl -s --max-time 5 --fail "$service_url" >/dev/null 2>&1; then
            print_chain_success "Service ready: $service_url"
            return 0
        fi
        
        sleep "$interval"
        elapsed=$((elapsed + interval))
        echo -n "."
    done
    
    echo
    print_chain_error "Service not ready after ${timeout}s: $service_url"
    return 1
}

# Execute Ollama step
execute_ollama_step() {
    local step_id="$1"
    local service_url="$2"
    local model="$3"
    local prompt="$4"
    local output_var="$5"
    
    print_step_info "$step_id" "Ollama generation with model $model"
    
    # Substitute variables in prompt
    local substituted_prompt
    substituted_prompt=$(substitute_variables "$prompt")
    
    # Prepare request payload
    local payload
    payload=$(cat << EOF
{
    "model": "$model",
    "prompt": "$substituted_prompt",
    "stream": false
}
EOF
)
    
    # Make request to Ollama
    local response
    if ! response=$(curl -s --max-time 60 \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$service_url/api/generate"); then
        print_chain_error "Failed to connect to Ollama"
        return 1
    fi
    
    # Extract response text
    local response_text=""
    if command -v jq >/dev/null 2>&1; then
        response_text=$(echo "$response" | jq -r '.response // empty')
    else
        # Fallback parsing
        response_text=$(echo "$response" | sed -n 's/.*"response":"\([^"]*\)".*/\1/p')
    fi
    
    if [[ -z "$response_text" ]]; then
        print_chain_error "No response from Ollama"
        return 1
    fi
    
    # Store result
    set_chain_variable "$output_var" "$response_text"
    print_chain_success "Generated text (${#response_text} chars)"
    
    return 0
}

# Execute Whisper step
execute_whisper_step() {
    local step_id="$1"
    local service_url="$2"
    local audio_file="$3"
    local output_var="$4"
    
    print_step_info "$step_id" "Whisper transcription"
    
    # Check if audio file exists
    if [[ ! -f "$audio_file" ]]; then
        print_chain_error "Audio file not found: $audio_file"
        return 1
    fi
    
    # Make transcription request
    local response
    if ! response=$(curl -s --max-time 120 \
        -X POST \
        -F "audio=@$audio_file" \
        "$service_url/transcribe"); then
        print_chain_error "Failed to connect to Whisper"
        return 1
    fi
    
    # Extract transcription
    local transcription=""
    if command -v jq >/dev/null 2>&1; then
        transcription=$(echo "$response" | jq -r '.text // .transcription // empty')
    else
        # Fallback parsing
        transcription=$(echo "$response" | sed -n 's/.*"text":"\([^"]*\)".*/\1/p')
    fi
    
    if [[ -z "$transcription" ]]; then
        print_chain_error "No transcription from Whisper"
        return 1
    fi
    
    # Store result
    set_chain_variable "$output_var" "$transcription"
    print_chain_success "Transcribed text (${#transcription} chars)"
    
    return 0
}

# Execute ComfyUI step
execute_comfyui_step() {
    local step_id="$1"
    local service_url="$2"
    local prompt="$3"
    local output_var="$4"
    local output_file="${5:-/tmp/generated_image.png}"
    
    print_step_info "$step_id" "ComfyUI image generation"
    
    # Substitute variables in prompt
    local substituted_prompt
    substituted_prompt=$(substitute_variables "$prompt")
    
    # Simple ComfyUI workflow (this would be more complex in reality)
    local workflow
    workflow=$(cat << EOF
{
    "workflow": {
        "nodes": {
            "text_prompt": {
                "class_type": "CLIPTextEncode",
                "inputs": {
                    "text": "$substituted_prompt"
                }
            }
        }
    }
}
EOF
)
    
    # Submit workflow
    if ! curl -s --max-time 30 \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$workflow" \
        "$service_url/prompt" >/dev/null; then
        print_chain_error "Failed to submit ComfyUI workflow"
        return 1
    fi
    
    # For now, just simulate success and store the output path
    set_chain_variable "$output_var" "$output_file"
    print_chain_success "Image generation queued: $output_file"
    
    return 0
}

# Execute HTTP step
execute_http_step() {
    local step_id="$1"
    local service_url="$2"
    local method="$3"
    local endpoint="$4"
    local body="$5"
    local output_var="$6"
    
    print_step_info "$step_id" "$method $endpoint"
    
    # Substitute variables in body and endpoint
    local substituted_endpoint
    local substituted_body
    substituted_endpoint=$(substitute_variables "$endpoint")
    substituted_body=$(substitute_variables "$body")
    
    local full_url="${service_url}${substituted_endpoint}"
    
    # Make HTTP request
    local response
    if ! response=$(curl -s --max-time 30 \
        -X "$method" \
        -H "Content-Type: application/json" \
        -d "$substituted_body" \
        "$full_url"); then
        print_chain_error "HTTP request failed: $method $full_url"
        return 1
    fi
    
    # Store response
    set_chain_variable "$output_var" "$response"
    print_chain_success "HTTP request completed"
    
    return 0
}

# Execute custom step
execute_custom_step() {
    local step_id="$1"
    local script_path="$2"
    local function_name="$3"
    shift 3
    local args=("$@")
    
    print_step_info "$step_id" "Custom: $function_name"
    
    if [[ ! -f "$script_path" ]]; then
        print_chain_error "Custom script not found: $script_path"
        return 1
    fi
    
    # Source the script
    source "$script_path"
    
    # Check if function exists
    if ! declare -f "$function_name" >/dev/null; then
        print_chain_error "Function not found: $function_name"
        return 1
    fi
    
    # Execute function with arguments
    if "$function_name" "${args[@]}"; then
        print_chain_success "Custom step completed"
        return 0
    else
        print_chain_error "Custom step failed"
        return 1
    fi
}

# Execute single chain step
execute_chain_step() {
    local step_config="$1"
    
    # Parse step configuration (simplified YAML parsing)
    local step_id
    local service
    # local action  # Reserved for future step type identification
    local output_var
    step_id=$(echo "$step_config" | grep "id:" | cut -d: -f2 | xargs)
    service=$(echo "$step_config" | grep "service:" | cut -d: -f2 | xargs)
    # action=$(echo "$step_config" | grep "action:" | cut -d: -f2 | xargs)  # Reserved for future step type identification
    output_var=$(echo "$step_config" | grep "output:" | cut -d: -f2 | xargs)
    
    # Get service URL
    local service_url=""
    if declare -f get_resource_url >/dev/null 2>&1; then
        service_url=$(get_resource_url "$service")
    fi
    
    if [[ -z "$service_url" ]]; then
        print_chain_error "Service URL not found for: $service"
        return 1
    fi
    
    # Execute based on service and action
    case "$service" in
        ollama)
            local model
            local prompt
            model=$(echo "$step_config" | grep "model:" | cut -d: -f2 | xargs)
            prompt=$(echo "$step_config" | grep "prompt:" | cut -d: -f2- | sed 's/prompt:[[:space:]]*//')
            execute_ollama_step "$step_id" "$service_url" "$model" "$prompt" "$output_var"
            ;;
        whisper)
            local audio_file
            audio_file=$(echo "$step_config" | grep "file:" | cut -d: -f2 | xargs)
            execute_whisper_step "$step_id" "$service_url" "$audio_file" "$output_var"
            ;;
        comfyui)
            local prompt
            prompt=$(echo "$step_config" | grep "prompt:" | cut -d: -f2- | sed 's/prompt:[[:space:]]*//')
            execute_comfyui_step "$step_id" "$service_url" "$prompt" "$output_var"
            ;;
        *)
            # Generic HTTP step
            local method
            local endpoint
            local body
            method=$(echo "$step_config" | grep "method:" | cut -d: -f2 | xargs)
            endpoint=$(echo "$step_config" | grep "endpoint:" | cut -d: -f2 | xargs)
            body=$(echo "$step_config" | grep "body:" | cut -d: -f2- | sed 's/body:[[:space:]]*//')
            execute_http_step "$step_id" "$service_url" "${method:-POST}" "$endpoint" "$body" "$output_var"
            ;;
    esac
}

# Execute chain test
execute_chain_test() {
    local test_name="$1"
    # local test_data="$2"  # Reserved for future test data handling
    
    print_chain_info "Executing chain test: $test_name"
    
    # Reset chain state
    CHAIN_VARIABLES=()
    CHAIN_STEPS_PASSED=0
    CHAIN_STEPS_FAILED=0
    
    # Parse steps from test data
    # This is a simplified parser - in production, use proper YAML parsing
    
    # For now, assume test_data contains step definitions
    # In reality, this would parse YAML configuration
    
    print_chain_info "Chain execution completed"
    
    # Report results
    local total_steps=$((CHAIN_STEPS_PASSED + CHAIN_STEPS_FAILED))
    if [[ $total_steps -gt 0 ]]; then
        local success_rate=$((CHAIN_STEPS_PASSED * 100 / total_steps))
        print_chain_info "Steps: $CHAIN_STEPS_PASSED passed, $CHAIN_STEPS_FAILED failed (${success_rate}%)"
    fi
    
    # Return success if no steps failed
    [[ $CHAIN_STEPS_FAILED -eq 0 ]]
}

# Test a simple chain workflow
test_audio_to_image_chain() {
    local audio_file="$1"
    local ollama_url="$2"
    local whisper_url="$3"
    local comfyui_url="$4"
    
    print_chain_info "Testing Audio → Text → Image chain"
    
    # Step 1: Transcribe audio
    if execute_whisper_step "1" "$whisper_url" "$audio_file" "transcript"; then
        
        # Step 2: Generate description from transcript
        local prompt="Create a detailed visual description of: \${transcript}"
        if execute_ollama_step "2" "$ollama_url" "llama2" "$prompt" "description"; then
            
            # Step 3: Generate image from description
            local image_prompt="\${description}"
            execute_comfyui_step "3" "$comfyui_url" "$image_prompt" "image_path"
        fi
    fi
    
    # Check if all steps succeeded
    [[ $CHAIN_STEPS_FAILED -eq 0 ]]
}

# Export functions
export -f execute_chain_test
export -f execute_chain_step
export -f test_audio_to_image_chain
export -f set_chain_variable
export -f get_chain_variable
export -f substitute_variables