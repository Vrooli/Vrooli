#!/usr/bin/env bash
# PHP Code Extractor Library
# Extracts functions, classes, methods, and metadata from PHP files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract functions from PHP file
# 
# Finds function definitions with visibility and static modifiers
#
# Arguments:
#   $1 - Path to PHP file
#   $2 - Context (component type like 'api', 'cli', etc.)
#   $3 - Parent name (scenario name)
# Returns: JSON lines with function information
#######################################
extractor::lib::php::extract_functions() {
    local file="$1"
    local context="${2:-php}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    log::debug "Extracting PHP functions from: $file" >&2
    
    # Find function definitions (including methods)
    local functions=$(grep -E "^\s*(public|private|protected|static)?\s*(static)?\s*function\s+([a-zA-Z_][a-zA-Z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/.*function\s+([a-zA-Z_][a-zA-Z0-9_]*).*/\1/' || echo "")
    
    if [[ -n "$functions" ]]; then
        while IFS= read -r function_name; do
            [[ -z "$function_name" ]] && continue
            
            # Get full function declaration
            local func_line=$(grep -E "function\s+$function_name" "$file" | head -1)
            
            # Extract visibility
            local visibility="public"  # PHP default
            if echo "$func_line" | grep -q "private"; then
                visibility="private"
            elif echo "$func_line" | grep -q "protected"; then
                visibility="protected"
            fi
            
            # Check if static
            local is_static="false"
            if echo "$func_line" | grep -q "static"; then
                is_static="true"
            fi
            
            # Determine function type
            local func_type="function"
            if [[ "$function_name" == "__construct" ]]; then
                func_type="constructor"
            elif [[ "$function_name" == "__destruct" ]]; then
                func_type="destructor"
            elif [[ "$function_name" =~ ^__(get|set|call|callStatic|toString|invoke|isset|unset|sleep|wakeup|serialize|unserialize|clone|debugInfo)$ ]]; then
                func_type="magic_method"
            elif echo "$func_line" | grep -q "class.*{" && echo "$func_line" | grep -v "function"; then
                func_type="method"
            fi
            
            # Look for PHPDoc comments
            local line_num=$(grep -n "function.*$function_name" "$file" | head -1 | cut -d: -f1)
            local description=""
            local return_type=""
            local parameters=""
            
            if [[ -n "$line_num" ]]; then
                # Look backwards for PHPDoc
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
                        if [[ "$line_content" =~ \*[[:space:]]*@return[[:space:]]+([^[:space:]]+) ]]; then
                            return_type="${BASH_REMATCH[1]}"
                        elif [[ "$line_content" =~ \*[[:space:]]*@param[[:space:]]+([^[:space:]]+)[[:space:]]+([^[:space:]]+)[[:space:]]*(.*)$ ]]; then
                            local param_info="${BASH_REMATCH[2]} (${BASH_REMATCH[1]})"
                            if [[ -n "${BASH_REMATCH[3]}" ]]; then
                                param_info="$param_info - ${BASH_REMATCH[3]}"
                            fi
                            parameters="$param_info; $parameters"
                        elif [[ "$line_content" =~ \*[[:space:]]*([^@].*)$ ]] && [[ -z "$description" ]]; then
                            description="${BASH_REMATCH[1]}"
                        fi
                    fi
                    ((check_line--))
                done
            fi
            
            # Get function signature
            local signature=$(grep "function.*$function_name" "$file" | head -1 | sed 's/^[[:space:]]*//')
            
            # Extract namespace if present
            local namespace=$(grep "^namespace " "$file" | head -1 | sed 's/namespace //; s/;//' || echo "")
            
            # Build content
            local content="Function: $function_name | Context: $context | Parent: $parent_name"
            content="$content | Visibility: $visibility | Type: $func_type"
            [[ "$is_static" == "true" ]] && content="$content | Static: true"
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
                --arg visibility "$visibility" \
                --arg signature "$signature" \
                --arg is_static "$is_static" \
                --arg return_type "$return_type" \
                --arg parameters "$parameters" \
                --arg namespace "$namespace" \
                --arg description "$description" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "php",
                        function_name: $function_name,
                        function_type: $function_type,
                        visibility: $visibility,
                        signature: $signature,
                        is_static: ($is_static == "true"),
                        return_type: $return_type,
                        parameters: $parameters,
                        namespace: $namespace,
                        description: $description,
                        content_type: "code_function",
                        extraction_method: "php_parser"
                    }
                }' | jq -c
        done <<< "$functions"
    fi
}

#######################################
# Extract classes from PHP file
# 
# Finds class definitions with inheritance, interfaces, and traits
#
# Arguments:
#   $1 - Path to PHP file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with class information
#######################################
extractor::lib::php::extract_classes() {
    local file="$1"
    local context="${2:-php}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find class definitions
    local classes=$(grep -E "^\s*(abstract|final)?\s*(class|interface|trait)\s+([A-Za-z_][A-Za-z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/.*\s+(class|interface|trait)\s+([A-Za-z_][A-Za-z0-9_]*).*/\2/' || echo "")
    
    if [[ -n "$classes" ]]; then
        while IFS= read -r class_name; do
            [[ -z "$class_name" ]] && continue
            
            # Get full class declaration
            local class_line=$(grep -E "(class|interface|trait)\s+$class_name" "$file" | head -1)
            
            # Determine class type
            local class_type="class"
            if echo "$class_line" | grep -q "interface"; then
                class_type="interface"
            elif echo "$class_line" | grep -q "trait"; then
                class_type="trait"
            fi
            
            # Check modifiers
            local is_abstract="false"
            local is_final="false"
            if echo "$class_line" | grep -q "abstract"; then
                is_abstract="true"
            elif echo "$class_line" | grep -q "final"; then
                is_final="true"
            fi
            
            # Extract inheritance
            local extends=""
            if [[ "$class_line" =~ extends[[:space:]]+([A-Za-z_][A-Za-z0-9_]*) ]]; then
                extends="${BASH_REMATCH[1]}"
            fi
            
            # Extract implemented interfaces
            local implements=""
            if [[ "$class_line" =~ implements[[:space:]]+([^{]+) ]]; then
                implements="${BASH_REMATCH[1]}"
                implements=$(echo "$implements" | sed 's/[[:space:]]*{.*//; s/,/ /g')
            fi
            
            # Count methods, properties, and constants
            local class_line_num=$(grep -n "(class|interface|trait).*$class_name" "$file" | head -1 | cut -d: -f1)
            local end_line_num=$(awk -v start=$class_line_num 'NR > start && /^[[:space:]]*}[[:space:]]*$/ {print NR; exit}' "$file")
            
            local method_count=0
            local property_count=0
            local const_count=0
            
            if [[ -n "$class_line_num" && -n "$end_line_num" ]]; then
                method_count=$(sed -n "${class_line_num},${end_line_num}p" "$file" | grep -c "function " 2>/dev/null || echo "0")
                property_count=$(sed -n "${class_line_num},${end_line_num}p" "$file" | grep -c "^\s*\(public\|private\|protected\).*\\$" 2>/dev/null || echo "0")
                const_count=$(sed -n "${class_line_num},${end_line_num}p" "$file" | grep -c "const " 2>/dev/null || echo "0")
            fi
            
            # Extract used traits
            local uses=""
            if [[ -n "$class_line_num" && -n "$end_line_num" ]]; then
                uses=$(sed -n "${class_line_num},${end_line_num}p" "$file" | \
                    grep -E "^\s*use\s+" | \
                    sed -E 's/^\s*use\s+//; s/;//' | \
                    tr '\n' ', ' | sed 's/,$//')
            fi
            
            # Extract namespace
            local namespace=$(grep "^namespace " "$file" | head -1 | sed 's/namespace //; s/;//' || echo "")
            
            # Build content
            local content="Class: $class_name | Context: $context | Parent: $parent_name | Type: $class_type"
            content="$content | Methods: $method_count | Properties: $property_count"
            [[ "$is_abstract" == "true" ]] && content="$content | Abstract: true"
            [[ "$is_final" == "true" ]] && content="$content | Final: true"
            [[ -n "$extends" ]] && content="$content | Extends: $extends"
            [[ -n "$implements" ]] && content="$content | Implements: $implements"
            [[ -n "$uses" ]] && content="$content | Uses: $uses"
            [[ -n "$namespace" ]] && content="$content | Namespace: $namespace"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg class_name "$class_name" \
                --arg class_type "$class_type" \
                --arg method_count "$method_count" \
                --arg property_count "$property_count" \
                --arg const_count "$const_count" \
                --arg is_abstract "$is_abstract" \
                --arg is_final "$is_final" \
                --arg extends "$extends" \
                --arg implements "$implements" \
                --arg uses "$uses" \
                --arg namespace "$namespace" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "php",
                        class_name: $class_name,
                        class_type: $class_type,
                        method_count: ($method_count | tonumber),
                        property_count: ($property_count | tonumber),
                        constant_count: ($const_count | tonumber),
                        is_abstract: ($is_abstract == "true"),
                        is_final: ($is_final == "true"),
                        extends: $extends,
                        implements: $implements,
                        uses_traits: $uses,
                        namespace: $namespace,
                        content_type: "code_class",
                        extraction_method: "php_parser"
                    }
                }' | jq -c
        done <<< "$classes"
    fi
}

#######################################
# Extract composer.json information
# 
# Gets PHP package dependencies and project metadata
#
# Arguments:
#   $1 - Path to composer.json
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with composer information
#######################################
extractor::lib::php::extract_composer() {
    local file="$1"
    local context="${2:-php}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local composer_json=$(cat "$file" 2>/dev/null || echo "{}")
    
    # Extract key information
    local name=$(echo "$composer_json" | jq -r '.name // empty')
    local description=$(echo "$composer_json" | jq -r '.description // empty')
    local php_version=$(echo "$composer_json" | jq -r '.require.php // empty')
    
    # Count dependencies
    local require_count=$(echo "$composer_json" | jq '.require | length' 2>/dev/null || echo "0")
    local require_dev_count=$(echo "$composer_json" | jq '."require-dev" | length' 2>/dev/null || echo "0")
    
    # Check for popular frameworks
    local frameworks=()
    local laravel_version=""
    local symfony_version=""
    
    if echo "$composer_json" | jq -e '.require["laravel/framework"]' >/dev/null 2>&1; then
        frameworks+=("laravel")
        laravel_version=$(echo "$composer_json" | jq -r '.require["laravel/framework"]')
    fi
    
    if echo "$composer_json" | jq -e '.require | keys | map(select(startswith("symfony/"))) | length > 0' >/dev/null 2>&1; then
        frameworks+=("symfony")
        symfony_version=$(echo "$composer_json" | jq -r '.require | to_entries | map(select(.key | startswith("symfony/"))) | .[0].value // empty')
    fi
    
    # Check for testing frameworks
    local test_frameworks=()
    if echo "$composer_json" | jq -e '."require-dev"["phpunit/phpunit"]' >/dev/null 2>&1; then
        test_frameworks+=("phpunit")
    fi
    if echo "$composer_json" | jq -e '."require-dev"["pestphp/pest"]' >/dev/null 2>&1; then
        test_frameworks+=("pest")
    fi
    
    # Get autoload information
    local has_autoload="false"
    local autoload_psr4_count=0
    if echo "$composer_json" | jq -e '.autoload."psr-4"' >/dev/null 2>&1; then
        has_autoload="true"
        autoload_psr4_count=$(echo "$composer_json" | jq '.autoload."psr-4" | length')
    fi
    
    # Build content
    local content="Composer: $(basename "$(dirname "$file")") | Context: $context | Parent: $parent_name"
    [[ -n "$name" ]] && content="$content | Name: $name"
    [[ -n "$php_version" ]] && content="$content | PHP: $php_version"
    content="$content | Dependencies: $require_count"
    [[ ${#frameworks[@]} -gt 0 ]] && content="$content | Frameworks: $(IFS=,; echo "${frameworks[*]}")"
    [[ ${#test_frameworks[@]} -gt 0 ]] && content="$content | Testing: $(IFS=,; echo "${test_frameworks[*]}")"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg name "$name" \
        --arg description "$description" \
        --arg php_version "$php_version" \
        --arg require_count "$require_count" \
        --arg require_dev_count "$require_dev_count" \
        --arg laravel_version "$laravel_version" \
        --arg symfony_version "$symfony_version" \
        --arg has_autoload "$has_autoload" \
        --arg autoload_psr4_count "$autoload_psr4_count" \
        --argjson frameworks "$(printf '%s\n' "${frameworks[@]}" 2>/dev/null | jq -R . | jq -s . || echo '[]')" \
        --argjson test_frameworks "$(printf '%s\n' "${test_frameworks[@]}" 2>/dev/null | jq -R . | jq -s . || echo '[]')" \
        --argjson full_config "$composer_json" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "php",
                package_name: $name,
                description: $description,
                php_version: $php_version,
                require_count: ($require_count | tonumber),
                require_dev_count: ($require_dev_count | tonumber),
                laravel_version: $laravel_version,
                symfony_version: $symfony_version,
                frameworks: $frameworks,
                test_frameworks: $test_frameworks,
                has_autoload: ($has_autoload == "true"),
                autoload_psr4_count: ($autoload_psr4_count | tonumber),
                full_configuration: $full_config,
                content_type: "package_config",
                extraction_method: "composer_parser"
            }
        }' | jq -c
}

#######################################
# Extract all information from PHP files
# 
# Main entry point that extracts classes, functions, and composer info
#
# Arguments:
#   $1 - Path to file or directory
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with all PHP information
#######################################
extractor::lib::php::extract_all() {
    local path="$1"
    local context="${2:-php}"
    local parent_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        case "$path" in
            *.php)
                extractor::lib::php::extract_functions "$path" "$context" "$parent_name" 2>/dev/null || true
                extractor::lib::php::extract_classes "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
            composer.json)
                extractor::lib::php::extract_composer "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
        esac
    elif [[ -d "$path" ]]; then
        # Directory - find relevant files
        while IFS= read -r file; do
            extractor::lib::php::extract_all "$file" "$context" "$parent_name"
        done < <(find "$path" -type f \( -name "*.php" -o -name "composer.json" \) 2>/dev/null)
    fi
}

# Export functions
export -f extractor::lib::php::extract_functions
export -f extractor::lib::php::extract_classes
export -f extractor::lib::php::extract_composer
export -f extractor::lib::php::extract_all