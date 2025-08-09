#!/usr/bin/env bash
# Modular network diagnostics wrapper
# Combines all network diagnostic modules into a cohesive tool
set -eo pipefail

# Source var.sh with relative path first
LIB_NETWORK_DIAGNOSTICS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${LIB_NETWORK_DIAGNOSTICS_DIR}/../../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/diagnostics/network_diagnostics_core.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/diagnostics/network_diagnostics_tcp.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/diagnostics/network_diagnostics_analysis.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/diagnostics/network_diagnostics_fixes.sh"

# Set appropriate SUDO_MODE if not already set
if [[ -z "${SUDO_MODE:-}" ]]; then
    if [[ $EUID -eq 0 ]] || sudo -n true 2>/dev/null; then
        export SUDO_MODE="error"
    else
        export SUDO_MODE="skip"
    fi
fi

# Main diagnostics function that combines all modules
network_diagnostics::run() {
    log::header "Running Comprehensive Network Diagnostics (Modular)"
    
    # Run core diagnostics first
    log::info "Starting core network tests..."
    local core_exit_code=0
    network_diagnostics_core::run || core_exit_code=$?
    
    if [[ $core_exit_code -eq "${ERROR_NO_INTERNET}" ]]; then
        log::error "Critical network failure detected. Cannot continue."
        return "${ERROR_NO_INTERNET}"
    fi
    
    # Run additional analysis if core tests passed or had minor issues
    log::info ""
    log::subheader "Advanced Network Analysis"
    
    # Check TCP settings
    network_diagnostics_tcp::check_tcp_settings
    
    # Check PMTU status
    network_diagnostics_tcp::check_pmtu_status
    
    # Check time synchronization
    network_diagnostics_analysis::check_time_sync
    
    # Check IPv4 vs IPv6
    network_diagnostics_analysis::check_ipv4_vs_ipv6 "github.com"
    
    # Check IP preference
    network_diagnostics_analysis::check_ip_preference
    
    # If we detected issues, offer fixes
    if [[ $core_exit_code -ne 0 ]]; then
        log::info ""
        log::subheader "Attempting Automatic Fixes"
        
        # Check if TSO is enabled and might be causing issues
        local primary_iface
        primary_iface=$(ip route | grep default | awk '{print $5}' | head -1 2>/dev/null || echo "")
        
        if [[ -n "$primary_iface" ]] && command -v ethtool >/dev/null 2>&1; then
            local tso_status
            tso_status=$(ethtool -k "$primary_iface" 2>/dev/null | grep "tcp-segmentation-offload" | awk '{print $2}' || echo "off")
            
            if [[ "$tso_status" == "on" ]]; then
                log::warning "TCP Segmentation Offload is enabled - this often causes GitHub/TLS issues"
                
                if flow::can_run_sudo "disable TSO"; then
                    log::info "Attempting to disable TSO on $primary_iface..."
                    if sudo ethtool -K "$primary_iface" tso off gso off 2>/dev/null; then
                        log::success "✓ Disabled TSO on $primary_iface"
                        
                        # Test if this fixed the issue
                        if timeout 5 curl -s https://github.com >/dev/null 2>&1; then
                            log::success "✓ GitHub HTTPS now works!"
                            
                            # Make it permanent
                            network_diagnostics_tcp::make_tso_permanent "$primary_iface"
                        fi
                    fi
                fi
            fi
        fi
        
        # Try DNS fixes
        log::info "Checking DNS configuration..."
        network_diagnostics_fixes::fix_dns_issues
        
        # Try IPv6 fixes if needed
        local ipv6_issues=false
        if ! ping -6 -c 1 -W 2 google.com >/dev/null 2>&1 && ping -4 -c 1 -W 2 google.com >/dev/null 2>&1; then
            ipv6_issues=true
        fi
        
        if [[ "$ipv6_issues" == "true" ]]; then
            log::info "IPv6 connectivity issues detected, applying fixes..."
            network_diagnostics_fixes::fix_ipv6_issues
        fi
        
        # Check for UFW blocking
        if command -v ufw >/dev/null 2>&1; then
            network_diagnostics_fixes::fix_ufw_blocking
        fi
    fi
    
    # Final summary
    log::info ""
    log::subheader "Diagnostic Summary"
    
    if [[ $core_exit_code -eq 0 ]]; then
        log::success "✅ All core network tests passed"
        log::info "Your network connectivity appears to be working correctly."
    elif [[ $core_exit_code -eq "${ERROR_NO_INTERNET}" ]]; then
        log::error "❌ No internet connectivity"
        log::info "Please check your network connection and try again."
    else
        log::warning "⚠️  Some network tests failed"
        log::info "Automatic fixes have been attempted. You may need to:"
        log::info "  1. Restart your network connection"
        log::info "  2. Check with your network administrator"
        log::info "  3. Try using a different network"
    fi
    
    return $core_exit_code
}

# Export the main function
export -f network_diagnostics::run

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    network_diagnostics::run "$@"
fi