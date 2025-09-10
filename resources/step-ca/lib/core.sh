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
    if [[ ! -f "$CONFIG_DIR/ca.json" ]]; then
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
    
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        status="running"
        # Check health
        if timeout 5 curl -sk "https://localhost:$STEPCA_PORT/health" >/dev/null 2>&1; then
            health="healthy"
            details="CA is operational"
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
  "container": "$CONTAINER_NAME"
}
EOF
    else
        echo "📊 Step-CA Status:"
        echo "  Status: $status"
        echo "  Health: $health"
        echo "  Details: $details"
        echo "  Port: $STEPCA_PORT"
        echo "  Container: $CONTAINER_NAME"
    fi
}

# Show logs
show_logs() {
    local lines="${1:-50}"
    
    if docker ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "📜 Step-CA Logs (last $lines lines):"
        docker logs --tail "$lines" "$CONTAINER_NAME" 2>&1
    else
        echo "⚠️  Container $CONTAINER_NAME not found"
        return 1
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
    if [[ ! -f "$CONFIG_DIR/ca.json" ]]; then
        echo "  Initializing Certificate Authority..."
        
        # Generate password
        openssl rand -base64 32 > "$CONFIG_DIR/password.txt"
        chmod 600 "$CONFIG_DIR/password.txt"
        
        # Run initialization
        docker run --rm \
            -v "$CONFIG_DIR:/home/step" \
            -e "DOCKER_STEPCA_INIT_NAME=Vrooli CA" \
            -e "DOCKER_STEPCA_INIT_DNS_NAMES=localhost,vrooli-step-ca" \
            -e "DOCKER_STEPCA_INIT_PROVISIONER_NAME=admin" \
            -e "DOCKER_STEPCA_INIT_PASSWORD_FILE=/home/step/password.txt" \
            "$DOCKER_IMAGE" step ca init --no-db
        
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
    
    # Ensure network exists
    if ! docker network ls --format "{{.Name}}" | grep -q "^vrooli-network$"; then
        echo "  Creating Docker network..."
        docker network create vrooli-network >/dev/null 2>&1 || true
    fi
    
    # Start container
    docker run -d \
        --name "$CONTAINER_NAME" \
        --network vrooli-network \
        -p "$STEPCA_PORT:9000" \
        -v "$CONFIG_DIR:/home/step" \
        --restart unless-stopped \
        "$DOCKER_IMAGE"
    
    # Wait for startup if requested
    if [[ "$wait_flag" == "--wait" ]]; then
        echo "  Waiting for Step-CA to be ready..."
        local retries=30
        while [[ $retries -gt 0 ]]; do
            if timeout 5 curl -sk "https://localhost:$STEPCA_PORT/health" >/dev/null 2>&1; then
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
    
    # TODO: Implement actual certificate issuance via Step CLI or API
    echo "  Certificate issuance would be implemented here"
    echo "  CN: $cn"
    echo "  SANs: ${san:-none}"
    echo "  Duration: $duration"
    echo "  Type: $type"
}

# List certificates
content_list() {
    echo "📋 Issued Certificates:"
    echo "  Certificate listing would be implemented here"
    # TODO: Query Step-CA database for certificates
}

# Get certificate details
content_get() {
    local cn="${2:-}"
    
    if [[ -z "$cn" ]]; then
        echo "❌ Common name (--cn) is required"
        return 1
    fi
    
    echo "📄 Certificate Details for $cn:"
    echo "  Certificate details would be shown here"
    # TODO: Retrieve certificate from Step-CA
}

# Remove (revoke) certificate
content_remove() {
    local cn="${2:-}"
    
    if [[ -z "$cn" ]]; then
        echo "❌ Common name (--cn) is required"
        return 1
    fi
    
    echo "🗑️  Revoking certificate for $cn..."
    echo "  Certificate revocation would be implemented here"
    # TODO: Revoke certificate via Step-CA API
}

# Execute CA operation
content_execute() {
    local operation="${2:-}"
    
    echo "⚙️  Executing CA operation: $operation"
    echo "  CA operations would be implemented here"
    # TODO: Execute various CA operations
}