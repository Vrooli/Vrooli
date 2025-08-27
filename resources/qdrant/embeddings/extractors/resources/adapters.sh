#!/usr/bin/env bash
# Resource Adapters Extractor
# Extracts information about cross-resource adapters
#
# Adapters enable resources to integrate with each other,
# allowing one resource to provide functionality for another.
# Example: browserless can provide UI automation for other resources.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract adapter information
# 
# Discovers and analyzes adapters in the adapters/ directory
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON lines with adapter information
#######################################
qdrant::extract::resource_adapters() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    if [[ ! -d "$dir/adapters" ]]; then
        return 1
    fi
    
    log::debug "Extracting adapters for $resource_name" >&2
    
    # Find all adapter files
    local adapter_files=()
    while IFS= read -r file; do
        adapter_files+=("$file")
    done < <(find "$dir/adapters" -type f -name "*.sh" 2>/dev/null | sort)
    
    if [[ ${#adapter_files[@]} -eq 0 ]]; then
        return 1
    fi
    
    # Process each adapter file
    for adapter_file in "${adapter_files[@]}"; do
        local adapter_name=$(basename "$adapter_file" .sh)
        local relative_path="${adapter_file#$dir/}"
        
        # Extract adapter metadata from comments
        local description=""
        local target_resources=()
        local provided_capabilities=()
        
        # Look for description in header comments
        description=$(grep -E "^#.*[Aa]dapter" "$adapter_file" 2>/dev/null | \
            head -1 | sed 's/^#[[:space:]]*//' || echo "")
        
        # Find target resources mentioned in the adapter
        local targets=$(grep -oE "resource-[a-z][a-z0-9-]*" "$adapter_file" 2>/dev/null | \
            sed 's/resource-//' | sort -u)
        
        for target in $targets; do
            if [[ "$target" != "$resource_name" ]]; then
                target_resources+=("$target")
            fi
        done
        
        # Extract functions that suggest capabilities
        local functions=$(grep -E "^[a-z_]+::[a-z_]+\(\)" "$adapter_file" 2>/dev/null | \
            sed 's/().*//' || echo "")
        
        # Detect common adapter patterns
        if grep -q "adapter::init" "$adapter_file" 2>/dev/null; then
            provided_capabilities+=("initialization")
        fi
        if grep -q "adapter::connect" "$adapter_file" 2>/dev/null; then
            provided_capabilities+=("connection")
        fi
        if grep -q "adapter::execute" "$adapter_file" 2>/dev/null; then
            provided_capabilities+=("execution")
        fi
        if grep -q "adapter::transform" "$adapter_file" 2>/dev/null; then
            provided_capabilities+=("transformation")
        fi
        
        # Convert to JSON arrays
        local targets_json="[]"
        if [[ ${#target_resources[@]} -gt 0 ]]; then
            targets_json=$(printf '%s\n' "${target_resources[@]}" | jq -R . | jq -s .)
        fi
        
        local capabilities_json="[]"
        if [[ ${#provided_capabilities[@]} -gt 0 ]]; then
            capabilities_json=$(printf '%s\n' "${provided_capabilities[@]}" | jq -R . | jq -s .)
        fi
        
        local functions_json="[]"
        if [[ -n "$functions" ]]; then
            functions_json=$(echo "$functions" | jq -R . | jq -s .)
        fi
        
        # Build content
        local content="Resource: $resource_name | Adapter: $adapter_name"
        [[ -n "$description" ]] && content="$content | $description"
        [[ ${#target_resources[@]} -gt 0 ]] && \
            content="$content | Targets: $(echo "$targets_json" | jq -r 'join(", ")')"
        
        # Output as JSON line
        jq -n \
            --arg content "$content" \
            --arg resource "$resource_name" \
            --arg source_file "$adapter_file" \
            --arg adapter_name "$adapter_name" \
            --arg relative_path "$relative_path" \
            --arg description "$description" \
            --argjson target_resources "$targets_json" \
            --argjson capabilities "$capabilities_json" \
            --argjson functions "$functions_json" \
            '{
                content: $content,
                metadata: {
                    resource: $resource,
                    source_file: $source_file,
                    component_type: "adapter",
                    adapter_name: $adapter_name,
                    relative_path: $relative_path,
                    description: $description,
                    target_resources: $target_resources,
                    provided_capabilities: $capabilities,
                    functions: $functions,
                    content_type: "resource_adapter",
                    extraction_method: "adapter_parser"
                }
            }' | jq -c
    done
}

#######################################
# Extract adapter registry information
# 
# Processes registry.sh files that manage adapter registration
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON line with registry information
#######################################
qdrant::extract::adapter_registry() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    local registry_file="$dir/adapters/registry.sh"
    if [[ ! -f "$registry_file" ]]; then
        return 1
    fi
    
    log::debug "Extracting adapter registry for $resource_name" >&2
    
    # Extract registered adapters
    local registered_adapters=$(grep -E "register::|ADAPTERS\[" "$registry_file" 2>/dev/null | \
        grep -oE '"[^"]+"' | tr -d '"' | sort -u)
    
    local adapters_json="[]"
    if [[ -n "$registered_adapters" ]]; then
        adapters_json=$(echo "$registered_adapters" | jq -R . | jq -s .)
    fi
    
    local adapter_count=$(echo "$adapters_json" | jq 'length')
    
    # Build content
    local content="Resource: $resource_name | Type: Adapter Registry"
    content="$content | Registered Adapters: $adapter_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg resource "$resource_name" \
        --arg source_file "$registry_file" \
        --argjson registered_adapters "$adapters_json" \
        --arg adapter_count "$adapter_count" \
        '{
            content: $content,
            metadata: {
                resource: $resource,
                source_file: $source_file,
                component_type: "adapter_registry",
                registered_adapters: $registered_adapters,
                adapter_count: ($adapter_count | tonumber),
                content_type: "resource_adapter",
                extraction_method: "registry_parser"
            }
        }' | jq -c
}

#######################################
# Extract integration patterns
# 
# Analyzes how adapters enable cross-resource integration
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON line with integration patterns
#######################################
qdrant::extract::integration_patterns() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    if [[ ! -d "$dir/adapters" ]]; then
        return 1
    fi
    
    log::debug "Extracting integration patterns for $resource_name" >&2
    
    # Detect common integration patterns
    local patterns=()
    
    # Check for UI automation pattern (like browserless)
    if grep -r "puppeteer\|playwright\|selenium" "$dir/adapters" &>/dev/null; then
        patterns+=("ui-automation")
    fi
    
    # Check for API bridge pattern
    if grep -r "api::\|REST\|GraphQL" "$dir/adapters" &>/dev/null; then
        patterns+=("api-bridge")
    fi
    
    # Check for data transformation pattern
    if grep -r "transform::\|convert::\|parse::" "$dir/adapters" &>/dev/null; then
        patterns+=("data-transformation")
    fi
    
    # Check for event forwarding pattern
    if grep -r "webhook\|event::\|subscribe::" "$dir/adapters" &>/dev/null; then
        patterns+=("event-forwarding")
    fi
    
    # Check for authentication proxy pattern
    if grep -r "auth::\|token::\|oauth" "$dir/adapters" &>/dev/null; then
        patterns+=("auth-proxy")
    fi
    
    if [[ ${#patterns[@]} -gt 0 ]]; then
        local patterns_json=$(printf '%s\n' "${patterns[@]}" | jq -R . | jq -s .)
        
        # Build content
        local content="Resource: $resource_name | Type: Integration Patterns"
        content="$content | Patterns: $(echo "$patterns_json" | jq -r 'join(", ")')"
        
        # Output as JSON line
        jq -n \
            --arg content "$content" \
            --arg resource "$resource_name" \
            --arg source_dir "$dir/adapters" \
            --argjson patterns "$patterns_json" \
            --arg pattern_count "${#patterns[@]}" \
            '{
                content: $content,
                metadata: {
                    resource: $resource,
                    source_directory: $source_dir,
                    component_type: "integration_patterns",
                    patterns: $patterns,
                    pattern_count: ($pattern_count | tonumber),
                    content_type: "resource_adapter",
                    extraction_method: "pattern_detector"
                }
            }' | jq -c
    fi
}

#######################################
# Extract all adapter information
# 
# Main function that calls all adapter extractors
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON lines for all adapter components
#######################################
qdrant::extract::resource_adapters_all() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        return 1
    fi
    
    # Extract individual adapters
    qdrant::extract::resource_adapters "$dir" 2>/dev/null || true
    
    # Extract adapter registry
    qdrant::extract::adapter_registry "$dir" 2>/dev/null || true
    
    # Extract integration patterns
    qdrant::extract::integration_patterns "$dir" 2>/dev/null || true
}

# Export functions for use by resources.sh
export -f qdrant::extract::resource_adapters
export -f qdrant::extract::adapter_registry
export -f qdrant::extract::integration_patterns
export -f qdrant::extract::resource_adapters_all