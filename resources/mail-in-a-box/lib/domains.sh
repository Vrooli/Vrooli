#!/bin/bash

# Source log functions
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || {
    # Fallback log functions if log.sh not available
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
}

source "$(dirname "${BASH_SOURCE[0]}")/core.sh"

# Multi-domain management for Mail-in-a-Box

# Add a new domain to the mail server
mailinabox_add_domain() {
    local domain="$1"
    
    if [[ -z "$domain" ]]; then
        log::error "Domain name required"
        echo "Usage: vrooli resource mail-in-a-box content add-domain example.com"
        return 1
    fi
    
    # Validate domain format
    if ! echo "$domain" | grep -qE '^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$'; then
        log::error "Invalid domain format: $domain"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    log::info "Adding domain: $domain"
    
    # Add domain to postfix configuration
    docker exec "$MAILINABOX_CONTAINER_NAME" bash -c "
        # Add to virtual mailbox domains
        echo '$domain' >> /tmp/docker-mailserver/postfix-virtual-mailbox-domains.cf
        
        # Sort and remove duplicates
        sort -u /tmp/docker-mailserver/postfix-virtual-mailbox-domains.cf -o /tmp/docker-mailserver/postfix-virtual-mailbox-domains.cf
        
        # Reload postfix
        postfix reload
    " || {
        log::error "Failed to add domain to postfix"
        return 1
    }
    
    # Add DNS check info
    log::success "Domain $domain added successfully"
    echo ""
    echo "⚠️  DNS Configuration Required:"
    echo "Add these DNS records for $domain:"
    echo ""
    echo "MX Record:"
    echo "  $domain. 10 mail.$domain."
    echo ""
    echo "A Records:"
    echo "  mail.$domain. → [Your Server IP]"
    echo ""
    echo "SPF Record (TXT):"
    echo "  $domain. → \"v=spf1 mx -all\""
    echo ""
    echo "DKIM Record (TXT):"
    echo "  Run: vrooli resource mail-in-a-box content get-dkim $domain"
    echo ""
    
    return 0
}

# List configured domains
mailinabox_list_domains() {
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    log::info "Configured email domains:"
    
    docker exec "$MAILINABOX_CONTAINER_NAME" bash -c "
        if [[ -f /tmp/docker-mailserver/postfix-virtual-mailbox-domains.cf ]]; then
            cat /tmp/docker-mailserver/postfix-virtual-mailbox-domains.cf | sort -u
        else
            echo 'No additional domains configured'
            echo 'Default domain: ${MAILINABOX_PRIMARY_HOSTNAME}'
        fi
    " || {
        log::error "Failed to list domains"
        return 1
    }
    
    return 0
}

# Remove a domain
mailinabox_remove_domain() {
    local domain="$1"
    
    if [[ -z "$domain" ]]; then
        log::error "Domain name required"
        echo "Usage: vrooli resource mail-in-a-box content remove-domain example.com"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    log::info "Removing domain: $domain"
    
    # Remove domain from postfix configuration
    docker exec "$MAILINABOX_CONTAINER_NAME" bash -c "
        if [[ -f /tmp/docker-mailserver/postfix-virtual-mailbox-domains.cf ]]; then
            # Remove the domain
            grep -v '^$domain\$' /tmp/docker-mailserver/postfix-virtual-mailbox-domains.cf > /tmp/docker-mailserver/postfix-virtual-mailbox-domains.cf.tmp
            mv /tmp/docker-mailserver/postfix-virtual-mailbox-domains.cf.tmp /tmp/docker-mailserver/postfix-virtual-mailbox-domains.cf
            
            # Reload postfix
            postfix reload
            
            echo 'Domain removed successfully'
        else
            echo 'Domain configuration file not found'
        fi
    " || {
        log::error "Failed to remove domain"
        return 1
    }
    
    log::success "Domain $domain removed"
    return 0
}

# Get DKIM key for a domain
mailinabox_get_dkim() {
    local domain="${1:-$MAILINABOX_PRIMARY_HOSTNAME}"
    
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    log::info "DKIM key for domain: $domain"
    
    docker exec "$MAILINABOX_CONTAINER_NAME" bash -c "
        DKIM_KEY_FILE='/tmp/docker-mailserver/opendkim/keys/${domain}/mail.txt'
        if [[ -f \"\$DKIM_KEY_FILE\" ]]; then
            echo 'DKIM TXT Record:'
            cat \"\$DKIM_KEY_FILE\" | tr -d '\n' | sed 's/.*\"v=DKIM1/v=DKIM1/' | sed 's/\".*\$//'
        else
            echo 'DKIM key not found for domain: $domain'
            echo 'Generate with: docker exec $MAILINABOX_CONTAINER_NAME setup config dkim'
        fi
    " || {
        log::error "Failed to get DKIM key"
        return 1
    }
    
    return 0
}

# Verify domain configuration
mailinabox_verify_domain() {
    local domain="${1:-$MAILINABOX_PRIMARY_HOSTNAME}"
    
    log::info "Verifying domain configuration for: $domain"
    echo ""
    
    # Check MX records
    echo "MX Records:"
    dig +short MX "$domain" || echo "  No MX records found"
    echo ""
    
    # Check A record for mail subdomain
    echo "Mail server A record:"
    dig +short A "mail.$domain" || echo "  No A record found for mail.$domain"
    echo ""
    
    # Check SPF record
    echo "SPF Record:"
    dig +short TXT "$domain" | grep -E "v=spf1" || echo "  No SPF record found"
    echo ""
    
    # Check DMARC record
    echo "DMARC Record:"
    dig +short TXT "_dmarc.$domain" || echo "  No DMARC record found (optional)"
    echo ""
    
    # Check reverse DNS
    local server_ip
    server_ip=$(dig +short A "mail.$domain" | head -1)
    if [[ -n "$server_ip" ]]; then
        echo "Reverse DNS for $server_ip:"
        dig +short -x "$server_ip" || echo "  No PTR record found"
    fi
    
    return 0
}

# Export functions for CLI use
export -f mailinabox_add_domain
export -f mailinabox_list_domains
export -f mailinabox_remove_domain
export -f mailinabox_get_dkim
export -f mailinabox_verify_domain