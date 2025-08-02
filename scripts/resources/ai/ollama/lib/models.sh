#!/usr/bin/env bash

# Ollama Model Management Functions
# This file contains all model catalog utilities, installation, and management functions

#######################################
# Get model information from catalog
# Arguments:
#   $1 - model name (e.g., "llama3.1:8b")
# Outputs: size_gb|capabilities|description
#######################################
ollama::get_model_info() {
    local model="$1"
    if ollama::is_model_known "$model"; then
        echo "${MODEL_CATALOG[$model]}"
    else
        echo "unknown|unknown|Model not found in catalog"
    fi
}

#######################################
# Get model size from catalog
# Arguments:
#   $1 - model name
# Outputs: size in GB
#######################################
ollama::get_model_size() {
    local model="$1"
    local info
    info=$(ollama::get_model_info "$model")
    echo "$info" | cut -d'|' -f1
}

#######################################
# Check if model is in catalog
# Arguments:
#   $1 - model name
# Returns: 0 if in catalog, 1 otherwise
#######################################
ollama::is_model_known() {
    local model="$1"
    for key in "${!MODEL_CATALOG[@]}"; do
        if [[ "$key" == "$model" ]]; then
            return 0
        fi
    done
    return 1
}

#######################################
# Display available models from catalog
#######################################
ollama::show_available_models() {
    log::header "$MSG_MODELS_HEADER"
    
    printf "%-20s %-8s %-30s %s\n" "MODEL" "SIZE" "CAPABILITIES" "DESCRIPTION"
    printf "%-20s %-8s %-30s %s\n" "----" "----" "----" "----"
    
    # Sort models by category and size
    local sorted_models=()
    
    # Current/Recommended models first
    for model in "llama3.1:8b" "deepseek-r1:8b" "qwen2.5-coder:7b"; do
        if ollama::is_model_known "$model"; then
            sorted_models+=("$model")
        fi
    done
    
    # Then other models alphabetically
    for model in "${!MODEL_CATALOG[@]}"; do
        if [[ ! " ${sorted_models[*]} " =~ " $model " ]]; then
            sorted_models+=("$model")
        fi
    done
    
    for model in "${sorted_models[@]}"; do
        local info="${MODEL_CATALOG[$model]}"
        local size=$(echo "$info" | cut -d'|' -f1)
        local capabilities=$(echo "$info" | cut -d'|' -f2)
        local description=$(echo "$info" | cut -d'|' -f3)
        
        # Highlight default models
        local marker=""
        if [[ " ${DEFAULT_MODELS[*]} " =~ " $model " ]]; then
            marker="✅"
        elif [[ "$capabilities" == *"legacy"* ]]; then
            marker="⚠️ "
        fi
        
        printf "%-20s %-8s %-30s %s %s\n" "$model" "${size}GB" "$capabilities" "$marker" "$description"
    done
    
    echo
    log::info "$MSG_MODELS_LEGEND"
    log::info "$MSG_MODELS_TOTAL_SIZE"
}

#######################################
# Calculate total size of default models
#######################################
ollama::calculate_default_size() {
    local total=0
    for model in "${DEFAULT_MODELS[@]}"; do
        local size
        size=$(ollama::get_model_size "$model")
        if [[ "$size" =~ ^[0-9]+\.?[0-9]*$ ]]; then
            total=$(echo "$total + $size" | bc -l 2>/dev/null || echo "$total")
        fi
    done
    printf "%.1f" "$total"
}

#######################################
# Validate model list against catalog
# Arguments:
#   $@ - list of models to validate
# Returns: 0 if all valid, 1 if any invalid
#######################################
ollama::validate_model_list() {
    local models=("$@")
    local invalid_models=()
    local total_size=0
    
    for model in "${models[@]}"; do
        if ! ollama::is_model_known "$model"; then
            invalid_models+=("$model")
        else
            local size
            size=$(ollama::get_model_size "$model")
            if [[ "$size" =~ ^[0-9]+\.?[0-9]*$ ]]; then
                total_size=$(echo "$total_size + $size" | bc -l 2>/dev/null || echo "$total_size")
            fi
        fi
    done
    
    if [[ ${#invalid_models[@]} -gt 0 ]]; then
        log::warn "$(MSG_UNKNOWN_MODELS "${invalid_models[@]}")"
        log::info "$MSG_USE_AVAILABLE_MODELS"
        return 1
    fi
    
    log::info "$(MSG_MODELS_VALIDATED "${models[@]}")"
    return 0
}

#######################################
# Get list of installed models
# Outputs: space-separated list of model names
#######################################
ollama::get_installed_models() {
    if ! ollama::is_healthy; then
        return 1
    fi
    
    if system::is_command "ollama"; then
        # Extract model names from ollama list output (skip header line)
        ollama list 2>/dev/null | tail -n +2 | awk '{print $1}' | tr '\n' ' ' | sed 's/ $//'
    else
        return 1
    fi
}

#######################################
# Get the best available model for a given use case
# Arguments:
#   $1 - use case (optional: "general", "code", "reasoning", "vision")
# Outputs: best model name, or empty if none available
#######################################
ollama::get_best_available_model() {
    local use_case="${1:-general}"
    local installed_models
    
    # Get list of installed models
    installed_models=$(ollama::get_installed_models)
    if [[ -z "$installed_models" ]]; then
        return 1
    fi
    
    # Convert to array
    local installed_array=()
    IFS=' ' read -ra installed_array <<< "$installed_models"
    
    # Define priority order based on use case
    local priority_models=()
    case "$use_case" in
        "code"|"programming")
            priority_models=(
                "qwen2.5-coder:7b" "qwen2.5-coder:32b" "deepseek-coder:6.7b"
                "deepseek-r1:8b" "deepseek-r1:14b"  # reasoning models are also good for code
                "llama3.1:8b" "llama3.3:8b" "codellama:7b"
            )
            ;;
        "reasoning"|"math")
            priority_models=(
                "deepseek-r1:8b" "deepseek-r1:14b" "deepseek-r1:1.5b"
                "qwen2.5:14b" "phi-4:14b"
                "llama3.1:8b" "llama3.3:8b"
            )
            ;;
        "vision"|"image")
            priority_models=(
                "llava:13b" "qwen2-vl:7b" "llama3.2-vision:11b"
                "llama3.1:8b"  # fallback to general model
            )
            ;;
        "general"|*)
            priority_models=(
                "llama3.1:8b" "llama3.3:8b" "deepseek-r1:8b"
                "qwen2.5:14b" "phi-4:14b" "mistral-small:22b"
                "qwen2.5-coder:7b"  # code models are often good general models too
                "llama2:7b"  # legacy fallback
            )
            ;;
    esac
    
    # Find the first priority model that's installed
    for priority_model in "${priority_models[@]}"; do
        for installed_model in "${installed_array[@]}"; do
            if [[ "$installed_model" == "$priority_model" ]]; then
                echo "$priority_model"
                return 0
            fi
        done
    done
    
    # If no priority match, return the first installed model
    if [[ ${#installed_array[@]} -gt 0 ]]; then
        echo "${installed_array[0]}"
        return 0
    fi
    
    return 1
}

#######################################
# Validate that a model exists and is available
# Arguments:
#   $1 - model name
# Returns: 0 if valid and available, 1 otherwise
#######################################
ollama::validate_model_available() {
    local model="$1"
    local installed_models
    
    installed_models=$(ollama::get_installed_models)
    if [[ -z "$installed_models" ]]; then
        return 1
    fi
    
    # Check if model is in the installed list
    if [[ " $installed_models " =~ " $model " ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Parse models from input string
# Uses global MODELS_INPUT variable
# Outputs: space-separated model list
#######################################
ollama::parse_models() {
    local input="$MODELS_INPUT"
    
    if [[ -z "$input" ]]; then
        echo "${DEFAULT_MODELS[*]}"
    else
        # Parse comma-separated list and clean up
        echo "$input" | tr ',' ' ' | tr -s ' '
    fi
}

#######################################
# Pull a single model
# Arguments:
#   $1 - model name
#######################################
ollama::pull_model() {
    local model="$1"
    
    log::info "Pulling model: $model"
    
    if ollama pull "$model"; then
        log::success "$MSG_MODEL_PULL_SUCCESS"
    else
        log::error "$MSG_MODEL_PULL_FAILED"
        return 1
    fi
}

#######################################
# Install default models with enhanced error handling and rollback
#######################################
ollama::install_models() {
    if [[ "$SKIP_MODELS" == "yes" ]]; then
        log::info "Skipping model installation (--skip-models specified)"
        return 0
    fi
    
    log::header "📦 Installing Ollama Models"
    
    # Validate Ollama API availability with retries
    local api_check_attempts=3
    local api_available=false
    
    for ((i=1; i<=api_check_attempts; i++)); do
        if ollama::is_healthy; then
            api_available=true
            break
        fi
        
        if [[ $i -lt $api_check_attempts ]]; then
            log::info "Ollama API not ready (attempt $i/$api_check_attempts), waiting 5 seconds..."
            sleep 5
        fi
    done
    
    if [[ "$api_available" != "true" ]]; then
        resources::handle_error \
            "Ollama API is not available after $api_check_attempts attempts" \
            "system" \
            "Check service status: systemctl status $OLLAMA_SERVICE_NAME; journalctl -u $OLLAMA_SERVICE_NAME -n 20"
        return 1
    fi
    
    # Parse and validate models list
    local models
    models=$(ollama::parse_models)
    
    if [[ -z "$models" ]]; then
        log::warn "$MSG_MODEL_NONE_SPECIFIED"
        return 0
    fi
    
    # Convert to array for validation
    local models_array=()
    IFS=' ' read -ra models_array <<< "$models"
    
    # Validate against catalog and show information
    log::info "Validating models against catalog..."
    if ! ollama::validate_model_list "${models_array[@]}"; then
        log::error "$MSG_MODEL_VALIDATION_FAILED"
        log::info "Run '$0 --action available' to see available models"
        return 1
    fi
    
    # Display model information
    log::info "📦 Models to install with catalog information:"
    for model in "${models_array[@]}"; do
        if ollama::is_model_known "$model"; then
            local info
            info=$(ollama::get_model_info "$model")
            local size=$(echo "$info" | cut -d'|' -f1)
            local capabilities=$(echo "$info" | cut -d'|' -f2)
            local description=$(echo "$info" | cut -d'|' -f3)
            log::info "  • $model (${size}GB) - $description"
        else
            log::info "  • $model (size unknown) - Not in catalog"
        fi
    done
    
    # Check available disk space (models can be several GB each)
    local available_space_gb
    if available_space_gb=$(df "$HOME" --output=avail --block-size=1G | tail -n1 | tr -d ' '); then
        if [[ $available_space_gb -lt 10 ]]; then
            log::warn "$(MSG_LOW_DISK_SPACE "$available_space_gb")"
            log::info "Each model typically requires 2-8GB. Consider freeing up space if downloads fail."
        fi
    fi
    
    local success_count=0
    local total_count=0
    local installed_models=()
    local failed_models=()
    
    # Track installed models for rollback
    for model in $models; do
        total_count=$((total_count + 1))
        
        log::info "[$total_count/$(echo $models | wc -w)] Installing model: $model"
        
        # Check if model already exists
        if ollama list 2>/dev/null | grep -q "^$model"; then
            log::info "Model $model already installed, skipping"
            success_count=$((success_count + 1))
            continue
        fi
        
        # Install model with progress
        if ollama::pull_model "$model"; then
            success_count=$((success_count + 1))
            installed_models+=("$model")
            
            # Add rollback action for model removal (low priority since models are large)
            resources::add_rollback_action \
                "Remove installed model: $model" \
                "ollama rm \\\"$model\\\" 2>/dev/null || true" \
                5
            
            log::success "$MSG_MODEL_INSTALL_SUCCESS"
        else
            failed_models+=("$model")
            log::error "$MSG_MODEL_INSTALL_FAILED"
        fi
        
        # Brief pause between models to avoid overwhelming the system
        if [[ $total_count -lt $(echo $models | wc -w) ]]; then
            sleep 1
        fi
    done
    
    # Summary report
    log::info "Model installation summary:"
    log::info "  • Successfully installed: $success_count/$total_count models"
    
    if [[ ${#installed_models[@]} -gt 0 ]]; then
        log::success "$MSG_MODELS_INSTALLED"
    fi
    
    if [[ ${#failed_models[@]} -gt 0 ]]; then
        log::warn "$MSG_MODELS_FAILED"
        log::info "  • Retry failed models with: ollama pull <model-name>"
    fi
    
    # Validate at least one model was installed successfully
    if [[ $success_count -eq 0 ]]; then
        resources::handle_error \
            "No models were installed successfully" \
            "network" \
            "Check internet connection and try installing models manually: ollama pull llama3.1:8b"
        return 1
    fi
    
    # Show final model list
    echo
    log::info "📋 Available models:"
    ollama list 2>/dev/null || log::warn "$MSG_LIST_MODELS_FAILED"
    
    return 0
}

#######################################
# List installed models
#######################################
ollama::list_models() {
    if ! ollama::is_healthy; then
        log::error "$MSG_OLLAMA_API_UNAVAILABLE"
        return 1
    fi
    
    log::header "📋 Installed Ollama Models"
    
    if system::is_command "ollama"; then
        ollama list
    else
        log::error "$MSG_OLLAMA_NOT_INSTALLED"
        return 1
    fi
}

# Export functions for use in tests and subshells
export -f ollama::get_model_info ollama::get_model_size ollama::is_model_known
export -f ollama::show_available_models ollama::calculate_default_size
export -f ollama::validate_model_list ollama::get_installed_models
export -f ollama::get_best_available_model ollama::validate_model_available
export -f ollama::parse_models ollama::pull_model ollama::install_models
export -f ollama::list_models