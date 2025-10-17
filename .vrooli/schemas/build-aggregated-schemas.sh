#!/bin/bash
# Build Aggregated Resource Schemas for IDE Support
# Scans all resource schema.json files and creates combined schemas for auto-completion

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
RESOURCES_DIR="$ROOT_DIR/resources"
OUTPUT_FILE="$SCRIPT_DIR/resource-definitions.json"

# Function to log with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2
}

# Function to get resource name from directory path
get_resource_name() {
    basename "$1"
}

# Function to validate JSON schema
validate_schema() {
    local schema_file="$1"
    
    # Basic JSON validation
    if ! jq empty "$schema_file" >/dev/null 2>&1; then
        return 1
    fi
    
    # Check for required fields
    local id title
    id=$(jq -r '."$id" // empty' "$schema_file")
    title=$(jq -r '.title // empty' "$schema_file")
    
    if [[ -z "$id" || -z "$title" ]]; then
        return 1
    fi
    
    return 0
}

# Function to extract schema properties for a resource
extract_resource_schema() {
    local schema_file="$1"
    local resource_name="$2"
    
    # Create the resource schema with proper references
    jq --arg resource_name "$resource_name" '{
        type: "object",
        properties: .properties,
        examples: .examples,
        description: .description,
        title: .title,
        resourceName: $resource_name
    }' "$schema_file"
}

# Main function to build aggregated schema
build_aggregated_schema() {
    local temp_file
    temp_file=$(mktemp)
    
    log "Building aggregated resource schemas..."
    
    # Start building the JSON structure
    echo '{
        "$schema": "http://json-schema.org/draft-07/schema#",
        "$id": "https://vrooli.com/schemas/resource-definitions.json", 
        "title": "Resource Definitions for IDE Support",
        "description": "Auto-generated aggregated schemas for all available resources",
        "definitions": {
            "resourceSchemas": {' > "$temp_file"
    
    local first=true
    local resource_count=0
    
    # Scan for resource schemas
    for resource_dir in "$RESOURCES_DIR"/*; do
        [[ -d "$resource_dir" ]] || continue
        
        local resource_name
        resource_name=$(get_resource_name "$resource_dir")
        local schema_file="$resource_dir/config/schema.json"
        
        if [[ ! -f "$schema_file" ]]; then
            log "⚠️  Schema not found for resource: $resource_name"
            continue
        fi
        
        if ! validate_schema "$schema_file"; then
            log "❌ Invalid schema for resource: $resource_name"
            continue
        fi
        
        log "✓ Processing schema for: $resource_name"
        
        # Add comma separator for all but first entry
        if [[ "$first" == true ]]; then
            first=false
        else
            echo "," >> "$temp_file"
        fi
        
        # Add resource schema
        echo "                \"$resource_name\": " >> "$temp_file"
        extract_resource_schema "$schema_file" "$resource_name" >> "$temp_file"
        
        ((resource_count++))
    done
    
    # Close the JSON structure
    echo '
            }
        },
        "resourceCatalog": {
            "type": "object",
            "properties": {' >> "$temp_file"
    
    # Add individual resource definitions for property-level auto-completion
    first=true
    local valid_resources=()
    
    # First pass: collect valid resources
    for resource_dir in "$RESOURCES_DIR"/*; do
        [[ -d "$resource_dir" ]] || continue
        
        local resource_name
        resource_name=$(get_resource_name "$resource_dir")
        local schema_file="$resource_dir/config/schema.json"
        
        if [[ -f "$schema_file" ]] && validate_schema "$schema_file"; then
            valid_resources+=("$resource_name")
        fi
    done
    
    # Second pass: add them with correct comma handling
    for i in "${!valid_resources[@]}"; do
        local resource_name="${valid_resources[$i]}"
        
        # Add comma if not first entry
        if [[ $i -gt 0 ]]; then
            echo "," >> "$temp_file"
        fi
        
        # Create a reference to the resource schema
        echo "                \"$resource_name\": {
                    \"allOf\": [
                        {\"\$ref\": \"../resources.schema.json#/definitions/resourceConfig\"},
                        {\"\$ref\": \"#/definitions/resourceSchemas/$resource_name\"}
                    ]
                }" >> "$temp_file"
    done
    
    echo '
            }
        }
    }
}' >> "$temp_file"
    
    # Move temp file to final location
    mv "$temp_file" "$OUTPUT_FILE"
    
    log "✅ Aggregated schema built successfully: $OUTPUT_FILE"
    log "📊 Processed $resource_count resource schemas"
    
    return 0
}

# Function to build service.json schema with resource references
build_service_schema() {
    local service_schema_file="$SCRIPT_DIR/service.schema.json"
    
    log "Building enhanced service.json schema..."
    
    # Read the base service schema and enhance it with resource definitions
    cat > "$service_schema_file" << 'EOF'
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://vrooli.com/schemas/service.schema.json",
  "title": "Service Configuration Schema",
  "description": "Enhanced service configuration with resource auto-completion support",
  "type": "object",
  "properties": {
    "$schema": {
      "type": "string",
      "description": "Schema reference"
    },
    "version": {
      "$ref": "common.schema.json#/definitions/semver"
    },
    "service": {
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "displayName": {"type": "string"},
        "description": {"type": "string"},
        "version": {"type": "string"}
      }
    },
    "ports": {
      "type": "object",
      "additionalProperties": {
        "type": "object",
        "properties": {
          "env_var": {"type": "string"},
          "fixed": {"type": "integer"},
          "description": {"type": "string"}
        }
      }
    },
    "resources": {
      "$ref": "resource-definitions.json#/resourceCatalog",
      "description": "Resource configurations with auto-completion support"
    },
    "deployment": {
      "type": "object"
    },
    "lifecycle": {
      "type": "object"
    }
  },
  "required": ["resources"]
}
EOF
    
    log "✅ Enhanced service schema created: $service_schema_file"
}

# Function to create a resource catalog for CLI discovery
create_resource_catalog() {
    local catalog_file="$SCRIPT_DIR/resource-catalog.json"
    
    log "Creating resource catalog..."
    
    echo '{
    "resources": [' > "$catalog_file"
    
    local first=true
    
    for resource_dir in "$RESOURCES_DIR"/*; do
        [[ -d "$resource_dir" ]] || continue
        
        local resource_name
        resource_name=$(get_resource_name "$resource_dir")
        local schema_file="$resource_dir/config/schema.json"
        
        if [[ ! -f "$schema_file" ]] || ! validate_schema "$schema_file"; then
            continue
        fi
        
        if [[ "$first" == true ]]; then
            first=false
        else
            echo "," >> "$catalog_file"
        fi
        
        # Extract resource metadata
        jq --arg name "$resource_name" '{
            name: $name,
            title: .title,
            description: .description,
            examples: (.examples | length)
        }' "$schema_file" >> "$catalog_file"
    done
    
    echo '
    ]
}' >> "$catalog_file"
    
    log "✅ Resource catalog created: $catalog_file"
}

# Function to update CLI commands with schema path
update_cli_commands() {
    local discovery_script="$ROOT_DIR/cli/commands/resource-discovery.sh"
    
    if [[ -f "$discovery_script" ]]; then
        log "📝 Resource discovery script is ready to use aggregated schemas"
        log "   Schema definitions: $OUTPUT_FILE"
        log "   Resource catalog: $SCRIPT_DIR/resource-catalog.json"
    fi
}

# Function to validate the generated schemas
validate_output() {
    log "Validating generated schemas..."
    
    if [[ ! -f "$OUTPUT_FILE" ]]; then
        log "❌ Output file not created: $OUTPUT_FILE"
        return 1
    fi
    
    if ! jq empty "$OUTPUT_FILE" >/dev/null 2>&1; then
        log "❌ Generated schema is not valid JSON"
        return 1
    fi
    
    local resource_count
    resource_count=$(jq '.definitions.resourceSchemas | length' "$OUTPUT_FILE")
    
    if [[ "$resource_count" -eq 0 ]]; then
        log "⚠️  No resource schemas were processed"
        return 1
    fi
    
    log "✅ Generated schema is valid with $resource_count resource definitions"
    return 0
}

# Main execution
main() {
    log "Starting schema aggregation process..."
    
    if [[ ! -d "$RESOURCES_DIR" ]]; then
        log "❌ Resources directory not found: $RESOURCES_DIR"
        exit 1
    fi
    
    # Build the main aggregated schema
    if build_aggregated_schema; then
        log "✅ Resource definitions built successfully"
    else
        log "❌ Failed to build resource definitions"
        exit 1
    fi
    
    # Build enhanced service schema
    build_service_schema
    
    # Create resource catalog for CLI
    create_resource_catalog
    
    # Update CLI commands
    update_cli_commands
    
    # Validate output
    if validate_output; then
        log "✅ All schemas generated and validated successfully"
        
        # Show summary
        local total_resources
        total_resources=$(find "$RESOURCES_DIR" -name "schema.json" | wc -l)
        local processed_resources
        processed_resources=$(jq '.definitions.resourceSchemas | length' "$OUTPUT_FILE")
        
        echo ""
        echo "📋 Schema Aggregation Summary:"
        echo "   Total resources found: $total_resources"
        echo "   Successfully processed: $processed_resources"
        echo "   Output files:"
        echo "     - $OUTPUT_FILE"
        echo "     - $SCRIPT_DIR/service.schema.json"
        echo "     - $SCRIPT_DIR/resource-catalog.json"
        echo ""
        echo "🎯 Next steps:"
        echo "   - IDEs can now use these schemas for auto-completion"
        echo "   - Use 'vrooli resource catalog' to explore resources"
        echo "   - Reference schemas with \$schema in your service.json files"
        
    else
        log "❌ Schema validation failed"
        exit 1
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi