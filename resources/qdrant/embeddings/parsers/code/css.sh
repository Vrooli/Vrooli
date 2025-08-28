#!/usr/bin/env bash
# CSS Language Parser for Qdrant Embeddings
# Extracts selectors, rules, variables, animations, and media queries from CSS/SCSS/LESS files
#
# Handles:
# - CSS class and ID selectors
# - CSS variables (custom properties)
# - Media queries and breakpoints
# - Animations and transitions
# - SCSS/Sass mixins and nesting
# - CSS modules and imports

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract CSS selectors and rules
# 
# Finds class names, IDs, and element selectors
#
# Arguments:
#   $1 - Path to CSS file
# Returns: JSON with selector information
#######################################
extractor::lib::css::extract_selectors() {
    local file="$1"
    local selectors=()
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract class selectors
    local class_count=$(grep -oE '\.[a-zA-Z][-_a-zA-Z0-9]*' "$file" 2>/dev/null | sort -u | wc -l || echo "0")
    
    # Extract ID selectors
    local id_count=$(grep -oE '#[a-zA-Z][-_a-zA-Z0-9]*' "$file" 2>/dev/null | sort -u | wc -l || echo "0")
    
    # Extract element selectors
    local element_count=$(grep -oE '^[a-zA-Z]+[[:space:]]*\{' "$file" 2>/dev/null | wc -l || echo "0")
    
    # Extract pseudo-classes
    local pseudo_count=$(grep -oE ':[a-zA-Z-]+' "$file" 2>/dev/null | sort -u | wc -l || echo "0")
    
    jq -n \
        --arg classes "$class_count" \
        --arg ids "$id_count" \
        --arg elements "$element_count" \
        --arg pseudo "$pseudo_count" \
        '{
            class_selectors: ($classes | tonumber),
            id_selectors: ($ids | tonumber),
            element_selectors: ($elements | tonumber),
            pseudo_classes: ($pseudo | tonumber)
        }'
}

#######################################
# Extract CSS variables and custom properties
# 
# Finds CSS custom properties and Sass/LESS variables
#
# Arguments:
#   $1 - Path to CSS file
# Returns: JSON with variable information
#######################################
extractor::lib::css::extract_variables() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "{}"
        return
    fi
    
    # CSS custom properties (--variable-name)
    local css_vars=$(grep -oE '--[-_a-zA-Z0-9]+' "$file" 2>/dev/null | sort -u | wc -l || echo "0")
    
    # Sass variables ($variable)
    local sass_vars=$(grep -oE '\$[-_a-zA-Z0-9]+' "$file" 2>/dev/null | sort -u | wc -l || echo "0")
    
    # LESS variables (@variable)
    local less_vars=$(grep -oE '@[-_a-zA-Z0-9]+' "$file" 2>/dev/null | sort -u | wc -l || echo "0")
    
    jq -n \
        --arg css "$css_vars" \
        --arg sass "$sass_vars" \
        --arg less "$less_vars" \
        '{
            css_custom_properties: ($css | tonumber),
            sass_variables: ($sass | tonumber),
            less_variables: ($less | tonumber)
        }'
}

#######################################
# Extract media queries and breakpoints
# 
# Finds responsive design breakpoints
#
# Arguments:
#   $1 - Path to CSS file
# Returns: JSON with media query information
#######################################
extractor::lib::css::extract_media_queries() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "{}"
        return
    fi
    
    # Count media queries
    local media_count=$(grep -c '@media' "$file" 2>/dev/null || echo "0")
    
    # Extract breakpoint values
    local breakpoints=$(grep -oE '@media.*\(.*px\)' "$file" 2>/dev/null | \
        grep -oE '[0-9]+px' | sed 's/px//' | sort -nu | tr '\n' ',' | sed 's/,$//')
    
    # Check for common breakpoint patterns
    local has_mobile="false"
    local has_tablet="false"
    local has_desktop="false"
    
    if echo "$breakpoints" | grep -qE '(320|375|414|480)'; then
        has_mobile="true"
    fi
    if echo "$breakpoints" | grep -qE '(768|834|1024)'; then
        has_tablet="true"
    fi
    if echo "$breakpoints" | grep -qE '(1280|1440|1920)'; then
        has_desktop="true"
    fi
    
    jq -n \
        --arg count "$media_count" \
        --arg breakpoints "$breakpoints" \
        --arg mobile "$has_mobile" \
        --arg tablet "$has_tablet" \
        --arg desktop "$has_desktop" \
        '{
            media_query_count: ($count | tonumber),
            breakpoints: $breakpoints,
            has_mobile: ($mobile == "true"),
            has_tablet: ($tablet == "true"),
            has_desktop: ($desktop == "true")
        }'
}

#######################################
# Extract animations and transitions
# 
# Finds CSS animations and transitions
#
# Arguments:
#   $1 - Path to CSS file
# Returns: JSON with animation information
#######################################
extractor::lib::css::extract_animations() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "{}"
        return
    fi
    
    # Count animations
    local keyframes_count=$(grep -c '@keyframes' "$file" 2>/dev/null || echo "0")
    
    # Count transitions
    local transition_count=$(grep -c 'transition:' "$file" 2>/dev/null || echo "0")
    
    # Count transforms
    local transform_count=$(grep -c 'transform:' "$file" 2>/dev/null || echo "0")
    
    # Extract animation names
    local animation_names=$(grep -oE '@keyframes\s+[a-zA-Z][-_a-zA-Z0-9]*' "$file" 2>/dev/null | \
        sed 's/@keyframes\s*//' | sort -u | tr '\n' ',' | sed 's/,$//')
    
    jq -n \
        --arg keyframes "$keyframes_count" \
        --arg transitions "$transition_count" \
        --arg transforms "$transform_count" \
        --arg names "$animation_names" \
        '{
            keyframes_count: ($keyframes | tonumber),
            transition_count: ($transitions | tonumber),
            transform_count: ($transforms | tonumber),
            animation_names: $names
        }'
}

#######################################
# Extract preprocessor features
# 
# Detects SCSS/Sass/LESS specific features
#
# Arguments:
#   $1 - Path to CSS file
# Returns: JSON with preprocessor information
#######################################
extractor::lib::css::extract_preprocessor_features() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "{}"
        return
    fi
    
    # Detect mixins
    local mixin_count=$(grep -cE '@mixin|@include' "$file" 2>/dev/null || echo "0")
    
    # Detect nesting (simplified check)
    local has_nesting="false"
    if grep -qE '^\s+&' "$file" 2>/dev/null; then
        has_nesting="true"
    fi
    
    # Detect extends
    local extend_count=$(grep -cE '@extend|%' "$file" 2>/dev/null || echo "0")
    
    # Detect imports
    local import_count=$(grep -cE '@import|@use' "$file" 2>/dev/null || echo "0")
    
    jq -n \
        --arg mixins "$mixin_count" \
        --arg nesting "$has_nesting" \
        --arg extends "$extend_count" \
        --arg imports "$import_count" \
        '{
            mixin_count: ($mixins | tonumber),
            has_nesting: ($nesting == "true"),
            extend_count: ($extends | tonumber),
            import_count: ($imports | tonumber)
        }'
}

#######################################
# Detect CSS framework usage
# 
# Identifies common CSS frameworks
#
# Arguments:
#   $1 - Path to CSS file
# Returns: Framework name or "custom"
#######################################
extractor::lib::css::detect_framework() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "unknown"
        return
    fi
    
    # Check for framework signatures
    if grep -qE 'bootstrap|\.btn-primary|\.container-fluid' "$file" 2>/dev/null; then
        echo "bootstrap"
    elif grep -qE 'tailwind|\.text-gray-|\.bg-blue-' "$file" 2>/dev/null; then
        echo "tailwind"
    elif grep -qE 'bulma|\.hero|\.section' "$file" 2>/dev/null; then
        echo "bulma"
    elif grep -qE 'foundation|\.grid-x|\.cell' "$file" 2>/dev/null; then
        echo "foundation"
    elif grep -qE 'materialize|\.waves-effect|\.card-panel' "$file" 2>/dev/null; then
        echo "materialize"
    elif grep -qE 'semantic-ui|\.ui\.button' "$file" 2>/dev/null; then
        echo "semantic-ui"
    else
        echo "custom"
    fi
}

#######################################
# Extract all CSS information from a file
# 
# Main extraction function that combines all CSS extractions
#
# Arguments:
#   $1 - CSS file path or directory
#   $2 - Component type (styles, theme, etc.)
#   $3 - Scenario/resource name
# Returns: JSON lines with all CSS information
#######################################
extractor::lib::css::extract_all() {
    local path="$1"
    local component_type="${2:-styles}"
    local scenario_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        local file="$path"
        local filename=$(basename "$file")
        local file_ext="${filename##*.}"
        
        # Check if it's a CSS-related file
        case "$file_ext" in
            css|scss|sass|less|styl)
                ;;
            *)
                return 1
                ;;
        esac
        
        # Get file statistics
        local line_count=$(wc -l < "$file" 2>/dev/null || echo "0")
        local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
        
        # Detect preprocessor type
        local preprocessor="css"
        case "$file_ext" in
            scss|sass) preprocessor="sass" ;;
            less) preprocessor="less" ;;
            styl) preprocessor="stylus" ;;
        esac
        
        # Extract components
        local selectors=$(extractor::lib::css::extract_selectors "$file")
        local variables=$(extractor::lib::css::extract_variables "$file")
        local media=$(extractor::lib::css::extract_media_queries "$file")
        local animations=$(extractor::lib::css::extract_animations "$file")
        local preprocessor_features=$(extractor::lib::css::extract_preprocessor_features "$file")
        local framework=$(extractor::lib::css::detect_framework "$file")
        
        # Build content summary
        local content="CSS: $filename | Component: $component_type"
        content="$content | Preprocessor: $preprocessor"
        [[ "$framework" != "custom" ]] && content="$content | Framework: $framework"
        
        local selector_count=$(echo "$selectors" | jq '.class_selectors + .id_selectors')
        content="$content | Selectors: $selector_count"
        
        local var_count=$(echo "$variables" | jq '.css_custom_properties + .sass_variables + .less_variables')
        [[ $var_count -gt 0 ]] && content="$content | Variables: $var_count"
        
        local media_count=$(echo "$media" | jq '.media_query_count')
        [[ $media_count -gt 0 ]] && content="$content | Media Queries: $media_count"
        
        # Output main file overview
        jq -n \
            --arg content "$content" \
            --arg scenario "$scenario_name" \
            --arg source_file "$file" \
            --arg filename "$filename" \
            --arg component_type "$component_type" \
            --arg preprocessor "$preprocessor" \
            --arg framework "$framework" \
            --arg line_count "$line_count" \
            --arg file_size "$file_size" \
            --argjson selectors "$selectors" \
            --argjson variables "$variables" \
            --argjson media "$media" \
            --argjson animations "$animations" \
            --argjson preprocessor_features "$preprocessor_features" \
            '{
                content: $content,
                metadata: {
                    scenario: $scenario,
                    source_file: $source_file,
                    filename: $filename,
                    component_type: $component_type,
                    language: "css",
                    preprocessor: $preprocessor,
                    framework: $framework,
                    line_count: ($line_count | tonumber),
                    file_size: ($file_size | tonumber),
                    selectors: $selectors,
                    variables: $variables,
                    media_queries: $media,
                    animations: $animations,
                    preprocessor_features: $preprocessor_features,
                    content_type: "stylesheet",
                    extraction_method: "css_parser"
                }
            }' | jq -c
    elif [[ -d "$path" ]]; then
        # Directory - find all CSS-related files
        local css_files=()
        while IFS= read -r file; do
            css_files+=("$file")
        done < <(find "$path" -type f \( -name "*.css" -o -name "*.scss" -o -name "*.sass" -o -name "*.less" -o -name "*.styl" \) 2>/dev/null)
        
        if [[ ${#css_files[@]} -eq 0 ]]; then
            return 1
        fi
        
        for file in "${css_files[@]}"; do
            extractor::lib::css::extract_all "$file" "$component_type" "$scenario_name"
        done
    fi
}

#######################################
# Check if directory contains CSS files
# 
# Helper function to detect CSS presence
#
# Arguments:
#   $1 - Directory path
# Returns: 0 if CSS files found, 1 otherwise
#######################################
extractor::lib::css::has_css_files() {
    local dir="$1"
    
    if find "$dir" -type f \( -name "*.css" -o -name "*.scss" -o -name "*.sass" -o -name "*.less" -o -name "*.styl" \) 2>/dev/null | grep -q .; then
        return 0
    else
        return 1
    fi
}

# Export all functions
export -f extractor::lib::css::extract_selectors
export -f extractor::lib::css::extract_variables
export -f extractor::lib::css::extract_media_queries
export -f extractor::lib::css::extract_animations
export -f extractor::lib::css::extract_preprocessor_features
export -f extractor::lib::css::detect_framework
export -f extractor::lib::css::extract_all
export -f extractor::lib::css::has_css_files