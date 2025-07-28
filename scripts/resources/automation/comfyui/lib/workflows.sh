#!/usr/bin/env bash
# ComfyUI Workflow Management
# Handles workflow import, export, and execution

#######################################
# Execute a workflow via API
#######################################
comfyui::execute_workflow() {
    log::header "🎨 Executing Workflow"
    
    # Check prerequisites
    if ! comfyui::check_ready; then
        return 1
    fi
    
    # Validate workflow file
    if [[ -z "${WORKFLOW_PATH:-}" ]]; then
        log::error "No workflow specified. Use --workflow <path>"
        return 1
    fi
    
    if [[ ! -f "$WORKFLOW_PATH" ]]; then
        log::error "Workflow file not found: $WORKFLOW_PATH"
        return 1
    fi
    
    # Read workflow content
    local workflow_content
    workflow_content=$(cat "$WORKFLOW_PATH") || {
        log::error "Failed to read workflow file"
        return 1
    }
    
    # Validate JSON
    if ! echo "$workflow_content" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in workflow file"
        return 1
    fi
    
    log::info "Workflow: $(basename "$WORKFLOW_PATH")"
    
    # Check for required models
    comfyui::check_workflow_models "$workflow_content"
    
    # Set output directory
    local output_dir="${OUTPUT_DIR:-${COMFYUI_DATA_DIR}/outputs}"
    mkdir -p "$output_dir"
    
    # Generate client ID if not provided
    local client_id="${CLIENT_ID:-comfyui-cli-$(date +%s)}"
    
    # Prepare API payload
    local payload
    payload=$(jq -n \
        --arg client_id "$client_id" \
        --argjson prompt "$workflow_content" \
        '{client_id: $client_id, prompt: $prompt}')
    
    # Submit workflow to API
    log::info "Submitting workflow to ComfyUI API..."
    
    local port="${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
    local api_url="http://localhost:$port/prompt"
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "$api_url") || {
        log::error "Failed to submit workflow to API"
        return 1
    }
    
    # Check response
    local prompt_id
    prompt_id=$(echo "$response" | jq -r '.prompt_id // empty') || {
        log::error "Invalid API response"
        echo "$response"
        return 1
    }
    
    if [[ -z "$prompt_id" ]]; then
        log::error "No prompt ID in response"
        echo "$response"
        
        # Check for common errors
        if echo "$response" | grep -q "required_models"; then
            log::error "Missing required models for this workflow"
            comfyui::show_missing_models "$response"
        fi
        
        return 1
    fi
    
    log::success "Workflow submitted successfully"
    log::info "Prompt ID: $prompt_id"
    
    # Monitor execution
    log::info "Monitoring workflow execution..."
    
    if comfyui::monitor_workflow "$prompt_id" "$client_id"; then
        # Retrieve outputs
        log::info "Retrieving generated images..."
        
        if comfyui::retrieve_outputs "$prompt_id" "$output_dir"; then
            log::success "✅ Workflow execution completed!"
            log::info "Outputs saved to: $output_dir"
            
            # List generated files
            echo
            log::header "📸 Generated Files"
            find "$output_dir" -name "*${prompt_id}*" -type f -printf "  - %f\n" 2>/dev/null | sort
            
            return 0
        else
            log::error "Failed to retrieve outputs"
            return 1
        fi
    else
        log::error "Workflow execution failed"
        return 1
    fi
}

#######################################
# Monitor workflow execution via WebSocket
#######################################
comfyui::monitor_workflow() {
    local prompt_id="$1"
    local client_id="$2"
    
    local port="${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
    local history_url="http://localhost:$port/history/$prompt_id"
    
    # Poll for completion (WebSocket monitoring would be better but requires additional tools)
    local max_wait=300  # 5 minutes
    local waited=0
    local check_interval=2
    
    while [[ $waited -lt $max_wait ]]; do
        # Check history endpoint
        local history
        history=$(curl -s "$history_url") || {
            log::error "Failed to query workflow status"
            return 1
        }
        
        # Check if prompt_id exists in history
        if echo "$history" | jq -e ".[\"$prompt_id\"]" >/dev/null 2>&1; then
            # Check status
            local status
            status=$(echo "$history" | jq -r ".[\"$prompt_id\"].status.status_str // empty")
            
            if [[ "$status" == "success" ]]; then
                log::success "Workflow completed successfully"
                return 0
            elif [[ "$status" == "error" ]]; then
                log::error "Workflow failed with error"
                
                # Show error details
                local error_msg
                error_msg=$(echo "$history" | jq -r ".[\"$prompt_id\"].status.messages // empty")
                if [[ -n "$error_msg" ]]; then
                    echo "$error_msg"
                fi
                
                return 1
            fi
        fi
        
        # Show progress
        echo -n "."
        
        sleep $check_interval
        waited=$((waited + check_interval))
    done
    
    echo
    log::error "Workflow execution timed out after ${max_wait} seconds"
    return 1
}

#######################################
# Import workflow file
#######################################
comfyui::import_workflow() {
    log::header "📥 Importing Workflow"
    
    if [[ -z "${WORKFLOW_PATH:-}" ]]; then
        log::error "No workflow specified. Use --workflow <path>"
        return 1
    fi
    
    if [[ ! -f "$WORKFLOW_PATH" ]]; then
        log::error "Workflow file not found: $WORKFLOW_PATH"
        return 1
    fi
    
    # Validate JSON
    if ! jq . "$WORKFLOW_PATH" >/dev/null 2>&1; then
        log::error "Invalid JSON in workflow file"
        return 1
    fi
    
    local filename
    filename=$(basename "$WORKFLOW_PATH")
    local target_path="${COMFYUI_DATA_DIR}/workflows/$filename"
    
    # Check if already exists
    if [[ -f "$target_path" ]] && [[ "$FORCE" != "yes" ]]; then
        log::warn "Workflow already exists: $filename"
        log::info "Use --force yes to overwrite"
        return 1
    fi
    
    # Create workflows directory if needed
    mkdir -p "${COMFYUI_DATA_DIR}/workflows"
    
    # Copy workflow
    if cp "$WORKFLOW_PATH" "$target_path"; then
        log::success "Workflow imported successfully"
        log::info "Location: $target_path"
        
        # Analyze workflow
        log::info "Analyzing workflow..."
        comfyui::analyze_workflow "$target_path"
        
        return 0
    else
        log::error "Failed to import workflow"
        return 1
    fi
}

#######################################
# Analyze workflow for requirements
#######################################
comfyui::analyze_workflow() {
    local workflow_path="$1"
    
    local workflow_content
    workflow_content=$(cat "$workflow_path")
    
    # Extract workflow information
    local node_count
    node_count=$(echo "$workflow_content" | jq 'length')
    
    echo
    log::info "Workflow Analysis:"
    log::info "  Nodes: $node_count"
    
    # Check for common node types
    local has_checkpoint=false
    local has_vae=false
    local has_lora=false
    local has_controlnet=false
    
    if echo "$workflow_content" | jq -r '.[].class_type // empty' | grep -q "CheckpointLoaderSimple"; then
        has_checkpoint=true
    fi
    
    if echo "$workflow_content" | jq -r '.[].class_type // empty' | grep -q "VAELoader"; then
        has_vae=true
    fi
    
    if echo "$workflow_content" | jq -r '.[].class_type // empty' | grep -q "LoraLoader"; then
        has_lora=true
    fi
    
    if echo "$workflow_content" | jq -r '.[].class_type // empty' | grep -q "ControlNet"; then
        has_controlnet=true
    fi
    
    log::info "  Uses checkpoint: $([ "$has_checkpoint" = true ] && echo "Yes" || echo "No")"
    log::info "  Uses VAE: $([ "$has_vae" = true ] && echo "Yes" || echo "No")"
    log::info "  Uses LoRA: $([ "$has_lora" = true ] && echo "Yes" || echo "No")"
    log::info "  Uses ControlNet: $([ "$has_controlnet" = true ] && echo "Yes" || echo "No")"
    
    # Check for specific models
    comfyui::check_workflow_models "$workflow_content"
}

#######################################
# Check workflow for required models
#######################################
comfyui::check_workflow_models() {
    local workflow_content="$1"
    
    echo
    log::info "Checking for required models..."
    
    local missing_models=()
    
    # Extract model references
    local models
    models=$(echo "$workflow_content" | jq -r '
        .. | 
        select(type == "object") | 
        select(.ckpt_name? or .vae_name? or .lora_name? or .control_net_name?) | 
        .ckpt_name // .vae_name // .lora_name // .control_net_name
    ' 2>/dev/null | sort -u)
    
    if [[ -z "$models" ]]; then
        log::info "No specific models referenced in workflow"
        return 0
    fi
    
    while IFS= read -r model; do
        if [[ -n "$model" ]]; then
            # Check if model exists
            local found=false
            
            # Check in various model directories
            for dir in checkpoints vae loras controlnet; do
                if [[ -f "${COMFYUI_DATA_DIR}/models/$dir/$model" ]]; then
                    found=true
                    break
                fi
            done
            
            if [[ "$found" == "true" ]]; then
                log::success "  ✅ $model"
            else
                log::warn "  ❌ $model (missing)"
                missing_models+=("$model")
            fi
        fi
    done <<< "$models"
    
    if [[ ${#missing_models[@]} -gt 0 ]]; then
        echo
        log::warn "Missing models detected!"
        log::info "The workflow may fail without these models."
        log::info "Download them to the appropriate directories under:"
        log::info "  ${COMFYUI_DATA_DIR}/models/"
    fi
}

#######################################
# Retrieve outputs from ComfyUI
#######################################
comfyui::retrieve_outputs() {
    local prompt_id="$1"
    local output_dir="$2"
    
    local port="${COMFYUI_CUSTOM_PORT:-$COMFYUI_DEFAULT_PORT}"
    local history_url="http://localhost:$port/history/$prompt_id"
    
    # Get output information from history
    local history
    history=$(curl -s "$history_url") || {
        log::error "Failed to retrieve history"
        return 1
    }
    
    # Extract output images
    local outputs
    outputs=$(echo "$history" | jq -r "
        .[\"$prompt_id\"].outputs // {} | 
        to_entries[] | 
        select(.value.images) | 
        .value.images[] | 
        .filename
    " 2>/dev/null)
    
    if [[ -z "$outputs" ]]; then
        log::warn "No output images found for this workflow"
        return 0
    fi
    
    local retrieved=0
    while IFS= read -r filename; do
        if [[ -n "$filename" ]]; then
            log::info "Retrieving: $filename"
            
            # Copy from container output directory
            local source_path="/workspace/ComfyUI/outputs/$filename"
            local target_path="$output_dir/$filename"
            
            if comfyui::copy_from_container "$source_path" "$target_path"; then
                retrieved=$((retrieved + 1))
                log::success "Retrieved: $filename"
            else
                log::error "Failed to retrieve: $filename"
            fi
        fi
    done <<< "$outputs"
    
    if [[ $retrieved -gt 0 ]]; then
        log::success "Retrieved $retrieved output(s)"
        return 0
    else
        log::error "Failed to retrieve any outputs"
        return 1
    fi
}

#######################################
# List available workflows
#######################################
comfyui::list_workflows() {
    log::header "📋 Available Workflows"
    
    local workflow_dir="${COMFYUI_DATA_DIR}/workflows"
    
    if [[ ! -d "$workflow_dir" ]] || [[ -z "$(ls -A "$workflow_dir" 2>/dev/null)" ]]; then
        log::warn "No workflows found"
        log::info "Import workflows with: $0 --action import-workflow --workflow <file>"
        return 0
    fi
    
    echo
    find "$workflow_dir" -name "*.json" -type f | while read -r workflow; do
        local name
        name=$(basename "$workflow")
        local size
        size=$(du -h "$workflow" | cut -f1)
        
        echo "  - $name ($size)"
        
        # Quick analysis
        local node_count
        node_count=$(jq 'length' "$workflow" 2>/dev/null || echo "?")
        echo "    Nodes: $node_count"
    done
    
    echo
    log::info "To execute a workflow:"
    log::info "  $0 --action execute-workflow --workflow <path>"
}

#######################################
# Show missing models from error response
#######################################
comfyui::show_missing_models() {
    local response="$1"
    
    echo
    log::header "❌ Missing Models"
    
    local missing
    missing=$(echo "$response" | jq -r '.error.required_models[]? // empty' 2>/dev/null)
    
    if [[ -n "$missing" ]]; then
        echo "$missing" | while read -r model; do
            log::error "  - $model"
        done
        
        echo
        log::info "Download these models to the appropriate directories:"
        log::info "  ${COMFYUI_DATA_DIR}/models/"
    fi
}