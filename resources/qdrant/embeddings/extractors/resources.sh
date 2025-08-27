#!/usr/bin/env bash
# Modern Resource Content Extractor for Qdrant Embeddings
# Orchestrates modular extractors to output structured JSON lines
#
# This is the main entry point that sources specialized extractors:
# - cli.sh: Command-line interface and library functions
# - config.sh: Configuration files (capabilities.yaml, config.yaml, .env, Docker)
# - documentation.sh: Markdown documentation files
# - dependencies.sh: Package dependencies (npm, pip, go, etc.)
# - adapters.sh: Cross-resource integration adapters

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Define paths
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
EXTRACTOR_DIR="${EMBEDDINGS_DIR}/extractors"
RESOURCES_EXTRACTOR_DIR="${EXTRACTOR_DIR}/resources"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source modular extractors
source "${RESOURCES_EXTRACTOR_DIR}/cli.sh"
source "${RESOURCES_EXTRACTOR_DIR}/config.sh"
source "${RESOURCES_EXTRACTOR_DIR}/documentation.sh"
source "${RESOURCES_EXTRACTOR_DIR}/dependencies.sh"
source "${RESOURCES_EXTRACTOR_DIR}/adapters.sh"

#######################################
# Extract all resource information as JSON
# 
# Main orchestrator that calls all modular extractors
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON lines for all resource components
#######################################
qdrant::extract::resource_all_json() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        log::error "Resource directory not found: $dir" >&2
        return 1
    fi
    
    local resource_name=$(basename "$dir")
    log::debug "Extracting resource: $resource_name" >&2
    
    # Extract CLI components (cli.sh, lib functions, cross-references)
    qdrant::extract::resource_cli_all "$dir" 2>/dev/null || true
    
    # Extract configuration (capabilities.yaml, config.yaml, .env, Docker)
    qdrant::extract::resource_config_all "$dir" 2>/dev/null || true
    
    # Extract documentation (markdown files with smart section extraction)
    qdrant::extract::resource_documentation "$dir" 2>/dev/null || true
    
    # Extract dependencies (package.json, requirements.txt, go.mod, etc.)
    qdrant::extract::resource_dependencies_all "$dir" 2>/dev/null || true
    
    # Extract adapters (cross-resource integrations)
    qdrant::extract::resource_adapters_all "$dir" 2>/dev/null || true
}

#######################################
# Process batch of resources
# 
# Discovers and processes all resource directories
#
# Arguments:
#   $1 - Parent directory (optional, defaults to APP_ROOT)
#   $2 - Output file (optional, defaults to stdout)
# Returns: Number of resources processed
#######################################
qdrant::extract::resources_batch_json() {
    local parent_dir="${1:-$APP_ROOT}"
    local output_file="${2:-/dev/stdout}"
    
    if [[ ! -d "$parent_dir" ]]; then
        log::error "Parent directory not found: $parent_dir" >&2
        return 1
    fi
    
    log::info "Processing resources in: $parent_dir" >&2
    
    # Clear output file if specified
    [[ "$output_file" != "/dev/stdout" ]] && > "$output_file"
    
    local resource_dirs=()
    local resource_count=0
    local total_sections=0
    
    # Find all resource directories
    if [[ -d "$parent_dir/resources" ]]; then
        while IFS= read -r resource_dir; do
            # A valid resource has at least cli.sh or docs/
            if [[ -f "$resource_dir/cli.sh" ]] || [[ -d "$resource_dir/docs" ]] || [[ -d "$resource_dir/lib" ]]; then
                resource_dirs+=("$resource_dir")
            fi
        done < <(find "$parent_dir/resources" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort)
    fi
    
    if [[ ${#resource_dirs[@]} -eq 0 ]]; then
        log::warn "No resources found in $parent_dir/resources" >&2
        return 0
    fi
    
    log::info "Found ${#resource_dirs[@]} resources to process" >&2
    
    # Process each resource
    for resource_dir in "${resource_dirs[@]}"; do
        local resource_name=$(basename "$resource_dir")
        
        # Count lines before processing
        local before_count=0
        [[ "$output_file" != "/dev/stdout" ]] && before_count=$(wc -l < "$output_file" 2>/dev/null || echo 0)
        
        # Extract all components
        qdrant::extract::resource_all_json "$resource_dir" >> "$output_file"
        
        # Count lines after processing
        local after_count=0
        [[ "$output_file" != "/dev/stdout" ]] && after_count=$(wc -l < "$output_file" 2>/dev/null || echo 0)
        
        local sections_added=$((after_count - before_count))
        
        if [[ $sections_added -gt 0 ]]; then
            ((resource_count++))
            ((total_sections += sections_added))
            log::debug "  Extracted $sections_added sections from $resource_name" >&2
        fi
    done
    
    log::success "Processed $resource_count resources with $total_sections total sections" >&2
    echo "$resource_count"
}

#######################################
# Analyze resources and provide summary
# 
# Provides detailed analysis of available resources
#
# Arguments:
#   $1 - Parent directory (optional)
# Returns: Analysis report to stderr
#######################################
qdrant::extract::analyze_resources() {
    local parent_dir="${1:-$APP_ROOT}"
    
    echo "=== Resource Analysis ===" >&2
    echo >&2
    
    if [[ ! -d "$parent_dir/resources" ]]; then
        echo "❌ No resources directory found" >&2
        return 1
    fi
    
    # Count resources by component type
    local total=0
    local with_cli=0
    local with_docs=0
    local with_config=0
    local with_capabilities=0
    local with_docker=0
    local with_deps=0
    local with_adapters=0
    
    while IFS= read -r resource_dir; do
        ((total++))
        local name=$(basename "$resource_dir")
        
        # Check for various components
        [[ -f "$resource_dir/cli.sh" ]] && ((with_cli++))
        [[ -d "$resource_dir/docs" ]] || [[ -f "$resource_dir/README.md" ]] && ((with_docs++))
        [[ -f "$resource_dir/config.yaml" ]] || [[ -f "$resource_dir/config.yml" ]] && ((with_config++))
        [[ -f "$resource_dir/capabilities.yaml" ]] || [[ -f "$resource_dir/capabilities.yml" ]] && ((with_capabilities++))
        [[ -f "$resource_dir/Dockerfile" ]] || [[ -f "$resource_dir/docker-compose.yml" ]] && ((with_docker++))
        [[ -f "$resource_dir/package.json" ]] || [[ -f "$resource_dir/requirements.txt" ]] || \
            [[ -f "$resource_dir/go.mod" ]] || [[ -f "$resource_dir/Cargo.toml" ]] && ((with_deps++))
        [[ -d "$resource_dir/adapters" ]] && ((with_adapters++))
        
        echo "  📦 $name" >&2
    done < <(find "$parent_dir/resources" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort)
    
    echo >&2
    echo "Summary:" >&2
    echo "  Total resources: $total" >&2
    echo >&2
    echo "Component Coverage:" >&2
    echo "  CLI interface: $with_cli/$total" >&2
    echo "  Documentation: $with_docs/$total" >&2
    echo "  Configuration: $with_config/$total" >&2
    echo "  Capabilities: $with_capabilities/$total" >&2
    echo "  Docker support: $with_docker/$total" >&2
    echo "  Dependencies: $with_deps/$total" >&2
    echo "  Adapters: $with_adapters/$total" >&2
    echo >&2
    
    # Identify resources with adapters (integration providers)
    if [[ $with_adapters -gt 0 ]]; then
        echo "Integration Providers:" >&2
        while IFS= read -r resource_dir; do
            if [[ -d "$resource_dir/adapters" ]]; then
                local name=$(basename "$resource_dir")
                local adapter_count=$(find "$resource_dir/adapters" -type f -name "*.sh" 2>/dev/null | wc -l)
                echo "  🔗 $name ($adapter_count adapters)" >&2
            fi
        done < <(find "$parent_dir/resources" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort)
        echo >&2
    fi
    
    # Test JSON extraction on first resource
    local first_resource=$(find "$parent_dir/resources" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | head -1)
    if [[ -n "$first_resource" ]]; then
        echo "Extraction Test on $(basename "$first_resource"):" >&2
        
        local sample_output=$(qdrant::extract::resource_all_json "$first_resource" 2>/dev/null | head -5)
        local valid_count=0
        local line_count=0
        
        while IFS= read -r line; do
            ((line_count++))
            if echo "$line" | jq -e '.metadata' >/dev/null 2>&1; then
                ((valid_count++))
            fi
        done <<< "$sample_output"
        
        if [[ $valid_count -eq $line_count && $line_count -gt 0 ]]; then
            echo "  ✅ All $line_count lines are valid JSON" >&2
            
            # Show component types found
            echo "  Component types extracted:" >&2
            echo "$sample_output" | jq -r '.metadata.component_type' | sort -u | while read -r type; do
                echo "    • $type" >&2
            done
        else
            echo "  ⚠️  JSON validation: $valid_count/$line_count valid" >&2
        fi
    fi
}

#######################################
# Detect resource category from name
# 
# Categorizes resources based on naming patterns
#
# Arguments:
#   $1 - Resource name
# Returns: Category string
#######################################
qdrant::extract::detect_resource_category() {
    local resource_name="$1"
    
    case "$resource_name" in
        ollama|litellm|gemini|openai|claude-code|openrouter|llamaindex|langchain|huggingface)
            echo "ai"
            ;;
        postgres|redis|qdrant|minio|neo4j|questdb|vault|postgis|mysql|mongodb)
            echo "storage"
            ;;
        n8n|windmill|node-red|huginn|airflow|prefect)
            echo "automation"
            ;;
        browserless|agent-s2|autogpt|crewai|autogen-studio)
            echo "agents"
            ;;
        judge0|ffmpeg|blender|obs-studio|kicad|openscad|pandoc)
            echo "execution"
            ;;
        btcpay|erpnext|odoo|keycloak|mail-in-a-box|home-assistant)
            echo "services"
            ;;
        *)
            echo "utility"
            ;;
    esac
}

# Export main functions
export -f qdrant::extract::resource_all_json
export -f qdrant::extract::resources_batch_json
export -f qdrant::extract::analyze_resources
export -f qdrant::extract::detect_resource_category

# Legacy function names for backward compatibility
qdrant::extract::resource_cli_json() {
    qdrant::extract::resource_cli "$@"
}
qdrant::extract::resource_config_json() {
    qdrant::extract::resource_config_all "$@"
}
qdrant::extract::resource_dependencies_json() {
    qdrant::extract::resource_dependencies_all "$@"
}
qdrant::extract::resource_integrations_json() {
    qdrant::extract::resource_adapters_all "$@"
}

export -f qdrant::extract::resource_cli_json
export -f qdrant::extract::resource_config_json
export -f qdrant::extract::resource_dependencies_json
export -f qdrant::extract::resource_integrations_json