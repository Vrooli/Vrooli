#!/usr/bin/env bash
set -euo pipefail

# Whisper Data Injection Adapter
# This script handles injection of models and configurations into Whisper
# Part of the Vrooli resource data injection system

DESCRIPTION="Inject models and configurations into Whisper speech-to-text service"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source var.sh first to get directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"

# Source common utilities using var.sh variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source Whisper configuration if available
if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default Whisper settings
readonly DEFAULT_WHISPER_HOST="http://localhost:9005"
readonly DEFAULT_WHISPER_DATA_DIR="${HOME}/.whisper"
readonly DEFAULT_WHISPER_MODELS_DIR="${DEFAULT_WHISPER_DATA_DIR}/models"

# Whisper settings (can be overridden by environment)
WHISPER_HOST="${WHISPER_HOST:-$DEFAULT_WHISPER_HOST}"
WHISPER_DATA_DIR="${WHISPER_DATA_DIR:-$DEFAULT_WHISPER_DATA_DIR}"
WHISPER_MODELS_DIR="${WHISPER_MODELS_DIR:-$DEFAULT_WHISPER_MODELS_DIR}"

# Operation tracking
declare -a WHISPER_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
inject::usage() {
    cat << EOF
Whisper Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects models and configurations into Whisper based on scenario configuration.
    Supports validation, injection, status checks, and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the model injection
    --status      Check status of injected models
    --rollback    Rollback injected models
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "models": [
        {
          "name": "medium",
          "download": true,
          "preload": true
        },
        {
          "name": "large-v3",
          "download": true,
          "preload": false
        }
      ],
      "configurations": [
        {
          "key": "language",
          "value": "en"
        }
      ],
      "audio_samples": [
        {
          "file": "path/to/audio.mp3",
          "name": "sample_audio"
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"models": [{"name": "medium", "download": true}]}'
    
    # Download and inject models
    $0 --inject '{"models": [{"name": "base", "download": true, "preload": true}]}'
    
    # Upload audio samples
    $0 --inject '{"audio_samples": [{"file": "samples/test.mp3", "name": "test"}]}'

EOF
}

#######################################
# Check if Whisper is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
inject::check_accessibility() {
    # Check if Whisper is running
    if curl -s --max-time 5 "${WHISPER_HOST}/docs" >/dev/null 2>&1; then
        log::debug "Whisper is accessible at $WHISPER_HOST"
        return 0
    else
        log::error "Whisper is not accessible at $WHISPER_HOST"
        log::info "Ensure Whisper is running: ./scripts/resources/ai/whisper/manage.sh --action start"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    WHISPER_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Whisper rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
inject::execute_rollback() {
    if [[ ${#WHISPER_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Whisper rollback actions to execute"
        return 0
    fi
    
    log::info "Executing Whisper rollback actions..."
    
    local success_count=0
    local total_count=${#WHISPER_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#WHISPER_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${WHISPER_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "Whisper rollback completed: $success_count/$total_count actions successful"
    WHISPER_ROLLBACK_ACTIONS=()
}

#######################################
# Validate model configuration
# Arguments:
#   $1 - models configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
inject::validate_models() {
    local models_config="$1"
    
    log::debug "Validating model configurations..."
    
    # Check if models is an array
    local models_type
    models_type=$(echo "$models_config" | jq -r 'type')
    
    if [[ "$models_type" != "array" ]]; then
        log::error "Models configuration must be an array, got: $models_type"
        return 1
    fi
    
    # Validate each model
    local model_count
    model_count=$(echo "$models_config" | jq 'length')
    
    local valid_models=("tiny" "base" "small" "medium" "large" "large-v2" "large-v3")
    
    for ((i=0; i<model_count; i++)); do
        local model
        model=$(echo "$models_config" | jq -c ".[$i]")
        
        # Check required fields
        local name download
        name=$(echo "$model" | jq -r '.name // empty')
        download=$(echo "$model" | jq -r '.download // false')
        
        if [[ -z "$name" ]]; then
            log::error "Model at index $i missing required 'name' field"
            return 1
        fi
        
        # Validate model name
        local is_valid=false
        for valid_name in "${valid_models[@]}"; do
            if [[ "$name" == "$valid_name" ]]; then
                is_valid=true
                break
            fi
        done
        
        if [[ "$is_valid" == false ]]; then
            log::error "Invalid model name: $name. Valid models: ${valid_models[*]}"
            return 1
        fi
        
        log::debug "Model '$name' configuration is valid"
    done
    
    log::success "All model configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
inject::validate_config() {
    local config="$1"
    
    log::info "Validating Whisper injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Whisper injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_models has_configurations has_audio
    has_models=$(echo "$config" | jq -e '.models' >/dev/null 2>&1 && echo "true" || echo "false")
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    has_audio=$(echo "$config" | jq -e '.audio_samples' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_models" == "false" && "$has_configurations" == "false" && "$has_audio" == "false" ]]; then
        log::error "Whisper injection configuration must have 'models', 'configurations', or 'audio_samples'"
        return 1
    fi
    
    # Validate models if present
    if [[ "$has_models" == "true" ]]; then
        local models
        models=$(echo "$config" | jq -c '.models')
        
        if ! inject::validate_models "$models"; then
            return 1
        fi
    fi
    
    # Validate audio samples if present
    if [[ "$has_audio" == "true" ]]; then
        local audio_samples
        audio_samples=$(echo "$config" | jq -c '.audio_samples')
        
        local audio_count
        audio_count=$(echo "$audio_samples" | jq 'length')
        
        for ((i=0; i<audio_count; i++)); do
            local audio
            audio=$(echo "$audio_samples" | jq -c ".[$i]")
            
            local file
            file=$(echo "$audio" | jq -r '.file // empty')
            
            if [[ -z "$file" ]]; then
                log::error "Audio sample at index $i missing required 'file' field"
                return 1
            fi
            
            # Check if file exists
            local audio_path="$var_ROOT_DIR/$file"
            if [[ ! -f "$audio_path" ]]; then
                log::error "Audio file not found: $audio_path"
                return 1
            fi
        done
    fi
    
    log::success "Whisper injection configuration is valid"
    return 0
}

#######################################
# Download model
# Arguments:
#   $1 - model configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::download_model() {
    local model_config="$1"
    
    local name preload
    name=$(echo "$model_config" | jq -r '.name')
    preload=$(echo "$model_config" | jq -r '.preload // false')
    
    log::info "Downloading model: $name"
    
    # Create models directory if it doesn't exist
    mkdir -p "$WHISPER_MODELS_DIR"
    
    # For Whisper, models are typically downloaded automatically on first use
    # We can trigger a download by making a transcription request with the model
    if [[ "$preload" == "true" ]]; then
        log::info "Preloading model: $name"
        
        # Create a small test audio file for preloading
        local test_audio="/tmp/whisper_test_${name}.wav"
        
        # Generate a 1-second silent audio file using sox if available
        if system::is_command "sox"; then
            sox -n -r 16000 -c 1 "$test_audio" trim 0.0 1.0 2>/dev/null || true
        fi
        
        # If test audio exists, use it to preload the model
        if [[ -f "$test_audio" ]]; then
            curl -s -X POST "${WHISPER_HOST}/asr" \
                -F "audio_file=@${test_audio}" \
                -F "model_size=${name}" \
                >/dev/null 2>&1 || true
            
            rm -f "$test_audio"
        fi
    fi
    
    log::success "Model '$name' prepared"
    
    # Add rollback action (models are cached, so we just note it)
    inject::add_rollback_action \
        "Remove model cache: $name" \
        "trash::safe_remove '${WHISPER_MODELS_DIR}/${name}'* --no-confirm 2>/dev/null || true"
    
    return 0
}

#######################################
# Upload audio sample
# Arguments:
#   $1 - audio configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::upload_audio() {
    local audio_config="$1"
    
    local file name
    file=$(echo "$audio_config" | jq -r '.file')
    name=$(echo "$audio_config" | jq -r '.name // empty')
    
    # Resolve file path
    local audio_path="$var_ROOT_DIR/$file"
    
    if [[ -z "$name" ]]; then
        name=$(basename "$file")
    fi
    
    log::info "Uploading audio sample: $name"
    
    # Create uploads directory if it doesn't exist
    local uploads_dir="${WHISPER_DATA_DIR}/uploads"
    mkdir -p "$uploads_dir"
    
    # Copy audio file to uploads directory
    cp "$audio_path" "$uploads_dir/$name"
    
    log::success "Uploaded audio sample: $name"
    
    # Add rollback action
    inject::add_rollback_action \
        "Remove audio sample: $name" \
        "rm -f '${uploads_dir}/${name}' 2>/dev/null || true"
    
    return 0
}

#######################################
# Inject models
# Arguments:
#   $1 - models configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::inject_models() {
    local models_config="$1"
    
    log::info "Injecting Whisper models..."
    
    local model_count
    model_count=$(echo "$models_config" | jq 'length')
    
    if [[ "$model_count" -eq 0 ]]; then
        log::info "No models to inject"
        return 0
    fi
    
    local failed_models=()
    
    for ((i=0; i<model_count; i++)); do
        local model
        model=$(echo "$models_config" | jq -c ".[$i]")
        
        local name download
        name=$(echo "$model" | jq -r '.name')
        download=$(echo "$model" | jq -r '.download // false')
        
        if [[ "$download" == "true" ]]; then
            if ! inject::download_model "$model"; then
                failed_models+=("$name")
            fi
        fi
    done
    
    if [[ ${#failed_models[@]} -eq 0 ]]; then
        log::success "All models injected successfully"
        return 0
    else
        log::error "Failed to inject models: ${failed_models[*]}"
        return 1
    fi
}

#######################################
# Inject audio samples
# Arguments:
#   $1 - audio samples configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::inject_audio_samples() {
    local audio_config="$1"
    
    log::info "Injecting audio samples..."
    
    local audio_count
    audio_count=$(echo "$audio_config" | jq 'length')
    
    if [[ "$audio_count" -eq 0 ]]; then
        log::info "No audio samples to inject"
        return 0
    fi
    
    local failed_samples=()
    
    for ((i=0; i<audio_count; i++)); do
        local audio
        audio=$(echo "$audio_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$audio" | jq -r '.name // empty')
        
        if [[ -z "$name" ]]; then
            name=$(echo "$audio" | jq -r '.file' | xargs basename)
        fi
        
        if ! inject::upload_audio "$audio"; then
            failed_samples+=("$name")
        fi
    done
    
    if [[ ${#failed_samples[@]} -eq 0 ]]; then
        log::success "All audio samples injected successfully"
        return 0
    else
        log::error "Failed to inject audio samples: ${failed_samples[*]}"
        return 1
    fi
}

#######################################
# Apply configurations
# Arguments:
#   $1 - configurations JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::apply_configurations() {
    local configurations="$1"
    
    log::info "Applying Whisper configurations..."
    
    local config_count
    config_count=$(echo "$configurations" | jq 'length')
    
    if [[ "$config_count" -eq 0 ]]; then
        log::info "No configurations to apply"
        return 0
    fi
    
    # Note: Whisper configuration is typically done via environment variables
    # This is a placeholder for future configuration injection capabilities
    log::warn "Configuration injection not yet fully implemented for Whisper"
    log::info "Set environment variables before starting Whisper for configuration"
    
    return 0
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into Whisper"
    
    # Check Whisper accessibility
    if ! inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    WHISPER_ROLLBACK_ACTIONS=()
    
    # Inject models if present
    local has_models
    has_models=$(echo "$config" | jq -e '.models' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_models" == "true" ]]; then
        local models
        models=$(echo "$config" | jq -c '.models')
        
        if ! inject::inject_models "$models"; then
            log::error "Failed to inject models"
            inject::execute_rollback
            return 1
        fi
    fi
    
    # Apply configurations if present
    local has_configurations
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_configurations" == "true" ]]; then
        local configurations
        configurations=$(echo "$config" | jq -c '.configurations')
        
        if ! inject::apply_configurations "$configurations"; then
            log::error "Failed to apply configurations"
            inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject audio samples if present
    local has_audio
    has_audio=$(echo "$config" | jq -e '.audio_samples' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_audio" == "true" ]]; then
        local audio_samples
        audio_samples=$(echo "$config" | jq -c '.audio_samples')
        
        if ! inject::inject_audio_samples "$audio_samples"; then
            log::error "Failed to inject audio samples"
            inject::execute_rollback
            return 1
        fi
    fi
    
    log::success "✅ Whisper data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking Whisper injection status"
    
    # Check Whisper accessibility
    if ! inject::check_accessibility; then
        return 1
    fi
    
    # Check model directory
    log::info "Checking model cache..."
    
    if [[ -d "$WHISPER_MODELS_DIR" ]]; then
        local model_files
        model_files=$(find "$WHISPER_MODELS_DIR" -type f -name "*.pt" 2>/dev/null | wc -l)
        
        if [[ "$model_files" -gt 0 ]]; then
            log::info "Found $model_files model files in cache"
        else
            log::warn "No model files found in cache"
        fi
    else
        log::warn "Model directory does not exist: $WHISPER_MODELS_DIR"
    fi
    
    # Check audio samples
    local uploads_dir="${WHISPER_DATA_DIR}/uploads"
    
    if [[ -d "$uploads_dir" ]]; then
        local audio_files
        audio_files=$(find "$uploads_dir" -type f \( -name "*.mp3" -o -name "*.wav" -o -name "*.m4a" \) 2>/dev/null | wc -l)
        
        if [[ "$audio_files" -gt 0 ]]; then
            log::info "Found $audio_files audio files in uploads"
        else
            log::info "No audio files in uploads directory"
        fi
    fi
    
    # Test API endpoint
    log::info "Testing Whisper API..."
    
    if curl -s "${WHISPER_HOST}/docs" | grep -q "Whisper" 2>/dev/null; then
        log::success "✅ Whisper API is responding"
    else
        log::error "❌ Whisper API not responding properly"
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            inject::validate_config "$config"
            ;;
        "--inject")
            inject::inject_data "$config"
            ;;
        "--status")
            inject::check_status "$config"
            ;;
        "--rollback")
            inject::execute_rollback
            ;;
        "--help")
            inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        inject::usage
        exit 1
    fi
    
    inject::main "$@"
fi