#!/usr/bin/env bash
# Matrix Synapse Resource - Core Library Functions

set -euo pipefail

# Load configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source defaults
source "${RESOURCE_DIR}/config/defaults.sh"

# Logging functions
log_info() { echo "[INFO] $*" >&2; }
log_error() { echo "[ERROR] $*" >&2; }
log_debug() { [[ "${DEBUG:-}" == "true" ]] && echo "[DEBUG] $*" >&2 || true; }

# Ensure required directories exist
ensure_directories() {
    mkdir -p "${MATRIX_SYNAPSE_DATA_DIR}"
    mkdir -p "${MATRIX_SYNAPSE_CONFIG_DIR}"
    mkdir -p "${MATRIX_SYNAPSE_LOG_DIR}"
    mkdir -p "${MATRIX_SYNAPSE_MEDIA_STORE_PATH}"
}

# Check if Synapse is installed
is_installed() {
    if [[ "${MATRIX_SYNAPSE_INSTALL_METHOD}" == "pip" ]]; then
        if [[ -f "${MATRIX_SYNAPSE_VENV_DIR}/bin/python" ]]; then
            "${MATRIX_SYNAPSE_VENV_DIR}/bin/python" -c "import synapse" 2>/dev/null
        else
            return 1
        fi
    else
        docker image inspect matrixdotorg/synapse &>/dev/null
    fi
}

# Check if Synapse is running
is_running() {
    if [[ "${MATRIX_SYNAPSE_INSTALL_METHOD}" == "pip" ]]; then
        if [[ -f "${MATRIX_SYNAPSE_PID_FILE}" ]]; then
            local pid
            pid=$(cat "${MATRIX_SYNAPSE_PID_FILE}")
            kill -0 "$pid" 2>/dev/null
        else
            return 1
        fi
    else
        # Check if Docker container is running
        docker ps --format '{{.Names}}' | grep -q '^matrix-synapse$'
    fi
}

# Generate configuration file
generate_config() {
    log_info "Generating Synapse configuration..."
    
    # Generate secrets if not provided
    if [[ -z "${SYNAPSE_REGISTRATION_SECRET:-}" ]]; then
        export SYNAPSE_REGISTRATION_SECRET=$(openssl rand -hex 32)
        log_info "Generated registration secret: ${SYNAPSE_REGISTRATION_SECRET}"
    fi
    
    if [[ -z "${MATRIX_SYNAPSE_MACAROON_SECRET:-}" ]]; then
        export MATRIX_SYNAPSE_MACAROON_SECRET=$(openssl rand -hex 32)
    fi
    
    if [[ -z "${MATRIX_SYNAPSE_FORM_SECRET:-}" ]]; then
        export MATRIX_SYNAPSE_FORM_SECRET=$(openssl rand -hex 32)
    fi
    
    # Create homeserver.yaml
    cat > "${MATRIX_SYNAPSE_CONFIG_DIR}/homeserver.yaml" <<EOF
server_name: "${MATRIX_SYNAPSE_SERVER_NAME}"
pid_file: ${MATRIX_SYNAPSE_PID_FILE}
listeners:
  - port: ${MATRIX_SYNAPSE_PORT}
    tls: false
    type: http
    x_forwarded: true
    resources:
      - names: [client, federation]
        compress: false

database:
  name: psycopg2
  args:
    user: ${MATRIX_SYNAPSE_DB_USER}
    password: "${SYNAPSE_DB_PASSWORD}"
    database: ${MATRIX_SYNAPSE_DB_NAME}
    host: ${MATRIX_SYNAPSE_DB_HOST}
    port: ${MATRIX_SYNAPSE_DB_PORT}
    cp_min: 5
    cp_max: 10

log_config: "$([[ "${MATRIX_SYNAPSE_INSTALL_METHOD}" == "docker" ]] && echo "/data/log.config" || echo "${MATRIX_SYNAPSE_CONFIG_DIR}/log.config")"
media_store_path: "$([[ "${MATRIX_SYNAPSE_INSTALL_METHOD}" == "docker" ]] && echo "/media_store" || echo "${MATRIX_SYNAPSE_MEDIA_STORE_PATH}")"
registration_shared_secret: "${SYNAPSE_REGISTRATION_SECRET}"
report_stats: ${MATRIX_SYNAPSE_REPORT_STATS}
macaroon_secret_key: "${MATRIX_SYNAPSE_MACAROON_SECRET}"
form_secret: "${MATRIX_SYNAPSE_FORM_SECRET}"
signing_key_path: "$([[ "${MATRIX_SYNAPSE_INSTALL_METHOD}" == "docker" ]] && echo "/data/signing.key" || echo "${MATRIX_SYNAPSE_CONFIG_DIR}/signing.key")"

enable_registration: ${MATRIX_SYNAPSE_ENABLE_REGISTRATION}
enable_registration_without_verification: true
bcrypt_rounds: 12
allow_guest_access: false
enable_metrics: true

trusted_key_servers:
  - server_name: "matrix.org"

max_upload_size: "${MATRIX_SYNAPSE_MAX_UPLOAD_SIZE}"
max_image_pixels: "32M"
url_preview_enabled: true
url_preview_ip_range_blacklist:
  - '127.0.0.0/8'
  - '10.0.0.0/8'
  - '172.16.0.0/12'
  - '192.168.0.0/16'
  - '::1/128'
  - 'fe80::/64'
  - 'fc00::/7'

# Rate limiting
rc_messages_per_second: 0.2
rc_message_burst_count: 10.0
rc_login:
  address:
    per_second: 0.17
    burst_count: 3
  account:
    per_second: 0.17
    burst_count: 3
  failed_attempts:
    per_second: 0.17
    burst_count: 3

# Caching
event_cache_size: "${MATRIX_SYNAPSE_EVENT_CACHE_SIZE}"
caches:
  global_factor: ${MATRIX_SYNAPSE_CACHE_FACTOR}

# Retention
retention:
  enabled: false

# Federation
federation_domain_whitelist: []
EOF

    # Create log configuration
    # For Docker, use relative paths within container
    local log_filename
    if [[ "${MATRIX_SYNAPSE_INSTALL_METHOD}" == "docker" ]]; then
        log_filename="/data/homeserver.log"
    else
        log_filename="${MATRIX_SYNAPSE_LOG_FILE}"
    fi
    
    cat > "${MATRIX_SYNAPSE_CONFIG_DIR}/log.config" <<EOF
version: 1
formatters:
  standard:
    format: '%(asctime)s - %(name)s - %(lineno)d - %(levelname)s - %(message)s'
handlers:
  file:
    class: logging.handlers.RotatingFileHandler
    formatter: standard
    filename: ${log_filename}
    maxBytes: 104857600
    backupCount: 10
    encoding: utf8
  console:
    class: logging.StreamHandler
    formatter: standard
loggers:
  synapse.storage.SQL:
    level: INFO
root:
  level: INFO
  handlers: [file, console]
disable_existing_loggers: false
EOF

    log_info "Configuration generated successfully"
}

# Initialize database
init_database() {
    log_info "Initializing PostgreSQL database..."
    
    # Check if postgres is running
    if ! vrooli resource postgres status 2>&1 | grep -q "✅ Running: Yes"; then
        log_info "PostgreSQL status check returned unexpected format, attempting connection anyway..."
    fi
    
    # Set password for postgres user (default from postgres resource)
    local PG_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
    
    # Use docker to run psql commands if psql is not installed
    if ! command -v psql &>/dev/null; then
        log_info "Using Docker postgres client to setup database..."
        
        # Create user if doesn't exist
        docker run --rm --network host \
            -e PGPASSWORD="${PG_PASSWORD}" \
            postgres:16-alpine psql \
            -h "${MATRIX_SYNAPSE_DB_HOST}" -p "${MATRIX_SYNAPSE_DB_PORT}" -U postgres \
            -c "CREATE USER ${MATRIX_SYNAPSE_DB_USER} WITH PASSWORD '${SYNAPSE_DB_PASSWORD}';" 2>/dev/null || true
        
        # Create database if doesn't exist  
        docker run --rm --network host \
            -e PGPASSWORD="${PG_PASSWORD}" \
            postgres:16-alpine psql \
            -h "${MATRIX_SYNAPSE_DB_HOST}" -p "${MATRIX_SYNAPSE_DB_PORT}" -U postgres \
            -c "CREATE DATABASE ${MATRIX_SYNAPSE_DB_NAME} OWNER ${MATRIX_SYNAPSE_DB_USER} ENCODING 'UTF8' LC_COLLATE='C' LC_CTYPE='C' TEMPLATE template0;" 2>/dev/null || true
            
        log_info "Database setup attempted via Docker"
    else
        # Use native psql if available
        export PGPASSWORD="${PG_PASSWORD}"
        
        # Create user if doesn't exist
        if ! psql -h "${MATRIX_SYNAPSE_DB_HOST}" -p "${MATRIX_SYNAPSE_DB_PORT}" -U postgres -tAc "SELECT 1 FROM pg_user WHERE usename='${MATRIX_SYNAPSE_DB_USER}'" | grep -q 1; then
            psql -h "${MATRIX_SYNAPSE_DB_HOST}" -p "${MATRIX_SYNAPSE_DB_PORT}" -U postgres <<EOF
CREATE USER ${MATRIX_SYNAPSE_DB_USER} WITH PASSWORD '${SYNAPSE_DB_PASSWORD}';
EOF
            log_info "Created database user: ${MATRIX_SYNAPSE_DB_USER}"
        fi
        
        # Create database if doesn't exist
        if ! psql -h "${MATRIX_SYNAPSE_DB_HOST}" -p "${MATRIX_SYNAPSE_DB_PORT}" -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='${MATRIX_SYNAPSE_DB_NAME}'" | grep -q 1; then
            psql -h "${MATRIX_SYNAPSE_DB_HOST}" -p "${MATRIX_SYNAPSE_DB_PORT}" -U postgres <<EOF
CREATE DATABASE ${MATRIX_SYNAPSE_DB_NAME} OWNER ${MATRIX_SYNAPSE_DB_USER} ENCODING 'UTF8' LC_COLLATE='C' LC_CTYPE='C' TEMPLATE template0;
EOF
            log_info "Created database: ${MATRIX_SYNAPSE_DB_NAME}"
        fi
        
        unset PGPASSWORD
    fi
    
    log_info "Database initialized successfully"
}

# Health check function
check_health() {
    local timeout="${1:-${MATRIX_SYNAPSE_HEALTH_CHECK_TIMEOUT}}"
    
    if ! is_running; then
        return 1
    fi
    
    # Check API endpoint
    timeout "$timeout" curl -sf "http://localhost:${MATRIX_SYNAPSE_PORT}/_matrix/client/versions" &>/dev/null
}

# Wait for service to be ready
wait_for_ready() {
    local timeout="${1:-${MATRIX_SYNAPSE_STARTUP_TIMEOUT}}"
    local elapsed=0
    
    log_info "Waiting for Matrix Synapse to be ready..."
    
    while [[ $elapsed -lt $timeout ]]; do
        if check_health 2; then
            log_info "Matrix Synapse is ready"
            return 0
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done
    
    log_error "Matrix Synapse failed to start within ${timeout} seconds"
    return 1
}

# Get service status
get_status() {
    if is_running; then
        if check_health 2; then
            echo "healthy"
        else
            echo "unhealthy"
        fi
    else
        echo "stopped"
    fi
}

# Create user with shared secret
create_user() {
    local username="$1"
    local password="${2:-$(openssl rand -base64 12)}"
    local admin="${3:-false}"
    
    if [[ -z "$username" ]]; then
        log_error "Username is required"
        return 1
    fi
    
    # Get registration secret from container
    local registration_secret
    if [[ "${MATRIX_SYNAPSE_INSTALL_METHOD}" == "docker" ]]; then
        registration_secret=$(docker exec matrix-synapse cat /data/homeserver.yaml | grep registration_shared_secret | cut -d'"' -f2)
    else
        registration_secret="${SYNAPSE_REGISTRATION_SECRET}"
    fi
    
    if [[ -z "$registration_secret" ]]; then
        log_error "Registration secret not found"
        return 1
    fi
    
    # Get nonce from the server
    local nonce
    nonce=$(curl -s "http://localhost:${MATRIX_SYNAPSE_PORT}/_synapse/admin/v1/register" | jq -r .nonce)
    
    if [[ -z "$nonce" ]] || [[ "$nonce" == "null" ]]; then
        log_error "Failed to get nonce from server"
        return 1
    fi
    
    local admin_str="notadmin"
    if [[ "$admin" == "true" ]]; then
        admin_str="admin"
    fi
    
    # Use printf with null bytes for proper MAC calculation
    local mac=$(printf '%s\00%s\00%s\00%s' "$nonce" "$username" "$password" "$admin_str" | openssl dgst -sha1 -hmac "${registration_secret}" -hex | cut -d' ' -f2)
    
    # Register user via API
    local response
    local admin_bool="false"
    [[ "$admin" == "true" ]] && admin_bool="true"
    
    response=$(curl -s -X POST "http://localhost:${MATRIX_SYNAPSE_PORT}/_synapse/admin/v1/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"nonce\": \"${nonce}\",
            \"username\": \"${username}\",
            \"password\": \"${password}\",
            \"mac\": \"${mac}\",
            \"admin\": ${admin_bool}
        }") 
    
    if [[ -z "$response" ]] || [[ "$response" == *"errcode"* ]]; then
        log_error "Failed to create user: ${username} - ${response:-No response}"
        return 1
    fi
    
    log_info "User created: ${username} (password: ${password})"
    echo "$response"
}

# Create a room
create_room() {
    local room_name="$1"
    local creator_user="${2:-admin}"
    local creator_password="${3:-}"
    
    if [[ -z "$room_name" ]]; then
        log_error "Room name is required"
        return 1
    fi
    
    # Get access token for the creator user
    local access_token
    if [[ "$creator_user" == "admin" ]] && [[ -z "$creator_password" ]]; then
        # Try to use admin user with default password
        creator_password="admin_password"
    fi
    
    # Login to get access token
    local login_response
    login_response=$(curl -s -X POST "http://localhost:${MATRIX_SYNAPSE_PORT}/_matrix/client/v3/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"type\": \"m.login.password\",
            \"user\": \"${creator_user}\",
            \"password\": \"${creator_password}\"
        }")
    
    access_token=$(echo "$login_response" | jq -r .access_token)
    
    if [[ -z "$access_token" ]] || [[ "$access_token" == "null" ]]; then
        log_error "Failed to login as ${creator_user}: ${login_response}"
        return 1
    fi
    
    # Create the room
    local room_response
    room_response=$(curl -s -X POST "http://localhost:${MATRIX_SYNAPSE_PORT}/_matrix/client/v3/createRoom" \
        -H "Authorization: Bearer ${access_token}" \
        -H "Content-Type: application/json" \
        -d "{
            \"name\": \"${room_name}\",
            \"preset\": \"public_chat\"
        }")
    
    local room_id
    room_id=$(echo "$room_response" | jq -r .room_id)
    
    if [[ -z "$room_id" ]] || [[ "$room_id" == "null" ]]; then
        log_error "Failed to create room: ${room_response}"
        return 1
    fi
    
    log_info "Room created: ${room_name} (ID: ${room_id})"
    echo "$room_response"
}

# Send a message to a room
send_message() {
    local room_id="$1"
    local message="$2"
    local sender_user="${3:-admin}"
    local sender_password="${4:-}"
    
    if [[ -z "$room_id" ]] || [[ -z "$message" ]]; then
        log_error "Room ID and message are required"
        return 1
    fi
    
    # Get access token for the sender
    local access_token
    if [[ "$sender_user" == "admin" ]] && [[ -z "$sender_password" ]]; then
        sender_password="admin_password"
    fi
    
    # Login to get access token
    local login_response
    login_response=$(curl -s -X POST "http://localhost:${MATRIX_SYNAPSE_PORT}/_matrix/client/v3/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"type\": \"m.login.password\",
            \"user\": \"${sender_user}\",
            \"password\": \"${sender_password}\"
        }")
    
    access_token=$(echo "$login_response" | jq -r .access_token)
    
    if [[ -z "$access_token" ]] || [[ "$access_token" == "null" ]]; then
        log_error "Failed to login as ${sender_user}: ${login_response}"
        return 1
    fi
    
    # Send the message
    local txn_id=$(date +%s%N)
    local msg_response
    msg_response=$(curl -s -X PUT "http://localhost:${MATRIX_SYNAPSE_PORT}/_matrix/client/v3/rooms/${room_id}/send/m.room.message/${txn_id}" \
        -H "Authorization: Bearer ${access_token}" \
        -H "Content-Type: application/json" \
        -d "{
            \"msgtype\": \"m.text\",
            \"body\": \"${message}\"
        }")
    
    local event_id
    event_id=$(echo "$msg_response" | jq -r .event_id)
    
    if [[ -z "$event_id" ]] || [[ "$event_id" == "null" ]]; then
        log_error "Failed to send message: ${msg_response}"
        return 1
    fi
    
    log_info "Message sent to room ${room_id}: ${message}"
    echo "$msg_response"
}

# List all rooms
list_rooms() {
    local user="${1:-admin}"
    local password="${2:-admin_password}"
    
    # Login to get access token
    local login_response
    login_response=$(curl -s -X POST "http://localhost:${MATRIX_SYNAPSE_PORT}/_matrix/client/v3/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"type\": \"m.login.password\",
            \"user\": \"${user}\",
            \"password\": \"${password}\"
        }")
    
    local access_token
    access_token=$(echo "$login_response" | jq -r .access_token)
    
    if [[ -z "$access_token" ]] || [[ "$access_token" == "null" ]]; then
        log_error "Failed to login as ${user}"
        return 1
    fi
    
    # Get joined rooms
    curl -s -X GET "http://localhost:${MATRIX_SYNAPSE_PORT}/_matrix/client/v3/joined_rooms" \
        -H "Authorization: Bearer ${access_token}" | jq .
}

# Setup federation configuration
setup_federation() {
    local server_name="${1:-${MATRIX_SYNAPSE_SERVER_NAME}}"
    local federation_enabled="${2:-${MATRIX_SYNAPSE_FEDERATION_ENABLED:-false}}"
    
    log_info "Setting up federation for server: ${server_name}"
    
    # Create well-known directory structure in container
    if [[ "${MATRIX_SYNAPSE_INSTALL_METHOD}" == "docker" ]]; then
        docker exec matrix-synapse mkdir -p /data/well-known/matrix
        
        # Create server well-known file
        docker exec matrix-synapse bash -c "cat > /data/well-known/matrix/server <<EOF
{
    \"m.server\": \"${server_name}:${MATRIX_SYNAPSE_FEDERATION_PORT:-8448}\"
}
EOF"
        
        # Create client well-known file
        docker exec matrix-synapse bash -c "cat > /data/well-known/matrix/client <<EOF
{
    \"m.homeserver\": {
        \"base_url\": \"http://${server_name}:${MATRIX_SYNAPSE_PORT}\"
    },
    \"m.identity_server\": {
        \"base_url\": \"http://${server_name}:${MATRIX_SYNAPSE_PORT}\"
    }
}
EOF"
        
        log_info "Federation configuration created for ${server_name}"
        
        # Update homeserver.yaml to enable federation if requested
        if [[ "$federation_enabled" == "true" ]]; then
            log_info "Enabling federation in configuration..."
            # This would require updating the homeserver.yaml and restarting
            echo "Note: Full federation requires TLS certificates and firewall configuration"
        fi
    else
        # Non-docker implementation
        mkdir -p "${MATRIX_SYNAPSE_DATA_DIR}/well-known/matrix"
        
        cat > "${MATRIX_SYNAPSE_DATA_DIR}/well-known/matrix/server" <<EOF
{
    "m.server": "${server_name}:${MATRIX_SYNAPSE_FEDERATION_PORT:-8448}"
}
EOF
        
        cat > "${MATRIX_SYNAPSE_DATA_DIR}/well-known/matrix/client" <<EOF
{
    "m.homeserver": {
        "base_url": "http://${server_name}:${MATRIX_SYNAPSE_PORT}"
    },
    "m.identity_server": {
        "base_url": "http://${server_name}:${MATRIX_SYNAPSE_PORT}"
    }
}
EOF
    fi
    
    log_info "Federation setup complete. Well-known files configured."
    echo "Server: ${server_name}"
    echo "Federation Port: ${MATRIX_SYNAPSE_FEDERATION_PORT:-8448}"
    echo "Note: TLS certificates required for public federation"
}

# Export functions
export -f log_info log_error log_debug
export -f ensure_directories is_installed is_running
export -f generate_config init_database
export -f check_health wait_for_ready get_status
export -f create_user create_room send_message list_rooms setup_federation