#!/usr/bin/env bash

################################################################################
# Service JSON Processor
# 
# Centralized JSON processing for Vrooli service.json files with proper
# error handling, validation, and resource extraction.
#
# This module fixes critical bugs in JSON path resolution and provides
# clean, testable functions for working with service configurations.
#
# SCHEMA REFERENCE:
#   All service.json files follow: .vrooli/schemas/service.schema.json
#   
#   Key structure:
#   - .service.{name,version,displayName}    (service metadata)
#   - .resources.{category}.{resource}       (nested resource definitions)
#   - .resources.{category}.{resource}.initialization (setup data)
#
################################################################################

set -euo pipefail

# Module metadata - prevent multiple sourcing
if [[ -z "${SJP_VERSION:-}" ]]; then
    readonly SJP_VERSION="1.0.0"
    readonly SJP_MODULE="service-json-processor"
fi

################################################################################
# Core JSON Processing Functions
################################################################################

# Validate that JSON content is syntactically correct
# Usage: sjp_validate_json_syntax <json_content>
# Returns: 0 if valid, 1 if invalid
sjp_validate_json_syntax() {
    local json_content="$1"
    
    if [[ -z "$json_content" ]]; then
        echo "ERROR: Empty JSON content provided" >&2
        return 1
    fi
    
    if ! echo "$json_content" | jq empty 2>/dev/null; then
        echo "ERROR: Invalid JSON syntax" >&2
        return 1
    fi
    
    return 0
}

# Validate that a file contains valid JSON
# Usage: sjp_validate_json_file <file_path>
# Returns: 0 if valid, 1 if invalid
sjp_validate_json_file() {
    local file_path="$1"
    
    if [[ ! -f "$file_path" ]]; then
        echo "ERROR: File not found: $file_path" >&2
        return 1
    fi
    
    if ! jq empty "$file_path" 2>/dev/null; then
        echo "ERROR: Invalid JSON in file: $file_path" >&2
        return 1
    fi
    
    return 0
}

# Extract service metadata from service.json
# Usage: sjp_get_service_info <json_content> <field>
# Fields: name, version, displayName, description, type
sjp_get_service_info() {
    local json_content="$1"
    local field="$2"
    
    if ! sjp_validate_json_syntax "$json_content"; then
        return 1
    fi
    
    local value
    case "$field" in
        name|version|displayName|description|type)
            value=$(echo "$json_content" | jq -r ".service.$field // empty")
            ;;
        *)
            echo "ERROR: Unknown service field: $field" >&2
            return 1
            ;;
    esac
    
    if [[ "$value" == "null" || -z "$value" ]]; then
        echo "ERROR: Service field '$field' not found or empty" >&2
        return 1
    fi
    
    echo "$value"
    return 0
}

# Get all resource categories from service.json
# Usage: sjp_get_resource_categories <json_content>
# Returns: List of category names (storage, automation, ai, etc.)
sjp_get_resource_categories() {
    local json_content="$1"
    
    if ! sjp_validate_json_syntax "$json_content"; then
        return 1
    fi
    
    local categories
    if ! categories=$(echo "$json_content" | jq -r '
        .resources // {} | 
        keys[] // empty
    ' 2>/dev/null); then
        echo "ERROR: Failed to extract resource categories" >&2
        return 1
    fi
    
    echo "$categories"
    return 0
}

# Get all resources within a specific category
# Usage: sjp_get_resources_in_category <json_content> <category>
# Returns: List of resource names within the category
sjp_get_resources_in_category() {
    local json_content="$1"
    local category="$2"
    
    if ! sjp_validate_json_syntax "$json_content"; then
        return 1
    fi
    
    local resources
    if ! resources=$(echo "$json_content" | jq -r \
        --arg category "$category" '
        .resources[$category] // {} | 
        keys[] // empty
    ' 2>/dev/null); then
        echo "ERROR: Failed to extract resources from category: $category" >&2
        return 1
    fi
    
    echo "$resources"
    return 0
}

# Check if a resource has a specific property
# Usage: sjp_resource_has_property <json_content> <category> <resource> <property>
# Returns: 0 if property exists and is truthy, 1 otherwise
sjp_resource_has_property() {
    local json_content="$1"
    local category="$2"
    local resource="$3"
    local property="$4"
    
    if ! sjp_validate_json_syntax "$json_content"; then
        return 1
    fi
    
    local has_property
    if ! has_property=$(echo "$json_content" | jq -r \
        --arg category "$category" \
        --arg resource "$resource" \
        --arg property "$property" '
        .resources[$category][$resource][$property] // false
    ' 2>/dev/null); then
        return 1
    fi
    
    if [[ "$has_property" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

# Get resources that match a specific condition
# Usage: sjp_get_resources_by_condition <json_content> <condition>
# Example conditions: "required == true", "enabled == true", "initialization != null"
sjp_get_resources_by_condition() {
    local json_content="$1"
    local condition="$2"
    
    if ! sjp_validate_json_syntax "$json_content"; then
        return 1
    fi
    
    local matching_resources
    # Handle common conditions with proper field access syntax
    local jq_condition
    case "$condition" in
        "required == true")
            jq_condition=".required == true"
            ;;
        "enabled == true")
            jq_condition=".enabled == true"
            ;;
        "initialization != null")
            jq_condition=".initialization != null"
            ;;
        "enabled == true and required == true")
            jq_condition=".enabled == true and .required == true"
            ;;
        *)
            # For other conditions, try to auto-fix by adding dots
            jq_condition=$(echo "$condition" | sed 's/\b\([a-zA-Z_][a-zA-Z0-9_]*\)\b/.\1/g')
            ;;
    esac
    
    local jq_filter="
        .resources // {} | 
        to_entries[] as \$category |
        \$category.value | 
        to_entries[] as \$resource |
        \$resource.value |
        if ($jq_condition) then 
            \"\(\$category.key).\(\$resource.key)\"
        else 
            empty 
        end
    "
    
    if ! matching_resources=$(echo "$json_content" | jq -r "$jq_filter" 2>/dev/null); then
        echo "ERROR: Failed to filter resources by condition: $condition" >&2
        return 1
    fi
    
    echo "$matching_resources"
    return 0
}

################################################################################
# File Reference Extraction Functions
################################################################################

# Extract data files from resource initialization
# Usage: sjp_get_data_files <json_content> [category] [resource]
# If category/resource specified, only check that specific resource
sjp_get_data_files() {
    local json_content="$1"
    local target_category="${2:-}"
    local target_resource="${3:-}"
    
    if ! sjp_validate_json_syntax "$json_content"; then
        return 1
    fi
    
    local jq_filter
    if [[ -n "$target_category" && -n "$target_resource" ]]; then
        # Target specific resource
        jq_filter="
            .resources[\"$target_category\"][\"$target_resource\"].initialization.data[]? |
            select(.file != null) |
            .file
        "
    else
        # All resources
        jq_filter='
            .resources // {} | 
            to_entries[] | 
            .value | 
            to_entries[] | 
            select(.value.initialization.data // false) | 
            .value.initialization.data[] |
            select(.file != null) |
            .file
        '
    fi
    
    local data_files
    if ! data_files=$(echo "$json_content" | jq -r "$jq_filter" 2>/dev/null); then
        echo "ERROR: Failed to extract data files" >&2
        return 1
    fi
    
    echo "$data_files"
    return 0
}

# Extract workflow files from resource initialization
# Usage: sjp_get_workflow_files <json_content> [category] [resource]
sjp_get_workflow_files() {
    local json_content="$1"
    local target_category="${2:-}"
    local target_resource="${3:-}"
    
    if ! sjp_validate_json_syntax "$json_content"; then
        return 1
    fi
    
    local jq_filter
    if [[ -n "$target_category" && -n "$target_resource" ]]; then
        # Target specific resource
        jq_filter="
            .resources[\"$target_category\"][\"$target_resource\"].initialization.workflows[]? |
            select(.file != null) |
            .file
        "
    else
        # All resources
        jq_filter='
            .resources // {} | 
            to_entries[] | 
            .value | 
            to_entries[] | 
            select(.value.initialization.workflows // false) | 
            .value.initialization.workflows[] |
            select(.file != null) |
            .file
        '
    fi
    
    local workflow_files
    if ! workflow_files=$(echo "$json_content" | jq -r "$jq_filter" 2>/dev/null); then
        echo "ERROR: Failed to extract workflow files" >&2
        return 1
    fi
    
    echo "$workflow_files"
    return 0
}

# Extract script files from resource initialization
# Usage: sjp_get_script_files <json_content> [category] [resource]
sjp_get_script_files() {
    local json_content="$1"
    local target_category="${2:-}"
    local target_resource="${3:-}"
    
    if ! sjp_validate_json_syntax "$json_content"; then
        return 1
    fi
    
    local jq_filter
    if [[ -n "$target_category" && -n "$target_resource" ]]; then
        # Target specific resource
        jq_filter="
            .resources[\"$target_category\"][\"$target_resource\"].initialization.scripts[]? |
            select(.file != null) |
            .file
        "
    else
        # All resources
        jq_filter='
            .resources // {} | 
            to_entries[] | 
            .value | 
            to_entries[] | 
            select(.value.initialization.scripts // false) | 
            .value.initialization.scripts[] |
            select(.file != null) |
            .file
        '
    fi
    
    local script_files
    if ! script_files=$(echo "$json_content" | jq -r "$jq_filter" 2>/dev/null); then
        echo "ERROR: Failed to extract script files" >&2
        return 1
    fi
    
    echo "$script_files"
    return 0
}

# Extract app files from resource initialization
# Usage: sjp_get_app_files <json_content> [category] [resource]
sjp_get_app_files() {
    local json_content="$1"
    local target_category="${2:-}"
    local target_resource="${3:-}"
    
    if ! sjp_validate_json_syntax "$json_content"; then
        return 1
    fi
    
    local jq_filter
    if [[ -n "$target_category" && -n "$target_resource" ]]; then
        # Target specific resource
        jq_filter="
            .resources[\"$target_category\"][\"$target_resource\"].initialization.apps[]? |
            select(.file != null) |
            .file
        "
    else
        # All resources
        jq_filter='
            .resources // {} | 
            to_entries[] | 
            .value | 
            to_entries[] | 
            select(.value.initialization.apps // false) | 
            .value.initialization.apps[] |
            select(.file != null) |
            .file
        '
    fi
    
    local app_files
    if ! app_files=$(echo "$json_content" | jq -r "$jq_filter" 2>/dev/null); then
        echo "ERROR: Failed to extract app files" >&2
        return 1
    fi
    
    echo "$app_files"
    return 0
}

# Extract all referenced files from service.json
# Usage: sjp_get_all_referenced_files <json_content>
# Returns: All files referenced in initialization sections
sjp_get_all_referenced_files() {
    local json_content="$1"
    
    if ! sjp_validate_json_syntax "$json_content"; then
        return 1
    fi
    
    local all_files=""
    
    # Get data files
    local data_files
    if data_files=$(sjp_get_data_files "$json_content" 2>/dev/null); then
        all_files="$all_files$data_files"$'\n'
    fi
    
    # Get workflow files
    local workflow_files
    if workflow_files=$(sjp_get_workflow_files "$json_content" 2>/dev/null); then
        all_files="$all_files$workflow_files"$'\n'
    fi
    
    # Get script files
    local script_files
    if script_files=$(sjp_get_script_files "$json_content" 2>/dev/null); then
        all_files="$all_files$script_files"$'\n'
    fi
    
    # Get app files
    local app_files
    if app_files=$(sjp_get_app_files "$json_content" 2>/dev/null); then
        all_files="$all_files$app_files"$'\n'
    fi
    
    # Clean up and deduplicate
    echo "$all_files" | grep -v '^$' | sort -u
    return 0
}

################################################################################
# Resource Analysis Functions
################################################################################

# Get resource initialization summary
# Usage: sjp_get_resource_summary <json_content> <category> <resource>
# Returns: JSON object with resource details
sjp_get_resource_summary() {
    local json_content="$1"
    local category="$2"
    local resource="$3"
    
    if ! sjp_validate_json_syntax "$json_content"; then
        return 1
    fi
    
    local summary
    # Build the summary using simpler JQ operations to avoid quoting issues
    if ! summary=$(echo "$json_content" | jq \
        --arg category "$category" \
        --arg resource "$resource" '
        {
            category: $category,
            resource: $resource,
            enabled: (.resources[$category][$resource].enabled // false),
            required: (.resources[$category][$resource].required // false),
            type: (.resources[$category][$resource].type // null),
            version: (.resources[$category][$resource].version // null),
            has_initialization: ((.resources[$category][$resource].initialization // null) != null),
            initialization_types: [
                (if .resources[$category][$resource].initialization.data then "data" else empty end),
                (if .resources[$category][$resource].initialization.workflows then "workflows" else empty end),
                (if .resources[$category][$resource].initialization.scripts then "scripts" else empty end),
                (if .resources[$category][$resource].initialization.apps then "apps" else empty end)
            ]
        }' 2>/dev/null); then
        echo "ERROR: Failed to generate resource summary for $category.$resource" >&2
        return 1
    fi
    
    echo "$summary"
    return 0
}

# Check for resource conflicts (e.g., port conflicts)
# Usage: sjp_check_resource_conflicts <json_content>
# Returns: JSON array of conflicts found
sjp_check_resource_conflicts() {
    local json_content="$1"
    
    if ! sjp_validate_json_syntax "$json_content"; then
        return 1
    fi
    
    # Check for port conflicts
    local port_usage
    if ! port_usage=$(echo "$json_content" | jq -r '
        .resources // {} | 
        to_entries[] as $category |
        $category.value | 
        to_entries[] as $resource |
        if $resource.value.port then
            "\($resource.value.port):\($category.key).\($resource.key)"
        else
            empty
        end
    ' 2>/dev/null); then
        echo "ERROR: Failed to analyze port usage" >&2
        return 1
    fi
    
    # Find duplicate ports by extracting just port numbers and checking for duplicates
    local conflicts="[]"
    
    if [[ -n "$port_usage" ]]; then
        # Extract ports and find duplicates
        local ports_only
        ports_only=$(echo "$port_usage" | cut -d: -f1 | sort | uniq -c | awk '$1 > 1 {print $2}')
        
        if [[ -n "$ports_only" ]]; then
            local conflict_objects=""
            while IFS= read -r duplicate_port; do
                [[ -z "$duplicate_port" ]] && continue
                # Get all resources using this port
                local resources_using_port
                resources_using_port=$(echo "$port_usage" | grep "^${duplicate_port}:" | cut -d: -f2- | paste -sd,)
                
                conflict_objects="${conflict_objects}{\"type\": \"port_conflict\", \"port\": $duplicate_port, \"resources\": \"$resources_using_port\"}"$'\n'
            done <<< "$ports_only"
            
            if [[ -n "$conflict_objects" ]]; then
                conflicts=$(echo "$conflict_objects" | jq -s '.')
            fi
        fi
    fi
    
    echo "$conflicts"
    return 0
}

################################################################################
# Utility Functions
################################################################################

# Check if jq is available and working
# Usage: sjp_check_dependencies
# Returns: 0 if all dependencies available, 1 otherwise
sjp_check_dependencies() {
    if ! command -v jq >/dev/null 2>&1; then
        echo "ERROR: jq is required but not installed" >&2
        return 1
    fi
    
    # Test jq with simple input
    if ! echo '{"test": true}' | jq '.test' >/dev/null 2>&1; then
        echo "ERROR: jq is not functioning correctly" >&2
        return 1
    fi
    
    return 0
}

# Get module version and info
# Usage: sjp_version
sjp_version() {
    echo "Service JSON Processor version $SJP_VERSION"
}

################################################################################
# Main Function for CLI Usage
################################################################################

# Main function when script is run directly
main() {
    if [[ $# -eq 0 ]]; then
        echo "Service JSON Processor v$SJP_VERSION"
        echo ""
        echo "Usage: $0 <command> [args...]"
        echo ""
        echo "Commands:"
        echo "  check-deps                    - Check if dependencies are available"
        echo "  validate-file <file>          - Validate JSON file syntax"
        echo "  get-service-info <file> <field> - Get service field (name|version|displayName)"
        echo "  get-categories <file>         - List resource categories"
        echo "  get-resources <file> <category> - List resources in category"
        echo "  get-files <file> <type>       - Get files by type (data|workflows|scripts|apps|all)"
        echo "  check-conflicts <file>        - Check for resource conflicts"
        echo "  version                       - Show version information"
        echo ""
        return 0
    fi
    
    local command="$1"
    shift
    
    case "$command" in
        check-deps)
            sjp_check_dependencies
            ;;
        validate-file)
            [[ $# -eq 1 ]] || { echo "Usage: validate-file <file>" >&2; return 1; }
            sjp_validate_json_file "$1"
            ;;
        get-service-info)
            [[ $# -eq 2 ]] || { echo "Usage: get-service-info <file> <field>" >&2; return 1; }
            local content; content=$(cat "$1") || return 1
            sjp_get_service_info "$content" "$2"
            ;;
        get-categories)
            [[ $# -eq 1 ]] || { echo "Usage: get-categories <file>" >&2; return 1; }
            local content; content=$(cat "$1") || return 1
            sjp_get_resource_categories "$content"
            ;;
        get-resources)
            [[ $# -eq 2 ]] || { echo "Usage: get-resources <file> <category>" >&2; return 1; }
            local content; content=$(cat "$1") || return 1
            sjp_get_resources_in_category "$content" "$2"
            ;;
        get-files)
            [[ $# -eq 2 ]] || { echo "Usage: get-files <file> <type>" >&2; return 1; }
            local content; content=$(cat "$1") || return 1
            case "$2" in
                data) sjp_get_data_files "$content" ;;
                workflows) sjp_get_workflow_files "$content" ;;
                scripts) sjp_get_script_files "$content" ;;
                apps) sjp_get_app_files "$content" ;;
                all) sjp_get_all_referenced_files "$content" ;;
                *) echo "Invalid file type: $2" >&2; return 1 ;;
            esac
            ;;
        check-conflicts)
            [[ $# -eq 1 ]] || { echo "Usage: check-conflicts <file>" >&2; return 1; }
            local content; content=$(cat "$1") || return 1
            sjp_check_resource_conflicts "$content"
            ;;
        version)
            sjp_version
            ;;
        *)
            echo "Unknown command: $command" >&2
            return 1
            ;;
    esac
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi