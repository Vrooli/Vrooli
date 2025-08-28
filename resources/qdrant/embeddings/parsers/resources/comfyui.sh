#!/usr/bin/env bash
# ComfyUI Workflow Parser for Qdrant Embeddings
# Extracts semantic information from ComfyUI workflow JSON files
#
# Handles:
# - Workflow node definitions and connections
# - Model loading and usage patterns
# - Image generation pipelines
# - Sampling and conditioning parameters
# - Input/output configurations

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract workflow metadata
# 
# Gets basic workflow information
#
# Arguments:
#   $1 - Path to ComfyUI workflow JSON file
# Returns: JSON with workflow metadata
#######################################
extractor::lib::comfyui::extract_metadata() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Validate JSON format
    if ! jq empty "$file" 2>/dev/null; then
        log::debug "Invalid JSON format in ComfyUI workflow: $file" >&2
        return 1
    fi
    
    # Extract basic metadata
    local workflow_name=$(jq -r '.name // ""' "$file" 2>/dev/null)
    local version=$(jq -r '.version // .comfyui_version // ""' "$file" 2>/dev/null)
    local description=$(jq -r '.description // ""' "$file" 2>/dev/null)
    
    # Count nodes
    local node_count=0
    local connection_count=0
    
    # Check if it's the newer format (nodes array) or older format (nodes object)
    if jq -e '.workflow.nodes | type == "array"' "$file" >/dev/null 2>/dev/null; then
        node_count=$(jq '.workflow.nodes | length' "$file" 2>/dev/null || echo "0")
    elif jq -e '.workflow.nodes | type == "object"' "$file" >/dev/null 2>/dev/null; then
        node_count=$(jq '.workflow.nodes | keys | length' "$file" 2>/dev/null || echo "0")
    elif jq -e '.nodes | type == "object"' "$file" >/dev/null 2>/dev/null; then
        node_count=$(jq '.nodes | keys | length' "$file" 2>/dev/null || echo "0")
    fi
    
    # Count connections/links
    if jq -e '.workflow.links' "$file" >/dev/null 2>/dev/null; then
        connection_count=$(jq '.workflow.links | length' "$file" 2>/dev/null || echo "0")
    elif jq -e '.links' "$file" >/dev/null 2>/dev/null; then
        connection_count=$(jq '.links | length' "$file" 2>/dev/null || echo "0")
    fi
    
    # Extract last modified or creation info if available
    local last_modified=$(jq -r '.last_modified // .modified // ""' "$file" 2>/dev/null)
    
    jq -n \
        --arg name "$workflow_name" \
        --arg version "$version" \
        --arg description "$description" \
        --arg node_count "$node_count" \
        --arg connection_count "$connection_count" \
        --arg last_modified "$last_modified" \
        '{
            workflow_name: $name,
            version: $version,
            description: $description,
            node_count: ($node_count | tonumber),
            connection_count: ($connection_count | tonumber),
            last_modified: $last_modified
        }'
}

#######################################
# Extract node information
# 
# Analyzes workflow nodes and their types
#
# Arguments:
#   $1 - Path to ComfyUI workflow JSON file
# Returns: JSON with node analysis
#######################################
extractor::lib::comfyui::extract_nodes() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local node_types=()
    local model_nodes=()
    local generation_nodes=()
    
    # Extract node types from different possible structures
    local nodes_data=""
    if jq -e '.workflow.nodes' "$file" >/dev/null 2>/dev/null; then
        nodes_data=$(jq -c '.workflow.nodes' "$file")
    elif jq -e '.nodes' "$file" >/dev/null 2>/dev/null; then
        nodes_data=$(jq -c '.nodes' "$file")
    fi
    
    if [[ -n "$nodes_data" && "$nodes_data" != "null" ]]; then
        # Handle both array and object node formats
        if echo "$nodes_data" | jq -e 'type == "array"' >/dev/null 2>/dev/null; then
            # Array format
            while IFS= read -r node; do
                [[ -z "$node" ]] && continue
                local node_type=$(echo "$node" | jq -r '.type // .class_type // ""')
                [[ -n "$node_type" ]] && node_types+=("$node_type")
            done < <(echo "$nodes_data" | jq -c '.[]')
        else
            # Object format
            while IFS= read -r node; do
                [[ -z "$node" ]] && continue
                local node_type=$(echo "$node" | jq -r '.type // .class_type // ""')
                [[ -n "$node_type" ]] && node_types+=("$node_type")
            done < <(echo "$nodes_data" | jq -c '.[]')
        fi
    fi
    
    # Categorize nodes
    for node_type in "${node_types[@]}"; do
        case "$node_type" in
            *"CheckpointLoader"*|*"ModelLoader"*|*"LoraLoader"*|*"VAELoader"*)
                model_nodes+=("$node_type")
                ;;
            *"KSampler"*|*"Generate"*|*"Sample"*|*"Denoise"*)
                generation_nodes+=("$node_type")
                ;;
        esac
    done
    
    # Get unique node types
    local unique_types=($(printf '%s\n' "${node_types[@]}" | sort -u))
    local unique_types_json=$(printf '%s\n' "${unique_types[@]}" | jq -R . | jq -s '.')
    
    # Count specific node categories
    local checkpoint_loaders=$(printf '%s\n' "${node_types[@]}" | grep -c "CheckpointLoader" || echo "0")
    local samplers=$(printf '%s\n' "${node_types[@]}" | grep -c -E "(KSampler|Sample)" || echo "0")
    local conditioning=$(printf '%s\n' "${node_types[@]}" | grep -c -E "(CLIP|Conditioning)" || echo "0")
    
    jq -n \
        --argjson node_types "$unique_types_json" \
        --arg checkpoint_count "$checkpoint_loaders" \
        --arg sampler_count "$samplers" \
        --arg conditioning_count "$conditioning" \
        '{
            node_types: $node_types,
            checkpoint_loaders: ($checkpoint_count | tonumber),
            samplers: ($sampler_count | tonumber),
            conditioning_nodes: ($conditioning_count | tonumber)
        }'
}

#######################################
# Extract model information
# 
# Identifies models and checkpoints used
#
# Arguments:
#   $1 - Path to ComfyUI workflow JSON file
# Returns: JSON with model information
#######################################
extractor::lib::comfyui::extract_models() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local models=()
    local loras=()
    local vaes=()
    local embeddings=()
    
    # Extract model references from the workflow
    # This searches through all node inputs for model file references
    while IFS= read -r model_ref; do
        [[ -z "$model_ref" ]] && continue
        
        # Categorize by file extension or naming pattern
        if [[ "$model_ref" == *.safetensors ]] || [[ "$model_ref" == *.ckpt ]] || [[ "$model_ref" == *.pth ]]; then
            if [[ "$model_ref" == *"lora"* ]] || [[ "$model_ref" == *"LoRA"* ]]; then
                loras+=("$model_ref")
            elif [[ "$model_ref" == *"vae"* ]] || [[ "$model_ref" == *"VAE"* ]]; then
                vaes+=("$model_ref")
            elif [[ "$model_ref" == *"embedding"* ]] || [[ "$model_ref" == *"textual"* ]]; then
                embeddings+=("$model_ref")
            else
                models+=("$model_ref")
            fi
        fi
    done < <(jq -r '.. | .ckpt_name? // .model_name? // .lora_name? // .vae_name? // empty' "$file" 2>/dev/null | sort -u)
    
    # Convert to JSON arrays
    local models_json="[]"
    local loras_json="[]"
    local vaes_json="[]"
    local embeddings_json="[]"
    
    [[ ${#models[@]} -gt 0 ]] && models_json=$(printf '%s\n' "${models[@]}" | jq -R . | jq -s '.')
    [[ ${#loras[@]} -gt 0 ]] && loras_json=$(printf '%s\n' "${loras[@]}" | jq -R . | jq -s '.')
    [[ ${#vaes[@]} -gt 0 ]] && vaes_json=$(printf '%s\n' "${vaes[@]}" | jq -R . | jq -s '.')
    [[ ${#embeddings[@]} -gt 0 ]] && embeddings_json=$(printf '%s\n' "${embeddings[@]}" | jq -R . | jq -s '.')
    
    jq -n \
        --argjson models "$models_json" \
        --argjson loras "$loras_json" \
        --argjson vaes "$vaes_json" \
        --argjson embeddings "$embeddings_json" \
        '{
            base_models: $models,
            lora_models: $loras,
            vae_models: $vaes,
            embeddings: $embeddings,
            total_models: (($models | length) + ($loras | length) + ($vaes | length) + ($embeddings | length))
        }'
}

#######################################
# Extract generation parameters
# 
# Analyzes sampling and generation settings
#
# Arguments:
#   $1 - Path to ComfyUI workflow JSON file
# Returns: JSON with generation parameters
#######################################
extractor::lib::comfyui::extract_generation_params() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract common generation parameters
    local width=$(jq -r '.. | .width? // empty' "$file" 2>/dev/null | head -1 || echo "")
    local height=$(jq -r '.. | .height? // empty' "$file" 2>/dev/null | head -1 || echo "")
    local steps=$(jq -r '.. | .steps? // empty' "$file" 2>/dev/null | head -1 || echo "")
    local cfg_scale=$(jq -r '.. | .cfg? // empty' "$file" 2>/dev/null | head -1 || echo "")
    local sampler_name=$(jq -r '.. | .sampler_name? // empty' "$file" 2>/dev/null | head -1 || echo "")
    local scheduler=$(jq -r '.. | .scheduler? // empty' "$file" 2>/dev/null | head -1 || echo "")
    local denoise=$(jq -r '.. | .denoise? // empty' "$file" 2>/dev/null | head -1 || echo "")
    
    # Check for batch settings
    local batch_size=$(jq -r '.. | .batch_size? // empty' "$file" 2>/dev/null | head -1 || echo "1")
    
    # Check for seed settings
    local has_seed="false"
    if jq -e '.. | .seed?' "$file" >/dev/null 2>/dev/null; then
        has_seed="true"
    fi
    
    # Check for ControlNet usage
    local has_controlnet="false"
    if jq -r '.. | .class_type? // empty' "$file" 2>/dev/null | grep -qi "controlnet"; then
        has_controlnet="true"
    fi
    
    # Check for upscaling
    local has_upscale="false"
    if jq -r '.. | .class_type? // .type? // empty' "$file" 2>/dev/null | grep -qiE "(upscale|esrgan|realesrgan)"; then
        has_upscale="true"
    fi
    
    jq -n \
        --arg width "$width" \
        --arg height "$height" \
        --arg steps "$steps" \
        --arg cfg "$cfg_scale" \
        --arg sampler "$sampler_name" \
        --arg scheduler "$scheduler" \
        --arg denoise "$denoise" \
        --arg batch "$batch_size" \
        --arg has_seed "$has_seed" \
        --arg controlnet "$has_controlnet" \
        --arg upscale "$has_upscale" \
        '{
            width: (if $width != "" then ($width | tonumber) else null end),
            height: (if $height != "" then ($height | tonumber) else null end),
            steps: (if $steps != "" then ($steps | tonumber) else null end),
            cfg_scale: (if $cfg != "" then ($cfg | tonumber) else null end),
            sampler_name: $sampler,
            scheduler: $scheduler,
            denoise: (if $denoise != "" then ($denoise | tonumber) else null end),
            batch_size: ($batch | tonumber),
            has_seed_control: ($has_seed == "true"),
            has_controlnet: ($controlnet == "true"),
            has_upscaling: ($upscale == "true")
        }'
}

#######################################
# Analyze workflow purpose
# 
# Determines workflow functionality based on nodes and structure
#
# Arguments:
#   $1 - Path to ComfyUI workflow JSON file
# Returns: JSON with purpose analysis
#######################################
extractor::lib::comfyui::analyze_purpose() {
    local file="$1"
    local purposes=()
    
    if [[ ! -f "$file" ]]; then
        echo '{"purposes": [], "primary_purpose": "unknown"}'
        return
    fi
    
    local filename=$(basename "$file" | tr '[:upper:]' '[:lower:]')
    local content=$(cat "$file" 2>/dev/null | tr '[:upper:]' '[:lower:]')
    
    # Analyze filename for hints
    if [[ "$filename" == *"txt2img"* ]] || [[ "$filename" == *"text2image"* ]]; then
        purposes+=("text_to_image")
    elif [[ "$filename" == *"img2img"* ]] || [[ "$filename" == *"image2image"* ]]; then
        purposes+=("image_to_image")
    elif [[ "$filename" == *"inpaint"* ]]; then
        purposes+=("inpainting")
    elif [[ "$filename" == *"upscale"* ]]; then
        purposes+=("upscaling")
    elif [[ "$filename" == *"controlnet"* ]]; then
        purposes+=("controlled_generation")
    elif [[ "$filename" == *"batch"* ]]; then
        purposes+=("batch_processing")
    elif [[ "$filename" == *"animation"* ]] || [[ "$filename" == *"video"* ]]; then
        purposes+=("animation_generation")
    fi
    
    # Analyze content for patterns
    if echo "$content" | grep -qi "controlnet"; then
        purposes+=("controlled_generation")
    fi
    
    if echo "$content" | grep -qi "upscale"; then
        purposes+=("image_enhancement")
    fi
    
    if echo "$content" | grep -qi "inpaint"; then
        purposes+=("image_editing")
    fi
    
    if echo "$content" | grep -qi "lora"; then
        purposes+=("style_transfer")
    fi
    
    if echo "$content" | grep -qiE "(batch|multiple)"; then
        purposes+=("batch_generation")
    fi
    
    if echo "$content" | grep -qiE "(face|portrait)"; then
        purposes+=("portrait_generation")
    fi
    
    if echo "$content" | grep -qiE "(landscape|scenic)"; then
        purposes+=("landscape_generation")
    fi
    
    # Check for advanced techniques
    if echo "$content" | grep -qiE "(hires|highres)"; then
        purposes+=("high_resolution")
    fi
    
    if echo "$content" | grep -qiE "(regional|area)"; then
        purposes+=("regional_prompting")
    fi
    
    # Determine primary purpose
    local primary_purpose="image_generation"
    if [[ ${#purposes[@]} -gt 0 ]]; then
        primary_purpose="${purposes[0]}"
    fi
    
    local purposes_json=$(printf '%s\n' "${purposes[@]}" | sort -u | jq -R . | jq -s '.')
    
    jq -n \
        --argjson purposes "$purposes_json" \
        --arg primary "$primary_purpose" \
        '{
            purposes: $purposes,
            primary_purpose: $primary
        }'
}

#######################################
# Extract all ComfyUI workflow information
# 
# Main extraction function that combines all analyses
#
# Arguments:
#   $1 - ComfyUI workflow file path or directory
#   $2 - Component type (workflow, generation, etc.)
#   $3 - Resource name
# Returns: JSON lines with all workflow information
#######################################
extractor::lib::comfyui::extract_all() {
    local path="$1"
    local component_type="${2:-workflow}"
    local resource_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        local file="$path"
        local filename=$(basename "$file")
        local file_ext="${filename##*.}"
        
        # Check if it's a JSON file
        case "$file_ext" in
            json)
                ;;
            *)
                return 1
                ;;
        esac
        
        # Get file statistics
        local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
        
        # Extract all components
        local metadata=$(extractor::lib::comfyui::extract_metadata "$file")
        local nodes=$(extractor::lib::comfyui::extract_nodes "$file")
        local models=$(extractor::lib::comfyui::extract_models "$file")
        local gen_params=$(extractor::lib::comfyui::extract_generation_params "$file")
        local purpose=$(extractor::lib::comfyui::analyze_purpose "$file")
        
        # Get key metrics
        local workflow_name=$(echo "$metadata" | jq -r '.workflow_name')
        [[ "$workflow_name" == "" || "$workflow_name" == "null" ]] && workflow_name="$filename"
        
        local node_count=$(echo "$metadata" | jq -r '.node_count')
        local primary_purpose=$(echo "$purpose" | jq -r '.primary_purpose')
        local model_count=$(echo "$models" | jq -r '.total_models')
        local resolution="${width}x${height}"
        
        local width=$(echo "$gen_params" | jq -r '.width // "unknown"')
        local height=$(echo "$gen_params" | jq -r '.height // "unknown"')
        if [[ "$width" != "null" && "$height" != "null" && "$width" != "unknown" && "$height" != "unknown" ]]; then
            resolution="${width}x${height}"
        else
            resolution="unknown"
        fi
        
        # Build content summary
        local content="ComfyUI Workflow: $workflow_name | Type: $component_type | Resource: $resource_name"
        content="$content | Purpose: $primary_purpose | Nodes: $node_count"
        [[ $model_count -gt 0 ]] && content="$content | Models: $model_count"
        [[ "$resolution" != "unknown" ]] && content="$content | Resolution: $resolution"
        
        # Check for special features
        local has_controlnet=$(echo "$gen_params" | jq -r '.has_controlnet')
        [[ "$has_controlnet" == "true" ]] && content="$content | Uses ControlNet"
        
        local has_upscaling=$(echo "$gen_params" | jq -r '.has_upscaling')
        [[ "$has_upscaling" == "true" ]] && content="$content | Has Upscaling"
        
        # Output comprehensive workflow analysis
        jq -n \
            --arg content "$content" \
            --arg resource "$resource_name" \
            --arg source_file "$file" \
            --arg filename "$filename" \
            --arg component_type "$component_type" \
            --arg file_size "$file_size" \
            --argjson metadata "$metadata" \
            --argjson nodes "$nodes" \
            --argjson models "$models" \
            --argjson gen_params "$gen_params" \
            --argjson purpose "$purpose" \
            '{
                content: $content,
                metadata: {
                    resource: $resource,
                    source_file: $source_file,
                    filename: $filename,
                    component_type: $component_type,
                    workflow_type: "comfyui",
                    file_size: ($file_size | tonumber),
                    workflow_metadata: $metadata,
                    nodes: $nodes,
                    models: $models,
                    generation_params: $gen_params,
                    purpose: $purpose,
                    content_type: "comfyui_workflow",
                    extraction_method: "comfyui_parser"
                }
            }' | jq -c
            
        # Output entries for each model used (for better searchability)
        echo "$models" | jq -r '.base_models[]?' 2>/dev/null | while read -r model; do
            [[ -z "$model" || "$model" == "null" ]] && continue
            
            local model_content="ComfyUI Model: $model in $workflow_name | Resource: $resource_name"
            
            jq -n \
                --arg content "$model_content" \
                --arg resource "$resource_name" \
                --arg source_file "$file" \
                --arg workflow_name "$workflow_name" \
                --arg model_name "$model" \
                --arg component_type "$component_type" \
                '{
                    content: $content,
                    metadata: {
                        resource: $resource,
                        source_file: $source_file,
                        workflow_name: $workflow_name,
                        model_name: $model_name,
                        model_type: "base_model",
                        component_type: $component_type,
                        content_type: "comfyui_model",
                        extraction_method: "comfyui_parser"
                    }
                }' | jq -c
        done
        
    elif [[ -d "$path" ]]; then
        # Directory - find all ComfyUI workflow files
        local workflow_files=()
        while IFS= read -r file; do
            workflow_files+=("$file")
        done < <(find "$path" -type f -name "*.json" 2>/dev/null)
        
        if [[ ${#workflow_files[@]} -eq 0 ]]; then
            return 1
        fi
        
        for file in "${workflow_files[@]}"; do
            # Check if it's likely a ComfyUI workflow
            if jq -e '.workflow.nodes or .nodes' "$file" >/dev/null 2>/dev/null; then
                extractor::lib::comfyui::extract_all "$file" "$component_type" "$resource_name"
            fi
        done
    fi
}

#######################################
# Check if file is a ComfyUI workflow
# 
# Validates if JSON file is a ComfyUI workflow definition
#
# Arguments:
#   $1 - File path
# Returns: 0 if ComfyUI workflow, 1 otherwise
#######################################
extractor::lib::comfyui::is_workflow() {
    local file="$1"
    
    if [[ ! -f "$file" ]] || [[ ! "$file" == *.json ]]; then
        return 1
    fi
    
    # Check for ComfyUI workflow structure
    if jq -e '.workflow.nodes or .nodes' "$file" >/dev/null 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Export all functions
export -f extractor::lib::comfyui::extract_metadata
export -f extractor::lib::comfyui::extract_nodes
export -f extractor::lib::comfyui::extract_models
export -f extractor::lib::comfyui::extract_generation_params
export -f extractor::lib::comfyui::analyze_purpose
export -f extractor::lib::comfyui::extract_all
export -f extractor::lib::comfyui::is_workflow