#!/usr/bin/env bash
# Rust Code Extractor Library
# Extracts functions, structs, traits, and metadata from Rust files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract functions from Rust file
# 
# Finds function definitions and associated methods
#
# Arguments:
#   $1 - Path to Rust file
#   $2 - Context (component type like 'api', 'cli', etc.)
#   $3 - Parent name (scenario name)
# Returns: JSON lines with function information
#######################################
extractor::lib::rust::extract_functions() {
    local file="$1"
    local context="${2:-rust}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    log::debug "Extracting Rust functions from: $file" >&2
    
    # Find function definitions (fn keyword)
    local functions=$(grep -E "^[[:space:]]*fn[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/^[[:space:]]*fn[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' || echo "")
    
    # Find impl block methods
    local methods=$(awk '/^impl[[:space:]]/ { in_impl=1; impl_type=$2 } 
                         in_impl && /^[[:space:]]*fn[[:space:]]+[a-zA-Z_]/ { 
                             match($0, /fn[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)/, arr)
                             if (arr[1]) print arr[1] ":" impl_type
                         }
                         in_impl && /^}/ { in_impl=0 }' "$file" 2>/dev/null || echo "")
    
    # Process standalone functions
    if [[ -n "$functions" ]]; then
        while IFS= read -r func_name; do
            [[ -z "$func_name" ]] && continue
            
            # Check if function is public
            local is_public="false"
            if grep -q "pub fn $func_name" "$file" 2>/dev/null; then
                is_public="true"
            fi
            
            # Get function signature
            local signature=$(grep "fn $func_name" "$file" 2>/dev/null | head -1 | sed 's/^[[:space:]]*//' || echo "")
            
            # Look for doc comments (///)
            local description=""
            local line_num=$(grep -n "fn $func_name" "$file" | head -1 | cut -d: -f1)
            
            if [[ -n "$line_num" ]]; then
                local desc_line=$((line_num - 1))
                while [[ $desc_line -gt 0 ]]; do
                    local line_content=$(sed -n "${desc_line}p" "$file")
                    if [[ "$line_content" =~ ^[[:space:]]*///[[:space:]]*(.+)$ ]]; then
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
            [[ "$is_public" == "true" ]] && content="$content | Public: true"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg function_name "$func_name" \
                --arg signature "$signature" \
                --arg description "$description" \
                --arg is_public "$is_public" \
                --arg function_type "function" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "rust",
                        function_name: $function_name,
                        function_type: $function_type,
                        signature: $signature,
                        description: $description,
                        is_public: ($is_public == "true"),
                        content_type: "code_function",
                        extraction_method: "rust_parser"
                    }
                }' | jq -c
        done <<< "$functions"
    fi
    
    # Process impl methods
    if [[ -n "$methods" ]]; then
        while IFS= read -r method_info; do
            [[ -z "$method_info" ]] && continue
            
            local method_name="${method_info%:*}"
            local impl_type="${method_info#*:}"
            
            # Check if method is public
            local is_public="false"
            if grep -q "pub fn $method_name" "$file" 2>/dev/null; then
                is_public="true"
            fi
            
            # Build content
            local content="Method: $method_name | Context: $context | Parent: $parent_name"
            content="$content | Impl: $impl_type"
            [[ "$is_public" == "true" ]] && content="$content | Public: true"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg function_name "$method_name" \
                --arg impl_type "$impl_type" \
                --arg is_public "$is_public" \
                --arg function_type "method" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "rust",
                        function_name: $function_name,
                        function_type: $function_type,
                        impl_type: $impl_type,
                        is_public: ($is_public == "true"),
                        content_type: "code_function",
                        extraction_method: "rust_parser"
                    }
                }' | jq -c
        done <<< "$methods"
    fi
}

#######################################
# Extract structs and traits from Rust file
# 
# Finds struct and trait definitions
#
# Arguments:
#   $1 - Path to Rust file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with type information
#######################################
extractor::lib::rust::extract_types() {
    local file="$1"
    local context="${2:-rust}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find struct definitions
    local structs=$(grep -E "^[[:space:]]*struct[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/^[[:space:]]*struct[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' || echo "")
    
    # Find trait definitions
    local traits=$(grep -E "^[[:space:]]*trait[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/^[[:space:]]*trait[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' || echo "")
    
    # Find enum definitions
    local enums=$(grep -E "^[[:space:]]*enum[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/^[[:space:]]*enum[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' || echo "")
    
    # Process structs
    if [[ -n "$structs" ]]; then
        while IFS= read -r struct_name; do
            [[ -z "$struct_name" ]] && continue
            
            # Check if public
            local is_public="false"
            if grep -q "pub struct $struct_name" "$file" 2>/dev/null; then
                is_public="true"
            fi
            
            # Count fields (rough approximation)
            local field_count=0
            local in_struct=false
            local brace_count=0
            
            while IFS= read -r line; do
                if [[ "$line" =~ struct[[:space:]]+$struct_name ]]; then
                    in_struct=true
                fi
                
                if [[ "$in_struct" == "true" ]]; then
                    if [[ "$line" == *"{"* ]]; then
                        brace_count=$((brace_count + 1))
                    fi
                    
                    # Count field lines (field_name: type)
                    if [[ "$line" =~ ^[[:space:]]*[a-zA-Z_][a-zA-Z0-9_]*:[[:space:]] ]]; then
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
            [[ "$is_public" == "true" ]] && content="$content | Public: true"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg type_name "$struct_name" \
                --arg type_kind "struct" \
                --arg field_count "$field_count" \
                --arg is_public "$is_public" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "rust",
                        type_name: $type_name,
                        type_kind: $type_kind,
                        field_count: ($field_count | tonumber),
                        is_public: ($is_public == "true"),
                        content_type: "code_type",
                        extraction_method: "rust_parser"
                    }
                }' | jq -c
        done <<< "$structs"
    fi
    
    # Process traits
    if [[ -n "$traits" ]]; then
        while IFS= read -r trait_name; do
            [[ -z "$trait_name" ]] && continue
            
            local is_public="false"
            if grep -q "pub trait $trait_name" "$file" 2>/dev/null; then
                is_public="true"
            fi
            
            # Build content
            local content="Trait: $trait_name | Context: $context | Parent: $parent_name"
            [[ "$is_public" == "true" ]] && content="$content | Public: true"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg type_name "$trait_name" \
                --arg type_kind "trait" \
                --arg is_public "$is_public" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "rust",
                        type_name: $type_name,
                        type_kind: $type_kind,
                        is_public: ($is_public == "true"),
                        content_type: "code_type",
                        extraction_method: "rust_parser"
                    }
                }' | jq -c
        done <<< "$traits"
    fi
    
    # Process enums
    if [[ -n "$enums" ]]; then
        while IFS= read -r enum_name; do
            [[ -z "$enum_name" ]] && continue
            
            local is_public="false"
            if grep -q "pub enum $enum_name" "$file" 2>/dev/null; then
                is_public="true"
            fi
            
            # Build content
            local content="Enum: $enum_name | Context: $context | Parent: $parent_name"
            [[ "$is_public" == "true" ]] && content="$content | Public: true"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg type_name "$enum_name" \
                --arg type_kind "enum" \
                --arg is_public "$is_public" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "rust",
                        type_name: $type_name,
                        type_kind: $type_kind,
                        is_public: ($is_public == "true"),
                        content_type: "code_type",
                        extraction_method: "rust_parser"
                    }
                }' | jq -c
        done <<< "$enums"
    fi
}

#######################################
# Extract Cargo.toml information
# 
# Gets package metadata and dependencies
#
# Arguments:
#   $1 - Path to Cargo.toml file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with Cargo information
#######################################
extractor::lib::rust::extract_cargo_toml() {
    local file="$1"
    local context="${2:-rust}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract package metadata (basic TOML parsing)
    local name=$(grep "^name = " "$file" | head -1 | sed 's/.*= *"//' | tr -d '"')
    local version=$(grep "^version = " "$file" | head -1 | sed 's/.*= *"//' | tr -d '"')
    local edition=$(grep "^edition = " "$file" | head -1 | sed 's/.*= *"//' | tr -d '"')
    local description=$(grep "^description = " "$file" | head -1 | sed 's/.*= *"//' | tr -d '"')
    
    # Extract dependencies (count lines in [dependencies] section)
    local dep_count=0
    local in_deps=false
    while IFS= read -r line; do
        if [[ "$line" =~ ^\[dependencies\] ]]; then
            in_deps=true
            continue
        elif [[ "$line" =~ ^\[ ]]; then
            in_deps=false
        elif [[ "$in_deps" == "true" && "$line" =~ ^[a-zA-Z] ]]; then
            ((dep_count++))
        fi
    done < "$file"
    
    # Build content
    local content="Cargo: $name | Version: $version | Context: $context | Parent: $parent_name"
    [[ -n "$description" ]] && content="$content | Description: $description"
    [[ -n "$edition" ]] && content="$content | Edition: $edition"
    content="$content | Dependencies: $dep_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg package_name "$name" \
        --arg package_version "$version" \
        --arg edition "$edition" \
        --arg description "$description" \
        --arg dependency_count "$dep_count" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "rust",
                package_name: $package_name,
                package_version: $package_version,
                edition: $edition,
                description: $description,
                dependency_count: ($dependency_count | tonumber),
                content_type: "package_config",
                extraction_method: "cargo_toml_parser"
            }
        }' | jq -c
}

#######################################
# Extract all information from Rust files
# 
# Main entry point that extracts functions, types, and Cargo info
#
# Arguments:
#   $1 - Path to file or directory
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with all Rust information
#######################################
extractor::lib::rust::extract_all() {
    local path="$1"
    local context="${2:-rust}"
    local parent_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        case "$path" in
            *.rs)
                extractor::lib::rust::extract_functions "$path" "$context" "$parent_name" 2>/dev/null || true
                extractor::lib::rust::extract_types "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
            Cargo.toml)
                extractor::lib::rust::extract_cargo_toml "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
        esac
    elif [[ -d "$path" ]]; then
        # Directory - find relevant files
        while IFS= read -r file; do
            extractor::lib::rust::extract_all "$file" "$context" "$parent_name"
        done < <(find "$path" -type f \( -name "*.rs" -o -name "Cargo.toml" \) 2>/dev/null)
    fi
}

# Export functions
export -f extractor::lib::rust::extract_functions
export -f extractor::lib::rust::extract_types
export -f extractor::lib::rust::extract_cargo_toml
export -f extractor::lib::rust::extract_all