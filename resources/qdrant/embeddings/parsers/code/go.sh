#!/usr/bin/env bash
# Go Code Extractor Library
# Extracts functions, types, packages, and metadata from Go files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract functions from Go file
# 
# Finds function definitions and methods
#
# Arguments:
#   $1 - Path to Go file
#   $2 - Context (component type like 'api', 'cli', etc.)
#   $3 - Parent name (scenario name)
# Returns: JSON lines with function information
#######################################
extractor::lib::go::extract_functions() {
    local file="$1"
    local context="${2:-go}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    log::debug "Extracting Go functions from: $file" >&2
    
    # Find function definitions
    local functions=$(grep -E "^func[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/^func[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' || echo "")
    
    # Find method definitions (func (receiver) MethodName)
    local methods=$(grep -E "^func[[:space:]]+\([^)]+\)[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/^func[[:space:]]+\([^)]+\)[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' || echo "")
    
    # Process functions
    if [[ -n "$functions" ]]; then
        while IFS= read -r func_name; do
            [[ -z "$func_name" ]] && continue
            
            # Check if function is exported (starts with capital letter)
            local is_exported="false"
            if [[ "$func_name" =~ ^[A-Z] ]]; then
                is_exported="true"
            fi
            
            # Get function signature
            local signature=$(grep "^func $func_name" "$file" 2>/dev/null | head -1 || echo "")
            
            # Look for comment above function
            local description=""
            local line_num=$(grep -n "^func $func_name" "$file" | head -1 | cut -d: -f1)
            
            if [[ -n "$line_num" ]]; then
                local desc_line=$((line_num - 1))
                while [[ $desc_line -gt 0 ]]; do
                    local line_content=$(sed -n "${desc_line}p" "$file")
                    if [[ "$line_content" =~ ^//[[:space:]]*(.+)$ ]]; then
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
                --arg signature "$signature" \
                --arg description "$description" \
                --arg is_exported "$is_exported" \
                --arg function_type "function" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "go",
                        function_name: $function_name,
                        function_type: $function_type,
                        signature: $signature,
                        description: $description,
                        is_exported: ($is_exported == "true"),
                        content_type: "code_function",
                        extraction_method: "go_parser"
                    }
                }' | jq -c
        done <<< "$functions"
    fi
    
    # Process methods
    if [[ -n "$methods" ]]; then
        while IFS= read -r method_name; do
            [[ -z "$method_name" ]] && continue
            
            # Check if method is exported
            local is_exported="false"
            if [[ "$method_name" =~ ^[A-Z] ]]; then
                is_exported="true"
            fi
            
            # Get method signature and receiver
            local signature=$(grep -E "^func[[:space:]]+\([^)]+\)[[:space:]]+$method_name" "$file" 2>/dev/null | head -1 || echo "")
            local receiver=""
            if [[ "$signature" =~ func[[:space:]]+\(.*\) ]]; then
                receiver="receiver"
            fi
            
            # Build content
            local content="Method: $method_name | Context: $context | Parent: $parent_name"
            [[ -n "$receiver" ]] && content="$content | Receiver: $receiver"
            [[ "$is_exported" == "true" ]] && content="$content | Exported: true"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg function_name "$method_name" \
                --arg signature "$signature" \
                --arg receiver "$receiver" \
                --arg is_exported "$is_exported" \
                --arg function_type "method" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "go",
                        function_name: $function_name,
                        function_type: $function_type,
                        signature: $signature,
                        receiver: $receiver,
                        is_exported: ($is_exported == "true"),
                        content_type: "code_function",
                        extraction_method: "go_parser"
                    }
                }' | jq -c
        done <<< "$methods"
    fi
}

#######################################
# Extract types from Go file
# 
# Finds struct and interface definitions
#
# Arguments:
#   $1 - Path to Go file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with type information
#######################################
extractor::lib::go::extract_types() {
    local file="$1"
    local context="${2:-go}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find struct definitions
    local structs=$(grep -E "^type[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]+struct" "$file" 2>/dev/null | \
        sed -E 's/^type[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]+struct.*/\1/' || echo "")
    
    # Find interface definitions
    local interfaces=$(grep -E "^type[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]+interface" "$file" 2>/dev/null | \
        sed -E 's/^type[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]+interface.*/\1/' || echo "")
    
    # Process structs
    if [[ -n "$structs" ]]; then
        while IFS= read -r struct_name; do
            [[ -z "$struct_name" ]] && continue
            
            # Check if exported
            local is_exported="false"
            if [[ "$struct_name" =~ ^[A-Z] ]]; then
                is_exported="true"
            fi
            
            # Count fields (rough approximation)
            local field_count=0
            local in_struct=false
            local brace_count=0
            
            while IFS= read -r line; do
                if [[ "$line" =~ type[[:space:]]+$struct_name[[:space:]]+struct ]]; then
                    in_struct=true
                fi
                
                if [[ "$in_struct" == "true" ]]; then
                    if [[ "$line" == *"{"* ]]; then
                        brace_count=$((brace_count + 1))
                    fi
                    
                    # Count field lines (lines with field pattern)
                    if [[ "$line" =~ ^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*[[:space:]]+ ]]; then
                        ((field_count++))
                    fi
                    
                    if [[ "$line" == *"}"* ]]; then
                        brace_count=$((brace_count - 1))
                        if [[ $brace_count -eq 0 ]]; then
                            break
                        fi
                    fi
                fi
            done < "$file"
            
            # Build content
            local content="Struct: $struct_name | Context: $context | Parent: $parent_name"
            content="$content | Fields: $field_count"
            [[ "$is_exported" == "true" ]] && content="$content | Exported: true"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg type_name "$struct_name" \
                --arg type_kind "struct" \
                --arg field_count "$field_count" \
                --arg is_exported "$is_exported" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "go",
                        type_name: $type_name,
                        type_kind: $type_kind,
                        field_count: ($field_count | tonumber),
                        is_exported: ($is_exported == "true"),
                        content_type: "code_type",
                        extraction_method: "go_parser"
                    }
                }' | jq -c
        done <<< "$structs"
    fi
    
    # Process interfaces (similar logic)
    if [[ -n "$interfaces" ]]; then
        while IFS= read -r interface_name; do
            [[ -z "$interface_name" ]] && continue
            
            local is_exported="false"
            if [[ "$interface_name" =~ ^[A-Z] ]]; then
                is_exported="true"
            fi
            
            # Build content
            local content="Interface: $interface_name | Context: $context | Parent: $parent_name"
            [[ "$is_exported" == "true" ]] && content="$content | Exported: true"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg type_name "$interface_name" \
                --arg type_kind "interface" \
                --arg is_exported "$is_exported" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "go",
                        type_name: $type_name,
                        type_kind: $type_kind,
                        is_exported: ($is_exported == "true"),
                        content_type: "code_type",
                        extraction_method: "go_parser"
                    }
                }' | jq -c
        done <<< "$interfaces"
    fi
}

#######################################
# Extract Go module information
# 
# Gets module name and dependencies from go.mod
#
# Arguments:
#   $1 - Path to go.mod file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with module information
#######################################
extractor::lib::go::extract_module_info() {
    local file="$1"
    local context="${2:-go}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract module name and Go version
    local module_name=$(grep "^module " "$file" | cut -d' ' -f2)
    local go_version=$(grep "^go " "$file" | cut -d' ' -f2)
    
    # Count dependencies
    local dep_count=0
    if grep -q "^require (" "$file"; then
        dep_count=$(awk '/^require \(/, /^\)/ { if ($1 != "require" && $1 != ")" && $1 != "") count++ } END { print count+0 }' "$file")
    else
        dep_count=$(grep -c "^require " "$file" 2>/dev/null || echo "0")
    fi
    
    # Build content
    local content="Module: $module_name | Context: $context | Parent: $parent_name"
    [[ -n "$go_version" ]] && content="$content | Go: $go_version"
    content="$content | Dependencies: $dep_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg module_name "$module_name" \
        --arg go_version "$go_version" \
        --arg dependency_count "$dep_count" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "go",
                module_name: $module_name,
                go_version: $go_version,
                dependency_count: ($dependency_count | tonumber),
                content_type: "module_config",
                extraction_method: "go_mod_parser"
            }
        }' | jq -c
}

#######################################
# Extract all information from Go files
# 
# Main entry point that extracts functions, types, and module info
#
# Arguments:
#   $1 - Path to file or directory
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with all Go information
#######################################
extractor::lib::go::extract_all() {
    local path="$1"
    local context="${2:-go}"
    local parent_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        case "$path" in
            *.go)
                extractor::lib::go::extract_functions "$path" "$context" "$parent_name" 2>/dev/null || true
                extractor::lib::go::extract_types "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
            go.mod)
                extractor::lib::go::extract_module_info "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
        esac
    elif [[ -d "$path" ]]; then
        # Directory - find relevant files
        while IFS= read -r file; do
            extractor::lib::go::extract_all "$file" "$context" "$parent_name"
        done < <(find "$path" -type f \( -name "*.go" -o -name "go.mod" \) 2>/dev/null)
    fi
}

# Export functions
export -f extractor::lib::go::extract_functions
export -f extractor::lib::go::extract_types
export -f extractor::lib::go::extract_module_info
export -f extractor::lib::go::extract_all