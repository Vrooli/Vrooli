#!/usr/bin/env bash
# C# Code Extractor Library
# Extracts classes, methods, properties, and metadata from C# files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract methods from C# file
# 
# Finds method definitions and their visibility
#
# Arguments:
#   $1 - Path to C# file
#   $2 - Context (component type like 'api', 'cli', etc.)
#   $3 - Parent name (scenario name)
# Returns: JSON lines with method information
#######################################
extractor::lib::csharp::extract_methods() {
    local file="$1"
    local context="${2:-csharp}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    log::debug "Extracting C# methods from: $file" >&2
    
    # Find method definitions (public, private, protected, internal)
    local methods=$(grep -E "^[[:space:]]*(public|private|protected|internal)[[:space:]]+(static[[:space:]]+)?[a-zA-Z_<>]+[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]*\(" "$file" 2>/dev/null | \
        sed -E 's/.*[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]*\(.*/\1/' || echo "")
    
    if [[ -n "$methods" ]]; then
        while IFS= read -r method_name; do
            [[ -z "$method_name" ]] && continue
            
            # Skip common non-method matches
            [[ "$method_name" =~ ^(class|interface|enum|using|namespace)$ ]] && continue
            
            # Get method signature and attributes
            local method_line=$(grep -E "[[:space:]]+$method_name[[:space:]]*\(" "$file" | head -1)
            local visibility="private"  # default in C#
            local is_static="false"
            local return_type=""
            local is_async="false"
            
            if [[ "$method_line" =~ public ]]; then
                visibility="public"
            elif [[ "$method_line" =~ protected ]]; then
                visibility="protected"
            elif [[ "$method_line" =~ internal ]]; then
                visibility="internal"
            fi
            
            [[ "$method_line" =~ static ]] && is_static="true"
            [[ "$method_line" =~ async ]] && is_async="true"
            
            # Extract return type (before method name)
            if [[ "$method_line" =~ [[:space:]]([A-Za-z_][A-Za-z0-9_<>]*)[[:space:]]+$method_name[[:space:]]*\( ]]; then
                return_type="${BASH_REMATCH[1]}"
            fi
            
            # Look for XML documentation comments
            local description=""
            local line_num=$(grep -n "$method_name.*(" "$file" | head -1 | cut -d: -f1)
            
            if [[ -n "$line_num" ]]; then
                local desc_line=$((line_num - 1))
                while [[ $desc_line -gt 0 ]]; do
                    local line_content=$(sed -n "${desc_line}p" "$file")
                    if [[ "$line_content" =~ ^[[:space:]]*///[[:space:]]*(.+)$ ]]; then
                        local comment="${BASH_REMATCH[1]}"
                        # Extract from <summary> tags
                        if [[ "$comment" =~ \<summary\>(.+)\</summary\> ]]; then
                            description="${BASH_REMATCH[1]}"
                            break
                        elif [[ "$comment" != *"<"* && -z "$description" ]]; then
                            description="$comment"
                        fi
                        ((desc_line--))
                    else
                        break
                    fi
                done
            fi
            
            # Determine method type
            local method_type="method"
            if [[ "$method_name" == "Main" ]]; then
                method_type="main"
            elif [[ "$method_name" =~ ^Get[A-Z] ]]; then
                method_type="getter"
            elif [[ "$method_name" =~ ^Set[A-Z] ]]; then
                method_type="setter"
            elif [[ "$is_async" == "true" ]]; then
                method_type="async_method"
            fi
            
            # Build content
            local content="Method: $method_name | Context: $context | Parent: $parent_name"
            content="$content | Visibility: $visibility | Type: $method_type"
            [[ -n "$return_type" ]] && content="$content | Returns: $return_type"
            [[ "$is_static" == "true" ]] && content="$content | Static: true"
            [[ "$is_async" == "true" ]] && content="$content | Async: true"
            [[ -n "$description" ]] && content="$content | Description: $description"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg method_name "$method_name" \
                --arg method_type "$method_type" \
                --arg visibility "$visibility" \
                --arg return_type "$return_type" \
                --arg is_static "$is_static" \
                --arg is_async "$is_async" \
                --arg description "$description" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "csharp",
                        method_name: $method_name,
                        method_type: $method_type,
                        visibility: $visibility,
                        return_type: $return_type,
                        is_static: ($is_static == "true"),
                        is_async: ($is_async == "true"),
                        description: $description,
                        content_type: "code_function",
                        extraction_method: "csharp_parser"
                    }
                }' | jq -c
        done <<< "$methods"
    fi
}

#######################################
# Extract classes and interfaces from C# file
# 
# Finds class and interface definitions
#
# Arguments:
#   $1 - Path to C# file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with class information
#######################################
extractor::lib::csharp::extract_classes() {
    local file="$1"
    local context="${2:-csharp}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find class, interface, struct, and enum definitions
    local classes=$(grep -E "^[[:space:]]*(public|internal|private)?[[:space:]]*(class|interface|struct|enum)[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/.*[[:space:]](class|interface|struct|enum)[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*).*/\2:\1/' || echo "")
    
    if [[ -n "$classes" ]]; then
        while IFS= read -r class_info; do
            [[ -z "$class_info" ]] && continue
            
            local class_name="${class_info%:*}"
            local class_type="${class_info#*:}"
            
            # Check visibility
            local visibility="internal"  # default in C#
            if grep -q "public.*$class_type.*$class_name" "$file" 2>/dev/null; then
                visibility="public"
            elif grep -q "private.*$class_type.*$class_name" "$file" 2>/dev/null; then
                visibility="private"
            fi
            
            # Count methods and properties in class
            local method_count=0
            local property_count=0
            local in_class=false
            local brace_count=0
            
            while IFS= read -r line; do
                if [[ "$line" =~ $class_type[[:space:]]+$class_name ]]; then
                    in_class=true
                fi
                
                if [[ "$in_class" == "true" ]]; then
                    # Count braces to track class scope
                    local open_braces="${line//[^\{]/}"
                    local close_braces="${line//[^\}]/}"
                    brace_count=$((brace_count + ${#open_braces} - ${#close_braces}))
                    
                    # Count method definitions
                    if [[ "$line" =~ ^[[:space:]]*(public|private|protected|internal)[[:space:]].*[[:space:]]+[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*\( ]]; then
                        ((method_count++))
                    fi
                    
                    # Count property definitions
                    if [[ "$line" =~ ^[[:space:]]*(public|private|protected|internal)[[:space:]].*[[:space:]]+[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*\{.*get.*set ]]; then
                        ((property_count++))
                    elif [[ "$line" =~ get[[:space:]]*\;.*set[[:space:]]*\; ]]; then
                        ((property_count++))
                    fi
                    
                    if [[ $brace_count -eq 0 && "$line" =~ \} ]]; then
                        break
                    fi
                fi
            done < "$file"
            
            # Extract inheritance information
            local inheritance=""
            local class_line=$(grep "$class_type $class_name" "$file" | head -1)
            if [[ "$class_line" =~ :[[:space:]]*([^{]+) ]]; then
                inheritance="${BASH_REMATCH[1]}"
                # Clean up inheritance (remove extra whitespace)
                inheritance=$(echo "$inheritance" | sed 's/,/ /g' | xargs)
            fi
            
            # Build content
            local content="$class_type: $class_name | Context: $context | Parent: $parent_name"
            content="$content | Visibility: $visibility | Methods: $method_count | Properties: $property_count"
            [[ -n "$inheritance" ]] && content="$content | Inherits: $inheritance"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg class_name "$class_name" \
                --arg class_type "$class_type" \
                --arg visibility "$visibility" \
                --arg method_count "$method_count" \
                --arg property_count "$property_count" \
                --arg inheritance "$inheritance" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "csharp",
                        class_name: $class_name,
                        class_type: $class_type,
                        visibility: $visibility,
                        method_count: ($method_count | tonumber),
                        property_count: ($property_count | tonumber),
                        inheritance: $inheritance,
                        content_type: "code_class",
                        extraction_method: "csharp_parser"
                    }
                }' | jq -c
        done <<< "$classes"
    fi
}

#######################################
# Extract .csproj project information
# 
# Gets .NET project metadata and dependencies
#
# Arguments:
#   $1 - Path to .csproj file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with project information
#######################################
extractor::lib::csharp::extract_csproj() {
    local file="$1"
    local context="${2:-csharp}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract basic project info
    local target_framework=$(grep "<TargetFramework>" "$file" | sed 's/<[^>]*>//g' | xargs)
    local assembly_name=$(grep "<AssemblyName>" "$file" | sed 's/<[^>]*>//g' | xargs)
    local root_namespace=$(grep "<RootNamespace>" "$file" | sed 's/<[^>]*>//g' | xargs)
    local output_type=$(grep "<OutputType>" "$file" | sed 's/<[^>]*>//g' | xargs)
    
    # If no assembly name, use filename
    [[ -z "$assembly_name" ]] && assembly_name=$(basename "$file" .csproj)
    
    # Count package references
    local package_count=$(grep -c "<PackageReference" "$file" 2>/dev/null || echo "0")
    local project_ref_count=$(grep -c "<ProjectReference" "$file" 2>/dev/null || echo "0")
    
    # Check for nullable reference types
    local nullable=$(grep "<Nullable>" "$file" | sed 's/<[^>]*>//g' | xargs)
    
    # Check for language version
    local lang_version=$(grep "<LangVersion>" "$file" | sed 's/<[^>]*>//g' | xargs)
    
    # Build content
    local content="Project: $assembly_name | Context: $context | Parent: $parent_name"
    [[ -n "$target_framework" ]] && content="$content | Framework: $target_framework"
    [[ -n "$output_type" ]] && content="$content | Type: $output_type"
    content="$content | Packages: $package_count | ProjectRefs: $project_ref_count"
    [[ -n "$lang_version" ]] && content="$content | C#: $lang_version"
    [[ -n "$nullable" ]] && content="$content | Nullable: $nullable"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg assembly_name "$assembly_name" \
        --arg target_framework "$target_framework" \
        --arg output_type "$output_type" \
        --arg root_namespace "$root_namespace" \
        --arg package_count "$package_count" \
        --arg project_ref_count "$project_ref_count" \
        --arg nullable "$nullable" \
        --arg lang_version "$lang_version" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "csharp",
                assembly_name: $assembly_name,
                target_framework: $target_framework,
                output_type: $output_type,
                root_namespace: $root_namespace,
                package_count: ($package_count | tonumber),
                project_reference_count: ($project_ref_count | tonumber),
                nullable_enabled: $nullable,
                language_version: $lang_version,
                content_type: "project_config",
                extraction_method: "csproj_parser"
            }
        }' | jq -c
}

#######################################
# Extract all information from C# files
# 
# Main entry point that extracts classes, methods, and project info
#
# Arguments:
#   $1 - Path to file or directory
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with all C# information
#######################################
extractor::lib::csharp::extract_all() {
    local path="$1"
    local context="${2:-csharp}"
    local parent_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        case "$path" in
            *.cs)
                extractor::lib::csharp::extract_methods "$path" "$context" "$parent_name" 2>/dev/null || true
                extractor::lib::csharp::extract_classes "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
            *.csproj)
                extractor::lib::csharp::extract_csproj "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
        esac
    elif [[ -d "$path" ]]; then
        # Directory - find relevant files
        while IFS= read -r file; do
            extractor::lib::csharp::extract_all "$file" "$context" "$parent_name"
        done < <(find "$path" -type f \( -name "*.cs" -o -name "*.csproj" \) 2>/dev/null)
    fi
}

# Export functions
export -f extractor::lib::csharp::extract_methods
export -f extractor::lib::csharp::extract_classes
export -f extractor::lib::csharp::extract_csproj
export -f extractor::lib::csharp::extract_all