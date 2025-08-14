#!/usr/bin/env bash
# n8n Auto-Credential Management System
# Discovers resources and creates n8n credentials automatically

set -euo pipefail

# Source guard to prevent multiple sourcing
[[ -n "${_N8N_AUTO_CREDENTIALS_SOURCED:-}" ]] && return 0
export _N8N_AUTO_CREDENTIALS_SOURCED=1

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
# Auto-discover all running resources using the CLI
# Returns: JSON array of running resources with connection info
#######################################
n8n::discover_resources() {
    log::debug "Discovering running resources via CLI for credential creation..."
    
    # Use vrooli CLI to get running resources with connection info
    local cli_path="${var_ROOT_DIR}/cli/commands/resource-commands.sh"
    
    # Check if CLI is available
    if [[ ! -f "$cli_path" ]]; then
        log::error "CLI not found at $cli_path"
        echo '[]'
        return 1
    fi
    
    # Call CLI to get running resources with connection info
    local discovered_resources
    discovered_resources=$(bash "$cli_path" list --format json --include-connection-info --only-running 2>/dev/null || echo '[]')
    
    # Filter out n8n itself to avoid recursion and ensure we have connection info
    echo "$discovered_resources" | jq '[.[] | select(.name != "n8n" and .connection != {})]'
}

# Note: n8n::extract_resource_info function removed - now handled by CLI

#######################################
# Create credential configuration for each resource type
# Args: $1 - resource info JSON object from CLI
# Returns: credential configuration JSON
#######################################
n8n::create_credential_config() {
    local resource_info="$1"
    
    local name category connection
    name=$(echo "$resource_info" | jq -r '.name')
    category=$(echo "$resource_info" | jq -r '.category')
    connection=$(echo "$resource_info" | jq -c '.connection')
    
    # Get credential type from registry
    local type="${RESOURCE_CREDENTIAL_REGISTRY[$name]:-httpBasicAuth}"
    
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
            local host port user password region original_host
            host=$(echo "$connection" | jq -r '.host')
            original_host=$(echo "$connection" | jq -r '.original_host // .host')
            port=$(echo "$connection" | jq -r '.port // 9000')
            user=$(echo "$connection" | jq -r '.user // "minioadmin"')
            password=$(echo "$connection" | jq -r '.password // "minioadmin"')
            region=$(echo "$connection" | jq -r '.region // "us-east-1"')
            
            # Use original host for endpoint URL to avoid Docker networking issues in S3
            local endpoint="http://${original_host}:${port}"
            
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
            local host port original_host
            host=$(echo "$connection" | jq -r '.host')
            original_host=$(echo "$connection" | jq -r '.original_host // .host')
            port=$(echo "$connection" | jq -r '.port // 11434')
            
            # Use original host for base URL to avoid Docker networking issues
            local base_url="http://${original_host}:${port}"
            
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
            local host port user password original_host
            host=$(echo "$connection" | jq -r '.host')
            original_host=$(echo "$connection" | jq -r '.original_host // .host')
            port=$(echo "$connection" | jq -r '.port // 80')
            user=$(echo "$connection" | jq -r '.user // ""')
            password=$(echo "$connection" | jq -r '.password // ""')
            
            # Note: We use the Docker-adjusted host for basic auth since these are typically internal services
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
# Create Redis credentials for all databases (0-15)
# Args: $1 - Redis resource info JSON object from CLI
# Returns: 0 if successful, 1 if failed
#######################################
n8n::create_redis_database_credentials() {
    local redis_resource_info="$1"
    
    if [[ -z "$redis_resource_info" ]]; then
        log::error "No Redis resource info provided for database credential creation"
        return 1
    fi
    
    local connection
    connection=$(echo "$redis_resource_info" | jq -c '.connection')
    
    local host port password
    host=$(echo "$connection" | jq -r '.host')
    port=$(echo "$connection" | jq -r '.port // 6379')
    password=$(echo "$connection" | jq -r '.password // ""')
    
    log::info "🗄️  Creating Redis credentials for all databases (0-15)..."
    
    local created_count=0
    local skipped_count=0
    local failed_count=0
    
    # Create credentials for databases 0-15
    for db in {0..15}; do
        local credential_name="vrooli-redis-${db}"
        
        # Check if credential already exists
        if n8n::credential_exists "$credential_name"; then
            log::debug "Credential already exists: $credential_name"
            ((skipped_count++))
            continue
        fi
        
        # Create credential configuration for this database
        local credential_config
        credential_config=$(jq -n \
            --arg name "$credential_name" \
            --arg host "$host" \
            --arg port "$port" \
            --arg password "$password" \
            --arg database "$db" \
            '{
                name: $name,
                type: "redis",
                data: {
                    host: $host,
                    port: ($port | tonumber),
                    password: $password,
                    database: ($database | tonumber)
                }
            }')
        
        # Create the credential
        if n8n::create_credential "$credential_config" 2>/dev/null; then
            log::debug "✅ Created Redis credential: $credential_name (database $db)"
            ((created_count++))
        else
            log::error "❌ Failed to create Redis credential: $credential_name"
            ((failed_count++))
        fi
    done
    
    # Summary
    log::info "📊 Redis database credentials summary:"
    log::info "   Created: $created_count"
    log::info "   Existed: $skipped_count" 
    log::info "   Failed:  $failed_count"
    log::info "   Total:   16"
    
    return 0
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
    
    # Create additional Redis database credentials if Redis was discovered
    local redis_resource_info
    redis_resource_info=$(echo "$discovered_resources" | jq -c '.[] | select(.name == "redis")')
    
    if [[ -n "$redis_resource_info" && "$redis_resource_info" != "null" ]]; then
        log::info ""
        log::info "🗄️  Setting up Redis multi-database credentials..."
        if n8n::create_redis_database_credentials "$redis_resource_info"; then
            log::success "✅ Redis multi-database credentials configured"
        else
            log::warn "⚠️  Redis multi-database credential creation had issues (not fatal)"
        fi
    else
        log::debug "No Redis resource found for multi-database credential creation"
    fi
    
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