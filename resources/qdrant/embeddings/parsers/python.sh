#!/usr/bin/env bash
# Python Code Extractor Library
# Extracts functions, classes, imports, and metadata from Python files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract functions from Python file
# 
# Finds function definitions and methods
#
# Arguments:
#   $1 - Path to Python file
#   $2 - Context (component type like 'api', 'cli', etc.)
#   $3 - Parent name (scenario name)
# Returns: JSON lines with function information
#######################################
extractor::lib::python::extract_functions() {
    local file="$1"
    local context="${2:-python}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    log::debug "Extracting Python functions from: $file" >&2
    
    # Find function definitions
    local functions=$(grep -E "^def[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/^def[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' || echo "")
    
    if [[ -n "$functions" ]]; then
        while IFS= read -r func_name; do
            [[ -z "$func_name" ]] && continue
            
            # Check if function is "private" (starts with _)
            local is_private="false"
            if [[ "$func_name" =~ ^_ ]]; then
                is_private="true"
            fi
            
            # Get function signature
            local signature=$(grep "^def $func_name" "$file" 2>/dev/null | head -1 || echo "")
            
            # Look for docstring or comment
            local description=""
            local line_num=$(grep -n "^def $func_name" "$file" | head -1 | cut -d: -f1)
            
            if [[ -n "$line_num" ]]; then
                # Look for docstring (next non-empty line after function def)
                local desc_line=$((line_num + 1))
                local line_content=$(sed -n "${desc_line}p" "$file")
                
                # Check for docstring
                if [[ "$line_content" =~ ^[[:space:]]*\"\"\"(.*)\"\"\" ]]; then
                    description="${BASH_REMATCH[1]}"
                elif [[ "$line_content" =~ ^[[:space:]]*\"\"\"(.*) ]]; then
                    # Multi-line docstring start
                    description="${BASH_REMATCH[1]}"
                    ((desc_line++))
                    while [[ $desc_line -le $((line_num + 5)) ]]; do
                        line_content=$(sed -n "${desc_line}p" "$file")
                        if [[ "$line_content" =~ \"\"\" ]]; then
                            break
                        elif [[ -n "$line_content" ]]; then
                            description="$description $line_content"
                        fi
                        ((desc_line++))
                    done
                elif [[ "$line_content" =~ ^[[:space:]]*#[[:space:]]*(.+)$ ]]; then
                    description="${BASH_REMATCH[1]}"
                fi
            fi
            
            # Detect if it's a method (inside a class)
            local function_type="function"
            local class_context=""
            
            # Simple heuristic: if there's indentation before def, it's likely a method
            local def_line=$(grep -n "^def $func_name" "$file" | head -1)
            if [[ -n "$def_line" ]]; then
                local line_before=$(($(echo "$def_line" | cut -d: -f1) - 1))
                # Look backwards for class definition
                while [[ $line_before -gt 0 ]]; do
                    local check_line=$(sed -n "${line_before}p" "$file")
                    if [[ "$check_line" =~ ^class[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*) ]]; then
                        function_type="method"
                        class_context="${BASH_REMATCH[1]}"
                        break
                    elif [[ "$check_line" =~ ^def[[:space:]]+ ]]; then
                        # Found another function at top level, stop
                        break
                    elif [[ "$check_line" =~ ^[a-zA-Z] ]]; then
                        # Found top-level statement, stop
                        break
                    fi
                    ((line_before--))
                done
            fi
            
            # Build content
            local content="Function: $func_name | Type: $function_type | Context: $context | Parent: $parent_name"
            [[ -n "$class_context" ]] && content="$content | Class: $class_context"
            [[ -n "$description" ]] && content="$content | Description: $description"
            [[ "$is_private" == "true" ]] && content="$content | Private: true"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg function_name "$func_name" \
                --arg function_type "$function_type" \
                --arg class_context "$class_context" \
                --arg signature "$signature" \
                --arg description "$description" \
                --arg is_private "$is_private" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "python",
                        function_name: $function_name,
                        function_type: $function_type,
                        class_context: $class_context,
                        signature: $signature,
                        description: $description,
                        is_private: ($is_private == "true"),
                        content_type: "code_function",
                        extraction_method: "python_parser"
                    }
                }' | jq -c
        done <<< "$functions"
    fi
}

#######################################
# Extract classes from Python file
# 
# Finds class definitions and their methods
#
# Arguments:
#   $1 - Path to Python file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with class information
#######################################
extractor::lib::python::extract_classes() {
    local file="$1"
    local context="${2:-python}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find class definitions
    local classes=$(grep -E "^class[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/^class[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' || echo "")
    
    if [[ -n "$classes" ]]; then
        while IFS= read -r class_name; do
            [[ -z "$class_name" ]] && continue
            
            # Count methods in class
            local method_count=0
            local in_class=false
            local indent_level=0
            
            while IFS= read -r line; do
                if [[ "$line" =~ ^class[[:space:]]+$class_name ]]; then
                    in_class=true
                    # Determine base indentation level
                    continue
                fi
                
                if [[ "$in_class" == "true" ]]; then
                    # If we hit another top-level class or function, we're done
                    if [[ "$line" =~ ^(class|def)[[:space:]]+ ]]; then
                        break
                    fi
                    
                    # Count method definitions (indented def statements)
                    if [[ "$line" =~ ^[[:space:]]+def[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*) ]]; then
                        ((method_count++))
                    fi
                fi
            done < "$file"
            
            # Get inheritance info
            local inheritance=""
            local class_line=$(grep "^class $class_name" "$file" | head -1)
            if [[ "$class_line" =~ class[[:space:]]+$class_name\(([^)]+)\) ]]; then
                inheritance="${BASH_REMATCH[1]}"
            fi
            
            # Build content
            local content="Class: $class_name | Context: $context | Parent: $parent_name"
            content="$content | Methods: $method_count"
            [[ -n "$inheritance" ]] && content="$content | Inherits: $inheritance"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg class_name "$class_name" \
                --arg method_count "$method_count" \
                --arg inheritance "$inheritance" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "python",
                        class_name: $class_name,
                        method_count: ($method_count | tonumber),
                        inheritance: $inheritance,
                        content_type: "code_class",
                        extraction_method: "python_parser"
                    }
                }' | jq -c
        done <<< "$classes"
    fi
}

#######################################
# Extract requirements.txt information
# 
# Gets Python dependencies from requirements file
#
# Arguments:
#   $1 - Path to requirements.txt file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with requirements information
#######################################
extractor::lib::python::extract_requirements() {
    local file="$1"
    local context="${2:-python}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract package names (ignore versions and comments)
    local packages=$(grep -E "^[a-zA-Z][a-zA-Z0-9_-]*" "$file" 2>/dev/null | \
        sed 's/[<>=!].*//' | sort -u || echo "")
    
    local package_count=0
    local packages_json="[]"
    
    if [[ -n "$packages" ]]; then
        packages_json=$(echo "$packages" | jq -R . | jq -s .)
        package_count=$(echo "$packages_json" | jq 'length')
    fi
    
    # Build content
    local content="Requirements: $(basename "$file") | Context: $context | Parent: $parent_name"
    content="$content | Packages: $package_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg package_count "$package_count" \
        --argjson packages "$packages_json" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "python",
                package_count: ($package_count | tonumber),
                packages: $packages,
                content_type: "requirements_config",
                extraction_method: "requirements_parser"
            }
        }' | jq -c
}

#######################################
# Extract all information from Python files
# 
# Main entry point that extracts functions, classes, and requirements
#
# Arguments:
#   $1 - Path to file or directory
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with all Python information
#######################################
extractor::lib::python::extract_all() {
    local path="$1"
    local context="${2:-python}"
    local parent_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        case "$path" in
            *.py)
                extractor::lib::python::extract_functions "$path" "$context" "$parent_name" 2>/dev/null || true
                extractor::lib::python::extract_classes "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
            requirements*.txt)
                extractor::lib::python::extract_requirements "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
        esac
    elif [[ -d "$path" ]]; then
        # Directory - find relevant files
        while IFS= read -r file; do
            extractor::lib::python::extract_all "$file" "$context" "$parent_name"
        done < <(find "$path" -type f \( -name "*.py" -o -name "requirements*.txt" \) 2>/dev/null)
    fi
}

# Export functions
export -f extractor::lib::python::extract_functions
export -f extractor::lib::python::extract_classes
export -f extractor::lib::python::extract_requirements
export -f extractor::lib::python::extract_all