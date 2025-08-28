#!/usr/bin/env bash
# Shell/Bash Code Extractor Library
# Extracts functions, descriptions, and metadata from shell scripts

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract functions from shell script
# 
# Finds function definitions, descriptions, and exported functions
#
# Arguments:
#   $1 - Path to shell script
#   $2 - Context (component type like 'api', 'cli', etc.)
#   $3 - Parent name (scenario/resource name)
# Returns: JSON lines with function information
#######################################
extractor::lib::shell::extract_functions() {
    local file="$1"
    local context="${2:-shell}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    log::debug "Extracting shell functions from: $file" >&2
    
    # Extract function definitions and their descriptions
    local functions_data=()
    
    # Find all function definitions
    local functions=$(grep -E "^[a-zA-Z_][a-zA-Z0-9_:]*\(\)" "$file" 2>/dev/null | sed 's/().*$//' || echo "")
    
    if [[ -n "$functions" ]]; then
        while IFS= read -r func_name; do
            [[ -z "$func_name" ]] && continue
            
            # Look for description comment above function
            local line_num=$(grep -n "^${func_name}()" "$file" | head -1 | cut -d: -f1)
            local description=""
            
            if [[ -n "$line_num" ]]; then
                # Look for comment block above function
                local desc_line=$((line_num - 1))
                while [[ $desc_line -gt 0 ]]; do
                    local line_content=$(sed -n "${desc_line}p" "$file")
                    if [[ "$line_content" =~ ^#[[:space:]](.+)$ ]]; then
                        local comment="${BASH_REMATCH[1]}"
                        if [[ -z "$description" ]]; then
                            description="$comment"
                        else
                            description="$comment $description"
                        fi
                        ((desc_line--))
                    else
                        break
                    fi
                done
            fi
            
            # Check if function is exported
            local is_exported="false"
            if grep -q "export -f $func_name" "$file" 2>/dev/null; then
                is_exported="true"
            fi
            
            # Build content
            local content="Function: $func_name | Context: $context | Parent: $parent_name"
            [[ -n "$description" ]] && content="$content | Description: $description"
            [[ "$is_exported" == "true" ]] && content="$content | Exported: true"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg function_name "$func_name" \
                --arg description "$description" \
                --arg is_exported "$is_exported" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "shell",
                        function_name: $function_name,
                        description: $description,
                        is_exported: ($is_exported == "true"),
                        content_type: "code_function",
                        extraction_method: "shell_parser"
                    }
                }' | jq -c
                
        done <<< "$functions"
    fi
}

#######################################
# Extract script metadata and overview
# 
# Gets script description, usage, and general information
#
# Arguments:
#   $1 - Path to shell script
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with script metadata
#######################################
extractor::lib::shell::extract_script_info() {
    local file="$1"
    local context="${2:-shell}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local filename=$(basename "$file")
    
    # Extract script description from header comments
    local description=""
    local usage=""
    local line_num=1
    
    while IFS= read -r line; do
        if [[ "$line" =~ ^#!.*$ ]]; then
            # Skip shebang
            ((line_num++))
            continue
        elif [[ "$line" =~ ^#[[:space:]]*(.*)$ ]]; then
            local comment="${BASH_REMATCH[1]}"
            
            # Look for description patterns
            if [[ "$comment" =~ ^Description:[[:space:]]*(.+)$ ]]; then
                description="${BASH_REMATCH[1]}"
            elif [[ "$comment" =~ ^Usage:[[:space:]]*(.+)$ ]]; then
                usage="${BASH_REMATCH[1]}"
            elif [[ -z "$description" && -n "$comment" && "$comment" != "" ]]; then
                # Use first substantial comment as description
                description="$comment"
            fi
            
            ((line_num++))
            
            # Stop after reasonable header scan
            [[ $line_num -gt 20 ]] && break
        elif [[ "$line" =~ ^[[:space:]]*$ ]]; then
            # Skip empty lines
            ((line_num++))
        else
            # Hit non-comment code, stop
            break
        fi
    done < "$file"
    
    # Count functions and exports
    local function_count=$(grep -c "^[a-zA-Z_][a-zA-Z0-9_:]*\(\)" "$file" 2>/dev/null || echo "0")
    local export_count=$(grep -c "^export -f " "$file" 2>/dev/null || echo "0")
    
    # Get file stats
    local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
    local line_count=$(wc -l < "$file" 2>/dev/null || echo "0")
    
    # Build content
    local content="Script: $filename | Context: $context | Parent: $parent_name"
    [[ -n "$description" ]] && content="$content | Description: $description"
    content="$content | Functions: $function_count | Lines: $line_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg filename "$filename" \
        --arg description "$description" \
        --arg usage "$usage" \
        --arg function_count "$function_count" \
        --arg export_count "$export_count" \
        --arg file_size "$file_size" \
        --arg line_count "$line_count" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "shell",
                filename: $filename,
                description: $description,
                usage: $usage,
                function_count: ($function_count | tonumber),
                export_count: ($export_count | tonumber),
                file_size: ($file_size | tonumber),
                line_count: ($line_count | tonumber),
                content_type: "code_script",
                extraction_method: "shell_parser"
            }
        }' | jq -c
}

#######################################
# Extract all information from shell script
# 
# Main entry point that extracts both functions and script info
#
# Arguments:
#   $1 - Path to shell script
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with all shell information
#######################################
extractor::lib::shell::extract_all() {
    local file="$1"
    local context="${2:-shell}"
    local parent_name="${3:-unknown}"
    
    # Extract script overview
    extractor::lib::shell::extract_script_info "$file" "$context" "$parent_name" 2>/dev/null || true
    
    # Extract individual functions
    extractor::lib::shell::extract_functions "$file" "$context" "$parent_name" 2>/dev/null || true
}

# Export functions
export -f extractor::lib::shell::extract_functions
export -f extractor::lib::shell::extract_script_info
export -f extractor::lib::shell::extract_all