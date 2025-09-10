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
        command -v synctl &>/dev/null || python3 -c "import synapse" 2>/dev/null
    else
        docker image inspect matrixdotorg/synapse &>/dev/null
    fi
}

# Check if Synapse is running
is_running() {
    if [[ -f "${MATRIX_SYNAPSE_PID_FILE}" ]]; then
        local pid
        pid=$(cat "${MATRIX_SYNAPSE_PID_FILE}")
        kill -0 "$pid" 2>/dev/null
    else
        return 1
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

log_config: "${MATRIX_SYNAPSE_CONFIG_DIR}/log.config"
media_store_path: "${MATRIX_SYNAPSE_MEDIA_STORE_PATH}"
registration_shared_secret: "${SYNAPSE_REGISTRATION_SECRET}"
report_stats: ${MATRIX_SYNAPSE_REPORT_STATS}
macaroon_secret_key: "${MATRIX_SYNAPSE_MACAROON_SECRET}"
form_secret: "${MATRIX_SYNAPSE_FORM_SECRET}"
signing_key_path: "${MATRIX_SYNAPSE_CONFIG_DIR}/signing.key"

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
    cat > "${MATRIX_SYNAPSE_CONFIG_DIR}/log.config" <<EOF
version: 1
formatters:
  standard:
    format: '%(asctime)s - %(name)s - %(lineno)d - %(levelname)s - %(message)s'
handlers:
  file:
    class: logging.handlers.RotatingFileHandler
    formatter: standard
    filename: ${MATRIX_SYNAPSE_LOG_FILE}
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
    if ! vrooli resource postgres status &>/dev/null; then
        log_error "PostgreSQL is not running. Please start it first."
        return 1
    fi
    
    # Create database and user if they don't exist
    export PGPASSWORD="${SYNAPSE_DB_PASSWORD}"
    
    # Create user
    psql -h "${MATRIX_SYNAPSE_DB_HOST}" -p "${MATRIX_SYNAPSE_DB_PORT}" -U postgres <<EOF 2>/dev/null || true
CREATE USER ${MATRIX_SYNAPSE_DB_USER} WITH PASSWORD '${SYNAPSE_DB_PASSWORD}';
EOF
    
    # Create database
    psql -h "${MATRIX_SYNAPSE_DB_HOST}" -p "${MATRIX_SYNAPSE_DB_PORT}" -U postgres <<EOF 2>/dev/null || true
CREATE DATABASE ${MATRIX_SYNAPSE_DB_NAME} OWNER ${MATRIX_SYNAPSE_DB_USER} ENCODING 'UTF8' LC_COLLATE='C' LC_CTYPE='C' TEMPLATE template0;
EOF
    
    unset PGPASSWORD
    
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
        echo "running"
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
    
    # Generate MAC for shared secret registration
    local nonce=$(openssl rand -hex 16)
    local mac_data="${nonce}\x00${username}\x00${password}\x00"
    if [[ "$admin" == "true" ]]; then
        mac_data="${mac_data}admin"
    else
        mac_data="${mac_data}notadmin"
    fi
    
    local mac=$(echo -ne "${mac_data}" | openssl dgst -sha1 -hmac "${SYNAPSE_REGISTRATION_SECRET}" -binary | xxd -p -c 256)
    
    # Register user via API
    local response
    response=$(curl -sf -X POST "http://localhost:${MATRIX_SYNAPSE_PORT}/_synapse/admin/v1/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"nonce\": \"${nonce}\",
            \"username\": \"${username}\",
            \"password\": \"${password}\",
            \"mac\": \"${mac}\",
            \"admin\": ${admin}
        }" 2>/dev/null) || {
        log_error "Failed to create user: ${username}"
        return 1
    }
    
    log_info "User created: ${username} (password: ${password})"
    echo "$response"
}

# Export functions
export -f log_info log_error log_debug
export -f ensure_directories is_installed is_running
export -f generate_config init_database
export -f check_health wait_for_ready get_status
export -f create_user