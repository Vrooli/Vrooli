#!/bin/bash

# Content management functions for Mail-in-a-Box resource

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
MAILINABOX_CONTENT_LIB_DIR="${APP_ROOT}/resources/mail-in-a-box/lib"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
source "$MAILINABOX_CONTENT_LIB_DIR/core.sh"

# Add email account
mailinabox_add_account() {
    local email="${1:-}"
    local password="${2:-}"
    
    if [[ -z "$email" ]]; then
        log::error "Email address required"
        echo "Usage: $0 content add <email> [password]"
        return 1
    fi
    
    # Validate email format
    if ! validate_email "$email"; then
        return 1
    fi
    
    # Generate password if not provided
    if [[ -z "$password" ]]; then
        password=$(openssl rand -base64 12)
        log::info "Generated password: $password"
    fi
    
    # Check if container is running
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    # Add user using docker-mailserver's addmailuser command
    log::info "Adding email account: $email"
    if docker exec -i "$MAILINABOX_CONTAINER_NAME" addmailuser "$email" "$password" 2>/dev/null; then
        log::success "Email account created: $email"
        log::info "Password: $password"
        return 0
    else
        log::error "Failed to create email account"
        return 1
    fi
}

# List email accounts
mailinabox_list_accounts() {
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    log::header "📧 Email Accounts"
    
    # List users from docker-mailserver
    docker exec "$MAILINABOX_CONTAINER_NAME" ls -la /tmp/docker-mailserver/postfix-accounts.cf 2>/dev/null | grep -v "^total" || {
        log::warning "No accounts file found"
        return 1
    }
    
    # Try to read accounts file
    local accounts=$(docker exec "$MAILINABOX_CONTAINER_NAME" cat /tmp/docker-mailserver/postfix-accounts.cf 2>/dev/null | cut -d'|' -f1)
    
    if [[ -n "$accounts" ]]; then
        echo "$accounts"
    else
        log::info "No email accounts configured"
    fi
}

# Delete email account
mailinabox_delete_account() {
    local email="${1:-}"
    
    if [[ -z "$email" ]]; then
        log::error "Email address required"
        echo "Usage: $0 content remove <email>"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    log::info "Removing email account: $email"
    if docker exec "$MAILINABOX_CONTAINER_NAME" delmailuser "$email" 2>/dev/null; then
        log::success "Email account removed: $email"
        return 0
    else
        log::error "Failed to remove email account"
        return 1
    fi
}

# Add email alias
mailinabox_add_alias() {
    local alias="${1:-}"
    local target="${2:-}"
    
    if [[ -z "$alias" ]] || [[ -z "$target" ]]; then
        log::error "Both alias and target email addresses required"
        echo "Usage: $0 content add-alias <alias> <target>"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    log::info "Adding alias: $alias -> $target"
    if docker exec "$MAILINABOX_CONTAINER_NAME" addalias "$alias" "$target" 2>/dev/null; then
        log::success "Alias created: $alias -> $target"
        return 0
    else
        log::error "Failed to create alias"
        return 1
    fi
}

# Add custom domain
mailinabox_add_domain() {
    local domain="${1:-}"
    
    if [[ -z "$domain" ]]; then
        log::error "Domain name required"
        echo "Usage: $0 content add-domain <domain>"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    log::info "Adding domain: $domain"
    
    # For docker-mailserver, domains are automatically detected from email addresses
    # So we create a postmaster account for the domain
    local postmaster="postmaster@${domain}"
    local password=$(openssl rand -base64 12)
    
    if docker exec -i "$MAILINABOX_CONTAINER_NAME" addmailuser "$postmaster" "$password" 2>/dev/null; then
        log::success "Domain added with postmaster account: $postmaster"
        log::info "Postmaster password: $password"
        return 0
    else
        log::error "Failed to add domain"
        return 1
    fi
}

# Get mail server configuration
mailinabox_get_config() {
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    log::header "📧 Mail Server Configuration"
    echo ""
    echo "SMTP Server: ${MAILINABOX_BIND_ADDRESS}"
    echo "SMTP Port: ${MAILINABOX_PORT_SMTP}"
    echo "SMTP Submission: ${MAILINABOX_PORT_SUBMISSION}"
    echo "IMAP Port: ${MAILINABOX_PORT_IMAPS}"
    echo "POP3 Port: ${MAILINABOX_PORT_POP3S}"
    echo "Primary Hostname: ${MAILINABOX_PRIMARY_HOSTNAME}"
    echo ""
    echo "Security: TLS/SSL enabled"
    echo "Authentication: Required for sending"
    echo ""
}