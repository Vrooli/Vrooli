#!/usr/bin/env bash
# Scenario CLI Extractor
# Extracts CLI implementation details regardless of language
#
# Uses language-specific handlers to extract:
# - CLI commands and subcommands
# - Scripts and utilities
# - Configuration and setup files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source unified code extractor (handles all language detection and dispatch)
source "${APP_ROOT}/resources/qdrant/embeddings/lib/code-extractor.sh"

#######################################
# Detect CLI implementation language
# 
# Wrapper around unified detector for backward compatibility
#
# Arguments:
#   $1 - Path to cli directory
# Returns: Primary language name
#######################################
qdrant::extract::detect_cli_language() {
    local cli_dir="$1"
    qdrant::lib::detect_primary_language "$cli_dir"
}

#######################################
# Extract CLI overview and structure
# 
# Provides high-level information about the CLI implementation
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON line with CLI overview
#######################################
qdrant::extract::scenario_cli_overview() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    local cli_dir="$dir/cli"
    
    if [[ ! -d "$cli_dir" ]]; then
        return 1
    fi
    
    log::debug "Extracting CLI overview for $scenario_name" >&2
    
    # Detect implementation details
    local primary_language=$(qdrant::lib::detect_primary_language "$cli_dir")
    
    # Count different file types
    local total_files=$(find "$cli_dir" -type f 2>/dev/null | wc -l)
    local script_files=0
    local config_files=0
    
    # Count by category
    while IFS= read -r file; do
        case "$file" in
            *.go|*.js|*.ts|*.py|*.sh)
                ((script_files++))
                ;;
            *.json|*.yaml|*.yml|*.toml|*.env)
                ((config_files++))
                ;;
        esac
    done < <(find "$cli_dir" -type f 2>/dev/null)
    
    # Check for common CLI patterns
    local has_main_entry="false"
    local has_commands="false"
    local has_help="false"
    local has_config="false"
    local has_tests="false"
    
    # Look for main entry points
    [[ -f "$cli_dir/main.sh" ]] || [[ -f "$cli_dir/cli.sh" ]] || [[ -f "$cli_dir/main.go" ]] || [[ -f "$cli_dir/index.js" ]] && has_main_entry="true"
    
    # Look for command structure
    [[ -d "$cli_dir/commands" ]] || [[ -d "$cli_dir/cmd" ]] && has_commands="true"
    
    # Check for help/documentation
    [[ -f "$cli_dir/README.md" ]] || [[ -f "$cli_dir/USAGE.md" ]] && has_help="true"
    
    # Check for configuration
    [[ -f "$cli_dir/config.json" ]] || [[ -f "$cli_dir/.env" ]] && has_config="true"
    
    # Check for tests
    [[ -d "$cli_dir/test" ]] || [[ -d "$cli_dir/tests" ]] && has_tests="true"
    
    # Estimate command count (rough heuristic)
    local estimated_commands=0
    case "$primary_language" in
        shell)
            # Count function definitions and case patterns
            estimated_commands=$(find "$cli_dir" -name "*.sh" -exec grep -c "^[a-zA-Z_][a-zA-Z0-9_]*\(\)" {} \; 2>/dev/null | awk '{sum+=$1} END {print sum+0}')
            ;;
        go)
            # Count main and command functions
            estimated_commands=$(find "$cli_dir" -name "*.go" -exec grep -c "func.*Command\|func main" {} \; 2>/dev/null | awk '{sum+=$1} END {print sum+0}')
            ;;
        javascript|python)
            # Count exported functions
            estimated_commands=$(find "$cli_dir" \( -name "*.js" -o -name "*.py" \) -exec grep -c "^def \|^function \|^export" {} \; 2>/dev/null | awk '{sum+=$1} END {print sum+0}')
            ;;
    esac
    
    # Build content
    local content="CLI: $scenario_name | Language: $primary_language | Files: $total_files"
    content="$content | Scripts: $script_files | Commands: ~$estimated_commands"
    [[ "$has_main_entry" == "true" ]] && content="$content | Entry: yes"
    [[ "$has_commands" == "true" ]] && content="$content | CommandDir: yes"
    [[ "$has_help" == "true" ]] && content="$content | Help: yes"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg scenario "$scenario_name" \
        --arg source_dir "$cli_dir" \
        --arg primary_language "$primary_language" \
        --arg total_files "$total_files" \
        --arg script_files "$script_files" \
        --arg config_files "$config_files" \
        --arg estimated_commands "$estimated_commands" \
        --arg has_main_entry "$has_main_entry" \
        --arg has_commands "$has_commands" \
        --arg has_help "$has_help" \
        --arg has_config "$has_config" \
        --arg has_tests "$has_tests" \
        '{
            content: $content,
            metadata: {
                scenario: $scenario,
                source_directory: $source_dir,
                component_type: "cli",
                primary_language: $primary_language,
                total_files: ($total_files | tonumber),
                script_files: ($script_files | tonumber),
                config_files: ($config_files | tonumber),
                estimated_commands: ($estimated_commands | tonumber),
                has_main_entry: ($has_main_entry == "true"),
                has_commands: ($has_commands == "true"),
                has_help: ($has_help == "true"),
                has_config: ($has_config == "true"),
                has_tests: ($has_tests == "true"),
                content_type: "scenario_cli",
                extraction_method: "cli_overview_analyzer"
            }
        }' | jq -c
}

#######################################
# Extract CLI implementation details
# 
# Uses appropriate language handler to extract code details
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines with CLI implementation details
#######################################
qdrant::extract::scenario_cli_implementation() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    local cli_dir="$dir/cli"
    
    if [[ ! -d "$cli_dir" ]]; then
        return 1
    fi
    
    log::debug "Extracting CLI implementation for $scenario_name" >&2
    
    # Use unified code extractor with auto strategy (handles single and multi-language)
    qdrant::lib::extract_code "$cli_dir" "cli" "$scenario_name" "auto"
}

#######################################
# Extract CLI usage and help information
# 
# Processes help files and usage documentation
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines with CLI usage information
#######################################
qdrant::extract::scenario_cli_usage() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    local cli_dir="$dir/cli"
    
    if [[ ! -d "$cli_dir" ]]; then
        return 1
    fi
    
    # Look for usage documentation
    local help_files=()
    [[ -f "$cli_dir/README.md" ]] && help_files+=("$cli_dir/README.md")
    [[ -f "$cli_dir/USAGE.md" ]] && help_files+=("$cli_dir/USAGE.md")
    [[ -f "$cli_dir/HELP.md" ]] && help_files+=("$cli_dir/HELP.md")
    
    for help_file in "${help_files[@]}"; do
        local filename=$(basename "$help_file")
        
        # Extract first few paragraphs as usage summary
        local usage_summary=$(head -10 "$help_file" 2>/dev/null | grep -v "^#" | grep -v "^$" | head -3 | tr '\n' ' ' || echo "")
        
        # Look for command examples
        local examples=$(grep -E "^\s*\$|^\s*#.*\$" "$help_file" 2>/dev/null | head -5 | tr '\n' '; ' || echo "")
        
        local content="Usage: $filename | CLI: $scenario_name"
        [[ -n "$usage_summary" ]] && content="$content | Summary: $usage_summary"
        [[ -n "$examples" ]] && content="$content | Examples: $examples"
        
        jq -n \
            --arg content "$content" \
            --arg scenario "$scenario_name" \
            --arg source_file "$help_file" \
            --arg filename "$filename" \
            --arg usage_summary "$usage_summary" \
            --arg examples "$examples" \
            '{
                content: $content,
                metadata: {
                    scenario: $scenario,
                    source_file: $source_file,
                    component_type: "cli",
                    filename: $filename,
                    usage_summary: $usage_summary,
                    examples: $examples,
                    content_type: "scenario_cli",
                    extraction_method: "cli_usage_parser"
                }
            }' | jq -c
    done
}

#######################################
# Extract all CLI information
# 
# Main function that calls all CLI extractors
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines for all CLI components
#######################################
qdrant::extract::scenario_cli_all() {
    local dir="$1"
    
    if [[ ! -d "$dir/cli" ]]; then
        return 1
    fi
    
    # Extract CLI overview
    qdrant::extract::scenario_cli_overview "$dir" 2>/dev/null || true
    
    # Extract implementation details
    qdrant::extract::scenario_cli_implementation "$dir" 2>/dev/null || true
    
    # Extract usage information
    qdrant::extract::scenario_cli_usage "$dir" 2>/dev/null || true
}

# Export functions for use by main.sh
export -f qdrant::extract::detect_cli_language
export -f qdrant::extract::scenario_cli_overview
export -f qdrant::extract::scenario_cli_implementation
export -f qdrant::extract::scenario_cli_usage
export -f qdrant::extract::scenario_cli_all