#!/usr/bin/env bash
# Resource CLI Extractor
# Extracts command-line interface information from resource CLI files
#
# Parses cli.sh files to extract:
# - Available commands and subcommands
# - Command descriptions and help text
# - Library capabilities (based on lib/*.sh files)
# - Resource dependencies and integrations

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract CLI commands and descriptions
# 
# Parses cli.sh to find available commands and their descriptions
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON line with CLI information
#######################################
qdrant::extract::resource_cli() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]] || [[ ! -f "$dir/cli.sh" ]]; then
        return 1
    fi
    
    local resource_name=$(basename "$dir")
    local cli_file="$dir/cli.sh"
    
    log::debug "Extracting CLI information for $resource_name" >&2
    
    # Extract description from CLI header
    local description=""
    if grep -q "^# Description:" "$cli_file"; then
        description=$(grep "^# Description:" "$cli_file" | cut -d: -f2- | sed 's/^ *//')
    elif grep -q "^# $resource_name" "$cli_file"; then
        # Try to get description from header comment
        description=$(grep -A 1 "^# $resource_name" "$cli_file" | tail -1 | sed 's/^# *//')
    fi
    
    # Extract available commands from case statements
    local commands_raw=$(grep -E "^[[:space:]]*(\"[^\"]+\"|'[^']+'|[a-z_-]+)\)" "$cli_file" 2>/dev/null | \
        sed -E "s/^[[:space:]]*(\"([^\"]+)\"|'([^']+)'|([a-z_-]+))\).*/\2\3\4/" | \
        grep -v -E "^(true|false|yes|no|[0-9]+|[\$*])$" || echo "")
    
    # Convert commands to JSON array
    local commands_json="[]"
    if [[ -n "$commands_raw" ]]; then
        commands_json=$(echo "$commands_raw" | jq -R . | jq -s .)
    fi
    
    # Extract command descriptions (look for echo statements after case patterns)
    local command_descriptions="{}"
    if [[ -n "$commands_raw" ]]; then
        local desc_json="{}"
        while IFS= read -r cmd; do
            # Look for description/usage after the command pattern
            local desc=$(grep -A 5 "\"$cmd\")" "$cli_file" 2>/dev/null | \
                grep -E "(echo|log::|printf)" | head -1 | \
                sed -E 's/.*echo[[:space:]]+"([^"]+)".*/\1/' | \
                sed -E 's/.*log::[^[:space:]]+[[:space:]]+"([^"]+)".*/\1/')
            
            if [[ -n "$desc" && "$desc" != *"$cmd"* ]]; then
                desc_json=$(echo "$desc_json" | jq --arg cmd "$cmd" --arg desc "$desc" '.[$cmd] = $desc')
            fi
        done <<< "$commands_raw"
        command_descriptions="$desc_json"
    fi
    
    # Check for library capabilities
    local capabilities_json="[]"
    if [[ -d "$dir/lib" ]]; then
        local caps=()
        
        # Standard capability detection based on lib files
        [[ -f "$dir/lib/api.sh" ]] && caps+=("API")
        [[ -f "$dir/lib/auth.sh" ]] && caps+=("Authentication")
        [[ -f "$dir/lib/collections.sh" ]] && caps+=("Collections")
        [[ -f "$dir/lib/models.sh" ]] && caps+=("Models")
        [[ -f "$dir/lib/search.sh" ]] && caps+=("Search")
        [[ -f "$dir/lib/backup.sh" ]] && caps+=("Backup")
        [[ -f "$dir/lib/workflows.sh" ]] && caps+=("Workflows")
        [[ -f "$dir/lib/database.sh" ]] && caps+=("Database")
        [[ -f "$dir/lib/storage.sh" ]] && caps+=("Storage")
        [[ -f "$dir/lib/queue.sh" ]] && caps+=("Queue")
        [[ -f "$dir/lib/monitoring.sh" ]] && caps+=("Monitoring")
        [[ -f "$dir/lib/security.sh" ]] && caps+=("Security")
        
        if [[ ${#caps[@]} -gt 0 ]]; then
            capabilities_json=$(printf '%s\n' "${caps[@]}" | jq -R . | jq -s .)
        fi
    fi
    
    # Count library files
    local lib_count=0
    if [[ -d "$dir/lib" ]]; then
        lib_count=$(find "$dir/lib" -type f -name "*.sh" 2>/dev/null | wc -l)
    fi
    
    # Build enriched content
    local content="Resource: $resource_name | Type: CLI"
    [[ -n "$description" ]] && content="$content | Description: $description"
    content="$content | Commands: $(echo "$commands_json" | jq -r 'join(", ")')"
    [[ $(echo "$capabilities_json" | jq 'length') -gt 0 ]] && \
        content="$content | Capabilities: $(echo "$capabilities_json" | jq -r 'join(", ")')"
    content="$content | Library files: $lib_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg resource "$resource_name" \
        --arg source_file "$cli_file" \
        --arg description "$description" \
        --argjson commands "$commands_json" \
        --argjson command_descriptions "$command_descriptions" \
        --argjson capabilities "$capabilities_json" \
        --arg lib_count "$lib_count" \
        '{
            content: $content,
            metadata: {
                resource: $resource,
                source_file: $source_file,
                component_type: "cli",
                description: $description,
                commands: $commands,
                command_descriptions: $command_descriptions,
                capabilities: $capabilities,
                library_files: ($lib_count | tonumber),
                content_type: "resource_cli",
                extraction_method: "cli_parser"
            }
        }' | jq -c
}

#######################################
# Extract library function signatures
# 
# Analyzes lib/*.sh files to extract function names and purposes
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON lines with library function information
#######################################
qdrant::extract::resource_lib_functions() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    if [[ ! -d "$dir/lib" ]]; then
        return 1
    fi
    
    log::debug "Extracting library functions for $resource_name" >&2
    
    # Process each library file
    for lib_file in "$dir/lib"/*.sh; do
        [[ ! -f "$lib_file" ]] && continue
        
        local lib_name=$(basename "$lib_file" .sh)
        
        # Extract function names and their descriptions
        local functions=$(grep -E "^[a-z_]+::[a-z_]+\(\)" "$lib_file" 2>/dev/null | \
            sed 's/().*//' || echo "")
        
        local functions_json="[]"
        if [[ -n "$functions" ]]; then
            functions_json=$(echo "$functions" | jq -R . | jq -s .)
        fi
        
        local func_count=$(echo "$functions_json" | jq 'length')
        
        if [[ $func_count -gt 0 ]]; then
            # Build content
            local content="Resource: $resource_name | Library: $lib_name"
            content="$content | Functions: $func_count"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg resource "$resource_name" \
                --arg source_file "$lib_file" \
                --arg library "$lib_name" \
                --argjson functions "$functions_json" \
                --arg func_count "$func_count" \
                '{
                    content: $content,
                    metadata: {
                        resource: $resource,
                        source_file: $source_file,
                        component_type: "library",
                        library_name: $library,
                        functions: $functions,
                        function_count: ($func_count | tonumber),
                        content_type: "resource_library",
                        extraction_method: "function_parser"
                    }
                }' | jq -c
        fi
    done
}

#######################################
# Extract cross-resource references
# 
# Finds references to other resources in the CLI code
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON line with cross-references
#######################################
qdrant::extract::resource_cross_refs() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    if [[ ! -f "$dir/cli.sh" ]]; then
        return 1
    fi
    
    log::debug "Extracting cross-references for $resource_name" >&2
    
    # Find references to other resources
    local refs=()
    local refs_raw=$(grep -h "resource-" "$dir/cli.sh" 2>/dev/null | \
        grep -oE "resource-[a-z][a-z0-9-]*" | sort -u)
    
    for ref in $refs_raw; do
        local ref_name=${ref#resource-}
        # Skip self-references
        if [[ "$ref_name" != "$resource_name" ]]; then
            refs+=("$ref_name")
        fi
    done
    
    if [[ ${#refs[@]} -gt 0 ]]; then
        local refs_json=$(printf '%s\n' "${refs[@]}" | jq -R . | jq -s .)
        
        # Build content
        local content="Resource: $resource_name | Type: Cross-References"
        content="$content | References: $(echo "$refs_json" | jq -r 'join(", ")')"
        
        # Output as JSON line
        jq -n \
            --arg content "$content" \
            --arg resource "$resource_name" \
            --arg source_file "$dir/cli.sh" \
            --argjson references "$refs_json" \
            --arg ref_count "${#refs[@]}" \
            '{
                content: $content,
                metadata: {
                    resource: $resource,
                    source_file: $source_file,
                    component_type: "cross_references",
                    references: $references,
                    reference_count: ($ref_count | tonumber),
                    content_type: "resource_integration",
                    extraction_method: "reference_scanner"
                }
            }' | jq -c
    fi
}

#######################################
# Extract all CLI-related information
# 
# Main function that calls all CLI extractors
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON lines for all CLI components
#######################################
qdrant::extract::resource_cli_all() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        return 1
    fi
    
    # Extract main CLI information
    qdrant::extract::resource_cli "$dir" 2>/dev/null || true
    
    # Extract library functions
    qdrant::extract::resource_lib_functions "$dir" 2>/dev/null || true
    
    # Extract cross-references
    qdrant::extract::resource_cross_refs "$dir" 2>/dev/null || true
}

# Export functions for use by resources.sh
export -f qdrant::extract::resource_cli
export -f qdrant::extract::resource_lib_functions
export -f qdrant::extract::resource_cross_refs
export -f qdrant::extract::resource_cli_all