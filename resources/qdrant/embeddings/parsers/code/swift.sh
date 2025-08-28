#!/usr/bin/env bash
# Swift Code Extractor Library
# Extracts functions, classes, structs, protocols, and metadata from Swift files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract functions from Swift file
# 
# Finds function definitions with access control and modifiers
#
# Arguments:
#   $1 - Path to Swift file
#   $2 - Context (component type like 'api', 'cli', etc.)
#   $3 - Parent name (scenario name)
# Returns: JSON lines with function information
#######################################
extractor::lib::swift::extract_functions() {
    local file="$1"
    local context="${2:-swift}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    log::debug "Extracting Swift functions from: $file" >&2
    
    # Find function definitions
    local functions=$(grep -E "^\s*(public|private|internal|fileprivate|open)?\s*(static|class)?\s*(override)?\s*func\s+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/.*func\s+([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' || echo "")
    
    if [[ -n "$functions" ]]; then
        while IFS= read -r function_name; do
            [[ -z "$function_name" ]] && continue
            
            # Get full function declaration
            local func_line=$(grep -E "func\s+$function_name" "$file" | head -1)
            
            # Extract access control
            local access_control="internal"  # Swift default
            if echo "$func_line" | grep -q "public"; then
                access_control="public"
            elif echo "$func_line" | grep -q "private"; then
                access_control="private"
            elif echo "$func_line" | grep -q "fileprivate"; then
                access_control="fileprivate"
            elif echo "$func_line" | grep -q "open"; then
                access_control="open"
            fi
            
            # Check modifiers
            local is_static="false"
            local is_class="false"
            local is_override="false"
            local is_mutating="false"
            
            if echo "$func_line" | grep -q "static"; then
                is_static="true"
            elif echo "$func_line" | grep -q "class func"; then
                is_class="true"
            fi
            
            if echo "$func_line" | grep -q "override"; then
                is_override="true"
            fi
            
            if echo "$func_line" | grep -q "mutating"; then
                is_mutating="true"
            fi
            
            # Determine function type
            local func_type="function"
            if [[ "$function_name" == "init" ]]; then
                func_type="initializer"
            elif [[ "$function_name" == "deinit" ]]; then
                func_type="deinitializer"
            elif echo "$func_line" | grep -q "subscript"; then
                func_type="subscript"
            elif [[ "$is_static" == "true" ]] || [[ "$is_class" == "true" ]]; then
                func_type="type_method"
            fi
            
            # Extract return type
            local return_type=""
            if [[ "$func_line" =~ ->[[:space:]]*([A-Za-z_][A-Za-z0-9_<>?!]*) ]]; then
                return_type="${BASH_REMATCH[1]}"
            fi
            
            # Check for async/throws modifiers
            local is_async="false"
            local can_throw="false"
            if echo "$func_line" | grep -q "async"; then
                is_async="true"
            fi
            if echo "$func_line" | grep -q "throws"; then
                can_throw="true"
            fi
            
            # Look for documentation comments
            local line_num=$(grep -n "func.*$function_name" "$file" | head -1 | cut -d: -f1)
            local description=""
            local parameters=""
            
            if [[ -n "$line_num" ]]; then
                # Look backwards for Swift doc comments (///)
                local check_line=$((line_num - 1))
                while [[ $check_line -gt 0 ]]; do
                    local line_content=$(sed -n "${check_line}p" "$file")
                    if [[ "$line_content" =~ ^[[:space:]]*///[[:space:]]*(.*)$ ]]; then
                        local comment="${BASH_REMATCH[1]}"
                        if [[ "$comment" =~ ^-[[:space:]]*Parameter[[:space:]]+([^:]+):[[:space:]]*(.*)$ ]]; then
                            parameters="${BASH_REMATCH[1]}: ${BASH_REMATCH[2]}; $parameters"
                        elif [[ "$comment" =~ ^-[[:space:]]*Returns:[[:space:]]*(.*)$ ]]; then
                            # Return description already extracted from type annotation
                            :
                        elif [[ -z "$description" ]]; then
                            description="$comment"
                        fi
                        ((check_line--))
                    else
                        break
                    fi
                done
            fi
            
            # Get function signature
            local signature=$(grep "func.*$function_name" "$file" | head -1 | sed 's/^[[:space:]]*//')
            
            # Build content
            local content="Function: $function_name | Context: $context | Parent: $parent_name"
            content="$content | Access: $access_control | Type: $func_type"
            [[ "$is_static" == "true" ]] && content="$content | Static: true"
            [[ "$is_class" == "true" ]] && content="$content | Class method: true"
            [[ "$is_override" == "true" ]] && content="$content | Override: true"
            [[ "$is_mutating" == "true" ]] && content="$content | Mutating: true"
            [[ "$is_async" == "true" ]] && content="$content | Async: true"
            [[ "$can_throw" == "true" ]] && content="$content | Throws: true"
            [[ -n "$return_type" ]] && content="$content | Returns: $return_type"
            [[ -n "$description" ]] && content="$content | Description: $description"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg function_name "$function_name" \
                --arg function_type "$func_type" \
                --arg access_control "$access_control" \
                --arg signature "$signature" \
                --arg is_static "$is_static" \
                --arg is_class "$is_class" \
                --arg is_override "$is_override" \
                --arg is_mutating "$is_mutating" \
                --arg is_async "$is_async" \
                --arg can_throw "$can_throw" \
                --arg return_type "$return_type" \
                --arg parameters "$parameters" \
                --arg description "$description" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "swift",
                        function_name: $function_name,
                        function_type: $function_type,
                        access_control: $access_control,
                        signature: $signature,
                        is_static: ($is_static == "true"),
                        is_class_method: ($is_class == "true"),
                        is_override: ($is_override == "true"),
                        is_mutating: ($is_mutating == "true"),
                        is_async: ($is_async == "true"),
                        can_throw: ($can_throw == "true"),
                        return_type: $return_type,
                        parameters: $parameters,
                        description: $description,
                        content_type: "code_function",
                        extraction_method: "swift_parser"
                    }
                }' | jq -c
        done <<< "$functions"
    fi
}

#######################################
# Extract types (classes, structs, protocols, enums) from Swift file
# 
# Finds type definitions with conformances and inheritance
#
# Arguments:
#   $1 - Path to Swift file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with type information
#######################################
extractor::lib::swift::extract_types() {
    local file="$1"
    local context="${2:-swift}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find type definitions
    local types=$(grep -E "^\s*(public|private|internal|fileprivate|open)?\s*(final|open)?\s*(class|struct|protocol|enum|extension)\s+([A-Za-z_][A-Za-z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/.*(class|struct|protocol|enum|extension)\s+([A-Za-z_][A-Za-z0-9_]*).*/\2/' || echo "")
    
    if [[ -n "$types" ]]; then
        while IFS= read -r type_name; do
            [[ -z "$type_name" ]] && continue
            
            # Get full type declaration
            local type_line=$(grep -E "(class|struct|protocol|enum|extension)\s+$type_name" "$file" | head -1)
            
            # Determine type category
            local type_category="class"
            if echo "$type_line" | grep -q "struct"; then
                type_category="struct"
            elif echo "$type_line" | grep -q "protocol"; then
                type_category="protocol"
            elif echo "$type_line" | grep -q "enum"; then
                type_category="enum"
            elif echo "$type_line" | grep -q "extension"; then
                type_category="extension"
            fi
            
            # Extract access control
            local access_control="internal"
            if echo "$type_line" | grep -q "public"; then
                access_control="public"
            elif echo "$type_line" | grep -q "private"; then
                access_control="private"
            elif echo "$type_line" | grep -q "fileprivate"; then
                access_control="fileprivate"
            elif echo "$type_line" | grep -q "open"; then
                access_control="open"
            fi
            
            # Check modifiers
            local is_final="false"
            local is_open="false"
            if echo "$type_line" | grep -q "final"; then
                is_final="true"
            elif echo "$type_line" | grep -q "open"; then
                is_open="true"
            fi
            
            # Extract inheritance/conformances
            local inheritance=""
            local conformances=""
            if [[ "$type_line" =~ :[[:space:]]*([^{]+) ]]; then
                local conformance_list="${BASH_REMATCH[1]}"
                conformance_list=$(echo "$conformance_list" | tr ',' '\n' | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
                
                # For classes, first item is usually superclass
                if [[ "$type_category" == "class" ]]; then
                    inheritance=$(echo "$conformance_list" | head -1)
                    conformances=$(echo "$conformance_list" | tail -n +2 | tr '\n' ', ' | sed 's/,$//')
                else
                    conformances=$(echo "$conformance_list" | tr '\n' ', ' | sed 's/,$//')
                fi
            fi
            
            # Count members within type
            local type_line_num=$(grep -n "(class|struct|protocol|enum|extension).*$type_name" "$file" | head -1 | cut -d: -f1)
            local end_line_num=$(awk -v start=$type_line_num 'NR > start && /^[[:space:]]*}[[:space:]]*$/ {print NR; exit}' "$file")
            
            local method_count=0
            local property_count=0
            local enum_case_count=0
            
            if [[ -n "$type_line_num" && -n "$end_line_num" ]]; then
                method_count=$(sed -n "${type_line_num},${end_line_num}p" "$file" | grep -c "func " 2>/dev/null || echo "0")
                property_count=$(sed -n "${type_line_num},${end_line_num}p" "$file" | grep -c "var \|let " 2>/dev/null || echo "0")
                if [[ "$type_category" == "enum" ]]; then
                    enum_case_count=$(sed -n "${type_line_num},${end_line_num}p" "$file" | grep -c "case " 2>/dev/null || echo "0")
                fi
            fi
            
            # Build content
            local content="Type: $type_name | Context: $context | Parent: $parent_name | Category: $type_category"
            content="$content | Access: $access_control | Methods: $method_count | Properties: $property_count"
            [[ "$is_final" == "true" ]] && content="$content | Final: true"
            [[ "$is_open" == "true" ]] && content="$content | Open: true"
            [[ -n "$inheritance" ]] && content="$content | Inherits: $inheritance"
            [[ -n "$conformances" ]] && content="$content | Conforms: $conformances"
            [[ "$type_category" == "enum" ]] && [[ $enum_case_count -gt 0 ]] && content="$content | Cases: $enum_case_count"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg type_name "$type_name" \
                --arg type_category "$type_category" \
                --arg access_control "$access_control" \
                --arg method_count "$method_count" \
                --arg property_count "$property_count" \
                --arg enum_case_count "$enum_case_count" \
                --arg is_final "$is_final" \
                --arg is_open "$is_open" \
                --arg inheritance "$inheritance" \
                --arg conformances "$conformances" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "swift",
                        type_name: $type_name,
                        type_category: $type_category,
                        access_control: $access_control,
                        method_count: ($method_count | tonumber),
                        property_count: ($property_count | tonumber),
                        enum_case_count: ($enum_case_count | tonumber),
                        is_final: ($is_final == "true"),
                        is_open: ($is_open == "true"),
                        inheritance: $inheritance,
                        conformances: $conformances,
                        content_type: "code_class",
                        extraction_method: "swift_parser"
                    }
                }' | jq -c
        done <<< "$types"
    fi
}

#######################################
# Extract Package.swift information
# 
# Gets Swift package dependencies and project metadata
#
# Arguments:
#   $1 - Path to Package.swift
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with Package information
#######################################
extractor::lib::swift::extract_package() {
    local file="$1"
    local context="${2:-swift}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract package name
    local package_name=$(grep "name:" "$file" | head -1 | sed 's/.*name:[[:space:]]*"//; s/".*//')
    
    # Extract Swift version
    local swift_version=$(grep -E "(swift-tools-version|swiftLanguageVersions)" "$file" | head -1 | sed 's/.*://; s/[^0-9.]//g')
    
    # Count dependencies
    local dependency_count=$(grep -c "\.package" "$file" 2>/dev/null || echo "0")
    
    # Count targets
    local target_count=$(grep -c "\.target\|\.executableTarget\|\.testTarget" "$file" 2>/dev/null || echo "0")
    
    # Extract platforms
    local platforms=$(grep -A 5 "platforms:" "$file" | grep -o "\.[a-zA-Z]*" | tr '\n' ', ' | sed 's/^\.//g; s/,\./,/g; s/,$//')
    
    # Check for common frameworks
    local has_foundation="false"
    local has_uikit="false"
    local has_swiftui="false"
    
    if grep -q "Foundation" "$file" 2>/dev/null; then
        has_foundation="true"
    fi
    if grep -q "UIKit" "$file" 2>/dev/null; then
        has_uikit="true"
    fi
    if grep -q "SwiftUI" "$file" 2>/dev/null; then
        has_swiftui="true"
    fi
    
    # Extract popular dependencies
    local dependencies=$(grep "url:" "$file" | sed 's/.*url:[[:space:]]*"//; s/".*//' | sed 's/.*\///; s/\.git$//' | head -10 | tr '\n' ', ' | sed 's/,$//')
    
    # Build content
    local content="Package: $(basename "$(dirname "$file")") | Context: $context | Parent: $parent_name"
    [[ -n "$package_name" ]] && content="$content | Name: $package_name"
    [[ -n "$swift_version" ]] && content="$content | Swift: $swift_version"
    content="$content | Dependencies: $dependency_count | Targets: $target_count"
    [[ -n "$platforms" ]] && content="$content | Platforms: $platforms"
    [[ "$has_swiftui" == "true" ]] && content="$content | SwiftUI: true"
    [[ -n "$dependencies" ]] && content="$content | Deps: $dependencies"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg package_name "$package_name" \
        --arg swift_version "$swift_version" \
        --arg dependency_count "$dependency_count" \
        --arg target_count "$target_count" \
        --arg platforms "$platforms" \
        --arg has_foundation "$has_foundation" \
        --arg has_uikit "$has_uikit" \
        --arg has_swiftui "$has_swiftui" \
        --arg dependencies "$dependencies" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "swift",
                package_name: $package_name,
                swift_version: $swift_version,
                dependency_count: ($dependency_count | tonumber),
                target_count: ($target_count | tonumber),
                platforms: $platforms,
                has_foundation: ($has_foundation == "true"),
                has_uikit: ($has_uikit == "true"),
                has_swiftui: ($has_swiftui == "true"),
                dependencies: $dependencies,
                content_type: "package_config",
                extraction_method: "swift_package_parser"
            }
        }' | jq -c
}

#######################################
# Extract all information from Swift files
# 
# Main entry point that extracts types, functions, and package info
#
# Arguments:
#   $1 - Path to file or directory
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with all Swift information
#######################################
extractor::lib::swift::extract_all() {
    local path="$1"
    local context="${2:-swift}"
    local parent_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        case "$path" in
            *.swift)
                extractor::lib::swift::extract_functions "$path" "$context" "$parent_name" 2>/dev/null || true
                extractor::lib::swift::extract_types "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
            Package.swift)
                extractor::lib::swift::extract_package "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
        esac
    elif [[ -d "$path" ]]; then
        # Directory - find relevant files
        while IFS= read -r file; do
            extractor::lib::swift::extract_all "$file" "$context" "$parent_name"
        done < <(find "$path" -type f \( -name "*.swift" -o -name "Package.swift" \) 2>/dev/null)
    fi
}

# Export functions
export -f extractor::lib::swift::extract_functions
export -f extractor::lib::swift::extract_types
export -f extractor::lib::swift::extract_package
export -f extractor::lib::swift::extract_all