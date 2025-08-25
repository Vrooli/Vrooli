#!/bin/bash

# Injection functions for Mail-in-a-Box resource
# Allows adding email accounts, domains, and aliases

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
MAILINABOX_INJECT_LIB_DIR="${APP_ROOT}/resources/mail-in-a-box/lib"

# Source dependencies
source "$MAILINABOX_INJECT_LIB_DIR/core.sh"

# Add email account
mailinabox_add_account() {
    local email="$1"
    local password="$2"
    
    if [[ -z "$email" ]] || [[ -z "$password" ]]; then
        format_error "Email and password are required"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        format_error "Mail-in-a-Box is not running"
        return 1
    fi
    
    format_info "Adding email account: $email"
    
    # Use Mail-in-a-Box management commands
    docker exec "$MAILINABOX_CONTAINER_NAME" \
        management/cli.py user add "$email" --password "$password"
    
    if [[ $? -eq 0 ]]; then
        format_success "Email account added: $email"
        return 0
    else
        format_error "Failed to add email account"
        return 1
    fi
}

# Add email alias
mailinabox_add_alias() {
    local alias="$1"
    local target="$2"
    
    if [[ -z "$alias" ]] || [[ -z "$target" ]]; then
        format_error "Alias and target email are required"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        format_error "Mail-in-a-Box is not running"
        return 1
    fi
    
    format_info "Adding email alias: $alias → $target"
    
    docker exec "$MAILINABOX_CONTAINER_NAME" \
        management/cli.py alias add "$alias" "$target"
    
    if [[ $? -eq 0 ]]; then
        format_success "Email alias added: $alias → $target"
        return 0
    else
        format_error "Failed to add email alias"
        return 1
    fi
}

# Add custom domain
mailinabox_add_domain() {
    local domain="$1"
    
    if [[ -z "$domain" ]]; then
        format_error "Domain is required"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        format_error "Mail-in-a-Box is not running"
        return 1
    fi
    
    format_info "Adding domain: $domain"
    
    # Add domain to Mail-in-a-Box
    docker exec "$MAILINABOX_CONTAINER_NAME" \
        management/cli.py domain add "$domain"
    
    if [[ $? -eq 0 ]]; then
        format_success "Domain added: $domain"
        format_info "DNS records need to be configured for this domain"
        return 0
    else
        format_error "Failed to add domain"
        return 1
    fi
}

# Import email data from file
mailinabox_inject_file() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        format_error "File not found: $file"
        return 1
    fi
    
    if ! mailinabox_is_running; then
        format_error "Mail-in-a-Box is not running"
        return 1
    fi
    
    # Parse file type and process accordingly
    case "$file" in
        *.json)
            format_info "Processing JSON email configuration..."
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
            format_info "Processing CSV email list..."
            # Process CSV file
            while IFS=, read -r email password; do
                if [[ -n "$email" ]] && [[ -n "$password" ]]; then
                    mailinabox_add_account "$email" "$password"
                fi
            done < "$file"
            ;;
        *)
            format_error "Unsupported file format. Use JSON or CSV"
            return 1
            ;;
    esac
    
    format_success "Email data imported from $file"
    return 0
}