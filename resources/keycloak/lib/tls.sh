#!/usr/bin/env bash
set -euo pipefail

# TLS/HTTPS Configuration for Keycloak
# Provides certificate management and HTTPS configuration

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KEYCLOAK_LIB_DIR="${APP_ROOT}/resources/keycloak/lib"

# Source dependencies
source "${KEYCLOAK_LIB_DIR}/common.sh"

# Certificate paths
KEYCLOAK_CERT_DIR="${KEYCLOAK_CERT_DIR:-${HOME}/.keycloak/certificates}"
KEYCLOAK_CERT_FILE="${KEYCLOAK_CERT_FILE:-${KEYCLOAK_CERT_DIR}/tls.crt}"
KEYCLOAK_KEY_FILE="${KEYCLOAK_KEY_FILE:-${KEYCLOAK_CERT_DIR}/tls.key}"
KEYCLOAK_KEYSTORE="${KEYCLOAK_KEYSTORE:-${KEYCLOAK_CERT_DIR}/keystore.jks}"
KEYCLOAK_KEYSTORE_PASSWORD="${KEYCLOAK_KEYSTORE_PASSWORD:-changeit}"

# Generate self-signed certificate for development
keycloak::tls::generate_self_signed() {
    local domain="${1:-localhost}"
    local days="${2:-365}"
    
    log::info "Generating self-signed certificate for ${domain}"
    
    # Create certificate directory
    mkdir -p "${KEYCLOAK_CERT_DIR}"
    
    # Generate private key and certificate
    openssl req -x509 -newkey rsa:4096 -keyout "${KEYCLOAK_KEY_FILE}" -out "${KEYCLOAK_CERT_FILE}" \
        -days "${days}" -nodes -subj "/CN=${domain}/O=Vrooli/OU=Development" \
        -addext "subjectAltName=DNS:${domain},DNS:localhost,IP:127.0.0.1" 2>/dev/null
    
    if [[ -f "${KEYCLOAK_CERT_FILE}" ]] && [[ -f "${KEYCLOAK_KEY_FILE}" ]]; then
        log::success "Self-signed certificate generated"
        log::info "Certificate: ${KEYCLOAK_CERT_FILE}"
        log::info "Private key: ${KEYCLOAK_KEY_FILE}"
        
        # Create Java keystore for Keycloak
        keycloak::tls::create_keystore
        return 0
    else
        log::error "Failed to generate self-signed certificate"
        return 1
    fi
}

# Create Java keystore from certificate and key
keycloak::tls::create_keystore() {
    log::info "Creating Java keystore for Keycloak"
    
    # Convert to PKCS12 first
    openssl pkcs12 -export -in "${KEYCLOAK_CERT_FILE}" -inkey "${KEYCLOAK_KEY_FILE}" \
        -out "${KEYCLOAK_CERT_DIR}/keystore.p12" -name keycloak \
        -password "pass:${KEYCLOAK_KEYSTORE_PASSWORD}" 2>/dev/null
    
    # Convert PKCS12 to JKS
    keytool -importkeystore \
        -srckeystore "${KEYCLOAK_CERT_DIR}/keystore.p12" -srcstoretype PKCS12 \
        -srcstorepass "${KEYCLOAK_KEYSTORE_PASSWORD}" \
        -destkeystore "${KEYCLOAK_KEYSTORE}" -deststoretype JKS \
        -deststorepass "${KEYCLOAK_KEYSTORE_PASSWORD}" \
        -noprompt 2>/dev/null || true
    
    if [[ -f "${KEYCLOAK_KEYSTORE}" ]]; then
        log::success "Keystore created: ${KEYCLOAK_KEYSTORE}"
        return 0
    else
        log::warning "Failed to create JKS keystore, using PKCS12 format"
        mv "${KEYCLOAK_CERT_DIR}/keystore.p12" "${KEYCLOAK_KEYSTORE}"
        return 0
    fi
}

# Import existing certificate
keycloak::tls::import_certificate() {
    local cert_file="${1}"
    local key_file="${2}"
    
    if [[ ! -f "${cert_file}" ]]; then
        log::error "Certificate file not found: ${cert_file}"
        return 1
    fi
    
    if [[ ! -f "${key_file}" ]]; then
        log::error "Key file not found: ${key_file}"
        return 1
    fi
    
    log::info "Importing certificate from ${cert_file}"
    
    # Create certificate directory
    mkdir -p "${KEYCLOAK_CERT_DIR}"
    
    # Copy certificate and key
    cp "${cert_file}" "${KEYCLOAK_CERT_FILE}"
    cp "${key_file}" "${KEYCLOAK_KEY_FILE}"
    
    # Create keystore
    keycloak::tls::create_keystore
    
    log::success "Certificate imported successfully"
}

# Enable HTTPS in Keycloak
keycloak::tls::enable_https() {
    log::info "Enabling HTTPS for Keycloak"
    
    # Check if certificates exist
    if [[ ! -f "${KEYCLOAK_KEYSTORE}" ]]; then
        log::warning "No keystore found, generating self-signed certificate"
        keycloak::tls::generate_self_signed || return 1
    fi
    
    # Check if Keycloak is running
    if ! keycloak::is_running; then
        log::error "Keycloak is not running. Start it first with: vrooli resource keycloak manage start"
        return 1
    fi
    
    # Copy keystore to container
    docker cp "${KEYCLOAK_KEYSTORE}" "${KEYCLOAK_CONTAINER_NAME}:/opt/keycloak/conf/server.keystore"
    
    # Update Keycloak configuration for HTTPS
    docker exec "${KEYCLOAK_CONTAINER_NAME}" /opt/keycloak/bin/kc.sh config \
        --https-certificate-file=/opt/keycloak/conf/server.keystore \
        --https-certificate-key-file=/opt/keycloak/conf/server.keystore \
        --https-key-store-password="${KEYCLOAK_KEYSTORE_PASSWORD}" \
        --https-port=8443 \
        --https-protocols=TLSv1.3,TLSv1.2
    
    log::success "HTTPS configuration updated"
    log::warning "Restart Keycloak for changes to take effect: vrooli resource keycloak manage restart"
}

# Disable HTTPS (revert to HTTP only)
keycloak::tls::disable_https() {
    log::info "Disabling HTTPS for Keycloak"
    
    if ! keycloak::is_running; then
        log::error "Keycloak is not running"
        return 1
    fi
    
    # Reset to HTTP only configuration
    docker exec "${KEYCLOAK_CONTAINER_NAME}" /opt/keycloak/bin/kc.sh config \
        --http-enabled=true \
        --hostname-strict-https=false
    
    log::success "HTTPS disabled"
    log::warning "Restart Keycloak for changes to take effect: vrooli resource keycloak manage restart"
}

# Check certificate expiry
keycloak::tls::check_expiry() {
    if [[ ! -f "${KEYCLOAK_CERT_FILE}" ]]; then
        log::warning "No certificate found"
        return 1
    fi
    
    local expiry_date
    expiry_date=$(openssl x509 -enddate -noout -in "${KEYCLOAK_CERT_FILE}" | cut -d= -f2)
    local expiry_epoch
    expiry_epoch=$(date -d "${expiry_date}" +%s)
    local current_epoch
    current_epoch=$(date +%s)
    local days_remaining=$(( (expiry_epoch - current_epoch) / 86400 ))
    
    log::info "Certificate expiry: ${expiry_date}"
    
    if [[ ${days_remaining} -lt 0 ]]; then
        log::error "Certificate has expired!"
        return 1
    elif [[ ${days_remaining} -lt 30 ]]; then
        log::warning "Certificate expires in ${days_remaining} days"
    else
        log::success "Certificate valid for ${days_remaining} more days"
    fi
}

# Show certificate details
keycloak::tls::show_certificate() {
    if [[ ! -f "${KEYCLOAK_CERT_FILE}" ]]; then
        log::warning "No certificate found at ${KEYCLOAK_CERT_FILE}"
        return 1
    fi
    
    log::info "Certificate details:"
    openssl x509 -text -noout -in "${KEYCLOAK_CERT_FILE}" | grep -E "Subject:|Issuer:|Not Before:|Not After:|Subject Alternative Name:" -A1
}

# Renew certificate (for self-signed)
keycloak::tls::renew_certificate() {
    log::info "Renewing certificate"
    
    # Backup existing certificate
    if [[ -f "${KEYCLOAK_CERT_FILE}" ]]; then
        local backup_name="${KEYCLOAK_CERT_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
        cp "${KEYCLOAK_CERT_FILE}" "${backup_name}"
        log::info "Backed up existing certificate to ${backup_name}"
    fi
    
    # Generate new certificate
    keycloak::tls::generate_self_signed
    
    # Enable HTTPS with new certificate
    keycloak::tls::enable_https
}

# Main TLS command handler
keycloak::tls::main() {
    local subcommand="${1:-}"
    shift || true
    
    case "${subcommand}" in
        generate)
            keycloak::tls::generate_self_signed "$@"
            ;;
        import)
            keycloak::tls::import_certificate "$@"
            ;;
        enable)
            keycloak::tls::enable_https
            ;;
        disable)
            keycloak::tls::disable_https
            ;;
        check)
            keycloak::tls::check_expiry
            ;;
        show)
            keycloak::tls::show_certificate
            ;;
        renew)
            keycloak::tls::renew_certificate
            ;;
        *)
            log::error "Unknown TLS subcommand: ${subcommand}"
            echo "Usage: keycloak tls [generate|import|enable|disable|check|show|renew]"
            return 1
            ;;
    esac
}