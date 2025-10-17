#!/usr/bin/env bash
# Scenario API Extractor
# Extracts API implementation details regardless of language
#
# Uses language-specific handlers to extract:
# - API endpoints and routes
# - Functions and methods
# - Database models and schemas
# - Configuration files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source unified code extractor (handles all language detection and dispatch)
source "${APP_ROOT}/resources/qdrant/embeddings/lib/code-extractor.sh"

#######################################
# Detect API implementation language
# 
# Wrapper around unified detector for backward compatibility
#
# Arguments:
#   $1 - Path to api directory
# Returns: Primary language name
#######################################
qdrant::extract::detect_api_language() {
    local api_dir="$1"
    qdrant::lib::detect_primary_language "$api_dir"
}

#######################################
# Extract API overview and structure
# 
# Provides high-level information about the API implementation
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON line with API overview
#######################################
qdrant::extract::scenario_api_overview() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    local api_dir="$dir/api"
    
    if [[ ! -d "$api_dir" ]]; then
        return 1
    fi
    
    log::debug "Extracting API overview for $scenario_name" >&2
    
    # Detect implementation details
    local primary_language=$(qdrant::lib::detect_primary_language "$api_dir")
    
    # Count different file types
    local total_files=$(find "$api_dir" -type f 2>/dev/null | wc -l)
    local code_files=0
    local config_files=0
    
    # Count by category
    while IFS= read -r file; do
        case "$file" in
            *.go|*.js|*.ts|*.py|*.sh)
                ((code_files++))
                ;;
            *.json|*.yaml|*.yml|*.toml|*.env)
                ((config_files++))
                ;;
        esac
    done < <(find "$api_dir" -type f 2>/dev/null)
    
    # Check for common API patterns
    local has_routes="false"
    local has_models="false"
    local has_middleware="false"
    local has_tests="false"
    
    # Look for common directory structures
    [[ -d "$api_dir/routes" ]] || [[ -d "$api_dir/handlers" ]] && has_routes="true"
    [[ -d "$api_dir/models" ]] || [[ -d "$api_dir/schemas" ]] && has_models="true"
    [[ -d "$api_dir/middleware" ]] && has_middleware="true"
    [[ -d "$api_dir/test" ]] || [[ -d "$api_dir/tests" ]] && has_tests="true"
    
    # Check for API documentation
    local has_docs="false"
    [[ -f "$api_dir/README.md" ]] || [[ -f "$api_dir/API.md" ]] && has_docs="true"
    
    # Build content
    local content="API: $scenario_name | Language: $primary_language | Files: $total_files"
    content="$content | Code: $code_files | Config: $config_files"
    [[ "$has_routes" == "true" ]] && content="$content | Routes: yes"
    [[ "$has_models" == "true" ]] && content="$content | Models: yes"
    [[ "$has_middleware" == "true" ]] && content="$content | Middleware: yes"
    [[ "$has_tests" == "true" ]] && content="$content | Tests: yes"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg scenario "$scenario_name" \
        --arg source_dir "$api_dir" \
        --arg primary_language "$primary_language" \
        --arg total_files "$total_files" \
        --arg code_files "$code_files" \
        --arg config_files "$config_files" \
        --arg has_routes "$has_routes" \
        --arg has_models "$has_models" \
        --arg has_middleware "$has_middleware" \
        --arg has_tests "$has_tests" \
        --arg has_docs "$has_docs" \
        '{
            content: $content,
            metadata: {
                scenario: $scenario,
                source_directory: $source_dir,
                component_type: "api",
                primary_language: $primary_language,
                total_files: ($total_files | tonumber),
                code_files: ($code_files | tonumber),
                config_files: ($config_files | tonumber),
                has_routes: ($has_routes == "true"),
                has_models: ($has_models == "true"),
                has_middleware: ($has_middleware == "true"),
                has_tests: ($has_tests == "true"),
                has_documentation: ($has_docs == "true"),
                content_type: "scenario_api",
                extraction_method: "api_overview_analyzer"
            }
        }' | jq -c
}

#######################################
# Extract API implementation details
# 
# Uses unified code extractor with automatic language detection
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines with API implementation details
#######################################
qdrant::extract::scenario_api_implementation() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    local api_dir="$dir/api"
    
    if [[ ! -d "$api_dir" ]]; then
        return 1
    fi
    
    log::debug "Extracting API implementation for $scenario_name" >&2
    
    # Use unified code extractor with auto strategy (handles single and multi-language)
    qdrant::lib::extract_code "$api_dir" "api" "$scenario_name" "auto"
}

#######################################
# Extract API configuration files
# 
# Processes configuration files specific to API implementation
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines with API configuration
#######################################
qdrant::extract::scenario_api_config() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    local api_dir="$dir/api"
    
    if [[ ! -d "$api_dir" ]]; then
        return 1
    fi
    
    # Find configuration files
    while IFS= read -r config_file; do
        local filename=$(basename "$config_file")
        local file_ext="${filename##*.}"
        
        case "$file_ext" in
            json)
                if [[ "$filename" != "package.json" ]]; then  # package.json handled by language extractor
                    local config_json=$(cat "$config_file" 2>/dev/null || echo "{}")
                    local config_summary=$(echo "$config_json" | jq -r 'keys | join(", ")' 2>/dev/null || echo "")
                    
                    local content="Config: $filename | API: $scenario_name | Keys: $config_summary"
                    
                    jq -n \
                        --arg content "$content" \
                        --arg scenario "$scenario_name" \
                        --arg source_file "$config_file" \
                        --arg filename "$filename" \
                        --argjson config "$config_json" \
                        '{
                            content: $content,
                            metadata: {
                                scenario: $scenario,
                                source_file: $source_file,
                                component_type: "api",
                                config_type: "json",
                                filename: $filename,
                                configuration: $config,
                                content_type: "scenario_api",
                                extraction_method: "api_config_parser"
                            }
                        }' | jq -c
                fi
                ;;
            env)
                # Extract environment variable names (not values for security)
                local env_vars=$(grep -E "^[A-Z_][A-Z0-9_]*=" "$config_file" 2>/dev/null | cut -d= -f1 | head -10 | tr '\n' ', ' || echo "")
                
                local content="Config: $filename | API: $scenario_name | Variables: $env_vars"
                
                jq -n \
                    --arg content "$content" \
                    --arg scenario "$scenario_name" \
                    --arg source_file "$config_file" \
                    --arg filename "$filename" \
                    --arg env_vars "$env_vars" \
                    '{
                        content: $content,
                        metadata: {
                            scenario: $scenario,
                            source_file: $source_file,
                            component_type: "api",
                            config_type: "environment",
                            filename: $filename,
                            environment_variables: $env_vars,
                            content_type: "scenario_api",
                            extraction_method: "api_config_parser"
                        }
                    }' | jq -c
                ;;
        esac
    done < <(find "$api_dir" -type f \( -name "*.json" -o -name "*.env" -o -name "*.yaml" -o -name "*.yml" \) 2>/dev/null)
}

#######################################
# Extract all API information
# 
# Main function that calls all API extractors
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines for all API components
#######################################
qdrant::extract::scenario_api_all() {
    local dir="$1"
    
    if [[ ! -d "$dir/api" ]]; then
        return 1
    fi
    
    # Extract API overview
    qdrant::extract::scenario_api_overview "$dir" 2>/dev/null || true
    
    # Extract implementation details
    qdrant::extract::scenario_api_implementation "$dir" 2>/dev/null || true
    
    # Extract configuration files
    qdrant::extract::scenario_api_config "$dir" 2>/dev/null || true
}

# Export functions for use by main.sh
export -f qdrant::extract::detect_api_language
export -f qdrant::extract::scenario_api_overview
export -f qdrant::extract::scenario_api_implementation  
export -f qdrant::extract::scenario_api_config
export -f qdrant::extract::scenario_api_all