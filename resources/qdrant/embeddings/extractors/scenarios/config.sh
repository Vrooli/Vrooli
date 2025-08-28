#!/usr/bin/env bash
# Scenario Configuration Extractor
# Extracts metadata from service.json and .vrooli/ directory contents
#
# Handles scenario configuration including:
# - service.json (name, description, tags, etc.)
# - .vrooli/ directory metadata files
# - Any YAML configuration files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract service.json configuration
# 
# Parses service.json for scenario metadata and configuration
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON line with service configuration
#######################################
qdrant::extract::scenario_service_config() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    
    local service_file=""
    [[ -f "$dir/service.json" ]] && service_file="$dir/service.json"
    [[ -f "$dir/.vrooli/service.json" ]] && service_file="${service_file:-$dir/.vrooli/service.json}"
    
    if [[ -z "$service_file" ]]; then
        return 1
    fi
    
    log::debug "Extracting service config for $scenario_name" >&2
    
    # Extract core metadata
    local name=$(jq -r '.name // empty' "$service_file" 2>/dev/null)
    local description=$(jq -r '.description // empty' "$service_file" 2>/dev/null)
    local version=$(jq -r '.version // empty' "$service_file" 2>/dev/null)
    local category=$(jq -r '.category // empty' "$service_file" 2>/dev/null)
    local status=$(jq -r '.status // empty' "$service_file" 2>/dev/null)
    
    # Extract tags and resources
    local tags_json=$(jq -c '.tags // []' "$service_file" 2>/dev/null || echo "[]")
    local resources_json=$(jq -c '.resources // []' "$service_file" 2>/dev/null || echo "[]")
    local dependencies_json=$(jq -c '.dependencies // []' "$service_file" 2>/dev/null || echo "[]")
    
    # Count items
    local tag_count=$(echo "$tags_json" | jq 'length')
    local resource_count=$(echo "$resources_json" | jq 'length')
    local dependency_count=$(echo "$dependencies_json" | jq 'length')
    
    # Extract technical configuration if present
    local tech_stack_json=$(jq -c '.technical_stack // {}' "$service_file" 2>/dev/null || echo "{}")
    local ports_json=$(jq -c '.ports // []' "$service_file" 2>/dev/null || echo "[]")
    
    # Extract business information
    local value_prop=$(jq -r '.value_proposition // empty' "$service_file" 2>/dev/null)
    local target_users_json=$(jq -c '.target_users // []' "$service_file" 2>/dev/null || echo "[]")
    local revenue_model=$(jq -r '.revenue_model.type // empty' "$service_file" 2>/dev/null)
    
    # Build enriched content
    local content="Service: $name | Scenario: $scenario_name | Version: $version"
    [[ -n "$description" ]] && content="$content | Description: $description"
    [[ -n "$category" ]] && content="$content | Category: $category"
    [[ -n "$status" ]] && content="$content | Status: $status"
    
    if [[ $tag_count -gt 0 ]]; then
        content="$content | Tags: $(echo "$tags_json" | jq -r 'join(", ")')"
    fi
    
    if [[ $resource_count -gt 0 ]]; then
        content="$content | Resources: $(echo "$resources_json" | jq -r 'join(", ")')"
    fi
    
    [[ -n "$value_prop" ]] && content="$content | Value: $value_prop"
    [[ -n "$revenue_model" ]] && content="$content | Revenue: $revenue_model"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg scenario "$scenario_name" \
        --arg source_file "$service_file" \
        --arg name "$name" \
        --arg description "$description" \
        --arg version "$version" \
        --arg category "$category" \
        --arg status "$status" \
        --arg value_proposition "$value_prop" \
        --arg revenue_model "$revenue_model" \
        --argjson tags "$tags_json" \
        --argjson resources "$resources_json" \
        --argjson dependencies "$dependencies_json" \
        --argjson target_users "$target_users_json" \
        --argjson technical_stack "$tech_stack_json" \
        --argjson ports "$ports_json" \
        --arg tag_count "$tag_count" \
        --arg resource_count "$resource_count" \
        --arg dependency_count "$dependency_count" \
        '{
            content: $content,
            metadata: {
                scenario: $scenario,
                source_file: $source_file,
                component_type: "configuration",
                config_type: "service",
                name: $name,
                description: $description,
                version: $version,
                category: $category,
                status: $status,
                value_proposition: $value_proposition,
                revenue_model: $revenue_model,
                tags: $tags,
                resources: $resources,
                dependencies: $dependencies,
                target_users: $target_users,
                technical_stack: $technical_stack,
                ports: $ports,
                tag_count: ($tag_count | tonumber),
                resource_count: ($resource_count | tonumber),
                dependency_count: ($dependency_count | tonumber),
                content_type: "scenario_configuration",
                extraction_method: "service_json_parser"
            }
        }' | jq -c
}

#######################################
# Extract .vrooli directory metadata
# 
# Processes various metadata files in .vrooli/ directory
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines with .vrooli metadata
#######################################
qdrant::extract::scenario_vrooli_metadata() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    
    local vrooli_dir="$dir/.vrooli"
    if [[ ! -d "$vrooli_dir" ]]; then
        return 1
    fi
    
    log::debug "Extracting .vrooli metadata for $scenario_name" >&2
    
    # Find all JSON and YAML files in .vrooli directory
    while IFS= read -r file; do
        local filename=$(basename "$file")
        local file_type="${filename##*.}"
        
        # Skip service.json as it's handled separately
        [[ "$filename" == "service.json" ]] && continue
        
        local metadata_content=""
        local metadata_json="{}"
        
        case "$file_type" in
            json)
                metadata_json=$(cat "$file" 2>/dev/null || echo "{}")
                ;;
            yaml|yml)
                if command -v yq &>/dev/null; then
                    metadata_json=$(yq eval -o=json '.' "$file" 2>/dev/null || echo "{}")
                else
                    # Basic YAML extraction for key fields
                    local keys=$(grep "^[a-zA-Z]" "$file" | cut -d: -f1 | head -10 | jq -R . | jq -s .)
                    metadata_json=$(jq -n --argjson keys "$keys" '{extracted_keys: $keys}')
                fi
                ;;
            *)
                # Plain text files
                metadata_content=$(head -5 "$file" 2>/dev/null | tr '\n' ' ' || echo "")
                ;;
        esac
        
        # Build content
        local content="Metadata: $filename | Scenario: $scenario_name"
        [[ -n "$metadata_content" ]] && content="$content | Content: $metadata_content"
        
        # Output as JSON line
        jq -n \
            --arg content "$content" \
            --arg scenario "$scenario_name" \
            --arg source_file "$file" \
            --arg filename "$filename" \
            --arg file_type "$file_type" \
            --arg metadata_content "$metadata_content" \
            --argjson metadata "$metadata_json" \
            '{
                content: $content,
                metadata: {
                    scenario: $scenario,
                    source_file: $source_file,
                    component_type: "configuration",
                    config_type: "vrooli_metadata",
                    filename: $filename,
                    file_type: $file_type,
                    metadata_content: $metadata_content,
                    parsed_metadata: $metadata,
                    content_type: "scenario_configuration",
                    extraction_method: "vrooli_metadata_parser"
                }
            }' | jq -c
    done < <(find "$vrooli_dir" -type f 2>/dev/null | sort)
}

#######################################
# Extract scenario test configuration
# 
# Processes scenario-test.yaml files that define test scenarios
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON line with test configuration
#######################################
qdrant::extract::scenario_test_config() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    
    local test_file=""
    [[ -f "$dir/scenario-test.yaml" ]] && test_file="$dir/scenario-test.yaml"
    [[ -f "$dir/scenario-test.yml" ]] && test_file="${test_file:-$dir/scenario-test.yml}"
    
    if [[ -z "$test_file" ]]; then
        return 1
    fi
    
    log::debug "Extracting test config for $scenario_name" >&2
    
    local test_config_json="{}"
    if command -v yq &>/dev/null; then
        test_config_json=$(yq eval -o=json '.' "$test_file" 2>/dev/null || echo "{}")
    else
        # Basic extraction
        local test_name=$(grep "^name:" "$test_file" | cut -d: -f2- | tr -d ' "' || echo "")
        local test_count=$(grep -c "^  - " "$test_file" 2>/dev/null || echo "0")
        test_config_json=$(jq -n --arg name "$test_name" --arg count "$test_count" '{name: $name, test_count: ($count | tonumber)}')
    fi
    
    # Extract test information
    local test_name=$(echo "$test_config_json" | jq -r '.name // empty')
    local test_count=$(echo "$test_config_json" | jq '.tests | length' 2>/dev/null || echo "0")
    
    # Build content
    local content="Test Config: $test_name | Scenario: $scenario_name | Tests: $test_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg scenario "$scenario_name" \
        --arg source_file "$test_file" \
        --arg test_name "$test_name" \
        --arg test_count "$test_count" \
        --argjson config "$test_config_json" \
        '{
            content: $content,
            metadata: {
                scenario: $scenario,
                source_file: $source_file,
                component_type: "configuration",
                config_type: "test_config",
                test_name: $test_name,
                test_count: ($test_count | tonumber),
                test_configuration: $config,
                content_type: "scenario_configuration",
                extraction_method: "test_config_parser"
            }
        }' | jq -c
}

#######################################
# Extract all configuration from scenario
# 
# Main function that calls all configuration extractors
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines for all configuration components
#######################################
qdrant::extract::scenario_config_all() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        return 1
    fi
    
    # Extract service configuration
    qdrant::extract::scenario_service_config "$dir" 2>/dev/null || true
    
    # Extract .vrooli metadata
    qdrant::extract::scenario_vrooli_metadata "$dir" 2>/dev/null || true
    
    # Extract test configuration
    qdrant::extract::scenario_test_config "$dir" 2>/dev/null || true
}

# Export functions for use by main.sh
export -f qdrant::extract::scenario_service_config
export -f qdrant::extract::scenario_vrooli_metadata
export -f qdrant::extract::scenario_test_config
export -f qdrant::extract::scenario_config_all