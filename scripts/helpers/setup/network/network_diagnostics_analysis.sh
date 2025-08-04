#!/usr/bin/env bash
# Network analysis functions for diagnostics
# Handles TLS analysis, IPv4/IPv6 comparisons, and detailed debugging
set -eo pipefail

# Load dependencies
NETWORK_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
SETUP_DIR=$(dirname "${NETWORK_DIR}")
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"

# Analyze TLS handshake
network_diagnostics_analysis::analyze_tls_handshake() {
    local domain="${1:-github.com}"
    log::info "Analyzing TLS handshake to $domain..."
    
    if ! command -v openssl >/dev/null 2>&1; then
        log::warning "  ⚠️  OpenSSL not available for TLS analysis"
        return 1
    fi
    
    # Test TLS versions
    local tls_versions=("tls1" "tls1_1" "tls1_2" "tls1_3")
    local working_versions=()
    
    for version in "${tls_versions[@]}"; do
        if timeout 5 openssl s_client -connect "$domain:443" -"$version" </dev/null 2>/dev/null | grep -q "Verify return code: 0"; then
            working_versions+=("$version")
            log::success "  ✓ $version: Supported"
        else
            log::info "  ✗ $version: Not supported or failed"
        fi
    done
    
    if [[ ${#working_versions[@]} -eq 0 ]]; then
        log::error "  ✗ No TLS versions work with $domain"
        
        # Try to get more details about the failure
        log::info "  → Attempting detailed SSL diagnostics..."
        timeout 5 openssl s_client -connect "$domain:443" -showcerts </dev/null 2>&1 | head -20 || true
        return 1
    else
        log::success "  ✓ Working TLS versions: ${working_versions[*]}"
    fi
    
    # Check cipher suites
    log::info "  → Checking cipher suites..."
    local cipher_output
    cipher_output=$(timeout 5 openssl s_client -connect "$domain:443" -cipher 'ECDHE+AESGCM:ECDHE+AES256:!aNULL:!MD5:!DSS' </dev/null 2>&1 || echo "")
    
    if echo "$cipher_output" | grep -q "Cipher    :"; then
        local cipher=$(echo "$cipher_output" | grep "Cipher    :" | awk '{print $3}')
        log::info "  → Negotiated cipher: $cipher"
    fi
    
    # Check certificate chain
    log::info "  → Checking certificate chain..."
    local cert_output
    cert_output=$(timeout 5 openssl s_client -connect "$domain:443" -showcerts </dev/null 2>&1 || echo "")
    
    if echo "$cert_output" | grep -q "Certificate chain"; then
        local cert_count=$(echo "$cert_output" | grep -c "BEGIN CERTIFICATE" || echo "0")
        log::info "  → Certificate chain length: $cert_count"
    fi
    
    return 0
}

# Check IPv4 vs IPv6 connectivity
network_diagnostics_analysis::check_ipv4_vs_ipv6() {
    local domain="${1:-google.com}"
    log::info "Checking IPv4 vs IPv6 connectivity to $domain..."
    
    local ipv4_works=false
    local ipv6_works=false
    
    # Test IPv4
    if ping -4 -c 1 -W 2 "$domain" >/dev/null 2>&1; then
        ipv4_works=true
        log::success "  ✓ IPv4 connectivity works"
    else
        log::error "  ✗ IPv4 connectivity failed"
    fi
    
    # Test IPv6
    if ping -6 -c 1 -W 2 "$domain" >/dev/null 2>&1; then
        ipv6_works=true
        log::success "  ✓ IPv6 connectivity works"
    else
        log::info "  ✗ IPv6 connectivity failed (may not be available)"
    fi
    
    # Test DNS resolution
    log::info "  → Checking DNS resolution..."
    
    # IPv4 DNS
    local ipv4_addr
    ipv4_addr=$(getent ahostsv4 "$domain" 2>/dev/null | head -1 | awk '{print $1}' || echo "")
    if [[ -n "$ipv4_addr" ]]; then
        log::info "  → IPv4 address: $ipv4_addr"
    fi
    
    # IPv6 DNS
    local ipv6_addr
    ipv6_addr=$(getent ahostsv6 "$domain" 2>/dev/null | head -1 | awk '{print $1}' || echo "")
    if [[ -n "$ipv6_addr" ]] && [[ "$ipv6_addr" != "$ipv4_addr" ]]; then
        log::info "  → IPv6 address: $ipv6_addr"
    fi
    
    # Compare HTTP/HTTPS over IPv4 vs IPv6
    if command -v curl >/dev/null 2>&1; then
        log::info "  → Testing HTTP/HTTPS over different IP versions..."
        
        # IPv4 HTTPS
        if timeout 5 curl -4 -s -o /dev/null -w "%{http_code}" "https://$domain" 2>/dev/null | grep -q "200\|301\|302"; then
            log::success "  ✓ HTTPS over IPv4 works"
        else
            log::error "  ✗ HTTPS over IPv4 failed"
        fi
        
        # IPv6 HTTPS (only if IPv6 is available)
        if [[ -n "$ipv6_addr" ]]; then
            if timeout 5 curl -6 -s -o /dev/null -w "%{http_code}" "https://$domain" 2>/dev/null | grep -q "200\|301\|302"; then
                log::success "  ✓ HTTPS over IPv6 works"
            else
                log::info "  ✗ HTTPS over IPv6 failed"
            fi
        fi
    fi
    
    # Provide recommendation
    if [[ "$ipv4_works" == "true" ]] && [[ "$ipv6_works" == "false" ]]; then
        log::info "  → Recommendation: Use IPv4-only mode for better compatibility"
    elif [[ "$ipv4_works" == "false" ]] && [[ "$ipv6_works" == "true" ]]; then
        log::warning "  ⚠️  Only IPv6 works - may cause compatibility issues"
    fi
    
    return 0
}

# Check IP preference
network_diagnostics_analysis::check_ip_preference() {
    log::info "Checking system IP version preference..."
    
    # Check gai.conf (getaddrinfo configuration)
    if [[ -f /etc/gai.conf ]]; then
        local precedence
        precedence=$(grep -E "^precedence" /etc/gai.conf 2>/dev/null | head -5 || echo "")
        if [[ -n "$precedence" ]]; then
            log::info "  → IP address selection order (gai.conf):"
            echo "$precedence" | while read -r line; do
                log::info "    $line"
            done
        fi
    fi
    
    # Check if IPv6 is disabled
    local ipv6_disabled=false
    if [[ -f /proc/sys/net/ipv6/conf/all/disable_ipv6 ]]; then
        local disable_value
        disable_value=$(cat /proc/sys/net/ipv6/conf/all/disable_ipv6 2>/dev/null || echo "0")
        if [[ "$disable_value" == "1" ]]; then
            ipv6_disabled=true
            log::warning "  ⚠️  IPv6 is disabled system-wide"
        else
            log::info "  → IPv6 is enabled"
        fi
    fi
    
    # Check default route preference
    local default_route
    default_route=$(ip route show default 2>/dev/null | head -1 || echo "")
    if [[ -n "$default_route" ]]; then
        log::info "  → Default route: $default_route"
    fi
    
    return 0
}

# Check time synchronization
network_diagnostics_analysis::check_time_sync() {
    log::info "Checking time synchronization..."
    
    # Check if system time is reasonable
    local current_year
    current_year=$(date +%Y)
    if [[ $current_year -lt 2020 ]] || [[ $current_year -gt 2030 ]]; then
        log::error "  ✗ System time appears incorrect (year: $current_year)"
        log::info "  → This can cause TLS certificate validation failures"
        return 1
    else
        log::info "  ✓ System time appears reasonable"
    fi
    
    # Check NTP status if available
    if command -v timedatectl >/dev/null 2>&1; then
        local ntp_status
        ntp_status=$(timedatectl status 2>/dev/null | grep -i "ntp\|synchronized" || echo "")
        if [[ -n "$ntp_status" ]]; then
            log::info "  → NTP status: $ntp_status"
        fi
    elif command -v ntpstat >/dev/null 2>&1; then
        if ntpstat >/dev/null 2>&1; then
            log::success "  ✓ NTP synchronized"
        else
            log::warning "  ⚠️  NTP not synchronized"
        fi
    fi
    
    return 0
}

# Verbose HTTPS debugging
network_diagnostics_analysis::verbose_https_debug() {
    local test_command="${1:-curl -s https://github.com}"
    log::info "  → Running verbose HTTPS diagnostics..."
    
    # Extract URL from command
    local url
    url=$(echo "$test_command" | grep -oE 'https://[^ ]+' | head -1 || echo "https://github.com")
    
    # Run with verbose output to temp file
    local temp_file
    temp_file=$(mktemp)
    
    # Try the command with verbose output
    local verbose_cmd
    verbose_cmd=$(echo "$test_command" | sed 's/curl/curl -v/; s/-s//g')
    
    if timeout 10 bash -c "$verbose_cmd" >"$temp_file" 2>&1; then
        log::info "    Command succeeded with verbose output"
    else
        log::info "    Command failed. Key error indicators:"
        
        # Look for specific error patterns
        if grep -q "SSL_ERROR\|SSL3_GET_SERVER_CERTIFICATE\|certificate problem" "$temp_file" 2>/dev/null; then
            log::info "    • SSL/TLS certificate issue detected"
        fi
        
        if grep -q "Connection refused\|Failed to connect" "$temp_file" 2>/dev/null; then
            log::info "    • Connection refused or unreachable"
        fi
        
        if grep -q "Could not resolve\|Temporary failure in name resolution" "$temp_file" 2>/dev/null; then
            log::info "    • DNS resolution failure"
        fi
        
        if grep -q "Operation timed out\|Connection timed out" "$temp_file" 2>/dev/null; then
            log::info "    • Connection timeout"
        fi
        
        # Show first few lines of actual error
        local error_lines
        error_lines=$(grep -E "^curl:|SSL|TLS|error|Error" "$temp_file" 2>/dev/null | head -3 || echo "")
        if [[ -n "$error_lines" ]]; then
            echo "$error_lines" | while IFS= read -r line; do
                log::info "    $line"
            done
        fi
    fi
    
    rm -f "$temp_file"
    return 0
}

# Export functions for external use
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    # Script was sourced, export functions
    export -f network_diagnostics_analysis::analyze_tls_handshake
    export -f network_diagnostics_analysis::check_ipv4_vs_ipv6
    export -f network_diagnostics_analysis::check_ip_preference
    export -f network_diagnostics_analysis::check_time_sync
    export -f network_diagnostics_analysis::verbose_https_debug
fi