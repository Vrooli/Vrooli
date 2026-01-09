#!/bin/bash
# Resource Discovery Commands for Vrooli CLI
# Provides catalog, schema, and config generation for resources

set -euo pipefail

# Source required libraries
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source CLI framework
source "$ROOT_DIR/scripts/lib/cli-framework.sh" || {
    echo "ERROR: Failed to load CLI framework" >&2
    exit 1
}

# Global variables
RESOURCES_DIR="$ROOT_DIR/resources"
SCHEMAS_DIR="$ROOT_DIR/.vrooli/schemas"
JQ_RESOURCES_EXPR="$var_JQ_RESOURCES_EXPR"
[[ -z "$JQ_RESOURCES_EXPR" ]] && JQ_RESOURCES_EXPR='(.dependencies.resources // {})'

# Function to list all available resources with descriptions
resource_catalog() {
    local format="${1:-text}"
    local resources=()
    
    # Find all resources with schema.json files
    for resource_dir in "$RESOURCES_DIR"/*; do
        [[ -d "$resource_dir" ]] || continue
        
        local resource_name=$(basename "$resource_dir")
        local schema_file="$resource_dir/config/schema.json"
        
        if [[ -f "$schema_file" ]]; then
            local description=$(jq -r '.description // "No description available"' "$schema_file" 2>/dev/null || echo "No description available")
            resources+=("$resource_name|$description")
        fi
    done
    
    # Sort resources alphabetically
    IFS=$'\n' sorted=($(sort <<<"${resources[*]}")); unset IFS
    
    case "$format" in
        json)
            echo "["
            local first=true
            for resource_info in "${sorted[@]}"; do
                IFS='|' read -r name desc <<< "$resource_info"
                [[ "$first" == true ]] && first=false || echo ","
                printf '  {"name": "%s", "description": "%s"}' "$name" "$desc"
            done
            echo -e "\n]"
            ;;
        yaml)
            echo "resources:"
            for resource_info in "${sorted[@]}"; do
                IFS='|' read -r name desc <<< "$resource_info"
                echo "  - name: $name"
                echo "    description: \"$desc\""
            done
            ;;
        *)
            # Text format (default)
            echo "Available Resources:"
            echo "==================="
            for resource_info in "${sorted[@]}"; do
                IFS='|' read -r name desc <<< "$resource_info"
                printf "%-20s - %s\n" "$name" "$desc"
            done
            ;;
    esac
}

# Function to show schema for a specific resource
resource_schema() {
    local resource="${1:?Resource name required}"
    local format="${2:-json}"
    local schema_file="$RESOURCES_DIR/$resource/config/schema.json"
    
    if [[ ! -f "$schema_file" ]]; then
        echo "ERROR: Schema not found for resource: $resource" >&2
        echo "Hint: Use 'vrooli resource catalog' to see available resources" >&2
        return 1
    fi
    
    case "$format" in
        json)
            cat "$schema_file"
            ;;
        yaml)
            # Convert JSON to YAML using jq
            jq -r 'to_entries | map("\(.key): \(.value | tostring)") | .[]' "$schema_file"
            ;;
        properties)
            # Show only the properties section
            jq '.properties' "$schema_file"
            ;;
        examples)
            # Show only the examples
            jq '.examples' "$schema_file"
            ;;
        *)
            echo "ERROR: Unsupported format: $format" >&2
            echo "Supported formats: json, yaml, properties, examples" >&2
            return 1
            ;;
    esac
}

# Function to generate config snippet for a resource
resource_config() {
    local resource="${1:?Resource name required}"
    local example="${2:-minimal}"
    local output_format="${3:-json}"
    local schema_file="$RESOURCES_DIR/$resource/config/schema.json"
    
    if [[ ! -f "$schema_file" ]]; then
        echo "ERROR: Schema not found for resource: $resource" >&2
        return 1
    fi
    
    # Get examples from schema
    local examples=$(jq -r '.examples // []' "$schema_file")
    
    if [[ "$example" == "list" ]]; then
        # List available examples
        echo "Available examples for $resource:"
        echo "$examples" | jq -r '.[] | .title'
        return 0
    fi
    
    # Find the requested example
    local config
    if [[ "$example" == "minimal" ]] || [[ "$example" == "default" ]]; then
        # Use first example or create minimal config
        config=$(echo "$examples" | jq -r '.[0].config // {enabled: true}')
    else
        # Search for specific example by title
        config=$(echo "$examples" | jq -r --arg title "$example" '.[] | select(.title | ascii_downcase == ($title | ascii_downcase)) | .config')
        
        if [[ -z "$config" ]] || [[ "$config" == "null" ]]; then
            echo "ERROR: Example '$example' not found for resource: $resource" >&2
            echo "Hint: Use 'vrooli resource config $resource list' to see available examples" >&2
            return 1
        fi
    fi
    
    # Generate the config snippet
    case "$output_format" in
        json)
            echo "{"
            echo "  \"$resource\": $config"
            echo "}"
            ;;
        service)
            # Generate complete service.json snippet
            cat <<EOF
{
  "\$schema": ".vrooli/schemas/service.schema.json",
  "version": "1.0.0",
  "dependencies": {
    "resources": {
      "$resource": $config
    }
  }
}
EOF
            ;;
        flat)
            # Just the config without wrapping
            echo "$config"
            ;;
        *)
            echo "ERROR: Unsupported output format: $output_format" >&2
            return 1
            ;;
    esac
}

# Function to validate a service.json against resource schemas
resource_validate() {
    local service_file="${1:-service.json}"
    
    if [[ ! -f "$service_file" ]]; then
        echo "ERROR: File not found: $service_file" >&2
        return 1
    fi
    
    local resources=$(jq -r "${JQ_RESOURCES_EXPR} | keys[]" "$service_file" 2>/dev/null)
    
    if [[ -z "$resources" ]]; then
        echo "ERROR: No resources found in $service_file" >&2
        return 1
    fi
    
    local all_valid=true
    
    echo "Validating resources in $service_file..."
    
    for resource in $resources; do
        local resource_config=$(jq --arg resource "$resource" "${JQ_RESOURCES_EXPR} | .[$resource]" "$service_file")
        local resource_type="${resource}"
        
        # Check if explicit type is specified
        local explicit_type=$(echo "$resource_config" | jq -r '.type // empty')
        [[ -n "$explicit_type" ]] && resource_type="$explicit_type"
        
        local schema_file="$RESOURCES_DIR/$resource_type/config/schema.json"
        
        if [[ ! -f "$schema_file" ]]; then
            echo "  ❌ $resource: No schema found for resource type '$resource_type'"
            all_valid=false
            continue
        fi
        
        # Basic validation - check if enabled property exists
        local is_enabled=$(echo "$resource_config" | jq -r '.enabled // false')
        
        if [[ "$is_enabled" == "true" ]] || [[ "$is_enabled" == "false" ]]; then
            echo "  ✓ $resource: Valid configuration"
        else
            echo "  ⚠ $resource: Missing or invalid 'enabled' property"
        fi
    done
    
    if [[ "$all_valid" == true ]]; then
        echo "✅ All resources validated successfully"
        return 0
    else
        echo "⚠️ Some resources have validation issues"
        return 1
    fi
}

# Main command handler
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        catalog)
            resource_catalog "$@"
            ;;
        schema)
            resource_schema "$@"
            ;;
        config)
            resource_config "$@"
            ;;
        validate)
            resource_validate "$@"
            ;;
        help|--help|-h)
            cat <<EOF
Resource Discovery Commands

USAGE:
  vrooli resource catalog [format]           List all available resources
  vrooli resource schema <resource> [format] Show schema for a resource
  vrooli resource config <resource> [example] [format] Generate config snippet
  vrooli resource validate [service.json]    Validate resource configuration

COMMANDS:
  catalog   List all available resources with descriptions
            Formats: text (default), json, yaml
            
  schema    Show the configuration schema for a resource
            Formats: json (default), yaml, properties, examples
            
  config    Generate a configuration snippet for a resource
            Examples: minimal (default), development, production, or 'list'
            Formats: json (default), service, flat
            
  validate  Validate a service.json file against resource schemas

EXAMPLES:
  # List all available resources
  vrooli resource catalog
  
  # Show schema for postgres
  vrooli resource schema postgres
  
  # Get production config for postgres
  vrooli resource config postgres production
  
  
  # Validate service.json
  vrooli resource validate ./service.json

EOF
            ;;
        *)
            echo "ERROR: Unknown command: $command" >&2
            echo "Use 'vrooli resource help' for usage information" >&2
            return 1
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
