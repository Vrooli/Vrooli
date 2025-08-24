#!/usr/bin/env bash
# Shared utilities for resource credentials functionality
# Used by all resource CLIs to implement the 'credentials' command

set -euo pipefail

# Source guard to prevent multiple sourcing
[[ -n "${_CREDENTIALS_UTILS_SOURCED:-}" ]] && return 0
_CREDENTIALS_UTILS_SOURCED=1

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CREDENTIALS_LIB_DIR="${APP_ROOT}/scripts/resources/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || {
    # Fallback if var.sh not found
    VROOLI_ROOT="$APP_ROOT"
}
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${CREDENTIALS_LIB_DIR}/../../../lib/utils/log.sh}" 2>/dev/null || {
    # Fallback logging function if log.sh not found
    log::error() { echo "ERROR: $*" >&2; }
    log::info() { echo "INFO: $*"; }
}

# Credentials schema location
CREDENTIALS_SCHEMA="${CREDENTIALS_LIB_DIR}/credentials-schema.json"

#######################################
# Validate credentials JSON against schema
# Args: $1 - JSON string to validate
# Returns: 0 if valid, 1 if invalid
#######################################
credentials::validate_json() {
    local json="$1"
    
    # Basic JSON validation
    if ! echo "$json" | jq empty 2>/dev/null; then
        log::error "Invalid JSON format"
        return 1
    fi
    
    # Check required fields
    local resource status connections
    resource=$(echo "$json" | jq -r '.resource // empty')
    status=$(echo "$json" | jq -r '.status // empty') 
    connections=$(echo "$json" | jq -r '.connections // empty')
    
    if [[ -z "$resource" ]]; then
        log::error "Missing required field: resource"
        return 1
    fi
    
    if [[ -z "$status" ]]; then
        log::error "Missing required field: status"
        return 1
    fi
    
    if [[ -z "$connections" ]]; then
        log::error "Missing required field: connections"
        return 1
    fi
    
    # Validate status enum
    if ! echo "$status" | grep -qE '^(running|stopped|error)$'; then
        log::error "Invalid status value: $status (must be: running, stopped, or error)"
        return 1
    fi
    
    # Validate connections is array
    if ! echo "$json" | jq -e '.connections | type == "array"' >/dev/null; then
        log::error "Field 'connections' must be an array"
        return 1
    fi
    
    # Validate each connection has required fields
    local connection_count
    connection_count=$(echo "$json" | jq '.connections | length')
    
    for ((i=0; i<connection_count; i++)); do
        local connection
        connection=$(echo "$json" | jq -c ".connections[$i]")
        
        local conn_id conn_name conn_type conn_connection
        conn_id=$(echo "$connection" | jq -r '.id // empty')
        conn_name=$(echo "$connection" | jq -r '.name // empty')
        conn_type=$(echo "$connection" | jq -r '.n8n_credential_type // empty')
        conn_connection=$(echo "$connection" | jq -r '.connection // empty')
        
        if [[ -z "$conn_id" ]]; then
            log::error "Connection $i missing required field: id"
            return 1
        fi
        
        if [[ -z "$conn_name" ]]; then
            log::error "Connection $i missing required field: name"
            return 1
        fi
        
        if [[ -z "$conn_type" ]]; then
            log::error "Connection $i missing required field: n8n_credential_type"
            return 1
        fi
        
        if [[ -z "$conn_connection" ]] || [[ "$conn_connection" == "null" ]]; then
            log::error "Connection $i missing required field: connection"
            return 1
        fi
        
        # Validate connection has host
        local conn_host
        conn_host=$(echo "$connection" | jq -r '.connection.host // empty')
        if [[ -z "$conn_host" ]]; then
            log::error "Connection $i missing required field: connection.host"
            return 1
        fi
    done
    
    return 0
}

#######################################
# Build a standard credentials response
# Args: $1 - resource_name, $2 - status, $3 - connections_json_array
# Returns: formatted credentials JSON
#######################################
credentials::build_response() {
    local resource_name="$1"
    local status="$2" 
    local connections_array="$3"
    
    # Validate inputs
    if [[ -z "$resource_name" ]]; then
        log::error "Resource name is required"
        return 1
    fi
    
    if [[ -z "$status" ]]; then
        log::error "Status is required"
        return 1
    fi
    
    if [[ -z "$connections_array" ]]; then
        connections_array="[]"
    fi
    
    # Build response
    jq -n \
        --arg resource "$resource_name" \
        --arg status "$status" \
        --argjson connections "$connections_array" \
        '{
            resource: $resource,
            status: $status,
            connections: $connections
        }'
}

#######################################
# Build a connection object
# Args: $1 - id, $2 - name, $3 - n8n_type, $4 - connection_obj, $5 - auth_obj (optional), $6 - metadata_obj (optional)
# Returns: connection JSON object
#######################################
credentials::build_connection() {
    local conn_id="$1"
    local conn_name="$2"
    local n8n_type="$3"
    local connection_obj="$4"
    local auth_obj="${5:-{}}"
    local metadata_obj="${6:-{}}"
    
    # Ensure JSON objects are valid for --argjson
    # If they're already JSON strings, use them directly
    if ! echo "$connection_obj" | jq empty 2>/dev/null; then
        connection_obj="{}"
    fi
    if ! echo "$auth_obj" | jq empty 2>/dev/null; then
        auth_obj="{}"
    fi
    if ! echo "$metadata_obj" | jq empty 2>/dev/null; then
        metadata_obj="{}"
    fi
    
    jq -n \
        --arg id "$conn_id" \
        --arg name "$conn_name" \
        --arg n8n_credential_type "$n8n_type" \
        --argjson connection "$connection_obj" \
        --argjson auth "$auth_obj" \
        --argjson metadata "$metadata_obj" \
        '{
            id: $id,
            name: $name,
            n8n_credential_type: $n8n_credential_type,
            connection: $connection
        } | if ($auth | length > 0) then . + {auth: $auth} else . end
          | if ($metadata | length > 0) then . + {metadata: $metadata} else . end'
}

#######################################
# Build connection object for basic database
# Args: $1 - host, $2 - port, $3 - database, $4 - username, $5 - password, $6 - ssl (optional)
# Returns: complete database connection JSON
#######################################
credentials::build_database_connection() {
    local host="$1"
    local port="$2"
    local database="$3"
    local username="$4"
    local password="$5"
    local ssl="${6:-false}"
    
    local connection_obj
    connection_obj=$(jq -n \
        --arg host "$host" \
        --arg port "$port" \
        --arg database "$database" \
        --arg ssl "$ssl" \
        '{
            host: $host,
            port: ($port | tonumber),
            database: $database,
            ssl: ($ssl | test("true") | if . then true else false end)
        }')
    
    local auth_obj
    auth_obj=$(jq -n \
        --arg username "$username" \
        --arg password "$password" \
        '{
            username: $username,
            password: $password
        }')
    
    echo "$connection_obj|$auth_obj"
}

#######################################
# Build connection object for HTTP API
# Args: $1 - host, $2 - port, $3 - path (optional), $4 - ssl (optional), $5 - auth_type, $6 - auth_value
# Returns: complete HTTP API connection JSON  
#######################################
credentials::build_http_connection() {
    local host="$1"
    local port="$2"
    local path="${3:-}"
    local ssl="${4:-false}"
    local auth_type="${5:-}" # token, api_key, basic, header
    local auth_value="${6:-}"
    
    local connection_obj
    if [[ -n "$path" ]]; then
        connection_obj=$(jq -n \
            --arg host "$host" \
            --arg port "$port" \
            --arg path "$path" \
            --arg ssl "$ssl" \
            '{
                host: $host,
                port: ($port | tonumber),
                path: $path,
                ssl: ($ssl | test("true") | if . then true else false end)
            }')
    else
        connection_obj=$(jq -n \
            --arg host "$host" \
            --arg port "$port" \
            --arg ssl "$ssl" \
            '{
                host: $host,
                port: ($port | tonumber),
                ssl: ($ssl | test("true") | if . then true else false end)
            }')
    fi
    
    local auth_obj="{}"
    if [[ -n "$auth_type" && -n "$auth_value" ]]; then
        case "$auth_type" in
            token)
                auth_obj=$(jq -n --arg token "$auth_value" '{token: $token}')
                ;;
            api_key)
                auth_obj=$(jq -n --arg api_key "$auth_value" '{api_key: $api_key}')
                ;;
            basic)
                # auth_value should be "username:password"
                local username="${auth_value%%:*}"
                local password="${auth_value#*:}"
                auth_obj=$(jq -n \
                    --arg username "$username" \
                    --arg password "$password" \
                    '{username: $username, password: $password}')
                ;;
            header)
                # auth_value should be "header_name:header_value"
                local header_name="${auth_value%%:*}"
                local header_value="${auth_value#*:}"
                auth_obj=$(jq -n \
                    --arg header_name "$header_name" \
                    --arg header_value "$header_value" \
                    '{header_name: $header_name, header_value: $header_value}')
                ;;
        esac
    fi
    
    echo "$connection_obj|$auth_obj"
}

#######################################
# Get resource status by checking Docker container
# Args: $1 - container_name
# Returns: "running", "stopped", or "error"
#######################################
credentials::get_resource_status() {
    local container_name="$1"
    
    if ! command -v docker >/dev/null 2>&1; then
        echo "error"
        return
    fi
    
    if docker ps --format "{{.Names}}" | grep -q "^${container_name}$" 2>/dev/null; then
        echo "running"
    elif docker ps -a --format "{{.Names}}" | grep -q "^${container_name}$" 2>/dev/null; then
        echo "stopped" 
    else
        echo "error"
    fi
}

#######################################
# Show help for credentials command
#######################################
credentials::show_help() {
    local resource_name="${1:-resource}"
    
    cat << EOF
Get connection credentials for $resource_name

USAGE:
    vrooli resource $resource_name credentials [OPTIONS]

OPTIONS:
    --format FORMAT     Output format (json, pretty) [default: json]
    --show-sensitive    Show sensitive data like passwords [default: false]
    --help             Show this help message

EXAMPLES:
    vrooli resource $resource_name credentials
    vrooli resource $resource_name credentials --format pretty
    vrooli resource $resource_name credentials --show-sensitive

OUTPUT:
    Returns JSON containing connection information needed to create
    n8n credentials for this resource. Each resource can have multiple
    connections (e.g., Redis databases 0-15).
EOF
}

#######################################
# Parse credentials command arguments
# Sets global variables: CREDENTIALS_FORMAT, CREDENTIALS_SHOW_SENSITIVE
#######################################
credentials::parse_args() {
    CREDENTIALS_FORMAT="json"
    CREDENTIALS_SHOW_SENSITIVE="false"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                CREDENTIALS_FORMAT="$2"
                shift 2
                ;;
            --show-sensitive)
                CREDENTIALS_SHOW_SENSITIVE="true"
                shift
                ;;
            --help|-h)
                return 2  # Signal to show help
                ;;
            *)
                log::error "Unknown argument: $1"
                return 1
                ;;
        esac
    done
    
    # Validate format
    if [[ "$CREDENTIALS_FORMAT" != "json" && "$CREDENTIALS_FORMAT" != "pretty" ]]; then
        log::error "Invalid format: $CREDENTIALS_FORMAT (must be json or pretty)"
        return 1
    fi
    
    return 0
}

#######################################
# Mask sensitive data in credentials JSON
# Args: $1 - credentials JSON
# Returns: masked JSON
#######################################
credentials::mask_sensitive() {
    local json="$1"
    
    echo "$json" | jq '
        def mask_field: if . then "*****" else . end;
        
        .connections[] |= (
            if .auth then
                .auth |= {
                    username: .username,
                    password: (.password | mask_field),
                    token: (.token | mask_field), 
                    api_key: (.api_key | mask_field),
                    header_name: .header_name,
                    header_value: (.header_value | mask_field)
                }
            else . end
        )'
}

#######################################
# Format credentials output
# Args: $1 - credentials JSON
# Returns: formatted output based on CREDENTIALS_FORMAT
#######################################
credentials::format_output() {
    local json="$1"
    
    # Mask sensitive data if not explicitly requested
    if [[ "$CREDENTIALS_SHOW_SENSITIVE" == "false" ]]; then
        json=$(credentials::mask_sensitive "$json")
    fi
    
    case "$CREDENTIALS_FORMAT" in
        json)
            echo "$json" | jq '.'
            ;;
        pretty)
            echo "$json" | jq -r '
                "Resource: \(.resource)",
                "Status: \(.status)",
                "",
                "Connections:",
                (.connections[] | 
                    "  • \(.name) (\(.id))",
                    "    Type: \(.n8n_credential_type)",
                    "    Host: \(.connection.host):\(.connection.port // "default")",
                    (if .connection.database then "    Database: \(.connection.database)" else "" end),
                    (if .connection.path then "    Path: \(.connection.path)" else "" end),
                    (if .auth.username then "    Username: \(.auth.username)" else "" end),
                    ""
                )
            '
            ;;
    esac
}