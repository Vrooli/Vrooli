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

# Auto-configuration support for email clients

# Generate autoconfig XML for a domain
mailinabox_generate_autoconfig() {
    local domain="${1:-$MAILINABOX_PRIMARY_HOSTNAME}"
    local server_hostname="${2:-mail.$domain}"
    
    cat <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<clientConfig version="1.1">
  <emailProvider id="$domain">
    <domain>$domain</domain>
    <displayName>Mail-in-a-Box ($domain)</displayName>
    <displayShortName>$domain</displayShortName>
    
    <!-- IMAP Configuration -->
    <incomingServer type="imap">
      <hostname>$server_hostname</hostname>
      <port>993</port>
      <socketType>SSL</socketType>
      <authentication>password-cleartext</authentication>
      <username>%EMAILADDRESS%</username>
    </incomingServer>
    
    <!-- Alternative: POP3 Configuration -->
    <incomingServer type="pop3">
      <hostname>$server_hostname</hostname>
      <port>995</port>
      <socketType>SSL</socketType>
      <authentication>password-cleartext</authentication>
      <username>%EMAILADDRESS%</username>
      <leave_messages_on_server>true</leave_messages_on_server>
    </incomingServer>
    
    <!-- SMTP Configuration -->
    <outgoingServer type="smtp">
      <hostname>$server_hostname</hostname>
      <port>587</port>
      <socketType>STARTTLS</socketType>
      <authentication>password-cleartext</authentication>
      <username>%EMAILADDRESS%</username>
    </outgoingServer>
    
    <!-- Documentation -->
    <documentation url="http://$server_hostname">
      <descr lang="en">Mail-in-a-Box Settings</descr>
    </documentation>
  </emailProvider>
</clientConfig>
EOF
}

# Generate autodiscover XML (Microsoft Outlook)
mailinabox_generate_autodiscover() {
    local email="$1"
    local domain="${email#*@}"
    local server_hostname="${2:-mail.$domain}"
    
    cat <<EOF
<?xml version="1.0" encoding="utf-8"?>
<Autodiscover xmlns="http://schemas.microsoft.com/exchange/autodiscover/responseschema/2006">
  <Response xmlns="http://schemas.microsoft.com/exchange/autodiscover/outlook/responseschema/2006a">
    <User>
      <DisplayName>$email</DisplayName>
    </User>
    <Account>
      <AccountType>email</AccountType>
      <Action>settings</Action>
      <Protocol>
        <Type>IMAP</Type>
        <Server>$server_hostname</Server>
        <Port>993</Port>
        <LoginName>$email</LoginName>
        <SSL>on</SSL>
        <AuthRequired>on</AuthRequired>
      </Protocol>
      <Protocol>
        <Type>POP3</Type>
        <Server>$server_hostname</Server>
        <Port>995</Port>
        <LoginName>$email</LoginName>
        <SSL>on</SSL>
        <AuthRequired>on</AuthRequired>
      </Protocol>
      <Protocol>
        <Type>SMTP</Type>
        <Server>$server_hostname</Server>
        <Port>587</Port>
        <LoginName>$email</LoginName>
        <SSL>on</SSL>
        <AuthRequired>on</AuthRequired>
      </Protocol>
    </Account>
  </Response>
</Autodiscover>
EOF
}

# Setup autoconfig files for a domain
mailinabox_setup_autoconfig() {
    local domain="$1"
    
    if [[ -z "$domain" ]]; then
        log::error "Domain required"
        return 1
    fi
    
    log::info "Setting up autoconfig for domain: $domain"
    
    # Create autoconfig directory structure
    local autoconfig_dir="${MAILINABOX_DATA_DIR:-/var/lib/mailinabox}/autoconfig"
    mkdir -p "$autoconfig_dir/$domain/.well-known"
    
    # Generate Thunderbird/Evolution autoconfig
    mailinabox_generate_autoconfig "$domain" > "$autoconfig_dir/$domain/.well-known/autoconfig.xml"
    
    # Generate Microsoft autodiscover
    mkdir -p "$autoconfig_dir/$domain/autodiscover"
    mailinabox_generate_autodiscover "user@$domain" > "$autoconfig_dir/$domain/autodiscover/autodiscover.xml"
    
    log::success "Autoconfig files generated for $domain"
    echo "Files created at: $autoconfig_dir/$domain/"
    echo ""
    echo "To enable autoconfig, serve these URLs:"
    echo "  http://autoconfig.$domain/mail/config-v1.1.xml"
    echo "  http://$domain/.well-known/autoconfig.xml"
    echo "  https://autodiscover.$domain/autodiscover/autodiscover.xml"
    
    return 0
}

# Test autoconfig for a domain
mailinabox_test_autoconfig() {
    local domain="${1:-$MAILINABOX_PRIMARY_HOSTNAME}"
    
    log::info "Testing autoconfig for domain: $domain"
    
    # Check if autoconfig files exist
    local autoconfig_dir="${MAILINABOX_DATA_DIR:-/var/lib/mailinabox}/autoconfig"
    
    if [[ -f "$autoconfig_dir/$domain/.well-known/autoconfig.xml" ]]; then
        log::success "Thunderbird autoconfig file exists"
    else
        log_warning "Thunderbird autoconfig file not found"
    fi
    
    if [[ -f "$autoconfig_dir/$domain/autodiscover/autodiscover.xml" ]]; then
        log::success "Outlook autodiscover file exists"
    else
        log_warning "Outlook autodiscover file not found"
    fi
    
    # Test DNS records
    echo ""
    echo "DNS autoconfig records:"
    dig +short A "autoconfig.$domain" || echo "  autoconfig.$domain not configured"
    dig +short A "autodiscover.$domain" || echo "  autodiscover.$domain not configured"
    
    return 0
}

# Export functions
export -f mailinabox_generate_autoconfig
export -f mailinabox_generate_autodiscover
export -f mailinabox_setup_autoconfig
export -f mailinabox_test_autoconfig