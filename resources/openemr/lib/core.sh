#!/usr/bin/env bash
# OpenEMR Core Functionality

set -euo pipefail

# Get script directory (only if not already set)
if [[ -z "${SCRIPT_DIR:-}" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi
if [[ -z "${RESOURCE_DIR:-}" ]]; then
    RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"
fi

# Source dependencies
source "${RESOURCE_DIR}/../../scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/../../scripts/resources/common.sh"
source "${RESOURCE_DIR}/../../scripts/resources/port_registry.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

# Manage lifecycle operations
openemr::manage() {
    local action="${1:-}"
    shift 1 || true
    
    case "$action" in
        install)
            openemr::install "$@"
            ;;
        uninstall)
            openemr::uninstall "$@"
            ;;
        start)
            openemr::start "$@"
            ;;
        stop)
            openemr::stop "$@"
            ;;
        restart)
            openemr::restart "$@"
            ;;
        *)
            log::error "Unknown manage action: $action"
            return 1
            ;;
    esac
}

# Install OpenEMR and dependencies
openemr::install() {
    log::info "Installing OpenEMR resource..."
    
    # Check Docker is available
    if ! command -v docker &>/dev/null; then
        log::error "Docker is required but not installed"
        return 1
    fi
    
    # Create required directories
    local dirs=(
        "$OPENEMR_DATA_DIR"
        "$OPENEMR_CONFIG_DIR"
        "$OPENEMR_LOGS_DIR"
        "$OPENEMR_BACKUP_DIR"
    )
    
    for dir in "${dirs[@]}"; do
        mkdir -p "$dir"
        log::debug "Created directory: $dir"
    done
    
    # Copy docker-compose file to config directory
    cp "${RESOURCE_DIR}/docker/docker-compose.yml" "${OPENEMR_CONFIG_DIR}/" || return 1
    
    # Create Docker network
    if ! docker network ls | grep -q "openemr-network"; then
        docker network create "openemr-network" || return 1
        log::debug "Created Docker network: openemr-network"
    fi
    
    # Pull Docker images using docker-compose
    log::info "Pulling Docker images..."
    cd "${OPENEMR_CONFIG_DIR}"
    docker-compose pull || return 1
    
    # Initialize configuration files
    openemr::init_config || return 1
    
    # Save credentials
    openemr::save_credentials || return 1
    
    log::success "OpenEMR installed successfully"
    return 0
}

# Initialize OpenEMR configuration
openemr::init_config() {
    log::debug "Initializing OpenEMR configuration..."
    
    # Create Apache config override if needed
    cat > "${OPENEMR_CONFIG_DIR}/openemr.conf" << 'EOF'
# Enable API access
SetEnv ENABLE_API_LOG 1

<Directory "/var/www/localhost/htdocs/openemr/apis">
    AllowOverride All
    Require all granted
</Directory>

<Directory "/var/www/localhost/htdocs/openemr/oauth2">
    AllowOverride All
    Require all granted
</Directory>
EOF
    
    log::debug "Created configuration files"
    return 0
}

# Save credentials to file
openemr::save_credentials() {
    local creds_file="${OPENEMR_CONFIG_DIR}/credentials.json"
    
    cat > "$creds_file" << EOF
{
  "admin": {
    "username": "${OPENEMR_ADMIN_USER}",
    "password": "${OPENEMR_ADMIN_PASS}",
    "email": "${OPENEMR_ADMIN_EMAIL}"
  },
  "database": {
    "host": "localhost",
    "port": ${OPENEMR_DB_PORT},
    "database": "${OPENEMR_DB_NAME}",
    "username": "${OPENEMR_DB_USER}",
    "password": "${OPENEMR_DB_PASS}"
  },
  "api": {
    "base_url": "http://localhost:${OPENEMR_API_PORT}/apis/default",
    "fhir_url": "http://localhost:${OPENEMR_FHIR_PORT}/apis/default/fhir/r4"
  },
  "web": {
    "url": "http://localhost:${OPENEMR_PORT}",
    "portal_url": "http://localhost:${OPENEMR_PORTAL_PORT}"
  }
}
EOF
    
    chmod 600 "$creds_file"
    log::debug "Saved credentials to $creds_file"
    return 0
}

# Start OpenEMR services
openemr::start() {
    local wait_for_ready=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait)
                wait_for_ready=true
                ;;
        esac
        shift
    done
    
    log::info "Starting OpenEMR services..."
    
    local compose_file="${OPENEMR_CONFIG_DIR}/docker-compose.yml"
    
    if [[ ! -f "$compose_file" ]]; then
        log::error "Docker Compose file not found. Run 'manage install' first"
        return 1
    fi
    
    # Start services
    cd "$OPENEMR_CONFIG_DIR"
    docker-compose up -d || return 1
    
    if [[ "$wait_for_ready" == "true" ]]; then
        openemr::wait_for_health || return 1
    fi
    
    log::success "OpenEMR started successfully"
    return 0
}

# Wait for health check
openemr::wait_for_health() {
    log::info "Waiting for OpenEMR to be ready..."
    
    local max_attempts=60
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        if timeout 5 curl -sf "http://localhost:${OPENEMR_PORT}/interface/login/login.php?site=default" &>/dev/null; then
            log::success "OpenEMR is ready"
            return 0
        fi
        
        ((attempt++))
        log::debug "Health check attempt $attempt/$max_attempts"
        sleep 3
    done
    
    log::error "OpenEMR failed to become ready"
    return 1
}

# Stop OpenEMR services
openemr::stop() {
    log::info "Stopping OpenEMR services..."
    
    local compose_file="${OPENEMR_CONFIG_DIR}/docker-compose.yml"
    
    if [[ ! -f "$compose_file" ]]; then
        log::warning "Docker Compose file not found"
        return 0
    fi
    
    cd "$OPENEMR_CONFIG_DIR"
    docker-compose down || return 1
    
    log::success "OpenEMR stopped successfully"
    return 0
}

# Restart OpenEMR services
openemr::restart() {
    log::info "Restarting OpenEMR services..."
    
    openemr::stop || return 1
    sleep 2
    openemr::start --wait || return 1
    
    return 0
}

# Uninstall OpenEMR
openemr::uninstall() {
    log::info "Uninstalling OpenEMR resource..."
    
    # Stop services
    openemr::stop || true
    
    # Remove containers
    docker rm -f "$OPENEMR_WEB_CONTAINER" "$OPENEMR_DB_CONTAINER" &>/dev/null || true
    
    # Remove network
    docker network rm "$OPENEMR_NETWORK" &>/dev/null || true
    
    # Optionally remove data
    local remove_data=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --remove-data)
                remove_data=true
                ;;
        esac
        shift
    done
    
    if [[ "$remove_data" == "true" ]]; then
        rm -rf "$OPENEMR_DATA_DIR"
        log::warning "Removed OpenEMR data"
    fi
    
    log::success "OpenEMR uninstalled successfully"
    return 0
}

# Content management operations
openemr::content() {
    local action="${1:-}"
    shift 1 || true
    
    case "$action" in
        add)
            openemr::content::add "$@"
            ;;
        list)
            openemr::content::list "$@"
            ;;
        get)
            openemr::content::get "$@"
            ;;
        remove)
            openemr::content::remove "$@"
            ;;
        export)
            openemr::content::export "$@"
            ;;
        *)
            log::error "Unknown content action: $action"
            return 1
            ;;
    esac
}

# Add content (patient, appointment, etc.)
openemr::content::add() {
    local type="${1:-}"
    shift 1 || true
    
    case "$type" in
        patient)
            openemr::add_patient "$@"
            ;;
        appointment)
            openemr::add_appointment "$@"
            ;;
        *)
            log::error "Unknown content type: $type"
            return 1
            ;;
    esac
}

# Add a patient
openemr::add_patient() {
    local name=""
    local dob=""
    local gender="male"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --dob)
                dob="$2"
                shift 2
                ;;
            --gender)
                gender="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]] || [[ -z "$dob" ]]; then
        log::error "Patient name and date of birth are required"
        return 1
    fi
    
    # Split name into first and last
    local first_name="${name%% *}"
    local last_name="${name#* }"
    
    # Get API token
    local token=$(openemr::get_api_token)
    
    if [[ -z "$token" ]]; then
        log::error "Failed to get API token"
        return 1
    fi
    
    # Create patient via API
    local response=$(curl -sf -X POST \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "{
            \"fname\": \"$first_name\",
            \"lname\": \"$last_name\",
            \"DOB\": \"$dob\",
            \"sex\": \"$gender\"
        }" \
        "http://localhost:${OPENEMR_API_PORT}/apis/default/api/patient")
    
    if [[ $? -eq 0 ]]; then
        local patient_id=$(echo "$response" | jq -r '.pid')
        log::success "Created patient: $name (ID: $patient_id)"
        echo "$response"
    else
        log::error "Failed to create patient"
        return 1
    fi
}

# Get API token
openemr::get_api_token() {
    # For demo, using basic auth to get token
    # In production, would use proper OAuth2 flow
    
    local creds_file="${OPENEMR_CONFIG_DIR}/credentials.json"
    
    if [[ ! -f "$creds_file" ]]; then
        log::error "Credentials file not found"
        return 1
    fi
    
    local username=$(jq -r '.admin.username' "$creds_file")
    local password=$(jq -r '.admin.password' "$creds_file")
    
    # Note: This is a simplified token retrieval
    # Real implementation would use OpenEMR's OAuth2 endpoints
    echo "demo-token-${username}"
}

# Show resource status
openemr::status() {
    log::info "OpenEMR Resource Status"
    echo "========================"
    
    # Check containers
    local web_status="Stopped"
    local db_status="Stopped"
    
    if docker ps --format '{{.Names}}' | grep -q "^${OPENEMR_WEB_CONTAINER}$"; then
        web_status="Running"
    fi
    
    if docker ps --format '{{.Names}}' | grep -q "^${OPENEMR_DB_CONTAINER}$"; then
        db_status="Running"
    fi
    
    echo "Web Service: $web_status"
    echo "Database: $db_status"
    
    # Check health endpoint
    if [[ "$web_status" == "Running" ]]; then
        if timeout 5 curl -sf "http://localhost:${OPENEMR_PORT}/interface/login/login.php?site=default" &>/dev/null; then
            echo "Health Check: Healthy"
            # Also check API endpoints if enabled
            if timeout 5 curl -sf "http://localhost:${OPENEMR_PORT}/apis/default/auth" &>/dev/null; then
                echo "API Status: Available"
            else
                echo "API Status: Unavailable"
            fi
        else
            echo "Health Check: Unhealthy"
        fi
    fi
    
    # Show URLs
    echo ""
    echo "Access URLs:"
    echo "  Web Interface: http://localhost:${OPENEMR_PORT}"
    echo "  REST API: http://localhost:${OPENEMR_PORT}/apis/default"
    echo "  FHIR API: http://localhost:${OPENEMR_PORT}/apis/default/fhir/r4"
    echo "  Patient Portal: https://localhost:${OPENEMR_PORTAL_PORT}/portal"
    
    return 0
}

# Show logs
openemr::logs() {
    local tail_lines=50
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --tail)
                tail_lines="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    log::info "Showing last $tail_lines lines of logs"
    
    docker logs --tail "$tail_lines" "$OPENEMR_WEB_CONTAINER" 2>&1 || true
    
    return 0
}

# Display credentials
openemr::credentials() {
    local creds_file="${OPENEMR_CONFIG_DIR}/credentials.json"
    
    if [[ ! -f "$creds_file" ]]; then
        log::error "Credentials file not found. Run 'manage install' first"
        return 1
    fi
    
    log::info "OpenEMR Credentials"
    cat "$creds_file" | jq '.'
    
    return 0
}

# List content (patients, appointments, etc.)
openemr::content::list() {
    local type="${1:-}"
    shift 1 || true
    
    case "$type" in
        patient|patients)
            openemr::list_patients "$@"
            ;;
        appointment|appointments)
            openemr::list_appointments "$@"
            ;;
        provider|providers)
            openemr::list_providers "$@"
            ;;
        *)
            log::error "Unknown content type: $type. Available: patient, appointment, provider"
            return 1
            ;;
    esac
}

# List patients
openemr::list_patients() {
    log::info "Listing patients..."
    
    local token=$(openemr::get_api_token)
    
    if [[ -z "$token" ]]; then
        log::warning "API authentication not available"
        return 1
    fi
    
    local response=$(curl -sf -X GET \
        -H "Authorization: Bearer $token" \
        "http://localhost:${OPENEMR_PORT}/apis/default/api/patient")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to list patients"
        return 1
    fi
}

# List appointments
openemr::list_appointments() {
    log::info "Listing appointments..."
    
    local token=$(openemr::get_api_token)
    
    if [[ -z "$token" ]]; then
        log::warning "API authentication not available"
        return 1
    fi
    
    local response=$(curl -sf -X GET \
        -H "Authorization: Bearer $token" \
        "http://localhost:${OPENEMR_PORT}/apis/default/api/appointment")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to list appointments"
        return 1
    fi
}

# List providers
openemr::list_providers() {
    log::info "Listing providers..."
    
    local token=$(openemr::get_api_token)
    
    if [[ -z "$token" ]]; then
        log::warning "API authentication not available"
        return 1
    fi
    
    local response=$(curl -sf -X GET \
        -H "Authorization: Bearer $token" \
        "http://localhost:${OPENEMR_PORT}/apis/default/api/practitioner")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to list providers"
        return 1
    fi
}

# Get content details
openemr::content::get() {
    local type="${1:-}"
    local id="${2:-}"
    shift 2 || true
    
    if [[ -z "$id" ]]; then
        log::error "ID is required"
        return 1
    fi
    
    case "$type" in
        patient)
            openemr::get_patient "$id" "$@"
            ;;
        appointment)
            openemr::get_appointment "$id" "$@"
            ;;
        *)
            log::error "Unknown content type: $type"
            return 1
            ;;
    esac
}

# Get patient details
openemr::get_patient() {
    local patient_id="$1"
    
    if [[ -z "$patient_id" ]]; then
        log::error "Patient ID is required"
        return 1
    fi
    
    local token=$(openemr::get_api_token)
    
    if [[ -z "$token" ]]; then
        log::warning "API authentication not available"
        return 1
    fi
    
    local response=$(curl -sf -X GET \
        -H "Authorization: Bearer $token" \
        "http://localhost:${OPENEMR_PORT}/apis/default/api/patient/${patient_id}")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to get patient details"
        return 1
    fi
}

# Get appointment details
openemr::get_appointment() {
    local appointment_id="$1"
    
    if [[ -z "$appointment_id" ]]; then
        log::error "Appointment ID is required"
        return 1
    fi
    
    local token=$(openemr::get_api_token)
    
    if [[ -z "$token" ]]; then
        log::warning "API authentication not available"
        return 1
    fi
    
    local response=$(curl -sf -X GET \
        -H "Authorization: Bearer $token" \
        "http://localhost:${OPENEMR_PORT}/apis/default/api/appointment/${appointment_id}")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to get appointment details"
        return 1
    fi
}

# Remove content
openemr::content::remove() {
    local type="${1:-}"
    shift 1 || true
    
    case "$type" in
        patient)
            log::warning "Patient removal not supported for safety reasons"
            return 1
            ;;
        appointment)
            openemr::remove_appointment "$@"
            ;;
        *)
            log::error "Unknown content type: $type"
            return 1
            ;;
    esac
}

# Remove appointment
openemr::remove_appointment() {
    local appointment_id="${1:-}"
    
    if [[ -z "$appointment_id" ]]; then
        log::error "Appointment ID is required"
        return 1
    fi
    
    local token=$(openemr::get_api_token)
    
    if [[ -z "$token" ]]; then
        log::warning "API authentication not available"
        return 1
    fi
    
    local response=$(curl -sf -X DELETE \
        -H "Authorization: Bearer $token" \
        "http://localhost:${OPENEMR_PORT}/apis/default/api/appointment/${appointment_id}")
    
    if [[ $? -eq 0 ]]; then
        log::success "Appointment removed successfully"
        return 0
    else
        log::error "Failed to remove appointment"
        return 1
    fi
}

# Export content
openemr::content::export() {
    local type="${1:-}"
    shift 1 || true
    
    case "$type" in
        fhir)
            openemr::export_fhir "$@"
            ;;
        csv)
            openemr::export_csv "$@"
            ;;
        *)
            log::error "Unknown export type: $type. Available: fhir, csv"
            return 1
            ;;
    esac
}

# Export FHIR data
openemr::export_fhir() {
    log::info "Exporting FHIR data..."
    
    local token=$(openemr::get_api_token)
    
    if [[ -z "$token" ]]; then
        log::warning "API authentication not available"
        return 1
    fi
    
    # Request bulk FHIR export
    local response=$(curl -sf -X GET \
        -H "Authorization: Bearer $token" \
        -H "Accept: application/fhir+json" \
        "http://localhost:${OPENEMR_PORT}/apis/default/fhir/Patient")
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to export FHIR data"
        return 1
    fi
}

# Export CSV data
openemr::export_csv() {
    log::info "Exporting CSV data..."
    
    local output_file="${1:-openemr_export.csv}"
    
    # For now, export patient list as CSV
    local patients=$(openemr::list_patients 2>/dev/null)
    
    if [[ -n "$patients" ]]; then
        echo "$patients" | jq -r '
            ["ID", "First Name", "Last Name", "DOB", "Gender"] as $cols |
            $cols, (.[] | [.id, .fname, .lname, .DOB, .sex]) | @csv
        ' > "$output_file"
        
        log::success "Data exported to $output_file"
        return 0
    else
        log::error "Failed to export CSV data"
        return 1
    fi
}

# Add appointment
openemr::add_appointment() {
    local patient_id=""
    local provider_id=""
    local date=""
    local time=""
    local duration="15"
    local title="Office Visit"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --patient)
                patient_id="$2"
                shift 2
                ;;
            --provider)
                provider_id="$2"
                shift 2
                ;;
            --date)
                date="$2"
                shift 2
                ;;
            --time)
                time="$2"
                shift 2
                ;;
            --duration)
                duration="$2"
                shift 2
                ;;
            --title)
                title="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$patient_id" ]] || [[ -z "$date" ]] || [[ -z "$time" ]]; then
        log::error "Patient ID, date, and time are required"
        return 1
    fi
    
    local token=$(openemr::get_api_token)
    
    if [[ -z "$token" ]]; then
        log::warning "API authentication not available"
        return 1
    fi
    
    # Create appointment via API
    local appointment_data="{
        \"pid\": \"$patient_id\",
        \"pc_eid\": \"$provider_id\",
        \"pc_eventDate\": \"$date\",
        \"pc_startTime\": \"$time\",
        \"pc_duration\": \"$duration\",
        \"pc_title\": \"$title\",
        \"pc_apptstatus\": \"-\"
    }"
    
    local response=$(curl -sf -X POST \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$appointment_data" \
        "http://localhost:${OPENEMR_PORT}/apis/default/api/appointment")
    
    if [[ $? -eq 0 ]]; then
        log::success "Created appointment: $title on $date at $time"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to create appointment"
        return 1
    fi
}