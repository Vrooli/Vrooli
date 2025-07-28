#!/usr/bin/env bash
# ComfyUI Model Management
# Handles downloading, listing, and managing AI models

#######################################
# Download default models (SDXL)
#######################################
comfyui::download_default_models() {
    log::header "📦 Downloading Default Models"
    
    if ! comfyui::check_ready; then
        return 1
    fi
    
    local total_models=${#COMFYUI_DEFAULT_MODELS[@]}
    local downloaded=0
    local failed=0
    
    # Download each model
    for i in "${!COMFYUI_DEFAULT_MODELS[@]}"; do
        local model_url="${COMFYUI_DEFAULT_MODELS[$i]}"
        local model_name="${COMFYUI_MODEL_NAMES[$i]}"
        local model_number=$((i + 1))
        
        log::info "[$model_number/$total_models] Downloading: $model_name"
        
        # Determine target directory based on model type
        local target_dir
        if [[ "$model_name" == *"vae"* ]]; then
            target_dir="${COMFYUI_DATA_DIR}/models/vae"
        else
            target_dir="${COMFYUI_DATA_DIR}/models/checkpoints"
        fi
        
        local target_path="$target_dir/$model_name"
        
        # Check if already exists and verify it
        if [[ -f "$target_path" ]]; then
            # Get expected metadata
            local expected_size="${COMFYUI_MODEL_SIZES[$i]}"
            local expected_sha256="${COMFYUI_MODEL_SHA256[$i]}"
            
            log::info "Model exists, verifying: $model_name"
            if comfyui::verify_model "$target_path" "$expected_size" "$expected_sha256"; then
                log::success "Model verified: $model_name"
                downloaded=$((downloaded + 1))
                continue
            else
                log::warn "Existing model failed verification, re-downloading: $model_name"
                rm -f "$target_path"
            fi
        fi
        
        # Get expected metadata
        local expected_size="${COMFYUI_MODEL_SIZES[$i]}"
        local expected_sha256="${COMFYUI_MODEL_SHA256[$i]}"
        
        # Download with progress and verification
        if comfyui::download_model "$model_url" "$target_path" "$expected_size" "$expected_sha256"; then
            downloaded=$((downloaded + 1))
            log::success "Downloaded and verified: $model_name"
        else
            failed=$((failed + 1))
            log::error "Failed to download: $model_name"
            log::info "You can manually download from: $model_url"
            log::info "Expected size: $(numfmt --to=iec-i --suffix=B "$expected_size" 2>/dev/null || echo "$expected_size bytes")"
        fi
        
        echo
    done
    
    # Summary
    log::header "📊 Download Summary"
    log::info "Total models: $total_models"
    log::info "Downloaded: $downloaded"
    log::info "Failed: $failed"
    
    if [[ $failed -eq 0 ]]; then
        log::success "All models downloaded successfully!"
        return 0
    elif [[ $downloaded -gt 0 ]]; then
        log::warn "Some models failed to download"
        return 1
    else
        log::error "No models were downloaded successfully"
        return 1
    fi
}

#######################################
# Verify model integrity
#######################################
comfyui::verify_model() {
    local model_path="$1"
    local expected_size="$2"
    local expected_sha256="$3"
    
    # Check if file exists
    if [[ ! -f "$model_path" ]]; then
        log::error "Model file not found: $model_path"
        return 1
    fi
    
    # Verify size
    local actual_size
    actual_size=$(stat -c%s "$model_path" 2>/dev/null || stat -f%z "$model_path" 2>/dev/null)
    
    if [[ -z "$actual_size" ]]; then
        log::warn "Could not determine file size for verification"
    elif [[ "$actual_size" != "$expected_size" ]]; then
        log::error "Model size mismatch!"
        log::error "Expected: $expected_size bytes ($(numfmt --to=iec-i --suffix=B "$expected_size" 2>/dev/null || echo "$expected_size bytes"))"
        log::error "Actual: $actual_size bytes ($(numfmt --to=iec-i --suffix=B "$actual_size" 2>/dev/null || echo "$actual_size bytes"))"
        return 1
    else
        log::success "Size verification passed"
    fi
    
    # Verify SHA256 if sha256sum is available
    if system::is_command "sha256sum" && [[ -n "$expected_sha256" ]]; then
        log::info "Verifying SHA256 checksum (this may take a moment)..."
        local actual_sha256
        actual_sha256=$(sha256sum "$model_path" | cut -d' ' -f1)
        
        if [[ "$actual_sha256" != "$expected_sha256" ]]; then
            log::error "SHA256 checksum mismatch!"
            log::error "Expected: $expected_sha256"
            log::error "Actual: $actual_sha256"
            return 1
        else
            log::success "SHA256 verification passed"
        fi
    else
        log::info "SHA256 verification skipped (sha256sum not available)"
    fi
    
    return 0
}

#######################################
# Download a single model with retry
#######################################
comfyui::download_model() {
    local url="$1"
    local target_path="$2"
    local expected_size="${3:-}"
    local expected_sha256="${4:-}"
    local max_retries=3
    local retry_count=0
    
    # Create target directory if needed
    local target_dir
    target_dir=$(dirname "$target_path")
    mkdir -p "$target_dir"
    
    # Download with retry logic
    while [[ $retry_count -lt $max_retries ]]; do
        log::info "Download attempt $((retry_count + 1)) of $max_retries"
        
        # Try wget first (supports resume)
        if system::is_command "wget"; then
            if wget --progress=bar:force --continue -O "$target_path" "$url" 2>&1; then
                # Verify if metadata provided
                if [[ -n "$expected_size" ]]; then
                    if comfyui::verify_model "$target_path" "$expected_size" "$expected_sha256"; then
                        return 0
                    else
                        log::warn "Model verification failed, retrying download..."
                        rm -f "$target_path"
                    fi
                else
                    return 0
                fi
            else
                log::warn "Download failed, will retry..."
            fi
        # Fall back to curl
        elif system::is_command "curl"; then
            if curl -L --progress-bar -C - -o "$target_path" "$url"; then
                # Verify if metadata provided
                if [[ -n "$expected_size" ]]; then
                    if comfyui::verify_model "$target_path" "$expected_size" "$expected_sha256"; then
                        return 0
                    else
                        log::warn "Model verification failed, retrying download..."
                        rm -f "$target_path"
                    fi
                else
                    return 0
                fi
            else
                log::warn "Download failed, will retry..."
            fi
        else
            log::error "Neither wget nor curl is available for downloading"
            return 1
        fi
        
        retry_count=$((retry_count + 1))
        if [[ $retry_count -lt $max_retries ]]; then
            log::info "Waiting 5 seconds before retry..."
            sleep 5
        fi
    done
    
    log::error "Failed to download model after $max_retries attempts"
    rm -f "$target_path"
    return 1
}

#######################################
# List installed models
#######################################
comfyui::list_models() {
    log::header "📚 Installed Models"
    
    if [[ ! -d "${COMFYUI_DATA_DIR}/models" ]]; then
        log::warn "Models directory does not exist"
        log::info "Run '$0 --action download-models' to download default models"
        return 1
    fi
    
    local total_size=0
    local total_count=0
    
    # Check each model type directory
    local model_types=(
        "checkpoints:Checkpoint Models"
        "vae:VAE Models"
        "loras:LoRA Models"
        "controlnet:ControlNet Models"
        "clip:CLIP Models"
        "embeddings:Embeddings"
        "upscale_models:Upscale Models"
    )
    
    for type_info in "${model_types[@]}"; do
        IFS=':' read -r dir_name display_name <<< "$type_info"
        local model_dir="${COMFYUI_DATA_DIR}/models/$dir_name"
        
        if [[ -d "$model_dir" ]] && [[ -n "$(ls -A "$model_dir" 2>/dev/null)" ]]; then
            echo
            log::info "=== $display_name ==="
            
            # List models with sizes
            while IFS= read -r model_file; do
                if [[ -f "$model_file" ]]; then
                    local size
                    size=$(du -h "$model_file" | cut -f1)
                    local name
                    name=$(basename "$model_file")
                    
                    echo "  - $name ($size)"
                    
                    # Add to total size (in bytes for calculation)
                    local size_bytes
                    size_bytes=$(stat -f%z "$model_file" 2>/dev/null || stat -c%s "$model_file" 2>/dev/null || echo 0)
                    total_size=$((total_size + size_bytes))
                    total_count=$((total_count + 1))
                fi
            done < <(find "$model_dir" -type f -name "*.safetensors" -o -name "*.ckpt" -o -name "*.pt" -o -name "*.pth" -o -name "*.bin" 2>/dev/null | sort)
        fi
    done
    
    if [[ $total_count -eq 0 ]]; then
        log::warn "No models found"
        log::info "Download models with: $0 --action download-models"
        return 1
    fi
    
    # Convert total size to human-readable
    local total_size_human
    if [[ $total_size -gt 1073741824 ]]; then
        total_size_human=$(awk "BEGIN {printf \"%.1fGB\", $total_size/1073741824}")
    else
        total_size_human=$(awk "BEGIN {printf \"%.1fMB\", $total_size/1048576}")
    fi
    
    echo
    log::header "📊 Summary"
    log::info "Total models: $total_count"
    log::info "Total size: $total_size_human"
    
    # Show popular model suggestions
    echo
    log::header "💡 Popular Models to Download"
    echo "To download additional models, place them in the appropriate directory:"
    echo
    echo "Checkpoints: ${COMFYUI_DATA_DIR}/models/checkpoints/"
    echo "  - SDXL: https://huggingface.co/stabilityai/stable-diffusion-xl-base-1.0"
    echo "  - SD 1.5: https://huggingface.co/runwayml/stable-diffusion-v1-5"
    echo
    echo "VAE: ${COMFYUI_DATA_DIR}/models/vae/"
    echo "  - SDXL VAE: https://huggingface.co/stabilityai/sdxl-vae"
    echo
    echo "ControlNet: ${COMFYUI_DATA_DIR}/models/controlnet/"
    echo "  - OpenPose: https://huggingface.co/lllyasviel/ControlNet"
    
    return 0
}

#######################################
# Download specific models by name
#######################################
comfyui::download_models() {
    log::header "📦 Model Download Manager"
    
    if ! comfyui::check_ready; then
        return 1
    fi
    
    # If no specific models requested, download defaults
    if [[ -z "${MODELS:-}" ]]; then
        comfyui::download_default_models
        return $?
    fi
    
    # Parse comma-separated model list
    IFS=',' read -ra model_list <<< "$MODELS"
    
    log::info "Downloading ${#model_list[@]} models..."
    
    local downloaded=0
    local failed=0
    
    for model in "${model_list[@]}"; do
        model=$(echo "$model" | xargs)  # Trim whitespace
        
        log::info "Processing: $model"
        
        # Check if it's a URL
        if [[ "$model" =~ ^https?:// ]]; then
            # Extract filename from URL
            local filename
            filename=$(basename "$model")
            
            # Determine target directory
            local target_dir="${COMFYUI_DATA_DIR}/models/checkpoints"
            if [[ "$filename" == *"vae"* ]]; then
                target_dir="${COMFYUI_DATA_DIR}/models/vae"
            elif [[ "$filename" == *"lora"* ]]; then
                target_dir="${COMFYUI_DATA_DIR}/models/loras"
            fi
            
            local target_path="$target_dir/$filename"
            
            if comfyui::download_model "$model" "$target_path"; then
                downloaded=$((downloaded + 1))
                log::success "Downloaded: $filename"
            else
                failed=$((failed + 1))
                log::error "Failed to download: $filename"
            fi
        else
            # Treat as a model name/identifier
            log::warn "Model name lookup not yet implemented: $model"
            log::info "Please provide full URL for now"
            failed=$((failed + 1))
        fi
    done
    
    # Summary
    echo
    log::info "Download complete: $downloaded succeeded, $failed failed"
    
    if [[ $failed -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Clean up old/unused models
#######################################
comfyui::cleanup_models() {
    log::header "🧹 Model Cleanup"
    
    log::warn "This feature is not yet implemented"
    log::info "To manually clean up models:"
    log::info "1. Navigate to: ${COMFYUI_DATA_DIR}/models/"
    log::info "2. Remove unwanted model files"
    log::info "3. Restart ComfyUI: $0 --action restart"
    
    return 0
}

#######################################
# Import models from a directory
#######################################
comfyui::import_models() {
    local source_dir="$1"
    
    if [[ ! -d "$source_dir" ]]; then
        log::error "Source directory does not exist: $source_dir"
        return 1
    fi
    
    log::info "Importing models from: $source_dir"
    
    # Find all model files
    local model_files=()
    while IFS= read -r -d '' file; do
        model_files+=("$file")
    done < <(find "$source_dir" -type f \( -name "*.safetensors" -o -name "*.ckpt" -o -name "*.pt" -o -name "*.pth" \) -print0)
    
    if [[ ${#model_files[@]} -eq 0 ]]; then
        log::warn "No model files found in source directory"
        return 1
    fi
    
    log::info "Found ${#model_files[@]} model files to import"
    
    local imported=0
    for model_file in "${model_files[@]}"; do
        local filename
        filename=$(basename "$model_file")
        
        # Determine target directory
        local target_dir="${COMFYUI_DATA_DIR}/models/checkpoints"
        if [[ "$filename" == *"vae"* ]]; then
            target_dir="${COMFYUI_DATA_DIR}/models/vae"
        elif [[ "$filename" == *"lora"* ]]; then
            target_dir="${COMFYUI_DATA_DIR}/models/loras"
        fi
        
        local target_path="$target_dir/$filename"
        
        # Check if already exists
        if [[ -f "$target_path" ]]; then
            log::warn "Model already exists, skipping: $filename"
            continue
        fi
        
        # Copy model
        log::info "Importing: $filename"
        if cp "$model_file" "$target_path"; then
            imported=$((imported + 1))
            log::success "Imported: $filename"
        else
            log::error "Failed to import: $filename"
        fi
    done
    
    log::info "Import complete: $imported models imported"
    return 0
}