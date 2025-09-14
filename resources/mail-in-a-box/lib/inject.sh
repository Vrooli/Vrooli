#!/bin/bash

# Injection functions for Mail-in-a-Box resource
# Allows adding email accounts, domains, and aliases

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
MAILINABOX_INJECT_LIB_DIR="${APP_ROOT}/resources/mail-in-a-box/lib"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
source "$MAILINABOX_INJECT_LIB_DIR/core.sh"

# Add email account
mailinabox_add_account() {
    local email="$1"
    local password="$2"
    
    if [[ -z "$email" ]] || [[ -z "$password" ]]; then
        log::error "Email and password are required"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    log::info "Adding email account: $email"
    
    # Use docker-mailserver setup commands
    docker exec "$MAILINABOX_CONTAINER_NAME" \
        setup email add "$email" "$password"
    
    if [[ $? -eq 0 ]]; then
        log::success "Email account added: $email"
        return 0
    else
        log::error "Failed to add email account"
        return 1
    fi
}

# Add email alias
mailinabox_add_alias() {
    local alias="$1"
    local target="$2"
    
    if [[ -z "$alias" ]] || [[ -z "$target" ]]; then
        log::error "Alias and target email are required"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    log::info "Adding email alias: $alias → $target"
    
    # Use docker-mailserver setup commands
    docker exec "$MAILINABOX_CONTAINER_NAME" \
        setup alias add "$alias" "$target"
    
    if [[ $? -eq 0 ]]; then
        log::success "Email alias added: $alias → $target"
        return 0
    else
        log::error "Failed to add email alias"
        return 1
    fi
}

# Add custom domain
mailinabox_add_domain() {
    local domain="$1"
    
    if [[ -z "$domain" ]]; then
        log::error "Domain is required"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    log::info "Adding domain: $domain"
    
    # docker-mailserver doesn't have explicit domain add, domains are auto-detected from email addresses
    log::info "Domains are automatically configured when adding email accounts in docker-mailserver"
    log::success "Domain configuration noted: $domain"
    log::info "DNS records need to be configured for this domain"
    return 0
}

# Import email data from file
mailinabox_inject_file() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        log::error "Mail-in-a-Box is not running"
        return 1
    fi
    
    # Parse file type and process accordingly
    case "$file" in
        *.json)
            log::info "Processing JSON email configuration..."
            # Parse JSON and add accounts/aliases
            while IFS= read -r line; do
                if [[ "$line" =~ \"email\":\"([^\"]+)\" ]]; then
                    local email="${BASH_REMATCH[1]}"
                    if [[ "$line" =~ \"password\":\"([^\"]+)\" ]]; then
                        local password="${BASH_REMATCH[1]}"
                        mailinabox_add_account "$email" "$password"
                    fi
                fi
            done < "$file"
            ;;
        *.csv)
            log::info "Processing CSV email list..."
            # Process CSV file
            while IFS=, read -r email password; do
                if [[ -n "$email" ]] && [[ -n "$password" ]]; then
                    mailinabox_add_account "$email" "$password"
                fi
            done < "$file"
            ;;
        *)
            log::error "Unsupported file format. Use JSON or CSV"
            return 1
            ;;
    esac
    
    log::success "Email data imported from $file"
    return 0
}