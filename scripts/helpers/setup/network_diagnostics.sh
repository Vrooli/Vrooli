#!/usr/bin/env bash
# Comprehensive network diagnostics script
# Tests various network layers and protocols to identify connectivity issues
set -eo pipefail

# Remove debug trap - script now runs to completion successfully

SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/exit_codes.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/flow.sh"

# Set appropriate SUDO_MODE if not already set and we're running with sudo
if [[ -z "${SUDO_MODE:-}" ]]; then
    if [[ $EUID -eq 0 ]] || sudo -n true 2>/dev/null; then
        export SUDO_MODE="error"  # We have sudo access, enable privileged operations
    else
        export SUDO_MODE="skip"   # No sudo access, skip privileged operations
    fi
fi

# Track test results
declare -A TEST_RESULTS
CRITICAL_FAILURE=false

# Function to make TCP offload fix permanent
network_diagnostics::make_tso_permanent() {
    local interface="$1"
    log::info "Making TCP offload fix permanent for $interface..."
    
    # Method 1: systemd-networkd (most reliable)
    if systemctl is-enabled systemd-networkd >/dev/null 2>&1; then
        local networkd_file="/etc/systemd/network/99-tso-fix.network"
        log::info "  → Using systemd-networkd configuration"
        
        if sudo tee "$networkd_file" >/dev/null 2>&1 <<EOF
[Match]
Name=$interface

[Link]
GenericSegmentationOffload=false
TCPSegmentationOffload=false
EOF
        then
            log::success "  ✓ Created $networkd_file"
            sudo systemctl restart systemd-networkd 2>/dev/null || true
            log::success "  ✓ TCP offload fix is now permanent via systemd-networkd"
            return 0
        fi
    fi
    
    # Method 2: NetworkManager dispatcher (if NetworkManager is active)
    if systemctl is-active NetworkManager >/dev/null 2>&1; then
        local dispatcher_file="/etc/NetworkManager/dispatcher.d/99-disable-tso"
        log::info "  → Using NetworkManager dispatcher"
        
        if sudo tee "$dispatcher_file" >/dev/null 2>&1 <<EOF
#!/bin/bash
# Disable TCP/GSO offload to fix TLS issues
if [ "\$1" = "$interface" ] && [ "\$2" = "up" ]; then
    ethtool -K $interface tso off gso off 2>/dev/null || true
fi
EOF
        then
            sudo chmod +x "$dispatcher_file" 2>/dev/null
            log::success "  ✓ Created NetworkManager dispatcher script"
            log::success "  ✓ TCP offload fix is now permanent via NetworkManager"
            return 0
        fi
    fi
    
    # Method 3: rc.local fallback
    if [[ -f /etc/rc.local ]]; then
        log::info "  → Adding to /etc/rc.local as fallback"
        if ! grep -q "ethtool.*$interface.*tso off" /etc/rc.local 2>/dev/null; then
            sudo sed -i '/^exit 0/i ethtool -K '"$interface"' tso off gso off 2>/dev/null || true' /etc/rc.local 2>/dev/null
            log::success "  ✓ Added to /etc/rc.local"
        fi
    fi
    
    # Method 4: Create a simple systemd service
    local service_file="/etc/systemd/system/fix-tso.service"
    log::info "  → Creating systemd service as backup"
    
    if sudo tee "$service_file" >/dev/null 2>&1 <<EOF
[Unit]
Description=Disable TCP Segmentation Offload to fix TLS issues
After=network.target

[Service]
Type=oneshot
ExecStart=/sbin/ethtool -K $interface tso off gso off
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF
    then
        sudo systemctl enable fix-tso.service 2>/dev/null || true
        log::success "  ✓ Created and enabled fix-tso.service"
        log::success "  ✓ TCP offload fix is now permanent via systemd service"
        return 0
    fi
    
    log::warning "  → Could not make fix permanent automatically"
    log::info "  → Manual option: Add 'ethtool -K $interface tso off gso off' to your startup scripts"
    return 1
}

# Test various MTU sizes to find optimal
network_diagnostics::test_mtu_discovery() {
    local host="${1:-github.com}"
    local sizes=(1500 1472 1450 1400 1350)
    
    log::subheader "MTU Discovery Tests"
    for size in "${sizes[@]}"; do
        log::info "Testing MTU size $size to $host..."
        if ping -c 1 -s $((size - 28)) -M do "$host" >/dev/null 2>&1; then
            log::success "  ✓ MTU $size works"
        else
            log::error "  ✗ MTU $size fails (fragmentation needed)"
            break
        fi
    done
}

# Check PMTU probing status
network_diagnostics::check_pmtu_status() {
    log::info "Checking Path MTU Discovery settings..."
    local mtu_probing=$(sysctl -n net.ipv4.tcp_mtu_probing 2>/dev/null || echo "0")
    local base_mss=$(sysctl -n net.ipv4.tcp_base_mss 2>/dev/null || echo "512")
    
    if [[ "$mtu_probing" != "1" ]]; then
        log::warning "  → PMTU probing disabled (current: $mtu_probing, recommended: 1)"
        TEST_RESULTS["pmtu_probing"]="DISABLED"
    else
        log::success "  ✓ PMTU probing enabled"
        TEST_RESULTS["pmtu_probing"]="ENABLED"
    fi
}

# Detailed TLS handshake timing with OpenSSL
network_diagnostics::analyze_tls_handshake() {
    local host="${1:-github.com}"
    
    log::info "Analyzing TLS handshake to $host..."
    
    # Use openssl s_client for detailed analysis
    # Disable exit on error temporarily to handle timeout properly
    set +e
    local output
    local exit_code
    
    # Add debug logging
    log::info "  → Running openssl s_client (10s timeout)..."
    
    # Try a more aggressive timeout approach with enhanced debugging
    # Show more details to understand the failure
    output=$(timeout --kill-after=2s 10s openssl s_client -connect "$host:443" \
        -servername "$host" -no_ssl2 -no_ssl3 -showcerts -state -msg </dev/null 2>&1)
    exit_code=$?
    
    log::info "  → OpenSSL command completed with exit code: $exit_code"
    
    # Re-enable exit on error
    set -e
    
    if [[ $exit_code -eq 0 ]]; then
        # Extract cipher and protocol info
        local cipher=$(echo "$output" | grep -m1 "Cipher is" | cut -d' ' -f5- || echo "unknown")
        local protocol=$(echo "$output" | grep -m1 "Protocol  :" | cut -d':' -f2- | xargs || echo "unknown")
        log::success "  ✓ TLS handshake succeeded"
        log::info "    Protocol: $protocol"
        log::info "    Cipher: $cipher"
    elif [[ $exit_code -eq 124 ]]; then
        log::error "  ✗ TLS handshake timed out after 10 seconds"
        log::warning "    → Connection established but TLS negotiation hung"
        log::warning "    → Likely cause: MTU issues or middlebox interference"
        
        # Show any partial output we got
        if [[ -n "$output" ]]; then
            echo "$output" | grep -E "(SSL_connect:|error:|fail|alert)" | head -5 | while IFS= read -r line; do
                [[ -n "$line" ]] && log::info "    → Debug: $line"
            done
        fi
    else
        log::error "  ✗ TLS handshake failed (exit code: $exit_code)"
        
        # Show detailed error messages from OpenSSL output
        if [[ -n "$output" ]]; then
            echo "$output" | grep -E "(error|fail|refused|alert|SSL)" | head -5 | while IFS= read -r line; do
                [[ -n "$line" ]] && log::warning "    → $line"
            done
        fi
        
        # Check specific failure reasons with more patterns
        if echo "$output" | grep -q "alert handshake failure"; then
            log::warning "    → Cipher suite mismatch - no common ciphers with server"
        elif echo "$output" | grep -q "alert protocol version"; then
            log::warning "    → TLS protocol version not supported by server"
        elif echo "$output" | grep -q "Connection refused"; then
            log::warning "    → Connection refused by server"
        elif echo "$output" | grep -q "No route to host"; then
            log::warning "    → Network routing issue"
        elif echo "$output" | grep -q "certificate verify failed"; then
            log::warning "    → Certificate verification failed"
        elif echo "$output" | grep -q "SSL23_GET_SERVER_HELLO"; then
            log::warning "    → Server sent invalid/unexpected response (possible proxy/firewall)"
        elif echo "$output" | grep -q "sslv3 alert"; then
            local alert_type=$(echo "$output" | grep -o "sslv3 alert [^:]*" | head -1)
            log::warning "    → Server alert: $alert_type"
        elif echo "$output" | grep -q "tlsv1 alert"; then
            local alert_type=$(echo "$output" | grep -o "tlsv1 alert [^:]*" | head -1)
            log::warning "    → Server alert: $alert_type"
        else
            log::warning "    → Unknown TLS failure - check debug output above"
        fi
        
        # Show last SSL state if available
        if echo "$output" | grep -q "SSL_connect:"; then
            local last_state=$(echo "$output" | grep "SSL_connect:" | tail -1)
            log::info "    → Last SSL state: $last_state"
        fi
    fi
}

# Check Explicit Congestion Notification settings
network_diagnostics::check_tcp_settings() {
    log::subheader "TCP/Network Settings"
    
    # ECN setting
    local ecn=$(sysctl -n net.ipv4.tcp_ecn 2>/dev/null || echo "unknown")
    if [[ "$ecn" == "1" || "$ecn" == "2" ]]; then
        log::warning "  → ECN enabled (value: $ecn) - may cause issues with some routers"
        TEST_RESULTS["ecn"]="ENABLED"
    else
        log::info "  → ECN disabled"
        TEST_RESULTS["ecn"]="DISABLED"
    fi
    
    # Window scaling
    local ws=$(sysctl -n net.ipv4.tcp_window_scaling 2>/dev/null || echo "unknown")
    log::info "  → TCP window scaling: $ws"
    
    # Timestamps
    local ts=$(sysctl -n net.ipv4.tcp_timestamps 2>/dev/null || echo "unknown")
    log::info "  → TCP timestamps: $ts"
}

# Check system time (important for certificate validation)
network_diagnostics::check_time_sync() {
    log::info "Checking system time synchronization..."
    
    # Compare with time server
    if command -v ntpdate >/dev/null 2>&1; then
        local offset=$(ntpdate -q pool.ntp.org 2>/dev/null | grep -oP 'offset [+-]?\d+\.\d+' | awk '{print $2}')
        if [[ -n "$offset" ]]; then
            local abs_offset=${offset#-}  # Remove negative sign
            if (( $(echo "$abs_offset > 5" | bc -l) )); then
                log::warning "  → Time offset: ${offset}s (may cause certificate issues)"
            else
                log::success "  ✓ Time synchronized (offset: ${offset}s)"
            fi
        fi
    elif systemctl is-active --quiet systemd-timesyncd; then
        log::success "  ✓ systemd-timesyncd is active"
    else
        log::warning "  → Cannot verify time synchronization"
    fi
}

# Comprehensive IPv4 vs IPv6 diagnostics
network_diagnostics::check_ipv4_vs_ipv6() {
    local host="${1:-github.com}"
    
    log::subheader "IPv4 vs IPv6 Connectivity Analysis"
    
    # 1. Check what IPs the host resolves to
    log::info "Checking DNS resolution for $host..."
    local ipv4_addrs=()
    local ipv6_addrs=()
    
    # Get IPv4 addresses
    if command -v dig >/dev/null 2>&1; then
        mapfile -t ipv4_addrs < <(dig +short A "$host" 2>/dev/null | grep -E '^[0-9.]+$')
        mapfile -t ipv6_addrs < <(dig +short AAAA "$host" 2>/dev/null | grep -E '^[0-9a-fA-F:]+$')
    elif command -v host >/dev/null 2>&1; then
        mapfile -t ipv4_addrs < <(host -t A "$host" 2>/dev/null | grep "has address" | awk '{print $4}')
        mapfile -t ipv6_addrs < <(host -t AAAA "$host" 2>/dev/null | grep "has IPv6 address" | awk '{print $5}')
    else
        # Fallback to getent
        local all_ips=$(getent hosts "$host" 2>/dev/null | awk '{for(i=2;i<=NF;i++) print $i}')
        for ip in $all_ips; do
            if [[ "$ip" =~ ^[0-9.]+$ ]]; then
                ipv4_addrs+=("$ip")
            elif [[ "$ip" =~ : ]]; then
                ipv6_addrs+=("$ip")
            fi
        done
    fi
    
    log::info "  IPv4 addresses: ${#ipv4_addrs[@]} found"
    for ip in "${ipv4_addrs[@]}"; do
        log::info "    → $ip"
    done
    
    log::info "  IPv6 addresses: ${#ipv6_addrs[@]} found"
    for ip in "${ipv6_addrs[@]}"; do
        log::info "    → $ip"
    done
    
    # 2. Test connectivity to each protocol separately
    log::info "Testing IPv4-only connectivity to $host..."
    set +e
    
    # Test IPv4 ping
    if ping -4 -c 1 -W 2 "$host" >/dev/null 2>&1; then
        log::success "  ✓ IPv4 ping works"
        TEST_RESULTS["ipv4_ping_$host"]="PASS"
    else
        log::error "  ✗ IPv4 ping fails"
        TEST_RESULTS["ipv4_ping_$host"]="FAIL"
    fi
    
    # Test IPv4 HTTPS
    if timeout 10 curl -4 -s -o /dev/null -w "%{http_code}" --max-time 8 "https://$host" >/dev/null 2>&1; then
        log::success "  ✓ IPv4 HTTPS works"
        TEST_RESULTS["ipv4_https_$host"]="PASS"
    else
        log::error "  ✗ IPv4 HTTPS fails"
        TEST_RESULTS["ipv4_https_$host"]="FAIL"
    fi
    
    log::info "Testing IPv6-only connectivity to $host..."
    
    # Check if IPv6 is even available
    if [[ ${#ipv6_addrs[@]} -eq 0 ]]; then
        log::info "  → No IPv6 addresses available for $host"
        TEST_RESULTS["ipv6_available_$host"]="NO"
    else
        TEST_RESULTS["ipv6_available_$host"]="YES"
        
        # Test IPv6 ping
        if ping -6 -c 1 -W 2 "$host" >/dev/null 2>&1; then
            log::success "  ✓ IPv6 ping works"
            TEST_RESULTS["ipv6_ping_$host"]="PASS"
        else
            log::error "  ✗ IPv6 ping fails"
            TEST_RESULTS["ipv6_ping_$host"]="FAIL"
        fi
        
        # Test IPv6 HTTPS
        if timeout 10 curl -6 -s -o /dev/null -w "%{http_code}" --max-time 8 "https://$host" >/dev/null 2>&1; then
            log::success "  ✓ IPv6 HTTPS works"
            TEST_RESULTS["ipv6_https_$host"]="PASS"
        else
            log::error "  ✗ IPv6 HTTPS fails"
            TEST_RESULTS["ipv6_https_$host"]="FAIL"
        fi
    fi
    
    set -e
    
    # 3. Check system IPv6 configuration
    log::info "Checking system IPv6 configuration..."
    
    # Check if IPv6 is disabled globally
    local ipv6_disabled=$(sysctl -n net.ipv6.conf.all.disable_ipv6 2>/dev/null || echo "0")
    if [[ "$ipv6_disabled" == "1" ]]; then
        log::warning "  → IPv6 is globally disabled"
        TEST_RESULTS["ipv6_system_enabled"]="DISABLED"
    else
        log::info "  → IPv6 is enabled"
        TEST_RESULTS["ipv6_system_enabled"]="ENABLED"
        
        # Check IPv6 privacy extensions (can cause issues)
        local use_tempaddr=$(sysctl -n net.ipv6.conf.all.use_tempaddr 2>/dev/null || echo "0")
        if [[ "$use_tempaddr" == "2" ]]; then
            log::warning "  → IPv6 privacy extensions enabled (may cause issues)"
            TEST_RESULTS["ipv6_privacy_extensions"]="ENABLED"
        fi
    fi
    
    # 4. Check IPv6 routing
    local ipv6_default_route=$(ip -6 route show default 2>/dev/null | grep -c "default" || echo "0")
    if [[ "$ipv6_default_route" -eq 0 ]]; then
        log::warning "  → No IPv6 default route configured"
        TEST_RESULTS["ipv6_routing"]="NO_ROUTE"
    else
        log::info "  → IPv6 default route exists"
        TEST_RESULTS["ipv6_routing"]="OK"
    fi
    
    # 5. Diagnose IPv6 issues
    if [[ "${TEST_RESULTS["ipv6_available_$host"]}" == "YES" ]] && 
       [[ "${TEST_RESULTS["ipv4_https_$host"]}" == "PASS" ]] && 
       [[ "${TEST_RESULTS["ipv6_https_$host"]}" == "FAIL" ]]; then
        log::warning ""
        log::warning "IPv6 CONNECTIVITY ISSUE DETECTED:"
        log::info "  → $host works over IPv4 but fails over IPv6"
        log::info "  → System is likely preferring broken IPv6 connection"
        log::info "  → This is why $host fails while sites without IPv6 work"
        TEST_RESULTS["ipv6_broken_for_$host"]="YES"
    elif [[ "${TEST_RESULTS["ipv6_available_$host"]}" == "NO" ]] && 
         [[ "${TEST_RESULTS["ipv4_https_$host"]}" == "FAIL" ]]; then
        log::warning ""
        log::warning "IPv4-ONLY CONNECTIVITY ISSUE DETECTED:"
        log::info "  → $host has no IPv6 addresses (IPv4-only)"
        log::info "  → But IPv4 HTTPS connection also fails"
        log::info "  → This suggests server-specific or deep packet inspection issues"
        TEST_RESULTS["ipv4_only_failure_$host"]="YES"
    fi
}

# Check system's IPv6 vs IPv4 preference
network_diagnostics::check_ip_preference() {
    log::info "Checking system IP version preference..."
    
    # Check gai.conf for preferences
    local gai_conf="/etc/gai.conf"
    if [[ -f "$gai_conf" ]]; then
        if grep -q "^precedence ::ffff:0:0/96.*100" "$gai_conf" 2>/dev/null; then
            log::info "  → System configured to prefer IPv4"
            TEST_RESULTS["ip_preference"]="IPv4"
        else
            log::info "  → System prefers IPv6 (default)"
            TEST_RESULTS["ip_preference"]="IPv6"
        fi
    else
        log::info "  → No gai.conf found, using system defaults (IPv6 preferred)"
        TEST_RESULTS["ip_preference"]="IPv6"
    fi
    
    # Test actual behavior with a dual-stack host
    log::info "Testing actual connection preference..."
    local target="google.com"  # Known dual-stack host
    
    set +e
    local actual_ip=$(curl -s -o /dev/null -w "%{remote_ip}" --max-time 5 "https://$target" 2>/dev/null)
    set -e
    
    if [[ -n "$actual_ip" ]]; then
        if [[ "$actual_ip" =~ : ]]; then
            log::info "  → Actually connected via IPv6: $actual_ip"
            TEST_RESULTS["actual_ip_preference"]="IPv6"
        else
            log::info "  → Actually connected via IPv4: $actual_ip"
            TEST_RESULTS["actual_ip_preference"]="IPv4"
        fi
    fi
}

# Enable PMTU probing to fix path MTU issues
network_diagnostics::fix_pmtu_probing() {
    if [[ "${TEST_RESULTS[pmtu_probing]}" == "DISABLED" ]]; then
        log::info "Enabling Path MTU Discovery probing..."
        
        if flow::can_run_sudo "enable PMTU probing"; then
            # Enable PMTU probing
            sudo sysctl -w net.ipv4.tcp_mtu_probing=1 >/dev/null 2>&1
            sudo sysctl -w net.ipv4.tcp_base_mss=1024 >/dev/null 2>&1
            
            # Test if it helps
            if curl -s -o /dev/null -w "%{http_code}" --max-time 5 https://github.com >/dev/null 2>&1; then
                log::success "  ✓ PMTU probing enabled and GitHub now accessible!"
                
                # Make permanent
                echo "net.ipv4.tcp_mtu_probing = 1" | sudo tee -a /etc/sysctl.conf >/dev/null
                echo "net.ipv4.tcp_base_mss = 1024" | sudo tee -a /etc/sysctl.conf >/dev/null
            else
                log::warning "  → PMTU probing enabled but didn't resolve issue"
            fi
        else
            log::warning "  → MANUAL FIX: Run these commands with sudo:"
            log::info "    sudo sysctl -w net.ipv4.tcp_mtu_probing=1"
            log::info "    sudo sysctl -w net.ipv4.tcp_base_mss=1024"
        fi
    fi
}

# Test and adjust MTU size if needed
network_diagnostics::fix_mtu_size() {
    local interface=$(ip route | grep default | awk '{print $5}' | head -1)
    local current_mtu=$(ip link show "$interface" | grep -oP 'mtu \K\d+')
    
    if [[ "$current_mtu" -gt 1400 ]] && [[ "${TEST_RESULTS[github_https]:-${TEST_RESULTS["HTTPS to github.com"]:-FAIL}}" == "FAIL" ]]; then
        log::info "Testing reduced MTU (current: $current_mtu)..."
        
        if flow::can_run_sudo "adjust MTU"; then
            # Try progressively smaller MTUs
            for mtu in 1460 1400 1380; do
                sudo ip link set dev "$interface" mtu "$mtu"
                
                if curl -s -o /dev/null -w "%{http_code}" --max-time 5 https://github.com >/dev/null 2>&1; then
                    log::success "  ✓ MTU $mtu works! GitHub accessible"
                    log::info "  → To make permanent, add to network config"
                    break
                else
                    log::info "  → MTU $mtu didn't help"
                fi
            done
            
            # Restore original if nothing worked
            sudo ip link set dev "$interface" mtu "$current_mtu"
        else
            log::warning "  → MANUAL FIX: Try reducing MTU:"
            log::info "    sudo ip link set dev $interface mtu 1400"
        fi
    fi
}

# Disable ECN if it's causing issues
network_diagnostics::fix_ecn() {
    if [[ "${TEST_RESULTS[ecn]}" == "ENABLED" ]] && [[ "${TEST_RESULTS[github_https]:-${TEST_RESULTS["HTTPS to github.com"]:-FAIL}}" == "FAIL" ]]; then
        log::info "Disabling ECN (Explicit Congestion Notification)..."
        
        if flow::can_run_sudo "disable ECN"; then
            local old_ecn=$(sysctl -n net.ipv4.tcp_ecn)
            sudo sysctl -w net.ipv4.tcp_ecn=0 >/dev/null 2>&1
            
            if curl -s -o /dev/null -w "%{http_code}" --max-time 5 https://github.com >/dev/null 2>&1; then
                log::success "  ✓ Disabling ECN fixed the issue!"
                echo "net.ipv4.tcp_ecn = 0" | sudo tee -a /etc/sysctl.conf >/dev/null
            else
                log::info "  → Disabling ECN didn't help"
                sudo sysctl -w net.ipv4.tcp_ecn="$old_ecn" >/dev/null 2>&1
            fi
        else
            log::warning "  → MANUAL FIX: Disable ECN:"
            log::info "    sudo sysctl -w net.ipv4.tcp_ecn=0"
        fi
    fi
}

# Fix IPv6 connectivity issues
network_diagnostics::fix_ipv6_issues() {
    # Check if we detected broken IPv6
    local has_ipv6_issue=false
    for host in github.com gitlab.com bitbucket.org; do
        if [[ "${TEST_RESULTS["ipv6_broken_for_$host"]:-}" == "YES" ]]; then
            has_ipv6_issue=true
            break
        fi
    done
    
    if [[ "$has_ipv6_issue" != "true" ]]; then
        return 0
    fi
    
    log::warning "IPv6 connectivity issues detected. Attempting fixes..."
    
    # Fix 1: Prefer IPv4 over IPv6 (safest, most effective)
    log::info "Fix 1: Configuring system to prefer IPv4..."
    
    if flow::can_run_sudo "configure IPv4 preference"; then
        local gai_conf="/etc/gai.conf"
        local gai_backup="${gai_conf}.backup.$(date +%Y%m%d-%H%M%S)"
        
        # Backup existing config if it exists
        if [[ -f "$gai_conf" ]]; then
            sudo cp "$gai_conf" "$gai_backup" 2>/dev/null
            log::info "  → Backed up existing gai.conf"
        fi
        
        # Create/update gai.conf to prefer IPv4
        log::info "  → Updating address selection preference..."
        if sudo tee "$gai_conf" >/dev/null 2>&1 <<'EOF'
# Prefer IPv4 addresses over IPv6
# This fixes issues where IPv6 is partially broken
precedence ::ffff:0:0/96  100

# Standard IPv6 preferences (lower priority)
precedence ::/0           40
precedence 2002::/16      30
precedence ::/96          20
precedence ::1/128        50
label ::1/128             0
label ::/0                1
label 2002::/16           2
label ::ffff:0:0/96       4
label fec0::/10           5
label fc00::/7            6
label 2001:0::/32         7
EOF
        then
            log::success "  ✓ Configured system to prefer IPv4"
            TEST_RESULTS["ipv4_preference_set"]="YES"
            
            # Test if it fixed the issue
            log::info "  → Testing GitHub with IPv4 preference..."
            set +e
            if timeout 8 curl -s -o /dev/null --max-time 6 https://github.com 2>&1; then
                log::success "  🎉 GitHub now accessible with IPv4 preference!"
                TEST_RESULTS["ipv6_fix_success"]="YES"
            else
                log::warning "  → IPv4 preference didn't fully resolve the issue"
            fi
            set -e
        fi
    else
        log::warning "  → MANUAL FIX: Configure IPv4 preference:"
        log::info "    sudo tee /etc/gai.conf <<'EOF'"
        log::info "precedence ::ffff:0:0/96  100"
        log::info "EOF"
    fi
    
    # Fix 2: Disable IPv6 privacy extensions if enabled
    if [[ "${TEST_RESULTS["ipv6_privacy_extensions"]:-}" == "ENABLED" ]]; then
        log::info "Fix 2: Disabling IPv6 privacy extensions..."
        
        if flow::can_run_sudo "disable IPv6 privacy extensions"; then
            sudo sysctl -w net.ipv6.conf.all.use_tempaddr=0 >/dev/null 2>&1
            sudo sysctl -w net.ipv6.conf.default.use_tempaddr=0 >/dev/null 2>&1
            
            # Make permanent
            echo "net.ipv6.conf.all.use_tempaddr = 0" | sudo tee -a /etc/sysctl.conf >/dev/null
            echo "net.ipv6.conf.default.use_tempaddr = 0" | sudo tee -a /etc/sysctl.conf >/dev/null
            
            log::success "  ✓ Disabled IPv6 privacy extensions"
        else
            log::warning "  → MANUAL FIX: Disable IPv6 privacy extensions:"
            log::info "    sudo sysctl -w net.ipv6.conf.all.use_tempaddr=0"
        fi
    fi
    
    # Fix 3: If still failing, offer to disable IPv6 entirely (last resort)
    if [[ "${TEST_RESULTS["ipv6_fix_success"]:-}" != "YES" ]] && [[ "$has_ipv6_issue" == "true" ]]; then
        log::warning "Fix 3: IPv6 is still causing issues. Disable IPv6 entirely? [y/N]"
        
        if read -r -t 10 response 2>/dev/null && [[ "$response" =~ ^[Yy]$ ]]; then
            if flow::can_run_sudo "disable IPv6"; then
                log::info "  → Disabling IPv6 system-wide..."
                
                # Disable IPv6 via sysctl
                sudo sysctl -w net.ipv6.conf.all.disable_ipv6=1 >/dev/null 2>&1
                sudo sysctl -w net.ipv6.conf.default.disable_ipv6=1 >/dev/null 2>&1
                sudo sysctl -w net.ipv6.conf.lo.disable_ipv6=1 >/dev/null 2>&1
                
                # Make permanent
                echo "net.ipv6.conf.all.disable_ipv6 = 1" | sudo tee -a /etc/sysctl.conf >/dev/null
                echo "net.ipv6.conf.default.disable_ipv6 = 1" | sudo tee -a /etc/sysctl.conf >/dev/null
                echo "net.ipv6.conf.lo.disable_ipv6 = 1" | sudo tee -a /etc/sysctl.conf >/dev/null
                
                log::success "  ✓ IPv6 disabled system-wide"
                log::info "  → This ensures all connections use IPv4"
                
                # Final test
                set +e
                if timeout 8 curl -s -o /dev/null --max-time 6 https://github.com 2>&1; then
                    log::success "  🎉 GitHub now accessible with IPv6 disabled!"
                    TEST_RESULTS["ipv6_disabled_fix"]="YES"
                fi
                set -e
            else
                log::warning "  → MANUAL FIX: Disable IPv6:"
                log::info "    sudo sysctl -w net.ipv6.conf.all.disable_ipv6=1"
                log::info "    echo 'net.ipv6.conf.all.disable_ipv6 = 1' | sudo tee -a /etc/sysctl.conf"
            fi
        else
            log::info "  → Keeping IPv6 enabled. Alternative fixes:"
            log::info "    • Use IPv4-only DNS servers (8.8.8.8, 1.1.1.1)"
            log::info "    • Add IPv4 addresses to /etc/hosts for problem sites"
            log::info "    • Use a VPN that handles IPv6 properly"
        fi
    fi
}

# Fix IPv4-only connectivity issues (when IPv6 isn't the problem)
network_diagnostics::fix_ipv4_only_issues() {
    # Check if we detected IPv4-only failures
    local has_ipv4_issue=false
    for host in github.com gitlab.com bitbucket.org; do
        if [[ "${TEST_RESULTS["ipv4_only_failure_$host"]:-}" == "YES" ]]; then
            has_ipv4_issue=true
            break
        fi
    done
    
    if [[ "$has_ipv4_issue" != "true" ]]; then
        return 0
    fi
    
    log::warning "IPv4-only connectivity issues detected. Attempting specialized fixes..."
    
    # Fix 1: Test different TLS/cipher configurations
    log::info "Fix 1: Testing different TLS configurations..."
    
    # Test TLS 1.2 only
    set +e
    if timeout 10 curl -s --tlsv1.2 --tls-max 1.2 -o /dev/null "https://github.com" 2>&1; then
        log::success "  ✓ TLS 1.2 only works!"
        log::info "  → Configure applications to use TLS 1.2 specifically"
        TEST_RESULTS["tls12_only_fix"]="YES"
    else
        log::warning "  → TLS 1.2 only didn't help"
    fi
    
    # Test with specific cipher
    if timeout 10 curl -s --ciphers 'ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM:DHE+CHACHA20:!aNULL:!MD5:!DSS' -o /dev/null "https://github.com" 2>&1; then
        log::success "  ✓ Modern ciphers work!"
        TEST_RESULTS["modern_ciphers_fix"]="YES"
    else
        log::warning "  → Modern cipher selection didn't help"
    fi
    set -e
    
    # Fix 2: Certificate validation bypass (testing only)
    log::info "Fix 2: Testing certificate validation..."
    set +e
    if timeout 10 curl -s -k -o /dev/null "https://github.com" 2>&1; then
        log::warning "  → Connection works with certificate validation disabled"
        log::warning "  → This suggests certificate chain or time sync issues"
        log::info "  → Check system time and update ca-certificates"
        TEST_RESULTS["cert_validation_issue"]="YES"
    else
        log::info "  → Issue persists even without certificate validation"
    fi
    set -e
    
    # Fix 3: Test different User-Agent and HTTP versions
    log::info "Fix 3: Testing different request configurations..."
    set +e
    if timeout 10 curl -s -A "Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0" -o /dev/null "https://github.com" 2>&1; then
        log::success "  ✓ Firefox User-Agent works!"
        TEST_RESULTS["user_agent_fix"]="YES"
    else
        log::warning "  → Different User-Agent didn't help"
    fi
    
    if timeout 10 curl -s --http2 -o /dev/null "https://github.com" 2>&1; then
        log::success "  ✓ Forced HTTP/2 works!"
        TEST_RESULTS["http2_fix"]="YES"
    else
        log::warning "  → Forced HTTP/2 didn't help"
    fi
    set -e
    
    # Fix 4: Test bypassing potential proxy/middlebox issues
    log::info "Fix 4: Testing direct connection bypass..."
    
    # Try connecting to GitHub's IP directly
    local github_ip="140.82.112.3"  # From the DNS resolution earlier
    set +e
    if timeout 10 curl -s --resolve "github.com:443:$github_ip" -o /dev/null "https://github.com" 2>&1; then
        log::success "  ✓ Direct IP connection works!"
        log::info "  → DNS resolution may be returning problematic IPs"
        TEST_RESULTS["direct_ip_fix"]="YES"
    else
        log::warning "  → Direct IP connection didn't help"
    fi
    set -e
    
    # Fix 5: Test with different DNS servers (manual approach since --dns-servers may not be available)
    log::info "Fix 5: Testing with different DNS resolution..."
    set +e
    
    # Check if curl supports --dns-servers
    if curl --help 2>&1 | grep -q -- "--dns-servers"; then
        # Try Cloudflare DNS
        if timeout 10 curl -s --dns-servers 1.1.1.1 -o /dev/null "https://github.com" 2>&1; then
            log::success "  ✓ Cloudflare DNS (1.1.1.1) resolves the issue!"
            log::info "  → Configure system to use 1.1.1.1 as primary DNS"
            TEST_RESULTS["cloudflare_dns_fix"]="YES"
        else
            # Try Google DNS
            if timeout 10 curl -s --dns-servers 8.8.8.8 -o /dev/null "https://github.com" 2>&1; then
                log::success "  ✓ Google DNS (8.8.8.8) resolves the issue!"
                log::info "  → Configure system to use 8.8.8.8 as primary DNS"
                TEST_RESULTS["google_dns_fix"]="YES"
            else
                log::warning "  → Different DNS servers didn't help"
            fi
        fi
    else
        log::info "  → curl --dns-servers not supported, skipping DNS server tests"
        log::info "  → MANUAL TEST: Configure system DNS to use 1.1.1.1 or 8.8.8.8"
    fi
    set -e
    
    # Fix 6: Final test with combined approaches
    log::info "Fix 6: Testing combined approach..."
    set +e
    if curl --help 2>&1 | grep -q -- "--dns-servers"; then
        if timeout 10 curl -s --tlsv1.2 --tls-max 1.2 --dns-servers 1.1.1.1 \
            -A "Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0" \
            -o /dev/null "https://github.com" 2>&1; then
            log::success "  🎉 Combined approach works!"
            log::info "  → Use TLS 1.2 + Cloudflare DNS + Firefox User-Agent"
            TEST_RESULTS["combined_fix"]="YES"
        fi
    else
        # Try without DNS servers option
        if timeout 10 curl -s --tlsv1.2 --tls-max 1.2 \
            -A "Mozilla/5.0 (X11; Linux x86_64; rv:91.0) Gecko/20100101 Firefox/91.0" \
            -o /dev/null "https://github.com" 2>&1; then
            log::success "  🎉 Combined approach (without DNS override) works!"
            log::info "  → Use TLS 1.2 + Firefox User-Agent"
            TEST_RESULTS["combined_fix"]="YES"
        fi
    fi
    set -e
}

# Add specific host to use IPv4 only
network_diagnostics::add_ipv4_host_override() {
    local host="$1"
    local ipv4_addr="$2"
    
    if [[ -z "$host" ]] || [[ -z "$ipv4_addr" ]]; then
        return 1
    fi
    
    log::info "Adding IPv4 override for $host..."
    
    if flow::can_run_sudo "update hosts file"; then
        # Check if entry already exists
        if ! grep -q "^$ipv4_addr\s\+$host" /etc/hosts 2>/dev/null; then
            echo "$ipv4_addr $host" | sudo tee -a /etc/hosts >/dev/null
            log::success "  ✓ Added $host → $ipv4_addr to /etc/hosts"
        else
            log::info "  → Entry already exists in /etc/hosts"
        fi
    else
        log::warning "  → MANUAL: Add to /etc/hosts:"
        log::info "    $ipv4_addr $host"
    fi
}

# Enhanced diagnosis with specific recommendations
network_diagnostics::enhanced_diagnosis() {
    # Check for patterns in HTTPS failures
    local https_failures=0
    local https_successes=0
    local failed_domains=()
    local successful_domains=()
    
    for test_name in "${!TEST_RESULTS[@]}"; do
        if [[ "$test_name" =~ ^"HTTPS to " ]]; then
            local domain=$(echo "$test_name" | sed 's/HTTPS to //')
            if [[ "${TEST_RESULTS[$test_name]}" == "FAIL" ]]; then
                https_failures=$((https_failures + 1))
                failed_domains+=("$domain")
            else
                https_successes=$((https_successes + 1))
                successful_domains+=("$domain")
            fi
        fi
    done
    
    # More specific diagnosis based on test results
    if [[ $https_failures -gt 0 ]] && [[ $https_successes -gt 0 ]]; then
        log::warning "SELECTIVE HTTPS FAILURE DETECTED"
        log::info "Failed domains: ${failed_domains[*]}"
        log::info "Successful domains: ${successful_domains[*]}"
        log::info "Possible causes (in order of likelihood):"
        
        # Check MTU
        local primary_iface=$(ip route | grep default | awk '{print $5}' | head -1)
        if [[ -n "$primary_iface" ]]; then
            local current_mtu=$(ip link show "$primary_iface" | grep -oP 'mtu \K\d+')
            if [[ "$current_mtu" -gt 1450 ]]; then
                log::warning "1. MTU/PMTUD Black Hole Issue"
                log::info "   → Large packets being dropped silently"
                log::info "   → Try: Reduce MTU or enable PMTU probing"
            fi
        fi
        
        # Check PMTU probing
        if [[ "${TEST_RESULTS[pmtu_probing]}" == "DISABLED" ]]; then
            log::warning "2. Path MTU Discovery Disabled"
            log::info "   → System cannot adapt to network MTU changes"
            log::info "   → Enable with: sysctl -w net.ipv4.tcp_mtu_probing=1"
        fi
        
        # Check ECN
        if [[ "${TEST_RESULTS[ecn]}" == "ENABLED" ]]; then
            log::warning "3. ECN Incompatibility"
            log::info "   → Some routers/firewalls mishandle ECN packets"
            log::info "   → Disable with: sysctl -w net.ipv4.tcp_ecn=0"
        fi
        
        # Check time
        log::warning "4. Certificate Validation Issues"
        log::info "   → Check system time is correct"
        log::info "   → Update ca-certificates: sudo apt update && sudo apt install ca-certificates"
        
        # Check for IPv6 issues
        if [[ "${TEST_RESULTS["ipv6_broken_for_github.com"]:-}" == "YES" ]]; then
            log::warning "5. IPv6 Connectivity Issues"
            log::info "   → GitHub has IPv6 addresses but they don't work"
            log::info "   → System prefers broken IPv6 over working IPv4"
            log::info "   → Fix: Configure system to prefer IPv4 addresses"
        fi
        
        # Check for IPv4-only issues
        if [[ "${TEST_RESULTS["ipv4_only_failure_github.com"]:-}" == "YES" ]]; then
            log::warning "6. IPv4-Only Server Issues"
            log::info "   → GitHub has no IPv6 addresses (IPv4-only server)"
            log::info "   → But IPv4 HTTPS connection fails despite working ping"
            log::info "   → Likely causes: TLS version, cipher suites, or DNS resolution"
        fi
    fi
}

# Function to fix SearXNG uWSGI issue with better configuration options
network_diagnostics::fix_searxng_nginx() {
    log::info "Attempting automatic SearXNG configuration fix..."
    
    # Option 1: Try to find and fix SearXNG configuration directly
    local searxng_config_found=false
    local searxng_config_file=""
    
    # Common SearXNG config locations
    local possible_configs=(
        "/etc/searxng/uwsgi.ini"
        "/opt/searxng/uwsgi.ini" 
        "/usr/local/etc/searxng/uwsgi.ini"
        "./uwsgi.ini"
        "$(find /home -name "uwsgi.ini" -path "*/searxng/*" 2>/dev/null | head -1)"
        "$(find /opt -name "uwsgi.ini" -path "*searxng*" 2>/dev/null | head -1)"
    )
    
    for config in "${possible_configs[@]}"; do
        if [[ -n "$config" ]] && [[ -f "$config" ]]; then
            if grep -q "http-socket" "$config" 2>/dev/null; then
                searxng_config_found=true
                searxng_config_file="$config"
                log::info "  → Found SearXNG config: $config"
                break
            fi
        fi
    done
    
    if [[ "$searxng_config_found" == "true" ]]; then
        log::info "  → Found SearXNG configuration with http-socket issue"
        log::info "  → Attempting direct configuration fix (better than nginx)..."
        
        # Create backup and fix configuration
        if sudo cp "$searxng_config_file" "${searxng_config_file}.backup.$(date +%Y%m%d-%H%M%S)" 2>/dev/null; then
            log::success "  ✓ Backed up original config"
            
            # Fix the configuration: change http-socket to http
            if sudo sed -i 's/^http-socket\s*=\s*\(.*\)/http = \1/' "$searxng_config_file" 2>/dev/null; then
                log::success "  ✓ Fixed SearXNG config: changed http-socket to http"
                log::info "  → This allows direct HTTP access without uwsgi protocol"
                
                # Try to restart SearXNG service
                local restart_success=false
                for service_name in searxng searx uwsgi-searxng; do
                    if systemctl is-active "$service_name" >/dev/null 2>&1; then
                        log::info "  → Restarting $service_name service..."
                        if sudo systemctl restart "$service_name" 2>/dev/null; then
                            log::success "  ✓ Restarted $service_name"
                            restart_success=true
                            break
                        fi
                    fi
                done
                
                if [[ "$restart_success" != "true" ]]; then
                    log::warning "  → Could not find/restart SearXNG service automatically"
                    log::info "  → Manual restart: sudo systemctl restart searxng (or similar)"
                fi
                
                # Test the fix
                log::info "  → Testing SearXNG HTTP access after config fix..."
                sleep 2  # Give service time to restart
                set +e
                timeout 5 curl -s --connect-timeout 3 --max-time 5 http://localhost:9200 >/dev/null 2>&1
                local direct_http_result=$?
                set -e
                
                if [[ $direct_http_result -eq 0 ]]; then
                    log::success "  🎉 SearXNG now responds to HTTP requests directly!"
                    log::info "  → SearXNG accessible via: http://localhost:9200"
                    log::info "  → No additional services needed"
                    
                    TEST_RESULTS["SearXNG HTTP fix (port 9200)"]="PASS"
                    TEST_RESULTS["SearXNG uWSGI issue resolved"]="PASS"
                    return 0
                else
                    log::warning "  → Configuration updated but service may need manual restart"
                    log::info "  → Try: sudo systemctl restart searxng"
                fi
            else
                log::warning "  → Could not modify SearXNG configuration file"
            fi
        else
            log::warning "  → Could not backup SearXNG configuration (no sudo access)"
        fi
    fi
    
    # Fallback: nginx option (only if really needed)
    log::info "Direct configuration fix not available. Alternative options:"
    log::info "1. RECOMMENDED: Fix SearXNG config manually:"
    log::info "   - Find uwsgi.ini file (likely in /etc/searxng/ or /opt/searxng/)"
    log::info "   - Change 'http-socket = :9200' to 'http = :9200'"
    log::info "   - Restart SearXNG service"
    log::info "2. Use uwsgi protocol client instead of HTTP"
    log::info "3. Reconfigure SearXNG to use different port/protocol"
    
    # Only suggest nginx if user specifically wants it
    if command -v nginx >/dev/null 2>&1; then
        log::info "4. OPTIONAL: nginx reverse proxy (adds complexity for personal server)"
    fi
    
    return 1
}

# Verbose HTTPS debugging for failed tests
network_diagnostics::verbose_https_debug() {
    local original_command="$1"
    
    # Extract URL from the command
    local url=""
    if [[ "$original_command" =~ https://([^ ]+) ]]; then
        url="${BASH_REMATCH[0]}"
    else
        return  # Can't extract URL
    fi
    
    log::info "  → Running verbose diagnostics for $url..."
    
    # Create temporary file for verbose output
    local temp_file=$(mktemp /tmp/curl-debug-XXXXXX.txt)
    
    # Run curl with maximum verbosity
    set +e
    timeout 15 curl -v \
        --trace-ascii "$temp_file" \
        --trace-time \
        --connect-timeout 5 \
        --max-time 10 \
        --http1.1 \
        -o /dev/null \
        "$url" 2>&1 | while IFS= read -r line; do
            # Filter and display key information
            if [[ "$line" =~ ^(\*|>|<) ]] || [[ "$line" =~ "SSL" ]] || [[ "$line" =~ "TLS" ]] || [[ "$line" =~ "error" ]]; then
                log::info "    $line"
            fi
        done
    local curl_exit=$?
    set -e
    
    # Show exit code
    log::info "    → curl exit code: $curl_exit"
    
    # Check for specific error patterns in trace
    if grep -q "SSL_ERROR\|SSL certificate problem\|NSS error" "$temp_file" 2>/dev/null; then
        log::warning "    → SSL/TLS certificate error detected"
    fi
    
    if grep -q "Connection refused\|couldn't connect to host" "$temp_file" 2>/dev/null; then
        log::warning "    → Connection refused at TCP level"
    fi
    
    if grep -q "Operation timed out\|Timeout was reached" "$temp_file" 2>/dev/null; then
        log::warning "    → Connection timeout detected"
    fi
    
    if grep -q "CURLE_COULDNT_RESOLVE_HOST" "$temp_file" 2>/dev/null; then
        log::warning "    → DNS resolution failed"
    fi
    
    # Extract TLS handshake details if present
    if grep -q "SSL connection using" "$temp_file" 2>/dev/null; then
        local tls_info=$(grep "SSL connection using" "$temp_file" | head -1)
        log::info "    → TLS: $tls_info"
    fi
    
    # Show server certificate info if available
    if grep -q "subject:" "$temp_file" 2>/dev/null; then
        local cert_subject=$(grep "subject:" "$temp_file" | head -1)
        log::info "    → Cert: $cert_subject"
    fi
    
    # Clean up
    rm -f "$temp_file"
}

# Test wrapper function
run_test() {
    local test_name="$1"
    local test_command="$2"
    local is_critical="${3:-false}"
    
    log::info "Testing: $test_name"
    # Use explicit if-else to avoid script exit on test failure
    set +e  # Temporarily disable exit on error
    eval "$test_command" >/dev/null 2>&1
    local test_result=$?
    set -e  # Re-enable exit on error
    
    if [[ $test_result -eq 0 ]]; then
        TEST_RESULTS["$test_name"]="PASS"
        log::success "  ✓ $test_name"
        return 0
    else
        TEST_RESULTS["$test_name"]="FAIL"
        log::error "  ✗ $test_name"
        
        # Run verbose diagnostics for failed HTTPS tests
        if [[ "$test_name" =~ HTTPS ]] && [[ "$test_command" =~ curl.*https:// ]]; then
            network_diagnostics::verbose_https_debug "$test_command"
        fi
        
        if [[ "$is_critical" == "true" ]]; then
            CRITICAL_FAILURE=true
        fi
        return 0  # Don't exit the script, just continue testing
    fi
}

# Main diagnostics function
network_diagnostics::run() {
    log::header "Running Comprehensive Network Diagnostics"
    
    # 1. Basic connectivity tests
    log::subheader "Basic Connectivity"
    run_test "IPv4 ping to 8.8.8.8" "ping -4 -c 1 -W 2 8.8.8.8" true
    run_test "IPv4 ping to google.com" "ping -4 -c 1 -W 2 google.com"
    run_test "IPv6 ping to google.com" "ping -6 -c 1 -W 2 google.com 2>/dev/null"
    
    # 2. DNS resolution tests
    log::subheader "DNS Resolution"
    run_test "DNS lookup (getent)" "getent hosts google.com"
    run_test "DNS lookup (nslookup)" "command -v nslookup >/dev/null && nslookup google.com"
    run_test "DNS via 8.8.8.8" "command -v dig >/dev/null && dig @8.8.8.8 google.com +short"
    
    # 3. TCP connectivity tests
    log::subheader "TCP Connectivity"
    run_test "TCP port 80 (HTTP)" "nc -zv -w 2 google.com 80"
    run_test "TCP port 443 (HTTPS)" "nc -zv -w 2 google.com 443"
    run_test "TCP port 22 (SSH)" "nc -zv -w 2 github.com 22"
    
    # 4. HTTP protocol tests
    log::subheader "HTTP Protocol Tests"
    
    # HTTP/1.0
    if command -v curl >/dev/null 2>&1; then
        run_test "HTTP/1.0 to google.com" "timeout 5 curl -s --http1.0 --connect-timeout 3 --max-time 5 http://google.com"
        run_test "HTTP/1.1 to google.com" "timeout 5 curl -s --http1.1 --connect-timeout 3 --max-time 5 http://google.com"
        
        # Raw HTTP test with netcat
        run_test "Raw HTTP with netcat" "echo -e 'GET / HTTP/1.0\r\n\r\n' | timeout 3 nc google.com 80 | grep -q 'HTTP'"
    fi
    
    # 5. HTTPS/TLS tests
    log::subheader "HTTPS/TLS Tests"
    
    if command -v curl >/dev/null 2>&1; then
        # Pool of 20 diverse URLs to test different services and CDNs
        local url_pool=(
            "https://www.google.com"
            "https://github.com"
            "https://gitlab.com"
            "https://bitbucket.org"
            "https://www.cloudflare.com"
            "https://www.amazon.com"
            "https://www.microsoft.com"
            "https://stackoverflow.com"
            "https://www.wikipedia.org"
            "https://www.reddit.com"
            "https://www.twitter.com"
            "https://www.facebook.com"
            "https://www.youtube.com"
            "https://www.linkedin.com"
            "https://www.apple.com"
            "https://www.netflix.com"
            "https://www.spotify.com"
            "https://api.github.com"
            "https://registry.npmjs.org"
            "https://pypi.org"
        )
        
        # Shuffle the array to randomize selection
        local shuffled=()
        local indices=()
        for i in "${!url_pool[@]}"; do
            indices+=($i)
        done
        
        # Simple shuffle using RANDOM
        for ((i=${#indices[@]}-1; i>0; i--)); do
            j=$((RANDOM % (i+1)))
            # Swap elements
            tmp=${indices[i]}
            indices[i]=${indices[j]}
            indices[j]=$tmp
        done
        
        # Select first 5 URLs from shuffled array
        log::info "Testing random selection of 5 URLs from pool of ${#url_pool[@]}..."
        local selected_count=0
        for idx in "${indices[@]}"; do
            if [[ $selected_count -ge 5 ]]; then
                break
            fi
            
            local url="${url_pool[$idx]}"
            local domain=$(echo "$url" | sed 's|https://||; s|/.*||')
            
            # Basic HTTPS test
            run_test "HTTPS to $domain" "timeout 8 curl -s --http1.1 --connect-timeout 5 $url"
            
            # Also test TLS 1.2 specifically for the first 2 selected URLs
            if [[ $selected_count -lt 2 ]]; then
                run_test "TLS 1.2 to $domain" "timeout 8 curl -s --tlsv1.2 --http1.1 --connect-timeout 5 --max-time 8 $url"
            fi
            
            selected_count=$((selected_count + 1))
        done
        
        # Always test with modern ciphers on one URL
        run_test "HTTPS with modern ciphers" "timeout 8 curl -s --ciphers ECDHE+AESGCM --http1.1 --connect-timeout 5 https://www.cloudflare.com"
    fi
    
    # 6. Local service tests
    log::subheader "Local Services"
    
    # Test common local ports
    run_test "Localhost port 80" "nc -zv -w 1 localhost 80 2>/dev/null"
    run_test "Localhost port 8080" "nc -zv -w 1 localhost 8080 2>/dev/null"
    run_test "Localhost port 9200 (SearXNG)" "nc -zv -w 1 localhost 9200 2>/dev/null"
    
    # If port 9200 is open, test HTTP response
    if nc -z localhost 9200 2>/dev/null; then
        run_test "HTTP to localhost:9200" "timeout 3 curl -s --connect-timeout 2 --max-time 3 http://localhost:9200/stats"
    fi
    
    # Re-test SearXNG HTTP if it was potentially fixed
    if [[ "${TEST_RESULTS["HTTP to localhost:9200"]:-}" == "FAIL" ]] && nc -z localhost 9200 2>/dev/null; then
        run_test "SearXNG HTTP retest" "timeout 3 curl -s --connect-timeout 2 --max-time 3 http://localhost:9200"
    fi
    
    # 7. Network configuration checks
    log::subheader "Network Configuration"
    
    # Check MTU
    local primary_iface=$(ip route | grep default | awk '{print $5}' | head -1)
    if [[ -n "$primary_iface" ]]; then
        local mtu=$(ip link show "$primary_iface" | grep -oP 'mtu \K\d+')
        log::info "Primary interface MTU: $mtu"
    fi
    
    # Check for proxy settings
    if [[ -n "${http_proxy:-}" ]] || [[ -n "${https_proxy:-}" ]]; then
        log::warning "Proxy configured: ${http_proxy:-}${https_proxy:-}"
    else
        log::info "No proxy configured"
    fi
    
    # Check TCP offload settings
    if command -v ethtool >/dev/null 2>&1 && [[ -n "$primary_iface" ]]; then
        local tso_status=$(ethtool -k "$primary_iface" 2>/dev/null | grep "tcp-segmentation-offload" | awk '{print $2}')
        log::info "TCP segmentation offload on $primary_iface: $tso_status"
        if [[ "$tso_status" == "on" ]]; then
            TEST_RESULTS["tso_enabled"]="YES"
        else
            TEST_RESULTS["tso_enabled"]="NO"
        fi
    fi
    
    # New diagnostic checks
    network_diagnostics::check_pmtu_status
    network_diagnostics::check_tcp_settings
    network_diagnostics::check_time_sync
    
    # 8. Advanced diagnostic tests for issues detected
    # Check if any HTTPS tests failed
    local failed_https_domains=()
    for test_name in "${!TEST_RESULTS[@]}"; do
        if [[ "$test_name" =~ ^"HTTPS to " ]] && [[ "${TEST_RESULTS[$test_name]}" == "FAIL" ]]; then
            # Extract domain from test name
            local domain=$(echo "$test_name" | sed 's/HTTPS to //')
            failed_https_domains+=("$domain")
        fi
    done
    
    # Run advanced diagnostics on first failed domain (or github.com if in list)
    local diagnostic_domain=""
    if [[ ${#failed_https_domains[@]} -gt 0 ]]; then
        # Prefer github.com if it failed, otherwise use first failed domain
        for domain in "${failed_https_domains[@]}"; do
            if [[ "$domain" == "github.com" ]]; then
                diagnostic_domain="github.com"
                break
            fi
        done
        
        if [[ -z "$diagnostic_domain" ]]; then
            diagnostic_domain="${failed_https_domains[0]}"
        fi
        
        log::info "Running advanced diagnostics for failed HTTPS domain: $diagnostic_domain"
        
        # Run MTU discovery test
        network_diagnostics::test_mtu_discovery "$diagnostic_domain"
        
        # Analyze TLS handshake
        if command -v openssl >/dev/null 2>&1; then
            # Call the function but ensure script continues even if it fails
            network_diagnostics::analyze_tls_handshake "$diagnostic_domain" || true
        else
            log::info "  → OpenSSL not available, skipping TLS handshake analysis"
        fi
        
        # Run IPv4 vs IPv6 diagnostics
        log::info "Running IPv4 vs IPv6 diagnostics..."
        network_diagnostics::check_ipv4_vs_ipv6 "$diagnostic_domain" || log::warning "IPv4/IPv6 check failed, continuing..."
        network_diagnostics::check_ip_preference || log::warning "IP preference check failed, continuing..."
        log::info "Completed advanced diagnostics"
    fi
    
    # 9. Automatic Network Optimizations
    log::subheader "Automatic Network Optimizations"
    
    # Check if we have TLS issues that might be fixable
    local has_tls_issues=false
    local has_tcp_issues=false
    
    # Safely iterate over test results
    if [[ ${#TEST_RESULTS[@]} -gt 0 ]]; then
        for test_name in "${!TEST_RESULTS[@]}"; do
            if [[ "$test_name" =~ TLS|HTTPS ]] && [[ "${TEST_RESULTS[$test_name]}" != "PASS" ]]; then
                has_tls_issues=true
            fi
            if [[ "$test_name" =~ TCP ]] && [[ "${TEST_RESULTS[$test_name]}" != "PASS" ]]; then
                has_tcp_issues=true
            fi
        done
    fi
    
    # Attempt automatic fixes if we have permission and issues exist
    if [[ "$has_tls_issues" == "true" ]] || [[ "$has_tcp_issues" == "true" ]]; then
        log::info "Network issues detected. Attempting automatic optimizations..."
        
        # 1. Flush DNS cache (safe)
        log::info "Flushing DNS cache..."
        if flow::can_run_sudo "DNS cache flush"; then
            # Try multiple DNS cache clearing methods
            local dns_flushed=false
            
            # Method 1: systemd-resolved (most common on modern systems)
            if systemctl is-active systemd-resolved >/dev/null 2>&1; then
                if sudo systemd-resolve --flush-caches 2>/dev/null; then
                    log::success "  ✓ systemd-resolved DNS cache flushed"
                    dns_flushed=true
                fi
            fi
            
            # Method 2: systemctl flush-dns (some systems)
            if ! $dns_flushed && sudo systemctl flush-dns 2>/dev/null; then
                log::success "  ✓ systemctl DNS cache flushed"
                dns_flushed=true
            fi
            
            # Method 3: NetworkManager (if active)
            if ! $dns_flushed && systemctl is-active NetworkManager >/dev/null 2>&1; then
                if sudo systemctl restart NetworkManager 2>/dev/null; then
                    log::success "  ✓ NetworkManager DNS cache cleared"
                    dns_flushed=true
                fi
            fi
            
            if ! $dns_flushed; then
                log::warning "  → Could not flush DNS cache (methods failed)"
            fi
        else
            log::warning "  → DNS cache flush requires sudo access"
            log::info "  → SOLUTION: Run with sudo or configure passwordless sudo"
            log::info "  → Manual fix: sudo systemd-resolve --flush-caches"
            log::info "  → Alternative: sudo systemctl restart systemd-resolved"
        fi
        
        # 2. Test with TCP optimizations (Enhanced with automatic fix)
        local primary_iface=$(ip route | grep default | awk '{print $5}' | head -1)
        if [[ -n "$primary_iface" ]] && command -v ethtool >/dev/null 2>&1; then
            log::info "Testing TCP segmentation offload settings on $primary_iface..."
            
            # Check current settings
            local current_tso=$(ethtool -k "$primary_iface" 2>/dev/null | grep "tcp-segmentation-offload" | awk '{print $2}')
            local current_gso=$(ethtool -k "$primary_iface" 2>/dev/null | grep "generic-segmentation-offload" | awk '{print $2}')
            
            if [[ "$current_tso" == "on" ]] || [[ "$current_gso" == "on" ]]; then
                log::warning "  → TCP/GSO offload is enabled (likely causing GitHub TLS issues)"
                
                # Check if we can automatically apply the TSO fix
                if flow::can_run_sudo "TCP segmentation offload fix"; then
                    log::info "Testing TCP offload fix automatically..."
                    
                    # Apply temporary fix
                    if sudo ethtool -K "$primary_iface" tso off gso off 2>/dev/null; then
                        log::success "  ✓ TCP offload temporarily disabled"
                        
                        # Test GitHub specifically (the main issue)
                        log::info "  → Testing GitHub HTTPS with fix applied..."
                        set +e
                        timeout 8 curl -s --http1.1 --connect-timeout 5 --max-time 8 https://github.com >/dev/null 2>&1
                        local github_test_result=$?
                        set -e
                        
                        if [[ $github_test_result -eq 0 ]]; then
                            log::success "  🎉 GitHub HTTPS now works with TCP offload disabled!"
                            log::warning "  → This fix should be made permanent"
                            
                            # Offer to make permanent
                            log::info "Make TCP offload fix permanent? This will improve TLS reliability. [Y/n]"
                            if read -r -t 15 response 2>/dev/null; then
                                if [[ ! "$response" =~ ^[Nn]$ ]]; then
                                    # Apply permanent fix
                                    network_diagnostics::make_tso_permanent "$primary_iface"
                                fi
                            else
                                log::info "No response, making permanent by default for better TLS reliability..."
                                network_diagnostics::make_tso_permanent "$primary_iface"
                            fi
                            
                            # Update test results to reflect the fix
                            TEST_RESULTS["HTTPS to github.com (TSO fix)"]="PASS"
                            TEST_RESULTS["GitHub TLS issue resolved"]="PASS"
                            
                        else
                            log::info "  → GitHub still failing after TCP offload fix, reverting..."
                            # Revert changes since it didn't help
                            sudo ethtool -K "$primary_iface" tso on gso on 2>/dev/null || true
                        fi
                    else
                        log::warning "  → Could not modify TCP offload settings"
                    fi
                else
                    log::warning "  → TCP offload fix requires sudo access for automatic application"
                    log::info "  → WHAT IT DOES: Disables hardware TCP segmentation that breaks modern HTTPS sites"
                    log::info "  → WHY NEEDED: GitHub, Facebook, and many new sites fail without this fix"
                    log::info "  → IMMEDIATE FIX: sudo ethtool -K $primary_iface tso off gso off"
                    log::info "  → PERMANENT FIX: source this script and run: sudo network_diagnostics::make_tso_permanent $primary_iface"
                    log::info "  → ALTERNATIVE: Run this script with sudo or configure passwordless sudo"
                fi
            else
                log::success "  ✓ TCP offload already optimized"
            fi
        fi
        
        # 3. Try PMTU probing fix
        network_diagnostics::fix_pmtu_probing
        
        # 4. Try MTU adjustment fix
        network_diagnostics::fix_mtu_size
        
        # 5. Try ECN disable fix
        network_diagnostics::fix_ecn
        
        # 6. Try IPv6 fixes if detected
        network_diagnostics::fix_ipv6_issues
        
        # 7. Try IPv4-only fixes if detected
        network_diagnostics::fix_ipv4_only_issues
        
        # 8. Test different MTU sizes for fragmentation issues
        if [[ "$has_tcp_issues" == "true" ]]; then
            log::info "Testing MTU fragmentation..."
            local current_mtu=$(ip link show "$primary_iface" 2>/dev/null | grep -oP 'mtu \K[0-9]+')
            if [[ -n "$current_mtu" ]] && [[ "$current_mtu" -gt 1400 ]]; then
                # Test if smaller packets work better
                if ping -M 'do' -s 1200 -c 1 -W 2 8.8.8.8 >/dev/null 2>&1; then
                    log::success "  ✓ Small packets (1200 bytes) work fine"
                else
                    log::warning "  → Possible MTU/fragmentation issues detected"
                    log::info "  → Consider testing with smaller MTU: sudo ip link set dev $primary_iface mtu 1400"
                fi
            fi
        fi
        
        # 9. Test alternative DNS servers
        log::info "Testing alternative DNS servers..."
        if timeout 3 dig @1.1.1.1 google.com +short >/dev/null 2>&1; then
            log::success "  ✓ Cloudflare DNS (1.1.1.1) works"
        else
            log::warning "  → Cloudflare DNS issues detected"
        fi
        
        if timeout 3 dig @8.8.8.8 google.com +short >/dev/null 2>&1; then
            log::success "  ✓ Google DNS (8.8.8.8) works"
        else
            log::warning "  → Google DNS issues detected"
        fi
        
        # 10. Check for IPv6 interference (already handled by IPv6 fix function)
        # Note: Detailed IPv6 vs IPv4 testing is now done in network_diagnostics::check_ipv4_vs_ipv6
        
        # 11. Clear Firefox cache if browser issues detected
        if [[ "${TEST_RESULTS["HTTPS to github.com"]:-FAIL}" == "FAIL" ]] || [[ "${TEST_RESULTS["HTTPS to google.com"]:-FAIL}" == "FAIL" ]]; then
            log::info "Clearing Firefox browser cache..."
            
            # Find Firefox profile directories
            local firefox_cleared=false
            local firefox_profiles=()
            
            # Determine correct user home directory (handle sudo case)
            local user_home="$HOME"
            if [[ -n "${SUDO_USER:-}" ]] && [[ "$SUDO_USER" != "root" ]]; then
                user_home="/home/$SUDO_USER"
                log::info "  → Running with sudo, checking Firefox cache for user: $SUDO_USER"
            fi
            
            # Check common Firefox profile locations
            if [[ -d "$user_home/.mozilla/firefox" ]]; then
                mapfile -t firefox_profiles < <(find "$user_home/.mozilla/firefox" -name "*.default*" -type d 2>/dev/null)
            fi
            
            if [[ ${#firefox_profiles[@]} -gt 0 ]]; then
                for profile in "${firefox_profiles[@]}"; do
                    # Clear cache2 directory (modern Firefox cache)
                    if [[ -d "$profile/cache2" ]]; then
                        if rm -rf "$profile/cache2"/* 2>/dev/null; then
                            log::success "  ✓ Cleared Firefox cache: $(basename "$profile")"
                            firefox_cleared=true
                        fi
                    fi
                    
                    # Clear old cache directory (legacy Firefox)
                    if [[ -d "$profile/Cache" ]]; then
                        if rm -rf "$profile/Cache"/* 2>/dev/null; then
                            log::success "  ✓ Cleared Firefox legacy cache: $(basename "$profile")"
                            firefox_cleared=true
                        fi
                    fi
                    
                    # Clear offline cache
                    if [[ -d "$profile/OfflineCache" ]]; then
                        rm -rf "$profile/OfflineCache"/* 2>/dev/null
                    fi
                    
                    # Clear site data that might contain connection failures
                    if [[ -f "$profile/SiteSecurityServiceState.txt" ]]; then
                        rm -f "$profile/SiteSecurityServiceState.txt" 2>/dev/null
                        log::success "  ✓ Cleared site security state: $(basename "$profile")"
                        firefox_cleared=true
                    fi
                done
                
            else
                log::info "  → No Firefox profiles found at $user_home/.mozilla/firefox"
            fi
            
            # Also try clearing system-wide Firefox cache if running with appropriate permissions
            if [[ -d "$user_home/.cache/mozilla/firefox" ]]; then
                if rm -rf "$user_home/.cache/mozilla/firefox"/* 2>/dev/null; then
                    log::success "  ✓ Cleared Firefox system cache"
                    firefox_cleared=true
                fi
            fi
            
            if $firefox_cleared; then
                log::info "  → Firefox cache cleared. Restart Firefox to see improvements."
                log::info "  → Alternative: In Firefox press Ctrl+Shift+Delete → Clear Everything"
            else
                log::info "  → Could not auto-clear Firefox cache"
                log::info "  → MANUAL FIX: In Firefox press Ctrl+Shift+Delete"
                log::info "  → Select 'Everything' timeframe and check 'Cache' + 'Site Data'"
            fi
        fi
        
        # 12. Test with different User-Agent (some sites block default curl)
        log::info "Testing with browser User-Agent..."
        if timeout 5 curl -s --user-agent "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36" --http1.1 --connect-timeout 3 https://github.com >/dev/null 2>&1; then
            log::success "  ✓ GitHub works with browser User-Agent"
            log::info "  → Some sites block default curl User-Agent"
        fi
        
        # 13. Quick retest of critical issues after optimizations
        log::info "Re-testing critical HTTPS connections after optimizations..."
        local retest_passed=0
        local retest_total=0
        
        
        # Use shorter timeouts and more robust error handling for retesting
        if [[ "${TEST_RESULTS["HTTPS to google.com"]:-FAIL}" == "FAIL" ]]; then
            retest_total=$((retest_total + 1))
            log::info "  → Re-testing Google HTTPS..."
            set +e
            timeout 3 curl -s --http1.1 --connect-timeout 2 --max-time 3 https://www.google.com >/dev/null 2>&1
            local google_retest=$?
            set -e
            if [[ $google_retest -eq 0 ]]; then
                log::success "  ✓ Google HTTPS now works!"
                retest_passed=$((retest_passed + 1))
                TEST_RESULTS["HTTPS to google.com (retest)"]="PASS"
            else
                log::info "  → Google HTTPS still failing"
                TEST_RESULTS["HTTPS to google.com (retest)"]="FAIL"
            fi
        else
            log::info "  → Google HTTPS already working, skipping retest"
        fi
        
        local github_https_status="${TEST_RESULTS["HTTPS to github.com"]:-FAIL}"
        if [[ "$github_https_status" == "FAIL" ]]; then
            retest_total=$((retest_total + 1))
            log::info "  → Re-testing GitHub HTTPS..."
            set +e
            timeout 3 curl -s --http1.1 --connect-timeout 2 --max-time 3 https://github.com >/dev/null 2>&1
            local github_retest=$?
            set -e
            if [[ $github_retest -eq 0 ]]; then
                log::success "  ✓ GitHub HTTPS now works!"
                retest_passed=$((retest_passed + 1))
                TEST_RESULTS["HTTPS to github.com (retest)"]="PASS"
            else
                log::info "  → GitHub HTTPS still failing"
                TEST_RESULTS["HTTPS to github.com (retest)"]="FAIL"
            fi
        else
            log::info "  → GitHub HTTPS already working, skipping retest"
        fi
        
        
        if [[ $retest_total -gt 0 ]]; then
            if [[ $retest_passed -eq $retest_total ]]; then
                log::success "🎉 All HTTPS issues resolved by automatic optimizations!"
            elif [[ $retest_passed -gt 0 ]]; then
                log::warning "⚠️ Some HTTPS issues resolved ($retest_passed/$retest_total)"
            else
                log::info "→ No improvement from automatic optimizations"
            fi
        else
            log::info "→ No failed HTTPS tests to recheck"
        fi
        
        
    else
        log::success "No network issues detected, skipping optimizations"
    fi
    
    # 12. Summary
    log::header "Network Diagnostics Summary"
    
    local total_tests=${#TEST_RESULTS[@]}
    local passed_tests=0
    local failed_tests=0
    
    for result in "${TEST_RESULTS[@]}"; do
        case "$result" in
            "PASS") passed_tests=$((passed_tests + 1)) ;;
            *) failed_tests=$((failed_tests + 1)) ;;
        esac
    done
    
    log::info "Total tests: $total_tests"
    log::success "Passed: $passed_tests"
    log::error "Failed: $failed_tests"
    
    # Check if any fixes were applied
    local fixes_applied=false
    local tso_fix_applied=false
    local nginx_fix_applied=false
    local ipv6_fix_applied=false
    local ipv4_fix_applied=false
    
    if [[ ${#TEST_RESULTS[@]} -gt 0 ]]; then
        for test_name in "${!TEST_RESULTS[@]}"; do
            if [[ "$test_name" =~ "TSO fix"|"GitHub TLS issue resolved" ]] && [[ "${TEST_RESULTS[$test_name]}" == "PASS" ]]; then
                fixes_applied=true
                tso_fix_applied=true
            fi
            if [[ "$test_name" =~ "SearXNG HTTP fix"|"SearXNG uWSGI issue resolved" ]] && [[ "${TEST_RESULTS[$test_name]}" == "PASS" ]]; then
                fixes_applied=true
                nginx_fix_applied=true
            fi
            if [[ "$test_name" =~ "ipv6_fix_success"|"ipv6_disabled_fix"|"ipv4_preference_set" ]] && [[ "${TEST_RESULTS[$test_name]}" == "YES" ]]; then
                fixes_applied=true
                ipv6_fix_applied=true
            fi
            if [[ "$test_name" =~ "tls12_only_fix"|"modern_ciphers_fix"|"user_agent_fix"|"http2_fix"|"direct_ip_fix"|"cloudflare_dns_fix"|"google_dns_fix"|"combined_fix" ]] && [[ "${TEST_RESULTS[$test_name]}" == "YES" ]]; then
                fixes_applied=true
                ipv4_fix_applied=true
            fi
        done
    fi
    
    if [[ "$fixes_applied" == "true" ]]; then
        log::subheader "Applied Fixes Summary"
        if [[ "$tso_fix_applied" == "true" ]]; then
            log::success "✅ TCP Segmentation Offload fix applied - GitHub TLS issues resolved"
        fi
        if [[ "$nginx_fix_applied" == "true" ]]; then
            log::success "✅ SearXNG HTTP configuration fixed - accessible on port 9200"
        fi
        if [[ "$ipv6_fix_applied" == "true" ]]; then
            if [[ "${TEST_RESULTS["ipv6_disabled_fix"]:-}" == "YES" ]]; then
                log::success "✅ IPv6 disabled system-wide - connections now use IPv4 only"
            elif [[ "${TEST_RESULTS["ipv4_preference_set"]:-}" == "YES" ]]; then
                log::success "✅ System configured to prefer IPv4 - resolves dual-stack connectivity issues"
            fi
        fi
        if [[ "$ipv4_fix_applied" == "true" ]]; then
            local ipv4_fix_details=""
            if [[ "${TEST_RESULTS["tls12_only_fix"]:-}" == "YES" ]]; then
                ipv4_fix_details+="TLS 1.2 forced, "
            fi
            if [[ "${TEST_RESULTS["cloudflare_dns_fix"]:-}" == "YES" ]]; then
                ipv4_fix_details+="Cloudflare DNS, "
            elif [[ "${TEST_RESULTS["google_dns_fix"]:-}" == "YES" ]]; then
                ipv4_fix_details+="Google DNS, "
            fi
            if [[ "${TEST_RESULTS["user_agent_fix"]:-}" == "YES" ]]; then
                ipv4_fix_details+="Firefox User-Agent, "
            fi
            if [[ "${TEST_RESULTS["combined_fix"]:-}" == "YES" ]]; then
                log::success "✅ IPv4-only connectivity restored - combined fix applied ($ipv4_fix_details)"
            else
                log::success "✅ IPv4-only connectivity improved - specific fixes applied ($ipv4_fix_details)"
            fi
        fi
        log::info ""
    fi
    
    # Analyze results and provide diagnosis
    log::subheader "Diagnosis"
    
    # Check for specific patterns (including applied fixes)
    local has_basic_connectivity="${TEST_RESULTS["IPv4 ping to 8.8.8.8"]:-FAIL}"
    local has_dns="${TEST_RESULTS["DNS lookup (getent)"]:-FAIL}"
    local has_tcp="${TEST_RESULTS["TCP port 80 (HTTP)"]:-FAIL}"
    local has_http="${TEST_RESULTS["HTTP/1.0 to google.com"]:-FAIL}"
    local has_https_google="${TEST_RESULTS["HTTPS to google.com"]:-FAIL}"
    local has_https_github="${TEST_RESULTS["HTTPS to github.com"]:-FAIL}"
    local has_tls12_google="${TEST_RESULTS["TLS 1.2 to google.com"]:-FAIL}"
    local has_tls13_google="${TEST_RESULTS["TLS 1.3 to google.com"]:-FAIL}"
    
    # Check if fixes resolved issues
    local github_fixed_by_tso="${TEST_RESULTS["HTTPS to github.com (TSO fix)"]:-FAIL}"
    local searxng_fixed_by_nginx="${TEST_RESULTS["SearXNG HTTP fix (port 9200)"]:-FAIL}"
    
    # Detailed analysis based on test patterns
    if [[ "$has_basic_connectivity" == "PASS" ]] && [[ "$has_dns" == "PASS" ]]; then
        if [[ "$has_tcp" == "PASS" ]] && [[ "$has_http" == "PASS" ]]; then
            
            # Analyze HTTPS patterns
            if [[ "$has_https_google" == "FAIL" ]] && [[ "$has_https_github" == "FAIL" ]]; then
                if [[ "$has_tls12_google" == "PASS" ]] || [[ "$has_tls13_google" == "PASS" ]]; then
                    log::warning "PARTIAL TLS ISSUE: Domain or TLS version negotiation problems"
                    log::info "- Basic networking works (ping, DNS, TCP, HTTP)"
                    log::info "- HTTPS fails with default settings"
                    log::info "- But specific TLS versions work"
                    log::info "- This suggests TLS version negotiation or cipher suite issues"
                    log::info ""
                    log::info "Likely causes:"
                    log::info "1. TLS version negotiation problems"
                    log::info "2. Cipher suite incompatibility"  
                    log::info "3. TCP segmentation offload (TSO) issues"
                    log::info "4. Some sites require specific User-Agent headers"
                else
                    log::error "CRITICAL: Complete TLS/HTTPS failure"
                    log::warning "- All HTTPS connections fail including forced TLS versions"
                    log::warning "- This suggests deep packet inspection or driver issues"
                fi
            elif [[ "$has_https_google" == "PASS" ]] && [[ "$has_https_github" == "FAIL" ]]; then
                if [[ "$github_fixed_by_tso" == "PASS" ]]; then
                    log::success "GITHUB ISSUES RESOLVED: TCP offload fix successful"
                    log::info "- GitHub HTTPS was failing due to TCP segmentation offload"
                    log::info "- TSO fix has been applied and GitHub now works"
                    log::info "- Fix has been made permanent to prevent recurrence"
                else
                    # Check for selective HTTPS failures
                    local https_failures=0
                    local https_successes=0
                    local failed_domains=()
                    
                    for test_name in "${!TEST_RESULTS[@]}"; do
                        if [[ "$test_name" =~ ^"HTTPS to " ]]; then
                            local domain=$(echo "$test_name" | sed 's/HTTPS to //')
                            if [[ "${TEST_RESULTS[$test_name]}" == "FAIL" ]]; then
                                https_failures=$((https_failures + 1))
                                failed_domains+=("$domain")
                            else
                                https_successes=$((https_successes + 1))
                            fi
                        fi
                    done
                    
                    if [[ $https_failures -gt 0 ]] && [[ $https_successes -gt 0 ]]; then
                        log::warning "SELECTIVE HTTPS ISSUES: Some domains fail while others work"
                        log::info "- Failed domains: ${failed_domains[*]}"
                        log::info "- This suggests domain-specific blocking or stricter TLS requirements"
                        log::info "- May be due to TCP segmentation offload, SNI filtering, or middlebox interference"
                    elif [[ $https_failures -gt 0 ]]; then
                        log::error "GENERAL HTTPS FAILURE: All tested HTTPS connections failed"
                        log::info "- This suggests a system-wide TLS/SSL issue"
                        log::info "- Check firewall settings, proxy configuration, or certificate stores"
                    fi
                fi
            else
                log::success "HTTPS connectivity appears to be working"
            fi
            
        elif [[ "$has_tcp" == "FAIL" ]]; then
            log::error "TCP connectivity issues detected"
            log::warning "- Basic ping works but TCP connections fail"
            log::warning "- This suggests firewall or routing issues"
        fi
    elif [[ "$has_basic_connectivity" == "FAIL" ]]; then
        log::error "No basic network connectivity"
        log::warning "- Cannot reach external networks"  
        log::warning "- Check network cable/WiFi and gateway configuration"
    fi
    
    # Check for localhost issues (Enhanced with automatic nginx fix)
    if [[ "${TEST_RESULTS["Localhost port 9200 (SearXNG)"]:-FAIL}" == "PASS" ]] && 
       [[ "${TEST_RESULTS["HTTP to localhost:9200"]:-FAIL}" == "FAIL" ]]; then
        
        if [[ "$searxng_fixed_by_nginx" == "PASS" ]]; then
            log::info ""
            log::success "SearXNG uWSGI issue resolved:"
            log::info "- Configuration automatically fixed to accept HTTP requests"
            log::info "- SearXNG is now accessible via standard HTTP on port 9200"
            log::info "- No additional services needed"
        else
            log::warning ""
            log::warning "SearXNG uWSGI HTTP-socket issue detected:"
            log::warning "- Port 9200 is open but HTTP requests hang"
            log::warning "- uWSGI HTTP-socket expects uwsgi protocol, not HTTP"
            
            # Attempt automatic configuration fix
            network_diagnostics::fix_searxng_nginx || true
        fi
    fi
    
    # Run enhanced diagnosis for more specific recommendations
    log::info "Running enhanced diagnosis..."
    network_diagnostics::enhanced_diagnosis
    
    # Provide specific recommendations based on patterns
    log::info ""
    log::info "Recommended next steps:"
    
    # Only show recommendations for issues that haven't been automatically fixed
    if [[ "$has_tls12_google" == "PASS" ]] && [[ "$has_https_google" == "FAIL" ]]; then
        log::info "For remaining TLS issues:"
        log::info "1. Configure applications to prefer TLS 1.2/1.3 explicitly"
        log::info "2. Try: export CURL_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt"
        if [[ "$tso_fix_applied" != "true" ]]; then
            log::info "3. Test TCP offload fix: sudo ethtool -K \$(ip route | grep default | awk '{print \$5}') tso off gso off"
        fi
    fi
    
    if [[ "$has_https_github" == "FAIL" ]] && [[ "$github_fixed_by_tso" != "PASS" ]]; then
        log::info "For remaining git/GitHub issues:"
        log::info "1. Use SSH instead: git remote set-url origin git@github.com:user/repo.git"
        log::info "2. Or try: git config --global http.version HTTP/1.1"
        log::info "3. Or try: git config --global http.sslVersion tlsv1.2"
        if [[ "$tso_fix_applied" != "true" ]]; then
            log::info "4. Apply TCP offload fix manually: sudo ethtool -K \$(ip route | grep default | awk '{print \$5}') tso off gso off"
        fi
    elif [[ "$github_fixed_by_tso" == "PASS" ]]; then
        log::info "For git/GitHub (resolved):"
        log::success "✓ GitHub HTTPS connectivity restored via TCP offload fix"
        log::info "1. Test git operations: git ls-remote https://github.com/octocat/Hello-World.git"
        log::info "2. You can now use HTTPS URLs for git operations"
    fi
    
    if [[ "$searxng_fixed_by_nginx" == "PASS" ]]; then
        log::info "For SearXNG (resolved):"
        log::success "✓ SearXNG HTTP access fixed - now responds properly on port 9200"
        log::info "1. Access SearXNG: http://localhost:9200"
        log::info "2. SearXNG now accepts standard HTTP requests"
    fi
    
    # 10. UFW Firewall Check and Fix
    log::subheader "Firewall (UFW) Analysis"
    network_diagnostics::check_ufw_blocking
    
    # Provide helpful guidance based on results
    log::info "Proceeding to setup readiness assessment..."
    log::subheader "Setup Readiness Assessment"
    
    if [[ "$CRITICAL_FAILURE" == "true" ]]; then
        log::error "❌ CRITICAL: Basic network connectivity is broken"
        log::info ""
        log::info "🔍 What this means:"
        log::info "   • Your system cannot reach the internet for basic operations"
        log::info "   • Setup scripts cannot download dependencies or updates"
        log::info "   • Package managers (apt, npm, etc.) won't work"
        log::info ""
        log::info "⚠️  Why this blocks setup:"
        log::info "   • Setup requires downloading Docker images, packages, and dependencies"
        log::info "   • Without basic connectivity, the setup process will fail"
        log::info ""
        log::info "🛠️  How to proceed:"
        log::info "   1. Check your network connection (WiFi/Ethernet)"
        log::info "   2. Verify your router/gateway is working"
        log::info "   3. Try: ping 8.8.8.8"
        log::info "   4. Contact your network administrator if needed"
        log::info ""
        log::warning "Setup cannot continue until basic connectivity is restored."
        return "${ERROR_NO_INTERNET}"
        
    elif [[ $failed_tests -gt $((total_tests / 2)) ]]; then
        log::warning "⚠️  MAJOR: Significant network issues detected ($failed_tests/$total_tests tests failed)"
        log::info ""
        log::info "🔍 What this means:"
        log::info "   • Most advanced network features are not working properly"
        log::info "   • Setup may fail or encounter frequent errors"
        log::info "   • Some services may not function correctly"
        log::info ""
        log::info "⚠️  Impact on setup:"
        log::info "   • High likelihood of download failures"
        log::info "   • Docker image pulls may timeout"
        log::info "   • SSL/TLS connections may fail"
        log::info ""
        log::info "🛠️  Recommended actions:"
        log::info "   1. Review the specific failures above"
        log::info "   2. Try the automatic optimizations if available"
        log::info "   3. Consider fixing major issues before continuing"
        log::info ""
        log::warning "Continue with setup anyway? (high risk of failures) [y/N]"
        if read -r -t 10 response 2>/dev/null; then
            if [[ ! "$response" =~ ^[Yy]$ ]]; then
                log::info "Setup cancelled. Fix network issues and try again."
                return "${ERROR_NO_INTERNET}"
            fi
        else
            log::info "No response (timed out or non-interactive), defaulting to cancel setup."
            return "${ERROR_NO_INTERNET}"
        fi
        log::warning "⚠️  Continuing despite major network issues..."
        
    elif [[ $failed_tests -gt 0 ]]; then
        # Calculate risk level based on specific failures
        local risk_level="LOW"
        local github_fails=false
        local tls_issues=false
        
        # Check for specific high-impact failures (but account for fixes)
        if [[ "${TEST_RESULTS["HTTPS to github.com"]:-PASS}" == "FAIL" ]] && [[ "$github_fixed_by_tso" != "PASS" ]]; then
            github_fails=true
        fi
        
        if [[ ${#TEST_RESULTS[@]} -gt 0 ]]; then
            for test_name in "${!TEST_RESULTS[@]}"; do
                if [[ "$test_name" =~ TLS|HTTPS ]] && [[ "${TEST_RESULTS[$test_name]}" == "FAIL" ]]; then
                    if [[ "$test_name" =~ google\.com ]]; then
                        tls_issues=true
                        risk_level="MEDIUM"
                    fi
                fi
            done
        fi
        
        if [[ "$github_fails" == "true" ]] && [[ "$tls_issues" == "false" ]]; then
            log::info "ℹ️  MINOR: GitHub-specific connectivity issues ($failed_tests/$total_tests tests failed)"
            log::info ""
            log::info "🔍 What this means:"
            log::info "   • GitHub HTTPS connections are failing"
            log::info "   • Git operations over HTTPS won't work"
            log::info "   • Other web services appear to be working fine"
            log::info ""
            log::info "⚠️  Impact on setup:"
            log::info "   • Git clones over HTTPS will fail"
            log::info "   • npm packages from GitHub may fail to download"
            log::info "   • Most other operations should work normally"
            log::info ""
            log::info "🛠️  Workarounds available:"
            log::info "   • Use SSH for git operations (recommended)"
            log::info "   • Configure git to use HTTP/1.1: git config --global http.version HTTP/1.1"
            log::info "   • Use specific TLS version: git config --global http.sslVersion tlsv1.2"
            log::info ""
            log::success "✅ Setup can continue - GitHub issues are manageable"
            
        elif [[ "$risk_level" == "MEDIUM" ]]; then
            log::warning "⚠️  MODERATE: TLS/HTTPS connectivity issues ($failed_tests/$total_tests tests failed)"
            log::info ""
            log::info "🔍 What this means:"
            log::info "   • Some HTTPS connections are unreliable"
            log::info "   • SSL/TLS handshakes may timeout or fail"
            log::info "   • Package downloads might be slower or fail occasionally"
            log::info ""
            log::info "⚠️  Impact on setup:"
            log::info "   • Moderate risk of download failures"
            log::info "   • Some Docker images may fail to pull"
            log::info "   • Setup may take longer due to retries"
            log::info ""
            log::info "🛠️  Recommended actions:"
            log::info "   • Review the TCP offload suggestions above"
            log::info "   • Consider using HTTP mirrors where available"
            log::info "   • Monitor setup progress closely"
            log::info ""
            log::info "Continue with setup? (moderate risk) [Y/n]"
            if read -r -t 10 response 2>/dev/null; then
                if [[ "$response" =~ ^[Nn]$ ]]; then
                    log::info "Setup cancelled. Address network issues and try again."
                    return "${ERROR_NO_INTERNET}"
                fi
            else
                log::info "No response (timed out or non-interactive), defaulting to continue."
            fi
            log::info "📋 Continuing with moderate network risk..."
            
        else
            log::info "ℹ️  MINOR: Some network tests failed ($failed_tests/$total_tests tests failed)"
            log::info ""
            log::info "🔍 What this means:"
            log::info "   • Core networking functionality works"
            log::info "   • Some advanced features may have issues"
            log::info "   • Setup should complete successfully"
            log::info ""
            log::info "⚠️  Impact on setup:"
            log::info "   • Low risk of failures"
            log::info "   • Specific services may need manual configuration"
            log::info ""
            log::success "✅ Setup can continue with minimal risk"
        fi
        
    else
        log::success "🎉 Excellent! All network tests passed!"
        if [[ "$fixes_applied" == "true" ]]; then
            log::info ""
            log::info "🔧 Automatic fixes were applied during diagnostics:"
            if [[ "$tso_fix_applied" == "true" ]]; then
                log::info "   • TCP Segmentation Offload disabled (fixes GitHub TLS issues)"
            fi
            if [[ "$nginx_fix_applied" == "true" ]]; then
                log::info "   • SearXNG HTTP configuration fixed (port 9200 now works)"
            fi
        fi
        log::info ""
        log::info "✅ Your system has:"
        log::info "   • Full internet connectivity"
        log::info "   • Working DNS resolution"  
        log::info "   • Functional HTTP/HTTPS protocols"
        log::info "   • Proper TLS/SSL support"
        log::info ""
        log::success "🚀 Ready to proceed with setup!"
    fi
    
    # Advanced manual troubleshooting section (if issues remain)
    if [[ $failed_tests -gt 0 ]]; then
        log::header "Advanced Manual Troubleshooting"
        log::info "If automatic fixes didn't work, try these in order:"
        log::info ""
        log::info "1. Reset all network settings to defaults:"
        log::info "   sudo sysctl -p /etc/sysctl.conf"
        log::info ""
        log::info "2. Test with different network namespace:"
        log::info "   sudo ip netns add test"
        log::info "   sudo ip netns exec test curl https://github.com"
        log::info ""
        log::info "3. Capture packet trace for analysis:"
        log::info "   sudo tcpdump -i any -w github-fail.pcap host github.com"
        log::info "   # In another terminal: curl -v https://github.com"
        log::info "   # Analyze with: wireshark github-fail.pcap"
        log::info ""
        log::info "4. Test with different TLS library:"
        log::info "   # With OpenSSL: openssl s_client -connect github.com:443"
        log::info "   # With GnuTLS: gnutls-cli github.com"
        log::info ""
        log::info "5. Check for middlebox interference:"
        log::info "   # Test if problem persists on different network"
        log::info "   # Try mobile hotspot or VPN to isolate network issues"
    fi
    
    return 0
}

# Check if UFW is blocking outbound HTTP/HTTPS traffic
network_diagnostics::check_ufw_blocking() {
    log::info "Checking UFW firewall configuration..."
    
    # Skip if ufw not installed
    if ! command -v ufw >/dev/null 2>&1; then
        log::info "  UFW not installed - skipping firewall checks"
        return 0
    fi
    
    # Check if UFW is active
    local ufw_status
    if flow::can_run_sudo "check UFW status"; then
        ufw_status=$(sudo ufw status 2>/dev/null || echo "inactive")
        
        if [[ "$ufw_status" == "inactive" ]] || echo "$ufw_status" | grep -q "inactive"; then
            log::info "  UFW is inactive - not affecting network traffic"
            return 0
        fi
        
        log::info "  UFW is active - checking for blocking rules..."
        
        # Check default outgoing policy
        local default_policy
        default_policy=$(sudo ufw status verbose 2>/dev/null | grep "Default:" || echo "")
        
        if echo "$default_policy" | grep -q "deny (outgoing)"; then
            log::error "  ⚠️  UFW default policy DENIES outgoing traffic!"
            network_diagnostics::fix_ufw_blocking
            return
        fi
        
        # Check if we have explicit outbound rules for HTTP/HTTPS
        local has_http_out=false
        local has_https_out=false
        local ufw_numbered
        ufw_numbered=$(sudo ufw status numbered 2>/dev/null || echo "")
        
        if echo "$ufw_numbered" | grep -E "80/tcp.*ALLOW OUT" >/dev/null 2>&1; then
            has_http_out=true
        fi
        if echo "$ufw_numbered" | grep -E "443/tcp.*ALLOW OUT" >/dev/null 2>&1; then
            has_https_out=true
        fi
        
        # If we're having IPv4 HTTPS issues and no explicit outbound rules exist
        if [[ "${TEST_RESULTS["HTTPS_GOOGLE"]:-}" == "FAIL" ]] && [[ "$has_https_out" == "false" ]]; then
            log::warning "  ⚠️  No explicit outbound HTTPS rule found"
            log::info "  This may cause issues if default policy changes"
            network_diagnostics::fix_ufw_blocking
        elif [[ "$has_http_out" == "true" ]] && [[ "$has_https_out" == "true" ]]; then
            log::success "  ✓ UFW has proper outbound HTTP/HTTPS rules"
        else
            log::info "  UFW appears properly configured"
        fi
    else
        log::warning "  Cannot check UFW status (no sudo access)"
        log::info "  To check manually: sudo ufw status verbose"
    fi
}

# Fix UFW blocking issues
network_diagnostics::fix_ufw_blocking() {
    log::info "Attempting to fix UFW firewall issues..."
    
    if ! flow::can_run_sudo "fix UFW blocking"; then
        log::warning "Cannot fix UFW automatically (no sudo access)"
        log::info ""
        log::info "🔧 MANUAL FIX REQUIRED:"
        log::info "Run these commands to fix UFW blocking:"
        log::info ""
        log::info "  # Allow outbound HTTP/HTTPS:"
        log::info "  sudo ufw allow out 80/tcp"
        log::info "  sudo ufw allow out 443/tcp"
        log::info "  sudo ufw allow out 53"
        log::info "  sudo ufw allow out 53/udp"
        log::info ""
        log::info "  # Or reset UFW defaults:"
        log::info "  sudo ufw default allow outgoing"
        log::info "  sudo ufw reload"
        log::info ""
        return 1
    fi
    
    log::info "  Adding explicit outbound rules for HTTP/HTTPS..."
    
    # Add outbound rules
    sudo ufw allow out 80/tcp >/dev/null 2>&1
    sudo ufw allow out 443/tcp >/dev/null 2>&1
    sudo ufw allow out 53 >/dev/null 2>&1
    sudo ufw allow out 53/udp >/dev/null 2>&1
    
    # Ensure default policy allows outgoing
    sudo ufw default allow outgoing >/dev/null 2>&1
    
    # Reload UFW
    sudo ufw reload >/dev/null 2>&1
    
    log::success "  ✓ Added UFW outbound rules for HTTP/HTTPS/DNS"
    log::info "  → Testing if this fixed the issue..."
    
    # Re-test HTTPS
    set +e
    if curl -4 https://www.google.com -o /dev/null -s --connect-timeout 3; then
        log::success "  ✓ IPv4 HTTPS now works! UFW fix successful."
        TEST_RESULTS["UFW_FIX"]="SUCCESS"
    else
        log::warning "  → UFW rules added but IPv4 HTTPS still failing"
        log::info "  → May need to restart networking or check other firewall layers"
        TEST_RESULTS["UFW_FIX"]="PARTIAL"
    fi
    set -e
}

# If run directly, execute diagnostics
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    network_diagnostics::run "$@"
fi