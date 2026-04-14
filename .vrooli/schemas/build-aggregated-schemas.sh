#!/bin/bash
# Build aggregated scenario dependency schemas from resource manifests.

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

validate_manifest() {
    local manifest_file="$1"

    if ! jq empty "$manifest_file" >/dev/null 2>&1; then
        return 1
    fi

    local name driver portability
    name=$(jq -r '.name // empty' "$manifest_file")
    driver=$(jq -r '.driver // empty' "$manifest_file")
    portability=$(jq -r '.portability_tier // empty' "$manifest_file")
    [[ -n "$name" && -n "$driver" && -n "$portability" ]]
}

# Main function to build aggregated schema
build_aggregated_schema() {
    log "Building aggregated resource dependency schemas from resource manifests..."

    local valid_resources=()
    local resource_count=0

    for resource_dir in "$RESOURCES_DIR"/*; do
        [[ -d "$resource_dir" ]] || continue

        local resource_name
        resource_name=$(get_resource_name "$resource_dir")
        local manifest_file="$resource_dir/resource.json"

        if [[ ! -f "$manifest_file" ]]; then
            log "⚠️  Manifest not found for resource: $resource_name"
            continue
        fi

        if ! validate_manifest "$manifest_file"; then
            log "❌ Invalid manifest for resource: $resource_name"
            continue
        fi

        log "✓ Processing manifest for: $resource_name"
        valid_resources+=("$resource_name")
        ((resource_count++))
    done

    if [[ ${#valid_resources[@]} -eq 0 ]]; then
        log "❌ No valid resource schemas found"
        return 1
    fi

    local list_file
    list_file=$(mktemp)
    printf "%s\n" "${valid_resources[@]}" > "$list_file"

    if ! RESOURCES_DIR="$RESOURCES_DIR" OUTPUT_FILE="$OUTPUT_FILE" VALID_RESOURCES_FILE="$list_file" python3 - <<'PY'
import json
import os
from pathlib import Path

resources_dir = Path(os.environ['RESOURCES_DIR'])
output_file = Path(os.environ['OUTPUT_FILE'])
list_file = Path(os.environ['VALID_RESOURCES_FILE'])

with list_file.open() as fh:
    resource_names = [line.strip() for line in fh if line.strip()]

definitions = {}
for name in resource_names:
    manifest_path = resources_dir / name / 'resource.json'
    with manifest_path.open() as manifest_file:
        manifest_data = json.load(manifest_file)

    schema_data = manifest_data.get("dependency_schema") or {}
    definition = {"type": "object"}
    if isinstance(schema_data, dict):
        definition.update(schema_data)
    if "type" not in definition:
        definition["type"] = "object"
    if "properties" not in definition:
        definition["properties"] = {}
    if "additionalProperties" not in definition:
        definition["additionalProperties"] = True
    if not definition.get("title"):
        definition["title"] = manifest_data.get("display_name") or manifest_data.get("name") or name
    if not definition.get("description"):
        definition["description"] = manifest_data.get("description", "")
    definition["resourceName"] = name
    definitions[name] = definition

catalog_properties = {}
for name in resource_names:
    catalog_properties[name] = {
        "allOf": [
            {"$ref": "../resources.schema.json#/definitions/resourceConfig"},
            {"$ref": f"#/definitions/resourceSchemas/{name}"}
        ]
    }

document = {
    "$schema": "http://json-schema.org/draft-07/schema#",
    "$id": "https://vrooli.com/schemas/resource-definitions.json",
    "title": "Resource Definitions for IDE Support",
    "description": "Auto-generated aggregated dependency schemas for all available resources",
    "definitions": {
        "resourceSchemas": definitions
    },
    "resourceCatalog": {
        "type": "object",
        "properties": catalog_properties
    }
}

output_file.write_text(json.dumps(document, indent=2) + "\n")
PY
    then
        rm -f "$list_file"
        log "❌ Failed to build resource definitions"
        return 1
    fi

    rm -f "$list_file"

    log "✅ Aggregated schema built successfully: $OUTPUT_FILE"
    log "📊 Processed $resource_count resource schemas"

    return 0
}

# Function to build service.json schema with resource references
build_service_schema() {
    local service_schema_file="$SCRIPT_DIR/service.schema.json"

    log "Validating canonical service.json schema..."

    if [[ ! -f "$service_schema_file" ]]; then
        log "❌ Canonical service schema missing: $service_schema_file"
        exit 1
    fi

    if ! jq empty "$service_schema_file" >/dev/null 2>&1; then
        log "❌ Canonical service schema is not valid JSON"
        exit 1
    fi

    log "✅ Using existing service schema (not regenerated)"
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
        local manifest_file="$resource_dir/resource.json"

        if [[ ! -f "$manifest_file" ]] || ! validate_manifest "$manifest_file"; then
            continue
        fi
        
        if [[ "$first" == true ]]; then
            first=false
        else
            echo "," >> "$catalog_file"
        fi
        
        # Extract resource metadata from the manifest-native dependency schema.
        jq --arg name "$resource_name" '{
            name: $name,
            title: (.display_name // .name // $name),
            description: (.description // ""),
            template: (.template // ""),
            driver: (.driver // ""),
            examples: ((.dependency_schema.examples // []) | length)
        }' "$manifest_file" >> "$catalog_file"
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
        total_resources=$( (find "$RESOURCES_DIR" -path '*/resource.json' 2>/dev/null || true) | wc -l )
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
        echo "   - IDEs can now use these manifest-derived schemas for auto-completion"
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
