#!/usr/bin/env bash
# n8n Auto-Credential Management System
# Discovers resources and creates n8n credentials automatically

set -euo pipefail

# Get script directory and source dependencies
N8N_AUTO_CREDS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${N8N_AUTO_CREDS_DIR}/../../../../lib/utils/var.sh"

# Resource credential registry - maps resource types to n8n credential types
declare -A RESOURCE_CREDENTIAL_REGISTRY=(
    # Storage Resources
    ["postgres"]="postgres"
    ["redis"]="redis"
    ["minio"]="s3"
    ["vault"]="httpHeaderAuth"
    ["qdrant"]="httpHeaderAuth"
    ["questdb"]="postgres"
    
    # AI Resources  
    ["ollama"]="ollama"
    ["whisper"]="httpBasicAuth"
    ["unstructured-io"]="httpBasicAuth"
    
    # Automation Resources
    ["windmill"]="httpBasicAuth"
    ["node-red"]="httpBasicAuth"
    ["huginn"]="httpBasicAuth"
    ["comfyui"]="httpBasicAuth"
    ["n8n-original"]="httpBasicAuth"
    
    # Search Resources
    ["searxng"]="httpBasicAuth"
    
    # Execution Resources
    ["judge0"]="httpBasicAuth"
    
    # Agent Resources
    ["agent-s2"]="httpBasicAuth"
    ["claude-code"]="httpBasicAuth"
    ["browserless"]="httpBasicAuth"
)

#######################################
# Auto-discover all running resources
# Returns: JSON array of running resources with connection info
#######################################
n8n::discover_resources() {
    log::debug "Discovering running resources for credential creation..."
    
    local resources=()
    local resource_configs_dir="${var_SCRIPTS_RESOURCES_DIR}"
    
    # Scan all resource categories
    for category_dir in "${resource_configs_dir}"/*; do
        [[ ! -d "$category_dir" ]] && continue
        
        local category_name=$(basename "$category_dir")
        log::debug "Scanning category: $category_name"
        
        for resource_dir in "${category_dir}"/*; do
            [[ ! -d "$resource_dir" ]] && continue
            
            local resource_name=$(basename "$resource_dir")
            local manage_script="$resource_dir/manage.sh"
            
            # Skip if no management script
            [[ ! -f "$manage_script" ]] && continue
            
            # Skip n8n itself to avoid recursion
            [[ "$resource_name" == "n8n" ]] && continue
            
            log::debug "Checking resource: $resource_name"
            
            # Check if resource is running by examining the exit status and output
            # Look for positive status indicators, avoiding false positives from "not running" messages
            local status_output
            if status_output=$("$manage_script" --action status 2>&1) && \
               echo "$status_output" | grep -Eq "✅.*(Running|Health|Healthy)|Status:.*running|Container.*running" && \
               ! echo "$status_output" | grep -q "not running\|is not running\|stopped"; then
                local resource_info
                resource_info=$(n8n::extract_resource_info "$resource_name" "$resource_dir" "$category_name")
                
                if [[ -n "$resource_info" && "$resource_info" != "null" ]]; then
                    resources+=("$resource_info")
                    log::debug "✓ Discovered running resource: $resource_name"
                fi
            else
                log::debug "✗ Resource not running: $resource_name"
            fi
        done
    done
    
    # Output as JSON array
    if [[ ${#resources[@]} -gt 0 ]]; then
        printf '%s\n' "${resources[@]}" | jq -s '.'
    else
        echo '[]'
    fi
}

#######################################
# Extract connection info for a resource
# Args: $1 - resource name, $2 - resource directory, $3 - category
# Returns: JSON object with connection details
#######################################
n8n::extract_resource_info() {
    local resource_name="$1"
    local resource_dir="$2"
    local category="$3"
    
    # Source resource configuration to get connection details
    local defaults_file="$resource_dir/config/defaults.sh"
    [[ ! -f "$defaults_file" ]] && return
    
    # Extract configuration in subshell to avoid variable pollution
    local config_data
    config_data=$(
        # shellcheck disable=SC1090
        source "$defaults_file" 2>/dev/null || exit 1
        
        # Call export function if it exists
        if declare -f "${resource_name}::export_config" >/dev/null 2>&1; then
            "${resource_name}::export_config" 2>/dev/null || true
        fi
        
        # Extract common configuration patterns with multiple naming conventions
        local port host container_name user password database region endpoint
        
        # Try different variable naming conventions for port
        # Use safe variable expansion that doesn't create complex strings
        local resource_upper=$(echo "$resource_name" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
        eval "port=\${${resource_upper}_PORT:-}"
        [[ -z "$port" ]] && eval "port=\${${resource_upper}_DEFAULT_PORT:-}"
        
        # Host detection - always fall back to localhost if no variable is set  
        eval "host=\${${resource_upper}_HOST:-}"
        [[ -z "$host" ]] && host="localhost"
        
        # Container name patterns - default to resource name
        eval "container_name=\${${resource_upper}_CONTAINER_NAME:-}"
        [[ -z "$container_name" ]] && container_name="$resource_name"
        
        # User/password patterns - simplified to avoid complex expansions
        eval "user=\${${resource_upper}_USER:-}"
        [[ -z "$user" ]] && eval "user=\${${resource_upper}_DEFAULT_USER:-}"
        [[ -z "$user" ]] && eval "user=\${${resource_upper}_ROOT_USER:-}"
        
        eval "password=\${${resource_upper}_PASSWORD:-}"
        [[ -z "$password" ]] && eval "password=\${${resource_upper}_ROOT_PASSWORD:-}"
        
        # Database name
        eval "database=\${${resource_upper}_DATABASE:-}"
        [[ -z "$database" ]] && eval "database=\${${resource_upper}_DEFAULT_DB:-}"
        
        # Special handling for specific resources
        case "$resource_name" in
            minio)
                eval "region=\${MINIO_REGION:-us-east-1}"
                eval "endpoint=http://\${host}:\${port}"
                ;;
            vault)
                eval "endpoint=http://\${host}:\${port}"
                ;;
            ollama)
                eval "endpoint=http://\${host}:\${port}"
                ;;
        esac
        
        # Detect Docker networking adjustments
        local docker_host="$host"
        if docker ps --format "{{.Names}}" | grep -q "^n8n$" 2>/dev/null; then
            case "$host" in
                localhost|127.0.0.1)
                    # Check if target resource is also in Docker
                    if docker ps --format "{{.Names}}" | grep -q "^${container_name}$" 2>/dev/null; then
                        docker_host="$container_name"  # Container-to-container
                        log::debug "Docker networking: $resource_name -> $container_name"
                    else
                        docker_host="host.docker.internal"  # Container-to-host
                        log::debug "Docker networking: $resource_name -> host.docker.internal:$port"
                    fi
                    ;;
            esac
        fi
        
        # Skip if no port detected (likely not a network service)
        [[ -z "$port" ]] && exit 1
        
        # Get credential type from registry
        local cred_type="${RESOURCE_CREDENTIAL_REGISTRY[$resource_name]:-httpBasicAuth}"
        
        # Output as JSON with all available information
        jq -n \
            --arg name "$resource_name" \
            --arg category "$category" \
            --arg type "$cred_type" \
            --arg port "${port:-}" \
            --arg host "$docker_host" \
            --arg original_host "$host" \
            --arg user "${user:-}" \
            --arg password "${password:-}" \
            --arg database "${database:-}" \
            --arg container "$container_name" \
            --arg region "${region:-}" \
            --arg endpoint "${endpoint:-}" \
            '{
                name: $name,
                category: $category,
                credential_type: $type,
                connection: {
                    host: $host,
                    original_host: $original_host,
                    port: ($port | tonumber? // null),
                    user: $user,
                    password: $password,
                    database: $database,
                    container_name: $container,
                    region: $region,
                    endpoint: $endpoint
                }
            } | 
            # Clean up empty/null values
            .connection |= with_entries(select(.value != "" and .value != null)) |
            if .connection == {} then empty else . end'
    ) || return 1
    
    echo "$config_data"
}

#######################################
# Create credential configuration for each resource type
# Args: $1 - resource info JSON object
# Returns: credential configuration JSON
#######################################
n8n::create_credential_config() {
    local resource_info="$1"
    
    local name type connection
    name=$(echo "$resource_info" | jq -r '.name')
    type=$(echo "$resource_info" | jq -r '.credential_type')
    connection=$(echo "$resource_info" | jq -c '.connection')
    
    local credential_name="vrooli-$name"
    
    log::debug "Creating credential config for $name (type: $type)"
    
    case "$type" in
        postgres)
            local host port database user password
            host=$(echo "$connection" | jq -r '.host')
            port=$(echo "$connection" | jq -r '.port // 5432')
            database=$(echo "$connection" | jq -r '.database // "postgres"')
            user=$(echo "$connection" | jq -r '.user // "postgres"')
            password=$(echo "$connection" | jq -r '.password // ""')
            
            jq -n \
                --arg name "$credential_name" \
                --arg host "$host" \
                --arg port "$port" \
                --arg database "$database" \
                --arg user "$user" \
                --arg password "$password" \
                '{
                    name: $name,
                    type: "postgres", 
                    data: {
                        host: $host,
                        port: ($port | tonumber),
                        database: $database,
                        user: $user,
                        password: $password,
                        ssl: false,
                        allowUnauthorizedCerts: true
                    }
                }'
            ;;
        redis)
            local host port password
            host=$(echo "$connection" | jq -r '.host')
            port=$(echo "$connection" | jq -r '.port // 6379')
            password=$(echo "$connection" | jq -r '.password // ""')
            
            jq -n \
                --arg name "$credential_name" \
                --arg host "$host" \
                --arg port "$port" \
                --arg password "$password" \
                '{
                    name: $name,
                    type: "redis",
                    data: {
                        host: $host,
                        port: ($port | tonumber),
                        password: $password,
                        database: 0
                    }
                }'
            ;;
        s3)
            local host port user password region
            host=$(echo "$connection" | jq -r '.host')
            port=$(echo "$connection" | jq -r '.port // 9000')
            user=$(echo "$connection" | jq -r '.user // "minioadmin"')
            password=$(echo "$connection" | jq -r '.password // "minioadmin"')
            region=$(echo "$connection" | jq -r '.region // "us-east-1"')
            
            local endpoint="http://${host}:${port}"
            
            jq -n \
                --arg name "$credential_name" \
                --arg access_key "$user" \
                --arg secret_key "$password" \
                --arg region "$region" \
                --arg endpoint "$endpoint" \
                '{
                    name: $name,
                    type: "s3",
                    data: {
                        accessKeyId: $access_key,
                        secretAccessKey: $secret_key,
                        region: $region,
                        customEndpoint: $endpoint,
                        forcePathStyle: true,
                        ssl: false
                    }
                }'
            ;;
        ollama)
            local host port
            host=$(echo "$connection" | jq -r '.host')
            port=$(echo "$connection" | jq -r '.port // 11434')
            
            local base_url="http://${host}:${port}"
            
            jq -n \
                --arg name "$credential_name" \
                --arg base_url "$base_url" \
                '{
                    name: $name,
                    type: "ollama",
                    data: {
                        baseUrl: $base_url
                    }
                }'
            ;;
        httpBasicAuth|httpHeaderAuth)
            local host port user password
            host=$(echo "$connection" | jq -r '.host')
            port=$(echo "$connection" | jq -r '.port // 80')
            user=$(echo "$connection" | jq -r '.user // ""')
            password=$(echo "$connection" | jq -r '.password // ""')
            
            local base_url="http://${host}:${port}"
            
            local credential_data
            if [[ "$type" == "httpBasicAuth" ]]; then
                # Basic Auth credential
                credential_data=$(jq -n \
                    --arg name "$credential_name" \
                    --arg user "${user:-admin}" \
                    --arg password "${password:-password}" \
                    '{
                        name: $name,
                        type: "httpBasicAuth",
                        data: {
                            user: $user,
                            password: $password
                        }
                    }')
            elif [[ "$type" == "httpHeaderAuth" ]]; then
                # Header Auth credential  
                credential_data=$(jq -n \
                    --arg name "$credential_name" \
                    --arg header_name "Authorization" \
                    --arg header_value "Bearer ${password:-token}" \
                    '{
                        name: $name,
                        type: "httpHeaderAuth",
                        data: {
                            name: $header_name,
                            value: $header_value
                        }
                    }')
            else
                # Default to basic auth with generic credentials
                credential_data=$(jq -n \
                    --arg name "$credential_name" \
                    '{
                        name: $name,
                        type: "httpBasicAuth",
                        data: {
                            user: "admin",
                            password: "admin"
                        }
                    }')
            fi
            
            echo "$credential_data"
            ;;
        *)
            log::warn "Unknown credential type: $type for resource: $name"
            return 1
            ;;
    esac
}

#######################################
# Check if credential already exists
# Args: $1 - credential name
# Returns: 0 if exists, 1 if not
#######################################
n8n::credential_exists() {
    local credential_name="$1"
    
    local api_key
    api_key=$(n8n::resolve_api_key 2>/dev/null)
    [[ -z "$api_key" ]] && return 1
    
    local credentials_response
    credentials_response=$(curl -s \
        -H "X-N8N-API-KEY: $api_key" \
        -H "Content-Type: application/json" \
        "$N8N_API_BASE/credentials" 2>/dev/null || echo '{"data":[]}')
    
    echo "$credentials_response" | jq -e --arg name "$credential_name" \
        '.data[]? | select(.name == $name)' >/dev/null 2>&1
}

#######################################
# Get existing credential ID by name
# Args: $1 - credential name
# Returns: credential ID or empty string
#######################################
n8n::get_credential_id() {
    local credential_name="$1"
    
    local api_key
    api_key=$(n8n::resolve_api_key 2>/dev/null)
    [[ -z "$api_key" ]] && return 1
    
    local credentials_response
    credentials_response=$(curl -s \
        -H "X-N8N-API-KEY: $api_key" \
        -H "Content-Type: application/json" \
        "$N8N_API_BASE/credentials" 2>/dev/null || echo '{"data":[]}')
    
    echo "$credentials_response" | jq -r --arg name "$credential_name" \
        '.data[]? | select(.name == $name) | .id // empty'
}

#######################################
# Main auto-credential management function
#######################################
n8n::auto_manage_credentials() {
    log::info "🔍 Auto-discovering resources for credential creation..."
    
    # Check if n8n is accessible first
    if ! n8n::check_basic_health >/dev/null 2>&1; then
        log::error "n8n is not accessible for credential creation"
        return 1
    fi
    
    # Check API key availability
    local api_key
    api_key=$(n8n::resolve_api_key 2>/dev/null)
    if [[ -z "$api_key" ]]; then
        log::error "N8N_API_KEY required for auto-credential creation"
        log::info "Create an API key in n8n and save it with:"
        log::info "  ./manage.sh --action save-api-key --api-key YOUR_KEY"
        return 1
    fi
    
    # Discover all running resources
    local discovered_resources
    discovered_resources=$(n8n::discover_resources)
    
    local resource_count
    resource_count=$(echo "$discovered_resources" | jq 'length')
    
    if [[ "$resource_count" -eq 0 ]]; then
        log::info "No running resources discovered for credential creation"
        return 0
    fi
    
    log::info "📊 Discovered $resource_count running resources"
    
    # Process each discovered resource (using for loop to avoid stdin issues)
    local created_count=0
    local updated_count=0
    local skipped_count=0
    local failed_count=0
    
    # Create array from JSON
    readarray -t resource_array < <(echo "$discovered_resources" | jq -c '.[]')
    
    for resource_info in "${resource_array[@]}"; do
        [[ -z "$resource_info" ]] && continue
        
        local resource_name
        resource_name=$(echo "$resource_info" | jq -r '.name')
        
        local credential_name="vrooli-$resource_name"
        
        log::debug "Processing resource: $resource_name"
        
        # Create credential configuration
        local credential_config
        credential_config=$(n8n::create_credential_config "$resource_info")
        
        if [[ -z "$credential_config" || "$credential_config" == "null" ]]; then
            log::warn "Could not create credential config for: $resource_name"
            ((failed_count++))
            continue
        fi
        
        # Check if credential already exists
        if n8n::credential_exists "$credential_name"; then
            log::debug "Credential already exists: $credential_name"
            ((skipped_count++))
            continue
        fi
        
        # Create the credential using existing function (don't fail on individual errors)
        if n8n::create_credential "$credential_config" 2>/dev/null; then
            log::success "✅ Auto-created credential: $credential_name"
            ((created_count++))
        else
            log::error "❌ Failed to create credential: $credential_name (possibly invalid connection data)"
            ((failed_count++))
        fi
    done
    
    # Summary report
    log::info ""
    log::info "📊 Auto-credential summary:"
    log::info "   Created: $created_count"
    log::info "   Existed: $skipped_count"
    log::info "   Failed:  $failed_count"
    log::info "   Total:   $resource_count"
    
    return 0
}

#######################################
# Validate auto-created credentials are accessible
#######################################
n8n::validate_auto_credentials() {
    log::debug "Validating auto-created credentials..."
    
    local api_key
    api_key=$(n8n::resolve_api_key 2>/dev/null)
    [[ -z "$api_key" ]] && return 1
    
    # Get all vrooli-* credentials
    local credentials_response
    credentials_response=$(curl -s \
        -H "X-N8N-API-KEY: $api_key" \
        -H "Content-Type: application/json" \
        "$N8N_API_BASE/credentials" 2>/dev/null || echo '{"data":[]}')
    
    local vrooli_credentials
    vrooli_credentials=$(echo "$credentials_response" | jq -r '.data[]? | select(.name | startswith("vrooli-")) | .name' | sort)
    
    if [[ -n "$vrooli_credentials" ]]; then
        local credential_count
        credential_count=$(echo "$vrooli_credentials" | wc -l)
        log::info "🔐 Auto-credentials available ($credential_count total):"
        
        while IFS= read -r cred_name; do
            [[ -z "$cred_name" ]] && continue
            log::info "   ✅ $cred_name"
        done <<< "$vrooli_credentials"
    else
        log::info "ℹ️  No auto-credentials found"
    fi
    
    return 0
}

#######################################
# List all auto-discoverable resources (for debugging)
#######################################
n8n::list_discoverable_resources() {
    log::info "🔍 Scanning for discoverable resources..."
    
    local discovered_resources
    discovered_resources=$(n8n::discover_resources)
    
    local resource_count
    resource_count=$(echo "$discovered_resources" | jq 'length')
    
    if [[ "$resource_count" -eq 0 ]]; then
        log::info "No resources currently discoverable"
        return 0
    fi
    
    log::info "📊 Discoverable resources ($resource_count total):"
    log::info ""
    
    echo "$discovered_resources" | jq -r '.[] | 
        "   🔧 \(.name) (\(.category))" +
        "\n      Type: \(.credential_type)" +
        "\n      Host: \(.connection.host)" + 
        (if .connection.port then ":\(.connection.port)" else "" end) +
        (if .connection.user then "\n      User: \(.connection.user)" else "" end) +
        "\n"'
    
    return 0
}