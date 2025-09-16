#!/usr/bin/env bash
set -euo pipefail

# Let's Encrypt Certificate Automation for Keycloak
# Provides automated certificate acquisition and renewal via ACME protocol

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KEYCLOAK_LIB_DIR="${APP_ROOT}/resources/keycloak/lib"

# Source dependencies
source "${KEYCLOAK_LIB_DIR}/common.sh"
# Avoid circular dependency - only source tls.sh if not already loaded
if ! declare -f keycloak::tls::create_keystore &>/dev/null; then
    source "${KEYCLOAK_LIB_DIR}/tls.sh"
fi
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Let's Encrypt configuration
LETSENCRYPT_DIR="${LETSENCRYPT_DIR:-${HOME}/.keycloak/letsencrypt}"
LETSENCRYPT_CONFIG="${LETSENCRYPT_CONFIG:-${LETSENCRYPT_DIR}/cli.ini}"
LETSENCRYPT_WORK_DIR="${LETSENCRYPT_WORK_DIR:-${LETSENCRYPT_DIR}/work}"
LETSENCRYPT_LOG_DIR="${LETSENCRYPT_LOG_DIR:-${LETSENCRYPT_DIR}/logs}"
LETSENCRYPT_STAGING="${LETSENCRYPT_STAGING:-true}"

# ACME challenge configuration
ACME_CHALLENGE_DIR="${ACME_CHALLENGE_DIR:-${HOME}/.keycloak/acme-challenge}"
ACME_CHALLENGE_PORT="${ACME_CHALLENGE_PORT:-8080}"

# Check if certbot is installed
keycloak::letsencrypt::check_certbot() {
    if ! command -v certbot &> /dev/null; then
        log::warning "Certbot not installed. Installing..."
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y certbot
        elif command -v yum &> /dev/null; then
            sudo yum install -y certbot
        elif command -v dnf &> /dev/null; then
            sudo dnf install -y certbot
        else
            log::error "Cannot install certbot automatically. Please install manually."
            return 1
        fi
    fi
    return 0
}

# Initialize Let's Encrypt configuration
keycloak::letsencrypt::init() {
    local email="${1:-}"
    
    if [[ -z "${email}" ]]; then
        log::error "Email address required for Let's Encrypt registration"
        return 1
    fi
    
    log::info "Initializing Let's Encrypt configuration"
    
    # Create directories
    mkdir -p "${LETSENCRYPT_DIR}" "${LETSENCRYPT_WORK_DIR}" "${LETSENCRYPT_LOG_DIR}" "${ACME_CHALLENGE_DIR}"
    
    # Create certbot configuration
    cat > "${LETSENCRYPT_CONFIG}" <<EOF
# Certbot configuration for Keycloak
email = ${email}
agree-tos = True
non-interactive = True
work-dir = ${LETSENCRYPT_WORK_DIR}
logs-dir = ${LETSENCRYPT_LOG_DIR}
EOF
    
    if [[ "${LETSENCRYPT_STAGING}" == "true" ]]; then
        echo "staging = True" >> "${LETSENCRYPT_CONFIG}"
        log::warning "Using Let's Encrypt staging environment (for testing)"
    fi
    
    log::success "Let's Encrypt initialized with email: ${email}"
}

# Request certificate from Let's Encrypt
keycloak::letsencrypt::request_certificate() {
    local domain="${1:-}"
    local email="${2:-}"
    
    if [[ -z "${domain}" ]]; then
        log::error "Domain name required for certificate request"
        return 1
    fi
    
    # Initialize if needed
    if [[ ! -f "${LETSENCRYPT_CONFIG}" ]]; then
        if [[ -z "${email}" ]]; then
            log::error "Email required for first certificate request"
            return 1
        fi
        keycloak::letsencrypt::init "${email}"
    fi
    
    # Check certbot
    keycloak::letsencrypt::check_certbot || return 1
    
    log::info "Requesting certificate for domain: ${domain}"
    
    # Prepare command
    local certbot_cmd="certbot certonly"
    certbot_cmd+=" --config ${LETSENCRYPT_CONFIG}"
    certbot_cmd+=" --webroot"
    certbot_cmd+=" --webroot-path ${ACME_CHALLENGE_DIR}"
    certbot_cmd+=" -d ${domain}"
    
    # Add staging flag if enabled
    if [[ "${LETSENCRYPT_STAGING}" == "true" ]]; then
        certbot_cmd+=" --staging"
    fi
    
    # Start temporary web server for ACME challenge
    log::info "Starting ACME challenge server on port ${ACME_CHALLENGE_PORT}"
    python3 -m http.server ${ACME_CHALLENGE_PORT} --directory "${ACME_CHALLENGE_DIR}" &
    local server_pid=$!
    
    # Wait for server to start
    sleep 2
    
    # Request certificate
    if eval "${certbot_cmd}"; then
        log::success "Certificate obtained successfully"
        
        # Copy certificates to Keycloak directory
        local cert_path="/etc/letsencrypt/live/${domain}"
        if [[ -d "${cert_path}" ]]; then
            sudo cp "${cert_path}/fullchain.pem" "${KEYCLOAK_CERT_FILE}"
            sudo cp "${cert_path}/privkey.pem" "${KEYCLOAK_KEY_FILE}"
            sudo chown $(whoami):$(whoami) "${KEYCLOAK_CERT_FILE}" "${KEYCLOAK_KEY_FILE}"
            
            # Create Java keystore
            keycloak::tls::create_keystore
            
            log::success "Certificates installed to Keycloak"
        fi
    else
        log::error "Failed to obtain certificate"
        kill ${server_pid} 2>/dev/null
        return 1
    fi
    
    # Stop ACME challenge server
    kill ${server_pid} 2>/dev/null
    
    return 0
}

# Renew certificates
keycloak::letsencrypt::renew_certificates() {
    log::info "Checking for certificate renewals"
    
    keycloak::letsencrypt::check_certbot || return 1
    
    # Start ACME challenge server
    python3 -m http.server ${ACME_CHALLENGE_PORT} --directory "${ACME_CHALLENGE_DIR}" &
    local server_pid=$!
    sleep 2
    
    # Attempt renewal
    if certbot renew --config "${LETSENCRYPT_CONFIG}" --webroot-path "${ACME_CHALLENGE_DIR}"; then
        log::success "Certificates renewed successfully"
        
        # Copy renewed certificates
        for cert_dir in /etc/letsencrypt/live/*/; do
            if [[ -d "${cert_dir}" ]]; then
                local domain=$(basename "${cert_dir}")
                log::info "Updating certificate for ${domain}"
                
                sudo cp "${cert_dir}/fullchain.pem" "${KEYCLOAK_CERT_FILE}"
                sudo cp "${cert_dir}/privkey.pem" "${KEYCLOAK_KEY_FILE}"
                sudo chown $(whoami):$(whoami) "${KEYCLOAK_CERT_FILE}" "${KEYCLOAK_KEY_FILE}"
                
                # Recreate keystore
                keycloak::tls::create_keystore
                
                # Restart Keycloak to apply new certificate
                log::info "Restarting Keycloak to apply new certificate"
                docker restart vrooli-keycloak
                
                break # Only handle first domain for now
            fi
        done
    else
        log::warning "No certificates require renewal"
    fi
    
    # Stop ACME challenge server
    kill ${server_pid} 2>/dev/null
    
    return 0
}

# Setup automated renewal via cron
keycloak::letsencrypt::setup_auto_renewal() {
    local schedule="${1:-daily}"  # daily, weekly, monthly
    
    log::info "Setting up automated certificate renewal (${schedule})"
    
    # Create renewal script
    local renewal_script="${LETSENCRYPT_DIR}/auto-renew.sh"
    cat > "${renewal_script}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

# Auto-renewal script for Keycloak Let's Encrypt certificates
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/resources/keycloak/lib/letsencrypt.sh"

# Log renewal attempt
echo "[$(date)] Starting certificate renewal check" >> "${LETSENCRYPT_LOG_DIR}/auto-renewal.log"

# Perform renewal
if keycloak::letsencrypt::renew_certificates; then
    echo "[$(date)] Renewal completed successfully" >> "${LETSENCRYPT_LOG_DIR}/auto-renewal.log"
else
    echo "[$(date)] Renewal failed or not needed" >> "${LETSENCRYPT_LOG_DIR}/auto-renewal.log"
fi
EOF
    
    chmod +x "${renewal_script}"
    
    # Determine cron schedule
    local cron_schedule
    case "${schedule}" in
        daily)
            cron_schedule="0 2 * * *"  # 2 AM daily
            ;;
        weekly)
            cron_schedule="0 2 * * 0"  # 2 AM Sunday
            ;;
        monthly)
            cron_schedule="0 2 1 * *"  # 2 AM first of month
            ;;
        *)
            log::error "Invalid schedule. Use: daily, weekly, or monthly"
            return 1
            ;;
    esac
    
    # Add to crontab
    local cron_entry="${cron_schedule} ${renewal_script}"
    
    # Check if already in crontab
    if crontab -l 2>/dev/null | grep -q "${renewal_script}"; then
        log::warning "Auto-renewal already configured in crontab"
    else
        (crontab -l 2>/dev/null; echo "${cron_entry}") | crontab -
        log::success "Auto-renewal configured with ${schedule} schedule"
        log::info "Cron entry: ${cron_entry}"
    fi
    
    return 0
}

# Remove auto-renewal from cron
keycloak::letsencrypt::remove_auto_renewal() {
    log::info "Removing automated certificate renewal"
    
    local renewal_script="${LETSENCRYPT_DIR}/auto-renew.sh"
    
    # Remove from crontab
    if crontab -l 2>/dev/null | grep -q "${renewal_script}"; then
        crontab -l | grep -v "${renewal_script}" | crontab -
        log::success "Auto-renewal removed from crontab"
    else
        log::warning "No auto-renewal entry found in crontab"
    fi
    
    return 0
}

# Check certificate status
keycloak::letsencrypt::check_status() {
    log::info "Checking Let's Encrypt certificate status"
    
    if ! command -v certbot &> /dev/null; then
        log::warning "Certbot not installed"
        return 1
    fi
    
    # Check certificates
    if certbot certificates --config "${LETSENCRYPT_CONFIG}" 2>/dev/null; then
        return 0
    else
        log::warning "No certificates found or certbot not configured"
        return 1
    fi
}

# Revoke certificate
keycloak::letsencrypt::revoke_certificate() {
    local domain="${1:-}"
    local reason="${2:-unspecified}"
    
    if [[ -z "${domain}" ]]; then
        log::error "Domain name required for certificate revocation"
        return 1
    fi
    
    log::warning "Revoking certificate for domain: ${domain}"
    
    keycloak::letsencrypt::check_certbot || return 1
    
    local cert_path="/etc/letsencrypt/live/${domain}/cert.pem"
    
    if [[ -f "${cert_path}" ]]; then
        if certbot revoke --cert-path "${cert_path}" --reason "${reason}"; then
            log::success "Certificate revoked successfully"
            
            # Remove certificate files
            sudo rm -rf "/etc/letsencrypt/live/${domain}"
            sudo rm -rf "/etc/letsencrypt/archive/${domain}"
            sudo rm -f "/etc/letsencrypt/renewal/${domain}.conf"
            
            return 0
        else
            log::error "Failed to revoke certificate"
            return 1
        fi
    else
        log::error "Certificate not found for domain: ${domain}"
        return 1
    fi
}

# Test ACME challenge
keycloak::letsencrypt::test_challenge() {
    local port="${1:-${ACME_CHALLENGE_PORT}}"
    
    log::info "Testing ACME challenge setup on port ${port}"
    
    # Check if port is available
    if lsof -i:${port} &>/dev/null; then
        log::warning "Port ${port} is already in use, trying alternative port"
        port=$((port + 1))
        while lsof -i:${port} &>/dev/null && [ ${port} -lt 65535 ]; do
            port=$((port + 1))
        done
        log::info "Using port ${port} for test"
    fi
    
    # Create test file
    mkdir -p "${ACME_CHALLENGE_DIR}/.well-known/acme-challenge"
    echo "test-challenge-response" > "${ACME_CHALLENGE_DIR}/.well-known/acme-challenge/test"
    
    # Start server with better error handling
    python3 -m http.server ${port} --directory "${ACME_CHALLENGE_DIR}" 2>/dev/null &
    local server_pid=$!
    
    # Check if server started
    sleep 2
    if ! kill -0 ${server_pid} 2>/dev/null; then
        log::error "Failed to start test server on port ${port}"
        return 1
    fi
    
    # Test access
    if curl -sf "http://localhost:${port}/.well-known/acme-challenge/test" | grep -q "test-challenge-response"; then
        log::success "ACME challenge test successful on port ${port}"
        kill ${server_pid} 2>/dev/null
        return 0
    else
        log::error "ACME challenge test failed"
        kill ${server_pid} 2>/dev/null
        return 1
    fi
}

# Main CLI handler
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        init)
            keycloak::letsencrypt::init "${2:-}"
            ;;
        request)
            keycloak::letsencrypt::request_certificate "${2:-}" "${3:-}"
            ;;
        renew)
            keycloak::letsencrypt::renew_certificates
            ;;
        auto-renew)
            keycloak::letsencrypt::setup_auto_renewal "${2:-daily}"
            ;;
        remove-auto-renew)
            keycloak::letsencrypt::remove_auto_renewal
            ;;
        status)
            keycloak::letsencrypt::check_status
            ;;
        revoke)
            keycloak::letsencrypt::revoke_certificate "${2:-}" "${3:-unspecified}"
            ;;
        test)
            keycloak::letsencrypt::test_challenge "${2:-}"
            ;;
        *)
            echo "Usage: $0 {init|request|renew|auto-renew|remove-auto-renew|status|revoke|test}"
            exit 1
            ;;
    esac
fi