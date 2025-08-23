#!/usr/bin/env bash
# n8n Auto-Credential Management System
# NEW APPROACH: Uses resource CLI commands to get connection info

set -euo pipefail

# Source guard to prevent multiple sourcing
[[ -n "${_N8N_AUTO_CREDENTIALS_SOURCED:-}" ]] && return 0
export _N8N_AUTO_CREDENTIALS_SOURCED=1

# Get script directory and source dependencies
N8N_AUTO_CREDS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${N8N_AUTO_CREDS_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${N8N_AUTO_CREDS_DIR}/../../../../lib/utils/log.sh}"
# shellcheck disable=SC1091
source "${N8N_AUTO_CREDS_DIR}/credential-registry.sh"

# Set up API base URL if not already defined
if [[ -z "${N8N_API_BASE:-}" ]]; then
    N8N_API_BASE="${N8N_BASE_URL:-http://localhost:5678}/api/v1"
fi
if ! readonly -p | grep -q "N8N_API_BASE"; then
    readonly N8N_API_BASE
fi

#######################################
# Discover resources using Vrooli CLI and resource CLI credentials commands
# Returns: JSON array of resources with connection info
#######################################
n8n::discover_resources() {
    log::debug "Discovering enabled resources using Vrooli CLI..."
    
    # Get all resources from Vrooli CLI, filter by enabled status
    local all_resources
    if ! all_resources=$(bash "${var_ROOT_DIR}/cli/commands/resource-commands.sh" list --format json 2>/dev/null); then
        log::error "Failed to get resources from Vrooli CLI"
        echo '[]'
        return 1
    fi
    
    # Filter to only enabled resources
    local enabled_resources
    enabled_resources=$(echo "$all_resources" | jq '[.[] | select(.enabled == true)]')
    
    # Check if we got valid JSON
    if ! echo "$enabled_resources" | jq empty 2>/dev/null; then
        log::error "Invalid JSON response from Vrooli CLI"
        echo '[]'
        return 1
    fi
    
    log::debug "Found enabled resources, checking for credential support..."
    
    local discovered_resources=()
    local resource_count
    resource_count=$(echo "$enabled_resources" | jq 'length')
    
    # Process each enabled resource to get credentials
    for ((i=0; i<resource_count; i++)); do
        local resource_info
        resource_info=$(echo "$enabled_resources" | jq -c ".[$i]")
        
        local resource_name category
        resource_name=$(echo "$resource_info" | jq -r '.name')
        category=$(echo "$resource_info" | jq -r '.category')
        
        # Skip n8n itself to avoid recursion
        if [[ "$resource_name" == "n8n" ]]; then
            log::debug "Skipping n8n itself to avoid recursion"
            continue
        fi
        
        # Try to get credentials from resource CLI
        log::debug "Getting credentials for $resource_name..."
        local credentials_json
        if credentials_json=$(bash "${var_ROOT_DIR}/cli/vrooli" resource "$resource_name" credentials --format json 2>/dev/null); then
            # Validate the JSON response
            if echo "$credentials_json" | jq empty 2>/dev/null; then
                local status connections_count
                status=$(echo "$credentials_json" | jq -r '.status')
                connections_count=$(echo "$credentials_json" | jq '.connections | length')
                
                # Only include running resources with connections
                if [[ "$status" == "running" && "$connections_count" -gt 0 ]]; then
                    log::debug "✅ $resource_name is running with $connections_count connection(s)"
                    discovered_resources+=("$credentials_json")
                else
                    log::debug "⏭️  Skipping $resource_name - status: $status, connections: $connections_count"
                fi
            else
                log::debug "⚠️  $resource_name returned invalid credentials JSON"
            fi
        else
            log::debug "⚠️  $resource_name does not support credentials command or is not running"
        fi
    done
    
    # Output discovered resources as JSON array
    if [[ ${#discovered_resources[@]} -gt 0 ]]; then
        printf '%s\n' "${discovered_resources[@]}" | jq -s '.'
    else
        echo '[]'
    fi
}

#######################################
# Convert resource credentials JSON to n8n credential configurations
# Args: $1 - credentials JSON from resource CLI
# Returns: array of n8n credential configurations
#######################################
n8n::convert_to_n8n_credentials() {
    local credentials_json="$1"
    
    local resource_name
    resource_name=$(echo "$credentials_json" | jq -r '.resource')
    
    local n8n_credentials=()
    local connections_count
    connections_count=$(echo "$credentials_json" | jq '.connections | length')
    
    # Process each connection
    for ((i=0; i<connections_count; i++)); do
        local connection
        connection=$(echo "$credentials_json" | jq -c ".connections[$i]")
        
        local conn_id conn_name n8n_type connection_info auth_info
        conn_id=$(echo "$connection" | jq -r '.id')
        conn_name=$(echo "$connection" | jq -r '.name')
        n8n_type=$(echo "$connection" | jq -r '.n8n_credential_type')
        connection_info=$(echo "$connection" | jq -c '.connection')
        auth_info=$(echo "$connection" | jq -c '.auth // {}')
        
        # Build n8n credential name (resource-id or just resource if single connection)
        local credential_name
        if [[ "$connections_count" -eq 1 ]]; then
            credential_name="vrooli-$resource_name"
        else
            credential_name="vrooli-$resource_name-$conn_id"
        fi
        
        # Convert to n8n credential format based on type
        local n8n_credential
        case "$n8n_type" in
            postgres)
                n8n_credential=$(n8n::build_postgres_credential "$credential_name" "$connection_info" "$auth_info")
                ;;
            redis)
                n8n_credential=$(n8n::build_redis_credential "$credential_name" "$connection_info" "$auth_info")
                ;;
            s3)
                n8n_credential=$(n8n::build_s3_credential "$credential_name" "$connection_info" "$auth_info")
                ;;
            ollama)
                n8n_credential=$(n8n::build_ollama_credential "$credential_name" "$connection_info" "$auth_info")
                ;;
            httpBasicAuth)
                n8n_credential=$(n8n::build_basic_auth_credential "$credential_name" "$connection_info" "$auth_info")
                ;;
            httpHeaderAuth)
                n8n_credential=$(n8n::build_header_auth_credential "$credential_name" "$connection_info" "$auth_info")
                ;;
            *)
                log::warn "Unknown n8n credential type: $n8n_type for $resource_name"
                continue
                ;;
        esac
        
        if [[ -n "$n8n_credential" && "$n8n_credential" != "null" ]]; then
            n8n_credentials+=("$n8n_credential")
        fi
    done
    
    # Output as JSON array
    if [[ ${#n8n_credentials[@]} -gt 0 ]]; then
        printf '%s\n' "${n8n_credentials[@]}" | jq -s '.'
    else
        echo '[]'
    fi
}

#######################################
# Build PostgreSQL credential for n8n
#######################################
n8n::build_postgres_credential() {
    local name="$1"
    local connection="$2" 
    local auth="$3"
    
    local host port database user password ssl
    host=$(echo "$connection" | jq -r '.host')
    port=$(echo "$connection" | jq -r '.port // 5432')
    database=$(echo "$connection" | jq -r '.database // "postgres"')
    ssl=$(echo "$connection" | jq -r '.ssl // false')
    
    user=$(echo "$auth" | jq -r '.username // "postgres"')
    password=$(echo "$auth" | jq -r '.password // ""')
    
    # Convert boolean SSL to string format that n8n expects
    local ssl_mode
    if [[ "$ssl" == "true" ]]; then
        ssl_mode="require"
    else
        ssl_mode="disable"
    fi
    
    jq -n \
        --arg name "$name" \
        --arg host "$host" \
        --argjson port "$port" \
        --arg database "$database" \
        --arg user "$user" \
        --arg password "$password" \
        --arg ssl "$ssl_mode" \
        '{
            name: $name,
            type: "postgres",
            data: {
                host: $host,
                port: $port,
                database: $database,
                user: $user,
                password: $password,
                ssl: $ssl,
                sshTunnel: false
            }
        }'
}

#######################################
# Build Redis credential for n8n
#######################################
n8n::build_redis_credential() {
    local name="$1"
    local connection="$2"
    local auth="$3"
    
    local host port database password
    host=$(echo "$connection" | jq -r '.host')
    port=$(echo "$connection" | jq -r '.port // 6379')
    database=$(echo "$connection" | jq -r '.database // 0')
    password=$(echo "$auth" | jq -r '.password // ""')
    
    jq -n \
        --arg name "$name" \
        --arg host "$host" \
        --argjson port "$port" \
        --argjson database "$database" \
        --arg password "$password" \
        '{
            name: $name,
            type: "redis",
            data: {
                host: $host,
                port: $port,
                database: $database,
                password: $password
            }
        }'
}

#######################################
# Build S3 credential for n8n
#######################################
n8n::build_s3_credential() {
    local name="$1"
    local connection="$2"
    local auth="$3"
    
    local host port ssl access_key secret_key
    host=$(echo "$connection" | jq -r '.host')
    port=$(echo "$connection" | jq -r '.port // 9000')
    ssl=$(echo "$connection" | jq -r '.ssl // false')
    
    access_key=$(echo "$auth" | jq -r '.username // "minioadmin"')
    secret_key=$(echo "$auth" | jq -r '.password // "minioadmin"')
    
    local endpoint
    if [[ "$ssl" == "true" ]]; then
        endpoint="https://${host}:${port}"
    else
        endpoint="http://${host}:${port}"
    fi
    
    jq -n \
        --arg name "$name" \
        --arg access_key "$access_key" \
        --arg secret_key "$secret_key" \
        --arg endpoint "$endpoint" \
        '{
            name: $name,
            type: "s3",
            data: {
                accessKeyId: $access_key,
                secretAccessKey: $secret_key,
                region: "us-east-1",
                customEndpoint: $endpoint,
                forcePathStyle: true,
                ssl: false
            }
        }'
}

#######################################
# Build Ollama credential for n8n
#######################################
n8n::build_ollama_credential() {
    local name="$1"
    local connection="$2"
    local auth="$3"
    
    local host port ssl
    host=$(echo "$connection" | jq -r '.host')
    port=$(echo "$connection" | jq -r '.port // 11434')
    ssl=$(echo "$connection" | jq -r '.ssl // false')
    
    local base_url
    if [[ "$ssl" == "true" ]]; then
        base_url="https://${host}:${port}"
    else
        base_url="http://${host}:${port}"
    fi
    
    jq -n \
        --arg name "$name" \
        --arg base_url "$base_url" \
        '{
            name: $name,
            type: "ollama",
            data: {
                baseUrl: $base_url
            }
        }'
}

#######################################
# Build HTTP Basic Auth credential for n8n
#######################################
n8n::build_basic_auth_credential() {
    local name="$1"
    local connection="$2"
    local auth="$3"
    
    local username password
    username=$(echo "$auth" | jq -r '.username // "admin"')
    password=$(echo "$auth" | jq -r '.password // "admin"')
    
    jq -n \
        --arg name "$name" \
        --arg user "$username" \
        --arg password "$password" \
        '{
            name: $name,
            type: "httpBasicAuth",
            data: {
                user: $user,
                password: $password
            }
        }'
}

#######################################
# Build HTTP Header Auth credential for n8n
#######################################
n8n::build_header_auth_credential() {
    local name="$1"
    local connection="$2" 
    local auth="$3"
    
    local header_name header_value
    header_name=$(echo "$auth" | jq -r '.header_name // "Authorization"')
    header_value=$(echo "$auth" | jq -r '.header_value // .token // ("Bearer " + (.api_key // "token"))')
    
    jq -n \
        --arg name "$name" \
        --arg header_name "$header_name" \
        --arg header_value "$header_value" \
        '{
            name: $name,
            type: "httpHeaderAuth",
            data: {
                name: $header_name,
                value: $header_value
            }
        }'
}

#######################################
# Check if credential already exists using local registry
# Args: $1 - credential name
# Returns: 0 if exists, 1 if not
#######################################
n8n::credential_exists() {
    local credential_name="$1"
    
    credential_registry::exists "$credential_name"
}

#######################################
# Create n8n credential
# Args: $1 - credential JSON configuration
# Returns: 0 if successful, 1 if failed
#######################################
n8n::create_n8n_credential() {
    local credential_config="$1"
    
    local name type
    name=$(echo "$credential_config" | jq -r '.name')
    type=$(echo "$credential_config" | jq -r '.type')
    
    log::info "Creating n8n credential: $name (type: $type)"
    
    local api_key
    api_key=$(n8n::resolve_api_key)
    if [[ -z "$api_key" ]]; then
        log::error "N8N_API_KEY required for credential creation"
        return 1
    fi
    
    # Create the credential
    local response
    response=$(curl -s -w "\n__HTTP_CODE__:%{http_code}" \
        -X POST \
        -H "X-N8N-API-KEY: $api_key" \
        -H "Content-Type: application/json" \
        -d "$credential_config" \
        "$N8N_API_BASE/credentials" 2>/dev/null || echo "__HTTP_CODE__:000")
    
    local http_code
    http_code=$(echo "$response" | grep "__HTTP_CODE__:" | sed 's/.*__HTTP_CODE__://')
    response=$(echo "$response" | grep -v "__HTTP_CODE__:")
    
    if [[ "$http_code" == "200" ]] || [[ "$http_code" == "201" ]]; then
        local credential_id
        credential_id=$(echo "$response" | jq -r '.id // empty')
        if [[ -n "$credential_id" ]]; then
            log::success "✅ Created credential: $name (ID: $credential_id)"
            
            # Add to registry for future duplicate detection
            # Extract resource name from credential name (vrooli-resourcename format)
            local resource_name="${name#vrooli-}"
            resource_name="${resource_name%%-*}"  # Remove any suffix after first dash
            credential_registry::add "$name" "$credential_id" "$type" "$resource_name"
            
            return 0
        else
            log::error "❌ Created credential but no ID returned: $name"
            return 1
        fi
    else
        log::error "❌ Failed to create credential: $name (HTTP $http_code)"
        if [[ -n "$response" ]]; then
            local error_message
            error_message=$(echo "$response" | jq -r '.message // .error // .' 2>/dev/null || echo "$response")
            log::debug "API Error: $error_message"
        fi
        return 1
    fi
}

#######################################
# Main auto-credential management function
#######################################
n8n::auto_manage_credentials() {
    log::info "🔍 Auto-discovering resources using CLI-based approach..."
    
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
    
    # Discover all resources with credentials
    local discovered_resources
    discovered_resources=$(n8n::discover_resources)
    
    local resource_count
    resource_count=$(echo "$discovered_resources" | jq 'length')
    
    if [[ "$resource_count" -eq 0 ]]; then
        log::info "No running resources with credentials discovered"
        return 0
    fi
    
    log::info "📊 Discovered $resource_count resource(s) with credentials"
    
    # Track statistics
    local created_count=0
    local skipped_count=0
    local failed_count=0
    local total_credentials=0
    
    # Process each resource
    for ((i=0; i<resource_count; i++)); do
        local resource_credentials
        resource_credentials=$(echo "$discovered_resources" | jq -c ".[$i]")
        
        local resource_name
        resource_name=$(echo "$resource_credentials" | jq -r '.resource')
        
        log::debug "Processing credentials for: $resource_name"
        
        # Convert to n8n credential format
        local n8n_credentials
        n8n_credentials=$(n8n::convert_to_n8n_credentials "$resource_credentials")
        
        local credential_count
        credential_count=$(echo "$n8n_credentials" | jq 'length')
        total_credentials=$((total_credentials + credential_count))
        
        # Process each credential
        for ((j=0; j<credential_count; j++)); do
            local credential_config
            credential_config=$(echo "$n8n_credentials" | jq -c ".[$j]")
            
            local credential_name
            credential_name=$(echo "$credential_config" | jq -r '.name')
            
            # Check if credential already exists
            if n8n::credential_exists "$credential_name"; then
                log::debug "Credential already exists: $credential_name"
                ((skipped_count++))
                continue
            fi
            
            # Create the credential
            if n8n::create_n8n_credential "$credential_config"; then
                ((created_count++))
            else
                ((failed_count++))
            fi
        done
    done
    
    # Summary report
    log::info ""
    log::info "📊 Auto-credential summary:"
    log::info "   Resources: $resource_count"
    log::info "   Credentials: $total_credentials"
    log::info "   Created: $created_count"
    log::info "   Existed: $skipped_count"
    log::info "   Failed: $failed_count"
    
    return 0
}

#######################################
# Validate auto-created credentials
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
# List all discoverable resources (for debugging)
#######################################
n8n::list_discoverable_resources() {
    log::info "🔍 Scanning for resources with credential support..."
    
    local discovered_resources
    discovered_resources=$(n8n::discover_resources)
    
    local resource_count
    resource_count=$(echo "$discovered_resources" | jq 'length')
    
    if [[ "$resource_count" -eq 0 ]]; then
        log::info "No resources currently support credential discovery"
        return 0
    fi
    
    log::info "📊 Discoverable resources with credentials ($resource_count total):"
    log::info ""
    
    for ((i=0; i<resource_count; i++)); do
        local resource_data
        resource_data=$(echo "$discovered_resources" | jq -c ".[$i]")
        
        local resource_name status connections_count
        resource_name=$(echo "$resource_data" | jq -r '.resource')
        status=$(echo "$resource_data" | jq -r '.status')  
        connections_count=$(echo "$resource_data" | jq '.connections | length')
        
        log::info "   🔧 $resource_name (status: $status)"
        log::info "      Connections: $connections_count"
        
        # Show connection details
        for ((j=0; j<connections_count; j++)); do
            local connection
            connection=$(echo "$resource_data" | jq -c ".connections[$j]")
            
            local conn_name conn_type conn_host conn_port
            conn_name=$(echo "$connection" | jq -r '.name')
            conn_type=$(echo "$connection" | jq -r '.n8n_credential_type')
            conn_host=$(echo "$connection" | jq -r '.connection.host')
            conn_port=$(echo "$connection" | jq -r '.connection.port // "default"')
            
            log::info "        • $conn_name ($conn_type)"
            log::info "          Host: $conn_host:$conn_port"
        done
        log::info ""
    done
    
    return 0
}