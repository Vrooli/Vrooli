#!/usr/bin/env bash
# Resource Configuration Extractor
# Extracts configuration from various config files in resource directories
#
# Handles multiple configuration formats:
# - capabilities.yaml (resource capabilities and features)
# - config.yaml/yml (general configuration)
# - .env files (environment variables)
# - Dockerfile (container configuration)
# - docker-compose.yml (service orchestration)

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract capabilities from capabilities.yaml
# 
# Capabilities define what a resource can do, its features,
# and integration points with other resources.
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON line with capabilities information
#######################################
qdrant::extract::resource_capabilities() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    # Check for capabilities.yaml or capabilities.yml
    local caps_file=""
    [[ -f "$dir/capabilities.yaml" ]] && caps_file="$dir/capabilities.yaml"
    [[ -f "$dir/capabilities.yml" ]] && caps_file="$dir/capabilities.yml"
    
    if [[ -z "$caps_file" ]]; then
        return 1
    fi
    
    log::debug "Extracting capabilities for $resource_name" >&2
    
    # Extract capabilities using yq if available, otherwise basic parsing
    local capabilities_json="{}"
    
    if command -v yq &>/dev/null; then
        # Use yq for proper YAML parsing
        capabilities_json=$(yq eval -o=json '.' "$caps_file" 2>/dev/null || echo "{}")
    else
        # Fallback to basic extraction
        local features=$(grep "^features:" -A 10 "$caps_file" 2>/dev/null | grep "^  - " | sed 's/^  - //' | jq -R . | jq -s . || echo "[]")
        local integrations=$(grep "^integrations:" -A 10 "$caps_file" 2>/dev/null | grep "^  - " | sed 's/^  - //' | jq -R . | jq -s . || echo "[]")
        
        capabilities_json=$(jq -n \
            --argjson features "$features" \
            --argjson integrations "$integrations" \
            '{features: $features, integrations: $integrations}')
    fi
    
    # Build content summary
    local feature_count=$(echo "$capabilities_json" | jq '.features | length' 2>/dev/null || echo "0")
    local integration_count=$(echo "$capabilities_json" | jq '.integrations | length' 2>/dev/null || echo "0")
    
    local content="Resource: $resource_name | Type: Capabilities"
    content="$content | Features: $feature_count | Integrations: $integration_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg resource "$resource_name" \
        --arg source_file "$caps_file" \
        --argjson capabilities "$capabilities_json" \
        '{
            content: $content,
            metadata: {
                resource: $resource,
                source_file: $source_file,
                component_type: "capabilities",
                capabilities: $capabilities,
                content_type: "resource_configuration",
                extraction_method: "capabilities_parser"
            }
        }' | jq -c
}

#######################################
# Extract general configuration
# 
# Processes config.yaml/yml files for resource settings
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON line with configuration information
#######################################
qdrant::extract::resource_config_yaml() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    # Check for config.yaml or config.yml
    local config_file=""
    [[ -f "$dir/config.yaml" ]] && config_file="$dir/config.yaml"
    [[ -f "$dir/config.yml" ]] && config_file="$dir/config.yml"
    
    if [[ -z "$config_file" ]]; then
        return 1
    fi
    
    log::debug "Extracting YAML config for $resource_name" >&2
    
    # Extract settings
    local settings_json="{}"
    
    if command -v yq &>/dev/null; then
        # Use yq for proper YAML parsing, exclude sensitive fields
        settings_json=$(yq eval -o=json '.' "$config_file" 2>/dev/null | \
            jq 'with_entries(select(.key | test("password|secret|key|token") | not))' || echo "{}")
    else
        # Fallback to basic extraction
        local port=$(grep "^port:" "$config_file" 2>/dev/null | cut -d: -f2 | tr -d ' ' || echo "")
        local host=$(grep "^host:" "$config_file" 2>/dev/null | cut -d: -f2 | tr -d ' ' || echo "")
        
        if [[ -n "$port" ]] || [[ -n "$host" ]]; then
            settings_json=$(jq -n --arg port "$port" --arg host "$host" '{port: $port, host: $host}')
        fi
    fi
    
    # Build content
    local content="Resource: $resource_name | Type: Configuration | Format: YAML"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg resource "$resource_name" \
        --arg source_file "$config_file" \
        --argjson settings "$settings_json" \
        '{
            content: $content,
            metadata: {
                resource: $resource,
                source_file: $source_file,
                component_type: "configuration",
                format: "yaml",
                settings: $settings,
                content_type: "resource_configuration",
                extraction_method: "yaml_parser"
            }
        }' | jq -c
}

#######################################
# Extract environment variables
# 
# Processes .env files for environment configuration
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON line with environment variables
#######################################
qdrant::extract::resource_env() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    # Check for .env files
    local env_file=""
    [[ -f "$dir/.env" ]] && env_file="$dir/.env"
    [[ -f "$dir/.env.example" ]] && env_file="${env_file:-$dir/.env.example}"
    
    if [[ -z "$env_file" ]]; then
        return 1
    fi
    
    log::debug "Extracting environment config for $resource_name" >&2
    
    # Extract variable names only (not values for security)
    local env_vars=$(grep -E "^[A-Z_][A-Z0-9_]*=" "$env_file" 2>/dev/null | cut -d= -f1)
    local env_vars_json="[]"
    
    if [[ -n "$env_vars" ]]; then
        env_vars_json=$(echo "$env_vars" | jq -R . | jq -s .)
    fi
    
    local var_count=$(echo "$env_vars" | wc -w)
    
    # Build content
    local content="Resource: $resource_name | Type: Configuration | Format: Environment"
    content="$content | Variables: $var_count"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg resource "$resource_name" \
        --arg source_file "$env_file" \
        --argjson variables "$env_vars_json" \
        --arg var_count "$var_count" \
        '{
            content: $content,
            metadata: {
                resource: $resource,
                source_file: $source_file,
                component_type: "configuration",
                format: "environment",
                environment_variables: $variables,
                variable_count: ($var_count | tonumber),
                content_type: "resource_configuration",
                extraction_method: "env_parser"
            }
        }' | jq -c
}

#######################################
# Extract Docker configuration
# 
# Processes Dockerfile and docker-compose.yml
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON lines with Docker configuration
#######################################
qdrant::extract::resource_docker() {
    local dir="$1"
    local resource_name=$(basename "$dir")
    
    # Extract Dockerfile configuration
    if [[ -f "$dir/Dockerfile" ]]; then
        log::debug "Extracting Docker config for $resource_name" >&2
        
        local dockerfile="$dir/Dockerfile"
        
        # Extract base image
        local base_image=$(grep "^FROM " "$dockerfile" | head -1 | cut -d' ' -f2)
        
        # Extract exposed ports
        local ports=$(grep "^EXPOSE " "$dockerfile" | cut -d' ' -f2-)
        local ports_json="[]"
        if [[ -n "$ports" ]]; then
            ports_json=$(echo "$ports" | tr ' ' '\n' | jq -R . | jq -s .)
        fi
        
        # Extract environment variables
        local docker_env=$(grep "^ENV " "$dockerfile" | sed 's/^ENV //' | cut -d= -f1)
        local docker_env_json="[]"
        if [[ -n "$docker_env" ]]; then
            docker_env_json=$(echo "$docker_env" | jq -R . | jq -s .)
        fi
        
        # Build content
        local content="Resource: $resource_name | Type: Configuration | Format: Docker"
        content="$content | Base: $base_image"
        
        # Output as JSON line
        jq -n \
            --arg content "$content" \
            --arg resource "$resource_name" \
            --arg source_file "$dockerfile" \
            --arg base_image "$base_image" \
            --argjson exposed_ports "$ports_json" \
            --argjson env_vars "$docker_env_json" \
            '{
                content: $content,
                metadata: {
                    resource: $resource,
                    source_file: $source_file,
                    component_type: "configuration",
                    format: "docker",
                    base_image: $base_image,
                    exposed_ports: $exposed_ports,
                    environment_variables: $env_vars,
                    content_type: "resource_configuration",
                    extraction_method: "dockerfile_parser"
                }
            }' | jq -c
    fi
    
    # Extract Docker Compose configuration
    local compose_file=""
    [[ -f "$dir/docker-compose.yml" ]] && compose_file="$dir/docker-compose.yml"
    [[ -f "$dir/docker-compose.yaml" ]] && compose_file="${compose_file:-$dir/docker-compose.yaml}"
    
    if [[ -n "$compose_file" ]]; then
        log::debug "Extracting Docker Compose config for $resource_name" >&2
        
        # Extract service names
        local services_json="[]"
        if command -v yq &>/dev/null; then
            services_json=$(yq eval '.services | keys' "$compose_file" 2>/dev/null || echo "[]")
        else
            local services=$(grep "^  [a-z]" "$compose_file" 2>/dev/null | sed 's/:$//' | tr -d ' ')
            if [[ -n "$services" ]]; then
                services_json=$(echo "$services" | jq -R . | jq -s .)
            fi
        fi
        
        local service_count=$(echo "$services_json" | jq 'length')
        
        # Build content
        local content="Resource: $resource_name | Type: Configuration | Format: Docker Compose"
        content="$content | Services: $service_count"
        
        # Output as JSON line
        jq -n \
            --arg content "$content" \
            --arg resource "$resource_name" \
            --arg source_file "$compose_file" \
            --argjson services "$services_json" \
            --arg service_count "$service_count" \
            '{
                content: $content,
                metadata: {
                    resource: $resource,
                    source_file: $source_file,
                    component_type: "configuration",
                    format: "docker-compose",
                    services: $services,
                    service_count: ($service_count | tonumber),
                    content_type: "resource_configuration",
                    extraction_method: "compose_parser"
                }
            }' | jq -c
    fi
}

#######################################
# Extract all configuration from a resource
# 
# Main function that calls all configuration extractors
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON lines for all configuration components
#######################################
qdrant::extract::resource_config_all() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        return 1
    fi
    
    # Extract capabilities
    qdrant::extract::resource_capabilities "$dir" 2>/dev/null || true
    
    # Extract YAML configuration
    qdrant::extract::resource_config_yaml "$dir" 2>/dev/null || true
    
    # Extract environment variables
    qdrant::extract::resource_env "$dir" 2>/dev/null || true
    
    # Extract Docker configuration
    qdrant::extract::resource_docker "$dir" 2>/dev/null || true
}

# Export functions for use by resources.sh
export -f qdrant::extract::resource_capabilities
export -f qdrant::extract::resource_config_yaml
export -f qdrant::extract::resource_env
export -f qdrant::extract::resource_docker
export -f qdrant::extract::resource_config_all