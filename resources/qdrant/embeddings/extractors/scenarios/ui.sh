#!/usr/bin/env bash
# Scenario UI Extractor
# Extracts UI implementation details regardless of framework
#
# Uses language-specific handlers to extract:
# - Components and views
# - Styles and themes
# - Configuration files
# - Build and deployment configs

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source unified code extractor (handles all language detection and dispatch)
source "${APP_ROOT}/resources/qdrant/embeddings/lib/code-extractor.sh"

#######################################
# Detect UI framework and technology
# 
# Analyzes ui/ directory to determine framework and language
#
# Arguments:
#   $1 - Path to ui directory
# Returns: Framework:language combination
#######################################
qdrant::extract::detect_ui_framework() {
    local ui_dir="$1"
    
    if [[ ! -d "$ui_dir" ]]; then
        echo "none:unknown"
        return
    fi
    
    local framework="unknown"
    local language="unknown"
    
    # Check for package.json first (most common for modern UI)
    if [[ -f "$ui_dir/package.json" ]]; then
        language="javascript"
        
        # Check for popular frameworks in package.json
        local package_content=$(cat "$ui_dir/package.json" 2>/dev/null)
        
        if echo "$package_content" | grep -q "\"react\"" 2>/dev/null; then
            framework="react"
        elif echo "$package_content" | grep -q "\"vue\"" 2>/dev/null; then
            framework="vue"
        elif echo "$package_content" | grep -q "\"angular\"" 2>/dev/null; then
            framework="angular"
        elif echo "$package_content" | grep -q "\"svelte\"" 2>/dev/null; then
            framework="svelte"
        elif echo "$package_content" | grep -q "\"next\"" 2>/dev/null; then
            framework="nextjs"
        elif echo "$package_content" | grep -q "\"nuxt\"" 2>/dev/null; then
            framework="nuxtjs"
        elif echo "$package_content" | grep -q "\"vite\"" 2>/dev/null; then
            framework="vite"
        elif echo "$package_content" | grep -q "\"webpack\"" 2>/dev/null; then
            framework="webpack"
        else
            framework="javascript"
        fi
    elif [[ -f "$ui_dir/Package.swift" ]] || find "$ui_dir" -name "*.swift" 2>/dev/null | grep -q .; then
        language="swift"
        framework="swift_ui"
        
        # Check for iOS/macOS frameworks
        if find "$ui_dir" -name "*.swift" -exec grep -l "SwiftUI" {} \; 2>/dev/null | grep -q .; then
            framework="swiftui"
        elif find "$ui_dir" -name "*.swift" -exec grep -l "UIKit" {} \; 2>/dev/null | grep -q .; then
            framework="uikit"
        elif find "$ui_dir" -name "*.swift" -exec grep -l "AppKit" {} \; 2>/dev/null | grep -q .; then
            framework="appkit"
        fi
    elif [[ -f "$ui_dir/build.gradle" ]] || [[ -f "$ui_dir/build.gradle.kts" ]] || find "$ui_dir" -name "*.kt" 2>/dev/null | grep -q .; then
        language="kotlin"
        framework="kotlin_ui"
        
        # Check for Android/Compose
        if [[ -f "$ui_dir/build.gradle" ]] || [[ -f "$ui_dir/build.gradle.kts" ]]; then
            local gradle_file="$ui_dir/build.gradle"
            [[ -f "$ui_dir/build.gradle.kts" ]] && gradle_file="$ui_dir/build.gradle.kts"
            
            if grep -q "compose" "$gradle_file" 2>/dev/null; then
                framework="compose"
            elif grep -q "android" "$gradle_file" 2>/dev/null; then
                framework="android"
            fi
        fi
    elif [[ -f "$ui_dir/composer.json" ]] || find "$ui_dir" -name "*.php" 2>/dev/null | grep -q .; then
        language="php"
        framework="php_web"
        
        # Check for PHP web frameworks
        if [[ -f "$ui_dir/composer.json" ]]; then
            local composer_content=$(cat "$ui_dir/composer.json" 2>/dev/null)
            if echo "$composer_content" | grep -q "laravel"; then
                framework="laravel"
            elif echo "$composer_content" | grep -q "symfony"; then
                framework="symfony"
            elif echo "$composer_content" | grep -q "codeigniter"; then
                framework="codeigniter"
            fi
        fi
    elif [[ -f "$ui_dir"/*.csproj ]] || find "$ui_dir" -name "*.cs" 2>/dev/null | grep -q .; then
        language="csharp"
        framework="csharp_ui"
        
        # Check for .NET UI frameworks
        if find "$ui_dir" -name "*.cs" -exec grep -l "Blazor" {} \; 2>/dev/null | grep -q .; then
            framework="blazor"
        elif find "$ui_dir" -name "*.cs" -exec grep -l "WPF\|Windows.UI" {} \; 2>/dev/null | grep -q .; then
            framework="wpf"
        elif find "$ui_dir" -name "*.cs" -exec grep -l "Xamarin" {} \; 2>/dev/null | grep -q .; then
            framework="xamarin"
        fi
    elif [[ -f "$ui_dir/CMakeLists.txt" ]] || find "$ui_dir" -name "*.cpp" -o -name "*.h" 2>/dev/null | grep -q .; then
        language="cpp"
        framework="cpp_ui"
        
        # Check for C++ UI frameworks
        if [[ -f "$ui_dir/CMakeLists.txt" ]]; then
            local cmake_content=$(cat "$ui_dir/CMakeLists.txt" 2>/dev/null)
            if echo "$cmake_content" | grep -q "Qt"; then
                framework="qt"
            elif echo "$cmake_content" | grep -q "GTK"; then
                framework="gtk"
            elif echo "$cmake_content" | grep -q "wxWidgets"; then
                framework="wxwidgets"
            fi
        fi
    elif [[ -f "$ui_dir/go.mod" ]] || find "$ui_dir" -name "*.go" 2>/dev/null | grep -q .; then
        language="go"
        framework="go_web"
        
        # Check for specific Go web frameworks
        if [[ -f "$ui_dir/go.mod" ]]; then
            local go_mod=$(cat "$ui_dir/go.mod" 2>/dev/null)
            if echo "$go_mod" | grep -q "gin-gonic/gin"; then
                framework="gin"
            elif echo "$go_mod" | grep -q "gorilla/mux"; then
                framework="gorilla"
            elif echo "$go_mod" | grep -q "echo"; then
                framework="echo"
            fi
        fi
    elif find "$ui_dir" -name "*.py" 2>/dev/null | grep -q .; then
        language="python"
        framework="python_web"
        
        # Check for Python web frameworks
        if [[ -f "$ui_dir/requirements.txt" ]]; then
            local requirements=$(cat "$ui_dir/requirements.txt" 2>/dev/null)
            if echo "$requirements" | grep -q "streamlit"; then
                framework="streamlit"
            elif echo "$requirements" | grep -q "flask"; then
                framework="flask"
            elif echo "$requirements" | grep -q "django"; then
                framework="django"
            elif echo "$requirements" | grep -q "fastapi"; then
                framework="fastapi"
            fi
        fi
    elif find "$ui_dir" -name "*.html" -o -name "*.css" -o -name "*.js" 2>/dev/null | grep -q .; then
        language="html"
        framework="static_web"
    fi
    
    echo "$framework:$language"
}

#######################################
# Extract UI overview and structure
# 
# Provides high-level information about the UI implementation
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON line with UI overview
#######################################
qdrant::extract::scenario_ui_overview() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    local ui_dir="$dir/ui"
    
    if [[ ! -d "$ui_dir" ]]; then
        return 1
    fi
    
    log::debug "Extracting UI overview for $scenario_name" >&2
    
    # Detect framework and language
    local framework_info=$(qdrant::extract::detect_ui_framework "$ui_dir")
    local framework="${framework_info%:*}"
    local language="${framework_info#*:}"
    
    # Count different file types
    local total_files=$(find "$ui_dir" -type f 2>/dev/null | wc -l)
    local component_files=0
    local style_files=0
    local config_files=0
    local asset_files=0
    
    # Count by category
    while IFS= read -r file; do
        case "$file" in
            *.js|*.ts|*.jsx|*.tsx|*.vue|*.svelte|*.py|*.go|*.html)
                ((component_files++))
                ;;
            *.css|*.scss|*.sass|*.less|*.styl)
                ((style_files++))
                ;;
            *.json|*.yaml|*.yml|*.toml|*.env|*.conf)
                ((config_files++))
                ;;
            *.png|*.jpg|*.jpeg|*.gif|*.svg|*.ico|*.woff|*.woff2|*.ttf)
                ((asset_files++))
                ;;
        esac
    done < <(find "$ui_dir" -type f 2>/dev/null)
    
    # Check for common UI patterns
    local has_components="false"
    local has_pages="false"
    local has_routes="false"
    local has_state="false"
    local has_build="false"
    local has_tests="false"
    
    # Look for common directory structures
    [[ -d "$ui_dir/components" ]] || [[ -d "$ui_dir/src/components" ]] && has_components="true"
    [[ -d "$ui_dir/pages" ]] || [[ -d "$ui_dir/src/pages" ]] || [[ -d "$ui_dir/views" ]] && has_pages="true"
    [[ -d "$ui_dir/routes" ]] || [[ -d "$ui_dir/src/routes" ]] && has_routes="true"
    [[ -d "$ui_dir/store" ]] || [[ -d "$ui_dir/src/store" ]] || [[ -d "$ui_dir/state" ]] && has_state="true"
    [[ -f "$ui_dir/webpack.config.js" ]] || [[ -f "$ui_dir/vite.config.js" ]] || [[ -f "$ui_dir/next.config.js" ]] && has_build="true"
    [[ -d "$ui_dir/test" ]] || [[ -d "$ui_dir/__tests__" ]] || [[ -d "$ui_dir/src/__tests__" ]] && has_tests="true"
    
    # Estimate component count
    local estimated_components=0
    case "$framework" in
        react|vue|angular|svelte)
            estimated_components=$(find "$ui_dir" -name "*.jsx" -o -name "*.tsx" -o -name "*.vue" -o -name "*.svelte" 2>/dev/null | wc -l)
            ;;
        *)
            estimated_components=$component_files
            ;;
    esac
    
    # Build content
    local content="UI: $scenario_name | Framework: $framework | Language: $language"
    content="$content | Files: $total_files | Components: ~$estimated_components"
    content="$content | Styles: $style_files | Assets: $asset_files"
    [[ "$has_components" == "true" ]] && content="$content | ComponentDir: yes"
    [[ "$has_pages" == "true" ]] && content="$content | Pages: yes"
    [[ "$has_state" == "true" ]] && content="$content | State: yes"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg scenario "$scenario_name" \
        --arg source_dir "$ui_dir" \
        --arg framework "$framework" \
        --arg language "$language" \
        --arg total_files "$total_files" \
        --arg component_files "$component_files" \
        --arg style_files "$style_files" \
        --arg config_files "$config_files" \
        --arg asset_files "$asset_files" \
        --arg estimated_components "$estimated_components" \
        --arg has_components "$has_components" \
        --arg has_pages "$has_pages" \
        --arg has_routes "$has_routes" \
        --arg has_state "$has_state" \
        --arg has_build "$has_build" \
        --arg has_tests "$has_tests" \
        '{
            content: $content,
            metadata: {
                scenario: $scenario,
                source_directory: $source_dir,
                component_type: "ui",
                ui_framework: $framework,
                ui_language: $language,
                total_files: ($total_files | tonumber),
                component_files: ($component_files | tonumber),
                style_files: ($style_files | tonumber),
                config_files: ($config_files | tonumber),
                asset_files: ($asset_files | tonumber),
                estimated_components: ($estimated_components | tonumber),
                has_components: ($has_components == "true"),
                has_pages: ($has_pages == "true"),
                has_routes: ($has_routes == "true"),
                has_state_management: ($has_state == "true"),
                has_build_config: ($has_build == "true"),
                has_tests: ($has_tests == "true"),
                content_type: "scenario_ui",
                extraction_method: "ui_overview_analyzer"
            }
        }' | jq -c
}

#######################################
# Extract UI implementation details
# 
# Uses appropriate language handler to extract UI code
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines with UI implementation details
#######################################
qdrant::extract::scenario_ui_implementation() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    local ui_dir="$dir/ui"
    
    if [[ ! -d "$ui_dir" ]]; then
        return 1
    fi
    
    log::debug "Extracting UI implementation for $scenario_name" >&2
    
    # Use unified code extractor with auto strategy (handles single and multi-language)
    qdrant::lib::extract_code "$ui_dir" "ui" "$scenario_name" "auto"
}

#######################################
# Extract UI build and deployment configuration
# 
# Processes build tools and deployment configs
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines with UI build configuration
#######################################
qdrant::extract::scenario_ui_build_config() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    local ui_dir="$dir/ui"
    
    if [[ ! -d "$ui_dir" ]]; then
        return 1
    fi
    
    # Look for build configuration files
    local build_configs=()
    [[ -f "$ui_dir/webpack.config.js" ]] && build_configs+=("$ui_dir/webpack.config.js")
    [[ -f "$ui_dir/vite.config.js" ]] && build_configs+=("$ui_dir/vite.config.js")
    [[ -f "$ui_dir/vite.config.ts" ]] && build_configs+=("$ui_dir/vite.config.ts")
    [[ -f "$ui_dir/next.config.js" ]] && build_configs+=("$ui_dir/next.config.js")
    [[ -f "$ui_dir/nuxt.config.js" ]] && build_configs+=("$ui_dir/nuxt.config.js")
    [[ -f "$ui_dir/vue.config.js" ]] && build_configs+=("$ui_dir/vue.config.js")
    [[ -f "$ui_dir/angular.json" ]] && build_configs+=("$ui_dir/angular.json")
    [[ -f "$ui_dir/svelte.config.js" ]] && build_configs+=("$ui_dir/svelte.config.js")
    [[ -f "$ui_dir/tailwind.config.js" ]] && build_configs+=("$ui_dir/tailwind.config.js")
    [[ -f "$ui_dir/postcss.config.js" ]] && build_configs+=("$ui_dir/postcss.config.js")
    [[ -f "$ui_dir/tsconfig.json" ]] && build_configs+=("$ui_dir/tsconfig.json")
    [[ -f "$ui_dir/Dockerfile" ]] && build_configs+=("$ui_dir/Dockerfile")
    [[ -f "$ui_dir/.env" ]] && build_configs+=("$ui_dir/.env")
    [[ -f "$ui_dir/.env.example" ]] && build_configs+=("$ui_dir/.env.example")
    
    for config_file in "${build_configs[@]}"; do
        local filename=$(basename "$config_file")
        local config_type="${filename%.*}"
        
        # Extract basic configuration info
        local config_summary=""
        case "$filename" in
            *.json)
                config_summary=$(jq -r 'keys | join(", ")' "$config_file" 2>/dev/null || echo "")
                ;;
            *.js|*.ts)
                # Look for common configuration patterns
                config_summary=$(grep -E "module\.exports|export default|export" "$config_file" 2>/dev/null | head -3 | tr '\n' '; ' || echo "")
                ;;
            .env*)
                # Count environment variables
                local env_count=$(grep -c "^[A-Z_][A-Z0-9_]*=" "$config_file" 2>/dev/null || echo "0")
                config_summary="$env_count environment variables"
                ;;
            Dockerfile)
                # Extract base image and key instructions
                local base_image=$(grep "^FROM" "$config_file" | head -1 | cut -d' ' -f2 || echo "")
                config_summary="Base: $base_image"
                ;;
        esac
        
        local content="Build Config: $filename | UI: $scenario_name | Type: $config_type"
        [[ -n "$config_summary" ]] && content="$content | Summary: $config_summary"
        
        jq -n \
            --arg content "$content" \
            --arg scenario "$scenario_name" \
            --arg source_file "$config_file" \
            --arg filename "$filename" \
            --arg config_type "$config_type" \
            --arg config_summary "$config_summary" \
            '{
                content: $content,
                metadata: {
                    scenario: $scenario,
                    source_file: $source_file,
                    component_type: "ui",
                    config_type: $config_type,
                    filename: $filename,
                    config_summary: $config_summary,
                    content_type: "scenario_ui",
                    extraction_method: "ui_build_config_parser"
                }
            }' | jq -c
    done
}

#######################################
# Extract all UI information
# 
# Main function that calls all UI extractors
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines for all UI components
#######################################
qdrant::extract::scenario_ui_all() {
    local dir="$1"
    
    if [[ ! -d "$dir/ui" ]]; then
        return 1
    fi
    
    # Extract UI overview
    qdrant::extract::scenario_ui_overview "$dir" 2>/dev/null || true
    
    # Extract implementation details
    qdrant::extract::scenario_ui_implementation "$dir" 2>/dev/null || true
    
    # Extract build configuration
    qdrant::extract::scenario_ui_build_config "$dir" 2>/dev/null || true
}

# Export functions for use by main.sh
export -f qdrant::extract::detect_ui_framework
export -f qdrant::extract::scenario_ui_overview
export -f qdrant::extract::scenario_ui_implementation
export -f qdrant::extract::scenario_ui_build_config
export -f qdrant::extract::scenario_ui_all