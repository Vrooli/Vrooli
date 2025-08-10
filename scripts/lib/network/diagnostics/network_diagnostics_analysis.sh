#!/usr/bin/env bash
# Network analysis functions for diagnostics
# Handles TLS analysis, IPv4/IPv6 comparisons, and detailed debugging
set -eo pipefail

# Source var.sh with relative path first
LIB_NETWORK_DIAGNOSTICS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${LIB_NETWORK_DIAGNOSTICS_DIR}/../../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Targeted connection failure diagnosis
# Only runs when basic connectivity tests fail - provides focused, actionable diagnostics
network_diagnostics_analysis::diagnose_connection_failure() {
    local failed_test="$1"  # e.g., "HTTPS to google.com"
    local target_domain="${2:-google.com}"
    
    log::info "🔍 Diagnosing connection failure: $failed_test"
    
    # 1. Check system time (critical for TLS certificates)
    local current_year
    current_year=$(date +%Y)
    if [[ $current_year -lt 2020 ]] || [[ $current_year -gt 2030 ]]; then
        log::error "  ❌ System time is incorrect (year: $current_year)"
        log::info "  💡 Fix: Set correct system time - this causes TLS certificate failures"
        return 1
    fi
    
    # 2. Check IPv4 vs IPv6 preference issues
    if [[ "$failed_test" == *"HTTPS"* ]] || [[ "$failed_test" == *"ping"* ]]; then
        local ipv4_works=false
        local ipv6_works=false
        
        if ping -4 -c 1 -W 2 "$target_domain" >/dev/null 2>&1; then
            ipv4_works=true
        fi
        
        if ping -6 -c 1 -W 2 "$target_domain" >/dev/null 2>&1; then
            ipv6_works=true
        fi
        
        if [[ "$ipv4_works" == "true" ]] && [[ "$ipv6_works" == "false" ]]; then
            log::info "  ✅ IPv4 works, IPv6 doesn't - this is normal"
            log::info "  💡 System will automatically use IPv4"
        elif [[ "$ipv4_works" == "false" ]] && [[ "$ipv6_works" == "true" ]]; then
            log::warning "  ⚠️  Only IPv6 works - may need IPv4 preference fix"
            log::info "  💡 Fix: Configure system to prefer IPv4 (done by network fixes)"
        elif [[ "$ipv4_works" == "false" ]] && [[ "$ipv6_works" == "false" ]]; then
            log::error "  ❌ Neither IPv4 nor IPv6 work to $target_domain"
            log::info "  💡 This suggests DNS or routing issues"
        fi
    fi
    
    # 3. Quick TLS handshake check for HTTPS failures  
    if [[ "$failed_test" == *"HTTPS"* ]] && command -v openssl >/dev/null 2>&1; then
        log::info "  🔍 Testing TLS handshake..."
        if timeout 5 openssl s_client -connect "$target_domain:443" -verify_return_error </dev/null >/dev/null 2>&1; then
            log::success "  ✅ TLS handshake works"
        else
            # Get brief error details
            local tls_error
            tls_error=$(timeout 5 openssl s_client -connect "$target_domain:443" </dev/null 2>&1 | grep -E "verify error|certificate" | head -1 || echo "")
            if [[ -n "$tls_error" ]]; then
                log::warning "  ⚠️  TLS issue: ${tls_error}"
                log::info "  💡 This could be certificate, cipher, or TSO issues"
            else
                log::error "  ❌ TLS handshake failed"
            fi
        fi
    fi
    
    # 4. DNS resolution check
    if [[ "$failed_test" == *"DNS"* ]] || [[ "$failed_test" == *"getent"* ]]; then
        log::info "  🔍 Testing DNS resolution..."
        if getent hosts 8.8.8.8 >/dev/null 2>&1; then
            log::info "  ✅ DNS system works (reverse lookup of 8.8.8.8 succeeded)"
            log::info "  💡 Issue may be with specific domain '$target_domain'"
        else
            log::error "  ❌ DNS system appears broken"
            log::info "  💡 Fix: Add reliable DNS servers (done by network fixes)"
        fi
    fi
    
    # 5. Quick HTTPS error pattern detection
    if [[ "$failed_test" == *"HTTPS"* ]] && command -v curl >/dev/null 2>&1; then
        log::info "  🔍 Analyzing HTTPS error patterns..."
        local curl_error
        curl_error=$(timeout 5 curl -s "https://$target_domain" 2>&1 | head -3 || echo "")
        
        if [[ "$curl_error" == *"SSL"* ]] || [[ "$curl_error" == *"certificate"* ]]; then
            log::warning "  ⚠️  SSL/TLS certificate issue detected"
            log::info "  💡 Common causes: wrong system time, TSO issues, or certificate problems"
        elif [[ "$curl_error" == *"Could not resolve"* ]]; then
            log::warning "  ⚠️  DNS resolution failure"
            log::info "  💡 Fix: Add reliable DNS servers (done by network fixes)"
        elif [[ "$curl_error" == *"timed out"* ]]; then
            log::warning "  ⚠️  Connection timeout"
            log::info "  💡 This suggests network routing or firewall issues"
        fi
    fi
    
    return 0
}

# Export functions for external use
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    # Script was sourced, export functions
    export -f network_diagnostics_analysis::diagnose_connection_failure
fi