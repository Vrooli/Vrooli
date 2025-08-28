#!/usr/bin/env bash
# Ruby Code Extractor Library
# Extracts classes, methods, modules, and metadata from Ruby files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract methods from Ruby file
# 
# Finds method definitions and their visibility
#
# Arguments:
#   $1 - Path to Ruby file
#   $2 - Context (component type like 'api', 'cli', etc.)
#   $3 - Parent name (scenario name)
# Returns: JSON lines with method information
#######################################
extractor::lib::ruby::extract_methods() {
    local file="$1"
    local context="${2:-ruby}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    log::debug "Extracting Ruby methods from: $file" >&2
    
    # Find method definitions
    local methods=$(grep -E "^[[:space:]]*def[[:space:]]+([a-zA-Z_][a-zA-Z0-9_?!]*)" "$file" 2>/dev/null | \
        sed -E 's/^[[:space:]]*def[[:space:]]+([a-zA-Z_][a-zA-Z0-9_?!]*).*/\1/' || echo "")
    
    if [[ -n "$methods" ]]; then
        while IFS= read -r method_name; do
            [[ -z "$method_name" ]] && continue
            
            # Determine method type and visibility
            local method_type="method"
            local visibility="public"  # Ruby default
            local is_class_method="false"
            
            # Check if it's a class method (def self.method_name)
            if grep -q "def self\.$method_name\|def [A-Z].*\.$method_name" "$file" 2>/dev/null; then
                is_class_method="true"
                method_type="class_method"
            fi
            
            # Check for special method types
            if [[ "$method_name" == "initialize" ]]; then
                method_type="constructor"
            elif [[ "$method_name" =~ ^[a-z_]+[?]$ ]]; then
                method_type="predicate"
            elif [[ "$method_name" =~ ^[a-z_]+[!]$ ]]; then
                method_type="mutator"
            elif [[ "$method_name" =~ ^[a-z_]+[=]$ ]]; then
                method_type="setter"
            fi
            
            # Check visibility modifiers (look backwards from method definition)
            local line_num=$(grep -n "def.*$method_name" "$file" | head -1 | cut -d: -f1)
            if [[ -n "$line_num" ]]; then
                # Look backwards for visibility modifiers
                local check_line=$((line_num - 1))
                while [[ $check_line -gt 0 ]]; do
                    local line_content=$(sed -n "${check_line}p" "$file")
                    if [[ "$line_content" =~ ^[[:space:]]*private[[:space:]]*$ ]]; then
                        visibility="private"
                        break
                    elif [[ "$line_content" =~ ^[[:space:]]*protected[[:space:]]*$ ]]; then
                        visibility="protected"
                        break
                    elif [[ "$line_content" =~ ^[[:space:]]*(def|class|module) ]]; then
                        break
                    fi
                    ((check_line--))
                done
            fi
            
            # Look for comments above method
            local description=""
            if [[ -n "$line_num" ]]; then
                local desc_line=$((line_num - 1))
                while [[ $desc_line -gt 0 ]]; do
                    local line_content=$(sed -n "${desc_line}p" "$file")
                    if [[ "$line_content" =~ ^[[:space:]]*#[[:space:]]*(.+)$ ]]; then
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
            
            # Get method signature
            local signature=$(grep "def.*$method_name" "$file" | head -1 | sed 's/^[[:space:]]*//')
            
            # Build content
            local content="Method: $method_name | Context: $context | Parent: $parent_name"
            content="$content | Visibility: $visibility | Type: $method_type"
            [[ "$is_class_method" == "true" ]] && content="$content | Class method: true"
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
                --arg signature "$signature" \
                --arg is_class_method "$is_class_method" \
                --arg description "$description" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "ruby",
                        method_name: $method_name,
                        method_type: $method_type,
                        visibility: $visibility,
                        signature: $signature,
                        is_class_method: ($is_class_method == "true"),
                        description: $description,
                        content_type: "code_function",
                        extraction_method: "ruby_parser"
                    }
                }' | jq -c
        done <<< "$methods"
    fi
}

#######################################
# Extract classes and modules from Ruby file
# 
# Finds class and module definitions
#
# Arguments:
#   $1 - Path to Ruby file
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with class/module information
#######################################
extractor::lib::ruby::extract_classes() {
    local file="$1"
    local context="${2:-ruby}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find class definitions
    local classes=$(grep -E "^[[:space:]]*class[[:space:]]+([A-Za-z_][A-Za-z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/^[[:space:]]*class[[:space:]]+([A-Za-z_][A-Za-z0-9_]*).*/\1/' || echo "")
    
    # Find module definitions
    local modules=$(grep -E "^[[:space:]]*module[[:space:]]+([A-Za-z_][A-Za-z0-9_]*)" "$file" 2>/dev/null | \
        sed -E 's/^[[:space:]]*module[[:space:]]+([A-Za-z_][A-Za-z0-9_]*).*/\1/' || echo "")
    
    # Process classes
    if [[ -n "$classes" ]]; then
        while IFS= read -r class_name; do
            [[ -z "$class_name" ]] && continue
            
            # Count methods in class
            local method_count=0
            local attr_count=0
            local in_class=false
            
            # Simple approach - count methods between class and end
            local class_line_num=$(grep -n "^[[:space:]]*class[[:space:]]+$class_name" "$file" | head -1 | cut -d: -f1)
            local end_line_num=$(awk -v start=$class_line_num 'NR > start && /^[[:space:]]*end[[:space:]]*$/ {print NR; exit}' "$file")
            
            if [[ -n "$class_line_num" && -n "$end_line_num" ]]; then
                method_count=$(sed -n "${class_line_num},${end_line_num}p" "$file" | grep -c "^[[:space:]]*def " 2>/dev/null || echo "0")
                attr_count=$(sed -n "${class_line_num},${end_line_num}p" "$file" | grep -c "attr_\(accessor\|reader\|writer\)" 2>/dev/null || echo "0")
            fi
            
            # Extract inheritance information
            local inheritance=""
            local class_line=$(grep "^[[:space:]]*class[[:space:]]\+$class_name" "$file" | head -1)
            if [[ "$class_line" =~ \<[[:space:]]*([A-Za-z_:][A-Za-z0-9_:]*) ]]; then
                inheritance="${BASH_REMATCH[1]}"
            fi
            
            # Check for included modules
            local includes=""
            if [[ -n "$class_line_num" && -n "$end_line_num" ]]; then
                includes=$(sed -n "${class_line_num},${end_line_num}p" "$file" | \
                    grep -E "^[[:space:]]*(include|extend)" | \
                    sed -E 's/^[[:space:]]*(include|extend)[[:space:]]+//' | \
                    tr '\n' ', ' | sed 's/,$//')
            fi
            
            # Build content
            local content="Class: $class_name | Context: $context | Parent: $parent_name"
            content="$content | Methods: $method_count | Attributes: $attr_count"
            [[ -n "$inheritance" ]] && content="$content | Inherits: $inheritance"
            [[ -n "$includes" ]] && content="$content | Includes: $includes"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg class_name "$class_name" \
                --arg class_type "class" \
                --arg method_count "$method_count" \
                --arg attr_count "$attr_count" \
                --arg inheritance "$inheritance" \
                --arg includes "$includes" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "ruby",
                        class_name: $class_name,
                        class_type: $class_type,
                        method_count: ($method_count | tonumber),
                        attribute_count: ($attr_count | tonumber),
                        inheritance: $inheritance,
                        includes: $includes,
                        content_type: "code_class",
                        extraction_method: "ruby_parser"
                    }
                }' | jq -c
        done <<< "$classes"
    fi
    
    # Process modules
    if [[ -n "$modules" ]]; then
        while IFS= read -r module_name; do
            [[ -z "$module_name" ]] && continue
            
            # Count methods in module
            local method_count=0
            local module_line_num=$(grep -n "^[[:space:]]*module[[:space:]]+$module_name" "$file" | head -1 | cut -d: -f1)
            local end_line_num=$(awk -v start=$module_line_num 'NR > start && /^[[:space:]]*end[[:space:]]*$/ {print NR; exit}' "$file")
            
            if [[ -n "$module_line_num" && -n "$end_line_num" ]]; then
                method_count=$(sed -n "${module_line_num},${end_line_num}p" "$file" | grep -c "^[[:space:]]*def " 2>/dev/null || echo "0")
            fi
            
            # Build content
            local content="Module: $module_name | Context: $context | Parent: $parent_name"
            content="$content | Methods: $method_count"
            
            # Output as JSON line
            jq -n \
                --arg content "$content" \
                --arg parent "$parent_name" \
                --arg source_file "$file" \
                --arg context "$context" \
                --arg class_name "$module_name" \
                --arg class_type "module" \
                --arg method_count "$method_count" \
                '{
                    content: $content,
                    metadata: {
                        parent: $parent,
                        source_file: $source_file,
                        component_type: $context,
                        language: "ruby",
                        class_name: $class_name,
                        class_type: $class_type,
                        method_count: ($method_count | tonumber),
                        content_type: "code_class",
                        extraction_method: "ruby_parser"
                    }
                }' | jq -c
        done <<< "$modules"
    fi
}

#######################################
# Extract Gemfile information
# 
# Gets Ruby gem dependencies
#
# Arguments:
#   $1 - Path to Gemfile
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON line with Gemfile information
#######################################
extractor::lib::ruby::extract_gemfile() {
    local file="$1"
    local context="${2:-ruby}"
    local parent_name="${3:-unknown}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract Ruby version
    local ruby_version=$(grep "^ruby " "$file" | sed "s/ruby ['\"]//; s/['\"].*//")
    
    # Count gem dependencies
    local gem_count=$(grep -c "^gem " "$file" 2>/dev/null || echo "0")
    
    # Get source
    local source=$(grep "^source " "$file" | head -1 | sed "s/source ['\"]//; s/['\"].*//")
    
    # Check for Rails
    local has_rails="false"
    if grep -q "gem ['\"]rails['\"]" "$file" 2>/dev/null; then
        has_rails="true"
    fi
    
    # Get Rails version if present
    local rails_version=""
    if [[ "$has_rails" == "true" ]]; then
        rails_version=$(grep "gem ['\"]rails['\"]" "$file" | sed "s/.*['\"][[:space:]]*,[[:space:]]*['\"]//; s/['\"].*//")
    fi
    
    # Extract top gems
    local gems=$(grep "^gem " "$file" | head -10 | sed "s/gem ['\"]//; s/['\"].*//")
    local gems_json="[]"
    if [[ -n "$gems" ]]; then
        gems_json=$(echo "$gems" | jq -R . | jq -s .)
    fi
    
    # Build content
    local content="Gemfile: $(basename "$(dirname "$file")") | Context: $context | Parent: $parent_name"
    [[ -n "$ruby_version" ]] && content="$content | Ruby: $ruby_version"
    content="$content | Gems: $gem_count"
    [[ "$has_rails" == "true" ]] && content="$content | Rails: $rails_version"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg parent "$parent_name" \
        --arg source_file "$file" \
        --arg context "$context" \
        --arg ruby_version "$ruby_version" \
        --arg gem_count "$gem_count" \
        --arg source "$source" \
        --arg has_rails "$has_rails" \
        --arg rails_version "$rails_version" \
        --argjson gems "$gems_json" \
        '{
            content: $content,
            metadata: {
                parent: $parent,
                source_file: $source_file,
                component_type: $context,
                language: "ruby",
                ruby_version: $ruby_version,
                gem_count: ($gem_count | tonumber),
                gem_source: $source,
                has_rails: ($has_rails == "true"),
                rails_version: $rails_version,
                gems: $gems,
                content_type: "package_config",
                extraction_method: "gemfile_parser"
            }
        }' | jq -c
}

#######################################
# Extract all information from Ruby files
# 
# Main entry point that extracts classes, methods, and Gemfile info
#
# Arguments:
#   $1 - Path to file or directory
#   $2 - Context (component type)
#   $3 - Parent name
# Returns: JSON lines with all Ruby information
#######################################
extractor::lib::ruby::extract_all() {
    local path="$1"
    local context="${2:-ruby}"
    local parent_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        case "$path" in
            *.rb)
                extractor::lib::ruby::extract_methods "$path" "$context" "$parent_name" 2>/dev/null || true
                extractor::lib::ruby::extract_classes "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
            Gemfile)
                extractor::lib::ruby::extract_gemfile "$path" "$context" "$parent_name" 2>/dev/null || true
                ;;
        esac
    elif [[ -d "$path" ]]; then
        # Directory - find relevant files
        while IFS= read -r file; do
            extractor::lib::ruby::extract_all "$file" "$context" "$parent_name"
        done < <(find "$path" -type f \( -name "*.rb" -o -name "Gemfile" \) 2>/dev/null)
    fi
}

# Export functions
export -f extractor::lib::ruby::extract_methods
export -f extractor::lib::ruby::extract_classes
export -f extractor::lib::ruby::extract_gemfile
export -f extractor::lib::ruby::extract_all