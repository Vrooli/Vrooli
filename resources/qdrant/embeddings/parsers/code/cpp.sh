#!/usr/bin/env bash
# C++ Code Extractor Library
# Extracts functions, classes, namespaces, and metadata from C++ files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract functions from C++ file
# 
# Finds function definitions and declarations
#
# Arguments:
#   $1 - Path to C++ file
#   $2 - Context (component type like 'api', 'cli', etc.)
#   $3 - Parent name (scenario name)
# Returns: JSON lines with function information
#######################################
extractor::lib::cpp::extract_functions() {
    local file="$1"
    local context="${2:-cpp}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    log::debug "Extracting C++ functions from: $file" >&2
    
    # Find function definitions (avoiding class member declarations in headers)
    local functions=$(grep -E "^\s*(static|inline|virtual|explicit)?\s*(template\s*<[^>]*>)?\s*[a-zA-Z_:][a-zA-Z0-9_:<>]*\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(" "$file" 2>/dev/null | \
        grep -v "^\s*#\|^\s*//" | \
        sed -E 's/.*\s([a-zA-Z_][a-zA-Z0-9_]*)\s*\(.*/\1/' | \
        grep -v "^if$\|^for$\|^while$\|^switch$\|^catch$" || echo "")
    
    if [[ -n "$functions" ]]; then
        while IFS= read -r function_name; do
            [[ -z "$function_name" ]] && continue
            
            # Get full function declaration/definition
            local func_line=$(grep -E "\s$function_name\s*\(" "$file" | head -1)
            
            # Skip if it looks like a macro or control structure
            if echo "$func_line" | grep -qE "^\s*#|^\s*(if|for|while|switch|catch)\s*\("; then
                continue
            fi
            
            # Determine if it's in a header or implementation file
            local file_type="implementation"
            if [[ "$file" =~ \.(h|hpp|hxx)$ ]]; then
                file_type="header"
            fi
            
            # Check modifiers
            local is_static="false"
            local is_inline="false"
            local is_virtual="false"
            local is_explicit="false"
            local is_template="false"
            local is_const="false"
            
            if echo "$func_line" | grep -q "static"; then
                is_static="true"
            fi
            if echo "$func_line" | grep -q "inline"; then
                is_inline="true"
            fi
            if echo "$func_line" | grep -q "virtual"; then
                is_virtual="true"
            fi
            if echo "$func_line" | grep -q "explicit"; then
                is_explicit="true"
            fi
            if echo "$func_line" | grep -q "template"; then
                is_template="true"
            fi
            if echo "$func_line" | grep -q "const\s*{"; then
                is_const="true"
            fi
            
            # Determine function type
            local func_type="function"
            if [[ "$function_name" =~ ^~.*$ ]]; then
                func_type="destructor"
            elif echo "$func_line" | grep -qE "operator\s*[+\-*/<>=!&|%^~\[\]()]"; then
                func_type="operator"
            elif [[ "$is_explicit" == "true" ]]; then
                func_type="constructor"
            elif [[ "$is_template" == "true" ]]; then
                func_type="template_function"
            fi
            
            # Extract return type (rough heuristic)
            local return_type=""
            if [[ "$func_line" =~ ^[[:space:]]*(static|inline|virtual|explicit)?\s*(template\s*<[^>]*>)?\s*([a-zA-Z_:][a-zA-Z0-9_:<>*&]*)\s+$function_name ]]; then
                return_type="${BASH_REMATCH[3]}"
                # Clean up common modifiers that got captured
                return_type=$(echo "$return_type" | sed 's/static\|inline\|virtual\|explicit//g' | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
            fi
            
            # Check if it's a class method (rough heuristic based on indentation or scope)
            local is_method="false"
            local class_name=""
            
            # Look backwards for class definition
            local line_num=$(grep -n "$function_name\s*(" "$file" | head -1 | cut -d: -f1)
            if [[ -n "$line_num" ]]; then
                local check_line=$((line_num - 1))
                while [[ $check_line -gt 0 ]]; do
                    local line_content=$(sed -n "${check_line}p" "$file")
                    if [[ "$line_content" =~ ^[[:space:]]*(class|struct)[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*) ]]; then
                        is_method="true"
                        class_name="${BASH_REMATCH[2]}"
                        break
                    elif [[ "$line_content" =~ ^[[:space:]]*\}[[:space:]]*$ ]] || [[ "$line_content" =~ ^[[:space:]]*\}[[:space:]]*\;[[:space:]]*$ ]]; then
                        break
                    fi
                    ((check_line--))
                done
            fi
            
            # Extract namespace (look for namespace declarations)
            local namespace=""
            namespace=$(grep "^namespace " "$file" | head -1 | sed 's/namespace //; s/{.*//' | tr -d ' ')
            
            # Look for C++ style comments
            local description=""
            if [[ -n "$line_num" ]]; then
                local check_line=$((line_num - 1))
                while [[ $check_line -gt 0 ]]; do
                    local line_content=$(sed -n "${check_line}p" "$file")
                    if [[ "$line_content" =~ ^[[:space:]]*//[[:space:]]*(.*)$ ]]; then
                        local comment="${BASH_REMATCH[1]}"
                        if [[ -z "$description" ]]; then
                            description="$comment"
                        else
                            description="$comment $description"
                        fi
                        ((check_line--))
                    else
                        break
                    fi
                done
            fi
            
            # Get function signature (clean it up)
            local signature=$(echo "$func_line" | sed 's/^[[:space:]]*//; s/{.*//')
            
            # Build content
            local content="Function: $function_name | Context: $context | Parent: $parent_name | Type: $func_type"
            content="$content | File: $file_type"
            [[ "$is_method" == "true" ]] && content="$content | Class: $class_name"
            [[ "$is_static" == "true" ]] && content="$content | Static: true"
            [[ "$is_virtual" == "true" ]] && content="$content | Virtual: true"
            [[ "$is_template" == "true" ]] && content="$content | Template: true"
            [[ -n "$return_type" ]] && content="$content | Returns: $return_type"
            [[ -n "$namespace" ]] && content="$content | Namespace: $namespace"
            [[ -n "$description" ]] && content="$content | Description: $description"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg function_name "$function_name" \
                --arg function_type "$func_type" \
                --arg file_type "$file_type" \
                --arg signature "$signature" \
                --arg is_static "$is_static" \
                --arg is_inline "$is_inline" \
                --arg is_virtual "$is_virtual" \
                --arg is_explicit "$is_explicit" \
                --arg is_template "$is_template" \
                --arg is_const "$is_const" \
                --arg is_method "$is_method" \
                --arg class_name "$class_name" \
                --arg return_type "$return_type" \
                --arg namespace "$namespace" \
                --arg description "$description" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "cpp",
                        function_name: $function_name,
                        function_type: $function_type,
                        file_type: $file_type,
                        signature: $signature,
                        is_static: ($is_static == "true"),
                        is_inline: ($is_inline == "true"),
                        is_virtual: ($is_virtual == "true"),
                        is_explicit: ($is_explicit == "true"),
                        is_template: ($is_template == "true"),
                        is_const: ($is_const == "true"),
                        is_method: ($is_method == "true"),
                        class_name: $class_name,
                        return_type: $return_type,
                        namespace: $namespace,
                        description: $description,
                        content_type: "code_function",
                        extraction_method: "cpp_parser"
                    }
                }' | jq -c
        done <<< "$functions"
    fi
}

#######################################
# Extract classes and structs from C++ file
# 
# Finds class and struct definitions with inheritance
#
# Arguments:
#   $1 - Path to C++ file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with class information
#######################################
extractor::lib::cpp::extract_classes() {
    local file="$1"
    local context="${2:-cpp}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find class and struct definitions
    local classes=$(grep -E "^\s*(template\s*<[^>]*>)?\s*(class|struct)\s+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/.*(class|struct)\s+([a-zA-Z_][a-zA-Z0-9_]*).*/\2/' || echo "")
    
    if [[ -n "$classes" ]]; then
        while IFS= read -r class_name; do
            [[ -z "$class_name" ]] && continue
            
            # Get full class declaration
            local class_line=$(grep -E "(class|struct)\s+$class_name" "$file" | head -1)
            
            # Determine if it's a header or implementation file
            local file_type="implementation"
            if [[ "$file" =~ \.(h|hpp|hxx)$ ]]; then
                file_type="header"
            fi
            
            # Determine class type
            local class_type="class"
            if echo "$class_line" | grep -q "struct"; then
                class_type="struct"
            fi
            
            # Check if it's a template
            local is_template="false"
            if echo "$class_line" | grep -q "template"; then
                is_template="true"
                class_type="template_$class_type"
            fi
            
            # Extract inheritance
            local inheritance=""
            if [[ "$class_line" =~ :[[:space:]]*([^{]+) ]]; then
                inheritance="${BASH_REMATCH[1]}"
                inheritance=$(echo "$inheritance" | sed 's/{.*//' | tr ',' ' ' | sed 's/public\|private\|protected//g' | sed 's/^[[:space:]]*//; s/[[:space:]]*$//')
            fi
            
            # Count members within class
            local class_line_num=$(grep -n "(class|struct).*$class_name" "$file" | head -1 | cut -d: -f1)
            local end_line_num=$(awk -v start=$class_line_num 'NR > start && /^[[:space:]]*};/ {print NR; exit}' "$file")
            
            local method_count=0
            local member_count=0
            local constructor_count=0
            
            if [[ -n "$class_line_num" && -n "$end_line_num" ]]; then
                # Count methods (functions within class)
                method_count=$(sed -n "${class_line_num},${end_line_num}p" "$file" | grep -c "\s\+[a-zA-Z_][a-zA-Z0-9_]*\s*(" 2>/dev/null || echo "0")
                
                # Count member variables (lines ending with ;)
                member_count=$(sed -n "${class_line_num},${end_line_num}p" "$file" | grep -c "\s\+[a-zA-Z_][a-zA-Z0-9_<>*&:]*\s\+[a-zA-Z_][a-zA-Z0-9_]*\s*;" 2>/dev/null || echo "0")
                
                # Count constructors (including explicit)
                constructor_count=$(sed -n "${class_line_num},${end_line_num}p" "$file" | grep -c "$class_name\s*(" 2>/dev/null || echo "0")
            fi
            
            # Look for access specifiers
            local has_public="false"
            local has_private="false"
            local has_protected="false"
            
            if [[ -n "$class_line_num" && -n "$end_line_num" ]]; then
                if sed -n "${class_line_num},${end_line_num}p" "$file" | grep -q "public:"; then
                    has_public="true"
                fi
                if sed -n "${class_line_num},${end_line_num}p" "$file" | grep -q "private:"; then
                    has_private="true"
                fi
                if sed -n "${class_line_num},${end_line_num}p" "$file" | grep -q "protected:"; then
                    has_protected="true"
                fi
            fi
            
            # Extract namespace
            local namespace=""
            namespace=$(grep "^namespace " "$file" | head -1 | sed 's/namespace //; s/{.*//' | tr -d ' ')
            
            # Build content
            local content="Class: $class_name | Context: $context | Parent: $parent_name | Type: $class_type"
            content="$content | File: $file_type | Methods: $method_count | Members: $member_count"
            [[ "$is_template" == "true" ]] && content="$content | Template: true"
            [[ -n "$inheritance" ]] && content="$content | Inherits: $inheritance"
            [[ -n "$namespace" ]] && content="$content | Namespace: $namespace"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg class_name "$class_name" \
                --arg class_type "$class_type" \
                --arg file_type "$file_type" \
                --arg method_count "$method_count" \
                --arg member_count "$member_count" \
                --arg constructor_count "$constructor_count" \
                --arg is_template "$is_template" \
                --arg inheritance "$inheritance" \
                --arg has_public "$has_public" \
                --arg has_private "$has_private" \
                --arg has_protected "$has_protected" \
                --arg namespace "$namespace" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "cpp",
                        class_name: $class_name,
                        class_type: $class_type,
                        file_type: $file_type,
                        method_count: ($method_count | tonumber),
                        member_count: ($member_count | tonumber),
                        constructor_count: ($constructor_count | tonumber),
                        is_template: ($is_template == "true"),
                        inheritance: $inheritance,
                        has_public: ($has_public == "true"),
                        has_private: ($has_private == "true"),
                        has_protected: ($has_protected == "true"),
                        namespace: $namespace,
                        content_type: "code_class",
                        extraction_method: "cpp_parser"
                    }
                }' | jq -c
        done <<< "$classes"
    fi
}

#######################################
# Extract CMakeLists.txt information
# 
# Gets C++ build configuration and dependencies
#
# Arguments:
#   $1 - Path to CMakeLists.txt
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with CMake configuration
#######################################
extractor::lib::cpp::extract_cmake() {
    local file="$1"
    local context="${2:-cpp}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract project name
    local project_name=$(grep "project(" "$file" | head -1 | sed 's/project(//; s/).*//; s/[[:space:]]//g')
    
    # Extract CMake minimum version
    local cmake_version=$(grep "cmake_minimum_required" "$file" | head -1 | sed 's/.*VERSION[[:space:]]*//; s/).*//')
    
    # Extract C++ standard
    local cpp_standard=$(grep "CMAKE_CXX_STANDARD\|set_property.*CXX_STANDARD" "$file" | head -1 | sed 's/.*[[:space:]]\([0-9][0-9]\).*/\1/')
    
    # Count targets
    local target_count=$(grep -c "add_executable\|add_library" "$file" 2>/dev/null || echo "0")
    
    # Count find_package calls
    local package_count=$(grep -c "find_package" "$file" 2>/dev/null || echo "0")
    
    # Extract common libraries
    local libraries=()
    if grep -q "find_package.*Boost" "$file" 2>/dev/null; then
        libraries+=("boost")
    fi
    if grep -q "find_package.*Qt" "$file" 2>/dev/null; then
        libraries+=("qt")
    fi
    if grep -q "find_package.*OpenCV" "$file" 2>/dev/null; then
        libraries+=("opencv")
    fi
    if grep -q "find_package.*Eigen" "$file" 2>/dev/null; then
        libraries+=("eigen")
    fi
    if grep -q "pthread" "$file" 2>/dev/null; then
        libraries+=("pthread")
    fi
    
    # Check for testing
    local has_testing="false"
    if grep -q "enable_testing\|add_test\|gtest" "$file" 2>/dev/null; then
        has_testing="true"
    fi
    
    # Extract compiler flags
    local has_custom_flags="false"
    if grep -q "CMAKE_CXX_FLAGS\|target_compile_options" "$file" 2>/dev/null; then
        has_custom_flags="true"
    fi
    
    # Build content
    local content="CMake: $(basename "$(dirname "$file")") | Context: $context | Parent: $parent_name"
    [[ -n "$project_name" ]] && content="$content | Project: $project_name"
    [[ -n "$cmake_version" ]] && content="$content | CMake: $cmake_version"
    [[ -n "$cpp_standard" ]] && content="$content | C++: $cpp_standard"
    content="$content | Targets: $target_count | Packages: $package_count"
    [[ ${#libraries[@]} -gt 0 ]] && content="$content | Libraries: $(IFS=,; echo "${libraries[*]}")"
    [[ "$has_testing" == "true" ]] && content="$content | Testing: true"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg project_name "$project_name" \
        --arg cmake_version "$cmake_version" \
        --arg cpp_standard "$cpp_standard" \
        --arg target_count "$target_count" \
        --arg package_count "$package_count" \
        --arg has_testing "$has_testing" \
        --arg has_custom_flags "$has_custom_flags" \
        --argjson libraries "$(printf '%s\n' "${libraries[@]}" 2>/dev/null | jq -R . | jq -s . || echo '[]')" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "cpp",
                project_name: $project_name,
                cmake_version: $cmake_version,
                cpp_standard: $cpp_standard,
                target_count: ($target_count | tonumber),
                package_count: ($package_count | tonumber),
                has_testing: ($has_testing == "true"),
                has_custom_flags: ($has_custom_flags == "true"),
                libraries: $libraries,
                content_type: "package_config",
                extraction_method: "cmake_parser"
            }
        }' | jq -c
}

#######################################
# Extract all information from C++ files
# 
# Main entry point that extracts classes, functions, and build info
#
# Arguments:
#   $1 - Path to file or directory
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with all C++ information
#######################################
extractor::lib::cpp::extract_all() {
    local path="$1"
    local context="${2:-cpp}"
    local parent_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        case "$path" in
            *.cpp|*.cc|*.cxx|*.c|*.h|*.hpp|*.hxx)
                extractor::lib::cpp::extract_functions "$path" "$context" "$parent_name" 2>/dev/null || true
                extractor::lib::cpp::extract_classes "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
            CMakeLists.txt)
                extractor::lib::cpp::extract_cmake "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
        esac
    elif [[ -d "$path" ]]; then
        # Directory - find relevant files
        while IFS= read -r file; do
            extractor::lib::cpp::extract_all "$file" "$context" "$parent_name"
        done < <(find "$path" -type f \( -name "*.cpp" -o -name "*.cc" -o -name "*.cxx" -o -name "*.c" -o -name "*.h" -o -name "*.hpp" -o -name "*.hxx" -o -name "CMakeLists.txt" \) 2>/dev/null)
    fi
}

# Export functions
export -f extractor::lib::cpp::extract_functions
export -f extractor::lib::cpp::extract_classes
export -f extractor::lib::cpp::extract_cmake
export -f extractor::lib::cpp::extract_all