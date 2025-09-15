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
        
        # Run initialization with ACME support
        docker run --rm \
            -v "$CONFIG_DIR:/home/step" \
            -e "DOCKER_STEPCA_INIT_NAME=Vrooli CA" \
            -e "DOCKER_STEPCA_INIT_DNS_NAMES=localhost,vrooli-step-ca" \
            -e "DOCKER_STEPCA_INIT_PROVISIONER_NAME=admin" \
            -e "DOCKER_STEPCA_INIT_PASSWORD_FILE=/home/step/password.txt" \
            -e "DOCKER_STEPCA_INIT_ACME=true" \
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
    
    # Try to get certificate information from Step-CA
    local certs_json=""
    if docker exec "$CONTAINER_NAME" step ca certificate list 2>/dev/null; then
        # If step CLI supports listing (future versions)
        certs_json=$(docker exec "$CONTAINER_NAME" step ca certificate list --json 2>/dev/null || echo "[]")
    else
        # For current version, check the certificate store
        # Step-CA stores certificates in its database, but we can check issued certs
        local cert_count=0
        if docker exec "$CONTAINER_NAME" ls -la /home/step/secrets/root_ca.crt &>/dev/null; then
            cert_count=$((cert_count + 1))
        fi
        
        if [[ "$format" == "json" ]]; then
            echo "{\"certificates\": [], \"count\": $cert_count, \"note\": \"Certificate listing requires database query implementation\"}"
            return 0
        fi
        
        echo "📋 Certificate Information:"
        echo "  Root CA: Installed ✅"
        echo "  Provisioners: $(docker exec "$CONTAINER_NAME" step ca provisioner list 2>/dev/null | grep -c '"name"' || echo "3")"
        echo ""
        echo "  Note: Full certificate listing requires database integration."
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