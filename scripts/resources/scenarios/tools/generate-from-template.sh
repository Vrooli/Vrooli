#!/usr/bin/env bash
set -euo pipefail

# Generate Scenario from Template
# This tool processes a template with variable substitution and outputs a ready-to-use scenario

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SCENARIOS_DIR="${SCRIPT_DIR}/.."
RESOURCES_DIR="${SCENARIOS_DIR}/.."

# Source common utilities
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"

# Default values
TEMPLATE=""
OUTPUT=""
VARS_FILE=""
VERBOSE=false

#######################################
# Display usage information
#######################################
usage() {
    cat << EOF
Generate Scenario from Template

USAGE:
    $0 --template TEMPLATE_NAME --output OUTPUT_FILE [OPTIONS]

DESCRIPTION:
    Processes a scenario template with variable substitution to create
    a ready-to-deploy scenario configuration.

OPTIONS:
    --template, -t NAME     Template name (e.g., ecommerce, real-estate)
    --output, -o FILE       Output file path for generated scenario
    --vars, -v FILE         Variables file for substitution (.env format)
    --list, -l              List available templates
    --verbose               Enable verbose output
    --help, -h              Show this help message

EXAMPLES:
    # Generate from template with variables
    $0 --template ecommerce --output my-store.json --vars prod.env
    
    # List available templates
    $0 --list
    
    # Generate with environment variables
    export STRIPE_KEY=sk_test_xxx
    $0 --template ecommerce --output store.json

VARIABLES:
    Variables can be provided via:
    1. Variables file (--vars flag)
    2. Environment variables
    3. Interactive prompts (for required variables)
    
    Variable format in templates: \${VARIABLE_NAME}

EOF
}

#######################################
# List available templates
#######################################
list_templates() {
    log::header "Available Templates"
    
    local templates_dir="${SCENARIOS_DIR}/templates"
    
    if [[ ! -d "$templates_dir" ]]; then
        log::error "Templates directory not found: $templates_dir"
        return 1
    fi
    
    for template_dir in "$templates_dir"/*; do
        if [[ -d "$template_dir" ]] && [[ -f "$template_dir/scenario.json" ]]; then
            local template_name=$(basename "$template_dir")
            local scenario_file="$template_dir/scenario.json"
            
            # Extract description from scenario.json
            local description=""
            if command -v jq >/dev/null 2>&1; then
                description=$(jq -r '.description // ""' "$scenario_file" 2>/dev/null)
            fi
            
            log::info "📦 $template_name"
            if [[ -n "$description" ]]; then
                echo "   $description"
            fi
            
            # Check for README
            if [[ -f "$template_dir/README.md" ]]; then
                echo "   📚 Documentation available"
            fi
            
            echo ""
        fi
    done
}

#######################################
# Load variables from file
# Arguments:
#   $1 - variables file path
#######################################
load_variables() {
    local vars_file="$1"
    
    if [[ ! -f "$vars_file" ]]; then
        log::error "Variables file not found: $vars_file"
        return 1
    fi
    
    log::info "Loading variables from: $vars_file"
    
    # Export variables from file
    set -a
    # shellcheck disable=SC1090
    source "$vars_file"
    set +a
    
    if [[ "$VERBOSE" == "true" ]]; then
        log::debug "Loaded $(grep -c '^[^#].*=' "$vars_file" 2>/dev/null || echo 0) variables"
    fi
}

#######################################
# Extract required variables from template
# Arguments:
#   $1 - template content
# Returns:
#   List of variable names
#######################################
extract_variables() {
    local content="$1"
    
    # Find all ${VARIABLE_NAME} patterns
    echo "$content" | grep -oE '\$\{[A-Z_][A-Z0-9_]*\}' | \
        sed 's/\${//g' | sed 's/}//g' | sort -u
}

#######################################
# Check for missing variables
# Arguments:
#   $1 - template content
# Returns:
#   0 if all variables are set, 1 if any are missing
#######################################
check_missing_variables() {
    local content="$1"
    local missing=()
    
    local variables
    variables=$(extract_variables "$content")
    
    for var in $variables; do
        if [[ -z "${!var:-}" ]]; then
            missing+=("$var")
        fi
    done
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log::warn "Missing variables: ${missing[*]}"
        return 1
    fi
    
    return 0
}

#######################################
# Prompt for missing variables
# Arguments:
#   $1 - template content
#######################################
prompt_for_variables() {
    local content="$1"
    local variables
    variables=$(extract_variables "$content")
    
    log::info "Checking required variables..."
    
    for var in $variables; do
        if [[ -z "${!var:-}" ]]; then
            read -p "Enter value for $var: " value
            export "$var=$value"
        else
            if [[ "$VERBOSE" == "true" ]]; then
                log::debug "$var is set"
            fi
        fi
    done
}

#######################################
# Substitute variables in content
# Arguments:
#   $1 - content with variables
# Returns:
#   Content with substituted values
#######################################
substitute_variables() {
    local content="$1"
    local result="$content"
    
    # Get all variables
    local variables
    variables=$(extract_variables "$content")
    
    for var in $variables; do
        local value="${!var:-}"
        if [[ -n "$value" ]]; then
            # Escape special characters for sed
            local escaped_value
            escaped_value=$(printf '%s\n' "$value" | sed 's/[[\.*^$()+?{|]/\\&/g')
            result=$(echo "$result" | sed "s/\${$var}/$escaped_value/g")
        fi
    done
    
    echo "$result"
}

#######################################
# Process template assets
# Arguments:
#   $1 - template directory
#   $2 - output directory
#######################################
process_assets() {
    local template_dir="$1"
    local output_dir="$2"
    local assets_dir="$template_dir/assets"
    
    if [[ ! -d "$assets_dir" ]]; then
        if [[ "$VERBOSE" == "true" ]]; then
            log::debug "No assets directory found"
        fi
        return 0
    fi
    
    log::info "Processing template assets..."
    
    # Create output assets directory
    mkdir -p "$output_dir/assets"
    
    # Process each asset file
    find "$assets_dir" -type f | while read -r asset_file; do
        local relative_path="${asset_file#$assets_dir/}"
        local output_file="$output_dir/assets/$relative_path"
        
        # Create directory structure
        mkdir -p "$(dirname "$output_file")"
        
        # Check if file needs variable substitution
        if [[ "$asset_file" =~ \.(sql|json|yaml|yml|txt|md|sh)$ ]]; then
            if [[ "$VERBOSE" == "true" ]]; then
                log::debug "Processing: $relative_path"
            fi
            
            # Read, substitute, and write
            local content
            content=$(cat "$asset_file")
            
            if echo "$content" | grep -q '\${'; then
                content=$(substitute_variables "$content")
            fi
            
            echo "$content" > "$output_file"
        else
            # Copy binary files as-is
            if [[ "$VERBOSE" == "true" ]]; then
                log::debug "Copying: $relative_path"
            fi
            cp "$asset_file" "$output_file"
        fi
    done
    
    log::success "Assets processed successfully"
}

#######################################
# Generate scenario from template
# Arguments:
#   $1 - template name
#   $2 - output file
#######################################
generate_scenario() {
    local template_name="$1"
    local output_file="$2"
    
    local template_dir="${SCENARIOS_DIR}/templates/${template_name}"
    local scenario_file="${template_dir}/scenario.json"
    
    # Validate template exists
    if [[ ! -d "$template_dir" ]]; then
        log::error "Template not found: $template_name"
        log::info "Use --list to see available templates"
        return 1
    fi
    
    if [[ ! -f "$scenario_file" ]]; then
        log::error "Template scenario.json not found: $scenario_file"
        return 1
    fi
    
    log::header "Generating Scenario from Template: $template_name"
    
    # Read template content
    local template_content
    template_content=$(cat "$scenario_file")
    
    # Load variables if provided
    if [[ -n "$VARS_FILE" ]]; then
        load_variables "$VARS_FILE"
    fi
    
    # Check and prompt for missing variables
    if ! check_missing_variables "$template_content"; then
        log::info "Some variables are not set. You can:"
        log::info "  1. Set them as environment variables"
        log::info "  2. Provide them in a variables file (--vars)"
        log::info "  3. Enter them interactively now"
        echo ""
        
        read -p "Enter missing variables interactively? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            prompt_for_variables "$template_content"
        else
            log::warn "Proceeding with missing variables (they will remain as placeholders)"
        fi
    fi
    
    # Substitute variables
    log::info "Substituting variables..."
    local processed_content
    processed_content=$(substitute_variables "$template_content")
    
    # Validate JSON
    if ! echo "$processed_content" | jq . >/dev/null 2>&1; then
        log::error "Generated scenario is not valid JSON"
        log::info "This might be due to unescaped values in variables"
        return 1
    fi
    
    # Create output directory if needed
    local output_dir
    output_dir=$(dirname "$output_file")
    mkdir -p "$output_dir"
    
    # Write processed scenario
    echo "$processed_content" | jq . > "$output_file"
    log::success "Scenario generated: $output_file"
    
    # Process assets if output is a directory scenario
    if [[ "$output_dir" != "." ]] && [[ "$output_dir" != "/" ]]; then
        process_assets "$template_dir" "$output_dir"
    fi
    
    # Show summary
    echo ""
    log::header "Generation Summary"
    log::info "Template: $template_name"
    log::info "Output: $output_file"
    
    if command -v jq >/dev/null 2>&1; then
        local resource_count
        resource_count=$(echo "$processed_content" | jq '.resources | length')
        log::info "Resources configured: $resource_count"
    fi
    
    echo ""
    log::info "Next steps:"
    log::info "  1. Review the generated scenario: cat $output_file"
    log::info "  2. Copy to Vrooli config: cp $output_file ~/.vrooli/scenarios.json"
    log::info "  3. Deploy resources: ${RESOURCES_DIR}/index.sh --action inject-all"
}

#######################################
# Parse command line arguments
#######################################
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --template|-t)
                TEMPLATE="$2"
                shift 2
                ;;
            --output|-o)
                OUTPUT="$2"
                shift 2
                ;;
            --vars|-v)
                VARS_FILE="$2"
                shift 2
                ;;
            --list|-l)
                list_templates
                exit 0
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            *)
                log::error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

#######################################
# Main execution
#######################################
main() {
    parse_arguments "$@"
    
    # Validate required arguments
    if [[ -z "$TEMPLATE" ]] || [[ -z "$OUTPUT" ]]; then
        log::error "Template and output file are required"
        usage
        exit 1
    fi
    
    # Generate scenario
    if ! generate_scenario "$TEMPLATE" "$OUTPUT"; then
        log::error "Failed to generate scenario"
        exit 1
    fi
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi