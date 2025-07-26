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
        
        # Check if already exists
        if [[ -f "$target_path" ]]; then
            log::info "Model already exists, skipping: $model_name"
            downloaded=$((downloaded + 1))
            continue
        fi
        
        # Download with progress
        if comfyui::download_model "$model_url" "$target_path"; then
            downloaded=$((downloaded + 1))
            log::success "Downloaded: $model_name"
        else
            failed=$((failed + 1))
            log::error "Failed to download: $model_name"
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
# Download a single model
#######################################
comfyui::download_model() {
    local url="$1"
    local target_path="$2"
    
    # Create target directory if needed
    local target_dir
    target_dir=$(dirname "$target_path")
    mkdir -p "$target_dir"
    
    # Try wget first (better progress display)
    if system::is_command "wget"; then
        if wget --progress=bar:force -O "$target_path" "$url" 2>&1; then
            return 0
        else
            rm -f "$target_path"
            return 1
        fi
    fi
    
    # Fall back to curl
    if system::is_command "curl"; then
        if curl -L --progress-bar -o "$target_path" "$url"; then
            return 0
        else
            rm -f "$target_path"
            return 1
        fi
    fi
    
    log::error "Neither wget nor curl is available for downloading"
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
    if [[ -z "$MODELS" ]]; then
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