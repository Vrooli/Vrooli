#!/usr/bin/env bash
# Kotlin Code Extractor Library
# Extracts functions, classes, objects, and metadata from Kotlin files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract functions from Kotlin file
# 
# Finds function definitions with visibility modifiers and special types
#
# Arguments:
#   $1 - Path to Kotlin file
#   $2 - Context (component type like 'api', 'cli', etc.)
#   $3 - Parent name (scenario name)
# Returns: JSON lines with function information
#######################################
extractor::lib::kotlin::extract_functions() {
    local file="$1"
    local context="${2:-kotlin}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    log::debug "Extracting Kotlin functions from: $file" >&2
    
    # Find function definitions (including extension functions)
    local functions=$(grep -E "^\s*(public|private|internal|protected)?\s*(suspend|inline|infix|operator|tailrec)?\s*fun\s+([A-Za-z_][A-Za-z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/.*fun\s+([A-Za-z_][A-Za-z0-9_]*).*/\1/' || echo "")
    
    if [[ -n "$functions" ]]; then
        while IFS= read -r function_name; do
            [[ -z "$function_name" ]] && continue
            
            # Get full function declaration
            local func_line=$(grep -E "fun\s+$function_name" "$file" | head -1)
            
            # Extract visibility
            local visibility="public"  # Kotlin default
            if echo "$func_line" | grep -q "private"; then
                visibility="private"
            elif echo "$func_line" | grep -q "internal"; then
                visibility="internal"
            elif echo "$func_line" | grep -q "protected"; then
                visibility="protected"
            fi
            
            # Check function modifiers
            local is_suspend="false"
            local is_inline="false"
            local is_infix="false"
            local is_operator="false"
            local is_tailrec="false"
            local is_extension="false"
            
            if echo "$func_line" | grep -q "suspend"; then
                is_suspend="true"
            fi
            if echo "$func_line" | grep -q "inline"; then
                is_inline="true"
            fi
            if echo "$func_line" | grep -q "infix"; then
                is_infix="true"
            fi
            if echo "$func_line" | grep -q "operator"; then
                is_operator="true"
            fi
            if echo "$func_line" | grep -q "tailrec"; then
                is_tailrec="true"
            fi
            # Check if it's an extension function (has receiver type)
            if [[ "$func_line" =~ fun[[:space:]]+[A-Za-z_]+\.[A-Za-z_] ]]; then
                is_extension="true"
            fi
            
            # Determine function type
            local func_type="function"
            if [[ "$is_extension" == "true" ]]; then
                func_type="extension_function"
            elif [[ "$is_operator" == "true" ]]; then
                func_type="operator_function"
            elif [[ "$is_infix" == "true" ]]; then
                func_type="infix_function"
            elif [[ "$is_suspend" == "true" ]]; then
                func_type="suspend_function"
            fi
            
            # Extract return type
            local return_type=""
            if [[ "$func_line" =~ :[[:space:]]*([A-Za-z_][A-Za-z0-9_<>?!]*) ]]; then
                return_type="${BASH_REMATCH[1]}"
            fi
            
            # Extract receiver type for extension functions
            local receiver_type=""
            if [[ "$is_extension" == "true" ]]; then
                if [[ "$func_line" =~ fun[[:space:]]+([A-Za-z_<>?]+)\. ]]; then
                    receiver_type="${BASH_REMATCH[1]}"
                fi
            fi
            
            # Look for KDoc comments
            local line_num=$(grep -n "fun.*$function_name" "$file" | head -1 | cut -d: -f1)
            local description=""
            local parameters=""
            
            if [[ -n "$line_num" ]]; then
                # Look backwards for KDoc (/** ... */)
                local check_line=$((line_num - 1))
                local in_doc=false
                while [[ $check_line -gt 0 ]]; do
                    local line_content=$(sed -n "${check_line}p" "$file")
                    if [[ "$line_content" =~ \*/ ]]; then
                        in_doc=true
                        ((check_line--))
                        continue
                    elif [[ "$line_content" =~ /\*\* ]]; then
                        break
                    elif [[ "$in_doc" == "true" ]]; then
                        if [[ "$line_content" =~ \*[[:space:]]*@param[[:space:]]+([^[:space:]]+)[[:space:]]*(.*)$ ]]; then
                            parameters="${BASH_REMATCH[1]}: ${BASH_REMATCH[2]}; $parameters"
                        elif [[ "$line_content" =~ \*[[:space:]]*@return[[:space:]]*(.*)$ ]]; then
                            # Return description
                            :
                        elif [[ "$line_content" =~ \*[[:space:]]*([^@].*)$ ]] && [[ -z "$description" ]]; then
                            description="${BASH_REMATCH[1]}"
                        fi
                    fi
                    ((check_line--))
                done
            fi
            
            # Get function signature
            local signature=$(grep "fun.*$function_name" "$file" | head -1 | sed 's/^[[:space:]]*//')
            
            # Build content
            local content="Function: $function_name | Context: $context | Parent: $parent_name"
            content="$content | Visibility: $visibility | Type: $func_type"
            [[ "$is_suspend" == "true" ]] && content="$content | Suspend: true"
            [[ "$is_inline" == "true" ]] && content="$content | Inline: true"
            [[ "$is_extension" == "true" ]] && content="$content | Extension: $receiver_type"
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
                --arg visibility "$visibility" \
                --arg signature "$signature" \
                --arg is_suspend "$is_suspend" \
                --arg is_inline "$is_inline" \
                --arg is_infix "$is_infix" \
                --arg is_operator "$is_operator" \
                --arg is_tailrec "$is_tailrec" \
                --arg is_extension "$is_extension" \
                --arg return_type "$return_type" \
                --arg receiver_type "$receiver_type" \
                --arg parameters "$parameters" \
                --arg description "$description" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "kotlin",
                        function_name: $function_name,
                        function_type: $function_type,
                        visibility: $visibility,
                        signature: $signature,
                        is_suspend: ($is_suspend == "true"),
                        is_inline: ($is_inline == "true"),
                        is_infix: ($is_infix == "true"),
                        is_operator: ($is_operator == "true"),
                        is_tailrec: ($is_tailrec == "true"),
                        is_extension: ($is_extension == "true"),
                        return_type: $return_type,
                        receiver_type: $receiver_type,
                        parameters: $parameters,
                        description: $description,
                        content_type: "code_function",
                        extraction_method: "kotlin_parser"
                    }
                }' | jq -c
        done <<< "$functions"
    fi
}

#######################################
# Extract classes and objects from Kotlin file
# 
# Finds class, data class, sealed class, object, and companion object definitions
#
# Arguments:
#   $1 - Path to Kotlin file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with class information
#######################################
extractor::lib::kotlin::extract_classes() {
    local file="$1"
    local context="${2:-kotlin}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find class and object definitions
    local classes=$(grep -E "^\s*(public|private|internal|protected)?\s*(abstract|final|open|sealed|data|inner|enum)?\s*(class|object|interface)\s+([A-Za-z_][A-Za-z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/.*(class|object|interface)\s+([A-Za-z_][A-Za-z0-9_]*).*/\2/' || echo "")
    
    if [[ -n "$classes" ]]; then
        while IFS= read -r class_name; do
            [[ -z "$class_name" ]] && continue
            
            # Get full class declaration
            local class_line=$(grep -E "(class|object|interface)\s+$class_name" "$file" | head -1)
            
            # Determine class type
            local class_type="class"
            if echo "$class_line" | grep -q "object"; then
                class_type="object"
            elif echo "$class_line" | grep -q "interface"; then
                class_type="interface"
            fi
            
            # Extract visibility
            local visibility="public"  # Kotlin default
            if echo "$class_line" | grep -q "private"; then
                visibility="private"
            elif echo "$class_line" | grep -q "internal"; then
                visibility="internal"
            elif echo "$class_line" | grep -q "protected"; then
                visibility="protected"
            fi
            
            # Check class modifiers
            local is_data="false"
            local is_sealed="false"
            local is_abstract="false"
            local is_final="false"
            local is_open="false"
            local is_inner="false"
            local is_enum="false"
            local is_companion="false"
            
            if echo "$class_line" | grep -q "data"; then
                is_data="true"
            fi
            if echo "$class_line" | grep -q "sealed"; then
                is_sealed="true"
            fi
            if echo "$class_line" | grep -q "abstract"; then
                is_abstract="true"
            fi
            if echo "$class_line" | grep -q "final"; then
                is_final="true"
            fi
            if echo "$class_line" | grep -q "open"; then
                is_open="true"
            fi
            if echo "$class_line" | grep -q "inner"; then
                is_inner="true"
            fi
            if echo "$class_line" | grep -q "enum"; then
                is_enum="true"
                class_type="enum"
            fi
            if echo "$class_line" | grep -q "companion object"; then
                is_companion="true"
                class_type="companion_object"
            fi
            
            # Extract inheritance and interfaces
            local superclass=""
            local interfaces=""
            if [[ "$class_line" =~ :[[:space:]]*([^{]+) ]]; then
                local inheritance_list="${BASH_REMATCH[1]}"
                inheritance_list=$(echo "$inheritance_list" | tr ',' '\n' | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
                
                # First non-interface item is usually superclass
                superclass=$(echo "$inheritance_list" | head -1)
                interfaces=$(echo "$inheritance_list" | tail -n +2 | tr '\n' ', ' | sed 's/,$//')
            fi
            
            # Count members within class
            local class_line_num=$(grep -n "(class|object|interface).*$class_name" "$file" | head -1 | cut -d: -f1)
            local end_line_num=$(awk -v start=$class_line_num 'NR > start && /^[[:space:]]*}[[:space:]]*$/ {print NR; exit}' "$file")
            
            local method_count=0
            local property_count=0
            local constructor_count=0
            
            if [[ -n "$class_line_num" && -n "$end_line_num" ]]; then
                method_count=$(sed -n "${class_line_num},${end_line_num}p" "$file" | grep -c "fun " 2>/dev/null || echo "0")
                property_count=$(sed -n "${class_line_num},${end_line_num}p" "$file" | grep -c "val \|var " 2>/dev/null || echo "0")
                constructor_count=$(sed -n "${class_line_num},${end_line_num}p" "$file" | grep -c "constructor\|init" 2>/dev/null || echo "0")
            fi
            
            # Build content
            local content="Class: $class_name | Context: $context | Parent: $parent_name | Type: $class_type"
            content="$content | Visibility: $visibility | Methods: $method_count | Properties: $property_count"
            [[ "$is_data" == "true" ]] && content="$content | Data: true"
            [[ "$is_sealed" == "true" ]] && content="$content | Sealed: true"
            [[ "$is_abstract" == "true" ]] && content="$content | Abstract: true"
            [[ "$is_open" == "true" ]] && content="$content | Open: true"
            [[ -n "$superclass" ]] && content="$content | Extends: $superclass"
            [[ -n "$interfaces" ]] && content="$content | Implements: $interfaces"
            
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
                --arg constructor_count "$constructor_count" \
                --arg is_data "$is_data" \
                --arg is_sealed "$is_sealed" \
                --arg is_abstract "$is_abstract" \
                --arg is_final "$is_final" \
                --arg is_open "$is_open" \
                --arg is_inner "$is_inner" \
                --arg is_enum "$is_enum" \
                --arg is_companion "$is_companion" \
                --arg superclass "$superclass" \
                --arg interfaces "$interfaces" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "kotlin",
                        class_name: $class_name,
                        class_type: $class_type,
                        visibility: $visibility,
                        method_count: ($method_count | tonumber),
                        property_count: ($property_count | tonumber),
                        constructor_count: ($constructor_count | tonumber),
                        is_data: ($is_data == "true"),
                        is_sealed: ($is_sealed == "true"),
                        is_abstract: ($is_abstract == "true"),
                        is_final: ($is_final == "true"),
                        is_open: ($is_open == "true"),
                        is_inner: ($is_inner == "true"),
                        is_enum: ($is_enum == "true"),
                        is_companion: ($is_companion == "true"),
                        superclass: $superclass,
                        interfaces: $interfaces,
                        content_type: "code_class",
                        extraction_method: "kotlin_parser"
                    }
                }' | jq -c
        done <<< "$classes"
    fi
}

#######################################
# Extract build.gradle information
# 
# Gets Kotlin/Android dependencies and project metadata
#
# Arguments:
#   $1 - Path to build.gradle or build.gradle.kts
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with build configuration
#######################################
extractor::lib::kotlin::extract_gradle() {
    local file="$1"
    local context="${2:-kotlin}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Determine if it's Kotlin DSL or Groovy
    local is_kotlin_dsl="false"
    if [[ "$file" == *.kts ]]; then
        is_kotlin_dsl="true"
    fi
    
    # Extract Kotlin version
    local kotlin_version=""
    kotlin_version=$(grep -E "(kotlin_version|kotlinVersion)" "$file" | head -1 | sed 's/.*[=:].*["'\'']\([0-9.]*\)["'\''].*/\1/')
    
    # Count dependencies
    local dependency_count=$(grep -c "implementation\|api\|compile" "$file" 2>/dev/null || echo "0")
    
    # Check for Android
    local is_android="false"
    if grep -q "com.android" "$file" 2>/dev/null; then
        is_android="true"
    fi
    
    # Check for multiplatform
    local is_multiplatform="false"
    if grep -q "kotlin-multiplatform\|multiplatform" "$file" 2>/dev/null; then
        is_multiplatform="true"
    fi
    
    # Extract compile SDK version for Android
    local compile_sdk=""
    if [[ "$is_android" == "true" ]]; then
        compile_sdk=$(grep "compileSdk\|compileSdkVersion" "$file" | head -1 | sed 's/.*[=:].*\([0-9]*\).*/\1/')
    fi
    
    # Check for popular libraries
    local libraries=()
    if grep -q "coroutines" "$file" 2>/dev/null; then
        libraries+=("coroutines")
    fi
    if grep -q "ktor" "$file" 2>/dev/null; then
        libraries+=("ktor")
    fi
    if grep -q "exposed" "$file" 2>/dev/null; then
        libraries+=("exposed")
    fi
    if grep -q "kotlinx.serialization" "$file" 2>/dev/null; then
        libraries+=("serialization")
    fi
    if grep -q "compose" "$file" 2>/dev/null; then
        libraries+=("compose")
    fi
    
    # Extract plugins
    local plugins=$(grep -E "id\s+['\"]" "$file" | sed 's/.*id[[:space:]]*['\''"]//; s/['\''"].*//' | head -5 | tr '\n' ', ' | sed 's/,$//')
    
    # Build content
    local content="Gradle: $(basename "$(dirname "$file")") | Context: $context | Parent: $parent_name"
    [[ -n "$kotlin_version" ]] && content="$content | Kotlin: $kotlin_version"
    content="$content | Dependencies: $dependency_count"
    [[ "$is_android" == "true" ]] && content="$content | Android: $compile_sdk"
    [[ "$is_multiplatform" == "true" ]] && content="$content | Multiplatform: true"
    [[ ${#libraries[@]} -gt 0 ]] && content="$content | Libraries: $(IFS=,; echo "${libraries[*]}")"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg kotlin_version "$kotlin_version" \
        --arg dependency_count "$dependency_count" \
        --arg is_kotlin_dsl "$is_kotlin_dsl" \
        --arg is_android "$is_android" \
        --arg is_multiplatform "$is_multiplatform" \
        --arg compile_sdk "$compile_sdk" \
        --arg plugins "$plugins" \
        --argjson libraries "$(printf '%s\n' "${libraries[@]}" 2>/dev/null | jq -R . | jq -s . || echo '[]')" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "kotlin",
                kotlin_version: $kotlin_version,
                dependency_count: ($dependency_count | tonumber),
                is_kotlin_dsl: ($is_kotlin_dsl == "true"),
                is_android: ($is_android == "true"),
                is_multiplatform: ($is_multiplatform == "true"),
                compile_sdk: $compile_sdk,
                plugins: $plugins,
                libraries: $libraries,
                content_type: "package_config",
                extraction_method: "gradle_parser"
            }
        }' | jq -c
}

#######################################
# Extract all information from Kotlin files
# 
# Main entry point that extracts classes, functions, and build info
#
# Arguments:
#   $1 - Path to file or directory
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with all Kotlin information
#######################################
extractor::lib::kotlin::extract_all() {
    local path="$1"
    local context="${2:-kotlin}"
    local parent_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        case "$path" in
            *.kt|*.kts)
                extractor::lib::kotlin::extract_functions "$path" "$context" "$parent_name" 2>/dev/null || true
                extractor::lib::kotlin::extract_classes "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
            build.gradle|build.gradle.kts)
                extractor::lib::kotlin::extract_gradle "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
        esac
    elif [[ -d "$path" ]]; then
        # Directory - find relevant files
        while IFS= read -r file; do
            extractor::lib::kotlin::extract_all "$file" "$context" "$parent_name"
        done < <(find "$path" -type f \( -name "*.kt" -o -name "*.kts" -o -name "build.gradle*" \) 2>/dev/null)
    fi
}

# Export functions
export -f extractor::lib::kotlin::extract_functions
export -f extractor::lib::kotlin::extract_classes
export -f extractor::lib::kotlin::extract_gradle
export -f extractor::lib::kotlin::extract_all