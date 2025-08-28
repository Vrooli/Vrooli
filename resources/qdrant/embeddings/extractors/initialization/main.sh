#!/usr/bin/env bash
# Initialization System Orchestrator for Qdrant Embeddings
# Reads service.json to discover initialization files and dispatches to appropriate parsers
#
# This replaces the old workflows.sh with a more flexible, declarative approach
# that supports multiple initialization technologies

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Define paths
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
PARSERS_DIR="${EMBEDDINGS_DIR}/parsers/resources"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source unified embedding service
source "${EMBEDDINGS_DIR}/lib/embedding-service.sh"

# Source all resource parsers
source "${PARSERS_DIR}/n8n.sh"
source "${PARSERS_DIR}/windmill.sh"
source "${PARSERS_DIR}/postgres.sh"
source "${PARSERS_DIR}/qdrant.sh"
source "${PARSERS_DIR}/minio.sh"
source "${PARSERS_DIR}/comfyui.sh"
source "${PARSERS_DIR}/node-red.sh"
source "${PARSERS_DIR}/vault.sh"

#######################################
# Read service.json and extract initialization configs
# 
# Supports both categorized and flat resource structures:
# - resources.category.resource_name.initialization
# - resources.resource_name.initialization
#
# Arguments:
#   $1 - Path to service.json (optional, defaults to ./.vrooli/service.json)
# Returns: JSON array of initialization items
#######################################
qdrant::init::read_service_config() {
    local service_file="${1:-./.vrooli/service.json}"
    
    if [[ ! -f "$service_file" ]]; then
        log::error "Service file not found: $service_file"
        echo "[]"
        return 1
    fi
    
    # Validate JSON format
    if ! jq empty "$service_file" 2>/dev/null; then
        log::error "Invalid JSON format in service file: $service_file"
        echo "[]"
        return 1
    fi
    
    # Extract all initialization arrays from resources
    # This handles both nested (category.resource) and flat (resource) structures
    local init_items=$(jq -r '
        .resources | 
        to_entries | 
        map(
            # Check if value has initialization directly (flat structure)
            if .value.initialization then
                {
                    resource: .key,
                    category: null,
                    items: .value.initialization
                }
            # Check if value has nested resources (categorized structure)
            elif .value | type == "object" then
                .value | to_entries | map(
                    if .value.initialization then
                        {
                            resource: .key,
                            category: .key,
                            items: .value.initialization
                        }
                    else
                        empty
                    end
                )
            else
                empty
            end
        ) | 
        flatten | 
        map(.items[] + {resource: .resource, category: .category}) |
        map(select(.path != null))
    ' "$service_file" 2>/dev/null || echo "[]")
    
    echo "$init_items"
}

#######################################
# Dispatch initialization file to appropriate parser
# 
# Routes files to the correct parser based on type field
#
# Arguments:
#   $1 - File path
#   $2 - Type (n8n, windmill, postgres, etc.)
#   $3 - Purpose (optional)
#   $4 - Resource name
# Returns: JSON lines from parser
#######################################
qdrant::init::dispatch_to_parser() {
    local file="$1"
    local type="$2"
    local purpose="${3:-unknown}"
    local resource="${4:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        log::warn "Initialization file not found: $file"
        return 1
    fi
    
    log::debug "Processing $type file: $file (purpose: $purpose, resource: $resource)"
    
    case "$type" in
        n8n)
            extractor::lib::n8n::extract_all "$file" "initialization" "$resource"
            ;;
        windmill)
            extractor::lib::windmill::extract_all "$file" "initialization" "$resource"
            ;;
        postgres|postgresql|sql)
            extractor::lib::postgres::extract_all "$file" "initialization" "$resource"
            ;;
        qdrant)
            extractor::lib::qdrant::extract_all "$file" "initialization" "$resource"
            ;;
        minio|s3)
            extractor::lib::minio::extract_all "$file" "initialization" "$resource"
            ;;
        comfyui)
            extractor::lib::comfyui::extract_all "$file" "initialization" "$resource"
            ;;
        node-red|node_red|nodered)
            extractor::lib::node_red::extract_all "$file" "initialization" "$resource"
            ;;
        vault|hashicorp_vault)
            extractor::lib::vault::extract_all "$file" "initialization" "$resource"
            ;;
        docker|docker-compose)
            # Future: Add docker parser
            log::debug "Docker parser not yet implemented for: $file"
            ;;
        terraform|tf)
            # Future: Add terraform parser
            log::debug "Terraform parser not yet implemented for: $file"
            ;;
        kubernetes|k8s)
            # Future: Add kubernetes parser
            log::debug "Kubernetes parser not yet implemented for: $file"
            ;;
        ansible)
            # Future: Add ansible parser
            log::debug "Ansible parser not yet implemented for: $file"
            ;;
        environment|env)
            # Future: Add environment parser
            log::debug "Environment parser not yet implemented for: $file"
            ;;
        *)
            log::warn "Unknown initialization type '$type' for file: $file"
            # Attempt to infer type from file extension
            local file_ext="${file##*.}"
            case "$file_ext" in
                json)
                    # Check if it's an n8n workflow
                    if extractor::lib::n8n::is_workflow "$file"; then
                        log::debug "Detected as n8n workflow based on content"
                        extractor::lib::n8n::extract_all "$file" "initialization" "$resource"
                    elif extractor::lib::qdrant::is_collection_config "$file"; then
                        log::debug "Detected as Qdrant collection config based on content"
                        extractor::lib::qdrant::extract_all "$file" "initialization" "$resource"
                    elif extractor::lib::minio::is_bucket_config "$file"; then
                        log::debug "Detected as MinIO bucket config based on content"
                        extractor::lib::minio::extract_all "$file" "initialization" "$resource"
                    elif extractor::lib::comfyui::is_workflow "$file"; then
                        log::debug "Detected as ComfyUI workflow based on content"
                        extractor::lib::comfyui::extract_all "$file" "initialization" "$resource"
                    elif extractor::lib::node_red::is_flow "$file"; then
                        log::debug "Detected as Node-RED flow based on content"
                        extractor::lib::node_red::extract_all "$file" "initialization" "$resource"
                    fi
                    ;;
                yaml|yml)
                    # Check if it's a Windmill flow
                    if extractor::lib::windmill::is_flow "$file"; then
                        log::debug "Detected as Windmill flow based on content"
                        extractor::lib::windmill::extract_all "$file" "initialization" "$resource"
                    fi
                    ;;
                hcl|conf|policy)
                    if extractor::lib::vault::is_vault_config "$file"; then
                        log::debug "Detected as Vault config based on content"
                        extractor::lib::vault::extract_all "$file" "initialization" "$resource"
                    fi
                    ;;
                sql|ddl|dml)
                    log::debug "Detected as SQL file based on extension"
                    extractor::lib::postgres::extract_all "$file" "initialization" "$resource"
                    ;;
                *)
                    log::warn "Unable to determine parser for file: $file"
                    ;;
            esac
            ;;
    esac
}

#######################################
# Process all initialization files
# 
# Main processing function that reads config and extracts all files
#
# Arguments:
#   $1 - Service.json path (optional)
#   $2 - Output file for JSON lines
# Returns: Number of files processed
#######################################
qdrant::init::process_all() {
    local service_file="${1:-./.vrooli/service.json}"
    local output_file="${2:-}"
    
    if [[ -z "$output_file" ]]; then
        output_file="${TEMP_DIR:-/tmp}/initialization.jsonl"
    fi
    
    # Clear output file
    > "$output_file"
    
    # Read initialization configs from service.json
    local init_items=$(qdrant::init::read_service_config "$service_file")
    
    if [[ -z "$init_items" ]] || [[ "$init_items" == "[]" ]]; then
        log::warn "No initialization items found in service.json"
        echo "0"
        return 0
    fi
    
    local count=0
    local total=$(echo "$init_items" | jq 'length')
    
    log::info "Processing $total initialization files"
    
    # Process each initialization item
    echo "$init_items" | jq -c '.[]' | while IFS= read -r item; do
        local path=$(echo "$item" | jq -r '.path // ""')
        local type=$(echo "$item" | jq -r '.type // ""')
        local purpose=$(echo "$item" | jq -r '.purpose // "unknown"')
        local resource=$(echo "$item" | jq -r '.resource // "unknown"')
        local stage=$(echo "$item" | jq -r '.stage // ""')
        local order=$(echo "$item" | jq -r '.order // ""')
        local environment=$(echo "$item" | jq -r '.environment // ""')
        
        if [[ -z "$path" ]]; then
            log::warn "Initialization item missing path"
            continue
        fi
        
        # Resolve path relative to APP_ROOT
        if [[ ! "$path" = /* ]]; then
            path="${APP_ROOT}/${path}"
        fi
        
        # Add metadata to extraction
        local extraction=$(qdrant::init::dispatch_to_parser "$path" "$type" "$purpose" "$resource")
        
        if [[ -n "$extraction" ]]; then
            # Enhance extracted data with initialization metadata
            echo "$extraction" | while IFS= read -r json_line; do
                if [[ -n "$json_line" ]]; then
                    # Add initialization-specific metadata
                    echo "$json_line" | jq -c \
                        --arg stage "$stage" \
                        --arg order "$order" \
                        --arg environment "$environment" \
                        --arg purpose "$purpose" \
                        '.metadata.initialization = {
                            stage: $stage,
                            order: $order,
                            environment: $environment,
                            purpose: $purpose
                        }' >> "$output_file"
                    ((count++))
                fi
            done
        fi
        
        # Progress indicator
        log::debug "Processed initialization file: $path ($type)"
    done
    
    log::success "Extracted content from $count initialization entries"
    echo "$count"
}

#######################################
# Process initialization files for embeddings
# 
# Integration with unified embedding service
#
# Arguments:
#   $1 - App ID
# Returns: Number of initialization files processed
#######################################
qdrant::embeddings::process_initialization() {
    local app_id="$1"
    local collection="${app_id}-initialization"
    local count=0
    
    # Extract initialization data to temp file
    local output_file="${TEMP_DIR:-/tmp}/initialization.jsonl"
    local extracted_count=$(qdrant::init::process_all "" "$output_file")
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No initialization files found for processing"
        echo "0"
        return 0
    fi
    
    # Process each JSON line through unified embedding service
    while IFS= read -r json_line; do
        if [[ -n "$json_line" ]]; then
            # Parse JSON to extract content and metadata
            local content
            content=$(echo "$json_line" | jq -r '.content // empty' 2>/dev/null)
            
            local metadata
            metadata=$(echo "$json_line" | jq -c '.metadata // {}' 2>/dev/null)
            
            if [[ -n "$content" ]]; then
                # Process through unified embedding service with structured metadata
                if qdrant::embedding::process_item "$content" "initialization" "$collection" "$app_id" "$metadata"; then
                    ((count++))
                fi
            fi
        fi
    done < "$output_file"
    
    log::debug "Created $count initialization embeddings"
    echo "$count"
}

#######################################
# Analyze initialization dependencies
# 
# Determines initialization order and dependencies
#
# Arguments:
#   $1 - Service.json path (optional)
# Returns: JSON with dependency analysis
#######################################
qdrant::init::analyze_dependencies() {
    local service_file="${1:-./.vrooli/service.json}"
    
    local init_items=$(qdrant::init::read_service_config "$service_file")
    
    if [[ -z "$init_items" ]] || [[ "$init_items" == "[]" ]]; then
        echo '{"stages": [], "dependencies": []}'
        return
    fi
    
    # Group by stage and order
    local stages=$(echo "$init_items" | jq -r '
        group_by(.stage // "default") |
        map({
            stage: .[0].stage // "default",
            items: map({
                path: .path,
                type: .type,
                purpose: .purpose,
                order: (.order // 999)
            }) | sort_by(.order)
        })
    ')
    
    # Detect potential dependencies
    local dependencies=()
    
    # Check for database migrations before seed data
    if echo "$init_items" | jq -e 'any(.type == "postgres" and .purpose == "schema_setup")' >/dev/null 2>&1 && \
       echo "$init_items" | jq -e 'any(.type == "postgres" and .purpose == "seed_data")' >/dev/null 2>&1; then
        dependencies+=('{"from": "schema_setup", "to": "seed_data", "type": "postgres"}')
    fi
    
    # Check for n8n workflows depending on APIs
    if echo "$init_items" | jq -e 'any(.type == "n8n")' >/dev/null 2>&1; then
        dependencies+=('{"from": "api_setup", "to": "n8n_workflows", "type": "integration"}')
    fi
    
    local deps_json="[]"
    if [[ ${#dependencies[@]} -gt 0 ]]; then
        deps_json=$(printf '%s\n' "${dependencies[@]}" | jq -s '.')
    fi
    
    jq -n \
        --argjson stages "$stages" \
        --argjson dependencies "$deps_json" \
        '{
            stages: $stages,
            dependencies: $dependencies
        }'
}

#######################################
# List all discovered initialization files
# 
# Useful for debugging and verification
#
# Arguments:
#   $1 - Service.json path (optional)
# Returns: List of files with their types
#######################################
qdrant::init::list_files() {
    local service_file="${1:-./.vrooli/service.json}"
    
    local init_items=$(qdrant::init::read_service_config "$service_file")
    
    if [[ -z "$init_items" ]] || [[ "$init_items" == "[]" ]]; then
        log::warn "No initialization files found"
        return
    fi
    
    echo "$init_items" | jq -r '.[] | "\(.type // "unknown") - \(.path) [\(.purpose // "unknown")]"'
}

# Export functions
export -f qdrant::init::read_service_config
export -f qdrant::init::dispatch_to_parser
export -f qdrant::init::process_all
export -f qdrant::embeddings::process_initialization
export -f qdrant::init::analyze_dependencies
export -f qdrant::init::list_files