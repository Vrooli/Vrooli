#!/usr/bin/env bash
# JavaScript/TypeScript Code Extractor Library
# Extracts functions, classes, exports, and metadata from JS/TS files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract functions from JavaScript/TypeScript file
# 
# Finds function definitions, arrow functions, and methods
#
# Arguments:
#   $1 - Path to JS/TS file
#   $2 - Context (component type like 'api', 'ui', etc.)
#   $3 - Parent name (scenario name)
# Returns: JSON lines with function information
#######################################
extractor::lib::javascript::extract_functions() {
    local file="$1"
    local context="${2:-javascript}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    log::debug "Extracting JS/TS functions from: $file" >&2
    
    # Extract different function patterns
    local functions_found=()
    
    # Function declarations: function name() { }
    while IFS= read -r line; do
        if [[ "$line" =~ ^[[:space:]]*function[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]*\( ]]; then
            functions_found+=("${BASH_REMATCH[1]}:function")
        fi
    done < "$file"
    
    # Arrow functions: const name = () => { }
    while IFS= read -r line; do
        if [[ "$line" =~ ^[[:space:]]*const[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]*=[[:space:]]*.*=\> ]]; then
            functions_found+=("${BASH_REMATCH[1]}:arrow")
        fi
    done < "$file"
    
    # Method definitions: methodName() { } or async methodName() { }
    while IFS= read -r line; do
        if [[ "$line" =~ ^[[:space:]]*(async[[:space:]]+)?([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]*\(.*\)[[:space:]]*\{ ]]; then
            functions_found+=("${BASH_REMATCH[2]}:method")
        fi
    done < "$file"
    
    # Export functions
    for func_info in "${functions_found[@]}"; do
        local func_name="${func_info%:*}"
        local func_type="${func_info#*:}"
        
        # Check if function is exported
        local is_exported="false"
        if grep -q "export.*$func_name" "$file" 2>/dev/null; then
            is_exported="true"
        fi
        
        # Look for JSDoc or comment above function
        local description=""
        local line_num=$(grep -n "$func_name" "$file" | head -1 | cut -d: -f1)
        
        if [[ -n "$line_num" ]]; then
            local desc_line=$((line_num - 1))
            while [[ $desc_line -gt 0 ]]; do
                local line_content=$(sed -n "${desc_line}p" "$file")
                if [[ "$line_content" =~ ^[[:space:]]*//[[:space:]]*(.+)$ ]] || [[ "$line_content" =~ ^[[:space:]]*\*[[:space:]]*(.+)$ ]]; then
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
        
        # Build content
        local content="Function: $func_name | Type: $func_type | Context: $context | Parent: $parent_name"
        [[ -n "$description" ]] && content="$content | Description: $description"
        [[ "$is_exported" == "true" ]] && content="$content | Exported: true"
        
        # Output as JSON line
        jq -n \
            --arg content "$content" \
            --arg parent "$parent_name" \
            --arg source_file "$file" \
            --arg context "$context" \
            --arg function_name "$func_name" \
            --arg function_type "$func_type" \
            --arg description "$description" \
            --arg is_exported "$is_exported" \
            '{
                content: $content,
                metadata: {
                    parent: $parent,
                    source_file: $source_file,
                    component_type: $context,
                    language: "javascript",
                    function_name: $function_name,
                    function_type: $function_type,
                    description: $description,
                    is_exported: ($is_exported == "true"),
                    content_type: "code_function",
                    extraction_method: "javascript_parser"
                }
            }' | jq -c
    done
}

#######################################
# Extract classes from JavaScript/TypeScript file
# 
# Finds class definitions and their methods
#
# Arguments:
#   $1 - Path to JS/TS file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with class information
#######################################
extractor::lib::javascript::extract_classes() {
    local file="$1"
    local context="${2:-javascript}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find class definitions
    local classes=$(grep -E "^[[:space:]]*class[[:space:]]+[a-zA-Z_][a-zA-Z0-9_]*" "$file" 2>/dev/null | \
        sed -E 's/^[[:space:]]*class[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' || echo "")
    
    if [[ -n "$classes" ]]; then
        while IFS= read -r class_name; do
            [[ -z "$class_name" ]] && continue
            
            # Check if class is exported
            local is_exported="false"
            if grep -q "export.*class.*$class_name" "$file" 2>/dev/null; then
                is_exported="true"
            fi
            
            # Count methods in class (rough approximation)
            local method_count=0
            local in_class=false
            local brace_count=0
            
            while IFS= read -r line; do
                if [[ "$line" =~ class[[:space:]]+$class_name ]]; then
                    in_class=true
                fi
                
                if [[ "$in_class" == "true" ]]; then
                    # Count braces to know when we're out of the class
                    local open_braces="${line//[^\{]/}"
                    local close_braces="${line//[^\}]/}"
                    brace_count=$((brace_count + ${#open_braces} - ${#close_braces}))
                    
                    # Look for method patterns
                    if [[ "$line" =~ ^[[:space:]]*([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]*\( ]]; then
                        ((method_count++))
                    fi
                    
                    if [[ $brace_count -eq 0 && "$line" =~ \} ]]; then
                        break
                    fi
                fi
            done < "$file"
            
            # Build content
            local content="Class: $class_name | Context: $context | Parent: $parent_name"
            content="$content | Methods: $method_count"
            [[ "$is_exported" == "true" ]] && content="$content | Exported: true"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg class_name "$class_name" \
                --arg method_count "$method_count" \
                --arg is_exported "$is_exported" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "javascript",
                        class_name: $class_name,
                        method_count: ($method_count | tonumber),
                        is_exported: ($is_exported == "true"),
                        content_type: "code_class",
                        extraction_method: "javascript_parser"
                    }
                }' | jq -c
        done <<< "$classes"
    fi
}

#######################################
# Extract package.json information
# 
# Gets package metadata, dependencies, scripts
#
# Arguments:
#   $1 - Path to package.json file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with package information
#######################################
extractor::lib::javascript::extract_package_json() {
    local file="$1"
    local context="${2:-javascript}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract package metadata
    local name=$(jq -r '.name // empty' "$file" 2>/dev/null)
    local version=$(jq -r '.version // empty' "$file" 2>/dev/null)
    local description=$(jq -r '.description // empty' "$file" 2>/dev/null)
    
    # Count dependencies
    local dep_count=$(jq '.dependencies | length' "$file" 2>/dev/null || echo "0")
    local dev_dep_count=$(jq '.devDependencies | length' "$file" 2>/dev/null || echo "0")
    
    # Get script names
    local scripts_json=$(jq -c '.scripts // {}' "$file" 2>/dev/null || echo "{}")
    local script_count=$(echo "$scripts_json" | jq 'length')
    
    # Build content
    local content="Package: $name | Version: $version | Context: $context | Parent: $parent_name"
    [[ -n "$description" ]] && content="$content | Description: $description"
    content="$content | Dependencies: $dep_count | DevDeps: $dev_dep_count | Scripts: $script_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg name "$name" \
        --arg version "$version" \
        --arg description "$description" \
        --arg dep_count "$dep_count" \
        --arg dev_dep_count "$dev_dep_count" \
        --argjson scripts "$scripts_json" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "javascript",
                package_name: $name,
                package_version: $version,
                package_description: $description,
                dependency_count: ($dep_count | tonumber),
                dev_dependency_count: ($dev_dep_count | tonumber),
                scripts: $scripts,
                content_type: "package_config",
                extraction_method: "package_json_parser"
            }
        }' | jq -c
}

#######################################
# Extract all information from JavaScript/TypeScript files
# 
# Main entry point that extracts functions, classes, and package info
#
# Arguments:
#   $1 - Path to file or directory
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with all JS/TS information
#######################################
extractor::lib::javascript::extract_all() {
    local path="$1"
    local context="${2:-javascript}"
    local parent_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        case "$path" in
            *.json)
                if [[ "$(basename "$path")" == "package.json" ]]; then
                    extractor::lib::javascript::extract_package_json "$path" "$context" "$parent_name" 2>/dev/null || true
                fi
                ;;
            *.js|*.ts|*.jsx|*.tsx)
                extractor::lib::javascript::extract_functions "$path" "$context" "$parent_name" 2>/dev/null || true
                extractor::lib::javascript::extract_classes "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
        esac
    elif [[ -d "$path" ]]; then
        # Directory - find relevant files
        while IFS= read -r file; do
            extractor::lib::javascript::extract_all "$file" "$context" "$parent_name"
        done < <(find "$path" -type f \( -name "*.js" -o -name "*.ts" -o -name "*.jsx" -o -name "*.tsx" -o -name "package.json" \) 2>/dev/null)
    fi
}

# Export functions
export -f extractor::lib::javascript::extract_functions
export -f extractor::lib::javascript::extract_classes
export -f extractor::lib::javascript::extract_package_json
export -f extractor::lib::javascript::extract_all