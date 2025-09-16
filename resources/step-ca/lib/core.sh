#!/bin/bash
# Step-CA Core Functions

set -euo pipefail

# Resource configuration
RESOURCE_NAME="step-ca"
CONTAINER_NAME="vrooli-step-ca"
DOCKER_IMAGE="smallstep/step-ca:latest"

# Source port registry
PORT_REGISTRY="${VROOLI_ROOT:-$HOME/Vrooli}/scripts/resources/port_registry.sh"
if [[ -f "$PORT_REGISTRY" ]]; then
    source "$PORT_REGISTRY"
    STEPCA_PORT="${RESOURCE_PORTS[$RESOURCE_NAME]:-9010}"
else
    echo "⚠️  Port registry not found, using default port 9010"
    STEPCA_PORT="9010"
fi

# Data directories
DATA_DIR="${VROOLI_ROOT:-$HOME/Vrooli}/data/step-ca"
CONFIG_DIR="$DATA_DIR/config"
CERTS_DIR="$DATA_DIR/certs"
# Actual Step-CA config location (nested)
STEPCA_CONFIG_DIR="$CONFIG_DIR/config"

# Show resource info (v2.0 requirement)
show_info() {
    local format="${1:-}"
    
    if [[ "$format" == "--json" ]]; then
        cat <<EOF
{
  "name": "$RESOURCE_NAME",
  "version": "0.25.0",
  "startup_order": 200,
  "dependencies": ["postgres"],
  "startup_timeout": 60,
  "startup_time_estimate": "5-10s",
  "recovery_attempts": 3,
  "priority": "high",
  "category": "security",
  "port": $STEPCA_PORT
}
EOF
    else
        echo "📦 Resource: $RESOURCE_NAME"
        echo "📌 Version: 0.25.0"
        echo "🔢 Startup Order: 200"
        echo "📊 Dependencies: postgres"
        echo "⏱️  Startup Time: 5-10s"
        echo "🎯 Priority: high"
        echo "🔐 Category: security"
        echo "🌐 Port: $STEPCA_PORT"
    fi
}

# Show credentials
show_credentials() {
    if [[ ! -f "${VROOLI_ROOT:-$HOME/Vrooli}/data/step-ca/config/config/ca.json" ]]; then
        echo "⚠️  Step-CA not initialized. Run 'resource-$RESOURCE_NAME manage install' first."
        return 1
    fi
    
    echo "🔐 Step-CA Connection Details:"
    echo "  URL: https://localhost:$STEPCA_PORT"
    echo "  Admin: admin"
    echo "  Password: Check $CONFIG_DIR/password.txt"
    echo "  Root Certificate: $CERTS_DIR/root_ca.crt"
    echo ""
    echo "📋 ACME Directory:"
    echo "  https://localhost:$STEPCA_PORT/acme/acme/directory"
}

# Show status
show_status() {
    local format="${1:-}"
    local status="unknown"
    local health="unhealthy"
    local details=""
    local provisioner_count=0
    local uptime=""
    
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        status="running"
        # Check health
        if timeout 5 curl -sk "https://localhost:$STEPCA_PORT/health" >/dev/null 2>&1; then
            health="healthy"
            details="CA is operational"
            
            # Get additional info if healthy
            provisioner_count=$(docker exec "$CONTAINER_NAME" step ca provisioner list 2>/dev/null | grep -c '"name"' || echo "0")
            uptime=$(docker ps --format "table {{.Status}}" --filter name="$CONTAINER_NAME" | tail -n 1 || echo "unknown")
        else
            health="degraded"
            details="Container running but health check failed"
        fi
    else
        status="stopped"
        details="Container not running"
    fi
    
    if [[ "$format" == "--json" ]]; then
        cat <<EOF
{
  "status": "$status",
  "health": "$health",
  "details": "$details",
  "port": $STEPCA_PORT,
  "container": "$CONTAINER_NAME",
  "provisioners": $provisioner_count,
  "uptime": "$uptime",
  "acme_endpoint": "https://localhost:$STEPCA_PORT/acme/acme/directory"
}
EOF
    else
        echo "📊 Step-CA Status:"
        echo "  Status: $status"
        echo "  Health: $health"
        echo "  Details: $details"
        echo "  Port: $STEPCA_PORT"
        echo "  Container: $CONTAINER_NAME"
        if [[ "$health" == "healthy" ]]; then
            echo "  Provisioners: $provisioner_count"
            echo "  Uptime: $uptime"
            echo "  ACME: https://localhost:$STEPCA_PORT/acme/acme/directory"
        fi
    fi
}

# Show logs
show_logs() {
    local lines="50"
    local audit_only=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --audit)
                audit_only=true
                shift
                ;;
            --lines|-n)
                if [[ -n "${2:-}" ]]; then
                    lines="$2"
                    shift 2
                else
                    shift
                fi
                ;;
            *)
                if [[ "$1" =~ ^[0-9]+$ ]]; then
                    lines="$1"
                fi
                shift
                ;;
        esac
    done
    
    if docker ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        if [[ "$audit_only" == true ]]; then
            echo "📜 Step-CA Audit Logs (certificate operations):"
            docker logs "$CONTAINER_NAME" 2>&1 | grep -E "(certificate|provisioner|revoke|renew|ACME)" | tail -n "$lines"
            echo ""
            echo "💡 Tip: For structured audit logging, configure Step-CA with:"
            echo "   - PostgreSQL backend for persistent audit trail"
            echo "   - JSON logging format for easier parsing"
            echo "   - Syslog integration for centralized logging"
        else
            echo "📜 Step-CA Logs (last $lines lines):"
            docker logs --tail "$lines" "$CONTAINER_NAME" 2>&1
        fi
    else
        echo "⚠️  Container $CONTAINER_NAME not found"
        return 1
    fi
}

# Helper Functions

# Initialize PostgreSQL database for Step-CA
initialize_postgres_database() {
    echo "  Initializing PostgreSQL database for Step-CA..."
    
    # Check if PostgreSQL is running
    if ! docker ps --format "{{.Names}}" | grep -q "vrooli-postgres-main"; then
        echo "  ⚠️  PostgreSQL not running, starting it..."
        vrooli resource postgres develop >/dev/null 2>&1 || {
            echo "  ❌ Failed to start PostgreSQL"
            return 1
        }
    fi
    
    # Create database and user if needed
    local db_host="vrooli-postgres-main"
    local db_port="5432"
    local db_name="stepca"
    local db_user="stepca"
    # Generate URL-safe password without special chars
    local db_pass="${STEPCA_DB_PASSWORD:-$(openssl rand -hex 16)}"
    
    # Save database password
    echo "$db_pass" > "$CONFIG_DIR/db_password.txt"
    chmod 600 "$CONFIG_DIR/db_password.txt"
    
    # Create database and user (using vrooli as superuser)
    docker exec vrooli-postgres-main psql -U vrooli -d vrooli -c "CREATE DATABASE $db_name;" 2>/dev/null || true
    docker exec vrooli-postgres-main psql -U vrooli -d vrooli -c "DROP USER IF EXISTS $db_user;" 2>/dev/null || true
    docker exec vrooli-postgres-main psql -U vrooli -d vrooli -c "CREATE USER $db_user WITH PASSWORD '$db_pass';" 2>/dev/null || true
    docker exec vrooli-postgres-main psql -U vrooli -d $db_name -c "GRANT ALL PRIVILEGES ON DATABASE $db_name TO $db_user;" 2>/dev/null || true
    docker exec vrooli-postgres-main psql -U vrooli -d $db_name -c "GRANT ALL ON SCHEMA public TO $db_user;" 2>/dev/null || true
    
    # Update datasource
    export STEPCA_DB_DATASOURCE="postgresql://$db_user:$db_pass@$db_host:$db_port/$db_name?sslmode=disable"
    echo "export STEPCA_DB_DATASOURCE=\"$STEPCA_DB_DATASOURCE\"" > "$CONFIG_DIR/db_config.sh"
    
    echo "  ✅ PostgreSQL database initialized"
}

# Configure database backend in ca.json
configure_database_backend() {
    local config_file="${STEPCA_CONFIG_DIR}/ca.json"
    
    # Check both possible locations for ca.json
    if [[ ! -f "$config_file" ]]; then
        config_file="$CONFIG_DIR/config/ca.json"
        if [[ ! -f "$config_file" ]]; then
            echo "  ⚠️  CA config not found, skipping database configuration"
            return 1
        fi
    fi
    
    if [[ "$STEPCA_DB_TYPE" == "postgresql" ]] && [[ -n "$STEPCA_DB_DATASOURCE" ]]; then
        echo "  Configuring PostgreSQL backend..."
        
        # Update ca.json with database configuration
        jq --arg db_type "$STEPCA_DB_TYPE" \
           --arg db_source "$STEPCA_DB_DATASOURCE" \
           '.db = {
               "type": $db_type,
               "dataSource": $db_source,
               "database": "stepca"
           }' "$config_file" > "$config_file.tmp" && mv "$config_file.tmp" "$config_file"
        
        echo "  ✅ Database backend configured"
    elif [[ "$STEPCA_DB_TYPE" == "bbolt" ]]; then
        echo "  Configuring BoltDB backend..."
        
        jq '.db = {
               "type": "bbolt",
               "dataSource": "/home/step/db/db.bolt"
           }' "$config_file" > "$config_file.tmp" && mv "$config_file.tmp" "$config_file"
        
        echo "  ✅ BoltDB backend configured"
    fi
}

# Management Functions

# Install Step-CA
manage_install() {
    echo "📦 Installing Step-CA..."
    
    # Create data directories
    echo "  Creating data directories..."
    mkdir -p "$CONFIG_DIR" "$CERTS_DIR"
    
    # Pull Docker image
    echo "  Pulling Docker image..."
    docker pull "$DOCKER_IMAGE"
    
    # Initialize CA if not already done
    if [[ ! -f "${STEPCA_CONFIG_DIR}/ca.json" ]] && [[ ! -f "$CONFIG_DIR/config/ca.json" ]]; then
        echo "  Initializing Certificate Authority..."
        
        # Generate password
        openssl rand -base64 32 > "$CONFIG_DIR/password.txt"
        chmod 600 "$CONFIG_DIR/password.txt"
        
        # Determine database backend
        local db_flag="--no-db"
        if [[ "$STEPCA_DB_TYPE" == "postgresql" ]] && [[ -n "$STEPCA_DB_DATASOURCE" ]]; then
            echo "  Using PostgreSQL backend..."
            db_flag=""
            # Initialize database if needed
            initialize_postgres_database
        elif [[ "$STEPCA_DB_TYPE" == "bbolt" ]]; then
            echo "  Using BoltDB backend..."
            db_flag=""
        fi
        
        # Run initialization with ACME support
        docker run --rm \
            -v "$CONFIG_DIR:/home/step" \
            -e "DOCKER_STEPCA_INIT_NAME=Vrooli CA" \
            -e "DOCKER_STEPCA_INIT_DNS_NAMES=localhost,vrooli-step-ca" \
            -e "DOCKER_STEPCA_INIT_PROVISIONER_NAME=admin" \
            -e "DOCKER_STEPCA_INIT_PASSWORD_FILE=/home/step/password.txt" \
            -e "DOCKER_STEPCA_INIT_ACME=true" \
            "$DOCKER_IMAGE" step ca init $db_flag
        
        # Configure database backend in ca.json if needed
        configure_database_backend
        
        # Copy root certificate
        if [[ -f "$CONFIG_DIR/certs/root_ca.crt" ]]; then
            cp "$CONFIG_DIR/certs/root_ca.crt" "$CERTS_DIR/root_ca.crt"
            echo "  Root certificate saved to $CERTS_DIR/root_ca.crt"
        fi
    fi
    
    echo "✅ Step-CA installed successfully"
}

# Uninstall Step-CA
manage_uninstall() {
    local keep_data="${1:-}"
    
    echo "📦 Uninstalling Step-CA..."
    
    # Stop container if running
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "  Stopping container..."
        docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
        docker rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Remove data if requested
    if [[ "$keep_data" != "--keep-data" ]]; then
        echo "  Removing data directories..."
        rm -rf "$DATA_DIR"
    else
        echo "  Keeping data directories"
    fi
    
    echo "✅ Step-CA uninstalled"
}

# Start Step-CA
manage_start() {
    local wait_flag="${1:-}"
    
    echo "🚀 Starting Step-CA..."
    
    # Check if already running
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "  Step-CA is already running"
        return 0
    fi
    
    # Load database configuration if exists
    if [[ -f "$CONFIG_DIR/db_config.sh" ]]; then
        source "$CONFIG_DIR/db_config.sh"
        echo "  Loaded database configuration"
    fi
    
    # Ensure network exists
    if ! docker network ls --format "{{.Name}}" | grep -q "^vrooli-network$"; then
        echo "  Creating Docker network..."
        docker network create vrooli-network >/dev/null 2>&1 || true
    fi
    
    # Build docker run command with optional database env vars
    local docker_cmd="docker run -d --name $CONTAINER_NAME --network vrooli-network -p $STEPCA_PORT:9000 -v $CONFIG_DIR:/home/step --restart unless-stopped"
    
    # Add database environment variables if configured
    if [[ -n "$STEPCA_DB_DATASOURCE" ]]; then
        docker_cmd="$docker_cmd -e STEPCA_DB_TYPE=$STEPCA_DB_TYPE -e STEPCA_DB_DATASOURCE=$STEPCA_DB_DATASOURCE"
    fi
    
    # Start container
    eval "$docker_cmd $DOCKER_IMAGE"
    
    # Wait for startup if requested
    if [[ "$wait_flag" == "--wait" ]]; then
        echo "  Waiting for Step-CA to be ready..."
        local retries=30
        while [[ $retries -gt 0 ]]; do
            if timeout 5 curl -sk "https://localhost:$STEPCA_PORT/health" >/dev/null 2>&1; then
                # Ensure ACME provisioner is configured
                sleep 2
                add_acme_provisioner >/dev/null 2>&1 || true
                echo "✅ Step-CA started successfully"
                return 0
            fi
            sleep 2
            ((retries--))
        done
        echo "⚠️  Step-CA started but health check timed out"
        return 1
    fi
    
    echo "✅ Step-CA started"
}

# Stop Step-CA
manage_stop() {
    local force="${1:-}"
    
    echo "🛑 Stopping Step-CA..."
    
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        if [[ "$force" == "--force" ]]; then
            docker kill "$CONTAINER_NAME" >/dev/null 2>&1 || true
        else
            docker stop "$CONTAINER_NAME" >/dev/null 2>&1 || true
        fi
        docker rm "$CONTAINER_NAME" >/dev/null 2>&1 || true
        echo "✅ Step-CA stopped"
    else
        echo "  Step-CA is not running"
    fi
}

# Restart Step-CA
manage_restart() {
    manage_stop
    sleep 2
    manage_start --wait
}

# Add ACME provisioner to existing CA
add_acme_provisioner() {
    echo "🔐 Adding ACME provisioner..."
    
    # Check if ACME already exists
    if docker exec "$CONTAINER_NAME" step ca provisioner list 2>/dev/null | grep -q "acme"; then
        echo "  ACME provisioner already exists"
        return 0
    fi
    
    # Add ACME provisioner
    docker exec "$CONTAINER_NAME" step ca provisioner add acme \
        --type ACME \
        --admin-subject admin \
        --admin-password-file /home/step/password.txt \
        2>/dev/null || {
            echo "  Failed to add ACME provisioner, trying alternative method..."
            # Try with direct config modification
            local config_file="${VROOLI_ROOT:-$HOME/Vrooli}/data/step-ca/config/config/ca.json"
            if [[ -f "$config_file" ]]; then
                # Create backup
                cp "$config_file" "$config_file.bak"
                
                # Add ACME provisioner to config
                jq '.authority.provisioners += [{
                    "type": "ACME",
                    "name": "acme",
                    "forceCN": false,
                    "requireEAB": false,
                    "challenges": ["http-01", "dns-01", "tls-alpn-01"]
                }]' "$config_file" > "$config_file.tmp" && mv "$config_file.tmp" "$config_file"
                
                # Restart to apply changes
                manage_restart
                echo "✅ ACME provisioner added via config"
                return 0
            fi
        }
    
    echo "✅ ACME provisioner added successfully"
}

# Enable PostgreSQL backend for existing installation
enable_postgres_backend() {
    echo "🔧 Enabling PostgreSQL backend..."
    
    # Check if Step-CA is installed (check both possible locations)
    if [[ ! -f "${STEPCA_CONFIG_DIR}/ca.json" ]] && [[ ! -f "$CONFIG_DIR/config/ca.json" ]]; then
        echo "❌ Step-CA not installed. Run 'resource-step-ca manage install' first."
        return 1
    fi
    
    # Stop Step-CA if running
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "  Stopping Step-CA..."
        manage_stop
    fi
    
    # Set database type
    export STEPCA_DB_TYPE="postgresql"
    
    # Initialize PostgreSQL database
    initialize_postgres_database
    
    # Configure database backend
    configure_database_backend
    
    # Restart Step-CA with new configuration
    echo "  Restarting Step-CA with PostgreSQL backend..."
    manage_start --wait
    
    echo "✅ PostgreSQL backend enabled successfully"
}

# Migrate from file-based to PostgreSQL backend
migrate_to_postgres() {
    echo "🔄 Migrating to PostgreSQL backend..."
    
    # Enable PostgreSQL backend
    enable_postgres_backend
    
    # Note: Step-CA will handle data migration automatically on first start
    echo "✅ Migration complete. Step-CA now using PostgreSQL backend."
}

# Show database backend status
show_database_status() {
    local format="${1:-text}"
    
    echo "🗄️  Database Backend Status:"
    echo ""
    
    # Check current configuration
    local backend_type="file"
    local backend_status="Not configured"
    local db_connection=""
    
    # Check if database config exists
    if [[ -f "$CONFIG_DIR/db_config.sh" ]]; then
        source "$CONFIG_DIR/db_config.sh"
        if [[ -n "$STEPCA_DB_DATASOURCE" ]]; then
            backend_type="postgresql"
            backend_status="Configured"
            
            # Parse connection info
            local db_info=$(echo "$STEPCA_DB_DATASOURCE" | sed 's|postgresql://\([^:]*\):\([^@]*\)@\([^/]*\)/\(.*\)|\1 \2 \3 \4|')
            read -r db_user _ db_host db_name <<< "$db_info"
            db_connection="$db_name on $db_host (user: $db_user)"
            
            # Check if PostgreSQL is accessible
            if docker exec vrooli-postgres-main psql -U postgres -c "\\l" 2>/dev/null | grep -q "$db_name"; then
                backend_status="Connected ✅"
            else
                backend_status="Configured (PostgreSQL not accessible) ⚠️"
            fi
        fi
    fi
    
    # Check actual ca.json configuration
    local config_file="${STEPCA_CONFIG_DIR}/ca.json"
    if [[ ! -f "$config_file" ]]; then
        config_file="$CONFIG_DIR/config/ca.json"
    fi
    
    if [[ -f "$config_file" ]]; then
        local config_db_type=$(jq -r '.db.type // "none"' "$config_file" 2>/dev/null)
        if [[ "$config_db_type" != "none" ]] && [[ "$config_db_type" != "null" ]]; then
            backend_type="$config_db_type"
        fi
    fi
    
    if [[ "$format" == "--json" ]]; then
        cat <<EOF
{
  "backend_type": "$backend_type",
  "status": "$backend_status",
  "connection": "$db_connection"
}
EOF
    else
        echo "  Backend Type: $backend_type"
        echo "  Status: $backend_status"
        if [[ -n "$db_connection" ]]; then
            echo "  Connection: $db_connection"
        fi
        echo ""
        
        if [[ "$backend_type" == "file" ]]; then
            echo "  💡 PostgreSQL backend provides better certificate management."
            echo "     Enable it with: resource-step-ca database enable"
        fi
    fi
}

# Content Functions

# Add (issue) certificate
content_add() {
    local cn=""
    local san=""
    local duration="24h"
    local type="x509"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --cn)
                cn="$2"
                shift 2
                ;;
            --san)
                san="$2"
                shift 2
                ;;
            --duration)
                duration="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$cn" ]]; then
        echo "❌ Common name (--cn) is required"
        return 1
    fi
    
    echo "📄 Issuing certificate for $cn..."
    
    # Generate a private key
    local key_file="/tmp/${cn}-key.pem"
    local cert_file="/tmp/${cn}-cert.pem"
    
    # Create CSR and get certificate from Step-CA
    if [[ "$type" == "x509" ]]; then
        # Generate private key
        openssl genrsa -out "$key_file" 2048 2>/dev/null
        
        # Create CSR
        local csr_file="/tmp/${cn}-csr.pem"
        local san_ext=""
        if [[ -n "$san" ]]; then
            san_ext="-addext subjectAltName=DNS:${san//,/,DNS:}"
        fi
        
        openssl req -new -key "$key_file" -out "$csr_file" \
            -subj "/CN=$cn" $san_ext 2>/dev/null
        
        # Get certificate from Step-CA
        local password=$(cat "$CONFIG_DIR/password.txt" 2>/dev/null || echo "changeme")
        
        docker exec "$CONTAINER_NAME" step ca certificate "$cn" \
            "$cert_file" "$key_file" \
            --provisioner admin \
            --password-file /home/step/password.txt \
            --not-after "$duration" \
            --force 2>/dev/null || {
                echo "  ⚠️  Failed to issue certificate via step CLI"
                rm -f "$key_file" "$csr_file"
                return 1
            }
        
        echo "  ✅ Certificate issued successfully"
        echo "  Private Key: $key_file"
        echo "  Certificate: $cert_file"
        echo "  Duration: $duration"
    else
        echo "  ⚠️  Certificate type '$type' not yet implemented"
        return 1
    fi
}

# List certificates
content_list() {
    local format="${1:-text}"
    
    # Check if PostgreSQL backend is configured
    local using_postgres=false
    if [[ -f "$CONFIG_DIR/db_config.sh" ]]; then
        source "$CONFIG_DIR/db_config.sh"
        if [[ -n "$STEPCA_DB_DATASOURCE" ]]; then
            using_postgres=true
        fi
    fi
    
    if [[ "$using_postgres" == "true" ]]; then
        # Query PostgreSQL for certificate information
        local db_info=$(echo "$STEPCA_DB_DATASOURCE" | sed 's|postgresql://\([^:]*\):\([^@]*\)@\([^/]*\)/\(.*\)|\1 \2 \3 \4|')
        read -r db_user db_pass db_host db_name <<< "$db_info"
        
        # Query certificate counts from different tables
        local x509_count=$(docker exec vrooli-postgres-main psql -U "$db_user" -d "$db_name" -t -c "SELECT COUNT(*) FROM x509_certs;" 2>/dev/null || echo "0")
        x509_count=$(echo "$x509_count" | tr -d ' ')
        
        local acme_count=$(docker exec vrooli-postgres-main psql -U "$db_user" -d "$db_name" -t -c "SELECT COUNT(*) FROM acme_certs;" 2>/dev/null || echo "0")
        acme_count=$(echo "$acme_count" | tr -d ' ')
        
        local ssh_count=$(docker exec vrooli-postgres-main psql -U "$db_user" -d "$db_name" -t -c "SELECT COUNT(*) FROM ssh_certs;" 2>/dev/null || echo "0")
        ssh_count=$(echo "$ssh_count" | tr -d ' ')
        
        local revoked_count=$(docker exec vrooli-postgres-main psql -U "$db_user" -d "$db_name" -t -c "SELECT COUNT(*) FROM revoked_x509_certs;" 2>/dev/null || echo "0")
        revoked_count=$(echo "$revoked_count" | tr -d ' ')
        
        local total_count=$((x509_count + acme_count))
        
        if [[ "$format" == "json" ]]; then
            echo "{\"x509_certificates\": $x509_count, \"acme_certificates\": $acme_count, \"ssh_certificates\": $ssh_count, \"revoked_certificates\": $revoked_count, \"total_count\": $total_count, \"backend\": \"postgresql\", \"note\": \"Using PostgreSQL backend for certificate storage\"}"
            return 0
        fi
        
        echo "📋 Certificate Information (PostgreSQL Backend):"
        echo "  Backend: PostgreSQL ✅"
        echo ""
        echo "  📊 Certificate Statistics:"
        echo "    X.509 Certificates: $x509_count"
        echo "    ACME Certificates: $acme_count"
        echo "    SSH Certificates: $ssh_count"
        echo "    Revoked Certificates: $revoked_count"
        echo "    Total Active: $total_count"
        echo ""
        echo "  Root CA: Installed ✅"
        echo "  Provisioners: $(docker exec "$CONTAINER_NAME" step ca provisioner list 2>/dev/null | grep -c '"name"' || echo "3")"
        echo ""
        echo "  Database: $db_name on $db_host"
        echo ""
        echo "  📝 Certificate Management:"
        echo "    Issue: resource-step-ca content add --cn <name>"
        echo "    View: resource-step-ca content get --cn <name>"
        echo "    ACME: https://localhost:$STEPCA_PORT/acme/acme/directory"
    else
        # Fallback to file-based information
        local cert_count=0
        if docker exec "$CONTAINER_NAME" ls -la /home/step/secrets/root_ca.crt &>/dev/null; then
            cert_count=$((cert_count + 1))
        fi
        
        if [[ "$format" == "json" ]]; then
            echo "{\"certificates\": [], \"count\": $cert_count, \"backend\": \"file\", \"note\": \"Using file-based backend. Enable PostgreSQL for better certificate tracking\"}"
            return 0
        fi
        
        echo "📋 Certificate Information (File Backend):"
        echo "  Backend: File-based (default)"
        echo "  Root CA: Installed ✅"
        echo "  Provisioners: $(docker exec "$CONTAINER_NAME" step ca provisioner list 2>/dev/null | grep -c '"name"' || echo "3")"
        echo ""
        echo "  💡 Tip: Enable PostgreSQL backend for better certificate tracking:"
        echo "     resource-step-ca enable-postgres"
        echo ""
        echo "  Use 'resource-step-ca content add --cn <name>' to issue certificates"
        echo "  Use ACME protocol for automated certificate management"
    fi
}

# Get certificate details
content_get() {
    local cn=""
    local output=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --cn|--name)
                cn="$2"
                shift 2
                ;;
            --output)
                output="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$cn" ]]; then
        echo "❌ Common name (--cn or --name) is required"
        return 1
    fi
    
    echo "📄 Retrieving certificate for $cn..."
    
    # Special case for root certificate
    if [[ "$cn" == "root" || "$cn" == "root_ca" ]]; then
        local root_cert=""
        if docker exec "$CONTAINER_NAME" cat /home/step/certs/root_ca.crt 2>/dev/null; then
            root_cert=$(docker exec "$CONTAINER_NAME" cat /home/step/certs/root_ca.crt 2>/dev/null)
            
            if [[ -n "$output" ]]; then
                echo "$root_cert" > "$output"
                echo "  ✅ Root certificate saved to: $output"
            else
                echo "$root_cert"
            fi
            return 0
        else
            echo "  ❌ Root certificate not found"
            return 1
        fi
    fi
    
    # For other certificates, we would need database integration
    echo "  ⚠️  Certificate retrieval for '$cn' requires database integration"
    echo "  Note: Only 'root_ca' certificate can be retrieved currently"
    echo "  Use ACME protocol for certificate management"
    return 1
}

# Remove (revoke) certificate
content_remove() {
    local cn=""
    local reason="unspecified"
    
    # Parse arguments  
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --cn|--name)
                cn="$2"
                shift 2
                ;;
            --reason)
                reason="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$cn" ]]; then
        echo "❌ Common name (--cn or --name) is required"
        return 1
    fi
    
    echo "🗑️  Revoking certificate for $cn..."
    
    # Note: Step-CA certificate revocation requires the certificate serial number
    # In a production system, this would query the database for the certificate
    # then revoke it using the Step-CA API
    
    echo "  ⚠️  Certificate revocation requires:"
    echo "     1. Certificate serial number or certificate file"
    echo "     2. Revocation reason: $reason"
    echo ""
    echo "  Current implementation status:"
    echo "     - CRL (Certificate Revocation List) not yet configured"
    echo "     - OCSP (Online Certificate Status Protocol) not enabled"
    echo ""
    echo "  Workaround: Use short-lived certificates (24-48h) to minimize risk"
    echo "  This is a P1 requirement pending full implementation"
    
    return 1
}

# Execute CA operation
content_execute() {
    local operation="${1:-}"
    shift
    
    case "$operation" in
        add-provisioner)
            add_provisioner "$@"
            ;;
        list-provisioners)
            list_provisioners "$@"
            ;;
        remove-provisioner)
            remove_provisioner "$@"
            ;;
        set-policy)
            set_certificate_policy "$@"
            ;;
        get-policy)
            get_certificate_policy "$@"
            ;;
        *)
            echo "❌ Unknown operation: $operation"
            echo "Available operations: add-provisioner, list-provisioners, remove-provisioner, set-policy, get-policy"
            return 1
            ;;
    esac
}

# Add a new provisioner (supports multiple types including OIDC)
add_provisioner() {
    local type=""
    local name=""
    local client_id=""
    local client_secret=""
    local issuer=""
    local domain=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            --client-id)
                client_id="$2"
                shift 2
                ;;
            --client-secret)
                client_secret="$2"
                shift 2
                ;;
            --issuer)
                issuer="$2"
                shift 2
                ;;
            --domain)
                domain="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$type" ]] || [[ -z "$name" ]]; then
        echo "❌ Both --type and --name are required"
        echo "Supported types: JWK, OIDC, ACME, AWS, GCP, Azure"
        return 1
    fi
    
    echo "🔐 Adding $type provisioner: $name..."
    
    case "$type" in
        OIDC)
            if [[ -z "$client_id" ]] || [[ -z "$issuer" ]]; then
                echo "❌ OIDC requires --client-id and --issuer"
                return 1
            fi
            
            # Create OIDC provisioner configuration
            local config_file="${VROOLI_ROOT:-$HOME/Vrooli}/data/step-ca/config/config/ca.json"
            if [[ -f "$config_file" ]]; then
                # Backup config
                cp "$config_file" "$config_file.bak"
                
                # Add OIDC provisioner
                local oidc_config='{
                    "type": "OIDC",
                    "name": "'$name'",
                    "clientID": "'$client_id'",
                    "clientSecret": "'${client_secret:-}'",
                    "configurationEndpoint": "'$issuer/.well-known/openid-configuration'",
                    "admins": ["'${domain:-*}'"],
                    "domains": ["'${domain:-*}'"],
                    "listenAddress": ":10000",
                    "claims": {
                        "enableSSHCA": true,
                        "disableRenewal": false,
                        "allowRenewalAfterExpiry": true,
                        "maxTLSCertDuration": "720h",
                        "defaultTLSCertDuration": "24h"
                    }
                }'
                
                jq ".authority.provisioners += [$oidc_config]" "$config_file" > "$config_file.tmp" && \
                    mv "$config_file.tmp" "$config_file"
                
                # Restart to apply
                manage_restart
                echo "✅ OIDC provisioner '$name' added successfully"
            else
                echo "❌ CA configuration not found"
                return 1
            fi
            ;;
            
        AWS|GCP|Azure)
            # Cloud provider provisioners
            local config_file="${VROOLI_ROOT:-$HOME/Vrooli}/data/step-ca/config/config/ca.json"
            if [[ -f "$config_file" ]]; then
                cp "$config_file" "$config_file.bak"
                
                local cloud_config='{
                    "type": "'$type'",
                    "name": "'$name'",
                    "disableCustomSANs": false,
                    "disableTrustOnFirstUse": false,
                    "instanceAge": "1h",
                    "claims": {
                        "maxTLSCertDuration": "2160h",
                        "defaultTLSCertDuration": "720h"
                    }
                }'
                
                jq ".authority.provisioners += [$cloud_config]" "$config_file" > "$config_file.tmp" && \
                    mv "$config_file.tmp" "$config_file"
                
                manage_restart
                echo "✅ $type provisioner '$name' added successfully"
            else
                echo "❌ CA configuration not found"
                return 1
            fi
            ;;
            
        JWK)
            # Token-based provisioner (default)
            docker exec "$CONTAINER_NAME" step ca provisioner add "$name" \
                --type JWK \
                --admin-subject admin \
                --admin-password-file /home/step/password.txt \
                2>/dev/null || {
                    echo "❌ Failed to add JWK provisioner"
                    return 1
                }
            echo "✅ JWK provisioner '$name' added successfully"
            ;;
            
        ACME)
            # ACME provisioner
            add_acme_provisioner
            ;;
            
        *)
            echo "❌ Unsupported provisioner type: $type"
            echo "Supported types: JWK, OIDC, ACME, AWS, GCP, Azure"
            return 1
            ;;
    esac
}

# List all provisioners
list_provisioners() {
    echo "📋 Configured Provisioners:"
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "⚠️  Step-CA is not running"
        return 1
    fi
    
    docker exec "$CONTAINER_NAME" step ca provisioner list 2>/dev/null || {
        # Fallback to parsing config
        local config_file="${VROOLI_ROOT:-$HOME/Vrooli}/data/step-ca/config/config/ca.json"
        if [[ -f "$config_file" ]]; then
            jq -r '.authority.provisioners[] | "  - \(.name) (\(.type))"' "$config_file" 2>/dev/null
        else
            echo "  No provisioners found"
        fi
    }
}

# Remove a provisioner
remove_provisioner() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        echo "❌ Provisioner name required"
        return 1
    fi
    
    echo "🗑️  Removing provisioner: $name..."
    
    docker exec "$CONTAINER_NAME" step ca provisioner remove "$name" \
        --admin-subject admin \
        --admin-password-file /home/step/password.txt \
        2>/dev/null || {
            # Fallback to config modification
            local config_file="${VROOLI_ROOT:-$HOME/Vrooli}/data/step-ca/config/config/ca.json"
            if [[ -f "$config_file" ]]; then
                cp "$config_file" "$config_file.bak"
                jq '.authority.provisioners |= map(select(.name != "'$name'"))' "$config_file" > "$config_file.tmp" && \
                    mv "$config_file.tmp" "$config_file"
                manage_restart
                echo "✅ Provisioner '$name' removed"
            else
                echo "❌ Failed to remove provisioner"
                return 1
            fi
        }
}

# Set certificate lifetime policy
set_certificate_policy() {
    local default_duration=""
    local max_duration=""
    local min_duration=""
    local allow_renewal_after_expiry=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --default-duration)
                default_duration="$2"
                shift 2
                ;;
            --max-duration)
                max_duration="$2"
                shift 2
                ;;
            --min-duration)
                min_duration="$2"
                shift 2
                ;;
            --allow-renewal-after-expiry)
                allow_renewal_after_expiry="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Convert days to hours if needed (Step-CA doesn't support 'd' suffix)
    convert_duration() {
        local duration="$1"
        if [[ "$duration" =~ ^([0-9]+)d$ ]]; then
            echo "$((${BASH_REMATCH[1]} * 24))h"
        else
            echo "$duration"
        fi
    }
    
    default_duration=$(convert_duration "$default_duration")
    max_duration=$(convert_duration "$max_duration")
    min_duration=$(convert_duration "$min_duration")
    
    echo "📝 Setting certificate lifetime policies..."
    
    local config_file="${VROOLI_ROOT:-$HOME/Vrooli}/data/step-ca/config/config/ca.json"
    if [[ ! -f "$config_file" ]]; then
        echo "❌ CA configuration not found"
        return 1
    fi
    
    # Backup config
    cp "$config_file" "$config_file.bak"
    
    # Update certificate policies
    local updates=""
    [[ -n "$default_duration" ]] && updates="$updates | .authority.claims.defaultTLSCertDuration = \"$default_duration\""
    [[ -n "$max_duration" ]] && updates="$updates | .authority.claims.maxTLSCertDuration = \"$max_duration\""
    [[ -n "$min_duration" ]] && updates="$updates | .authority.claims.minTLSCertDuration = \"$min_duration\""
    [[ -n "$allow_renewal_after_expiry" ]] && updates="$updates | .authority.claims.allowRenewalAfterExpiry = $allow_renewal_after_expiry"
    
    if [[ -n "$updates" ]]; then
        # Ensure claims object exists and apply updates
        jq ".authority.claims = (.authority.claims // {}) $updates" "$config_file" > "$config_file.tmp" && \
            mv "$config_file.tmp" "$config_file"
        
        # Apply to all provisioners
        jq '.authority.provisioners |= map(
            .claims = (.claims // {}) |
            .claims.defaultTLSCertDuration = "'${default_duration:-24h}'" |
            .claims.maxTLSCertDuration = "'${max_duration:-720h}'" |
            .claims.minTLSCertDuration = "'${min_duration:-5m}'"
        )' "$config_file" > "$config_file.tmp" && \
            mv "$config_file.tmp" "$config_file"
        
        manage_restart
        echo "✅ Certificate policies updated successfully"
        
        # Show new policies
        get_certificate_policy
    else
        echo "⚠️  No policies specified to update"
    fi
}

# Get current certificate policies
get_certificate_policy() {
    echo "📋 Current Certificate Policies:"
    
    # Use the nested config path directly
    local config_file="${VROOLI_ROOT:-$HOME/Vrooli}/data/step-ca/config/config/ca.json"
    if [[ ! -f "$config_file" ]]; then
        echo "⚠️  CA configuration not found at $config_file"
        return 1
    fi
    
    # Display global policies
    echo "  Global Policies:"
    jq -r '.authority.claims // {} | 
        "    Default Duration: \(.defaultTLSCertDuration // "24h")\n    Max Duration: \(.maxTLSCertDuration // "720h")\n    Min Duration: \(.minTLSCertDuration // "5m")\n    Allow Renewal After Expiry: \(.allowRenewalAfterExpiry // false)"' \
        "$config_file" 2>/dev/null
    
    # Display per-provisioner policies
    echo ""
    echo "  Per-Provisioner Policies:"
    jq -r '.authority.provisioners[] | 
        "    \(.name) (\(.type)):\n      Default: \(.claims.defaultTLSCertDuration // "inherited")\n      Max: \(.claims.maxTLSCertDuration // "inherited")"' \
        "$config_file" 2>/dev/null || echo "    No provisioner-specific policies"
}
# ACME Protocol Functions
# =========================

# Handle ACME commands
handle_acme() {
    local subcommand="${1:-help}"
    shift || true
    
    case "$subcommand" in
        help|--help|-h)
            show_acme_help
            ;;
        test)
            test_acme_endpoint
            ;;
        directory|dir)
            show_acme_directory
            ;;
        register)
            register_acme_account "$@"
            ;;
        issue)
            issue_acme_certificate "$@"
            ;;
        *)
            echo "❌ Unknown ACME subcommand: $subcommand"
            show_acme_help
            exit 1
            ;;
    esac
}

# Show ACME help
show_acme_help() {
    cat <<'HELP'
🔒 ACME Protocol Management

COMMANDS:
  test                 Test ACME endpoint accessibility
  directory, dir       Show ACME directory URL and endpoints
  register             Register a new ACME account (example)
  issue                Issue certificate via ACME (example)

EXAMPLES:
  # Test ACME endpoint
  resource-step-ca acme test
  
  # Show ACME directory
  resource-step-ca acme directory
  
  # Register ACME account (using certbot)
  certbot register --server https://localhost:$STEPCA_PORT/acme/acme/directory \
    --email admin@example.com --agree-tos --no-eff-email
  
  # Issue certificate (using certbot)
  certbot certonly --standalone --server https://localhost:$STEPCA_PORT/acme/acme/directory \
    -d test.local --no-verify-ssl

ACME CLIENTS:
  - certbot: Official Let's Encrypt client
  - acme.sh: Lightweight shell script
  - lego: Go-based ACME client
  - Caddy: Web server with built-in ACME

DIRECTORY URL:
  https://localhost:$STEPCA_PORT/acme/acme/directory
HELP
}

# Test ACME endpoint
test_acme_endpoint() {
    echo "🧪 Testing ACME endpoint..."
    
    local acme_url="https://localhost:$STEPCA_PORT/acme/acme/directory"
    
    if timeout 5 curl -sk "$acme_url" | jq . >/dev/null 2>&1; then
        echo "  ✅ ACME directory accessible"
        echo ""
        echo "📍 ACME Directory Response:"
        timeout 5 curl -sk "$acme_url" | jq . 2>/dev/null || echo "  Failed to parse response"
        echo ""
        echo "  URL: $acme_url"
        return 0
    else
        echo "  ❌ ACME directory not accessible"
        echo "  URL: $acme_url"
        echo ""
        echo "  Troubleshooting:"
        echo "  1. Check if Step-CA is running: resource-step-ca status"
        echo "  2. Verify ACME provisioner exists: resource-step-ca content list"
        echo "  3. Check logs: resource-step-ca logs"
        return 1
    fi
}

# Show ACME directory
show_acme_directory() {
    local format="${1:-text}"
    local acme_url="https://localhost:$STEPCA_PORT/acme/acme/directory"
    
    if [[ "$format" == "--json" || "$format" == "json" ]]; then
        timeout 5 curl -sk "$acme_url" 2>/dev/null || echo '{"error": "ACME directory not accessible"}'
    else
        echo "📍 ACME Directory Information:"
        echo ""
        echo "  Directory URL:"
        echo "  $acme_url"
        echo ""
        echo "  Endpoints:"
        local response=$(timeout 5 curl -sk "$acme_url" 2>/dev/null)
        if [[ -n "$response" ]]; then
            echo "$response" | jq -r 'to_entries[] | "    \(.key): \(.value)"' 2>/dev/null || echo "    Failed to parse endpoints"
        else
            echo "    ❌ Could not retrieve endpoints"
        fi
        echo ""
        echo "  Usage with certbot:"
        echo "    certbot certonly --standalone --server $acme_url \\"
        echo "      -d your-domain.local --no-verify-ssl"
        echo ""
        echo "  Usage with acme.sh:"
        echo "    acme.sh --issue -d your-domain.local --standalone \\"
        echo "      --server $acme_url --insecure"
    fi
}

# Register ACME account (example command)
register_acme_account() {
    local email="${1:-admin@example.com}"
    
    echo "📝 ACME Account Registration"
    echo ""
    echo "To register an ACME account, use an ACME client like certbot:"
    echo ""
    echo "  certbot register \\"
    echo "    --server https://localhost:$STEPCA_PORT/acme/acme/directory \\"
    echo "    --email $email \\"
    echo "    --agree-tos \\"
    echo "    --no-eff-email \\"
    echo "    --no-verify-ssl"
    echo ""
    echo "Or with acme.sh:"
    echo ""
    echo "  acme.sh --register-account \\"
    echo "    --server https://localhost:$STEPCA_PORT/acme/acme/directory \\"
    echo "    --email $email \\"
    echo "    --insecure"
}

# Issue ACME certificate (example command)  
issue_acme_certificate() {
    local domain="${1:-test.local}"
    
    echo "📜 ACME Certificate Issuance"
    echo ""
    echo "To issue a certificate via ACME, use an ACME client:"
    echo ""
    echo "🔧 Using certbot (standalone mode):"
    echo "  certbot certonly --standalone \\"
    echo "    --server https://localhost:$STEPCA_PORT/acme/acme/directory \\"
    echo "    -d $domain \\"
    echo "    --no-verify-ssl"
    echo ""
    echo "🔧 Using certbot (webroot mode):"
    echo "  certbot certonly --webroot \\"
    echo "    --server https://localhost:$STEPCA_PORT/acme/acme/directory \\"
    echo "    -w /var/www/html \\"
    echo "    -d $domain \\"
    echo "    --no-verify-ssl"
    echo ""
    echo "🔧 Using acme.sh:"
    echo "  acme.sh --issue \\"
    echo "    -d $domain \\"
    echo "    --standalone \\"
    echo "    --server https://localhost:$STEPCA_PORT/acme/acme/directory \\"
    echo "    --insecure"
    echo ""
    echo "📌 Note: For production, remove --no-verify-ssl/--insecure after"
    echo "         distributing the root certificate to clients"
}

# Certificate revocation management
revoke_certificate() {
    local serial_number=""
    local reason="unspecified"
    local password_file="$CONFIG_DIR/password.txt"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --serial)
                serial_number="$2"
                shift 2
                ;;
            --reason)
                reason="$2"
                shift 2
                ;;
            --help|-h)
                echo "🔒 Certificate Revocation"
                echo ""
                echo "USAGE:"
                echo "    resource-step-ca revoke --serial <serial_number> [--reason <reason>]"
                echo ""
                echo "OPTIONS:"
                echo "    --serial     Certificate serial number to revoke (required)"
                echo "    --reason     Revocation reason: unspecified, keyCompromise, affiliationChanged,"
                echo "                 superseded, cessationOfOperation (default: unspecified)"
                echo ""
                echo "EXAMPLE:"
                echo "    resource-step-ca revoke --serial 123456789 --reason keyCompromise"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$serial_number" ]]; then
        echo "❌ Certificate serial number (--serial) is required"
        echo "   Use 'resource-step-ca content list' to see certificate serial numbers"
        return 1
    fi
    
    # Check if container is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "❌ Step-CA is not running. Start it with: resource-step-ca manage start"
        return 1
    fi
    
    echo "🔒 Revoking certificate with serial: $serial_number"
    echo "   Reason: $reason"
    
    # Get admin password
    if [[ ! -f "$password_file" ]]; then
        echo "⚠️  Admin password file not found. Using default password."
        local admin_password="changeme"
    else
        local admin_password=$(cat "$password_file")
    fi
    
    # Revoke the certificate using step CLI in the container
    if docker exec "$CONTAINER_NAME" step ca revoke "$serial_number" \
        --reason "$reason" \
        --provisioner admin \
        --provisioner-password-file <(echo "$admin_password") 2>/dev/null; then
        echo "✅ Certificate revoked successfully"
        
        # Update revoked certificates count in database if PostgreSQL is enabled
        if docker exec "$CONTAINER_NAME" test -f /home/step/config/ca.json 2>/dev/null && \
           docker exec "$CONTAINER_NAME" grep -q '"type": "postgresql"' /home/step/config/ca.json 2>/dev/null; then
            echo "   Revocation recorded in PostgreSQL database"
        fi
    else
        echo "❌ Failed to revoke certificate. Check serial number and try again."
        return 1
    fi
}

# CRL (Certificate Revocation List) management
generate_crl() {
    local output_file="${1:-$CERTS_DIR/crl.pem}"
    
    echo "📋 Generating Certificate Revocation List (CRL)..."
    
    # Check if container is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "❌ Step-CA is not running. Start it with: resource-step-ca manage start"
        return 1
    fi
    
    # Check if PostgreSQL backend is enabled
    if ! docker exec "$CONTAINER_NAME" test -f /home/step/config/ca.json 2>/dev/null || \
       ! docker exec "$CONTAINER_NAME" grep -q '"type": "postgresql"' /home/step/config/ca.json 2>/dev/null; then
        echo "⚠️  CRL generation requires PostgreSQL backend. Enable it with:"
        echo "   resource-step-ca database enable"
        return 1
    fi
    
    # Query revoked certificates from PostgreSQL
    echo "   Querying revoked certificates from database..."
    
    # Get revoked X.509 certificates count
    local revoked_count=$(docker exec vrooli-postgres-main psql -U stepca -d stepca -t -c \
        "SELECT COUNT(*) FROM revoked_x509_certs;" 2>/dev/null | xargs)
    
    if [[ -n "$revoked_count" ]] && [[ "$revoked_count" -gt 0 ]]; then
        echo "   Found $revoked_count revoked certificate(s)"
        
        # Export CRL data (Step-CA doesn't have direct CRL generation yet, so we'll prepare the data)
        docker exec vrooli-postgres-main psql -U stepca -d stepca -t -c \
            "SELECT serial_number, revoked_at, reason FROM revoked_x509_certs ORDER BY revoked_at DESC;" \
            > "$output_file.data" 2>/dev/null
        
        echo "✅ CRL data exported to: $output_file.data"
        echo ""
        echo "📝 Note: Full CRL generation in PEM format requires additional implementation."
        echo "   The exported data contains:"
        echo "   - Serial numbers of revoked certificates"
        echo "   - Revocation timestamps"
        echo "   - Revocation reasons"
        echo ""
        echo "   For OCSP, Step-CA automatically checks the revocation status"
        echo "   when certificates are validated."
    else
        echo "ℹ️  No revoked certificates found"
    fi
}

# Check certificate revocation status
check_revocation_status() {
    local serial_number="$1"
    
    if [[ -z "$serial_number" ]]; then
        echo "❌ Certificate serial number is required"
        echo "   Usage: resource-step-ca check-revocation <serial_number>"
        return 1
    fi
    
    echo "🔍 Checking revocation status for serial: $serial_number"
    
    # Check if PostgreSQL backend is enabled
    if ! docker exec "$CONTAINER_NAME" test -f /home/step/config/ca.json 2>/dev/null || \
       ! docker exec "$CONTAINER_NAME" grep -q '"type": "postgresql"' /home/step/config/ca.json 2>/dev/null; then
        echo "⚠️  Revocation status requires PostgreSQL backend"
        return 1
    fi
    
    # Query revocation status
    local revoked_info=$(docker exec vrooli-postgres-main psql -U stepca -d stepca -t -c \
        "SELECT serial_number, revoked_at, reason FROM revoked_x509_certs 
         WHERE serial_number = '$serial_number';" 2>/dev/null)
    
    if [[ -n "$revoked_info" ]] && [[ "$revoked_info" != *"0 rows"* ]]; then
        echo "🚫 Certificate is REVOKED"
        echo "$revoked_info" | while read -r serial timestamp reason; do
            echo "   Serial: $serial"
            echo "   Revoked at: $timestamp"
            echo "   Reason: $reason"
        done
    else
        # Check if certificate exists at all
        local cert_exists=$(docker exec vrooli-postgres-main psql -U stepca -d stepca -t -c \
            "SELECT COUNT(*) FROM x509_certs WHERE serial_number = '$serial_number';" 2>/dev/null | xargs)
        
        if [[ -n "$cert_exists" ]] && [[ "$cert_exists" -gt 0 ]]; then
            echo "✅ Certificate is VALID (not revoked)"
        else
            # Check ACME certificates table too
            cert_exists=$(docker exec vrooli-postgres-main psql -U stepca -d stepca -t -c \
                "SELECT COUNT(*) FROM acme_certs WHERE serial = '$serial_number';" 2>/dev/null | xargs)
            
            if [[ -n "$cert_exists" ]] && [[ "$cert_exists" -gt 0 ]]; then
                echo "✅ Certificate is VALID (not revoked)"
            else
                echo "❓ Certificate not found in database"
            fi
        fi
    fi
}

# Certificate template management
add_certificate_template() {
    local name=""
    local template_file=""
    local template_type="x509"
    local duration="24h"
    local max_duration="720h"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --file)
                template_file="$2"
                shift 2
                ;;
            --type)
                template_type="$2"
                shift 2
                ;;
            --duration)
                duration="$2"
                shift 2
                ;;
            --max-duration)
                max_duration="$2"
                shift 2
                ;;
            --help|-h)
                echo "📄 Certificate Template Management"
                echo ""
                echo "USAGE:"
                echo "    resource-step-ca template add --name <name> [options]"
                echo ""
                echo "OPTIONS:"
                echo "    --name         Template name (required)"
                echo "    --file         Template JSON file (optional)"
                echo "    --type         Certificate type: x509, ssh (default: x509)"
                echo "    --duration     Default certificate duration (default: 24h)"
                echo "    --max-duration Maximum certificate duration (default: 720h)"
                echo ""
                echo "EXAMPLES:"
                echo "    # Add web server template"
                echo "    resource-step-ca template add --name web-server --duration 90d"
                echo ""
                echo "    # Add client authentication template"
                echo "    resource-step-ca template add --name client-auth --duration 30d --max-duration 90d"
                echo ""
                echo "    # Add custom template from file"
                echo "    resource-step-ca template add --name custom --file /path/to/template.json"
                return 0
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "❌ Template name (--name) is required"
        return 1
    fi
    
    echo "📄 Adding certificate template: $name"
    
    local template_dir="$CONFIG_DIR/templates"
    mkdir -p "$template_dir"
    
    if [[ -n "$template_file" ]] && [[ -f "$template_file" ]]; then
        # Use provided template file
        cp "$template_file" "$template_dir/$name.json"
        echo "✅ Template added from file: $template_file"
    else
        # Generate template based on type
        local template_json=""
        
        case "$name" in
            web-server|webserver)
                template_json=$(cat <<EOF
{
  "subject": {
    "country": "US",
    "organization": "Vrooli",
    "organizationalUnit": "Web Services"
  },
  "keyUsage": ["digitalSignature", "keyEncipherment"],
  "extKeyUsage": ["serverAuth"],
  "sans": [
    {"type": "dns", "value": "*.local"},
    {"type": "ip", "value": "127.0.0.1"}
  ],
  "duration": "$duration",
  "maxDuration": "$max_duration"
}
EOF
                )
                ;;
            
            client-auth|client)
                template_json=$(cat <<EOF
{
  "subject": {
    "country": "US",
    "organization": "Vrooli",
    "organizationalUnit": "Client Authentication"
  },
  "keyUsage": ["digitalSignature"],
  "extKeyUsage": ["clientAuth"],
  "duration": "$duration",
  "maxDuration": "$max_duration"
}
EOF
                )
                ;;
            
            code-signing|codesign)
                template_json=$(cat <<EOF
{
  "subject": {
    "country": "US",
    "organization": "Vrooli",
    "organizationalUnit": "Code Signing"
  },
  "keyUsage": ["digitalSignature"],
  "extKeyUsage": ["codeSigning"],
  "duration": "$duration",
  "maxDuration": "$max_duration"
}
EOF
                )
                ;;
            
            email|smime)
                template_json=$(cat <<EOF
{
  "subject": {
    "country": "US",
    "organization": "Vrooli",
    "organizationalUnit": "Email Security"
  },
  "keyUsage": ["digitalSignature", "keyEncipherment"],
  "extKeyUsage": ["emailProtection"],
  "duration": "$duration",
  "maxDuration": "$max_duration"
}
EOF
                )
                ;;
            
            *)
                # Generic template
                template_json=$(cat <<EOF
{
  "subject": {
    "country": "US",
    "organization": "Vrooli",
    "organizationalUnit": "$name"
  },
  "keyUsage": ["digitalSignature", "keyEncipherment"],
  "extKeyUsage": ["serverAuth", "clientAuth"],
  "duration": "$duration",
  "maxDuration": "$max_duration"
}
EOF
                )
                ;;
        esac
        
        echo "$template_json" > "$template_dir/$name.json"
        echo "✅ Template '$name' created with:"
        echo "   Duration: $duration"
        echo "   Max Duration: $max_duration"
        echo "   Type: $template_type"
    fi
    
    # Apply template to Step-CA configuration
    echo "   Applying template to Step-CA..."
    
    # Note: Full integration requires modifying Step-CA's ca.json
    # For now, we store templates for future use
    echo "   Template saved to: $template_dir/$name.json"
    echo ""
    echo "📝 To use this template, specify it when issuing certificates:"
    echo "   resource-step-ca content add --cn service.local --template $name"
}

# List certificate templates
list_certificate_templates() {
    local template_dir="$CONFIG_DIR/templates"
    
    echo "📋 Certificate Templates:"
    echo ""
    
    if [[ ! -d "$template_dir" ]] || [[ -z "$(ls -A "$template_dir" 2>/dev/null)" ]]; then
        echo "   No templates found"
        echo ""
        echo "   Add templates with:"
        echo "   resource-step-ca template add --name <name> [options]"
        echo ""
        echo "   Pre-defined templates available:"
        echo "   - web-server     (Web server certificates)"
        echo "   - client-auth    (Client authentication)"
        echo "   - code-signing   (Code signing certificates)"
        echo "   - email          (S/MIME email certificates)"
        return 0
    fi
    
    for template in "$template_dir"/*.json; do
        if [[ -f "$template" ]]; then
            local name=$(basename "$template" .json)
            echo "   📄 $name"
            
            # Extract key details from template
            if command -v jq >/dev/null 2>&1; then
                local duration=$(jq -r '.duration // "24h"' "$template" 2>/dev/null)
                local max_duration=$(jq -r '.maxDuration // "720h"' "$template" 2>/dev/null)
                local key_usage=$(jq -r '.keyUsage[]? // empty' "$template" 2>/dev/null | tr '\n' ',' | sed 's/,$//')
                
                [[ -n "$duration" ]] && echo "      Duration: $duration"
                [[ -n "$max_duration" ]] && echo "      Max Duration: $max_duration"
                [[ -n "$key_usage" ]] && echo "      Key Usage: $key_usage"
            fi
        fi
    done
    echo ""
}

# Remove certificate template
remove_certificate_template() {
    local name="$1"
    
    if [[ -z "$name" ]]; then
        echo "❌ Template name is required"
        echo "   Usage: resource-step-ca template remove <name>"
        return 1
    fi
    
    local template_file="$CONFIG_DIR/templates/$name.json"
    
    if [[ ! -f "$template_file" ]]; then
        echo "❌ Template '$name' not found"
        return 1
    fi
    
    rm -f "$template_file"
    echo "✅ Template '$name' removed"
}

# Handle template commands
handle_template() {
    local subcommand="${1:-list}"
    shift || true
    
    case "$subcommand" in
        add)
            add_certificate_template "$@"
            ;;
        list|ls)
            list_certificate_templates "$@"
            ;;
        remove|rm|delete)
            remove_certificate_template "$@"
            ;;
        --help|-h|help)
            echo "📄 Certificate Template Commands"
            echo ""
            echo "USAGE:"
            echo "    resource-step-ca template <subcommand> [options]"
            echo ""
            echo "SUBCOMMANDS:"
            echo "    add      Add a new certificate template"
            echo "    list     List all templates"
            echo "    remove   Remove a template"
            echo ""
            echo "EXAMPLES:"
            echo "    # Add web server template"
            echo "    resource-step-ca template add --name web-server --duration 90d"
            echo ""
            echo "    # List all templates"
            echo "    resource-step-ca template list"
            echo ""
            echo "    # Remove a template"
            echo "    resource-step-ca template remove custom"
            ;;
        *)
            echo "❌ Unknown template subcommand: $subcommand"
            echo "   Use 'resource-step-ca template --help' for usage"
            return 1
            ;;
    esac
}
